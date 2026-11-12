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
