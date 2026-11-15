package compilermemory

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func integerTestRegion(t *testing.T, action context.Context) *Region {
	t.Helper()
	cleanup, cancel := context.WithTimeout(context.Background(), time.Minute)
	r, err := acquireWithOperations(action, cleanup, 1024, &simulatedRegionOperations{})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		cancel()
	})
	return r
}

func TestRegionJSONUint64PreservesExactIntegerValues(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value uint64
		valid bool
	}{
		{"0", 0, true}, {"1", 1, true},
		{"9007199254740993", 9007199254740993, true},
		{"18446744073709551615", math.MaxUint64, true},
		{"18446744073709551616", 0, false},
		{"999999999999999999999", 0, false},
		{"-1", 0, false}, {"-0", 0, false}, {"+1", 0, false},
		{"00", 0, false}, {"01", 0, false}, {"1.0", 0, false},
		{"1e0", 0, false}, {" 1", 0, false}, {"1 ", 0, false},
		{"null", 0, false}, {"true", 0, false}, {"\"1\"", 0, false},
		{"", 0, false}, {"1\x00", 0, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			r := integerTestRegion(t, action)
			s, err := r.Store([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			value, err := r.JSONUint64(s)
			if tc.valid {
				if err != nil || value != tc.value {
					t.Fatalf("got %d, %v; want %d", value, err, tc.value)
				}
			} else if !errors.Is(err, ErrJSONInteger) || value != 0 {
				t.Fatalf("invalid token returned %d, %v", value, err)
			}
		})
	}
}

func TestRegionJSONInt64PreservesExactIntegerValues(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value int64
		valid bool
	}{
		{"0", 0, true}, {"1", 1, true}, {"-1", -1, true},
		{"9007199254740993", 9007199254740993, true},
		{"9223372036854775807", math.MaxInt64, true},
		{"-9223372036854775808", math.MinInt64, true},
		{"9223372036854775808", 0, false}, {"-9223372036854775809", 0, false},
		{"-", 0, false}, {"-0", 0, false}, {"-00", 0, false},
		{"-01", 0, false}, {"+1", 0, false}, {"01", 0, false},
		{"--1", 0, false}, {"1.0", 0, false}, {"-1e0", 0, false},
		{" 1", 0, false}, {"1 ", 0, false}, {"null", 0, false},
		{"", 0, false}, {"1\x00", 0, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			r := integerTestRegion(t, action)
			s, err := r.Store([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			value, err := r.JSONInt64(s)
			if tc.valid {
				if err != nil || value != tc.value {
					t.Fatalf("got %d, %v; want %d", value, err, tc.value)
				}
			} else if !errors.Is(err, ErrJSONInteger) || value != 0 {
				t.Fatalf("invalid token returned %d, %v", value, err)
			}
		})
	}
}

func TestRegionJSONIntegersRefuseForeignClosedAndCopiedOwners(t *testing.T) {
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, read := range []struct {
		name string
		call func(*Region, Span) error
	}{
		{"unsigned", func(r *Region, s Span) error { _, err := r.JSONUint64(s); return err }},
		{"signed", func(r *Region, s Span) error { _, err := r.JSONInt64(s); return err }},
	} {
		t.Run(read.name, func(t *testing.T) {
			r := integerTestRegion(t, action)
			other := integerTestRegion(t, action)
			s, err := r.Store([]byte("42"))
			if err != nil {
				t.Fatal(err)
			}
			if err := read.call(nil, s); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("nil owner: %v", err)
			}
			if err := read.call(&Region{}, s); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("uninitialized owner: %v", err)
			}
			// A shallow copy must not validate the live owner's span. Copy fields
			// explicitly without copying its mutex or invoking a native operation.
			copied := &Region{self: r}
			if err := read.call(copied, s); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("copied owner: %v", err)
			}
			if err := read.call(other, s); !errors.Is(err, ErrRange) {
				t.Fatalf("foreign span: %v", err)
			}
			if err := read.call(r, Span{owner: r, offset: 1, length: 2}); !errors.Is(err, ErrRange) {
				t.Fatalf("uncommitted range: %v", err)
			}
			if err := r.Seal(); err != nil {
				t.Fatal(err)
			}
			if err := read.call(r, s); err != nil {
				t.Fatalf("sealed read: %v", err)
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}
			if err := read.call(r, s); !errors.Is(err, ErrClosed) {
				t.Fatalf("closed owner: %v", err)
			}
		})
	}
}

type integerReadContext struct {
	context.Context
	mu     sync.Mutex
	calls  int
	failAt int
}

func (c *integerReadContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.failAt != 0 && c.calls >= c.failAt {
		return context.Canceled
	}
	return c.Context.Err()
}

func TestRegionJSONIntegersClearValuesOnOriginalContextRefusal(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, checks := range []int{1, 2} {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			ctx := &integerReadContext{Context: action}
			r := integerTestRegion(t, ctx)
			s, err := r.Store([]byte("42"))
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			ctx.mu.Lock()
			ctx.failAt = ctx.calls + checks
			ctx.mu.Unlock()
			if signed {
				value, err := r.JSONInt64(s)
				if value != 0 || !errors.Is(err, context.Canceled) {
					t.Fatalf("signed refusal %d: got %d, %v", checks, value, err)
				}
			} else {
				value, err := r.JSONUint64(s)
				if value != 0 || !errors.Is(err, context.Canceled) {
					t.Fatalf("unsigned refusal %d: got %d, %v", checks, value, err)
				}
			}
			cancel()
		}
	}
}
