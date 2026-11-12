package compilermemory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRegionRejectsInvalidAcquisitionBeforeNativeWork(t *testing.T) {
	action, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), 2*time.Second)
	defer cancelCleanup()
	early, cancelEarly := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancelEarly()
	canceled, cancelCanceled := context.WithCancel(t.Context())
	cancelCanceled()
	for _, tc := range []struct {
		name            string
		action, cleanup context.Context
		capacity        int
	}{
		{name: "missing-action", cleanup: cleanup, capacity: 1},
		{name: "missing-cleanup", action: action, capacity: 1},
		{"unbounded-action", context.Background(), cleanup, 1},
		{"unbounded-cleanup", action, context.Background(), 1},
		{"cleanup-before-action", action, early, 1},
		{"canceled-action", canceled, cleanup, 1},
		{"canceled-cleanup", action, canceled, 1},
		{"zero-capacity", action, cleanup, 0},
		{"negative-capacity", action, cleanup, -1},
		{"page-rounding-overflow", action, cleanup, int(^uint(0) >> 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			region, err := Acquire(tc.action, tc.cleanup, tc.capacity)
			if region != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatalf("invalid acquisition reached native storage: region=%v err=%v", region != nil, err)
			}
		})
	}
}

func TestRegionRejectsAbsentAndDecodedOwners(t *testing.T) {
	var absent *Region
	for _, region := range []*Region{absent, {}} {
		if _, err := region.Store([]byte("data")); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned store", err)
		}
		if _, err := region.Slice(Span{}, 0, 0); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned slice", err)
		}
		if err := region.Read(Span{}, nil); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned read", err)
		}
		if _, err := region.Equal(Span{}, nil); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned comparison", err)
		}
		if err := region.Seal(); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned seal", err)
		}
		if err := region.Close(); !errors.Is(err, ErrConfiguration) {
			t.Fatal("unowned close", err)
		}
	}
	var decoded Region
	if err := json.Unmarshal([]byte(`{}`), &decoded); !errors.Is(err, ErrConfiguration) {
		t.Fatal("decoded owner", err)
	}
	var span Span
	if err := json.Unmarshal([]byte(`{}`), &span); !errors.Is(err, ErrConfiguration) {
		t.Fatal("decoded span", err)
	}
}
