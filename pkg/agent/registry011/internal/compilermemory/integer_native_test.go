//go:build darwin && registry_native_limits

package compilermemory

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestNativeRegionJSONIntegersReadOwnedSpans(t *testing.T) {
	action, cancelAction := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelAction()
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelCleanup()
	r, err := Acquire(action, cleanup, 1024)
	if err != nil {
		if r != nil {
			err = errors.Join(err, r.Close())
		}
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	unsigned, err := r.Store([]byte("18446744073709551615"))
	if err != nil {
		t.Fatal(err)
	}
	signed, err := r.Store([]byte("-9223372036854775808"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	if value, err := r.JSONUint64(unsigned); err != nil || value != math.MaxUint64 {
		t.Fatalf("native unsigned read: %d, %v", value, err)
	}
	if value, err := r.JSONInt64(signed); err != nil || value != math.MinInt64 {
		t.Fatalf("native signed read: %d, %v", value, err)
	}
	cancelAction()
	if value, err := r.JSONUint64(unsigned); value != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled native unsigned read: %d, %v", value, err)
	}
	if value, err := r.JSONInt64(signed); value != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled native signed read: %d, %v", value, err)
	}
}
