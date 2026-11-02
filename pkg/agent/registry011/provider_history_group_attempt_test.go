package registry011

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func historyMemberDispatchFixture(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	return historyMemberDispatchFixtureWithRows(t, count, nil)
}

func historyMemberDispatchFixtureWithRows(t *testing.T, count int, rows *providerHistoryDispatchBounds) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	return historyMemberDispatchFixtureWithOptions(t, count, rows, false)
}

func historyMemberDispatchFixtureWithOptions(t *testing.T, count int, rows *providerHistoryDispatchBounds, namespace bool) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	return historyMemberDispatchFixtureWithRecoverySelection(t, count, rows, namespace, false)
}

func historyMemberDispatchFixtureWithRecoverySelection(t *testing.T, count int, rows *providerHistoryDispatchBounds, namespace, legacy bool) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	return historyMemberDispatchFixtureWithRecoveryGroupSelection(t, count, rows, namespace, legacy, false)
}

func historyMemberDispatchFixtureWithRecoveryGroupSelection(t *testing.T, count int, rows *providerHistoryDispatchBounds, namespace, legacy, group bool) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	t.Helper()
	_, capture := providerFrontierBoundsFixture()
	capture.Timeout = 10 * time.Second
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{
		capture: &capture,
		pin:     true, pinLargeObject: true, historyProgram: true, historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyDispatchRows:      rows,
		historyRecoveryNamespace: namespace,
		historyRecoveryLegacy:    legacy,
		historyRecoveryGroup:     group,
	})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("genuine first operation", err)
	}
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if _, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil {
		t.Fatal("genuine first exchange", err)
	}
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]*providerIntegrationRequest, count)
	for i := range requests {
		method := "eth_chainId"
		if i%2 == 1 {
			method = "web3_clientVersion"
		}
		requests[i] = historyGroupRequest(t, method, []any{})
	}
	upper := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 2 << 20, metadata: 2 << 20, objects: 128, files: 256, clocks: 256}
	upper.execution[0], upper.execution[1], upper.execution[10] = uint64(count), uint64(count)*uint64(s.config.capture.WireBytes+1), 1<<48
	upper.custody[2], upper.custody[9], upper.custody[10] = 256<<20, 32<<20, 1<<58
	ticket, err := o.retainAdmissionCandidate(context.Background(), "selection", requests, upper)
	if err != nil {
		t.Fatal("genuine retained T", err)
	}
	ack, err := o.retainConsumptionCandidate(context.Background(), ticket)
	if err != nil {
		t.Fatal("genuine retained C", err)
	}
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil || !o.parent.dispatchBounds.valid() || !d.wholeResultPackFitsLocked() {
		t.Fatal("separate pre-H0 synthetic member profile and original C preparation", err)
	}
	return o, d
}

func TestProviderHistoryMemberDispatchRetainsOneSeveralAndMaximumOrdinals(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, count)
			s, m, pin, ack, held := o.store, o.meter, o.pin, o.last, o.store.stored
			window, clocks := o.windowStart, o.clockStart
			for ordinal := range count {
				member := d.coordinator.ticket.table.members[ordinal]
				response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
				if member.request.method == "web3_clientVersion" {
					response = `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
				}
				conn := providerIntegrationOwnedTLS(t, member.request.body, response)
				actual, reads, retained := m.actual, s.clock.read, o.retention
				original, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn)
				prefix, _ := providerHistoryMemberPrefix(ordinal + 1)
				if err != nil || original == nil || original.owner != original || original != d.originals[ordinal] || original.dispatch != d ||
					original.attempt == nil || original.attempt.owner != original.attempt || !original.attempt.used || original.attempt.ordinal != ordinal ||
					original.attempt.connection != nil || original.attempt.original != original || original.attempt.files != 2 || original.result == "" ||
					d.retained != ordinal+1 || d.attempt != nil || o.order.next != ordinal+1 || o.order.sent != prefix || o.order.state != providerHistoryOrderConfirmed ||
					m.actual[0] != actual[0]+1 || m.actual[1] <= actual[1] || s.clock.read != reads+2 || m.failed || m.closed ||
					o.phase != "dispatch-prepared" || o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held ||
					s.activeMeter.Load() != nil || o.windowStart != window || o.clockStart != clocks ||
					o.retention.objects != retained.objects+2 || o.retention.files != retained.files+2 {
					t.Fatal("member was not independently retained under the original C/hold/order", err)
				}
				last := original.attempt
				recovery := o.parent.recovery
				sealed := recovery.members[ordinal]
				if sealed == nil || sealed.owner != sealed || sealed.capture == nil || sealed.capture.original != original || sealed.capture.intent.attempt != last ||
					sealed.result != original.result || sealed.capture.response != original.response || sealed.capture.tls != original.tls ||
					sealed.capture.intent.member != member || recovery.completed != ordinal+1 || recovery.sent != prefix || recovery.order != o.order ||
					recovery.consumption != ack.claim.checkpoint || recovery.memberAttempt != nil || recovery.memberCapture != nil || recovery.memberOriginal != nil || !recovery.memberSelectionLocked(o) {
					t.Fatal("completed original lost its sealed consumed/sent/retained prefix")
				}
				if last.finalMetadataRead+1 != s.clock.read || last.endWork != m.actual || last.endWork[10] != last.finalMetadataWork[10]+1 {
					t.Fatal("final metadata was observed after the original end clock")
				}
				for i, value := range last.finalMetadataWork {
					if i != 10 && last.endWork[i] != value {
						t.Fatal("non-clock work followed the final metadata marker", i)
					}
				}
				for _, kind := range []string{"response", "tls"} {
					// Filename inspection uses a nil-observer DATA copy of the
					// dispatch; it supplies no usable owner or execution role.
					data := *d
					data.owner, data.meter = &data, nil
					name := data.originalNameLocked(ordinal, kind)
					if raw, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, name)); err != nil || len(raw) == 0 {
						t.Fatal("selected member original is absent", kind, err)
					}
				}
				after, afterReads := m.actual, s.clock.read
				if again, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, nil); err == nil || again != nil || m.actual != after || s.clock.read != afterReads {
					t.Fatal("duplicate ordinal reached dial, sample or observer")
				}
			}
			if o.order.state == providerHistoryOrderClosed || o.pin != ack.pin || !o.committedConsumptionShapeLocked() {
				t.Fatal("member receipts fabricated S or final successful closure")
			}
		})
	}
}

func TestProviderHistoryMemberEntryRefusesDetachedOrUnderivedInputsWithoutObservation(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	for _, mode := range []string{"nil-context", "cancelled", "nil-dispatch", "copied-dispatch", "copied-owner", "negative", "skipped", "changed-derivation", "underived", "copied-ack", "wrong-phase", "existing-attempt", "closed-meter"} {
		t.Run(mode, func(t *testing.T) {
			owner, input, ordinal, ctx := o, d, 0, context.Background()
			proof, inputAck, phase, attempt, closed := o.parent.bounds.derivation.dispatch, d.acknowledgment, o.phase, d.attempt, o.meter.closed
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			case "nil-dispatch":
				input = nil
			case "copied-dispatch":
				copy := *d
				input = &copy
			case "copied-owner":
				copy := *o
				owner = &copy
			case "negative":
				ordinal = -1
			case "skipped":
				ordinal = 1
			case "changed-derivation":
				o.parent.bounds.derivation.dispatch.tailMS++
			case "underived":
				o.parent.bounds.derivation.dispatch = providerHistoryDispatchBounds{}
			case "copied-ack":
				copy := *inputAck
				d.acknowledgment = &copy
			case "wrong-phase":
				o.phase = "consumed"
			case "existing-attempt":
				d.attempt = &providerHistoryDispatchAttempt{}
			case "closed-meter":
				o.meter.closed = true
			}
			actual, reads, held, retained, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.retention, o.pin
			got, err := owner.retainMemberCandidateWithConnection(ctx, input, ordinal, nil)
			o.parent.bounds.derivation.dispatch, d.acknowledgment, o.phase, d.attempt, o.meter.closed = proof, inputAck, phase, attempt, closed
			if err == nil || got != nil || o.meter.actual != actual || o.store.clock.read != reads || o.store.stored != held || o.retention != retained || o.pin != pin || o.failed || o.store.activeMeter.Load() != nil {
				t.Fatal("detached input sampled, dialed, allocated a member or poisoned the original")
			}
		})
	}
	var missing *providerHistoryGroupOwner
	if result, err := missing.retainMemberCandidate(context.Background(), nil, 0); err == nil || result != nil {
		t.Fatal("missing owner")
	}
}

func TestProviderHistoryDispatchSourceRowsRemainClosedFiniteData(t *testing.T) {
	b := providerHistoryDispatchBounds{member: candidateWorkVector{0: 1, 1: 100, 2: 200, 9: 300, 10: 400}, closure: candidateWorkVector{2: 20, 9: 30, 10: 40}, memberClocks: 2, closureClocks: 2, tailMS: 100}
	work, clocks, ok := b.remaining(3)
	if !ok || work[0] != 3 || work[1] != 300 || work[2] != 620 || work[9] != 930 || work[10] != 1240 || clocks != 8 {
		t.Fatal("member/closure rows overlap or skip a coordinate")
	}
	for _, mode := range []string{"zero", "count-zero", "count-max", "work-overflow", "clock-overflow", "tail-zero"} {
		input, count := b, 3
		switch mode {
		case "zero":
			input = providerHistoryDispatchBounds{}
		case "count-zero":
			count = 0
		case "count-max":
			count = providerHistoryGroupMembers + 1
		case "work-overflow":
			input.member[10] = math.MaxUint64
		case "clock-overflow":
			input.memberClocks = math.MaxUint64
		case "tail-zero":
			input.tailMS = 0
		}
		if _, _, valid := input.remaining(count); valid {
			t.Fatal("underived or overflowing source row accepted", mode)
		}
	}
	for _, value := range []any{&providerHistoryDispatchAttempt{}, &providerHistoryMemberOriginal{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON manufactured an original attempt or retained member")
		}
	}
}

func TestProviderHistoryMemberNextOrdinalRechecksPriorOriginalsBeforeWire(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	first := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, first.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	stored, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
	if err != nil || stored == nil || d.retained != 1 {
		t.Fatal("genuine first member precursor", err)
	}
	data := *d
	data.owner, data.meter = &data, nil
	path := filepath.Join(o.store.config.evidenceDirectory, data.originalNameLocked(0, "response"))
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		t.Fatal("owned original precursor", err)
	}
	// Finite owned-file unit fault. No altered payload is transmitted.
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	second := d.coordinator.ticket.table.members[1]
	owned := providerIntegrationOwnedTLS(t, second.request.body, `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`)
	actual, reads, held, pin, retained := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.retention
	if next, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, owned); err == nil || next != nil ||
		o.meter.actual[0] != actual[0] || o.meter.actual[1] != actual[1] || o.store.clock.read != reads+1 ||
		!o.failed || !o.meter.failed || !o.meter.closed || !o.store.failed || d.retained != 1 || d.originals[0] != stored || d.originals[1] != nil ||
		o.order.next != 1 || o.order.sent != 1 || o.order.state != providerHistoryOrderFailed || o.store.stored != held || o.pin != pin || o.retention != retained {
		t.Fatal("altered prior original was used to dispatch, acknowledged or refunded")
	}
	after, afterReads := o.meter.actual, o.store.clock.read
	if next, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, nil); err == nil || next != nil || o.meter.actual != after || o.store.clock.read != afterReads {
		t.Fatal("quarantined original was retried")
	}
}

func TestProviderHistoryMemberStorageUncertaintyKeepsTheAttemptAndHold(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	data := *d
	data.owner, data.meter = &data, nil
	name := data.originalNameLocked(0, "response")
	syncFile := o.store.evidence.syncFile
	injected := false
	o.store.evidence.syncFile = func(file *os.File) error {
		if filepath.Base(file.Name()) == name {
			injected = true
			return errProviderIntegration
		}
		return syncFile(file)
	}
	member := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	actual, held, pin, ack, before := o.meter.actual, o.store.stored, o.pin, o.last, o.retention
	if original, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err == nil || original != nil || !injected ||
		!o.failed || !o.meter.failed || !o.meter.closed || o.meter.actual[0] != actual[0]+1 || o.meter.actual[1] <= actual[1] ||
		d.attempt == nil || !d.attempt.used || d.attempt.connection != nil || d.attempt.files != 0 || d.originals[0] != nil || d.retained != 0 ||
		o.order.sent != 1 || o.order.next != 0 || o.order.state != providerHistoryOrderFailed || o.pin != pin || o.last != ack || o.store.stored != held ||
		o.retention.files != before.files+2 || o.retention.objects != before.objects+2 || o.store.activeMeter.Load() != nil {
		t.Fatal("uncertain storage fabricated a member ACK, restored an attempt or refunded its hold")
	}
	if _, err := os.Stat(filepath.Join(o.store.config.evidenceDirectory, name)); err != nil {
		t.Fatal("uncertain owned-file evidence was erased", err)
	}
	recovery := o.parent.recovery
	if recovery.memberAttempt == nil || recovery.memberCapture == nil || recovery.memberCapture.intent != recovery.memberAttempt || recovery.memberCapture.original != d.attempt.original ||
		recovery.memberAttempt.responseName != name || recovery.memberCapture.responseBytes == 0 || recovery.sent != 1 || recovery.completed != 0 || recovery.members[0] != nil ||
		!recovery.memberSelectionLocked(o) {
		t.Fatal("partial original lost its pre-write expected name, digest or attempted prefix")
	}
}

func TestProviderHistoryMemberRequiresOriginalPreH0DispatchRows(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || o.parent.dispatchBounds != (providerHistoryDispatchBounds{}) {
		t.Fatal("genuine older profile without dispatch source rows", err)
	}
	member := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	actual, reads, held := o.meter.actual, o.store.clock.read, o.store.stored
	if original, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err == nil || original != nil ||
		o.meter.actual != actual || o.store.clock.read != reads || o.store.stored != held || o.failed || d.attempt != nil || o.store.activeMeter.Load() != nil {
		t.Fatal("underived original rows reached a member observation or effect")
	}
}

func TestProviderHistoryMemberOverflowRowsRefuseBeforeEntryObservations(t *testing.T) {
	for _, mode := range []string{"work-product", "work-sum", "clock-product", "clock-sum"} {
		t.Run(mode, func(t *testing.T) {
			_, capture := providerFrontierBoundsFixture()
			rows := providerHistoryDispatchBounds{
				member:  candidateWorkVector{0: 1, 1: uint64(capture.WireBytes + 1), 2: 4 << 20, 9: 128 << 10, 10: 1 << 48},
				closure: candidateWorkVector{2: 32 << 20, 9: 1 << 20, 10: 1 << 50}, memberClocks: 2, closureClocks: 2, tailMS: 100,
			}
			switch mode {
			case "work-product":
				rows.member[10] = math.MaxUint64
			case "work-sum":
				rows.closure[10] = math.MaxUint64
			case "clock-product":
				rows.memberClocks = math.MaxUint64
			case "clock-sum":
				rows.closureClocks = math.MaxUint64
			}
			// The invalid numeric input is selected BEFORE this fresh H0.
			// No live subject or original derivation is raised or replaced.
			o, d := historyMemberDispatchFixtureWithRows(t, 2, &rows)
			o.store.mu.Lock()
			bound := o.parent.matchesLocked()
			o.store.mu.Unlock()
			_, _, finite := rows.remaining(2)
			if !bound || !rows.valid() || finite || o.parent.dispatchBounds != rows {
				t.Fatal("otherwise-valid original with overflow source rows is required")
			}
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			actual, reads, held, pin, phase := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.phase
			if got, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err == nil || got != nil ||
				o.meter.actual != actual || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.phase != phase ||
				o.failed || d.attempt != nil || d.retained != 0 || o.store.activeMeter.Load() != nil {
				t.Fatal("overflow source rows entered native observation or consumed a member")
			}
		})
	}
}

func TestProviderHistoryMemberCancellationAfterTLSOriginalSyncCannotPublish(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	data := *d
	data.owner, data.meter = &data, nil
	name := data.originalNameLocked(0, "tls")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	syncFile := o.store.evidence.syncFile
	cancelled := false
	o.store.evidence.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		if filepath.Base(file.Name()) == name {
			cancelled = true
			cancel() // Normal owned exchange, no hostile peer or injected clock.
		}
		return nil
	}
	member := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	actual, reads, pin, ack := o.meter.actual, o.store.clock.read, o.pin, o.last
	if got, err := o.retainMemberCandidateWithConnection(ctx, d, 0, conn); err == nil || got != nil || !cancelled ||
		o.meter.actual[0] != actual[0]+1 || o.meter.actual[1] <= actual[1] || o.store.clock.read != reads+2 ||
		!o.failed || !o.meter.failed || !o.meter.closed || d.retained != 0 || d.originals[0] != nil ||
		d.attempt == nil || !d.attempt.used || d.attempt.files != 2 || d.attempt.original == nil || d.attempt.original.result == "" ||
		o.order.next != 0 || o.order.sent != 1 || o.order.state != providerHistoryOrderFailed || o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil {
		t.Fatal("post-wire cancellation published a completed prefix or skipped original end verification")
	}
	recovery := o.parent.recovery
	if recovery.completed != 0 || recovery.sent != 1 || recovery.members[0] != nil || recovery.memberCapture == nil ||
		recovery.memberCapture.original != d.attempt.original || !recovery.memberSelectionLocked(o) {
		t.Fatal("physical complete originals after cancellation became a completed member checkpoint")
	}
}

func TestProviderHistoryMemberChangedObserverRolesRefuseWithoutInstallingAnAttempt(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	for _, mode := range []string{"nil-history-group", "foreign-history-group", "integration", "scan", "scan-body", "first-admission", "first-operation", "guardian", "continuing", "parent-meter", "authority", "upper", "maximum-clocks"} {
		t.Run(mode, func(t *testing.T) {
			m := o.meter
			if o.failed || m.failed || m.closed || d.attempt != nil || o.phase != "dispatch-prepared" || m.historyGroup != o || !m.authority {
				t.Fatal("healthy otherwise-valid original must precede the finite role fault")
			}
			upper, maximum, authority := m.upper, m.maximumClocks, m.authority
			switch mode {
			case "nil-history-group":
				m.historyGroup = nil
			case "foreign-history-group":
				m.historyGroup = &providerHistoryGroupOwner{}
			case "integration":
				m.integration = &providerIntegration{}
			case "scan":
				m.scan = &providerScanControl{}
			case "scan-body":
				m.scanBody = &providerScanBody{}
			case "first-admission":
				m.firstAdmission = &providerFirstAdmissionOwner{}
			case "first-operation":
				m.firstOperation = &providerFirstOperationOwner{}
			case "guardian":
				m.guardian = &providerGuardianOwner{}
			case "continuing":
				m.continuing = &providerContinuingOwner{}
			case "parent-meter":
				m.parent = &providerWorkMeter{}
			case "authority":
				m.authority = false
			case "upper":
				m.upper[10]--
			case "maximum-clocks":
				m.maximumClocks--
			}
			actual, reads, held, phase, pin := m.actual, o.store.clock.read, o.store.stored, o.phase, o.pin
			got, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, nil)
			m.historyGroup, m.integration, m.scan, m.scanBody = o, nil, nil, nil
			m.firstAdmission, m.firstOperation, m.guardian, m.continuing, m.parent = nil, nil, nil, nil, nil
			m.upper, m.maximumClocks, m.authority = upper, maximum, authority
			if err == nil || got != nil || m.actual != actual || o.store.clock.read != reads || o.store.stored != held || o.phase != phase ||
				o.pin != pin || o.failed || m.failed || m.closed || d.attempt != nil || o.store.activeMeter.Load() != nil {
				t.Fatal("detached observer role installed an attempt, observed native state or poisoned the original")
			}
		})
	}
}
