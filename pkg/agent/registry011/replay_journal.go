package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"
)

var ErrJournal = errors.New("registry011: journal unavailable or quarantined")

// JournalConfig is explicit local storage/work configuration, not Source
// enrollment or operating-policy approval. CodeIdentity/ABIIdentity bind the
// separately qualified full identities; a supplied hash cannot qualify them.
// The directory, external checkpoint retention and clock must be trusted.
type JournalConfig struct {
	Registry, Agent, GenesisHash, CodeIdentity, ABIIdentity string
	SourceID, PolicyGeneration                              string
	Deployment                                              FinalizedHeader
	Replay                                                  ReplayLimits
	Observation                                             ObservationLimits
	MaxBatches, MaxClaims, MaxIdentities                    int
	MaxStoredBytes                                          int64
}

// JournalCheckpoint must be independently durably retained before publication
// and supplied exactly on restart. A hash chain alone cannot detect whole-store
// rollback by a writer of the trusted directory or authenticate an external ack.
type JournalCheckpoint struct {
	Sequence, Version uint64
	Digest            string
	Anchor            FinalizedHeader
}

type journalEntry struct {
	Sequence           uint64
	Previous           string
	Started, Completed ClockSample
	Batch              ReplayBatch
}

// ReplayJournal retains complete batches, not an accepted state snapshot.
// Open replays all batches; only per-batch work counters reset. Recovery never
// restores publishability or a previous process's clock epoch. There is no
// automatic repair, reset, lock stealing, quarantine clearing or reconfiguration.
// No method returns a Source-ready or authenticated-absence credential.
type ReplayJournal struct {
	mu        sync.Mutex
	disk      *journalDisk
	config    JournalConfig
	checks    *ObservationChecks
	stream    replayStream
	tip       JournalCheckpoint
	used      int64
	poisoned  bool
	confirmed bool
	start     ClockSample
	completed ClockSample
	lastClock ClockSample
}

func journalConfiguration(c JournalConfig) ([]byte, *ObservationChecks, error) {
	checks, err := NewObservationChecks(c.Observation)
	if err != nil || c.Registry != "eip155:11155111:"+SepoliaRegistryAddress || !rpcAgent(c.Agent) || c.GenesisHash != sepoliaGenesis || !lowerHash(c.CodeIdentity) || !lowerHash(c.ABIIdentity) ||
		len(c.SourceID) == 0 || len(c.SourceID) > 256 || len(c.PolicyGeneration) == 0 || len(c.PolicyGeneration) > 256 || c.PolicyGeneration != c.Observation.PolicyGeneration || len(c.Observation.ClockID) > 256 || !validHeader(c.Deployment) || !replayLimits(c.Replay) ||
		!utf8.ValidString(c.SourceID) || !utf8.ValidString(c.PolicyGeneration) || !utf8.ValidString(c.Observation.ClockID) ||
		c.Observation.MaxAncestryHeaders > 1<<20 || c.MaxBatches <= 0 || c.MaxBatches > 1<<20 || c.MaxClaims <= 0 || c.MaxClaims > 1<<20 || c.MaxIdentities <= 0 || c.MaxIdentities > 1<<20 || c.MaxStoredBytes <= 0 || c.MaxStoredBytes > 1<<40 {
		return nil, nil, ErrConfiguration
	}
	b, err := json.Marshal(struct {
		Format string
		Config JournalConfig
	}{"sage-replay-journal-v1", c})
	if err != nil || int64(len(b)) > c.MaxStoredBytes {
		return nil, nil, ErrConfiguration
	}
	return b, checks, nil
}

// CreateReplayJournal requires an existing EMPTY directory. Creation is an
// explicit local action, not initial trust enrollment. Retain its sequence-zero
// checkpoint externally too; it has no observation to publish.
func CreateReplayJournal(ctx context.Context, directory string, config JournalConfig) (*ReplayJournal, error) {
	return openReplayJournal(ctx, directory, config, nil)
}

// OpenReplayJournal never creates missing history. The exact external checkpoint
// rejects both older prefixes and unacknowledged newer tails, without truncation.
func OpenReplayJournal(ctx context.Context, directory string, config JournalConfig, expected JournalCheckpoint) (*ReplayJournal, error) {
	return openReplayJournal(ctx, directory, config, &expected)
}

func openReplayJournal(ctx context.Context, directory string, config JournalConfig, expected *JournalCheckpoint) (*ReplayJournal, error) {
	raw, checks, err := journalConfiguration(config)
	if err != nil || ctx == nil {
		return nil, ErrConfiguration
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	disk, err := openJournalDisk(directory, expected == nil)
	if err != nil {
		return nil, err
	}
	j := &ReplayJournal{disk: disk, config: config, checks: checks, used: int64(len(raw)), tip: JournalCheckpoint{Digest: journalDigest(raw), Anchor: config.Deployment}}
	if expected == nil {
		err = disk.write("CONFIG", raw)
	} else {
		var stored []byte
		stored, err = disk.read("CONFIG", 65536)
		if err == nil && !bytes.Equal(stored, raw) {
			err = ErrConfiguration
		}
		if err == nil {
			err = j.recover(ctx, *expected)
		}
	}
	if err != nil {
		return nil, errors.Join(j.fail(err), j.Close())
	}
	return j, nil
}

func journalDigest(raw []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte("sage-replay-journal-v1\x00"))
	_, _ = h.Write(raw)
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func journalName(sequence uint64) string { return fmt.Sprintf("batch-%020d", sequence) }

func (j *ReplayJournal) recover(ctx context.Context, expected JournalCheckpoint) error {
	names, err := j.disk.batches(j.config.MaxBatches)
	if err != nil {
		return err
	}
	if uint64(len(names)) != expected.Sequence {
		return ErrJournal
	}
	for i, name := range names {
		if name != journalName(uint64(i)+1) {
			return ErrJournal
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, err := j.disk.read(name, min(int64(128<<20), j.config.MaxStoredBytes-j.used))
		if err != nil {
			return err
		}
		var entry journalEntry
		if err = journalDecode(raw, &entry, j.config); err != nil {
			return err
		}
		if entry.Sequence != j.tip.Sequence+1 || entry.Previous != j.tip.Digest {
			return ErrJournal
		}
		if err = j.checks.CheckReadWindow(entry.Started, entry.Completed, entry.Batch.SelectedFinalized); err != nil {
			return err
		}
		next, err := j.stream.apply(ctx, j.config, j.tip, entry.Batch)
		if err != nil {
			return err
		}
		j.stream, j.used = next, j.used+int64(len(raw))
		if err := j.clockOrder(entry.Started); err != nil {
			return err
		}
		j.lastClock = entry.Completed
		j.tip = JournalCheckpoint{entry.Sequence, next.state.version, journalDigest(raw), entry.Batch.Anchor}
	}
	if j.tip != expected {
		return ErrJournal
	}
	return nil
}

// Append commits an all-or-nothing, contiguous replay batch. Its checkpoint is
// not a publication acknowledgement. On any admitted operation failure the
// journal becomes unusable and persists quarantine if storage permits. PENDING
// remains across crashes or uncertain writes; recovery cannot adopt a prefix.
// Input buffers must not be concurrently mutated; no buffers are retained.
func (j *ReplayJournal) Append(ctx context.Context, batch ReplayBatch, start, completed ClockSample) (JournalCheckpoint, error) {
	if j == nil || ctx == nil {
		return JournalCheckpoint{}, ErrConfiguration
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.disk == nil || j.poisoned {
		return JournalCheckpoint{}, ErrJournal
	}
	j.confirmed = false
	if err := j.disk.write("PENDING", []byte("pending\n")); err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	if j.tip.Sequence >= uint64(j.config.MaxBatches) {
		return JournalCheckpoint{}, j.fail(ErrReplayUnavailable)
	}
	if err := j.checks.CheckReadWindow(start, completed, batch.SelectedFinalized); err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	if err := j.clockOrder(start); err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	next, err := j.stream.apply(ctx, j.config, j.tip, batch)
	if err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	entry := journalEntry{j.tip.Sequence + 1, j.tip.Digest, start, completed, batch}
	raw, err := json.Marshal(entry)
	if err != nil || len(raw) > 128<<20 || int64(len(raw)) > j.config.MaxStoredBytes-j.used {
		return JournalCheckpoint{}, j.fail(ErrReplayUnavailable)
	}
	if err = j.disk.write(journalName(entry.Sequence), raw); err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	if err = j.disk.remove("PENDING"); err != nil {
		return JournalCheckpoint{}, j.fail(err)
	}
	j.tip = JournalCheckpoint{entry.Sequence, next.state.version, journalDigest(raw), batch.Anchor}
	j.stream, j.used = next, j.used+int64(len(raw))
	j.lastClock = completed
	// A committed cursor is not a newly confirmed observation. Require fresh,
	// separately qualified pinned reads after the local commit, including on
	// the first run; do not refresh old bytes by merely replacing their clock.
	j.confirmed = false
	return j.tip, nil
}

// ConfirmObservation joins fresh separately qualified pinned reads to the exact
// durable cursor. Backfill behind SelectedFinalized cannot publish. Recovery
// requires new samples, never the historical start restored from a file.
func (j *ReplayJournal) ConfirmObservation(start, completed ClockSample, anchor FinalizedHeader, record, grants []byte) error {
	if j == nil {
		return ErrConfiguration
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.disk == nil || j.poisoned {
		return ErrJournal
	}
	j.confirmed = false
	if j.tip.Sequence == 0 || anchor != j.tip.Anchor || anchor != j.stream.selected || !j.stream.matches(record, grants) {
		return j.fail(ErrJournal)
	}
	if err := j.checks.CheckReadWindow(start, completed, anchor); err != nil {
		return j.fail(err)
	}
	if err := j.clockOrder(start); err != nil {
		return j.fail(err)
	}
	j.start, j.completed, j.lastClock, j.confirmed = start, completed, completed, true
	return nil
}

// Publish returns diagnostic state copies only after exact trusted external
// checkpoint readback and a post-commit freshness check using the original read
// start. The caller must qualify the ack's retention, fresh reads, node/ancestry,
// code, finality, local storage, clock and host independently. No SourceReady.
func (j *ReplayJournal) Publish(ack JournalCheckpoint, published ClockSample) (record, grants []byte, err error) {
	if j == nil {
		return nil, nil, ErrConfiguration
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.disk == nil || j.poisoned || !j.confirmed {
		return nil, nil, ErrJournal
	}
	if ack != j.tip || j.stream.state.version == 0 {
		return nil, nil, j.fail(ErrJournal)
	}
	if published.WallMS < j.completed.WallMS || published.MonotonicMS < j.completed.MonotonicMS {
		return nil, nil, j.fail(ErrClock)
	}
	if err := j.checks.CheckPublication(j.start, published, j.tip.Anchor); err != nil {
		return nil, nil, j.fail(err)
	}
	// Retain the exact schema's canonical bytes, never Go's member ordering.
	record, grants = j.stream.stateBytes()
	return record, grants, nil
}

func (j *ReplayJournal) clockOrder(next ClockSample) error {
	if j.lastClock.Epoch != "" && (next.WallMS < j.lastClock.WallMS || (next.Epoch == j.lastClock.Epoch && next.MonotonicMS < j.lastClock.MonotonicMS)) {
		return ErrClock
	}
	return nil
}

func (j *ReplayJournal) Checkpoint() (JournalCheckpoint, error) {
	if j == nil {
		return JournalCheckpoint{}, ErrConfiguration
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.disk == nil || j.poisoned {
		return JournalCheckpoint{}, ErrJournal
	}
	return j.tip, nil
}

// fail is called with the lock held (or during private construction). Failure
// to persist quarantine is itself returned, never hidden as successful recovery.
func (j *ReplayJournal) fail(err error) error {
	j.poisoned, j.confirmed = true, false
	return errors.Join(ErrJournal, err, j.disk.write("QUARANTINED", []byte("quarantined\n")))
}

func (j *ReplayJournal) Close() error {
	if j == nil {
		return ErrConfiguration
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.disk == nil {
		return ErrJournal
	}
	d := j.disk
	j.disk = nil
	j.confirmed = false
	return d.close()
}
