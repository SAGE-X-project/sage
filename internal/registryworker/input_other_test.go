//go:build !darwin

package registryworker

import (
	"errors"
	"testing"
)

func TestPreparationInputUnsupportedPlatformHasNoHeapFallback(t *testing.T) {
	if memory, err := acquirePreparationInputMemory(8); memory != nil || !errors.Is(err, ErrUnsupported) {
		t.Fatal("unsupported platform manufactured native input backing", err)
	}
}
