package compilermemory

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestRegionSealedSnapshotConsumesOnlyCommittedBytes(t *testing.T) {
	region := decodedStoreRegion(t, nil)
	for _, raw := range []string{"prefix:", "record"} {
		if _, err := region.Store([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := region.SealSnapshot()
	if err != nil || snapshot.owner != region || snapshot.offset != 0 || snapshot.length != 13 {
		t.Fatalf("snapshot=%+v, error=%v", snapshot, err)
	}
	field, err := region.Slice(snapshot, 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	var actual [6]byte
	if err := region.Read(field, actual[:]); err != nil || string(actual[:]) != "record" {
		t.Fatalf("snapshot field=%q, error=%v", actual, err)
	}
	if _, err := region.Slice(snapshot, 13, 1); !errors.Is(err, ErrRange) {
		t.Fatalf("snapshot exposed unused capacity: %v", err)
	}
	if _, err := region.Store([]byte("later")); err == nil {
		t.Fatal("snapshot allowed later mutation")
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if err := region.Read(field, actual[:]); !errors.Is(err, ErrClosed) {
		t.Fatalf("snapshot extended original lifetime: %v", err)
	}
}

func TestRegionSealedSnapshotRefusesInvalidOrCanceledOwners(t *testing.T) {
	var nilRegion *Region
	if span, err := nilRegion.SealSnapshot(); err == nil || span != (Span{}) {
		t.Fatalf("nil snapshot=%+v, error=%v", span, err)
	}
	region := decodedStoreRegion(t, nil)
	copied := &Region{self: region}
	if span, err := copied.SealSnapshot(); err == nil || span != (Span{}) {
		t.Fatalf("copied snapshot=%+v, error=%v", span, err)
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if span, err := region.SealSnapshot(); !errors.Is(err, ErrClosed) || span != (Span{}) {
		t.Fatalf("closed snapshot=%+v, error=%v", span, err)
	}
	var refused, completed bool
	for checks := 1; checks <= 16; checks++ {
		t.Run(strconv.Itoa(checks), func(t *testing.T) {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			controlled := &integerReadContext{Context: action}
			region := decodedStoreRegion(t, controlled)
			if _, err := region.Store([]byte("snapshot")); err != nil {
				t.Fatal(err)
			}
			controlled.mu.Lock()
			controlled.failAt = controlled.calls + checks
			controlled.mu.Unlock()
			span, err := region.SealSnapshot()
			if err != nil {
				refused = true
				if !errors.Is(err, context.Canceled) || span != (Span{}) {
					t.Fatalf("canceled snapshot=%+v, error=%v", span, err)
				}
				return
			}
			completed = true
			if span.owner != region || span.offset != 0 || span.length != 8 || !region.sealed {
				t.Fatalf("successful snapshot=%+v, sealed=%v", span, region.sealed)
			}
		})
	}
	if !refused || !completed {
		t.Fatalf("missing snapshot boundaries: refused=%v completed=%v", refused, completed)
	}
}
