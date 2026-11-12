package registryworker

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPreparationInputMappingKeepsSentinelAndPageRoundingSeparate(t *testing.T) {
	for _, c := range []struct {
		name                 string
		maximum, page, wanted int
	}{
		{"one-byte", 1, 4096, 4096},
		{"sentinel-last-byte", 4095, 4096, 4096},
		{"sentinel-next-page", 4096, 4096, 8192},
		{"second-page", 8191, 4096, 8192},
		{"maximum-input", 64 << 20, 4096, (64 << 20) + 4096},
		{"zero-input", 0, 4096, 0},
		{"negative-input", -1, 4096, 0},
		{"outside-input-domain", (64 << 20) + 1, 4096, 0},
		{"zero-page", 1, 0, 0},
		{"negative-page", 1, -1, 0},
		{"rounding-overflow", 1, int(^uint(0) >> 1), 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			n, ok := preparationInputMappingBytes(c.maximum, c.page)
			if n != c.wanted || ok != (c.wanted != 0) {
				t.Fatal("mapping and accepted input capacities differ", n, ok)
			}
		})
	}
}

type preparationInputErrorReader struct{}

func (preparationInputErrorReader) Read(p []byte) (int, error) {
	p[0] = 'x'
	return 1, io.ErrClosedPipe
}

func TestPreparationInputRefusalsPreserveConsumptionAndCleanup(t *testing.T) {
	for _, name := range []string{"within", "exact", "overflow", "empty", "read-error", "consumer-error", "cancel-before", "cleanup-before", "cancel-after-acquire", "cancel-consumer", "cleanup-consumer", "acquire-error", "nil-memory", "short-memory", "missing-close", "close-error", "nil-action", "nil-cleanup", "nil-reader", "nil-consumer", "nil-acquire", "bad-limit"} {
		t.Run(name, func(t *testing.T) {
			action, cancel := context.WithCancel(context.Background())
			cleanup, cancelCleanup := context.WithCancel(context.Background())
			defer cancel()
			defer cancelCleanup()
			var actionContext context.Context = action
			var cleanupContext context.Context = cleanup
			var reader io.Reader = strings.NewReader("raw")
			maximum := 8
			data := make([]byte, maximum+1)
			acquired, consumed, closed := 0, 0, 0
			fault := errors.New("bounded unit failure")
			var wanted error
			memory := &preparationInputMemory{data: data, close: func() error {
				closed++
				if name == "close-error" {
					return fault
				}
				clear(data)
				return nil
			}}
			acquire := func(n int) (*preparationInputMemory, error) {
				acquired++
				if n != maximum {
					t.Fatal("input allowance changed", n)
				}
				if name == "acquire-error" {
					return nil, fault
				}
				if name == "nil-memory" {
					return nil, nil
				}
				if name == "cancel-after-acquire" {
					cancel()
				}
				return memory, nil
			}
			consume := func(raw []byte) error {
				consumed++
				if len(raw) == 0 || &raw[0] != &data[0] || len(raw) != cap(raw) {
					t.Fatal("consumer did not borrow the exact bounded input")
				}
				if name == "consumer-error" {
					return fault
				}
				if name == "cancel-consumer" {
					cancel()
				}
				if name == "cleanup-consumer" {
					cancelCleanup()
				}
				return nil
			}
			switch name {
			case "exact":
				reader = strings.NewReader("12345678")
			case "overflow":
				reader, wanted = strings.NewReader("123456789"), ErrConfiguration
			case "empty":
				reader, wanted = strings.NewReader(""), ErrConfiguration
			case "read-error":
				reader, wanted = preparationInputErrorReader{}, io.ErrClosedPipe
			case "consumer-error", "acquire-error", "close-error":
				wanted = fault
			case "cancel-before", "cancel-after-acquire", "cancel-consumer":
				wanted = context.Canceled
				if name == "cancel-before" {
					cancel()
				}
			case "cleanup-before", "cleanup-consumer":
				wanted = context.Canceled
				if name == "cleanup-before" {
					cancelCleanup()
				}
			case "nil-memory", "short-memory", "missing-close":
				wanted = ErrUnavailable
				if name == "short-memory" {
					memory.data = data[:maximum]
				}
				if name == "missing-close" {
					memory.close = nil
				}
			case "nil-action":
				actionContext, wanted = nil, ErrConfiguration
			case "nil-cleanup":
				cleanupContext, wanted = nil, ErrConfiguration
			case "nil-reader":
				reader, wanted = nil, ErrConfiguration
			case "nil-consumer":
				consume, wanted = nil, ErrConfiguration
			case "nil-acquire":
				acquire, wanted = nil, ErrConfiguration
			case "bad-limit":
				maximum, wanted = 0, ErrConfiguration
			}
			err := consumePreparationInput(actionContext, cleanupContext, maximum, reader, consume, acquire)
			if !errors.Is(err, wanted) || wanted == nil && err != nil {
				t.Fatal("refusal or native cleanup error changed", err, wanted)
			}
			if acquired > 1 || consumed > 1 || closed > 1 {
				t.Fatal("an input attempt was retried")
			}
			if acquired == 1 && name != "acquire-error" && name != "nil-memory" && name != "missing-close" && name != "cleanup-consumer" && closed != 1 {
				t.Fatal("acquired input was not released", closed)
			}
			if name == "close-error" || name == "cleanup-consumer" || name == "missing-close" {
				if !errors.Is(err, ErrInputCleanup) {
					t.Fatal("native cleanup failure was hidden", err)
				}
			}
		})
	}
}

func TestPreparationInputClaimCannotBeCopiedOrRenewed(t *testing.T) {
	lifetime := &PreparationLifetime{action: context.Background(), cleanup: context.Background()}
	lifetime.self = lifetime
	copied := &PreparationLifetime{self: lifetime, action: lifetime.action, cleanup: lifetime.cleanup}
	empty := &PreparationLifetime{}
	for _, candidate := range []*PreparationLifetime{nil, empty, copied} {
		if err := candidate.ConsumeInput(nil); !errors.Is(err, ErrUnavailable) {
			t.Fatal("invalid lifetime accepted input", err)
		}
	}
	if lifetime.inputClaimed.Load() {
		t.Fatal("copy refusal spent the original claim")
	}
	if err := lifetime.ConsumeInput(nil); !errors.Is(err, ErrConfiguration) {
		t.Fatal("invalid input consumer changed", err)
	}
	if err := lifetime.ConsumeInput(nil); !errors.Is(err, ErrSpent) {
		t.Fatal("refused input claim was restored", err)
	}
	lifetime.closing.Store(true)
	if err := lifetime.ConsumeInput(nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal("closed lifetime accepted input", err)
	}
}

func TestPreparationInputMemoryCloseIsOneUseEvenAfterFailure(t *testing.T) {
	fault := errors.New("close failure")
	var calls atomic.Int32
	memory := &preparationInputMemory{data: []byte("raw"), close: func() error {
		calls.Add(1)
		return fault
	}}
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if err := memory.release(context.Background()); !errors.Is(err, fault) {
				t.Error("native close failure changed", err)
			}
		})
	}
	group.Wait()
	if calls.Load() != 1 || memory.data != nil || !memory.closed {
		t.Fatal("native cleanup was retried or kept a borrowed view")
	}
	var absent *preparationInputMemory
	if err := absent.release(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("absent memory was released", err)
	}
	if err := (&preparationInputMemory{}).release(nil); !errors.Is(err, ErrConfiguration) {
		t.Fatal("missing cleanup context accepted", err)
	}
}
