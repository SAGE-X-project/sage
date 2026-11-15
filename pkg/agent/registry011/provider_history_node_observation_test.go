package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

// Only reads the existing finite public fixture. It installs no transport,
// endpoint, tracer, request callback, node/EVM or replacement live Source.
func providerHistoryNodeRecordedContent(t *testing.T) (ReplayBlock, replayDecodedBlock) {
	t.Helper()
	block, _, _, receipts := nodeJSONFixtureBytes(t)
	header, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		t.Fatal(err)
	}
	raw := ReplayBlock{HeaderRLP: header, Receipts: receipts, Transactions: make([][]byte, len(block.Transactions()))}
	for i, tx := range block.Transactions() {
		raw.Transactions[i], err = tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
	}
	decoded, err := replayDecodeContent(context.Background(), raw, &replayBudget{limits: replayTestLimits()})
	if err != nil || len(decoded.txs) != 120 {
		t.Fatal("recorded binary content precursor", err)
	}
	return raw, decoded
}

func providerHistoryNodeFixtureResult(t *testing.T, raw []byte) json.RawMessage {
	t.Helper()
	var envelope struct{ Result json.RawMessage }
	if json.Unmarshal(raw, &envelope) != nil || len(envelope.Result) == 0 {
		t.Fatal("normal recorded envelope")
	}
	return envelope.Result
}

func TestProviderHistoryNodeGasObservationPreservesRecordedStrictChecks(t *testing.T) {
	raw, block := providerHistoryNodeRecordedContent(t)
	trace := providerHistoryNodeFixtureResult(t, gasWire(t))
	raw.Gas = &ReplayGasEvidence{SepoliaAmsterdamGasProfile, block.header.Hash().Hex(), trace}
	legacy := replayBudget{limits: replayTestLimits()}
	if err := replayCheckGas(context.Background(), raw, block, &legacy); err != nil {
		t.Fatal("normal fixed-profile precursor", err)
	}
	m := providerReplayUnitMeter()
	observed := replayBudget{limits: replayTestLimits(), observer: m}
	if err := replayCheckGas(context.Background(), raw, block, &observed); err != nil || m.failed || m.actual[2] < uint64(len(trace)) || m.actual[10] == 0 || observed.bytes != legacy.bytes {
		t.Fatal("recorded gas dimensions or original observer", err, m.actual)
	}
	for i, count := range m.actual {
		if i != 2 && i != 10 && count != 0 {
			t.Fatal("gas JSON invented network/semantic/disk operations", m.actual)
		}
	}
	profile, err := replayGasProfileObserved(trace, 120, providerReplayUnitMeter())
	if err != nil || profile != SepoliaAmsterdamGasProfile {
		t.Fatal("same exact native gas schema", profile, err)
	}
	missing := raw
	missing.Gas = nil
	if err := replayCheckGas(context.Background(), missing, block, &replayBudget{limits: replayTestLimits(), observer: providerReplayUnitMeter()}); err == nil {
		t.Fatal("observed JSON made missing native gas evidence acceptable")
	}
	unfunded := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
	b := replayBudget{limits: replayTestLimits(), observer: unfunded}
	if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || !unfunded.failed || unfunded.actual[10] == 0 {
		t.Fatal("original JSON byte shortage lost earlier header work", err, unfunded.actual)
	}
	before := unfunded.actual
	if err := replayCheckGas(context.Background(), raw, block, &b); err == nil || unfunded.actual != before {
		t.Fatal("gas parsing reset or reused the failed original observer")
	}
}

func TestProviderHistoryNodeRootGasObservationPreservesOriginalWitnessBytes(t *testing.T) {
	raw, block := providerHistoryNodeRecordedContent(t)
	trace := providerHistoryNodeFixtureResult(t, rootGasWire(t))
	legacy := replayBudget{limits: replayTestLimits()}
	wantProfile, want, err := nodeRootGas(context.Background(), trace, block, &legacy)
	if err != nil || wantProfile != SepoliaAmsterdamRootRegularGasProfile {
		t.Fatal("normal top-only root precursor", err)
	}
	m := providerReplayUnitMeter()
	budget := replayBudget{limits: replayTestLimits(), observer: m}
	profile, got, err := nodeRootGas(context.Background(), trace, block, &budget)
	if err != nil || profile != wantProfile || !bytes.Equal(got, want) || len(got) > len(trace) || budget.bytes != legacy.bytes || m.failed || m.actual[2] < uint64(len(trace)) {
		t.Fatal("fixed root dimensions, derived byte contract or encoder upper", err, m.actual)
	}
	raw.Gas = &ReplayGasEvidence{profile, block.header.Hash().Hex(), got}
	if err := replayCheckGas(context.Background(), raw, block, &replayBudget{limits: replayTestLimits(), observer: providerReplayUnitMeter()}); err != nil {
		t.Fatal("derived root witness bypassed or disagreed with strict settlement", err)
	}
	if _, _, err := nodeRootGas(context.Background(), trace, block, nil); err == nil {
		t.Fatal("missing original replay budget")
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{})
	if profile, result, err := nodeRootGas(context.Background(), trace, block, &replayBudget{limits: replayTestLimits(), observer: m}); err == nil || profile != "" || result != nil || !m.failed {
		t.Fatal("unfunded root parse supplied a partial witness", profile, err)
	}
}

func TestProviderHistoryNodeFrameObservationPreservesOwnedAndRecordedData(t *testing.T) {
	owned := replayTestInput(t).Blocks[0].Executions[0].Root
	input, err := json.Marshal(nodeTestFrame(owned))
	if err != nil {
		t.Fatal(err)
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 10: 1 << 60})
	want, oldErr := DecodeCallTracer(context.Background(), input, replayTestLimits())
	got, err := decodeCallTracerObserved(context.Background(), input, replayTestLimits(), m)
	if err != nil || oldErr != nil || !reflect.DeepEqual(got, want) || m.actual[5] != 2 || m.actual[6] != 1 || m.actual[2] < uint64(len(input)) || m.failed {
		t.Fatal("owned two-frame one-log precursor", err, oldErr, m.actual)
	}
	for _, filename := range []string{"testdata/erigon-core-trace.json", "testdata/node-deployment-nethermind-call-rpc.json"} {
		wire, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		raw := providerHistoryNodeFixtureResult(t, wire)
		want, oldErr := DecodeCallTracer(context.Background(), raw, replayTestLimits())
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 10: 1 << 60})
		got, err := decodeCallTracerObserved(context.Background(), raw, replayTestLimits(), m)
		if err != nil || oldErr != nil || !reflect.DeepEqual(got, want) || m.actual[5] == 0 || m.failed {
			t.Fatal("recorded frame diagnostics changed", filename, err, oldErr, m.actual)
		}
	}
}

func TestProviderHistoryNodeFrameOriginalShortagesKeepPartialResultsClosed(t *testing.T) {
	input, err := json.Marshal(nodeTestFrame(replayTestInput(t).Blocks[0].Executions[0].Root))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		upper candidateWorkVector
	}{
		{"work", candidateWorkVector{}},
		{"bytes", candidateWorkVector{5: 65536, 6: 65536, 10: 1 << 60}},
		{"second-frame", candidateWorkVector{2: 1 << 30, 5: 1, 6: 65536, 10: 1 << 60}},
		{"first-log", candidateWorkVector{2: 1 << 30, 5: 65536, 10: 1 << 60}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerHistoryContentUnitMeter(tc.upper)
			got, err := decodeCallTracerObserved(context.Background(), input, replayTestLimits(), m)
			if err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || !m.failed {
				t.Fatal("partial frame escaped the original allowance", err, m.actual)
			}
			before := m.actual
			if _, err := decodeCallTracerObserved(context.Background(), input, replayTestLimits(), m); err == nil || m.actual != before {
				t.Fatal("failed frame observer was renewed")
			}
		})
	}
	limits := replayTestLimits()
	limits.MaxFrames = 1
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 10: 1 << 60})
	if _, err := decodeCallTracerObserved(context.Background(), input, limits, m); err == nil || m.actual[5] != 1 || m.actual[6] != 0 {
		t.Fatal("unentered child/logs reported as processed", err, m.actual)
	}
}

func TestProviderHistoryNodeFrameStorageCountsConcreteCopies(t *testing.T) {
	if uint64(reflect.TypeFor[ReplayFrame]().Size()) != nodeFrameCopyBytes || uint64(reflect.TypeFor[ReplayFrameLog]().Size()) != nodeFrameLogCopyBytes {
		t.Fatal("concrete frame schema no longer matches the original copy sizes")
	}
	var frame ReplayFrame
	m := providerReplayUnitMeter()
	child := ReplayFrame{Type: "CALL", From: replayTestWallet}
	entry := ReplayFrameLog{Address: SepoliaRegistryAddress, AfterChildren: 1}
	if !nodeFrameStorageObserved(&frame, 1, 1, m) || !nodeFrameCopyObserved(&frame, 0, child, m) || !nodeFrameLogCopyObserved(&frame, 0, entry, m) || frame.Calls[0].From != child.From || frame.Logs[0].AfterChildren != 1 {
		t.Fatal("owned array initialization and concrete copies")
	}
	upper := m.actual
	upper[10]--
	short := providerHistoryContentUnitMeter(upper)
	var partial ReplayFrame
	if !nodeFrameStorageObserved(&partial, 1, 1, short) || !nodeFrameCopyObserved(&partial, 0, child, short) || nodeFrameLogCopyObserved(&partial, 0, entry, short) || !short.failed || partial.Logs[0].Address != "" {
		t.Fatal("late copy was performed without its original allowance", short.actual)
	}
	for _, counts := range [][2]int{{-1, 0}, {0, -1}, {65537, 0}, {0, 65537}} {
		if nodeFrameStorageObserved(new(ReplayFrame), counts[0], counts[1], providerReplayUnitMeter()) {
			t.Fatal("out-of-range frame/log storage", counts)
		}
	}
	if nodeFrameStorageObserved(nil, 0, 0, m) || nodeFrameStorageObserved(new(ReplayFrame), 0, 0, nil) || nodeFrameStorageObserved(&frame, 1, 1, m) || nodeFrameCopyObserved(&frame, -1, child, m) || nodeFrameLogCopyObserved(&frame, 1, entry, m) {
		t.Fatal("missing, existing or out-of-range storage")
	}
}
