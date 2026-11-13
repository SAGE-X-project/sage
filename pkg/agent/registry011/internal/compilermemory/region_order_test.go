package compilermemory

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRegionCommittedRangeOrderPreservesOwnership(t *testing.T) {
	region := decodedStoreRegion(t, nil)
	left, err := region.Store([]byte("first"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := region.Store([]byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		left, right Span
		want        bool
	}{{left, right, true}, {right, left, false}, {left, left, false}} {
		if actual, err := region.Precedes(item.left, item.right); err != nil || actual != item.want {
			t.Fatalf("order=%v, error=%v; want=%v", actual, err, item.want)
		}
	}
	foreign := decodedStoreRegion(t, nil)
	other, err := foreign.Store([]byte("other"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []Span{{}, other} {
		if actual, err := region.Precedes(left, invalid); actual || !errors.Is(err, ErrRange) {
			t.Fatalf("foreign right order=%v, error=%v", actual, err)
		}
		if actual, err := region.Precedes(invalid, right); actual || !errors.Is(err, ErrRange) {
			t.Fatalf("foreign left order=%v, error=%v", actual, err)
		}
	}
	for _, invalid := range []*Region{nil, {self: region}} {
		if actual, err := invalid.Precedes(left, right); actual || !errors.Is(err, ErrConfiguration) {
			t.Fatalf("invalid region order=%v, error=%v", actual, err)
		}
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if actual, err := region.Precedes(left, right); actual || !errors.Is(err, ErrClosed) {
		t.Fatalf("closed order=%v, error=%v", actual, err)
	}
}

func TestRegionCommittedRangeOrderKeepsOriginalContext(t *testing.T) {
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	controlled := &integerReadContext{Context: action}
	region := decodedStoreRegion(t, controlled)
	left, err := region.Store([]byte("left"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := region.Store([]byte("right"))
	if err != nil {
		t.Fatal(err)
	}
	controlled.mu.Lock()
	controlled.failAt = controlled.calls + 1
	controlled.mu.Unlock()
	if actual, err := region.Precedes(left, right); actual || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled order=%v, error=%v", actual, err)
	}
}

func TestRegionCommittedSpanLengthKeepsOwnerAndLifetime(t *testing.T) {
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	region := decodedStoreRegion(t, action)
	span, err := region.Store([]byte("committed"))
	if err != nil {
		t.Fatal(err)
	}
	if size, err := region.SpanLength(span); err != nil || size != len("committed") {
		t.Fatalf("committed size=%d, error=%v", size, err)
	}
	empty, err := region.Slice(span, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if size, err := region.SpanLength(empty); err != nil || size != 0 {
		t.Fatalf("committed empty size=%d, error=%v", size, err)
	}
	foreign := decodedStoreRegion(t, nil)
	other, err := foreign.Store([]byte("foreign"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []Span{{}, other, {owner: region, length: region.used + 1}} {
		if size, err := region.SpanLength(invalid); size != 0 || err != ErrRange {
			t.Fatalf("invalid size=%d, error=%v", size, err)
		}
	}
	for _, invalid := range []*Region{nil, {self: region}} {
		if size, err := invalid.SpanLength(span); size != 0 || err != ErrConfiguration {
			t.Fatalf("invalid owner size=%d, error=%v", size, err)
		}
	}
	cancel()
	if size, err := region.SpanLength(span); size != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled size=%d, error=%v", size, err)
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if size, err := region.SpanLength(span); size != 0 || err != ErrClosed {
		t.Fatalf("closed size=%d, error=%v", size, err)
	}
}
