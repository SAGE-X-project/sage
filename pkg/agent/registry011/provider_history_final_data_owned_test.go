package registry011

import (
	"context"
	"errors"
	"testing"
)

// Existing recorded public responses over normal owned TLS only. These UNIT
// originals cover one deployment block, not deployment-to-current execution,
// an operating profile, Source readiness or a production derivation.
func historyFinalDataOwnedResult(t *testing.T) (*providerCustodyStore, *providerHistoryFinalResult) {
	t.Helper()
	s, in, recorded := historyCollectionRecordedRunStoreWithFinal(t, true)
	out, err := s.history.runVerifiedCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	if err != nil || out == nil {
		historyFinalFailureDiagnostics(t, s.history.final)
		t.Fatal("genuine complete original collection and final reader", err)
	}
	return s, out
}

func historyFinalDataRootsReleased(t *testing.T, s *providerCustodyStore) {
	t.Helper()
	for _, directory := range []string{s.config.custodyDirectory, s.config.evidenceDirectory} {
		disk, err := openRetentionDisk(directory)
		if err != nil {
			t.Fatal("original cleanup did not release both owned root locks", err)
		}
		if err := disk.close(); err != nil {
			t.Fatal("ordinary cleanup-check handle", err)
		}
	}
}

func TestProviderHistoryFinalDataSurvivesBothOriginalRootsClosing(t *testing.T) {
	s, out := historyFinalDataOwnedResult(t)
	p, f := s.history, s.history.final
	before, work, reads, held := out.data, f.reader.meter.actual, s.clock.read, s.stored
	if before.program != p.bounds.program || before.costs != p.bounds.costs || before.profile != s.config.profile ||
		before.manifest != p.collectionDigest || before.model != out.checkpoint.model || before.commit != out.checkpoint.pin.commit ||
		!lowerHash(before.collection) || before.selected != out.cursor.selection.selected || before.groups != 4 || before.sequence != out.checkpoint.pin.sequence ||
		before.files != out.catalog.files || before.bytes != out.catalog.bytes || before.entries != out.catalog.entries || before.names != out.catalog.names ||
		before.retention != out.checkpoint.retention || before.work != out.work || before.clockID != out.begin.ClockID || before.epoch != out.begin.Epoch ||
		before.endWall != out.end.WallMS || before.endMonotonic != out.end.MonotonicMS || before.endUncertainty != out.end.UncertaintyMS {
		t.Fatal("fixed pre-C1 DATA lost original result identity, state or accounting")
	}
	// Invalid result copies cannot consume the authentic verified store.
	for _, name := range []string{"copied-owner", "self-owned-copy", "changed-data"} {
		copy := *out
		if name != "copied-owner" {
			copy.owner = &copy
		}
		if name == "changed-data" {
			copy.data.program = "unselected"
		}
		if got, err := copy.closeAndTakeDataCandidate(context.Background()); err == nil || got != (providerHistoryFinalData{}) || s.closed ||
			f.reader.meter.actual != work || s.clock.read != reads || s.stored != held {
			t.Fatal("copied or mismatched result closed originals or published DATA")
		}
	}
	got, err := out.closeAndTakeDataCandidate(context.Background())
	if err != nil || got != before || !s.closed || !p.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil ||
		f.reader.meter.actual != work || s.clock.read != reads || s.stored != held || f.result != nil || f.preparedData != (providerHistoryFinalData{}) {
		t.Fatal("cleanup lost detached DATA, sampled time or renewed original work", err)
	}
	historyFinalDataRootsReleased(t, s)
	if again, err := out.closeAndTakeDataCandidate(context.Background()); err == nil || again != (providerHistoryFinalData{}) ||
		got != before || s.clock.read != reads || f.reader.meter.actual != work {
		t.Fatal("closed original result was published twice or invalidated prior detached DATA")
	}
}

func TestProviderHistoryFinalDataCancellationStillClosesOriginals(t *testing.T) {
	for _, name := range []string{"cancelled", "nil-context"} {
		t.Run(name, func(t *testing.T) {
			s, out := historyFinalDataOwnedResult(t)
			work, reads, held := s.history.final.reader.meter.actual, s.clock.read, s.stored
			var ctx context.Context
			if name == "cancelled" {
				cancelled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = cancelled
			}
			got, err := out.closeAndTakeDataCandidate(ctx)
			if err == nil || name == "cancelled" && !errors.Is(err, context.Canceled) || got != (providerHistoryFinalData{}) ||
				!s.closed || !s.history.failed || !s.clock.historyRevoked || s.clock.read != reads || s.stored != held || s.history.final.reader.meter.actual != work ||
				s.history.final.result != nil || s.history.final.preparedData != (providerHistoryFinalData{}) {
				t.Fatal("cancelled publication leaked DATA, retained locks or resumed execution", err)
			}
			historyFinalDataRootsReleased(t, s)
		})
	}
}

func TestProviderHistoryFinalDataWithholdsResultAfterOwnedCleanupError(t *testing.T) {
	s, out := historyFinalDataOwnedResult(t)
	work, reads, held := s.history.final.reader.meter.actual, s.clock.read, s.stored
	// Deliberately close THIS fixture's own lock descriptor after complete
	// verification. Ordinary File.Close then reports the real cleanup error;
	// no path substitution, hook, foreign resource or attacker traffic is used.
	if err := s.evidence.lock.Close(); err != nil {
		t.Fatal("owned cleanup-error prerequisite", err)
	}
	got, err := out.closeAndTakeDataCandidate(context.Background())
	if err == nil || got != (providerHistoryFinalData{}) || !s.closed || !s.history.failed || !s.clock.historyRevoked ||
		s.clock.read != reads || s.stored != held || s.history.final.reader.meter.actual != work || s.activeMeter.Load() != nil ||
		s.history.final.result != nil || s.history.final.preparedData != (providerHistoryFinalData{}) {
		t.Fatal("cleanup failure published DATA or prevented terminal role closure", err)
	}
	historyFinalDataRootsReleased(t, s)
	if again, err := out.closeAndTakeDataCandidate(context.Background()); err == nil || again != (providerHistoryFinalData{}) {
		t.Fatal("cleanup failure was retried as successful publication")
	}
}
