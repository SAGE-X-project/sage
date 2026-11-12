//go:build !darwin

package compilermemory

import (
	"errors"
	"testing"
)

func TestUnsupportedRegionNeverFallsBackToHeapStorage(t *testing.T) {
	action, cleanup, _, _ := simulatedRegionContexts(t)
	region, err := Acquire(action, cleanup, 64)
	if region != nil || !errors.Is(err, ErrUnsupported) || !errors.Is(err, ErrUnavailable) {
		t.Fatal("unsupported acquisition was not refused", region != nil, err)
	}
	ops := nativeRegionOperations{}
	for _, err := range []error{ops.lockBytes(nil), ops.unlockBytes(nil), ops.unmapBytes(nil)} {
		if !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported primitive used a fallback", err)
		}
	}
}
