package registry011

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These finite offline fixtures use only owned temporary directories. Test
// signatures do not enroll an administrator, activate a Source or load a key.
func reconstructionFixture(t *testing.T) (string, JournalConfig, JournalConfig, JournalCheckpoint, ed25519.PrivateKey) {
	t.Helper()
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	j, dir := journalTestCreate(t, c)
	tip := journalTestAppend(t, j, journalTestBatch(t, in, 0, 11), 20)
	next := c
	next.PolicyGeneration = "next-owned-test-policy"
	next.Observation.PolicyGeneration = next.PolicyGeneration
	seed := sha256.Sum256([]byte("owned reconfiguration unit fixture; never an operating key"))
	return dir, c, next, tip, ed25519.NewKeyFromSeed(seed[:])
}

func reconstructionTestPlan(t *testing.T, dir string, c, next JournalConfig, tip JournalCheckpoint, key ed25519.PrivateKey) (ReconfigurationPlan, []byte) {
	t.Helper()
	p, e := PlanJournalReconfiguration(context.Background(), dir, c, next, tip, key.Public().(ed25519.PublicKey), "0x"+strings.Repeat("a", 64), "Retain the accepted branch under an explicit test generation", ReconstructionLimits{32, 8 << 20})
	if e != nil {
		t.Fatal(e)
	}
	m, e := ReconfigurationSigningMessage(p)
	if e != nil {
		t.Fatal(e)
	}
	return p, ed25519.Sign(key, m)
}

func TestReconstructionCandidatePreservesRefusalAndLifetime(t *testing.T) {
	dir, c, next, tip, key := reconstructionFixture(t)
	for name, value := range map[string]string{"PENDING": "pending\n", "QUARANTINED": "quarantined\n"} {
		if e := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); e != nil {
			t.Fatal(e)
		}
	}
	p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
	destination := t.TempDir()
	j, audit, e := RebuildReplayJournal(context.Background(), dir, destination, c, next, tip, key.Public().(ed25519.PublicKey), p, signature, ReconstructionLimits{32, 8 << 20})
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = j.Close() }()
	got, e := j.Checkpoint()
	if e != nil || got != p.Next || got.Digest == tip.Digest || got.Version != tip.Version || j.confirmed || audit.Plan != p || j.stream.state.record.State != "deactivated" || len(j.stream.claims) == 0 || len(j.stream.created) == 0 {
		t.Fatal("candidate lost retained state or acquired authority", e)
	}
	for name, value := range map[string]string{"LOCK": "locked\n", "PENDING": "pending\n", "QUARANTINED": "quarantined\n"} {
		b, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil || string(b) != value {
			t.Fatal("modified old marker", name, e)
		}
	}
	if _, _, e := j.Publish(got, journalTestClock(c.Deployment, 100)); e == nil {
		t.Fatal("candidate published without fresh confirmation")
	}
}

func TestReconstructionRefusesWrongAuthorizationBeforeCreation(t *testing.T) {
	dir, c, next, tip, key := reconstructionFixture(t)
	p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
	for name, edit := range map[string]func(*ReconfigurationPlan, *[]byte){
		"reason":          func(p *ReconfigurationPlan, _ *[]byte) { p.Reason = "another reason" },
		"nonce":           func(p *ReconfigurationPlan, _ *[]byte) { p.Nonce = "0x" + strings.Repeat("b", 64) },
		"checkpoint":      func(p *ReconfigurationPlan, _ *[]byte) { p.Previous.Version++ },
		"policy":          func(p *ReconfigurationPlan, _ *[]byte) { p.NextPolicy = "other" },
		"signature":       func(_ *ReconfigurationPlan, s *[]byte) { (*s)[0] ^= 1 },
		"short-signature": func(_ *ReconfigurationPlan, s *[]byte) { *s = (*s)[:3] },
	} {
		t.Run(name, func(t *testing.T) {
			q, s := p, bytes.Clone(signature)
			edit(&q, &s)
			destination := t.TempDir()
			j, _, e := RebuildReplayJournal(context.Background(), dir, destination, c, next, tip, key.Public().(ed25519.PublicKey), q, s, ReconstructionLimits{32, 8 << 20})
			names, _ := os.ReadDir(destination)
			if e == nil || j != nil || len(names) != 0 {
				t.Fatal("unauthorized candidate touched destination")
			}
		})
	}
}

func TestReconstructionRetainedPrefixRefusals(t *testing.T) {
	for _, name := range []string{"unknown", "extra-tail", "missing", "torn", "wrong-config", "bad-marker", "byte-budget", "batch-budget", "cancel"} {
		t.Run(name, func(t *testing.T) {
			dir, c, next, tip, key := reconstructionFixture(t)
			limits := ReconstructionLimits{32, 8 << 20}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "unknown":
				_ = os.WriteFile(filepath.Join(dir, "unknown"), []byte("x"), 0600)
			case "extra-tail":
				_ = os.WriteFile(filepath.Join(dir, journalName(2)), []byte("x"), 0600)
			case "missing":
				_ = os.Remove(filepath.Join(dir, journalName(1)))
			case "torn":
				_ = os.WriteFile(filepath.Join(dir, journalName(1)), []byte("{"), 0600)
			case "wrong-config":
				_ = os.WriteFile(filepath.Join(dir, "CONFIG"), []byte("{}"), 0600)
			case "bad-marker":
				_ = os.WriteFile(filepath.Join(dir, "PENDING"), []byte("different"), 0600)
			case "byte-budget":
				limits.MaxBytes = 1
			case "batch-budget":
				limits.MaxBatches = 0
			case "cancel":
				cancel()
			}
			p, e := PlanJournalReconfiguration(ctx, dir, c, next, tip, key.Public().(ed25519.PublicKey), "0x"+strings.Repeat("a", 64), "owned refusal", limits)
			if e == nil || p != (ReconfigurationPlan{}) {
				t.Fatal("accepted incomplete retained prefix")
			}
		})
	}
}

func TestReconstructionCannotChangeTrustScope(t *testing.T) {
	dir, c, good, tip, key := reconstructionFixture(t)
	for name, edit := range map[string]func(*JournalConfig){
		"same-generation": func(n *JournalConfig) { *n = c },
		"agent":           func(n *JournalConfig) { n.Agent = "another-agent" },
		"source":          func(n *JournalConfig) { n.SourceID = "another-source" },
		"code":            func(n *JournalConfig) { n.CodeIdentity = "0x" + strings.Repeat("3", 64) },
		"abi":             func(n *JournalConfig) { n.ABIIdentity = "0x" + strings.Repeat("4", 64) },
		"deployment":      func(n *JournalConfig) { n.Deployment.Number++ },
		"clock":           func(n *JournalConfig) { n.Observation.ClockID = "another-clock" },
	} {
		t.Run(name, func(t *testing.T) {
			n := good
			edit(&n)
			if _, e := PlanJournalReconfiguration(context.Background(), dir, c, n, tip, key.Public().(ed25519.PublicKey), "0x"+strings.Repeat("a", 64), "owned refusal", ReconstructionLimits{32, 8 << 20}); e == nil {
				t.Fatal("changed trust scope")
			}
		})
	}
}

func TestRetainedReadbackDetectsChangedPrefix(t *testing.T) {
	for _, kind := range []string{"rename", "content"} {
		t.Run(kind, func(t *testing.T) {
			dir, c, next, tip, _ := reconstructionFixture(t)
			_, e := reconstructPrefix(context.Background(), dir, c, next, tip, ReconstructionLimits{32, 8 << 20}, func(journalEntry) error {
				path := filepath.Join(dir, journalName(1))
				if kind == "rename" {
					return os.Rename(path, filepath.Join(dir, journalName(2)))
				}
				b, e := os.ReadFile(path)
				if e != nil {
					return e
				}
				return os.WriteFile(path, append(b, ' '), 0600)
			})
			if e == nil {
				t.Fatal("accepted changed retained prefix")
			}
		})
	}
}

func TestReconstructionChecksBothPolicyBatchCeilingsBeforeCreation(t *testing.T) {
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	j, dir := journalTestCreate(t, c)
	journalTestAppend(t, j, journalTestBatch(t, in, 0, 5), 20)
	tip := journalTestAppend(t, j, journalTestBatch(t, in, 6, 11), 30)
	next := c
	next.PolicyGeneration = "next-owned-generation"
	next.Observation.PolicyGeneration = next.PolicyGeneration
	seed := sha256.Sum256([]byte("owned cap unit fixture; never an operating key"))
	key := ed25519.NewKeyFromSeed(seed[:])
	p, signature := reconstructionTestPlan(t, dir, c, next, tip, key)
	for _, oldSide := range []bool{false, true} {
		old, newer := c, next
		if oldSide {
			old.MaxBatches = 1
		} else {
			newer.MaxBatches = 1
		}
		destination := t.TempDir()
		got, _, e := RebuildReplayJournal(context.Background(), dir, destination, old, newer, tip, key.Public().(ed25519.PublicKey), p, signature, ReconstructionLimits{32, 8 << 20})
		files, _ := os.ReadDir(destination)
		if e == nil || got != nil || len(files) != 0 {
			t.Fatal("policy cap refused only after creation")
		}
	}
}

func TestReconstructionReadbackRefusesMarkerChanges(t *testing.T) {
	for _, marker := range []struct{ name, text string }{{"LOCK", "locked\n"}, {"PENDING", "pending\n"}, {"QUARANTINED", "quarantined\n"}} {
		for _, add := range []bool{false, true} {
			kind := "remove"
			if add {
				kind = "add"
			}
			t.Run(marker.name+"/"+kind, func(t *testing.T) {
				dir, c, next, tip, _ := reconstructionFixture(t)
				path := filepath.Join(dir, marker.name)
				if add {
					if marker.name == "LOCK" {
						if e := os.Remove(path); e != nil {
							t.Fatal(e)
						}
					}
				} else if marker.name != "LOCK" {
					if e := os.WriteFile(path, []byte(marker.text), 0600); e != nil {
						t.Fatal(e)
					}
				}
				_, e := reconstructPrefix(context.Background(), dir, c, next, tip, ReconstructionLimits{32, 8 << 20}, func(journalEntry) error {
					if add {
						return os.WriteFile(path, []byte(marker.text), 0600)
					}
					return os.Remove(path)
				})
				if e == nil {
					t.Fatal("accepted changed refusal/ownership marker membership")
				}
			})
		}
	}
}
