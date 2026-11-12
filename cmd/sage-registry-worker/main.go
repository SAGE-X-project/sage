// sage-registry-worker is the fixed preparation child of the owning launcher.
// It emits diagnostic compiled DATA only; it cannot open a qualified Source.
package main

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/sage-x-project/sage/internal/registryworker"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) != 2 || os.Args[1] != "--sage-registry-history-worker" {
		return 64
	}
	lifetime, err := registryworker.OpenPreparationLifetime(context.Background())
	if err != nil {
		return 65
	}
	defer func() { _ = lifetime.Close() }()
	snapshot, err := lifetime.Snapshot()
	if err != nil {
		return 65
	}
	action, err := lifetime.Context()
	if err != nil {
		return 65
	}
	cleanup, err := lifetime.CleanupContext()
	if err != nil {
		return 65
	}
	stopSupervisor := make(chan struct{})
	defer close(stopSupervisor)
	go func() {
		select {
		case <-stopSupervisor:
			return
		case <-cleanup.Done():
			select {
			case <-stopSupervisor:
				return
			default:
				// Terminate only this fixed child, never a supplied PID or
				// unrelated process. The parent also keeps its own hard stop.
				os.Exit(125)
			}
		}
	}()
	// Create and retain the compiler BEFORE standard-input consumption. Its
	// action context is canceled by the live parent's original control pipe.
	session, err := registry011.NewNativeHistoryPreparationSession(action, cleanup, snapshot.Origin, snapshot.MaximumInputBytes, snapshot.MaximumResultBytes)
	if err != nil {
		return 65
	}
	defer func() { _ = session.Close() }()
	// A fixed nonsensitive phase marker lets runtime tests synchronize with
	// the real compiler-before-input boundary. It grants no readiness authority.
	if _, err := io.WriteString(os.Stderr, "sage-registry-preparation-input-ready\n"); err != nil {
		return 65
	}
	var journal *registryworker.JournalWriter
	var output []byte
	phase := 65
	prepareErr := lifetime.ConsumeInput(func(input []byte) error {
		phase = 66
		var err error
		journal, err = registryworker.OpenJournal(cleanup, input)
		if err != nil {
			return err
		}
		phase = 67
		return session.CompileOwned(input)
	})
	if journal == nil {
		return phase
	}
	// The raw mapping has been released. The SAME session still owns the
	// detached plan, original compiler, clock and input digest. Finish before
	// releasing the live control channel, which cancels the action context.
	if prepareErr == nil {
		output, prepareErr = session.FinishDiagnostic()
	}
	prepareErr = errors.Join(prepareErr, session.Close())
	// Native input release, logical plan disposal, control release and journal
	// Close must all precede stdout. Compiler heap/capacity remains unqualified.
	releaseErr := lifetime.ReleaseControl()
	if prepareErr != nil || releaseErr != nil {
		code := "compilation"
		if releaseErr != nil || errors.Is(prepareErr, registryworker.ErrInputCleanup) || errors.Is(prepareErr, registry011.ErrHistoryPreparationCleanup) {
			code = "cleanup"
		} else if errors.Is(prepareErr, context.Canceled) || errors.Is(prepareErr, context.DeadlineExceeded) {
			code = "canceled"
		}
		_ = journal.Fail(code)
		_ = journal.Close()
		return 67
	}
	if err := journal.Complete(output); err != nil {
		_ = journal.Close()
		return 68
	}
	if err := journal.Close(); err != nil {
		return 68
	}
	n, err := os.Stdout.Write(output)
	if err != nil || n != len(output) {
		return 69
	}
	return 0
}
