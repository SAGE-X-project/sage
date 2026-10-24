package registry011

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestReconstructionInvalidInputsAndDestination(t *testing.T) {
	dir, c, next, tip, key := reconstructionFixture(t)
	pub := key.Public().(ed25519.PublicKey)
	limits := ReconstructionLimits{32, 8 << 20}
	nonce := "0x" + strings.Repeat("a", 64)
	for name, edit := range map[string]func(*JournalConfig, *[]byte, *string, *string, *ReconstructionLimits){
		"invalid-old-policy": func(n *JournalConfig, _ *[]byte, _ *string, _ *string, _ *ReconstructionLimits) {
			n.Observation.ClockID = ""
		},
		"invalid-public-key": func(_ *JournalConfig, k *[]byte, _ *string, _ *string, _ *ReconstructionLimits) { *k = []byte{1} },
		"invalid-nonce":      func(_ *JournalConfig, _ *[]byte, n *string, _ *string, _ *ReconstructionLimits) { *n = "not-a-nonce" },
		"empty-reason":       func(_ *JournalConfig, _ *[]byte, _ *string, r *string, _ *ReconstructionLimits) { *r = "" },
		"oversize-reason": func(_ *JournalConfig, _ *[]byte, _ *string, r *string, _ *ReconstructionLimits) {
			*r = strings.Repeat("r", 1025)
		},
		"control-reason": func(_ *JournalConfig, _ *[]byte, _ *string, r *string, _ *ReconstructionLimits) { *r = "reason\n" },
		"invalid-utf8": func(_ *JournalConfig, _ *[]byte, _ *string, r *string, _ *ReconstructionLimits) {
			*r = string([]byte{255})
		},
		"too-many-batches": func(_ *JournalConfig, _ *[]byte, _ *string, _ *string, l *ReconstructionLimits) { l.MaxBatches = 4097 },
		"too-many-bytes": func(_ *JournalConfig, _ *[]byte, _ *string, _ *string, l *ReconstructionLimits) {
			l.MaxBytes = 128<<20 + 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			old, k, n, r, l := c, []byte(pub), nonce, "owned refusal", limits
			edit(&old, &k, &n, &r, &l)
			if _, e := PlanJournalReconfiguration(context.Background(), dir, old, next, tip, k, n, r, l); e == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	if _, e := PlanJournalReconfiguration(nil, dir, c, next, tip, pub, nonce, "owned refusal", limits); e == nil { //nolint:staticcheck // Deliberately verify the public nil-context refusal.
		t.Fatal("nil context")
	}
	if _, e := PlanJournalReconfiguration(context.Background(), filepath.Join(t.TempDir(), "absent"), c, next, tip, pub, nonce, "owned refusal", limits); e == nil {
		t.Fatal("missing retained directory")
	}
	p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
	if _, e := ReconfigurationSigningMessage(ReconfigurationPlan{}); e == nil {
		t.Fatal("invalid signing plan")
	}
	destination := t.TempDir()
	if e := os.WriteFile(filepath.Join(destination, "owned-existing-data"), []byte("retain"), 0600); e != nil {
		t.Fatal(e)
	}
	if got, _, e := RebuildReplayJournal(context.Background(), dir, destination, c, next, tip, pub, p, signature, limits); e == nil || got != nil {
		t.Fatal("nonempty destination accepted")
	}
	b, e := os.ReadFile(filepath.Join(destination, "owned-existing-data"))
	if e != nil || string(b) != "retain" {
		t.Fatal("existing destination changed", e)
	}
}

func TestReconstructionPreservesEachAcceptedLifecyclePrefix(t *testing.T) {
	for _, end := range []int{-1, 0, 1, 5, 7, 9, 10, 11} {
		t.Run("prefix-"+strconv.Itoa(end), func(t *testing.T) {
			in := replayTestInput(t)
			c := journalTestConfig(t, in)
			old, dir := journalTestCreate(t, c)
			tip, e := old.Checkpoint()
			if e != nil {
				t.Fatal(e)
			}
			if end >= 0 {
				tip = journalTestAppend(t, old, journalTestBatch(t, in, 0, end), 20)
			}
			next := c
			next.PolicyGeneration = "next-owned-generation"
			next.Observation.PolicyGeneration = next.PolicyGeneration
			seed := sha256.Sum256([]byte("owned lifecycle-prefix reconfiguration test only"))
			key := ed25519.NewKeyFromSeed(seed[:])
			p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
			candidate, _, e := RebuildReplayJournal(context.Background(), dir, t.TempDir(), c, next, tip, key.Public().(ed25519.PublicKey), p, signature, ReconstructionLimits{32, 8 << 20})
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = candidate.Close() }()
			beforeR, beforeG := old.stream.stateBytes()
			afterR, afterG := candidate.stream.stateBytes()
			if string(beforeR) != string(afterR) || string(beforeG) != string(afterG) || !sameLifetime(old.stream, candidate.stream) || candidate.confirmed {
				t.Fatal("lifecycle prefix changed or acquired publication")
			}
		})
	}
}

func TestReconstructionRefusesUnacknowledgedOrInvalidEntry(t *testing.T) {
	for _, kind := range []string{"wrong-sequence", "wrong-previous", "clock-window", "clock-order", "bad-replay", "bad-checkpoint", "configured-bytes", "too-many-files", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir, c, next, tip, key := reconstructionFixture(t)
			path := filepath.Join(dir, journalName(1))
			b, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			var entry journalEntry
			if json.Unmarshal(b, &entry) != nil {
				t.Fatal("fixture entry")
			}
			switch kind {
			case "wrong-sequence":
				entry.Sequence++
			case "wrong-previous":
				entry.Previous = "0x" + strings.Repeat("0", 64)
			case "clock-window":
				entry.Completed.WallMS += 10000
			case "clock-order":
				// Two individually valid windows may still regress between batches.
				in := replayTestInput(t)
				j, original := journalTestCreate(t, c)
				journalTestAppend(t, j, journalTestBatch(t, in, 0, 5), 20)
				tip = journalTestAppend(t, j, journalTestBatch(t, in, 6, 11), 30)
				dir = original
				path = filepath.Join(dir, journalName(2))
				b, e = os.ReadFile(path)
				if e != nil || json.Unmarshal(b, &entry) != nil {
					t.Fatal("second fixture", e)
				}
				entry.Started, entry.Completed = journalTestClock(in.Anchor, 10), journalTestClock(in.Anchor, 11)
			case "bad-replay":
				entry.Batch.Record = []byte("{}")
			case "bad-checkpoint":
				tip.Version++
			case "configured-bytes":
				next.MaxStoredBytes = 4096
			case "too-many-files":
				for i := 0; i < 40; i++ {
					if e := os.WriteFile(filepath.Join(dir, "owned-extra-"+strings.Repeat("x", i)), []byte("x"), 0600); e != nil {
						t.Fatal(e)
					}
				}
			case "symlink":
				if e := os.Remove(path); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink("CONFIG", path); e != nil {
					t.Fatal(e)
				}
			}
			if kind != "bad-checkpoint" && kind != "configured-bytes" && kind != "too-many-files" && kind != "symlink" {
				b, e = json.Marshal(entry)
				if e != nil || os.WriteFile(path, b, 0600) != nil {
					t.Fatal("write finite entry", e)
				}
			}
			if _, e := PlanJournalReconfiguration(context.Background(), dir, c, next, tip, key.Public().(ed25519.PublicKey), "0x"+strings.Repeat("a", 64), "owned invalid entry", ReconstructionLimits{32, 8 << 20}); e == nil {
				t.Fatal("accepted invalid retained evidence")
			}
		})
	}
}

func TestReconstructionFinalReadbackRefusals(t *testing.T) {
	for _, kind := range []string{"config", "missing", "cancel", "yield-error"} {
		t.Run(kind, func(t *testing.T) {
			dir, c, next, tip, _ := reconstructionFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, e := reconstructPrefix(ctx, dir, c, next, tip, ReconstructionLimits{32, 8 << 20}, func(journalEntry) error {
				switch kind {
				case "config":
					return os.WriteFile(filepath.Join(dir, "CONFIG"), []byte("{}"), 0600)
				case "missing":
					return os.Remove(filepath.Join(dir, journalName(1)))
				case "cancel":
					cancel()
					return nil
				default:
					return errors.New("owned destination unavailable")
				}
			})
			if e == nil {
				t.Fatal("readback or copy failure accepted")
			}
		})
	}
}

// Cancellation is triggered only by the owned destination reaching creation.
// This exercises a valid cancellation boundary without timing a filesystem race.
type reconstructionCancelContext struct {
	context.Context
	cancel  context.CancelFunc
	created string
}

func (c reconstructionCancelContext) Err() error {
	if _, e := os.Stat(c.created); e == nil {
		c.cancel()
	}
	return c.Context.Err()
}

func TestReconstructionCancellationRetainsFailedCandidateAndOriginal(t *testing.T) {
	dir, c, next, tip, key := reconstructionFixture(t)
	p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
	original, e := os.ReadFile(filepath.Join(dir, journalName(1)))
	if e != nil {
		t.Fatal(e)
	}
	destination := t.TempDir()
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := reconstructionCancelContext{base, cancel, filepath.Join(destination, "CONFIG")}
	j, audit, e := RebuildReplayJournal(ctx, dir, destination, c, next, tip, key.Public().(ed25519.PublicKey), p, signature, ReconstructionLimits{32, 8 << 20})
	if e == nil || j != nil || audit != (ReconfigurationAudit{}) || !errors.Is(e, context.Canceled) {
		t.Fatal("failed candidate returned", e)
	}
	marker, e := os.ReadFile(filepath.Join(destination, "QUARANTINED"))
	if e != nil || string(marker) != "quarantined\n" {
		t.Fatal("failed candidate was erased or not quarantined", e)
	}
	retained, e := os.ReadFile(filepath.Join(dir, journalName(1)))
	if e != nil || string(retained) != string(original) {
		t.Fatal("old accepted prefix changed", e)
	}
	marker, e = os.ReadFile(filepath.Join(dir, "LOCK"))
	if e != nil || string(marker) != "locked\n" {
		t.Fatal("old ownership changed", e)
	}
}

func TestReconstructionRequiresExactLifetimeIndexes(t *testing.T) {
	id := replayClaimID{actor: "owned-actor"}
	a := replayStream{claims: map[replayClaimID]replayClaimState{id: {block: 7, consumed: true}}, created: map[string]bool{"terminal-identity": true}}
	for _, kind := range []string{"claim-count", "claim-binding", "claim-consumption", "identity-count", "identity-binding", "identity-state", "admission-time"} {
		t.Run(kind, func(t *testing.T) {
			b := replayStream{claims: map[replayClaimID]replayClaimState{id: {block: 7, consumed: true}}, created: map[string]bool{"terminal-identity": true}}
			switch kind {
			case "claim-count":
				delete(b.claims, id)
			case "claim-binding":
				delete(b.claims, id)
				b.claims[replayClaimID{actor: "other-owned-actor"}] = replayClaimState{block: 7, consumed: true}
			case "claim-consumption":
				b.claims[id] = replayClaimState{block: 7}
			case "identity-count":
				delete(b.created, "terminal-identity")
			case "identity-binding":
				delete(b.created, "terminal-identity")
				b.created["another-owned-identity"] = true
			case "identity-state":
				b.created["terminal-identity"] = false
			case "admission-time":
				b.state.admitted++
			}
			if sameLifetime(a, b) {
				t.Fatal("lost or changed lifetime history accepted")
			}
		})
	}
}
