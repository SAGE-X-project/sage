package registryrunner

import (
	"context"
	"errors"
	"testing"

	"github.com/sage-x-project/sage/internal/registryworker"
)

func TestPrepareRequiresBothOwnedChannels(t *testing.T) {
	for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}} {
		out, err := Prepare(context.Background(), registryworker.Configuration{
			RequireJournal: flags[0], RequirePreparation: flags[1],
		}, []byte(`{}`))
		if !errors.Is(err, registryworker.ErrConfiguration) || out.Worker.Started || out.Prepared.SourceReady {
			t.Fatalf("missing owned channel accepted: flags=%v err=%v", flags, err)
		}
	}
	out, err := Prepare(context.Background(), registryworker.Configuration{RequireJournal: true, RequirePreparation: true}, []byte(`{}`))
	if !errors.Is(err, registryworker.ErrConfiguration) || out.Worker.Started || out.Worker.Output != nil {
		t.Fatal("unconfigured owner produced preparation evidence")
	}
}
