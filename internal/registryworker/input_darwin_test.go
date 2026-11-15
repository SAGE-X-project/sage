//go:build darwin

package registryworker

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"
)

func TestPreparationInputUsesAndReleasesItsRealLockedMapping(t *testing.T) {
	page := os.Getpagesize()
	// The two native acquisitions together use three OS pages, including the
	// overflow sentinel and page rounding. They never qualify the full profile.
	for _, maximum := range []int{page - 1, page} {
		original := strings.Repeat("x", maximum)
		wanted := sha256.Sum256([]byte(original))
		var memory *preparationInputMemory
		acquisitions, consumers := 0, 0
		acquire := func(n int) (*preparationInputMemory, error) {
			acquisitions++
			var err error
			memory, err = acquirePreparationInputMemory(n)
			return memory, err
		}
		err := consumePreparationInput(context.Background(), context.Background(), maximum, strings.NewReader(original), func(raw []byte) error {
			consumers++
			if memory == nil || len(raw) != maximum || cap(raw) != maximum || &raw[0] != &memory.data[0] || sha256.Sum256(raw) != wanted {
				t.Fatal("stdin source was not consumed from the exact native mapping")
			}
			return nil
		}, acquire)
		if err != nil {
			t.Fatal("real native acquisition or release failed", err)
		}
		if acquisitions != 1 || consumers != 1 || memory == nil || !memory.closed || memory.data != nil {
			t.Fatal("native input lifetime was incomplete")
		}
	}
}
