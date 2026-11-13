package registry011

import (
	"bytes"
	"context"
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryGasBranchUpperMatchesIndependentPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	integer := uint64(reflect.TypeFor[big.Int]().Size())
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	sidecar := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	for _, row := range []struct {
		input         [9]uint64 // G,R,B,W,E,T,C,V,N.
		profile       string
		shape         [7]uint64 // F,F*nativeWindows,FN,strict/original pools,strict/original slots.
		decoded       [2]uint64
		strict        [4]uint64 // Constant,word,big.Int-size,sidecar-copy coefficients.
		normalization [3]uint64 // Constant,word,sidecar-copy coefficients.
		assembly      [2]uint64 // Constant,gas-size coefficient.
		room          uint64
	}{
		{[9]uint64{1, 1, 1, 1, 65536, 1, 2048, 1024, 1}, SepoliaAmsterdamGasProfile, [7]uint64{15, 15, 15, 2048, 2048, 1, 1}, [2]uint64{373470, 92160}, [4]uint64{5304000360, 210, 30, 30}, [3]uint64{188815080, 90, 0}, [2]uint64{31815, 15}, 67554509807616015},
		{[9]uint64{2, 3, 5, 2, 65536, 2, 8192, 1024, 3}, SepoliaAmsterdamRootRegularGasProfile, [7]uint64{29, 58, 87, 24576, 24576, 6, 6}, [2]uint64{8589800, 5731444}, [4]uint64{132442680594, 1972, 232, 232}, [3]uint64{61371096619, 1566, 116}, [2]uint64{719403, 87}, 261210771256115258},
		{[9]uint64{1, 1, 3, 3, 65536, 2, 131072, 1024, 1}, SepoliaAmsterdamNethermindGasProfile, [7]uint64{15, 15, 15, 65536, 65536, 2, 2}, [2]uint64{11806140, 2949120}, [4]uint64{1675080049140, 420, 60, 60}, [3]uint64{386549032695, 180, 0}, [2]uint64{984315, 15}, 67554509807616015},
		{[9]uint64{1, 1, 3, 1, 65536, 2, 131072, 1024, 3}, SepoliaAmsterdamRootExecutionGasProfile, [7]uint64{15, 15, 45, 65536, 196608, 6, 6}, [2]uint64{11806140, 23600670}, [4]uint64{1675080318960, 780, 60, 60}, [3]uint64{3481227918645, 540, 30}, [2]uint64{2952765, 45}, 67554509807616015},
		{[9]uint64{1, 1, 2, 2, 65536, 1, 8192, 1024, 0}, SepoliaAmsterdamGasProfile, [7]uint64{15, 0, 0, 0, 0, 0, 0}, [2]uint64{}, [4]uint64{}, [3]uint64{}, [2]uint64{}, 0},
		{[9]uint64{1, 32, 3, 3, 67108864, 16384, 8388608, 8388608, 3}, SepoliaAmsterdamRootExecutionGasProfile, [7]uint64{15, 45, 45, 25165824, 25165824, 49152, 49152}, [2]uint64{4767252480, 3209379840}, [4]uint64{142498413598351770, 10321920, 1474560, 1474560}, [3]uint64{80749001638379745, 8847360, 737280}, [2]uint64{377491005, 45}, 202663529422848045},
	} {
		i := row.input
		got, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: i[0], Requests: i[1], Blocks: i[2], Windows: i[3]}, i[4], i[5], i[6], i[7], i[8], row.profile)
		strict := candidateWorkVector{2: row.decoded[0], 10: row.strict[0] + row.strict[1]*w + row.strict[2]*integer + row.strict[3]*sidecar}
		normal := candidateWorkVector{2: row.decoded[1], 10: row.normalization[0] + row.normalization[1]*w + row.normalization[2]*sidecar}
		assembly := candidateWorkVector{10: row.assembly[0] + row.assembly[1]*gas}
		var actual, total candidateWorkVector
		for c, n := range strict {
			actual[c] = n + normal[c] + assembly[c]
			total[c] = actual[c]
		}
		total[10] += row.room
		if !ok || got.strict != strict || got.normalization != normal || got.assembly != assembly || got.actual != actual || got.total != total || got.preflight != (candidateWorkVector{10: row.room}) ||
			([7]uint64{got.journeys, got.windows, got.native, got.strictPool, got.originalPool, got.strictSlots, got.originalSlots}) != row.shape {
			t.Fatal("independent complete gas branch rows", got, ok, strict, normal, assembly)
		}
	}
}

func TestProviderHistoryGasBranchSidecarHashCopiesMatchOriginalObserver(t *testing.T) {
	// Existing finite unsigned codec DATA only: no blob/proof construction,
	// recovered account, peer, transaction broadcast or native-cost claim.
	plain := providerHistoryNodeJSONUnitTransactions()[4]
	network := plain.WithBlobTxSidecar(&types.BlobTxSidecar{})
	before, err := network.MarshalBinary()
	canonical, canonicalErr := plain.MarshalBinary()
	sidecar := network.BlobTxSidecar()
	if err != nil || canonicalErr != nil || sidecar == nil || bytes.Equal(before, canonical) {
		t.Fatal("ordinary empty-sidecar and stripped-twin precursor", err, canonicalErr)
	}
	// Keep the original64MiB Encode preflight funded. Actual-minus-one
	// would refuse before the copies and is not a late-copy boundary.
	plainMeter := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
	networkMeter := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
	want, plainOK := nodeTransactionHashObserved(plain, plainMeter)
	got, networkOK := nodeTransactionHashObserved(network, networkMeter)
	after, afterErr := network.MarshalBinary()
	plainAfter, plainErr := plain.MarshalBinary()
	if !plainOK || !networkOK || plainMeter.failed || networkMeter.failed || got != want || want != plain.Hash() ||
		afterErr != nil || plainErr != nil || !bytes.Equal(before, after) || !bytes.Equal(canonical, plainAfter) ||
		network.BlobTxSidecar() != sidecar || plain.BlobTxSidecar() != nil {
		t.Fatal("hash or original serialization/ownership changed", plainOK, networkOK, afterErr, plainErr)
	}
	difference := candidateWorkVector{10: uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2}
	for c, n := range difference {
		if networkMeter.actual[c] != plainMeter.actual[c]+n {
			t.Fatal("exact two-copy coordinate difference", c, networkMeter.actual, plainMeter.actual, difference)
		}
	}
}

func TestProviderHistoryGasBranchKeepsExplicitProfileAlternativesSeparate(t *testing.T) {
	l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
	profiles := []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile}
	var rows [4]providerHistoryGasBranchCost
	for i, p := range profiles {
		var ok bool
		rows[i], ok = providerHistoryGasBranchUpper(l, 65536, 1, 8192, 1024, 1, p)
		if !ok {
			t.Fatal("existing selected profile", p)
		}
	}
	for _, row := range rows[1:] {
		if row.strict != rows[0].strict || row.preflight != rows[0].preflight {
			t.Fatal("profile changed two original strict checks or room")
		}
	}
	if rows[0].normalization != rows[1].normalization || rows[2].normalization != rows[3].normalization || rows[2].normalization[2] <= rows[0].normalization[2] ||
		rows[1].assembly[10]-rows[0].assembly[10] != 180 || rows[3].assembly[10]-rows[2].assembly[10] != 60 {
		t.Fatal("raw/root alternatives were combined or selected text lost", rows)
	}
}

func TestProviderHistoryGasBranchBothStrictPassesUseTheSameOriginalObserver(t *testing.T) {
	raw, block := providerHistoryNodeRecordedContent(t)
	trace := providerHistoryNodeFixtureResult(t, gasWire(t))
	raw.Gas = &ReplayGasEvidence{SepoliaAmsterdamGasProfile, block.header.Hash().Hex(), trace}
	if replayCheckGas(context.Background(), raw, block, &replayBudget{limits: replayTestLimits()}) != nil {
		t.Fatal("ordinary strict gas precursor")
	}
	m := providerHistorySemanticUnitMeter()
	var first candidateWorkVector
	upper, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 64<<20, 16384, 8<<20, 8<<20, 1, SepoliaAmsterdamGasProfile)
	if !ok {
		t.Fatal("ordinary prospective gas bounds")
	}
	for pass := uint64(1); pass <= 2; pass++ {
		if replayCheckGas(context.Background(), raw, block, &replayBudget{limits: replayTestLimits(), observer: m}) != nil || m.failed {
			t.Fatal("original strict gas check")
		}
		if pass == 1 {
			first = m.actual
			if first[2] < 97*uint64(len(block.txs)) || first[10] == 0 {
				t.Fatal("original recoveries missing", first)
			}
		}
		var want candidateWorkVector
		for c, n := range first {
			want[c] = pass * n
		}
		if m.actual != want {
			t.Fatal("second strict pass borrowed a warm sender cache or reset work", m.actual, want)
		}
	}
	historyEventQueueWithin(t, m.actual, upper.strict)
}

func TestProviderHistoryGasBranchRootNormalizationAndBothStrictPassesKeepOriginalBytes(t *testing.T) {
	raw, block := providerHistoryNodeRecordedContent(t)
	original := providerHistoryNodeFixtureResult(t, rootGasWire(t))
	before := bytes.Clone(original)
	profile, want, err := nodeRootGas(context.Background(), original, block, &replayBudget{limits: replayTestLimits()})
	if err != nil || profile != SepoliaAmsterdamRootRegularGasProfile {
		t.Fatal("ordinary root normalization precursor", err)
	}
	m := providerHistorySemanticUnitMeter()
	acquisition := replayBudget{limits: replayTestLimits(), observer: m}
	got, trace, err := nodeRootGas(context.Background(), original, block, &acquisition)
	if err != nil || got != profile || !bytes.Equal(trace, want) || !bytes.Equal(original, before) {
		t.Fatal("normalizer changed original/derived contract", err)
	}
	normal := m.actual
	upper, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 64<<20, 16384, 8<<20, 8<<20, 1, profile)
	if !ok {
		t.Fatal("ordinary prospective root bounds")
	}
	historyEventQueueWithin(t, normal, upper.normalization)
	raw.Gas = &ReplayGasEvidence{profile, block.header.Hash().Hex(), trace}
	// The first strict check shares the normalizer's temporary byte budget;
	// the second has its own persistent-window precursor but the SAME meter.
	if replayCheckGas(context.Background(), raw, block, &acquisition) != nil {
		t.Fatal("original normalized acquisition strict check")
	}
	first := historyEventQueueDelta(t, m, normal)
	if replayCheckGas(context.Background(), raw, block, &replayBudget{limits: replayTestLimits(), observer: m}) != nil {
		t.Fatal("original normalized semantic strict check")
	}
	strict := historyEventQueueDelta(t, m, normal)
	for c, n := range first {
		if strict[c] != 2*n {
			t.Fatal("normalized strict original prefix changed", strict, first)
		}
	}
	if m.failed || !bytes.Equal(original, before) || !bytes.Equal(trace, want) {
		t.Fatal("original bytes or observer changed")
	}
	historyEventQueueWithin(t, strict, upper.strict)
}

func TestProviderHistoryGasBranchNativeClassificationUsesOnlyItsOriginalFieldShape(t *testing.T) {
	_, block := providerHistoryNodeRecordedContent(t)
	rootOriginal := providerHistoryNodeFixtureResult(t, rootGasWire(t))
	_, regular, err := nodeRootGas(context.Background(), rootOriginal, block, &replayBudget{limits: replayTestLimits()})
	if err != nil {
		t.Fatal("ordinary regular-gas shape", err)
	}
	m := providerHistorySemanticUnitMeter()
	for _, row := range []struct {
		trace   []byte
		profile string
	}{{providerHistoryNodeFixtureResult(t, gasWire(t)), SepoliaAmsterdamGasProfile}, {regular, SepoliaAmsterdamNethermindGasProfile}} {
		before := m.actual
		got, err := replayGasProfileObserved(row.trace, 16384, m)
		if err != nil || got != row.profile || m.failed {
			t.Fatal("original closed native classification", got, err)
		}
		upper, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 64<<20, 16384, 8<<20, 8<<20, 1, row.profile)
		if !ok {
			t.Fatal("prospective native classification ceiling")
		}
		historyEventQueueWithin(t, historyEventQueueDelta(t, m, before), upper.normalization)
	}
}

func TestProviderHistoryGasBranchDerivedFloorUsesCanonicalValuesAndAccessTupleMinimum(t *testing.T) {
	empty, err := rlp.EncodeToBytes(types.AccessTuple{Address: common.Address{}, StorageKeys: []common.Hash{}})
	if err != nil || len(empty) != 23 {
		t.Fatal("independent fixed address/empty-list RLP minimum", len(empty), err)
	}
	w := uint64(strconv.IntSize / 8)
	integer := uint64(reflect.TypeFor[big.Int]().Size())
	for _, tx := range providerHistoryNodeJSONUnitTransactions() {
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal("ordinary canonical transaction", err)
		}
		e := uint64(len(raw))
		m := providerHistorySemanticUnitMeter()
		if _, err := replayGasFloorDerivedObserved(tx, common.Address{1}, m); err != nil {
			t.Fatal("finite derived arithmetic DATA", err)
		}
		historyEventQueueWithin(t, m.actual, candidateWorkVector{10: e + integer + w + 62 + e/23})
	}
}

func TestProviderHistoryGasBranchEarlyCodecShortagesPreserveEnteredPrefixes(t *testing.T) {
	raw, block := providerHistoryNodeRecordedContent(t)
	trace := providerHistoryNodeFixtureResult(t, gasWire(t))
	root := providerHistoryNodeFixtureResult(t, rootGasWire(t))
	raw.Gas = &ReplayGasEvidence{SepoliaAmsterdamGasProfile, block.header.Hash().Hex(), trace}
	header, err := rlp.EncodeToBytes(block.header)
	if err != nil || len(trace) == 0 || len(root) == 0 {
		t.Fatal("ordinary source lengths", err)
	}
	h, n := uint64(len(header)), uint64(len(trace))
	prefix := candidateWorkVector{10: h*h + h + 2*n + 4494}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: n - 1, 10: 1 << 60})
	budget := replayBudget{limits: replayTestLimits(), observer: m}
	for attempt := 0; attempt < 2; attempt++ {
		if replayCheckGas(context.Background(), raw, block, &budget) == nil || !m.failed || m.actual != prefix {
			t.Fatal("strict scan/commitment prefix changed on codec refusal", m.actual, prefix)
		}
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{2: uint64(len(root) - 1), 10: 1 << 60})
	budget = replayBudget{limits: replayTestLimits(), observer: m}
	for attempt := 0; attempt < 2; attempt++ {
		p, out, err := nodeRootGas(context.Background(), root, block, &budget)
		if err == nil || p != "" || out != nil || !m.failed || m.actual != (candidateWorkVector{}) {
			t.Fatal("root first codec refusal returned partial DATA or resumed", p, err, m.actual)
		}
	}
}

func TestProviderHistoryGasBranchInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "versions", "stops", "evidence-small", "evidence-wide", "tx-zero", "tx-wide", "body-small", "body-wide", "value-zero", "value-wide", "native-wide", "profile"} {
		l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
		e, txs, c, v, n := uint64(65536), uint64(1), uint64(2048), uint64(1024), uint64(1)
		profile := SepoliaAmsterdamGasProfile
		switch mode {
		case "groups":
			l.Groups = 0
		case "requests":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "versions":
			l.Versions = 1<<20 + 1
		case "stops":
			l.PresentStops = 2
		case "evidence-small":
			e = 65535
		case "evidence-wide":
			e = 64<<20 + 1
		case "tx-zero":
			txs = 0
		case "tx-wide":
			txs = 16385
		case "body-small":
			c = 3
		case "body-wide":
			c = 8<<20 + 1
		case "value-zero":
			v = 0
		case "value-wide":
			v = c + 1
		case "native-wide":
			n = 2
		case "profile":
			profile = "unselected"
		}
		if got, ok := providerHistoryGasBranchUpper(l, e, txs, c, v, n, profile); ok || got != (providerHistoryGasBranchCost{}) {
			t.Fatal("invalid gas tuple leaked a partial cost", mode, got, ok)
		}
	}
}

func TestProviderHistoryGasBranchLateActualAndPreflightOverflowsReturnWholeZero(t *testing.T) {
	minimum, ok := new(big.Int).SetString("126663739519795200000", 10)
	if !ok || minimum.BitLen() <= 64 {
		t.Fatal("independent two-pass leading quadratic overflow")
	}
	for _, profile := range []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamRootRegularGasProfile} {
		got, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 10000, Windows: 10000}, 64<<20, 16384, 8<<20, 8<<20, 10000, profile)
		if ok || got != (providerHistoryGasBranchCost{}) {
			t.Fatal("wide two-pass gas actual leaked partial fields", got, ok)
		}
	}
	// Actual scalar work fits, while the unchanged signing room overflows.
	// This numeric tuple does not certify that four bytes hold a native block.
	got, ok := providerHistoryGasBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 10000, Requests: 10000, Blocks: 1, Windows: 1}, 65536, 1, 4, 1, 1, SepoliaAmsterdamGasProfile)
	if ok || got != (providerHistoryGasBranchCost{}) {
		t.Fatal("late gas room failure leaked an actual vector", got, ok)
	}
}
