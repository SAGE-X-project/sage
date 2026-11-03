package registry011

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// This is a separate normal owned-runtime UNIT subject. Its one complete
// public deployment block is not a substitute for deployment-to-current
// qualification. Every numeric selection precedes profile/anchor/H0; none
// supplies a production source/native/RAM derivation or operating policy.
func historyCollectionRecordedRunManifest(t *testing.T) (*providerHistoryCollectionManifest, map[string]json.RawMessage) {
	t.Helper()
	recorded := nodeRecorded(t)
	var envelope struct{ Result json.RawMessage }
	if json.Unmarshal(recorded["canonical-deployment"], &envelope) != nil {
		t.Fatal("recorded deployment header envelope")
	}
	header, err := providerDecodeHeader(envelope.Result)
	if err != nil || header.Number != sepoliaDeploymentBlock || header.Timestamp < sepoliaAmsterdamTime {
		t.Fatal("complete original deployment header", err)
	}
	in := historyCollectionInput(t)
	// Keep the journal's required chain-qualified Registry identity.
	in.journal.Deployment = header
	in.journal.CodeIdentity, in.journal.ABIIdentity = SepoliaEvidenceDigests()
	in.journal.Replay.MaxTransactions, in.journal.Replay.MaxFrames = 1024, 256
	in.journal.Replay.MaxLogs, in.journal.Replay.MaxEvidenceBytes = 4096, 8<<20
	in.encoding, in.gas = "debug-raw-rlp-v1", SepoliaAmsterdamGasProfile
	in.maximumSelected, in.blocks, in.native = header.Number, 1, 1
	in.effects, in.versions, in.presentStops = 0, 0, 0
	in.windowBlocks, in.contentBlocks = 1, 1
	in.capture.BodyBytes = 240 << 10
	in.firstRead.raw = 8 << 20
	manifest := historyCollectionManifest(t, in)
	if manifest.counts.groups != 4 || manifest.counts.requests != 29 {
		t.Fatal("recorded absent grammar differs from handwritten 18/3/1/7 members")
	}
	return manifest, recorded
}

func historyCollectionRecordedRunStore(t *testing.T) (*providerCustodyStore, providerHistoryCollectionInput, map[string]json.RawMessage) {
	t.Helper()
	manifest, recorded := historyCollectionRecordedRunManifest(t)
	in := manifest.input
	store := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{
		capture: &in.capture, pin: true, pinLargeObject: true, historyProgram: true,
		historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyRecoveryNamespace: true, historyRecoveryLegacy: true,
		historyCollection: manifest, historyCollectionAdmission: true, historyCollectionFirstP: true,
		historyCollectionFullPack: true, historyCollectionRecordedRun: true,
	})
	first, err := newProviderFirstOperationCandidate(store)
	if err != nil {
		t.Fatal("original first owner", err)
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, string(recorded["canonical-deployment"]))
	if result, err := first.executeCandidateWithConnection(context.Background(), connection); err != nil || result == nil {
		t.Fatal("original recorded deployment first P", err)
	}
	return store, in, recorded
}

// Check the SAME recorded manifest used by every controller fixture before
// a store, original owner, clock or transport is created. This early regression
// proves only input configuration; the full runtime tests remain mandatory.
func TestProviderHistoryCollectionRecordedManifestKeepsJournalIdentity(t *testing.T) {
	manifest, recorded := historyCollectionRecordedRunManifest(t)
	if !manifest.validSnapshot() || manifest.input.journal.Registry != "eip155:11155111:"+SepoliaRegistryAddress ||
		manifest.input.journal.Deployment.Number != sepoliaDeploymentBlock || manifest.input.maximumSelected != sepoliaDeploymentBlock ||
		manifest.counts.groups != 4 || manifest.counts.requests != 29 || len(recorded["canonical-deployment"]) == 0 {
		t.Fatal("shared recorded fixture lost its chain-qualified Registry or selected deployment")
	}
}

// Handwritten independent request order, rather than invoking the collector's
// planner to prepare its own expected traffic. All responses are normal public
// originals or the ordinary absent-record ABI. No external RPC is contacted.
func historyCollectionRecordedRunConnections(t *testing.T, in providerHistoryCollectionInput, recorded map[string]json.RawMessage) []*tls.Conn {
	t.Helper()
	var connections []*tls.Conn
	add := func(method string, params any, response []byte) {
		request, err := newProviderIntegrationRequest(method, params)
		if err != nil || request == nil || len(response) == 0 {
			t.Fatal("fixed normal recorded request/response", method, err)
		}
		connections = append(connections, providerIntegrationOwnedTLS(t, request.body, string(response)))
	}
	header := recorded["canonical-deployment"]
	selected := in.journal.Deployment
	block := map[string]any{"blockHash": selected.Hash, "requireCanonical": true}
	identity := []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x04ef898d"}, block}
	add("eth_chainId", []any{}, recorded["chain"])
	add("eth_getBlockByNumber", []any{"0x0", false}, recorded["genesis"])
	add("eth_call", identity, recorded["identity-at-anchor"])
	add("web3_clientVersion", []any{}, recorded["client"])
	add("eth_getBlockByNumber", []any{"finalized", false}, header)
	for _, code := range in.codes {
		add("eth_getCode", []any{code.Address, block}, recorded["code-"+code.Name])
	}
	add("debug_getRawBlock", []any{selected.Hash}, recorded["raw-block"])
	add("debug_getRawReceipts", []any{selected.Hash}, recorded["raw-receipts"])
	gas, err := os.ReadFile("testdata/node-deployment-gas-rpc.json")
	if err != nil {
		t.Fatal("recorded selected gas response", err)
	}
	add("debug_traceBlockByHash", []any{selected.Hash, map[string]any{"tracer": "stateGasTracer", "reexec": 0, "timeout": in.traceTimeout.String()}}, gas)
	params, err := providerHistoryRegistryParamsObserved(SepoliaRegistryAddress, in.journal.Agent, "record", 0, selected.Hash, providerReplayUnitMeter())
	if err != nil {
		t.Fatal("ordinary absent-record request", err)
	}
	absent := replayTestJSON(t, map[string]any{"jsonrpc": "2.0", "id": 1, "result": returnABI(t, "record", false, []byte{})})
	add("eth_call", params, absent)
	tag := "0x" + strconv.FormatUint(selected.Number, 16)
	add("eth_getBlockByNumber", []any{tag, false}, header)
	add("eth_getBlockByNumber", []any{tag, false}, header)
	add("eth_getBlockByNumber", []any{"finalized", false}, header)
	add("eth_chainId", []any{}, recorded["chain"])
	add("eth_getBlockByNumber", []any{"0x0", false}, recorded["genesis"])
	add("eth_call", identity, recorded["identity-at-anchor"])
	add("web3_clientVersion", []any{}, recorded["client"])
	if len(connections) != 29 {
		t.Fatal("handwritten normal request count")
	}
	return connections
}

func TestProviderHistoryCollectionRunConnectsAllOriginalStagesWithRecordedNormalTLS(t *testing.T) {
	store, in, recorded := historyCollectionRecordedRunStore(t)
	parent := store.history
	held, original, ceiling, future := store.stored, parent.original, parent.clockCeiling, parent.bounds.future
	connections := historyCollectionRecordedRunConnections(t, in, recorded)
	out, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections)
	if err != nil || out == nil {
		t.Fatal("normal owned collection did not reach its private final S", err)
	}
	group := parent.group
	if group == nil || group.failed || group.phase != "settled" || out.owner != out || out.group != group || out.ack != group.last ||
		group.coordinator.stage != 4 || group.custodian.stage != 4 || group.coordinator.sequence != 12 ||
		group.coordinator.collection.phase != "done" || group.custodian.collection.phase != "done" ||
		!group.coordinator.collection.semantic.atEnd() || !group.custodian.collection.semantic.atEnd() ||
		group.coordinator.collection.semantic.confirmed.state.version != 0 || group.meter.actual[0] != 29 ||
		group.coordinator == group.custodian || group.coordinator.collection == group.custodian.collection ||
		group.coordinator.collection.semantic == group.custodian.collection.semantic ||
		store.stored != held || parent.original != original || parent.clockCeiling != ceiling || parent.bounds.future != future ||
		store.activeMeter.Load() != nil || store.failed || parent.failed || group.meter.failed ||
		parent.recovery.confirmed == nil || parent.recovery.confirmed.ack != out.ack {
		t.Fatal("collection changed its original lifetime or shared independently reconstructed data")
	}
	work, clocks, pin := group.meter.actual, store.clock.read, group.pin
	if repeated, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections); err == nil || repeated != nil ||
		group.meter.actual != work || store.clock.read != clocks || group.pin != pin || group.failed {
		t.Fatal("completed original collection was entered a second time")
	}
}

// A fixed missing-fixture UNIT value stops before any member dial/write.
// There is no hostile responder or malformed network traffic. T and C have
// already been retained by the real owners, so their debit must remain final.
func TestProviderHistoryCollectionRunStopsAfterConsumptionWithoutRetryOrRefund(t *testing.T) {
	store, _, _ := historyCollectionRecordedRunStore(t)
	parent := store.history
	held, original, ceiling := store.stored, parent.original, parent.clockCeiling
	connections := make([]*tls.Conn, 29)
	if out, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections); err == nil || out != nil {
		t.Fatal("missing owned UNIT connection produced a completed collection")
	}
	group := parent.group
	if group == nil || !group.failed || !parent.failed || group.phase != "failed" ||
		group.coordinator == nil || group.coordinator.phase != "consumed" || group.coordinator.stage != 0 ||
		group.order == nil || group.order.consumed != (1<<18)-1 || group.order.sent != 0 || group.order.next != 0 ||
		group.meter.actual[0] != 0 || group.meter.actual[1] != 0 || group.dispatch == nil || group.dispatch.attempt != nil ||
		store.stored != held || parent.original != original || parent.clockCeiling != ceiling ||
		parent.recovery.confirmed == nil || parent.recovery.confirmed.frame.phase != "consume" ||
		parent.recovery.confirmed.ack != group.last {
		t.Fatal("terminal driver error lost the original consumed prefix or manufactured a member attempt")
	}
	work, clocks, pin := group.meter.actual, store.clock.read, group.pin
	originals := providerHistoryProvisionedOriginalBytes(t, store.config)
	if repeated, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections); err == nil || repeated != nil ||
		group.meter.actual != work || store.clock.read != clocks || group.pin != pin {
		t.Fatal("failed collection renewed its original observation")
	}
	after := providerHistoryProvisionedOriginalBytes(t, store.config)
	if len(after) != len(originals) {
		t.Fatal("failed re-entry changed retained original files")
	}
	for name, raw := range originals {
		if after[name] != raw {
			t.Fatal("failed re-entry rewrote an original", name)
		}
	}
}

// Only ordinary file synchronization inside t.TempDir fails. These cases
// cross the new controller boundary with real T/C/S owners; no network fault
// producer, external peer, replacement clock or operating allowance is used.
func TestProviderHistoryCollectionRunPreservesCommittedPrefixAfterStorageFailure(t *testing.T) {
	for _, fault := range []struct {
		phase    string
		stage    uint64
		requests uint64
	}{
		{"admit", 1, 0}, {"consume", 1, 0}, {"settle", 1, 18}, {"admit", 2, 18},
	} {
		t.Run(fault.phase+"-"+strconv.FormatUint(fault.stage, 10), func(t *testing.T) {
			store, in, recorded := historyCollectionRecordedRunStore(t)
			parent := store.history
			held, original, ceiling, future := store.stored, parent.original, parent.clockCeiling, parent.bounds.future
			connections := historyCollectionRecordedRunConnections(t, in, recorded)
			syncFile := store.evidence.syncFile
			reached := false
			var confirmed *providerHistoryRecoveryCheckpoint
			var ack *providerHistoryGroupAcknowledgment
			var pin providerFrontierHighwater
			store.evidence.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				claim := store.frontierFiles.historyPending
				if claim == nil || claim.frame.stage != fault.stage || claim.frame.phase != fault.phase || claim.count < 1 ||
					filepath.Base(file.Name()) != providerHistoryStoredName(claim.inventory.entries[0], nil) {
					return nil
				}
				reached = true
				confirmed, ack, pin = parent.recovery.confirmed, parent.group.last, parent.group.pin
				return errProviderIntegration
			}
			out, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections)
			store.evidence.syncFile = syncFile
			group := parent.group
			if err == nil || out != nil || !reached || group == nil || confirmed == nil ||
				parent.recovery.confirmed != confirmed || group.last != ack || group.pin != pin ||
				group.meter.actual[0] != fault.requests || !group.failed || !parent.failed || !store.failed ||
				!group.meter.failed || !group.meter.closed || store.activeMeter.Load() != nil ||
				store.stored != held || parent.original != original || parent.clockCeiling != ceiling || parent.bounds.future != future {
				t.Fatal("storage failure advanced the acknowledged prefix or renewed the original lifetime", err)
			}
			if fault.stage == 2 {
				archive := confirmed.memberArchive
				if confirmed.frame.stage != 1 || confirmed.frame.phase != "settle" || archive == nil ||
					!archive.shapeLocked(parent.recovery, confirmed) || archive.order.consumed != (1<<18)-1 || archive.order.sent != (1<<18)-1 {
					t.Fatal("later T failure lost the earlier complete original stage")
				}
			}
			historyCollectionRunAssertNoReentry(t, store, connections)
		})
	}
}

// A standard JSON-RPC internal-error reply is a benign service failure. The
// original member owns its terminal failed S; the controller must preserve
// that exact checkpoint and must not settle successfully or continue later.
func TestProviderHistoryCollectionRunPreservesKnownFailedSettlement(t *testing.T) {
	for _, ordinal := range []int{0, 18} {
		t.Run(strconv.Itoa(ordinal), func(t *testing.T) {
			store, in, recorded := historyCollectionRecordedRunStore(t)
			parent := store.history
			held, original, ceiling, future := store.stored, parent.original, parent.clockCeiling, parent.bounds.future
			connections := historyCollectionRecordedRunConnections(t, in, recorded)
			method, params := "eth_chainId", []any{}
			stage, count := uint64(1), 18
			if ordinal == 18 {
				method, params = "debug_getRawBlock", []any{in.journal.Deployment.Hash}
				stage, count = 2, 3
			}
			request, err := newProviderIntegrationRequest(method, params)
			if err != nil {
				t.Fatal("ordinary service-error request", err)
			}
			response := `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"internal error"}}`
			connections[ordinal] = providerIntegrationOwnedTLS(t, request.body, response)
			out, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections)
			group := parent.group
			if err == nil || out != nil || group == nil || !group.failed || !parent.failed || !store.failed ||
				!group.meter.failed || !group.meter.closed || store.activeMeter.Load() != nil ||
				group.meter.actual[0] != uint64(ordinal+1) || group.meter.actual[1] == 0 ||
				group.order == nil || group.order.state != providerHistoryOrderFailed || group.order.consumed != (1<<count)-1 ||
				group.order.sent != 1 || group.order.next != 0 || group.dispatch == nil || group.dispatch.retained != 0 ||
				store.stored != held || parent.original != original || parent.clockCeiling != ceiling || parent.bounds.future != future {
				t.Fatal("member failure was retried, refunded or promoted to collection success", err)
			}
			checkpoint := parent.recovery.confirmed
			if checkpoint == nil || !checkpoint.failedS || checkpoint.frame.stage != stage || checkpoint.frame.phase != "settle" ||
				checkpoint.ack != group.last || checkpoint.pin != group.pin || group.pin != store.frontierFiles.pin ||
				checkpoint.previous == nil || checkpoint.previous.frame.phase != "consume" || parent.recovery.pending != nil ||
				parent.recovery.completed != 0 || parent.recovery.sent != 1 || parent.recovery.memberAttempt == nil ||
				parent.recovery.memberAttempt.ordinal != 0 || parent.recovery.memberCapture != nil {
				t.Fatal("controller cleanup replaced the original member's terminal failed S")
			}
			if ordinal == 18 {
				previous := checkpoint.admission.checkpoint.previous
				if previous == nil || previous.frame.stage != 1 || previous.memberArchive == nil ||
					!previous.memberArchive.shapeLocked(parent.recovery, previous) || previous.memberArchive.order.sent != (1<<18)-1 {
					t.Fatal("known failure in the second stage lost the first original archive")
				}
			}
			historyCollectionRunAssertNoReentry(t, store, connections)
		})
	}
}

// Cancel through an ordinary context after the real owned TLS original sync.
// The synchronized bytes remain evidence, but cannot become a completed
// member, a successful S or a new lifetime after the cancellation.
func TestProviderHistoryCollectionRunStopsAfterEnteredMemberCancellation(t *testing.T) {
	store, in, recorded := historyCollectionRecordedRunStore(t)
	parent := store.history
	held, original, ceiling, future := store.stored, parent.original, parent.clockCeiling, parent.bounds.future
	connections := historyCollectionRecordedRunConnections(t, in, recorded)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	syncFile := store.evidence.syncFile
	cancelled := false
	store.evidence.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		attempt := parent.recovery.memberAttempt
		if attempt != nil && parent.group.phase == "retaining-member" && filepath.Base(file.Name()) == attempt.tlsName {
			cancelled = true
			cancel()
		}
		return nil
	}
	out, err := parent.runCollectionWithConnectionsCandidate(ctx, connections)
	store.evidence.syncFile = syncFile
	group := parent.group
	if err == nil || out != nil || !cancelled || ctx.Err() != context.Canceled || group == nil ||
		!group.failed || !parent.failed || !store.failed || !group.meter.failed || !group.meter.closed || store.activeMeter.Load() != nil ||
		group.meter.actual[0] != 1 || group.meter.actual[1] == 0 || group.order == nil || group.order.sent != 1 || group.order.next != 0 ||
		group.order.consumed != (1<<18)-1 || group.dispatch == nil || group.dispatch.retained != 0 ||
		group.dispatch.attempt == nil || group.dispatch.attempt.files != 2 || group.dispatch.attempt.original == nil ||
		store.stored != held || parent.original != original || parent.clockCeiling != ceiling || parent.bounds.future != future {
		t.Fatal("entered cancellation published success or renewed the original observation", err)
	}
	checkpoint := parent.recovery.confirmed
	if checkpoint == nil || checkpoint.failedS || checkpoint.frame.phase != "consume" || checkpoint.ack != group.last ||
		checkpoint.pin != group.pin || parent.recovery.completed != 0 || parent.recovery.memberCapture == nil ||
		parent.recovery.memberCapture.original != group.dispatch.attempt.original {
		t.Fatal("cancelled physical member originals became a completed logical prefix")
	}
	historyCollectionRunAssertNoReentry(t, store, connections)
}

func historyCollectionRunAssertNoReentry(t *testing.T, store *providerCustodyStore, connections []*tls.Conn) {
	t.Helper()
	parent, group := store.history, store.history.group
	work, clocks, pin, ack, checkpoint := group.meter.actual, store.clock.read, group.pin, group.last, parent.recovery.confirmed
	held, original, ceiling := store.stored, parent.original, parent.clockCeiling
	before := providerHistoryProvisionedOriginalBytes(t, store.config)
	if out, err := parent.runCollectionWithConnectionsCandidate(context.Background(), connections); err == nil || out != nil ||
		group.meter.actual != work || store.clock.read != clocks || group.pin != pin || group.last != ack ||
		parent.recovery.confirmed != checkpoint || store.stored != held || parent.original != original || parent.clockCeiling != ceiling {
		t.Fatal("failed collection reentry changed its original work, clock, hold or checkpoint")
	}
	after := providerHistoryProvisionedOriginalBytes(t, store.config)
	if len(before) != len(after) {
		t.Fatal("failed reentry changed retained originals")
	}
	for name, raw := range before {
		if after[name] != raw {
			t.Fatal("failed reentry rewrote an original", name)
		}
	}
}
