package app

import (
	"context"
	"errors"
	"testing"

	"github.com/sage-x-project/sage/internal/registryworker"
)

func TestPrepareRegistryHistoryRequiresExplicitWorkerConfiguration(t *testing.T) {
	out, err := PrepareRegistryHistory(context.Background(), registryworker.Configuration{}, []byte(`{}`))
	if !errors.Is(err, registryworker.ErrConfiguration) || out.Worker.Started || out.Prepared.SourceReady {
		t.Fatalf("application composition admitted unconfigured preparation: %v", err)
	}
}
