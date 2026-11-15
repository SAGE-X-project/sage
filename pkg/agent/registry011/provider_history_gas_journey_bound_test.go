package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryGasJourneyMatchesPortableWholeRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	s := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	j := uint64(reflect.TypeFor[big.Int]().Size())
	g := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	for _, row := range []struct {
		limits                providerHistoryGasJourneyLimits
		profile               string
		pools                 [8]uint64
		normalization, strict [4]uint64
		assembly, gas, room   uint64
	}{
		{providerHistoryGasJourneyLimits{1, 1, 1, 65536, 1024, 512, 2048, 2, 1}, SepoliaAmsterdamRootRegularGasProfile, [8]uint64{1024, 2048, 1024, 2048, 2, 4, 2, 4}, [4]uint64{8514, 11251325, 24, 2}, [4]uint64{12676, 8606200958, 24, 4}, 1105, 1, 5308417},
		{providerHistoryGasJourneyLimits{1, 2, 2, 4096, 8192, 2048, 67108864, 4, 2}, SepoliaAmsterdamRootRegularGasProfile, [8]uint64{16384, 16384, 8192, 16384, 8, 16, 8, 16}, [4]uint64{132360, 1249143274, 96, 8}, [4]uint64{99856, 19140610996, 96, 16}, 16546, 2, 4503633987174401},
		{providerHistoryGasJourneyLimits{1, 2, 2, 4096, 8192, 2048, 67108864, 4, 2}, SepoliaAmsterdamGasProfile, [8]uint64{16384, 16384, 8192, 16384, 8, 16, 0, 16}, [4]uint64{49152, 402802128, 48, 0}, [4]uint64{99856, 19140610996, 96, 16}, 16554, 2, 4503633987174401},
		{providerHistoryGasJourneyLimits{1, 2, 2, 1, 64, 1, 1, 16, 2}, SepoliaAmsterdamGasProfile, [8]uint64{128, 4, 2, 4, 32, 2, 0, 2}, [4]uint64{384, 27240, 192, 0}, [4]uint64{218, 17180326730, 14, 2}, 298, 2, 66050},
		{providerHistoryGasJourneyLimits{3, 12, 2, 65536, 4096, 1024, 67108864, 4, 10}, SepoliaAmsterdamGasProfile, [8]uint64{40960, 81920, 40960, 81920, 40, 80, 0, 32}, [4]uint64{122880, 503692560, 240, 0}, [4]uint64{494624, 90263054948, 480, 32}, 41810, 10, 4503633987174401},
		{providerHistoryGasJourneyLimits{1, 100, 2, 4096, 8192, 1024, 2048, 4, 100}, SepoliaAmsterdamGasProfile, [8]uint64{20480, 16384, 12288, 16384, 400, 800, 0, 16}, [4]uint64{61440, 503575200, 2400, 0}, [4]uint64{99856, 859856470472, 4800, 16}, 28980, 100, 5308417},
		{providerHistoryGasJourneyLimits{1, 100, 2, 4096, 8192, 1024, 2048, 4, 100}, SepoliaAmsterdamRootRegularGasProfile, [8]uint64{819200, 16384, 12288, 16384, 400, 800, 12, 16}, [4]uint64{6555532, 60454486600, 2472, 12}, [4]uint64{99856, 859856470472, 4800, 16}, 827300, 100, 5308417},
		{providerHistoryGasJourneyLimits{1, 1, 1, 65536, 1024, 512, 67108864, 2, 0}, SepoliaAmsterdamRootRegularGasProfile, [8]uint64{0, 0, 0, 0, 0, 0, 0, 0}, [4]uint64{0, 0, 0, 0}, [4]uint64{0, 0, 0, 0}, 0, 0, 0},
		{providerHistoryGasJourneyLimits{100, 100, 10, 67108864, 8388608, 8388608, 67108864, 16384, 100}, SepoliaAmsterdamRootRegularGasProfile, [8]uint64{838860800, 1677721600, 838860800, 1677721600, 1638400, 3276800, 1638400, 3276800}, [4]uint64{6974668800, 80924468202701300, 19660800, 1638400}, [4]uint64{10384179200, 119848374983965000, 19660800, 3276800}, 838868900, 100, 4503633987174401},
	} {
		got, ok := providerHistoryGasJourneyUpper(row.limits, row.profile)
		want := providerHistoryGasJourneyCost{
			normalization: candidateWorkVector{2: row.normalization[0], 10: row.normalization[1] + row.normalization[2]*w + row.normalization[3]*s},
			strict:        candidateWorkVector{2: row.strict[0], 10: row.strict[1] + row.strict[2]*w + row.strict[3]*(s+j)},
			assembly:      candidateWorkVector{10: row.assembly + row.gas*g},
			originalPool:  row.pools[0], strictPool: row.pools[1], canonicalOriginal: row.pools[2], canonicalStrict: row.pools[3],
			originalSlots: row.pools[4], strictSlots: row.pools[5], rootRows: row.pools[6], strictRows: row.pools[7],
		}
		for i := range want.normal {
			want.normal[i] = want.normalization[i] + want.strict[i] + want.assembly[i]
		}
		want.envelope = want.normal
		want.envelope[10] += row.room
		if !ok || got != want {
			t.Fatal("independent portable whole gas row", row, got, want, ok)
		}
	}
}

func TestProviderHistoryGasJourneyRejectsInvalidData(t *testing.T) {
	valid := providerHistoryGasJourneyLimits{2, 4, 2, 65536, 4096, 512, 2048, 4, 2}
	for _, mode := range []string{"blocks-zero", "blocks-large", "windows-zero", "windows-large", "pending-zero", "pending-large", "evidence-zero", "evidence-large", "body-small", "body-large", "native-bound-zero", "native-bound-large", "value-zero", "value-evidence", "value-body", "value-native", "transactions-zero", "transactions-large", "native-blocks-large", "profile"} {
		p, profile := valid, SepoliaAmsterdamGasProfile
		switch mode {
		case "blocks-zero":
			p.Blocks = 0
		case "blocks-large":
			p.Blocks = (1 << 20) + 1
		case "windows-zero":
			p.Windows = 0
		case "windows-large":
			p.Windows = p.Blocks + 1
		case "pending-zero":
			p.PendingBlocks = 0
		case "pending-large":
			p.PendingBlocks = 11
		case "evidence-zero":
			p.EvidenceBytes = 0
		case "evidence-large":
			p.EvidenceBytes = (64 << 20) + 1
		case "body-small":
			p.BodyBytes = 3
		case "body-large":
			p.BodyBytes = (8 << 20) + 1
		case "native-bound-zero":
			p.TransactionBytes = 0
		case "native-bound-large":
			p.TransactionBytes = (64 << 20) + 1
		case "value-zero":
			p.ValueBytes = 0
		case "value-evidence":
			p.EvidenceBytes = p.ValueBytes - 1
		case "value-body":
			p.BodyBytes = p.ValueBytes - 1
		case "value-native":
			p.TransactionBytes = p.ValueBytes - 1
		case "transactions-zero":
			p.Transactions = 0
		case "transactions-large":
			p.Transactions = 16385
		case "native-blocks-large":
			p.NativeBlocks = p.Blocks + 1
		case "profile":
			profile = "unsupported-profile"
		}
		if got, ok := providerHistoryGasJourneyUpper(p, profile); ok || got != (providerHistoryGasJourneyCost{}) {
			t.Fatal("invalid input leaked partial gas cost", mode, got, ok)
		}
	}
	for i := range 9 {
		p := valid
		fields := []*uint64{&p.Windows, &p.Blocks, &p.PendingBlocks, &p.EvidenceBytes, &p.BodyBytes, &p.ValueBytes, &p.TransactionBytes, &p.Transactions, &p.NativeBlocks}
		*fields[i] = ^uint64(0)
		if got, ok := providerHistoryGasJourneyUpper(p, SepoliaAmsterdamRootRegularGasProfile); ok || got != (providerHistoryGasJourneyCost{}) {
			t.Fatal("MAX input leaked partial gas cost", i, got, ok)
		}
	}
}

func TestProviderHistoryGasJourneyRejectsLateOverflowWithoutPartialCost(t *testing.T) {
	raw := new(big.Int).Mul(big.NewInt(9), big.NewInt(1<<20))
	raw.Mul(raw, big.NewInt(8<<20)).Mul(raw, big.NewInt(8<<20))
	if raw.String() != "664082786653543858176" || raw.IsUint64() {
		t.Fatal("independent complete raw-root lower term", raw)
	}
	p := providerHistoryGasJourneyLimits{1 << 20, 1 << 20, 10, 64 << 20, 8 << 20, 8 << 20, 64 << 20, 16384, 1 << 20}
	for _, profile := range []string{SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamGasProfile} {
		if got, ok := providerHistoryGasJourneyUpper(p, profile); ok || got != (providerHistoryGasJourneyCost{}) {
			t.Fatal("late arithmetic refusal leaked earlier counts", profile, got, ok)
		}
	}
}

func TestProviderHistoryGasJourneyPreservesMatchedChildrenWithoutMultiplyingRoom(t *testing.T) {
	p := providerHistoryGasJourneyLimits{1, 1, 1, 65536, 1024, 512, 2048, 2, 1}
	whole, ok := providerHistoryGasJourneyUpper(p, SepoliaAmsterdamRootRegularGasProfile)
	root, a := providerHistoryRootGasNormalizationPoolUpper(1024, 1024, 512, 2048, 2)
	strict, b := providerHistoryGasCheckPoolUpper(1024, 1024, 512, 2048, 2)
	if !ok || !a || !b || whole.normalization != root.normal {
		t.Fatal("complete normalizer row changed", whole, root)
	}
	for i := range whole.strict {
		if whole.strict[i] != 2*strict.normal[i] {
			t.Fatal("one of the two original strict passes lost", i)
		}
	}
	const room = uint64(5308417)
	if whole.envelope[10]-whole.normal[10] != room || root.envelope[10]-root.normal[10] != room || strict.envelope[10]-strict.normal[10] != room {
		t.Fatal("current incoming room was dropped or multiplied", whole)
	}
	// This is one component theorem, not permission to refund the original
	// initializer or SUM FULL admitted stage uppers in the collector.
}

// Finite private invocation-sequence DATA only. It is not an admitted cursor,
// canonical chain, acquired RPC response, actual Source or operating policy.
// Select limits before creating any target observer; keep its original wide H.
func historyGasJourneyFixtureQuote(t *testing.T, trace []byte, block replayDecodedBlock, profile string, maximum int) providerHistoryGasJourneyCost {
	t.Helper()
	var pool, largest uint64
	for _, tx := range block.txs {
		binary, err := tx.MarshalBinary()
		if err != nil || len(binary) == 0 {
			t.Fatal("canonical private transaction precursor", err)
		}
		pool += uint64(len(binary))
		largest = max(largest, uint64(len(binary)))
	}
	p := providerHistoryGasJourneyLimits{1, 1, 1, 1 << 20, max(1024, uint64(len(trace)), pool), max(1, largest), 64 << 20, uint64(maximum), 1}
	cost, ok := providerHistoryGasJourneyUpper(p, profile)
	if !ok {
		t.Fatal("finite private gas sequence quote", p)
	}
	return cost
}

func TestProviderHistoryGasJourneyExecutesRootPreparationAndBothStrictPasses(t *testing.T) {
	for _, profile := range []string{SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile} {
		field := "regularGasUsed"
		if profile == SepoliaAmsterdamRootExecutionGasProfile {
			field = "executionGasUsed"
		}
		fields, tx := historyRootGasNormalizationFields(t, field)
		original, _ := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, false)
		raw, block := historyGasCheckPoolFixture(t, profile, 1)
		if tx.Hash() != block.txs[0].Hash() {
			t.Fatal("same canonical transaction precursor")
		}
		cost := historyGasJourneyFixtureQuote(t, original, block, profile, 1)
		limits := replayTestLimits()
		limits.MaxEvidenceBytes = 1 << 20
		limits.MaxTransactions = 1
		m := providerHistoryContentUnitMeter(cost.envelope)
		temporary := replayBudget{limits: limits, observer: m}
		copy := providerReplayBytes(string(original), m)
		selected, normalized, err := nodeRootGas(context.Background(), copy, block, &temporary)
		if err != nil || !providerReplayTextEqual(selected, profile, m) || !m.event(providerCopyEvent, uint64(reflect.TypeFor[ReplayGasEvidence]().Size())) {
			t.Fatal("original preparation precursor", err, selected, m.actual)
		}
		raw.Gas = &ReplayGasEvidence{selected, raw.Gas.BlockHash, normalized}
		if replayCheckGas(context.Background(), raw, block, &temporary) != nil {
			t.Fatal("first original strict check")
		}
		semantic := replayBudget{limits: limits, observer: m}
		if replayCheckGas(context.Background(), raw, block, &semantic) != nil || m.failed || m.clocks != 0 || !bytes.Equal(original, copy) ||
			temporary.bytes != len(original)+len(selected)+len(raw.Gas.BlockHash)+len(normalized) || semantic.bytes != len(selected)+len(raw.Gas.BlockHash)+len(normalized) {
			t.Fatal("two strict passes or original budgets changed", m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasJourneyKeepsOriginalWideIncomingPreflight(t *testing.T) {
	fields, tx := historyRootGasNormalizationFields(t, "regularGasUsed")
	original, block := historyRootGasNormalizationTrace(t, []map[string]json.RawMessage{fields}, tx, false)
	cost := historyGasJourneyFixtureQuote(t, original, block, SepoliaAmsterdamRootRegularGasProfile, 1)
	m := providerHistoryContentUnitMeter(cost.normal)
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 1 << 20
	limits.MaxTransactions = 1
	b := replayBudget{limits: limits, observer: m}
	if _, _, err := nodeRootGas(context.Background(), original, block, &b); err == nil || !m.failed || m.actual[2] == 0 {
		t.Fatal("actual-only quote bypassed original wide preflight", err, m.actual)
	}
	if m.actual[10] > cost.normal[10] {
		t.Fatal("refusal erased or exceeded prior actuals", m.actual)
	}
}

func TestProviderHistoryGasJourneyExecutesDirectPreparationAndBothStrictPasses(t *testing.T) {
	for _, mode := range []string{"reth", "nethermind", "mismatch"} {
		actualProfile, selectedProfile := SepoliaAmsterdamGasProfile, SepoliaAmsterdamGasProfile
		if mode == "nethermind" {
			actualProfile, selectedProfile = SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamNethermindGasProfile
		}
		if mode == "mismatch" {
			actualProfile = SepoliaAmsterdamNethermindGasProfile
		}
		raw, block := historyGasCheckPoolFixture(t, actualProfile, 1)
		original := bytes.Clone(raw.Gas.Trace)
		cost := historyGasJourneyFixtureQuote(t, original, block, selectedProfile, 1)
		limits := replayTestLimits()
		limits.MaxEvidenceBytes = 1 << 20
		limits.MaxTransactions = 1
		m := providerHistoryContentUnitMeter(cost.envelope)
		copied := providerReplayBytes(string(original), m)
		profile, err := replayGasProfileObserved(copied, limits.MaxTransactions, m)
		if err != nil || profile != actualProfile {
			t.Fatal("original native shape classification", err, profile)
		}
		equal := providerReplayTextEqual(profile, selectedProfile, m)
		if mode == "mismatch" {
			if equal || m.failed || m.actual[2] == 0 {
				t.Fatal("ordinary selected-profile refusal retried or poisoned", m.actual)
			}
			historyEventQueueWithin(t, m.actual, cost.normal)
			continue
		}
		if !equal || !m.event(providerCopyEvent, uint64(reflect.TypeFor[ReplayGasEvidence]().Size())) {
			t.Fatal("selected gas descriptor")
		}
		raw.Gas.Trace = copied
		temporary, semantic := replayBudget{limits: limits, observer: m}, replayBudget{limits: limits, observer: m}
		if replayCheckGas(context.Background(), raw, block, &temporary) != nil || replayCheckGas(context.Background(), raw, block, &semantic) != nil || m.failed || m.clocks != 0 || !bytes.Equal(original, copied) {
			t.Fatal("both direct strict passes", mode, m.actual)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasJourneyRetainsPaddedEmptyArraySlots(t *testing.T) {
	const trace = "[                                                                ]"
	n := uint64(len(trace))
	w := uint64(strconv.IntSize / 8)
	const slots = uint64(16)
	g := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	for _, profile := range []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile} {
		raw, block := historyGasCheckPoolFixture(t, profile, 0)
		raw.Gas.Trace = []byte(trace)
		header, err := rlp.EncodeToBytes(block.header)
		if err != nil {
			t.Fatal("original header precursor", err)
		}
		h := uint64(len(header))
		cost := historyGasJourneyFixtureQuote(t, []byte(trace), block, profile, int(slots))
		limits := replayTestLimits()
		limits.MaxEvidenceBytes = 1 << 20
		limits.MaxTransactions = int(slots)
		m := providerHistoryContentUnitMeter(cost.envelope)
		copied := providerReplayBytes(trace, m)
		rows, ok := nodeRawArrayObserved(copied, limits.MaxTransactions, m)
		if !ok || len(rows) != 0 || !providerReplayTextEqual(profile, profile, m) || !m.event(providerCopyEvent, g) {
			t.Fatal("empty branch precursor", m.actual)
		}
		raw.Gas.Trace = copied
		first, second := replayBudget{limits: limits, observer: m}, replayBudget{limits: limits, observer: m}
		if replayCheckGas(context.Background(), raw, block, &first) != nil || replayCheckGas(context.Background(), raw, block, &second) != nil || m.failed || m.clocks != 0 {
			t.Fatal("original empty gas checks", m.actual)
		}
		array := n*n + 2 + 3*w*slots
		check := h*h + h + 4497 + 2*n*n + 2*n + 3*w*slots
		assembly := n + 2*uint64(len(profile)) + g + 3
		want := candidateWorkVector{2: 5 * n, 10: array + 2*check + assembly}
		if m.actual != want {
			t.Fatal("empty transactions lost allocated raw headers", m.actual, want)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasJourneyRetainsUnknownKeyRefusalPrefixes(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, inner := range []bool{false, true} {
		key := strings.Repeat("z", 4096)
		object := `{"` + key + `":true}`
		row := object
		if inner {
			row = `{"txHash":"x","result":` + object + `}`
		}
		trace := []byte("[" + row + "]")
		_, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamGasProfile, 0)
		cost := historyGasJourneyFixtureQuote(t, trace, block, SepoliaAmsterdamGasProfile, 1)
		m := providerHistoryContentUnitMeter(cost.envelope)
		profile, err := replayGasProfileObserved(trace, 1, m)
		n, u, k := uint64(len(trace)), uint64(len(row)), uint64(len(key))
		want := candidateWorkVector{2: n + u, 10: n*n + u*u + u + 2*k + 6*w + 19}
		if inner {
			v := uint64(len(object))
			want = candidateWorkVector{2: n + u + v, 10: n*n + u*u + v*v + u + v + 5*k + 6*w + 109}
		}
		if err == nil || profile != "" || m.failed || m.clocks != 0 || m.actual != want {
			t.Fatal("first unknown original key prefix", inner, err, m.actual, want)
		}
		historyEventQueueWithin(t, m.actual, cost.normal)
	}
}

func TestProviderHistoryGasJourneyKeepsPreForkZeroNativePath(t *testing.T) {
	p := providerHistoryGasJourneyLimits{1, 1, 1, 65536, 1024, 1, 64 << 20, 1, 0}
	cost, ok := providerHistoryGasJourneyUpper(p, SepoliaAmsterdamRootExecutionGasProfile)
	if !ok || cost != (providerHistoryGasJourneyCost{}) {
		t.Fatal("zero native count fabricated observed work or incoming room", cost, ok)
	}
	raw, block := historyGasCheckPoolFixture(t, SepoliaAmsterdamRootExecutionGasProfile, 0)
	block.header.Time = 0
	raw.Gas = nil
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 65536
	m := providerHistoryContentUnitMeter(cost.envelope)
	budget := replayBudget{limits: limits, observer: m}
	if replayCheckGas(context.Background(), raw, block, &budget) != nil || m.failed || m.actual != (candidateWorkVector{}) || budget.bytes != 0 {
		t.Fatal("original pre-fork path acquired observed gas work", m.actual)
	}
	raw.Gas = &ReplayGasEvidence{}
	if replayCheckGas(context.Background(), raw, block, &budget) == nil || m.failed || m.actual != (candidateWorkVector{}) {
		t.Fatal("pre-fork ordinary refusal changed observations", m.actual)
	}
}
