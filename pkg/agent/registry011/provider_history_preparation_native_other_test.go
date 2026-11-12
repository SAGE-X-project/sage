//go:build !darwin

package registry011

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func TestProviderHistoryPreparationNativeUnsupportedNeverSelectsGoDiagnostic(t *testing.T) {
	origin := time.Now()
	action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
	defer stopCleanup()
	native, err := NewNativeHistoryPreparationSession(action, cleanup, origin, 1024, 2048)
	if native != nil || !errors.Is(err, compilermemory.ErrUnsupported) {
		t.Fatalf("native route did not refuse the unsupported platform: %v, %v", native, err)
	}
	legacy, err := NewHistoryPreparationSession(action, origin, 1024, 2048)
	if err != nil || legacy == nil || legacy.native != nil {
		t.Fatalf("explicit separate diagnostic route changed: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
}
