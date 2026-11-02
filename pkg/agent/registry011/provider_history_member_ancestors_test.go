package registry011

import (
	"context"
	"strconv"
	"testing"
)

func TestProviderHistoryMemberReplayBindsEarlierCheckpointAndArchiveValues(t *testing.T) {
	for _, mode := range []string{"model", "head", "clock", "raw", "inventory", "retention", "admission", "initial-model", "initial-descriptor", "archive-result", "archive-capture", "archive-intent", "archive-order"} {
		for role := range 2 {
			t.Run(mode+"-"+strconv.Itoa(role), func(t *testing.T) {
				o := historyMultistageOwner(t, 3)
				for stage := uint64(1); stage <= 2; stage++ {
					out := historyMultistageSettle(t, o, stage)
					if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
						t.Fatal(err)
					}
				}
				d := historyMultistageDispatch(t, o, 3)
				historyMemberReplayEnter(t, o, d)
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err != nil || got == nil {
					t.Fatal("genuine third-stage positive control", role, err)
				}
				c2 := o.stageBefore.previous
				s1 := c2.previous.previous
				initial := o.parent.recovery.initial
				n := s1.memberArchive.members[0]
				capture, intent, original, attempt := n.capture, n.capture.intent, n.capture.original, n.capture.intent.attempt
				// Independent UNIT backups, not values borrowed from the seal.
				oldC2, oldS1, oldInitial := *c2, *s1, *initial
				oldAdmission, oldDescriptor := *c2.admission, *initial.descriptor
				oldComplete, oldCapture, oldIntent := *n, *capture, *intent
				oldOriginal, oldAttempt, oldOrder := *original, *attempt, *s1.memberArchive.order
				switch mode {
				case "model":
					c2.model = admissionTestDigest("f")
				case "head":
					c2.head += " "
				case "clock":
					c2.frame.clock.WallMS++
				case "raw":
					c2.frame.intent += " "
				case "inventory":
					c2.inventory.raw += " "
				case "retention":
					c2.retention.raw++
				case "admission":
					c2.admission.table = admissionTestDigest("f")
				case "initial-model":
					initial.model = admissionTestDigest("f")
				case "initial-descriptor":
					initial.descriptor.targetRaw += " "
				case "archive-result":
					n.result, original.result = admissionTestDigest("f"), admissionTestDigest("f")
				case "archive-capture":
					capture.response, original.response = admissionTestDigest("f"), admissionTestDigest("f")
				case "archive-intent":
					intent.member.request.inputDigest = admissionTestDigest("f")
					attempt.member.request.inputDigest = admissionTestDigest("f")
				case "archive-order":
					s1.memberArchive.order.state = providerHistoryOrderConfirmed
				}
				if mode != "initial-descriptor" {
					if old, err := o.replayCommittedConsumptionLocked(context.Background()); err == nil || old != nil || o.meter.failed {
						t.Fatal("unchanged full oracle must refuse this older semantic fact without failing the meter", mode, err)
					}
				}
				before, clock, sent := o.meter.actual, o.store.clock.read, o.order.sent
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err == nil || got != nil ||
					o.order.sent != sent || o.store.clock.read != clock || o.meter.failed || o.meter.actual[0] != before[0] {
					t.Fatal("older non-byte expectation escaped its original baseline", mode, role, err)
				}
				*c2, *s1, *initial = oldC2, oldS1, oldInitial
				*c2.admission, *initial.descriptor = oldAdmission, oldDescriptor
				*n, *capture, *intent = oldComplete, oldCapture, oldIntent
				*original, *attempt, *s1.memberArchive.order = oldOriginal, oldAttempt, oldOrder
				if got, err := d.replayMemberEquivalentLocked(context.Background(), role); err != nil || got == nil || got.head != d.coordinator.head {
					t.Fatal("restored UNIT values did not retain genuine positive equivalence", mode, role, err)
				}
			})
		}
	}
}

func TestProviderHistoryMemberReplayReleasesOriginalRootsDespiteDetachedCurrentPointer(t *testing.T) {
	for _, mode := range []string{"nil", "copy", "missing-current-seal"} {
		t.Run(mode, func(t *testing.T) {
			o, original := historyMemberDispatchFixture(t, 3)
			historyRetainNormalPrefix(t, o, original, 1)
			r, q := o.parent.recovery, original.memberReplaySeal
			pin, ack, held, sent, consumed := o.pin, o.last, o.store.stored, original.order.sent, original.order.consumed
			switch mode {
			case "nil":
				o.dispatch = nil
			case "copy":
				copy := *original
				copy.owner = &copy
				o.dispatch = &copy
			case "missing-current-seal":
				original.memberReplaySeal = nil
			}
			o.failLocked()
			o.releaseMemberReplayLocked() // Nil-safe, idempotent original cleanup.
			if !o.failed || !o.meter.failed || !o.meter.closed || !q.revoked || q.ancestors != nil || q.claims != ([2]providerHistoryMemberClaimInput{}) || q.fixed.groupModels != ([2]*providerHistoryGroupLedger{}) ||
				q.reads != ([2]*providerHistoryReplayReads{}) || original.memberReplaySeal != nil || original.replayReads != ([2]*providerHistoryReplayReads{}) ||
				r.memberReplaySeal != nil || r.originalDispatch != original || o.pin != pin || o.last != ack || o.store.stored != held ||
				original.order.sent != sent || original.order.consumed != consumed || original.retained != 1 || original.coordinator == nil || original.custodian == nil {
				t.Fatal("original registration remained reachable or durable original facts were reset", mode)
			}
			before := o.meter.actual
			if got, err := original.replayMemberEquivalentLocked(context.Background(), 0); err == nil || got != nil || o.meter.actual != before {
				t.Fatal("original revoked binding re-entered", mode, err)
			}
		})
	}
}
