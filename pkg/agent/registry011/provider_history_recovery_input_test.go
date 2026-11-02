package registry011

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historyRecoveryInputFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner) {
	t.Helper()
	o, _ := historyMemberDispatchFixture(t, 1)
	r := o.parent.recovery
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil || !r.inputReadyLocked(context.Background()) {
		t.Fatal("genuine original terminal read-only input precursor", err)
	}
	return o, r
}

func TestProviderHistoryRecoveryReadsOriginalCheckpointFilesWithSeparateObserver(t *testing.T) {
	o, r := historyRecoveryInputFixture(t)
	c := r.confirmed
	old, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, o.store.config)
	items := []struct {
		name string
		kind providerHistoryRecoveryFileKind
		want string
	}{
		{"head", providerHistoryRecoveryHead, c.head},
		{"highwater", providerHistoryRecoveryHighwater, c.head},
		{"intent", providerHistoryRecoveryIntent, c.frame.intent},
		{"commit", providerHistoryRecoveryCommit, c.frame.commit},
		{"frontier", providerHistoryRecoveryFrontier, c.frame.frontier},
		{"pending", providerHistoryRecoveryPending, c.frame.intent},
		{"inventory", providerHistoryRecoveryInventory, c.inventory.raw},
		{"object", providerHistoryRecoveryObject, c.claim.fresh[0].raw},
	}
	var total uint64
	for _, item := range items {
		t.Run(item.name, func(t *testing.T) {
			actual := r.meter.actual
			out, err := r.readCheckpointInputCandidate(context.Background(), c, item.kind, 0)
			if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || string(out.raw) != item.want || r.meter.actual[10] <= actual[10] {
				t.Fatal("selected actual file/EOF was not observed by the original recovery meter", item.name, err)
			}
			total += uint64(len(out.raw))
			out.raw[0] ^= 1 // Returned DATA cannot mutate an original/cache.
		})
	}
	if r.inputFiles != uint64(len(items)) || r.inputBytes != total || r.meter.actual[0] != 0 || r.meter.actual[1] != 0 || r.meter.actual[2] != 0 || r.meter.actual[9] != 0 ||
		o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || !o.failed || !o.meter.closed ||
		o.store.activeMeter.Load() != nil || !r.active || r.closed || r.failed || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
		t.Fatal("read-only observation renewed execution, changed originals or reset cumulative input accounting")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryInput{}) == nil {
		t.Fatal("serialized input DATA manufactured a recovery observation")
	}
}

func TestProviderHistoryRecoveryInputRefusesDetachedCheckpointAndSelectionBeforeIO(t *testing.T) {
	o, r := historyRecoveryInputFixture(t)
	c := r.confirmed
	for _, mode := range []string{"nil", "copy", "kind", "negative-index", "object-index", "initial-object"} {
		t.Run(mode, func(t *testing.T) {
			selected, kind, index := c, providerHistoryRecoveryHead, 0
			switch mode {
			case "nil":
				selected = nil
			case "copy":
				copy := *c
				copy.owner = &copy
				selected = &copy
			case "kind":
				kind = providerHistoryRecoveryFileKind(255)
			case "negative-index":
				index = -1
			case "object-index":
				kind, index = providerHistoryRecoveryObject, c.inventory.count
			case "initial-object":
				selected, kind = r.initial, providerHistoryRecoveryObject
			}
			actual, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			if out, err := r.readCheckpointInputCandidate(context.Background(), selected, kind, index); err == nil || out != nil || !r.active || r.closed ||
				r.meter.actual != actual || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("detached input selected a path, read a file or spent the lane", mode, err)
			}
		})
	}
	if out, err := r.readCheckpointInputCandidate(context.Background(), c, providerHistoryRecoveryHead, 0); err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
		t.Fatal("otherwise-valid input after isolated detached refusals", err)
	}
}

func TestProviderHistoryRecoveryInputKeepsCumulativeFileLimitWithoutRenewal(t *testing.T) {
	o, r := historyRecoveryInputFixture(t)
	old, reads, held := o.meter.actual, o.store.clock.read, o.store.stored
	for range r.bounds.maximumFiles {
		if out, err := r.readCheckpointInputCandidate(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0); err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
			t.Fatal("genuine finite cumulative input precursor", err)
		}
	}
	actual, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
	if files != r.bounds.maximumFiles || bytes == 0 {
		t.Fatal("actual prior input work was not accumulated")
	}
	if out, err := r.readCheckpointInputCandidate(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0); err == nil || out != nil || r.active || !r.closed || !r.failed ||
		r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != actual || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held {
		t.Fatal("exhausted input cap reached native reads or renewed execution", err)
	}
	if r.beginInspectionCandidate(context.Background(), o) == nil || r.inputFiles != files || r.inputBytes != bytes {
		t.Fatal("input quota refusal recreated a lane or reset its counters")
	}
}

func TestProviderHistoryRecoveryInputCancellationClosesOnlyOriginalReadOnlyLane(t *testing.T) {
	for _, mode := range []string{"nil", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			o, r := historyRecoveryInputFixture(t)
			var ctx context.Context
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			actual, old, reads, held := r.meter.actual, o.meter.actual, o.store.clock.read, o.store.stored
			if out, err := r.readCheckpointInputCandidate(ctx, r.confirmed, providerHistoryRecoveryHead, 0); err == nil || out != nil || r.active || !r.used || !r.failed || !r.closed ||
				r.meter.actual != actual || r.inputFiles != 0 || r.inputBytes != 0 || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.store.activeMeter.Load() != nil {
				t.Fatal("cancelled inspection read a file, refunded capacity or revived execution", err)
			}
		})
	}
}

func historyRecoveryPendingInputFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner) {
	t.Helper()
	o, upper := historyAdmissionWriterFixture(t)
	s := o.store
	firstSync := true
	syncFile := s.evidence.syncFile
	s.evidence.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		if firstSync {
			firstSync = false
			return errors.New("owned unit original sync uncertainty")
		}
		return nil
	}
	if out, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || out != nil || firstSync {
		t.Fatal("actual original first sync uncertainty was not reached", err)
	}
	r := o.parent.recovery
	if r.pending == nil || r.confirmed != r.initial || o.last != nil || !o.failed || r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("genuine pending-before-inventory terminal precursor")
	}
	return o, r
}

func TestProviderHistoryRecoveryInputReportsActualMissingPendingFileWithoutACK(t *testing.T) {
	o, r := historyRecoveryPendingInputFixture(t)
	old, reads, pin, held := o.meter.actual, o.store.clock.read, o.pin, o.store.stored
	out, err := r.readCheckpointInputCandidate(context.Background(), r.pending, providerHistoryRecoveryInventory, 0)
	if err != nil || out == nil || out.state != providerHistoryRecoveryMissingInput || len(out.raw) != 0 || r.inputFiles != 1 || r.inputBytes != 0 || r.meter.actual[10] == 0 ||
		r.confirmed != r.initial || o.last != nil || o.pin != pin || o.store.stored != held || o.meter.actual != old || o.store.clock.read != reads || !r.active {
		t.Fatal("missing retained file was hidden, adopted as confirmed or revived an ACK", err)
	}
	// The first original object was physically written before failed sync.
	// A separate file observation does not complete the missing inventory.
	if object, err := r.readCheckpointInputCandidate(context.Background(), r.pending, providerHistoryRecoveryObject, 0); err != nil || object == nil || object.state != providerHistoryRecoveryWholeInput ||
		r.inputFiles != 2 || r.inputBytes == 0 || r.confirmed != r.initial || o.last != nil {
		t.Fatal("partial physical publication was confused with an acknowledged prefix", err)
	}
}

func TestProviderHistoryRecoveryInputClassifiesOwnedFiniteTailFixtures(t *testing.T) {
	for _, mode := range []string{"empty", "oversize", "directory"} {
		t.Run(mode, func(t *testing.T) {
			o, r := historyRecoveryPendingInputFixture(t)
			name := providerHistoryInventoryName(r.pending.inventory.digest, nil)
			path := filepath.Join(o.store.config.evidenceDirectory, name)
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unit tail fixture must start with absent pending inventory", err)
			}
			// Unit-owned inert files only. No symlink/race, host bypass,
			// network message, shared directory or live RPC is attacked.
			switch mode {
			case "empty":
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(path, make([]byte, int(r.config.bounds.maximumMetadataPerCommit)+1), 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			want := providerHistoryRecoveryChangedInput
			if mode == "empty" {
				want = providerHistoryRecoveryPartialInput
			}
			old, reads, held, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.pin
			out, err := r.readCheckpointInputCandidate(context.Background(), r.pending, providerHistoryRecoveryInventory, 0)
			if err != nil || out == nil || out.state != want || len(out.raw) != 0 || r.inputFiles != 1 || r.inputBytes != 0 ||
				r.confirmed != r.initial || o.last != nil || o.pin != pin || o.store.stored != held || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("inert pending tail became complete or changed logical authority", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryInputEnvelopeIsFiniteAndReadOnly(t *testing.T) {
	for _, maximum := range []uint64{0, providerHistoryOriginalPackBytes + 1, ^uint64(0)} {
		if out, ok := providerHistoryRecoveryInputUpper(maximum); ok || out != (candidateWorkVector{}) {
			t.Fatal("invalid or unrepresentable file upper")
		}
	}
	for _, maximum := range []uint64{1, 8192, providerHistoryOriginalPackBytes} {
		out, ok := providerHistoryRecoveryInputUpper(maximum)
		if !ok || out[10] != 3*(maximum+1)+15 {
			t.Fatal("closed per-file read/allocate/EOF envelope", maximum)
		}
		for i, n := range out {
			if i != 10 && n != 0 {
				t.Fatal("read primitive acquired an effect/decode coordinate", i)
			}
		}
	}
}

func TestProviderHistoryRecoveryPostPathDistinguishesOperationalErrorsFromUncertainData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned-metadata")
	if err := os.WriteFile(path, []byte("inert unit metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "owned-other-metadata")
	if err := os.WriteFile(other, []byte("different inert unit metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	otherInfo, err := os.Stat(other)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name  string
		err   error
		want  providerHistoryRecoveryInputState
		fatal bool
	}{
		{"same", nil, providerHistoryRecoveryWholeInput, false},
		{"metadata-drift", nil, providerHistoryRecoveryChangedInput, false},
		{"missing-info", nil, 0, true},
		{"missing", os.ErrNotExist, providerHistoryRecoveryChangedInput, false},
		{"wrapped-missing", &os.PathError{Op: "lstat", Path: "owned unit", Err: os.ErrNotExist}, providerHistoryRecoveryChangedInput, false},
		{"permission", os.ErrPermission, 0, true},
		{"wrapped-permission", &os.PathError{Op: "lstat", Path: "owned unit", Err: os.ErrPermission}, 0, true},
		{"io", errors.New("owned unit operational I/O error"), 0, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			selected := info
			if item.err != nil {
				selected = nil // Native Lstat errors do not supply FileInfo.
			}
			if item.name == "metadata-drift" {
				selected = otherInfo
			}
			if item.name == "missing-info" {
				selected = nil
			}
			state, err := providerHistoryRecoveryPostPathState(providerHistoryRecoveryWholeInput, info, info, selected, item.err)
			if (err != nil) != item.fatal || state != item.want {
				t.Fatal("operational error became healthy DATA or path disappearance became a semantic verdict", item.name, state, err)
			}
		})
	}
}
