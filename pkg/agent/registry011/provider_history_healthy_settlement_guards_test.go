package registry011

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historySettlementPrefixData(t *testing.T, original *providerHistoryGroupLedger, c *providerHistoryRecoveryCheckpoint, m *providerWorkMeter) *providerHistoryHealthyPrefix {
	t.Helper()
	ledger := providerHistoryLedgerCopyObserved(original, m)
	if c == nil || c.previous == nil || ledger == nil || ledger.phase != "consumed" || ledger.ticket == nil || ledger.consumption == nil || ledger.head != c.previous.model {
		t.Fatal("independent original consumed-ledger DATA required")
	}
	return &providerHistoryHealthyPrefix{ledger: ledger, pin: c.previous.pin, inventory: c.previous.inventory.digest,
		clock: c.previous.frame.clock, retention: c.previous.retention}
}

// The standard context is cancelled only after this isolated parser has
// observed new decoded input. No timer, goroutine, filesystem race or observer
// counter mutation is needed to exercise the pre-member cancellation boundary.
type historySettlementParseCancelContext struct {
	context.Context
	cancel  context.CancelFunc
	meter   *providerWorkMeter
	decoded uint64
	checks  int
}

func (c *historySettlementParseCancelContext) Err() error {
	c.checks++
	if c.meter.actual[2] > c.decoded {
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistoryHealthySettlementRejectsInvalidParserDataBeforeArchivedReads(t *testing.T) {
	for _, mode := range []string{"policy-shape", "provider-shape", "missing-roots", "invalid-pack", "inactive-owner", "cancelled-after-pack"} {
		t.Run(mode, func(t *testing.T) {
			g, d, c := historyRecoveryArchivedSettlement(t, 1)
			g.store.mu.Lock()
			defer g.store.mu.Unlock()
			m := historyGroupCodecMeter()
			p := historySettlementPrefixData(t, d.coordinator, c, m)
			objects := c.claim.fresh
			policy := g.parent.policy
			decoded, ok := providerReplayJSON(providerReplayBytes(policy.raw, m), m).(map[string]any)
			provider, providerOK := decoded["provider"].(map[string]any)
			roots := contentString(provider["tls_roots_artifact"])
			if !ok || !providerOK || !lowerHash(roots) || !c.memberArchive.shapeLocked(g.parent.recovery, c) {
				t.Fatal("genuine settled archive and original policy required")
			}
			pack, results, err := parseProviderHistoryResultPackObserved(providerReplayBytes(objects[0].raw, m), objects[0].digest,
				p.ledger.ticket, p.ledger.consumption, c.completed, roots, g.parent.tls, g.store.config.bounds.maximumObjectBytes, m)
			if err != nil || pack == nil || results[0].digest != c.memberArchive.members[0].result {
				t.Fatal("positive original result-pack prerequisite", err)
			}
			// This detached partial-method subject owns no execution role.
			// The real checkpoint/archive is read-only; only expected DATA differs.
			subject := &providerHistoryGroupOwner{store: g.store, meter: m,
				parent: &providerHistoryProgramOwner{policy: policy, tls: g.parent.tls, recovery: g.parent.recovery}}
			var input context.Context = context.Background()
			var cancelled *historySettlementParseCancelContext
			switch mode {
			case "policy-shape":
				subject.parent.policy.raw = `{}`
			case "provider-shape":
				subject.parent.policy.raw = `{"provider":null}`
			case "missing-roots":
				subject.parent.policy.raw = `{"provider":{}}`
			case "invalid-pack":
				objects[0].raw = "!"
			case "cancelled-after-pack":
				base, cancel := context.WithCancel(context.Background())
				defer cancel()
				cancelled = &historySettlementParseCancelContext{Context: base, cancel: cancel, meter: m, decoded: m.actual[2]}
				input = cancelled
			}
			work, head, retention := m.actual, p.ledger.head, p.retention
			old, clocks, pin, ack, held := g.meter.actual, g.store.clock.read, g.pin, g.last, g.store.stored
			originals := providerHistoryProvisionedOriginalBytes(t, g.store.config)
			if err := subject.applyHealthySettlementLocked(input, p, c, &c.frame, objects); err == nil || m.failed ||
				m.actual[10] <= work[10] || p.ledger.head != head || p.retention != retention {
				t.Fatal("invalid private parser data reached archived originals or changed its ledger", mode, err)
			}
			if cancelled != nil && (cancelled.checks != 2 || cancelled.Context.Err() != context.Canceled) {
				t.Fatal("cancellation did not follow successful original result-pack decoding")
			}
			if subject.owner != nil || subject.parent.group != nil || subject.activeLocked(m) || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 ||
				g.meter.actual != old || g.store.clock.read != clocks || g.pin != pin || g.last != ack || g.store.stored != held ||
				g.store.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, g.store.config)) {
				t.Fatal("detached parser validation gained an owner or changed original execution", mode)
			}
		})
	}
}

func TestProviderHistoryHealthySettlementLateRefusalKeepsTheCurrentOriginalWriter(t *testing.T) {
	for _, mode := range []string{"retention-overflow", "settlement-operation"} {
		t.Run(mode, func(t *testing.T) {
			g := historyMultistageOwner(t, 2)
			d := historyMultistageDispatch(t, g, 1)
			historyRetainNormalPrefix(t, g, d, 1)
			if out, err := g.retainSettlementCandidate(context.Background(), d); err != nil || out == nil {
				t.Fatal("genuine first original S", err)
			}
			c, ack := g.parent.recovery.confirmed, g.last
			// Independent finite test DATA survives normal scratch retirement.
			// It is never installed as a live owner, producer model or allowance.
			saved := providerHistoryLedgerCopyObserved(d.coordinator, historyGroupCodecMeter())
			objects := c.claim.fresh
			if saved == nil || saved.phase != "consumed" || objects[0].kind != "results" || objects[1].kind != "settlement" ||
				g.advanceStageCandidate(context.Background(), ack) != nil || d.coordinator != nil || c.claim.fresh != ([8]providerHistoryStoredObject{}) {
				t.Fatal("ordinary stage advance must retire the original scratch")
			}
			s, m := g.store, g.meter
			reached := false
			syncFile := s.custody.syncFile
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				pending := s.frontierFiles.historyPending
				if reached || pending == nil || pending.frame == nil || pending.frame.phase != "admit" || pending.frame.stage != 2 ||
					filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", pending.frame.after.ordinal) {
					return nil
				}
				if !g.activeLocked(m) || !g.claimActiveLocked(pending) || g.last != ack || !g.healthyCheckpointShapeLocked(c) {
					t.Fatal("genuine second-stage writer required before private late validation")
				}
				old, clocks, pin, held, retained := m.actual, s.clock.read, g.pin, s.stored, g.retention
				clockCount := m.clocks
				originals := providerHistoryProvisionedOriginalBytes(t, s.config)
				control := historySettlementPrefixData(t, saved, c, m)
				if err := g.applyHealthySettlementLocked(context.Background(), control, c, &c.frame, objects); err != nil || control.ledger.head != c.model {
					t.Fatal("positive archived original S reconstruction under the current real observer", err)
				}
				p := historySettlementPrefixData(t, saved, c, m)
				frame := c.frame
				if mode == "retention-overflow" {
					p.retention.physical = math.MaxUint64 // Private expected accounting DATA only.
				} else {
					frame.operation += "x"
				}
				before := p.retention
				if err := g.applyHealthySettlementLocked(context.Background(), p, c, &frame, objects); err == nil || m.failed ||
					p.ledger.head != saved.head || p.ledger.failed || mode == "retention-overflow" && p.retention != before ||
					mode == "settlement-operation" && p.retention != control.retention {
					t.Fatal("late private original refusal changed a ledger or skipped its intended input prefix", mode, err)
				}
				if m.actual[10] <= old[10] || m.actual[0] != old[0] || m.actual[1] != old[1] || m.actual[9] != old[9] || m.clocks != clockCount ||
					s.clock.read != clocks || g.pin != pin || g.last != ack || s.stored != held || g.retention != retained ||
					s.frontierFiles.historyPending != pending || !g.activeLocked(m) || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
					t.Fatal("private late reconstruction renewed or changed the current original writer", mode)
				}
				reached = true
				return nil
			}
			request := historyGroupRequest(t, "web3_clientVersion", []any{})
			next, err := g.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{request}, historyMultistageTestUpper(s.config.capture))
			s.custody.syncFile = syncFile
			if err != nil || next == nil || !reached || g.failed || m.failed || g.last != next || next == ack || next.claim.frame.stage != 2 ||
				s.activeMeter.Load() != nil || !c.memberArchive.shapeLocked(g.parent.recovery, c) {
				t.Fatal("private late checks disrupted the genuine next original T", mode, err)
			}
		})
	}
}
