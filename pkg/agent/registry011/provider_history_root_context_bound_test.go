package registry011

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestProviderHistoryRootContextEnvelopeMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct{ body, transaction, decoded, logical uint64 }{
		{4, 67108864, 169, 4503634054310228},
		{1024, 2048, 2209, 7439648},
		{131072, 132096, 262305, 51877604640},
		{8388608, 8389632, 16777377, 211127759367456},
		{8388608, 67108864, 16777377, 4644371584608544},
	} {
		got, ok := providerHistoryRootContextRefusalUpper(row.body, row.transaction)
		want := candidateWorkVector{2: row.decoded, 10: row.logical}
		if !ok || got != want {
			t.Fatal("independent complete-root scalar/native row", row, got, want, ok)
		}
	}
}

func TestProviderHistoryRootContextEnvelopeRejectsOutOfDomainData(t *testing.T) {
	for _, row := range []struct{ body, transaction uint64 }{
		{0, 1024}, {3, 1024}, {(8 << 20) + 1, 1024},
		{1024, 0}, {1024, (64 << 20) + 1}, {^uint64(0), 1024}, {1024, ^uint64(0)},
	} {
		if got, ok := providerHistoryRootContextRefusalUpper(row.body, row.transaction); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid input published partial work", row, got, ok)
		}
	}
}

func historyRootContextString(t *testing.T, s string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Existing public finite V0/R1/S1 DATA, not an account key, broadcast, network
// request, deployment or live owner. The independent library supplies expected
// sender/destination text; the original observed route still recovers afresh.
func historyRootContextFixture(t *testing.T, create bool) (map[string]json.RawMessage, *types.Transaction) {
	t.Helper()
	destination := common.HexToAddress(SepoliaRegistryAddress)
	var to *common.Address
	kind := "CREATE"
	if !create {
		to, kind = &destination, "CALL"
	}
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2),
		Gas: 21000, To: to, Value: new(big.Int), Data: []byte{1, 2}, V: big.NewInt(0), R: big.NewInt(1), S: big.NewInt(1)})
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		t.Fatal("public finite recovery precursor", err)
	}
	if create {
		destination = crypto.CreateAddress(from, tx.Nonce())
	}
	return map[string]json.RawMessage{
		"from": historyRootContextString(t, replayAddress(from)), "type": historyRootContextString(t, kind),
		"gas": historyRootContextString(t, "0x5208"), "input": historyRootContextString(t, "0x0102"),
		"value": historyRootContextString(t, "0x0"), "to": historyRootContextString(t, replayAddress(destination)),
		"output": historyRootContextString(t, "0x"), "stateGasReservoir": historyRootContextString(t, "0x1"),
	}, tx
}

func historyRootContextQuote(t *testing.T, fields map[string]json.RawMessage) (candidateWorkVector, []byte) {
	t.Helper()
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	// Do not manufacture a collection owner to narrow the original native
	// preflight. Every inert meter below retains its real64MiB old-role ceiling.
	upper, ok := providerHistoryRootContextRefusalUpper(uint64(len(raw)), 64<<20)
	if !ok {
		t.Fatal("finite original root pool")
	}
	return upper, raw
}

func TestProviderHistoryRootContextEnvelopeKeepsAddressAfterSenderRefusal(t *testing.T) {
	for _, absent := range []bool{true, false} {
		fields, _ := historyRootContextFixture(t, false)
		upper, original := historyRootContextQuote(t, fields)
		prefix := providerHistoryContentUnitMeter(upper)
		if !historyJSONRootPrefixCalls(fields, prefix) || prefix.failed {
			t.Fatal("seven valid original scalar calls")
		}
		var tx *types.Transaction
		want := prefix.actual
		if !absent {
			tx = types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), Gas: 21000,
				Value: new(big.Int), V: big.NewInt(0), R: big.NewInt(0), S: big.NewInt(1)})
			// Original scalar validation enters compare65 before rejecting R0.
			want[10] += 66
		}
		want[10] += 1765 // encode42 is attempted even after an ordinary sender error.
		m := providerHistoryContentUnitMeter(upper)
		if nodeRootContextObserved(fields, tx, 1, m) || m.failed || m.actual != want || m.clocks != 0 {
			t.Fatal("sender refusal skipped its continued address conversion", absent, m.actual, want)
		}
		after, err := json.Marshal(fields)
		if err != nil || !bytes.Equal(original, after) {
			t.Fatal("original field DATA changed", err)
		}
		historyEventQueueWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryRootContextEnvelopeKeepsDestinationSiblings(t *testing.T) {
	for _, create := range []bool{false, true} {
		fields, tx := historyRootContextFixture(t, create)
		fields["to"] = historyRootContextString(t, "x")
		upper, _ := historyRootContextQuote(t, fields)
		wrongText := providerHistoryContentUnitMeter(upper)
		if nodeRootContextObserved(fields, tx, 1, wrongText) || wrongText.failed {
			t.Fatal("ordinary wrong destination precursor", create, wrongText.actual)
		}
		fields["to"] = json.RawMessage("true")
		upper, _ = historyRootContextQuote(t, fields)
		badShape := providerHistoryContentUnitMeter(upper)
		if nodeRootContextObserved(fields, tx, 1, badShape) || badShape.failed {
			t.Fatal("destination shape refusal", create, badShape.actual)
		}
		// Quoted x costs compare4+codec10, then a43-byte comparison44.
		// true costs compare5 and no codec/comparison. All native sibling
		// calls must remain on BOTH routes, so the difference is exactly53/3.
		want := badShape.actual
		want[2] += 3
		want[10] += 53
		if wrongText.actual != want || badShape.clocks != 0 || wrongText.clocks != 0 {
			t.Fatal("bad scalar skipped CALL address or CREATE address siblings", create, badShape.actual, wrongText.actual, want)
		}
		historyEventQueueWithin(t, badShape.actual, upper)
	}
}

func TestProviderHistoryRootContextEnvelopePreservesSuccessAndTailRefusals(t *testing.T) {
	for _, create := range []bool{false, true} {
		for _, mode := range []string{"success", "reverted", "output", "empty-error", "revert-wide", "reservoir-range", "status", "input", "amount"} {
			fields, tx := historyRootContextFixture(t, create)
			status, want := uint64(1), mode == "success" || mode == "reverted"
			switch mode {
			case "reverted":
				status, fields["error"] = 0, historyRootContextString(t, "reverted")
				fields["revertReason"] = historyRootContextString(t, "normal finite DATA")
			case "output":
				fields["output"] = historyRootContextString(t, "0xzz")
			case "empty-error":
				status, fields["error"] = 0, historyRootContextString(t, "")
			case "revert-wide":
				status, fields["error"] = 0, historyRootContextString(t, "reverted")
				fields["revertReason"] = historyRootContextString(t, strings.Repeat("z", 4097))
			case "reservoir-range":
				fields["stateGasReservoir"] = historyRootContextString(t, "0x"+strings.Repeat("f", 64))
			case "status":
				status = 2
			case "input":
				fields["input"] = historyRootContextString(t, "0x0103")
			case "amount":
				fields["value"] = historyRootContextString(t, "0x1")
			}
			upper, original := historyRootContextQuote(t, fields)
			m := providerHistoryContentUnitMeter(upper)
			if nodeRootContext(fields, tx, status) != want || nodeRootContextObserved(fields, tx, status, m) != want || m.failed ||
				m.actual[2] < 97 || m.actual[10] == 0 || m.clocks != 0 {
				t.Fatal("original root result or complete entered tail changed", create, mode, m.actual)
			}
			after, err := json.Marshal(fields)
			if err != nil || !bytes.Equal(original, after) {
				t.Fatal("root DATA changed", mode, err)
			}
			historyEventQueueWithin(t, m.actual, upper)
		}
	}
}

func TestProviderHistoryRootContextEnvelopeKeepsFailedRecoveryAllowance(t *testing.T) {
	fields, tx := historyRootContextFixture(t, true)
	upper, _ := historyRootContextQuote(t, fields)
	prefix := providerHistoryContentUnitMeter(upper)
	if !historyJSONRootPrefixCalls(fields, prefix) || prefix.failed {
		t.Fatal("ordinary seven-call scalar precursor")
	}
	// Choose the complete inert allowance before constructing the observer.
	// One byte below the concrete recovery codec cannot be renewed by a later
	// address call or a second attempt. No live collection owner is altered.
	upper[2] = prefix.actual[2] + 96
	m := providerHistoryContentUnitMeter(upper)
	if nodeRootContextObserved(fields, tx, 1, m) || !m.failed || m.actual[2] != prefix.actual[2] || m.actual[10] <= prefix.actual[10] {
		t.Fatal("unfunded recovery was entered or lost its earlier work", m.actual, prefix.actual)
	}
	before := m.actual
	if nodeRootContextObserved(fields, tx, 1, m) || m.actual != before || !m.failed || m.clocks != 0 {
		t.Fatal("failed original allowance was revived", m.actual, before)
	}
}
