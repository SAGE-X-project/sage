package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// All scenarios use owned temporary directories and inert, offline synthetic
// block/receipt/trace data from the existing handwritten 11-transition fixture.
// No RPC, operator key, transaction broadcast, attack producer or local VM.
func journalTestHeader(t *testing.T, b ReplayBlock) FinalizedHeader {
	t.Helper()
	var h types.Header
	if err := rlp.DecodeBytes(b.HeaderRLP, &h); err != nil {
		t.Fatal(err)
	}
	return replayHeader(&h)
}
func journalTestConfig(t *testing.T, in ReplayInput) JournalConfig {
	t.Helper()
	o := testLimits()
	o.MaxAcquisitionMS = 100
	o.MaxObservationAgeMS = 200
	o.MaxFinalizedLagMS = 1000
	o.MaxAncestryHeaders = 10000
	return JournalConfig{Registry: in.Registry, Agent: in.Agent, GenesisHash: sepoliaGenesis, CodeIdentity: "0x" + strings.Repeat("1", 64), ABIIdentity: "0x" + strings.Repeat("2", 64), SourceID: "owned-test-source", PolicyGeneration: o.PolicyGeneration, Deployment: journalTestHeader(t, in.Blocks[0]), Observation: o, Replay: replayTestLimits(), MaxBatches: 10000, MaxStoredBytes: 64 << 20, MaxClaims: 1000, MaxIdentities: 1000}
}
func journalTestBatch(t *testing.T, in ReplayInput, first, end int) ReplayBatch {
	t.Helper()
	b := ReplayBatch{SelectedFinalized: in.Anchor, Anchor: journalTestHeader(t, in.Blocks[end]), Blocks: in.Blocks[first : end+1]}
	base := max(0, first-1)
	for i := len(in.Blocks) - 2; i >= base; i-- {
		b.Parents = append(b.Parents, journalTestHeader(t, in.Blocks[i]))
	}
	for i := max(1, first); i <= end; i++ {
		b.History = append(b.History, in.History[i-1])
	}
	if end > 0 {
		steps := replayTestSteps(t)
		b.Record = steps[end-1].Record
		b.Grants = steps[end-1].Grants
	}
	return b
}
func journalTestClock(h FinalizedHeader, offset int64) ClockSample {
	return ClockSample{ClockID: "owned-clock", Epoch: "first-process", WallMS: int64(h.Timestamp)*1000 + offset, MonotonicMS: 10000 + offset}
}
func journalTestAppend(t *testing.T, j *ReplayJournal, b ReplayBatch, n int64) JournalCheckpoint {
	t.Helper()
	tip, e := j.Append(context.Background(), b, journalTestClock(b.SelectedFinalized, n), journalTestClock(b.SelectedFinalized, n+1))
	if e != nil {
		t.Fatal(e)
	}
	if b.Anchor == b.SelectedFinalized && j.stream.state.version > 0 {
		// A distinct fresh read is modelled here by the handwritten final bytes
		// at exactly this anchor. This offline fixture is not live RPC evidence.
		if e := j.ConfirmObservation(journalTestClock(b.SelectedFinalized, n+1), journalTestClock(b.SelectedFinalized, n+2), b.Anchor, b.Record, b.Grants); e != nil {
			t.Fatal(e)
		}
	}
	return tip
}
func journalTestCreate(t *testing.T, c JournalConfig) (*ReplayJournal, string) {
	t.Helper()
	dir := t.TempDir()
	j, e := CreateReplayJournal(context.Background(), dir, c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if j.disk != nil {
			_ = j.Close()
		}
	})
	return j, dir
}

func TestJournalIncrementalAllTransitionsAndFreshRestart(t *testing.T) {
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	j, dir := journalTestCreate(t, c)
	zero, e := j.Checkpoint()
	if e != nil || zero.Sequence != 0 || zero.Version != 0 || zero.Anchor != c.Deployment {
		t.Fatal(zero, e)
	}
	if _, _, e = j.Publish(zero, journalTestClock(in.Anchor, 0)); e == nil {
		t.Fatal("configuration is not publishable")
	}
	for _, bounds := range [][2]int{{0, 0}, {1, 3}, {4, 6}, {7, 9}} {
		tip := journalTestAppend(t, j, journalTestBatch(t, in, bounds[0], bounds[1]), int64(bounds[1]*3))
		if _, _, e = j.Publish(tip, journalTestClock(in.Anchor, int64(bounds[1]*3+2))); e == nil {
			t.Fatal("backfill published")
		}
	}
	tip := journalTestAppend(t, j, journalTestBatch(t, in, 10, 11), 40)
	r, g, e := j.Publish(tip, journalTestClock(in.Anchor, 42))
	if e != nil || !bytes.Equal(r, in.Record) || !bytes.Equal(g, in.Grants) {
		t.Fatal(e)
	}
	r[0] = 'x'
	g[0] = 'x'
	r, g, e = j.Publish(tip, journalTestClock(in.Anchor, 43))
	if e != nil || !bytes.Equal(r, in.Record) || !bytes.Equal(g, in.Grants) {
		t.Fatal("mutable output", e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e = OpenReplayJournal(context.Background(), dir, c, tip)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if _, _, e = j.Publish(tip, journalTestClock(in.Anchor, 44)); e == nil {
		t.Fatal("restart inherited publication")
	}
	s, eclock := journalTestClock(in.Anchor, 50), journalTestClock(in.Anchor, 51)
	s.Epoch = "second-process"
	eclock.Epoch = s.Epoch
	s.MonotonicMS = 1
	eclock.MonotonicMS = 2
	if e = j.ConfirmObservation(s, eclock, in.Anchor, in.Record, in.Grants); e != nil {
		t.Fatal(e)
	}
	p := eclock
	p.WallMS++
	p.MonotonicMS++
	r, g, e = j.Publish(tip, p)
	if e != nil || !bytes.Equal(r, in.Record) || !bytes.Equal(g, in.Grants) {
		t.Fatal(e)
	}
	if j.stream.state.version != 11 || j.stream.state.record.State != "deactivated" || len(j.stream.created) != 1 || len(j.stream.claims) != 1 {
		t.Fatal("lost lifetime state")
	}
	for _, k := range j.stream.state.record.Keys {
		if k.Name == "a" && k.State != "revoked" {
			t.Fatal("lost key tombstone")
		}
	}
}

func TestJournalConfigurationAndZeroValues(t *testing.T) {
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	for name, edit := range map[string]func(*JournalConfig){
		"registry": func(c *JournalConfig) { c.Registry = "other" }, "agent": func(c *JournalConfig) { c.Agent = "" }, "genesis": func(c *JournalConfig) { c.GenesisHash = "0x0" }, "code": func(c *JournalConfig) { c.CodeIdentity = "" }, "abi": func(c *JournalConfig) { c.ABIIdentity = "" }, "source": func(c *JournalConfig) { c.SourceID = "" }, "source-large": func(c *JournalConfig) { c.SourceID = strings.Repeat("s", 257) }, "utf8": func(c *JournalConfig) { c.SourceID = "\xff" }, "generation": func(c *JournalConfig) { c.PolicyGeneration = "other" }, "clock": func(c *JournalConfig) { c.Observation.ClockID = "" }, "clock-large": func(c *JournalConfig) { c.Observation.ClockID = strings.Repeat("c", 257) }, "deployment": func(c *JournalConfig) { c.Deployment.Hash = "bad" }, "replay": func(c *JournalConfig) { c.Replay.MaxBlocks = 0 }, "ancestry": func(c *JournalConfig) { c.Observation.MaxAncestryHeaders = 1<<20 + 1 }, "batch-zero": func(c *JournalConfig) { c.MaxBatches = 0 }, "batch-large": func(c *JournalConfig) { c.MaxBatches = 1<<20 + 1 }, "claim-zero": func(c *JournalConfig) { c.MaxClaims = 0 }, "claim-large": func(c *JournalConfig) { c.MaxClaims = 1<<20 + 1 }, "identity-zero": func(c *JournalConfig) { c.MaxIdentities = 0 }, "identity-large": func(c *JournalConfig) { c.MaxIdentities = 1<<20 + 1 }, "disk-zero": func(c *JournalConfig) { c.MaxStoredBytes = 0 }, "disk-large": func(c *JournalConfig) { c.MaxStoredBytes = 1<<40 + 1 }, "config-budget": func(c *JournalConfig) { c.MaxStoredBytes = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			x := c
			edit(&x)
			if _, e := CreateReplayJournal(context.Background(), t.TempDir(), x); !errors.Is(e, ErrConfiguration) {
				t.Fatal(e)
			}
		})
	}
	if _, e := CreateReplayJournal(nil, t.TempDir(), c); !errors.Is(e, ErrConfiguration) { //nolint:staticcheck // Verify refusal of an explicitly missing context.
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := CreateReplayJournal(ctx, t.TempDir(), c); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	for _, j := range []*ReplayJournal{nil, {}} {
		if _, e := j.Checkpoint(); e == nil {
			t.Fatal("zero checkpoint")
		}
		if _, e := j.Append(context.Background(), ReplayBatch{}, ClockSample{}, ClockSample{}); e == nil {
			t.Fatal("zero append")
		}
		if e := j.ConfirmObservation(ClockSample{}, ClockSample{}, FinalizedHeader{}, nil, nil); e == nil {
			t.Fatal("zero confirm")
		}
		if _, _, e := j.Publish(JournalCheckpoint{}, ClockSample{}); e == nil {
			t.Fatal("zero publish")
		}
		if e := j.Close(); e == nil {
			t.Fatal("zero close")
		}
	}
	j, _ := journalTestCreate(t, c)
	if _, e := j.Append(nil, ReplayBatch{}, ClockSample{}, ClockSample{}); !errors.Is(e, ErrConfiguration) { //nolint:staticcheck // Verify refusal of an explicitly missing context.
		t.Fatal(e)
	}
}

func TestJournalRefusalPersistsQuarantineAndNoPartialState(t *testing.T) {
	for name, edit := range map[string]func(*ReplayBatch){
		"missing-first": func(b *ReplayBatch) { b.Blocks = b.Blocks[1:] }, "missing-middle": func(b *ReplayBatch) { b.Blocks = append(b.Blocks[:2:2], b.Blocks[3:]...) }, "bad-second-body": func(b *ReplayBatch) { b.Blocks[3].HeaderRLP = []byte{0} }, "missing-history": func(b *ReplayBatch) { b.History = b.History[:len(b.History)-1] }, "wrong-version": func(b *ReplayBatch) { b.History[1].Version = 1 }, "wrong-command": func(b *ReplayBatch) { b.History[2].Command = []byte("{}") }, "wrong-record": func(b *ReplayBatch) { b.Record = []byte("{}") }, "wrong-grants": func(b *ReplayBatch) { b.Grants = []byte("[] ") }, "missing-parent": func(b *ReplayBatch) { b.Parents = b.Parents[:len(b.Parents)-1] }, "fork-parent": func(b *ReplayBatch) { b.Parents[0].Hash = "0x" + strings.Repeat("f", 64) }, "anchor": func(b *ReplayBatch) { b.Anchor.ParentHash = "0x" + strings.Repeat("f", 64) }, "selected": func(b *ReplayBatch) { b.SelectedFinalized.Timestamp++ }, "empty-blocks": func(b *ReplayBatch) { b.Blocks = nil }, "empty-entry": func(b *ReplayBatch) { b.History[0].Entry = nil }, "empty-command": func(b *ReplayBatch) { b.History[0].Command = nil },
	} {
		t.Run(name, func(t *testing.T) {
			in := replayTestInput(t)
			c := journalTestConfig(t, in)
			j, dir := journalTestCreate(t, c)
			before, _ := j.Checkpoint()
			b := replayTestClone(t, journalTestBatch(t, in, 0, 11))
			edit(&b)
			if _, e := j.Append(context.Background(), b, journalTestClock(in.Anchor, 0), journalTestClock(in.Anchor, 1)); e == nil {
				t.Fatal("accepted")
			}
			if j.tip != before || j.stream.state.version != 0 {
				t.Fatal("partial state")
			}
			if _, e := j.Checkpoint(); !errors.Is(e, ErrJournal) {
				t.Fatal(e)
			}
			if _, e := j.Append(context.Background(), b, ClockSample{}, ClockSample{}); !errors.Is(e, ErrJournal) {
				t.Fatal(e)
			}
			if e := j.ConfirmObservation(ClockSample{}, ClockSample{}, FinalizedHeader{}, nil, nil); !errors.Is(e, ErrJournal) {
				t.Fatal(e)
			}
			if e := j.Close(); e != nil {
				t.Fatal(e)
			}
			if _, e := OpenReplayJournal(context.Background(), dir, c, before); e == nil {
				t.Fatal("quarantine cleared")
			}
			if _, e := os.Stat(filepath.Join(dir, "QUARANTINED")); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestJournalRestartRejectsLostOrChangedData(t *testing.T) {
	for name, edit := range map[string]func(string, *JournalCheckpoint, *JournalConfig){
		"missing-config": func(d string, _ *JournalCheckpoint, _ *JournalConfig) { _ = os.Remove(filepath.Join(d, "CONFIG")) }, "changed-config": func(d string, _ *JournalCheckpoint, _ *JournalConfig) {
			_ = os.WriteFile(filepath.Join(d, "CONFIG"), []byte("{}"), 0600)
		}, "generation": func(_ string, _ *JournalCheckpoint, c *JournalConfig) {
			c.PolicyGeneration = "new"
			c.Observation.PolicyGeneration = "new"
		}, "missing-first-batch": func(d string, _ *JournalCheckpoint, _ *JournalConfig) {
			_ = os.Remove(filepath.Join(d, journalName(1)))
		}, "truncated-tail": func(d string, _ *JournalCheckpoint, _ *JournalConfig) {
			_ = os.WriteFile(filepath.Join(d, journalName(2)), []byte("{"), 0600)
		}, "pending": func(d string, _ *JournalCheckpoint, _ *JournalConfig) {
			_ = os.WriteFile(filepath.Join(d, "PENDING"), []byte("pending"), 0600)
		}, "unknown-file": func(d string, _ *JournalCheckpoint, _ *JournalConfig) {
			_ = os.WriteFile(filepath.Join(d, "unexpected"), []byte("x"), 0600)
		}, "wrong-digest": func(_ string, p *JournalCheckpoint, _ *JournalConfig) { p.Digest = "0x" + strings.Repeat("f", 64) }, "wrong-version": func(_ string, p *JournalCheckpoint, _ *JournalConfig) { p.Version-- }, "wrong-anchor": func(_ string, p *JournalCheckpoint, _ *JournalConfig) { p.Anchor.Timestamp++ }, "old-ack": func(_ string, p *JournalCheckpoint, _ *JournalConfig) { p.Sequence-- }, "future-ack": func(_ string, p *JournalCheckpoint, _ *JournalConfig) { p.Sequence++ }, "batch-budget": func(_ string, _ *JournalCheckpoint, c *JournalConfig) { c.MaxBatches = 1 }, "disk-budget": func(_ string, _ *JournalCheckpoint, c *JournalConfig) { c.MaxStoredBytes = 2000 },
	} {
		t.Run(name, func(t *testing.T) {
			in := replayTestInput(t)
			c := journalTestConfig(t, in)
			j, dir := journalTestCreate(t, c)
			journalTestAppend(t, j, journalTestBatch(t, in, 0, 5), 0)
			tip := journalTestAppend(t, j, journalTestBatch(t, in, 6, 11), 2)
			if e := j.Close(); e != nil {
				t.Fatal(e)
			}
			edit(dir, &tip, &c)
			if _, e := OpenReplayJournal(context.Background(), dir, c, tip); e == nil {
				t.Fatal("repaired or accepted changed journal")
			}
		})
	}
}

func TestJournalPublicationBoundaries(t *testing.T) {
	for _, name := range []string{"ack-sequence", "ack-digest", "ack-version", "ack-anchor", "publication-stale", "publication-before-completed", "publication-epoch", "fresh-confirm-wrong-record", "fresh-confirm-wrong-anchor", "fresh-confirm-stale", "fresh-confirm-rollback"} {
		t.Run(name, func(t *testing.T) {
			in := replayTestInput(t)
			j, _ := journalTestCreate(t, journalTestConfig(t, in))
			tip := journalTestAppend(t, j, journalTestBatch(t, in, 0, 11), 2)
			p := journalTestClock(in.Anchor, 4)
			var e error
			switch name {
			case "ack-sequence":
				tip.Sequence--
			case "ack-digest":
				tip.Digest = "bad"
			case "ack-version":
				tip.Version--
			case "ack-anchor":
				tip.Anchor.Timestamp++
			case "publication-stale":
				p.WallMS += 300
				p.MonotonicMS += 300
			case "publication-before-completed":
				p = journalTestClock(in.Anchor, 2)
			case "publication-epoch":
				p.Epoch = "restarted"
			case "fresh-confirm-wrong-record":
				e = j.ConfirmObservation(journalTestClock(in.Anchor, 5), journalTestClock(in.Anchor, 6), in.Anchor, []byte("{}"), in.Grants)
			case "fresh-confirm-wrong-anchor":
				h := in.Anchor
				h.Timestamp++
				e = j.ConfirmObservation(journalTestClock(in.Anchor, 5), journalTestClock(in.Anchor, 6), h, in.Record, in.Grants)
			case "fresh-confirm-stale":
				e = j.ConfirmObservation(journalTestClock(in.Anchor, 5), journalTestClock(in.Anchor, 200), in.Anchor, in.Record, in.Grants)
			case "fresh-confirm-rollback":
				e = j.ConfirmObservation(journalTestClock(in.Anchor, 0), journalTestClock(in.Anchor, 1), in.Anchor, in.Record, in.Grants)
			}
			if !strings.HasPrefix(name, "fresh-confirm-") {
				_, _, e = j.Publish(tip, p)
			}
			if e == nil {
				t.Fatal("accepted")
			}
			if _, e = j.Checkpoint(); e == nil {
				t.Fatal("quarantine did not stick")
			}
		})
	}
}

func TestJournalOwnedDiskFailuresAndExclusiveLock(t *testing.T) {
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	j, dir := journalTestCreate(t, c)
	tip, _ := j.Checkpoint()
	if _, e := OpenReplayJournal(context.Background(), dir, c, tip); e == nil {
		t.Fatal("stole lock")
	}
	if _, e := CreateReplayJournal(context.Background(), dir, c); e == nil {
		t.Fatal("reset nonempty directory")
	}
	if _, e := OpenReplayJournal(context.Background(), filepath.Join(t.TempDir(), "missing"), c, tip); e == nil {
		t.Fatal("created missing directory")
	}
	if e := j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e := OpenReplayJournal(context.Background(), dir, c, tip)
	if e != nil {
		t.Fatal(e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	for _, failAt := range []int{1, 3, 5} {
		t.Run("sync-"+string(rune('0'+failAt)), func(t *testing.T) {
			j, dir := journalTestCreate(t, c)
			before, _ := j.Checkpoint()
			calls := 0
			j.disk.syncFile = func(f *os.File) error {
				calls++
				if calls == failAt {
					return errors.New("owned injected sync failure")
				}
				return f.Sync()
			}
			_, e := j.Append(context.Background(), journalTestBatch(t, in, 0, 11), journalTestClock(in.Anchor, 0), journalTestClock(in.Anchor, 1))
			if e == nil || j.tip != before {
				t.Fatal("uncertain write committed", e)
			}
			j.disk.syncFile = (*os.File).Sync
			_ = j.Close()
			if _, e = OpenReplayJournal(context.Background(), dir, c, before); e == nil {
				t.Fatal("uncertain commit adopted")
			}
		})
	}
}

func TestJournalLocalEncodingRefusal(t *testing.T) {
	for _, raw := range []string{"{} ", "{\"Sequence\":1,\"Sequence\":2}", "{\"Extra\":1}", "[]", "{", "}", "\"open", strings.Repeat("[", 97) + strings.Repeat("]", 97)} {
		var e journalEntry
		if journalDecode([]byte(raw), &e, journalTestConfig(t, replayTestInput(t))) == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	x := journalEntry{Previous: "quoted\\\"value"}
	raw, _ := json.Marshal(x)
	var got journalEntry
	if journalDecode(raw, &got, journalTestConfig(t, replayTestInput(t))) != nil || got.Previous != x.Previous {
		t.Fatal("valid quoted data")
	}
}

func TestJournalConcurrentCheckpointReads(t *testing.T) {
	in := replayTestInput(t)
	j, _ := journalTestCreate(t, journalTestConfig(t, in))
	tip := journalTestAppend(t, j, journalTestBatch(t, in, 0, 11), 0)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := j.Checkpoint()
			if e != nil || p != tip {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
}

func TestJournalContinuesBeyondStatelessBlockCeiling(t *testing.T) {
	in := replayTestInput(t)
	for len(in.Blocks) < 4100 {
		h := &types.Header{Number: new(big.Int).SetUint64(uint64(len(in.Blocks))), Difficulty: new(big.Int), Time: uint64(len(in.Blocks)), GasLimit: 1000000, TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, UncleHash: types.EmptyUncleHash, ParentHash: common.HexToHash(in.Anchor.Hash)}
		raw, e := rlp.EncodeToBytes(h)
		if e != nil {
			t.Fatal(e)
		}
		in.Blocks = append(in.Blocks, ReplayBlock{HeaderRLP: raw})
		in.Anchor = replayHeader(h)
	}
	c := journalTestConfig(t, in)
	j, dir := journalTestCreate(t, c)
	first := journalTestBatch(t, in, 0, 11)
	journalTestAppend(t, j, first, 0)
	for i, p := range [][2]int{{12, 2059}, {2060, 4099}} {
		b := ReplayBatch{SelectedFinalized: in.Anchor, Anchor: journalTestHeader(t, in.Blocks[p[1]]), Blocks: in.Blocks[p[0] : p[1]+1], Record: in.Record, Grants: in.Grants}
		for i := len(in.Blocks) - 2; i >= p[0]-1; i-- {
			b.Parents = append(b.Parents, journalTestHeader(t, in.Blocks[i]))
		}
		journalTestAppend(t, j, b, int64((i+1)*3))
	}
	tip, _ := j.Checkpoint()
	if tip.Anchor.Number != 4099 || tip.Version != 11 {
		t.Fatal(tip)
	}
	if e := j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e := OpenReplayJournal(context.Background(), dir, c, tip)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if j.stream.state.record.State != "deactivated" || len(j.stream.claims) != 1 || !j.stream.created[replayTestAgent] {
		t.Fatal("lost tombstones")
	}
}

func TestJournalRequiresConfirmationEvenWithExactAck(t *testing.T) {
	in := replayTestInput(t)
	j, _ := journalTestCreate(t, journalTestConfig(t, in))
	b := journalTestBatch(t, in, 0, 11)
	tip, e := j.Append(context.Background(), b, journalTestClock(in.Anchor, 0), journalTestClock(in.Anchor, 1))
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = j.Publish(tip, journalTestClock(in.Anchor, 2)); e == nil {
		t.Fatal("local commit plus ack bypassed fresh confirmation")
	}
	if e = j.ConfirmObservation(journalTestClock(in.Anchor, 3), journalTestClock(in.Anchor, 4), in.Anchor, in.Record, in.Grants); e != nil {
		t.Fatal(e)
	}
	if _, _, e = j.Publish(tip, journalTestClock(in.Anchor, 5)); e != nil {
		t.Fatal(e)
	}
}

func TestJournalConfiguredBatchAndAcquisitionBudgets(t *testing.T) {
	for _, name := range []string{"max-batches", "max-bytes", "max-parents", "max-history", "history-bytes", "evidence-bytes", "clock-rollback", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			in := replayTestInput(t)
			c := journalTestConfig(t, in)
			switch name {
			case "max-batches":
				c.MaxBatches = 1
			case "max-bytes":
				c.MaxStoredBytes = 2000
			case "max-parents":
				c.Observation.MaxAncestryHeaders = 1
			case "max-history":
				c.Replay.History.MaxItems = 1
			case "history-bytes":
				c.Replay.History.MaxBytes = 1
			case "evidence-bytes":
				c.Replay.MaxEvidenceBytes = 1
			}
			j, _ := journalTestCreate(t, c)
			before, _ := j.Checkpoint()
			b := journalTestBatch(t, in, 0, 11)
			start, completed := journalTestClock(in.Anchor, 3), journalTestClock(in.Anchor, 4)
			if name == "max-batches" || name == "clock-rollback" {
				journalTestAppend(t, j, journalTestBatch(t, in, 0, 0), 2)
				before, _ = j.Checkpoint()
				b = journalTestBatch(t, in, 1, 11)
				if name == "clock-rollback" {
					start, completed = journalTestClock(in.Anchor, 0), journalTestClock(in.Anchor, 1)
				}
			}
			ctx := context.Background()
			if name == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, e := j.Append(ctx, b, start, completed); e == nil {
				t.Fatal("accepted exhausted or cancelled acquisition")
			}
			if j.tip != before {
				t.Fatal("partial commit")
			}
		})
	}
}

func TestJournalDiskRuntimeBounds(t *testing.T) {
	dir := t.TempDir()
	d, e := openJournalDisk(dir, true)
	if e != nil {
		t.Fatal(e)
	}
	defer d.close()
	if e = d.write("CONFIG", []byte("fixed")); e != nil {
		t.Fatal(e)
	}
	if _, e = d.read("missing", 100); e == nil {
		t.Fatal("missing read")
	}
	if _, e = d.read("CONFIG", 1); e == nil {
		t.Fatal("oversize read")
	}
	if _, e = d.read(".", 100); e == nil {
		t.Fatal("directory read")
	}
	if e = d.write(journalName(1), []byte("one")); e != nil {
		t.Fatal(e)
	}
	if e = d.write(journalName(1), []byte("replacement")); e == nil {
		t.Fatal("overwrote immutable batch")
	}
	if e = d.write(journalName(2), []byte("two")); e != nil {
		t.Fatal(e)
	}
	if _, e = d.batches(1); e == nil {
		t.Fatal("too many batches")
	}
	if e = d.write(journalName(3), []byte("three")); e != nil {
		t.Fatal(e)
	}
	if _, e = d.batches(1); e == nil {
		t.Fatal("unbounded inventory")
	}
	if e = d.remove("missing"); e == nil {
		t.Fatal("missing remove")
	}
	if e = d.root.Close(); e != nil {
		t.Fatal(e)
	}
	if e = d.sync(); e == nil {
		t.Fatal("closed sync")
	}
	if _, e = d.read("CONFIG", 100); e == nil {
		t.Fatal("closed read")
	}
	if _, e = d.batches(1); e == nil {
		t.Fatal("closed inventory")
	}
	if e = d.write("closed", nil); e == nil {
		t.Fatal("closed write")
	}
}

func TestJournalParseCountsBeforeTypedAllocation(t *testing.T) {
	c := journalTestConfig(t, replayTestInput(t))
	c.Replay.MaxBlocks = 1
	c.Replay.History.MaxItems = 1
	c.Observation.MaxAncestryHeaders = 1
	for _, raw := range []string{`{"Blocks":[{},{}]}`, `{"History":[{},{}]}`, `{"Parents":[{},{}]}`, `{"unknown":[0]}`, strings.Repeat(`{"x":`, 97) + `0` + strings.Repeat(`}`, 97)} {
		if journalScan([]byte(raw), c) == nil {
			t.Fatal("unbounded local parse")
		}
	}
	m := map[string]any{}
	for i := 0; i < 33; i++ {
		m[string(rune('A'+i))] = i
	}
	raw, _ := json.Marshal(m)
	if journalScan(raw, c) == nil {
		t.Fatal("unbounded object members")
	}
}

func TestReplayJournalLifetimeIndexCeilings(t *testing.T) {
	in := replayTestInput(t)
	a, o, e := replayABIs()
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"claim", "identity"} {
		t.Run(name, func(t *testing.T) {
			b := replayBudget{limits: replayTestLimits()}
			r := replayRun{registry: in.Registry, agent: in.Agent, contract: a, outcomes: o, budget: &b, claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}, history: in.History}
			decoded, e := replayDecodeBlock(context.Background(), in.Blocks[0], &b)
			if e != nil {
				t.Fatal(e)
			}
			if name == "claim" {
				r.claims[replayClaimID{actor: replayTestOperator}] = replayClaimState{}
				r.claimLimit = 1
				if e = r.block(context.Background(), in.Blocks[0], decoded); !errors.Is(e, ErrReplayUnavailable) {
					t.Fatal(e)
				}
				return
			}
			if e = r.block(context.Background(), in.Blocks[0], decoded); e != nil {
				t.Fatal(e)
			}
			r.created["other.identity"] = true
			r.createdLimit = 1
			decoded, e = replayDecodeBlock(context.Background(), in.Blocks[1], &b)
			if e != nil {
				t.Fatal(e)
			}
			if e = r.block(context.Background(), in.Blocks[1], decoded); !errors.Is(e, ErrReplayUnavailable) {
				t.Fatal(e)
			}
		})
	}
}
