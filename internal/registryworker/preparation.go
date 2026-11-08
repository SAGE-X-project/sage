package registryworker

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	preparationMagic         = "sage-worker-preparation-v1"
	preparationHeaderMaximum = 512
)

var preparationDescriptorClaimed atomic.Bool

// PreparationSnapshot is diagnostic DATA from the parent's original attempt.
// Its timestamps and limits never establish a native resource lease or a
// Registry derivation. The live parent still owns the actual process stop.
type PreparationSnapshot struct {
	Origin, ActionDeadline, HardStopDeadline, WholeDeadline time.Time
	MaximumInputBytes, MaximumResultBytes                   int
}

func (s PreparationSnapshot) valid() bool {
	return !s.Origin.IsZero() && s.Origin.Before(s.ActionDeadline) &&
		!s.HardStopDeadline.Before(s.ActionDeadline) && s.HardStopDeadline.Before(s.WholeDeadline) &&
		s.WholeDeadline.Sub(s.Origin) <= 31*24*time.Hour &&
		s.MaximumInputBytes > 0 && s.MaximumInputBytes <= 64<<20 &&
		s.MaximumResultBytes > 0 && s.MaximumResultBytes <= 64<<20
}

type preparationControl struct {
	read, write       *os.File
	snapshot          PreparationSnapshot
	cancel            context.CancelFunc
	done              chan struct{}
	readOnce          sync.Once
	readErr, writeErr error
}

func newPreparationControl(o *Owner, binding journalBinding) (*preparationControl, error) {
	if o == nil || o.self != o || !binding.valid() || o.action.Err() != nil {
		return nil, ErrUnavailable
	}
	action, actionOK := o.action.Deadline()
	hardStop, hardOK := o.hardStop.Deadline()
	whole, wholeOK := o.whole.Deadline()
	snapshot := PreparationSnapshot{o.origin, action, hardStop, whole, o.configuration.Limits.InputBytes, o.configuration.Limits.ResultBytes}
	if !actionOK || !hardOK || !wholeOK || !snapshot.valid() {
		return nil, ErrConfiguration
	}
	header := encodePreparationHeader(snapshot, binding)
	if len(header) > preparationHeaderMaximum {
		return nil, ErrConfiguration
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	// The single fixed record is smaller than the empty pipe's atomic write
	// capacity. No caller-controlled stream or additional messages are written.
	n, err := io.WriteString(w, header)
	if err != nil || n != len(header) {
		return nil, errors.Join(ErrUnavailable, err, r.Close(), w.Close())
	}
	ctx, cancel := context.WithCancel(o.action)
	c := &preparationControl{read: r, write: w, snapshot: snapshot, cancel: cancel, done: make(chan struct{})}
	go func() {
		<-ctx.Done()
		c.writeErr = c.write.Close()
		close(c.done)
	}()
	return c, nil
}

func (c *preparationControl) releaseRead() {
	c.readOnce.Do(func() { c.readErr = c.read.Close() })
}

func (c *preparationControl) close() error {
	c.cancel()
	c.releaseRead()
	<-c.done
	return errors.Join(c.readErr, c.writeErr)
}

func encodePreparationHeader(s PreparationSnapshot, b journalBinding) string {
	return strings.Join([]string{preparationMagic, b.nonce, b.input, b.worker,
		strconv.FormatInt(s.Origin.UnixNano(), 10), strconv.FormatInt(s.ActionDeadline.UnixNano(), 10),
		strconv.FormatInt(s.HardStopDeadline.UnixNano(), 10), strconv.FormatInt(s.WholeDeadline.UnixNano(), 10),
		strconv.Itoa(s.MaximumInputBytes), strconv.Itoa(s.MaximumResultBytes)}, " ") + "\n"
}

func parsePreparationHeader(raw string, binding journalBinding) (PreparationSnapshot, error) {
	var empty PreparationSnapshot
	if len(raw) > preparationHeaderMaximum || !strings.HasSuffix(raw, "\n") || !binding.valid() {
		return empty, ErrConfiguration
	}
	fields := strings.Split(strings.TrimSuffix(raw, "\n"), " ")
	if len(fields) != 10 || fields[0] != preparationMagic || fields[1] != binding.nonce || fields[2] != binding.input || fields[3] != binding.worker {
		return empty, ErrConfiguration
	}
	var values [6]int64
	for i := range values {
		n, err := strconv.ParseInt(fields[i+4], 10, 64)
		if err != nil || n <= 0 || strconv.FormatInt(n, 10) != fields[i+4] {
			return empty, ErrConfiguration
		}
		values[i] = n
	}
	if values[4] <= 0 || values[4] > 64<<20 || values[5] <= 0 || values[5] > 64<<20 {
		return empty, ErrConfiguration
	}
	snapshot := PreparationSnapshot{
		Origin: time.Unix(0, values[0]), ActionDeadline: time.Unix(0, values[1]),
		HardStopDeadline: time.Unix(0, values[2]), WholeDeadline: time.Unix(0, values[3]),
		MaximumInputBytes: int(values[4]), MaximumResultBytes: int(values[5]),
	}
	if !snapshot.valid() {
		return empty, ErrConfiguration
	}
	return snapshot, nil
}

// PreparationLifetime consumes the single inherited descriptor5. It couples
// cooperative preparation to the parent's live control pipe; a DATA snapshot
// alone cannot create it. It still supplies no physical-memory/backing promise,
// compiler funding, private issuer, protected-use permission or custody proof.
type PreparationLifetime struct {
	self                        *PreparationLifetime
	snapshot                    PreparationSnapshot
	action, cleanup             context.Context
	cancelAction, cancelCleanup context.CancelFunc
	file                        *os.File
	readerDone, parentDone      chan struct{}
	closing                     atomic.Bool
	releaseOnce                 sync.Once
	closeErr                    error
}

func (*PreparationLifetime) UnmarshalJSON([]byte) error { return ErrUnavailable }

// OpenPreparationLifetime must run before consuming the request. The fixed
// child observes action cancellation and enforces the original cleanup bound,
// including parent loss. The real parent retains its original monotonic
// action/hard-stop and process handle; serialized timestamps are not authority.
func OpenPreparationLifetime(ctx context.Context) (*PreparationLifetime, error) {
	if !preparationDescriptorClaimed.CompareAndSwap(false, true) {
		return nil, ErrSpent
	}
	if !platformSupported() {
		return nil, ErrUnsupported
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrConfiguration
	}
	environment := os.Getenv(journalEnvironment)
	if len(environment) != 3*64+2 {
		return nil, ErrConfiguration
	}
	parts := strings.Split(environment, ":")
	if len(parts) != 3 {
		return nil, ErrConfiguration
	}
	binding := journalBinding{parts[0], parts[1], parts[2]}
	if !binding.valid() {
		return nil, ErrConfiguration
	}
	f, err := inheritedPreparationPipe()
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.Join(ErrUnavailable, err, f.Close())
	}
	reader := bufio.NewReaderSize(io.LimitReader(f, preparationHeaderMaximum+1), preparationHeaderMaximum+1)
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err, f.Close())
	}
	snapshot, err := parsePreparationHeader(header, binding)
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	action, cancelAction := context.WithDeadline(ctx, snapshot.ActionDeadline)
	cleanup, cancelCleanup := context.WithDeadline(context.WithoutCancel(ctx), snapshot.HardStopDeadline)
	if action.Err() != nil {
		cancelAction()
		cancelCleanup()
		return nil, errors.Join(ErrUnavailable, f.Close())
	}
	l := &PreparationLifetime{snapshot: snapshot, action: action, cleanup: cleanup,
		cancelAction: cancelAction, cancelCleanup: cancelCleanup, file: f,
		readerDone: make(chan struct{}), parentDone: make(chan struct{})}
	l.self = l
	go func() {
		// Any extra byte is as invalid as EOF. The owner only holds the pipe
		// open or closes it; there are no renewal/reset messages.
		_, _ = reader.ReadByte()
		if !l.closing.Load() {
			cancelAction()
			// Loss of the parent stops action immediately. Cleanup retains
			// only its ORIGINAL hard-stop window, never a renewed timeout.
			close(l.parentDone)
		}
		close(l.readerDone)
	}()
	return l, nil
}

// Snapshot copies diagnostic DATA, never the live control descriptor.
func (l *PreparationLifetime) Snapshot() (PreparationSnapshot, error) {
	if l == nil || l.self != l || l.closing.Load() {
		return PreparationSnapshot{}, ErrUnavailable
	}
	return l.snapshot, nil
}

// Context observes the original action boundary, including control-pipe loss.
func (l *PreparationLifetime) Context() (context.Context, error) {
	if l == nil || l.self != l || l.closing.Load() {
		return nil, ErrUnavailable
	}
	return l.action, nil
}

// CleanupContext is limited to the original child hard-stop boundary.
func (l *PreparationLifetime) CleanupContext() (context.Context, error) {
	if l == nil || l.self != l || l.closing.Load() {
		return nil, ErrUnavailable
	}
	return l.cleanup, nil
}

// ParentDone stops action on control-pipe loss. The fixed child supervisor
// exits no later than the original cleanup boundary, including parent death.
func (l *PreparationLifetime) ParentDone() (<-chan struct{}, error) {
	if l == nil || l.self != l || l.closing.Load() {
		return nil, ErrUnavailable
	}
	return l.parentDone, nil
}

// ReleaseControl closes the preparation descriptor after work has stopped.
// The original cleanup context remains available for the terminal journal.
// No new control descriptor or preparation attempt can be opened afterward.
func (l *PreparationLifetime) ReleaseControl() error {
	if l == nil || l.self != l {
		return ErrUnavailable
	}
	l.releaseOnce.Do(func() {
		l.closing.Store(true)
		l.cancelAction()
		l.closeErr = l.file.Close()
		<-l.readerDone
	})
	return l.closeErr
}

// Close also ends the remaining cleanup context. It never refunds the claim.
func (l *PreparationLifetime) Close() error {
	if l == nil || l.self != l {
		return ErrUnavailable
	}
	err := l.ReleaseControl()
	l.cancelCleanup()
	return err
}
