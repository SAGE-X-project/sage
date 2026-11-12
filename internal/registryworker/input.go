package registryworker

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
)

var ErrInputCleanup = errors.New("registry worker: raw input cleanup incomplete")

// This owns only the raw input mapping. The compiler's detached heap copy,
// decoded objects and other native memory remain separate obligations.
type preparationInputMemory struct {
	mu      sync.Mutex
	data    []byte
	close   func() error
	closed  bool
	closing error
}

func (m *preparationInputMemory) release(ctx context.Context) error {
	if m == nil {
		return ErrUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return m.closing
	}
	if ctx == nil {
		return ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		// The fixed child exits under its original hard-stop. Expired native
		// cleanup remains incomplete; process teardown is not a successful close.
		return err
	}
	m.closed = true
	if m.close == nil {
		m.closing = ErrUnavailable
	} else {
		m.closing = m.close()
	}
	m.data = nil
	return m.closing
}

// ConsumeInput receives the original stdin bytes in owned native memory. The
// synchronous consumer must neither retain its borrowed slice nor start users
// that outlive the call. The fixed worker keeps Prepare's existing clone.
// No other memory, resource reservation or execution permission is certified.
func (l *PreparationLifetime) ConsumeInput(consume func([]byte) error) error {
	if l == nil || l.self != l || l.closing.Load() {
		return ErrUnavailable
	}
	if !l.inputClaimed.CompareAndSwap(false, true) {
		return ErrSpent
	}
	return consumePreparationInput(l.action, l.cleanup, l.snapshot.MaximumInputBytes, os.Stdin, consume, acquirePreparationInputMemory)
}

// The injected allocator is private to bounded unit scenarios. Production
// always supplies the fixed native implementation above, never caller code.
func consumePreparationInput(action, cleanup context.Context, maximum int, reader io.Reader, consume func([]byte) error, acquire func(int) (*preparationInputMemory, error)) (err error) {
	if action == nil || cleanup == nil || maximum <= 0 || maximum > 64<<20 || reader == nil || consume == nil || acquire == nil {
		return ErrConfiguration
	}
	if err := errors.Join(action.Err(), cleanup.Err()); err != nil {
		return err
	}
	memory, err := acquire(maximum)
	if err != nil || memory == nil {
		return errors.Join(ErrUnavailable, err)
	}
	// Every return, including a consumer error, joins native cleanup failure
	// into the result before the child can complete its journal or publish.
	defer func() {
		if closeErr := errors.Join(memory.release(cleanup), cleanup.Err()); closeErr != nil {
			err = errors.Join(err, ErrInputCleanup, closeErr)
		}
	}()
	if len(memory.data) < maximum+1 || memory.close == nil {
		return ErrUnavailable
	}
	if err := action.Err(); err != nil {
		return err
	}
	n, readErr := io.ReadFull(contextReader{ctx: action, reader: reader}, memory.data[:maximum+1])
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return readErr
	}
	if n == 0 || n > maximum {
		return ErrConfiguration
	}
	if err := action.Err(); err != nil {
		return err
	}
	consumeErr := consume(memory.data[:n:n])
	return errors.Join(consumeErr, action.Err())
}

// The successful input allowance remains N. N+1 is rejection-only space;
// physical mapping/locking rounds that amount up to whole OS pages.
func preparationInputMappingBytes(maximum, page int) (int, bool) {
	if maximum <= 0 || maximum > 64<<20 || page <= 0 {
		return 0, false
	}
	sentinel := maximum + 1
	maxInt := int(^uint(0) >> 1)
	if sentinel > maxInt-(page-1) {
		return 0, false
	}
	return (sentinel + page - 1) / page * page, true
}
