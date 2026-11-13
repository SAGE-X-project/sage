//go:build darwin

package compilermemory

import (
	"context"
	"testing"
	"time"
)

func TestRegionNativeDecodedStringStoreRetainsActualCopiedBytes(t *testing.T) {
	action, cancelAction := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelAction()
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelCleanup()
	raw := []byte(`"owned \uD83D\uDE00 \u0000 bytes"`)
	limits := JSONLimits{MaximumDepth: 2, MaximumNodes: 1}
	capacity, err := JSONCapacity(len(raw), limits)
	if err != nil {
		t.Fatal(err)
	}
	source, err := Acquire(action, cleanup, capacity)
	if source != nil {
		defer func() {
			if err := source.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	if err != nil {
		t.Fatal(err)
	}
	destination, err := Acquire(action, cleanup, 4096)
	if destination != nil {
		defer func() {
			if err := destination.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	if err != nil {
		t.Fatal(err)
	}
	input, err := source.Store(raw)
	if err != nil {
		t.Fatal(err)
	}
	document, err := source.ParseJSON(input, limits)
	if err != nil {
		t.Fatal(err)
	}
	span, err := document.Root().StoreDecodedString(destination)
	if err != nil {
		t.Fatal(err)
	}
	clear(raw)
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := destination.SealSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	want := "owned 😀 \x00 bytes"
	retained, err := destination.Slice(snapshot, 0, len(want))
	if err != nil || retained != span {
		t.Fatalf("retained snapshot span=%+v, error=%v; copied=%+v", retained, err, span)
	}
	actual := make([]byte, len(want))
	if err := destination.Read(retained, actual); err != nil || string(actual) != want {
		t.Fatalf("actual retained native bytes=%q, error=%v; want=%q", actual, err, want)
	}
}
