package registry011

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// These private lifecycle units simulate only a backend's outcomes. They do not
// claim native allocation, a real compiled plan or a qualified execution grant.
type simulatedNativeHistoryPath struct {
	compileErr      error
	readErr         error
	temporaryErr    error
	closeErr        error
	summary         HistoryPreparationSummary
	withoutPlan     bool
	beforeCompile   func()
	beforeRead      func()
	compiles        int
	reads           int
	temporaryCloses int
	closes          int
}

func (n *simulatedNativeHistoryPath) compileLocked(s *HistoryPreparationSession, _ []byte) error {
	n.compiles++
	if n.beforeCompile != nil {
		n.beforeCompile()
	}
	if n.compileErr == nil && !n.withoutPlan {
		s.plan = &providerHistorySourcePlan{}
	}
	return n.compileErr
}

func (n *simulatedNativeHistoryPath) readResultLocked(*HistoryPreparationSession) (HistoryPreparationSummary, error) {
	n.reads++
	if n.beforeRead != nil {
		n.beforeRead()
	}
	return n.summary, n.readErr
}

func (n *simulatedNativeHistoryPath) closeTemporary() error {
	n.temporaryCloses++
	return n.temporaryErr
}

func (n *simulatedNativeHistoryPath) closeAll() error {
	n.closes++
	return n.closeErr
}

func simulatedNativeHistorySession(t *testing.T, path *simulatedNativeHistoryPath, maximumResult int) (*HistoryPreparationSession, context.CancelFunc) {
	t.Helper()
	origin := time.Now()
	action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	t.Cleanup(cancel)
	session, err := NewHistoryPreparationSession(action, origin, 1024, maximumResult)
	if err != nil {
		t.Fatal(err)
	}
	session.native = path
	return session, cancel
}

func TestProviderHistoryPreparationSimulatedNativeRefusalsRemainTerminal(t *testing.T) {
	parseFailure := errors.New("simulated preparation failure")
	temporaryFailure := errors.New("simulated temporary cleanup failure")
	releaseFailure := errors.New("simulated retained cleanup failure")
	for _, tc := range []struct {
		name string
		path simulatedNativeHistoryPath
		want error
	}{
		{"compile", simulatedNativeHistoryPath{compileErr: parseFailure}, parseFailure},
		{"temporary", simulatedNativeHistoryPath{temporaryErr: temporaryFailure}, ErrHistoryPreparationCleanup},
		{"joined", simulatedNativeHistoryPath{compileErr: parseFailure, closeErr: errors.Join(ErrHistoryPreparationCleanup, releaseFailure)}, releaseFailure},
		{"missing-plan", simulatedNativeHistoryPath{withoutPlan: true}, ErrConfiguration},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session, _ := simulatedNativeHistorySession(t, &tc.path, 2048)
			if err := session.CompileOwned([]byte("{}")); !errors.Is(err, tc.want) {
				t.Fatalf("compile refusal lost cause: %v", err)
			}
			if !session.closed || !session.used.Load() || session.plan != nil {
				t.Fatal("refused attempt retained a live selection")
			}
			if err := session.CompileOwned([]byte("{}")); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("attempt retried: %v", err)
			}
			if output, err := session.FinishDiagnostic(); output != nil || err == nil {
				t.Fatalf("refused attempt produced output: %q, %v", output, err)
			}
			closeErr := session.Close()
			if tc.path.temporaryErr != nil && !errors.Is(closeErr, temporaryFailure) {
				t.Fatalf("temporary failure not cached: %v", closeErr)
			}
			if tc.path.closeErr != nil && !errors.Is(closeErr, releaseFailure) {
				t.Fatalf("retained failure not cached: %v", closeErr)
			}
			_ = session.Close()
			if tc.path.compiles != 1 || tc.path.temporaryCloses != 1 || tc.path.closes != 1 || tc.path.reads != 0 {
				t.Fatalf("operation repeated: %+v", tc.path)
			}
		})
	}
}

func TestProviderHistoryPreparationSimulatedNativeResultCannotFallBack(t *testing.T) {
	readFailure := errors.New("simulated native read failure")
	cleanupFailure := errors.New("simulated native cleanup failure")
	for _, mode := range []string{"read", "cleanup", "ready", "result-bound", "canceled", "post-read-cancel", "missing-plan"} {
		t.Run(mode, func(t *testing.T) {
			path := &simulatedNativeHistoryPath{summary: HistoryPreparationSummary{Format: historyPreparationResultFormat}}
			session, cancel := simulatedNativeHistorySession(t, path, 2048)
			if err := session.CompileOwned([]byte("{}")); err != nil {
				t.Fatal(err)
			}
			if err := session.CompileOwned([]byte("{}")); !errors.Is(err, ErrConfiguration) || path.closes != 0 {
				t.Fatal("losing duplicate destroyed the first native selection")
			}
			switch mode {
			case "read":
				path.readErr = readFailure
			case "cleanup":
				path.closeErr = errors.Join(ErrHistoryPreparationCleanup, cleanupFailure)
			case "ready":
				path.summary.SourceReady = true
			case "result-bound":
				session.maximumResult = 1
			case "canceled":
				cancel()
			case "post-read-cancel":
				path.beforeRead = cancel
			case "missing-plan":
				session.plan = nil
			}
			output, err := session.FinishDiagnostic()
			if output != nil || err == nil || !session.closed || session.plan != nil {
				t.Fatalf("native refusal became output: %q, %v", output, err)
			}
			if mode == "read" && !errors.Is(err, readFailure) {
				t.Fatalf("read failure replaced by a Go result: %v", err)
			}
			if mode == "cleanup" && !errors.Is(err, cleanupFailure) {
				t.Fatalf("cleanup error lost: %v", err)
			}
			if path.closes != 1 {
				t.Fatalf("cleanup attempts=%d", path.closes)
			}
			_ = session.Close()
			if path.closes != 1 {
				t.Fatal("repeated Close retried cleanup")
			}
		})
	}
}

func TestProviderHistoryPreparationSimulatedNativeWrongOrderAndConcurrentClose(t *testing.T) {
	path := &simulatedNativeHistoryPath{}
	session, _ := simulatedNativeHistorySession(t, path, 2048)
	if output, err := session.FinishDiagnostic(); output != nil || !errors.Is(err, ErrConfiguration) || path.closes != 1 {
		t.Fatal("finish-before-compile did not dispose its attempt")
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	path = &simulatedNativeHistoryPath{beforeCompile: func() { close(entered); <-release }}
	session, _ = simulatedNativeHistorySession(t, path, 2048)
	compiled := make(chan error, 1)
	go func() { compiled <- session.CompileOwned([]byte("{}")) }()
	<-entered
	if session.mu.TryLock() {
		session.mu.Unlock()
		t.Fatal("active producer did not retain its original session lock")
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			if err := session.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-compiled; err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	if path.compiles != 1 || path.closes != 1 || !session.closed {
		t.Fatal("concurrent Close did not preserve one owner and one cleanup")
	}
}
