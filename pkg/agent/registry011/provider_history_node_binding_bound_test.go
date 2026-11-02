package registry011

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Derive the signing list generically from the independent consensus RLP,
// rather than copying the original per-type tuple constructor. These are
// existing finite offline values; no key or network request is produced.
func historyNodeBindingPreimage(t *testing.T, tx *types.Transaction) []byte {
	t.Helper()
	raw, err := tx.MarshalBinary()
	if err != nil || len(raw) == 0 || tx.BlobTxSidecar() != nil {
		t.Fatal("normal consensus encoding precursor", err)
	}
	if tx.Type() != types.LegacyTxType {
		raw = raw[1:]
	}
	var fields []rlp.RawValue
	if rlp.DecodeBytes(raw, &fields) != nil || len(fields) < 3 {
		t.Fatal("normal signature-tail RLP precursor")
	}
	fields = fields[:len(fields)-3]
	if tx.Type() == types.LegacyTxType && tx.Protected() {
		chain, err := rlp.EncodeToBytes(big.NewInt(11155111))
		if err != nil {
			t.Fatal("fixed chain scalar", err)
		}
		fields = append(fields, rlp.RawValue(chain), rlp.RawValue{0x80}, rlp.RawValue{0x80})
	}
	preimage, err := rlp.EncodeToBytes(fields)
	if err != nil {
		t.Fatal("independent signature-tail projection", err)
	}
	return preimage
}

func TestProviderHistoryNodeSignatureUpperMatchesIndependentWholeRows(t *testing.T) {
	for _, row := range []struct {
		e, count, v, maximum, pool, signing, sender, decoded uint64
	}{
		{65536, 1, 1024, 1280, 65792, 84345349, 84359137, 97},
		{65536, 2, 8192, 8448, 66048, 558105610, 558133186, 194},
		{67108864, 16384, 8388608, 8388864, 71303168, 598152721809408, 598152947712000, 1589248},
		{67108864, 16384, 67108864, 67109120, 71303168, 4785093000380416, 4785093226283008, 1589248},
	} {
		want := providerHistoryNodeSignatureCost{
			signing: candidateWorkVector{10: row.signing}, sender: candidateWorkVector{2: row.decoded, 10: row.sender},
			room: candidateWorkVector{10: 4503633987174401}, preimageBytes: row.maximum, preimagePool: row.pool,
		}
		if got, ok := providerHistoryNodeSignatureUpper(row.e, row.count, row.v); !ok || got != want {
			t.Fatal("independent pooled signature rows", got, want)
		}
	}
}

func TestProviderHistoryNodeSigningExactRowsMatchGenericConsensusProjection(t *testing.T) {
	for i, tx := range providerHistoryNodeJSONUnitTransactions() {
		preimage := historyNodeBindingPreimage(t, tx)
		y := uint64(len(preimage))
		units := y*y + y + 2
		signer := types.LatestSignerForChainID(big.NewInt(11155111))
		if tx.Type() == types.LegacyTxType && !tx.Protected() {
			signer = types.HomesteadSigner{}
		}
		if tx.Type() != types.LegacyTxType {
			units = y*y + 2*y + 5
		}
		want := signer.Hash(tx)
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 4503633987174401 + units})
		got, err := nodeSigningHashObserved(tx, m)
		if err != nil || got != want || m.failed || m.actual != (candidateWorkVector{10: units}) {
			t.Fatal("original signing events differ from independent RLP shape", i, err, m.actual)
		}
		raw, err := tx.MarshalBinary()
		upper, ok := providerHistoryNodeSignatureUpper(uint64(len(raw)), 1, uint64(len(raw)))
		if err != nil || !ok || y > upper.preimageBytes || upper.signing[10] < units {
			t.Fatal("canonical byte ceiling missed the real signing list", i, err)
		}
	}
}

func TestProviderHistoryNodeSenderExactRowsIgnoreWarmCacheAndAccumulate(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	tx := block.txs[0]
	want, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		t.Fatal("ordinary recorded account precursor", err)
	}
	y := uint64(len(historyNodeBindingPreimage(t, tx)))
	units := y*y + y + 2
	if tx.Type() != types.LegacyTxType {
		units = y*y + 2*y + 5
	}
	units += 13788
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 194, 10: 4503633987174401 + 2*units})
	for call := uint64(1); call <= 2; call++ {
		got, err := nodeSenderObserved(tx, m)
		if err != nil || got != want || m.failed || m.actual != (candidateWorkVector{2: 97 * call, 10: units * call}) {
			t.Fatal("warm cache hid an original recovery or observer was reset", call, err, m.actual)
		}
	}
	raw, err := tx.MarshalBinary()
	upper, ok := providerHistoryNodeSignatureUpper(2*uint64(len(raw)), 2, uint64(len(raw)))
	if err != nil || !ok || upper.sender[2] < m.actual[2] || upper.sender[10] < m.actual[10] {
		t.Fatal("two actual passes exceeded the original pooled byte/count row", err)
	}
}

func TestProviderHistoryNodeSenderPreflightAndRecoveryShortagesKeepActualPrefixes(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	tx := block.txs[0]
	y := uint64(len(historyNodeBindingPreimage(t, tx)))
	signing := y*y + y + 2
	if tx.Type() != types.LegacyTxType {
		signing = y*y + 2*y + 5
	}
	for _, row := range []struct {
		limit, prefix candidateWorkVector
	}{
		{candidateWorkVector{2: 97, 10: 4503633987174466}, candidateWorkVector{10: 66}},
		{candidateWorkVector{2: 96, 10: 4503633987174401 + signing + 13788}, candidateWorkVector{10: signing + 4292}},
	} {
		m := providerHistoryContentUnitMeter(row.limit)
		if got, err := nodeSenderObserved(tx, m); err == nil || got != (common.Address{}) || !m.failed || m.actual != row.prefix {
			t.Fatal("original preflight/recovery refusal leaked account or lost prefix", err, m.actual, row.prefix)
		}
		before := m.actual
		if got, err := nodeSenderObserved(tx, m); err == nil || got != (common.Address{}) || m.actual != before {
			t.Fatal("same failed sender observer resumed")
		}
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 4503633987174400})
	if _, err := nodeSigningHashObserved(tx, m); err == nil || !m.failed || m.actual != (candidateWorkVector{}) {
		t.Fatal("signing preflight was bypassed or recorded as actual", err, m.actual)
	}
}

func TestProviderHistoryJSONBindingUpperMatchesIndependentWholeRows(t *testing.T) {
	for _, row := range []struct{ source, canonical, decoded, units, room uint64 }{
		{4, 1028, 2060, 3185046, 4296015937},
		{108, 1132, 2372, 3871758, 4323325505},
		{1024, 2048, 5120, 13657386, 4567597057},
		{65536, 66560, 198656, 17586403626, 38654705665},
		{67108864, 67109888, 201328640, 18014811567700266, 18031994990493697},
	} {
		want := providerHistoryJSONBindingCost{
			binding: candidateWorkVector{2: row.decoded, 10: row.units}, room: candidateWorkVector{10: row.room},
			sourceBytes: row.source, canonicalBytes: row.canonical,
		}
		if got, ok := providerHistoryJSONBindingUpper(row.source); !ok || got != want {
			t.Fatal("independent tight-output and original-preflight rows", got, want)
		}
	}
}

func TestProviderHistoryJSONBindingExactWithdrawalAndOriginalPreflight(t *testing.T) {
	v := types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	fields, raw := providerHistoryNodeJSONUnitFields(t, v)
	if len(raw) != 108 {
		t.Fatal("fixed ordinary hex-value JSON precursor", len(raw))
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 226, 10: 4323325505 + 27542})
	if !nodeJSONCanonicalObserved(fields, v, "", len(raw), m) || m.failed || m.actual != (candidateWorkVector{2: 226, 10: 27542}) {
		t.Fatal("four independent scalar lengths did not produce the exact binder row", m.actual)
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{2: 226, 10: 4323325504})
	if nodeJSONCanonicalObserved(fields, v, "", len(raw), m) || !m.failed || m.actual != (candidateWorkVector{}) {
		t.Fatal("original larger encode preflight was narrowed to actual output", m.actual)
	}
	before := m.actual
	if nodeJSONCanonicalObserved(fields, v, "", len(raw), m) || m.actual != before {
		t.Fatal("failed canonical observer resumed")
	}
}

func historyNodeBindingCheck(t *testing.T, fields map[string]json.RawMessage, value any, extras string, source []byte) {
	t.Helper()
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 10: 1 << 60})
	projection, ok := nodeJSONTypedValueObserved(value, m)
	if !ok {
		t.Fatal("normal fixed typed projection")
	}
	prefix := m.actual
	encoded, err := json.Marshal(projection)
	upper, quoted := providerHistoryJSONBindingUpper(uint64(len(source)))
	if err != nil || !quoted || uint64(len(encoded)) > upper.canonicalBytes {
		t.Fatal("successful closed compact projection outgrew original JSON plus optional keys", err)
	}
	if !nodeJSONCanonicalObserved(fields, value, extras, len(source), m) || m.failed {
		t.Fatal("normal original canonical binding")
	}
	for i, total := range m.actual {
		if prefix[i] > 1<<40 || total < 2*prefix[i] || total-2*prefix[i] > upper.binding[i] {
			t.Fatal("same-meter typed projections or actual binder exceed the independent row", i, total, prefix[i], upper.binding[i])
		}
	}
}

func TestProviderHistoryJSONBindingClosedTypesAliasesAndEscapesFitTheSourceBound(t *testing.T) {
	for _, tx := range providerHistoryNodeJSONUnitTransactions() {
		fields, raw := providerHistoryNodeJSONUnitFields(t, tx)
		historyNodeBindingCheck(t, fields, tx, "", raw)
		if _, present := fields["yParity"]; present {
			delete(fields, "yParity")
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatal("ordinary explicit-v alias", err)
			}
			historyNodeBindingCheck(t, fields, tx, "", raw)
		}
	}
	input := replayTestInput(t).Blocks[0]
	var header types.Header
	if rlp.DecodeBytes(input.HeaderRLP, &header) != nil {
		t.Fatal("normal header precursor")
	}
	fields, _ := providerHistoryNodeJSONUnitFields(t, header)
	for key, value := range fields {
		if string(value) == "null" {
			delete(fields, key)
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal("ordinary omitted-null header", err)
	}
	historyNodeBindingCheck(t, fields, header, "", raw)
	v := types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	fields, _ = providerHistoryNodeJSONUnitFields(t, v)
	fields["amount"] = json.RawMessage(`"\u0030x3"`)
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal("ordinary escaped hex scalar", err)
	}
	historyNodeBindingCheck(t, fields, v, "", raw)
	block, original, _, _ := nodeJSONFixtureBytes(t)
	fields = nil
	if json.Unmarshal(original, &fields) != nil {
		t.Fatal("recorded original header fields")
	}
	historyNodeBindingCheck(t, fields, *block.Header(), "size totalDifficulty transactions uncles withdrawals", original)
}

func TestProviderHistoryJSONBindingDecodedShortageNeverAcceptsPartialComparison(t *testing.T) {
	v := types.Withdrawal{Index: 1, Validator: 2, Address: common.Address{1}, Amount: 3}
	fields, raw := providerHistoryNodeJSONUnitFields(t, v)
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 225, 10: 4323325505 + 27542})
	if nodeJSONCanonicalObserved(fields, v, "", len(raw), m) || !m.failed || m.actual[2] < 108 || m.actual[2] > 225 || m.actual[10] <= 23330 {
		t.Fatal("canonical map/entered field prefix was lost or partial binding accepted", m.actual)
	}
	// Map order may choose a different last scalar. Only the independently
	// guaranteed prefix and sticky refusal are asserted; no fixed order is invented.
	before := m.actual
	if nodeJSONCanonicalObserved(fields, v, "", len(raw), m) || m.actual != before {
		t.Fatal("same failed comparison observer resumed")
	}
}

func TestProviderHistoryNodeBindingInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, row := range [][3]uint64{{0, 1, 1}, {64<<20 + 1, 1, 1}, {1, 0, 1}, {1, 16385, 1}, {1, 1, 0}, {1, 1, 2}, {^uint64(0), 1, 1}} {
		if got, ok := providerHistoryNodeSignatureUpper(row[0], row[1], row[2]); ok || got != (providerHistoryNodeSignatureCost{}) {
			t.Fatal("invalid transaction domain leaked a partial quote", row, got)
		}
	}
	for _, size := range []uint64{0, 3, 64<<20 + 1, ^uint64(0)} {
		if got, ok := providerHistoryJSONBindingUpper(size); ok || got != (providerHistoryJSONBindingCost{}) {
			t.Fatal("invalid JSON domain leaked a partial quote", size, got)
		}
	}
}
