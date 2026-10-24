package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
)

// Read-only inspection is distinct from normal OpenReplayJournal recovery. It
// never clears refusal markers or steals LOCK; only an exact external accepted
// prefix is eligible to become an authenticated reconstruction CANDIDATE.
func retainedNames(d *journalDisk, maximum int, expected uint64) ([]string, uint8, error) {
	f, e := d.root.Open(".")
	if e != nil {
		return nil, 0, e
	}
	names, e := f.Readdirnames(maximum + 5)
	closed := f.Close()
	if (e != nil && e != io.EOF) || closed != nil || len(names) >= maximum+5 {
		return nil, 0, ErrReconfiguration
	}
	var batches []string
	var markers uint8
	for _, name := range names {
		switch name {
		case "CONFIG":
		case "LOCK", "PENDING", "QUARANTINED":
			markers |= map[string]uint8{"LOCK": 1, "PENDING": 2, "QUARANTINED": 4}[name]
			want := map[string]string{"LOCK": "locked\n", "PENDING": "pending\n", "QUARANTINED": "quarantined\n"}[name]
			b, err := d.read(name, 64)
			if err != nil || string(b) != want {
				return nil, 0, ErrReconfiguration
			}
		default:
			if !strings.HasPrefix(name, "batch-") || len(name) != 26 {
				return nil, 0, ErrReconfiguration
			}
			batches = append(batches, name)
		}
	}
	if len(batches) > maximum || uint64(len(batches)) != expected {
		return nil, 0, ErrReconfiguration
	}
	sort.Strings(batches)
	return batches, markers, nil
}

func reconstructionStep(ctx context.Context, j *ReplayJournal, entry journalEntry, raw []byte) error {
	if entry.Sequence != j.tip.Sequence+1 || entry.Sequence > uint64(j.config.MaxBatches) || entry.Previous != j.tip.Digest || int64(len(raw)) > j.config.MaxStoredBytes-j.used {
		return ErrReconfiguration
	}
	if e := j.checks.CheckReadWindow(entry.Started, entry.Completed, entry.Batch.SelectedFinalized); e != nil {
		return e
	}
	if e := j.clockOrder(entry.Started); e != nil {
		return e
	}
	next, e := j.stream.apply(ctx, j.config, j.tip, entry.Batch)
	if e != nil {
		return e
	}
	j.stream, j.used, j.lastClock = next, j.used+int64(len(raw)), entry.Completed
	j.tip = JournalCheckpoint{entry.Sequence, next.state.version, journalDigest(raw), entry.Batch.Anchor}
	return nil
}

func reconstructPrefix(ctx context.Context, directory string, previous, next JournalConfig, expected JournalCheckpoint, limits ReconstructionLimits, yield func(journalEntry) error) (JournalCheckpoint, error) {
	var none JournalCheckpoint
	if e := ctx.Err(); e != nil {
		return none, e
	}
	if expected.Sequence > uint64(limits.MaxBatches) || expected.Sequence > uint64(previous.MaxBatches) || expected.Sequence > uint64(next.MaxBatches) {
		return none, ErrReconfiguration
	}
	root, e := os.OpenRoot(directory)
	if e != nil {
		return none, e
	}
	// Root.Close releases only this read handle. It never removes another
	// process's journal lock, unlike journalDisk.close used by its owner.
	defer func() { _ = root.Close() }()
	d := &journalDisk{root: root}
	names, markers, e := retainedNames(d, limits.MaxBatches, expected.Sequence)
	if e != nil {
		return none, e
	}
	a, ac, _ := journalConfiguration(previous)
	b, bc, _ := journalConfiguration(next)
	raw, e := d.read("CONFIG", min(int64(65536), limits.MaxBytes))
	if e != nil || !bytes.Equal(raw, a) {
		return none, ErrReconfiguration
	}
	old := &ReplayJournal{config: previous, checks: ac, used: int64(len(a)), tip: JournalCheckpoint{Digest: journalDigest(a), Anchor: previous.Deployment}}
	newer := &ReplayJournal{config: next, checks: bc, used: int64(len(b)), tip: JournalCheckpoint{Digest: journalDigest(b), Anchor: next.Deployment}}
	remaining := limits.MaxBytes - int64(len(a))
	observed := make([][32]byte, 0, len(names))
	for i, name := range names {
		if e := ctx.Err(); e != nil {
			return none, e
		}
		if name != journalName(uint64(i)+1) {
			return none, ErrReconfiguration
		}
		raw, e := d.read(name, min(int64(128<<20), remaining))
		if e != nil {
			return none, e
		}
		remaining -= int64(len(raw))
		observed = append(observed, sha256.Sum256(raw))
		var entry journalEntry
		if e := journalDecode(raw, &entry, previous); e != nil {
			return none, e
		}
		if e := reconstructionStep(ctx, old, entry, raw); e != nil {
			return none, e
		}
		entry.Previous = newer.tip.Digest
		rewritten, e := json.Marshal(entry)
		if e != nil {
			return none, e
		}
		if e := reconstructionStep(ctx, newer, entry, rewritten); e != nil {
			return none, e
		}
		// Both runs consumed identical evidence and identity. Preserve full
		// private lifetime indexes as well as canonical record/grant bytes.
		or, og := old.stream.stateBytes()
		nr, ng := newer.stream.stateBytes()
		if !bytes.Equal(or, nr) || !bytes.Equal(og, ng) || !sameLifetime(old.stream, newer.stream) {
			return none, ErrReconfiguration
		}
		if yield != nil {
			if e := yield(entry); e != nil {
				return none, e
			}
		}
	}
	if old.tip != expected {
		return none, ErrReconfiguration
	}
	// Recheck the inventory/config even when a writer appended while scanning.
	finalNames, finalMarkers, e := retainedNames(d, limits.MaxBatches, expected.Sequence)
	finalConfig, f := d.read("CONFIG", 65536)
	if e != nil || f != nil || finalMarkers != markers || len(finalNames) != len(names) || !bytes.Equal(finalConfig, a) {
		return none, errors.Join(ErrReconfiguration, e, f)
	}
	// Counts alone do not identify a prefix. Re-read every exact file and
	// compare its bytes, including changes after the initial replay. Each
	// complete inspection pass has the same explicit prefix-byte ceiling.
	remaining = limits.MaxBytes - int64(len(a))
	for i, name := range names {
		if e := ctx.Err(); e != nil {
			return none, e
		}
		if finalNames[i] != name {
			return none, ErrReconfiguration
		}
		raw, e := d.read(name, min(int64(128<<20), remaining))
		if e != nil || sha256.Sum256(raw) != observed[i] {
			return none, errors.Join(ErrReconfiguration, e)
		}
		remaining -= int64(len(raw))
	}
	if e := ctx.Err(); e != nil {
		return none, e
	}
	return newer.tip, nil
}

func sameLifetime(a, b replayStream) bool {
	if len(a.claims) != len(b.claims) || len(a.created) != len(b.created) || a.state.admitted != b.state.admitted {
		return false
	}
	for k, v := range a.claims {
		if w, ok := b.claims[k]; !ok || v != w {
			return false
		}
	}
	for k, v := range a.created {
		if w, ok := b.created[k]; !ok || v != w {
			return false
		}
	}
	return true
}
