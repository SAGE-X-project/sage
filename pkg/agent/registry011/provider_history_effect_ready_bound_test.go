package registry011

import (
	"context"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryEffectReadyMatchesIndependentPortableRows(t *testing.T) {
	effect := uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	block := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	sidecar := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	for _, row := range []struct {
		limits      providerHistoryEffectReadyLimits
		diagnostics [5]uint64
		parts       [8][5]uint64 // constant,effect,string,block,sidecar.
	}{
		{providerHistoryEffectReadyLimits{1, 1, 1, 65536, 2048, 3072, 1024, 2, 2, 1}, [5]uint64{1, 1024, 1, 1, 2}, [8][5]uint64{
			{84, 0, 0, 0, 0},       // selection
			{1, 1, 0, 0, 0},        // arrays
			{1053959, 0, 0, 0, 1},  // hashes
			{2, 0, 0, 2, 0},        // pending
			{3, 0, 3, 0, 0},        // traceHeaders
			{18, 0, 0, 0, 0},       // readyScans
			{1054067, 1, 3, 2, 1},  // normal
			{10491252, 1, 3, 2, 1}, // envelope
		}},
		{providerHistoryEffectReadyLimits{2, 5, 2, 65536, 8192, 9216, 1024, 3, 5, 10}, [5]uint64{10, 10240, 20, 1, 30}, [8][5]uint64{
			{824, 0, 0, 0, 0},          // selection
			{5, 20, 0, 0, 0},           // arrays
			{10539590, 0, 0, 0, 10},    // hashes
			{10, 0, 0, 10, 0},          // pending
			{25, 0, 30, 0, 0},          // traceHeaders
			{270, 0, 0, 0, 0},          // readyScans
			{10540724, 20, 30, 10, 10}, // normal
			{95475381, 20, 30, 10, 10}, // envelope
		}},
		{providerHistoryEffectReadyLimits{1, 100, 10, 65536, 131072, 132096, 4096, 2, 2, 100}, [5]uint64{22, 90112, 1000, 1, 1100}, [8][5]uint64{
			{924, 0, 0, 0, 0},                 // selection
			{100, 1000, 0, 0, 0},              // arrays
			{369284762, 0, 0, 0, 22},          // hashes
			{200, 0, 0, 200, 0},               // pending
			{300, 0, 300, 0, 0},               // traceHeaders
			{9900, 0, 0, 0, 0},                // readyScans
			{369296186, 1000, 300, 200, 22},   // normal
			{17818649403, 1000, 300, 200, 22}, // envelope
		}},
		{providerHistoryEffectReadyLimits{1, 1, 10, 1, 4, 1028, 1, 16, 1, 16}, [5]uint64{16, 1, 160, 1, 176}, [8][5]uint64{
			{72, 0, 0, 0, 0},          // selection
			{1, 160, 0, 0, 0},         // arrays
			{69746, 0, 0, 0, 16},      // hashes
			{2, 0, 0, 2, 0},           // pending
			{33, 0, 48, 0, 0},         // traceHeaders
			{1584, 0, 0, 0, 0},        // readyScans
			{71438, 160, 48, 2, 16},   // normal
			{1128223, 160, 48, 2, 16}, // envelope
		}},
		{providerHistoryEffectReadyLimits{1, 1, 1, 65536, 1024, 67108864, 1, 1, 1, 0}, [5]uint64{0, 0, 0, 0, 0}, [8][5]uint64{
			{42, 0, 0, 0, 0}, // selection
			{1, 0, 0, 0, 0},  // arrays
			{0, 0, 0, 0, 0},  // hashes
			{2, 0, 0, 2, 0},  // pending
			{1, 0, 0, 0, 0},  // traceHeaders
			{0, 0, 0, 0, 0},  // readyScans
			{46, 0, 0, 2, 0}, // normal
			{46, 0, 0, 2, 0}, // envelope
		}},
		{providerHistoryEffectReadyLimits{1, 4, 2, 1048576, 8192, 67108864, 1024, 64, 64, 32}, [5]uint64{32, 32768, 64, 1, 96}, [8][5]uint64{
			{8064, 0, 0, 0, 0},                // selection
			{4, 64, 0, 0, 0},                  // arrays
			{33726688, 0, 0, 0, 32},           // hashes
			{8, 0, 0, 8, 0},                   // pending
			{68, 0, 96, 0, 0},                 // traceHeaders
			{864, 0, 0, 0, 0},                 // readyScans
			{33735696, 64, 96, 8, 32},         // normal
			{4503599661106193, 64, 96, 8, 32}, // envelope
		}},
		{providerHistoryEffectReadyLimits{1, 4, 2, 1048576, 8192, 67108864, 1024, 64, 64, 33}, [5]uint64{33, 32768, 66, 2, 132}, [8][5]uint64{
			{8064, 0, 0, 0, 0},                // selection
			{4, 66, 0, 0, 0},                  // arrays
			{33731047, 0, 0, 0, 33},           // hashes
			{8, 0, 0, 8, 0},                   // pending
			{70, 0, 99, 0, 0},                 // traceHeaders
			{1188, 0, 0, 0, 0},                // readyScans
			{33740381, 66, 99, 8, 33},         // normal
			{4503599661110878, 66, 99, 8, 33}, // envelope
		}},
		{providerHistoryEffectReadyLimits{3, 3, 10, 67108864, 8388608, 67108864, 8388608, 16384, 65536, 1048576}, [5]uint64{49152, 25165824, 10485760, 5120, 5379194880}, [8][5]uint64{
			{7962624, 0, 0, 0, 0},                           // selection
			{3, 10485760, 0, 0, 0},                          // arrays
			{211106471952384, 0, 0, 0, 49152},               // hashes
			{6, 0, 0, 6, 0},                                 // pending
			{2097155, 0, 3145728, 0, 0},                     // traceHeaders
			{48412753920, 0, 0, 0, 0},                       // readyScans
			{211154894766092, 10485760, 3145728, 6, 49152},  // normal
			{4714754522136589, 10485760, 3145728, 6, 49152}, // envelope
		}},
	} {
		got, ok := providerHistoryEffectReadyUpper(row.limits)
		if !ok || ([5]uint64{got.selectedRows, got.canonicalPool, got.effectArraySlots, got.waitGroups, got.scanEntries}) != row.diagnostics {
			t.Fatal("original effect source dimensions", row, got, ok)
		}
		parts := [8]candidateWorkVector{got.selection, got.arrays, got.hashes, got.pending, got.traceHeaders, got.readyScans, got.normal, got.envelope}
		for i, v := range row.parts {
			want := candidateWorkVector{10: v[0] + v[1]*effect + v[2]*text + v[3]*block + v[4]*sidecar}
			if parts[i] != want {
				t.Fatal("independent complete effect component", row.limits, i, parts[i], want)
			}
		}
	}
}

func TestProviderHistoryEffectReadyRefusesInvalidAndLateOverflowWithoutPartialCost(t *testing.T) {
	valid := providerHistoryEffectReadyLimits{2, 4, 2, 65536, 4096, 64 << 20, 1024, 4, 8, 10}
	for _, mode := range []string{"blocks-zero", "blocks-large", "windows-zero", "windows-large", "pending-zero", "pending-large", "evidence-zero", "evidence-large", "body-small", "body-large", "native-zero", "native-large", "value-zero", "value-evidence", "value-body", "value-native", "transactions-zero", "transactions-large", "logs-zero", "logs-large", "effects-large", "late-overflow"} {
		p := valid
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
		case "native-zero":
			p.TransactionBytes = 0
		case "native-large":
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
		case "logs-zero":
			p.Logs = 0
		case "logs-large":
			p.Logs = 65537
		case "effects-large":
			p.Effects = (1 << 20) + 1
		case "late-overflow":
			p = providerHistoryEffectReadyLimits{1 << 20, 1 << 20, 10, 64 << 20, 8 << 20, 64 << 20, 8 << 20, 16384, 65536, 1 << 20}
		}
		if got, ok := providerHistoryEffectReadyUpper(p); ok || got != (providerHistoryEffectReadyCost{}) {
			t.Fatal("invalid or late-overflow effect data leaked a partial cost", mode, got, ok)
		}
	}
	for i := range 10 {
		p := valid
		fields := []*uint64{&p.Windows, &p.Blocks, &p.PendingBlocks, &p.EvidenceBytes, &p.BodyBytes, &p.TransactionBytes, &p.ValueBytes, &p.Transactions, &p.Logs, &p.Effects}
		*fields[i] = ^uint64(0)
		if got, ok := providerHistoryEffectReadyUpper(p); ok || got != (providerHistoryEffectReadyCost{}) {
			t.Fatal("MAX effect field leaked cost data", i, got, ok)
		}
	}
}

func TestProviderHistoryEffectReadyZeroEffectsKeepsOriginalEmptyControlWork(t *testing.T) {
	p := providerHistoryEffectReadyLimits{1, 1, 1, 65536, 1024, 64 << 20, 1, 1, 1, 0}
	got, ok := providerHistoryEffectReadyUpper(p)
	block := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	if !ok || got.selectedRows != 0 || got.canonicalPool != 0 || got.effectArraySlots != 0 || got.waitGroups != 0 || got.scanEntries != 0 || got.hashes != (candidateWorkVector{}) || got.readyScans != (candidateWorkVector{}) || got.normal != (candidateWorkVector{10: 46 + 2*block}) || got.normal != got.envelope {
		t.Fatal("zero effects erased empty control events or invented native work", got, ok)
	}
}

// Scalar source bounds come from the original fixture configuration, never
// from measurements of the target call. This private synthetic deployment-zero
// cursor cannot supply source, chain, gas or production admission authority.
func historyEffectReadyQuote(t *testing.T, c *providerHistoryCollectionCursor) providerHistoryEffectReadyCost {
	t.Helper()
	in := c.selection.input
	l := in.journal.Replay
	windows := (in.blocks + in.windowBlocks - 1) / in.windowBlocks
	bound, original := providerHistoryNativeInputBoundObserved(c.semantic.observer)
	if !original || bound.collection || bound.transaction != 64<<20 {
		t.Fatal("original wide inert native selector changed")
	}
	upper, ok := providerHistoryEffectReadyUpper(providerHistoryEffectReadyLimits{
		windows, in.blocks, in.contentBlocks, uint64(l.MaxEvidenceBytes), uint64(in.capture.BodyBytes), bound.transaction,
		min(uint64(l.MaxEvidenceBytes), uint64(in.capture.BodyBytes)), uint64(l.MaxTransactions), uint64(l.MaxLogs), in.effects,
	})
	if !ok {
		t.Fatal("ordinary original effect configuration", upper)
	}
	return upper
}

func TestProviderHistoryEffectReadySelectionKeepsExactCanonicalHashAndGrowthEvents(t *testing.T) {
	c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods) // Original window opening.
	upper := historyEffectReadyQuote(t, c)
	m := c.semantic.observer
	var measured candidateWorkVector
	for _, raw := range in.Blocks {
		// Complete canonical validation is a real observed precursor on the
		// SAME observer. Its own work remains intact and is not part of the
		// isolated retainEffects delta. No quota or counter is reset.
		budget := replayBudget{limits: c.selection.input.journal.Replay, observer: m}
		decoded, err := replayDecodeContent(context.Background(), raw, &budget)
		if err != nil {
			t.Fatal("canonical effect prerequisite", err)
		}
		old := len(c.effects)
		var selected, logical uint64
		for i, receipt := range decoded.receipts {
			logical += 2 * (20*uint64(len(receipt.Logs)) + 1)
			for _, entry := range receipt.Logs {
				if entry.Address != common.HexToAddress(SepoliaRegistryAddress) {
					continue
				}
				if decoded.txs[i].BlobTxSidecar() != nil {
					t.Fatal("ordinary fixture unexpectedly contains a sidecar")
				}
				binary, err := decoded.txs[i].MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				n := uint64(len(binary))
				logical += n*n + n + 4359
				selected++
				break
			}
		}
		logical += uint64(old+int(selected))*uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size()) + 1
		wantHeader := journalTestHeader(t, raw)
		before := m.actual
		if !c.retainEffectsObserved(decoded, wantHeader, m) {
			t.Fatal("original selected-effect retention")
		}
		delta := historyEventQueueDelta(t, m, before)
		if delta != (candidateWorkVector{10: logical}) || len(c.effects) != old+int(selected) || m.failed || m.clocks != 0 {
			t.Fatal("canonical selection lost a comparison/hash/copy prefix", delta, logical, c.effects)
		}
		for i := range measured {
			measured[i] += delta[i]
		}
	}
	if len(c.effects) != 40 || c.traces != 0 {
		t.Fatal("two ordinary block selections did not retain40 distinct receipt indexes")
	}
	parts := upper.selection
	for i := range parts {
		parts[i] += upper.arrays[i] + upper.hashes[i]
	}
	historyEventQueueWithin(t, measured, parts)
}

func TestProviderHistoryEffectReadyZeroCapacityRefusesBeforeArrayOrHashWork(t *testing.T) {
	in := historyCollectionStreamingInput(t)
	// Select X0 BEFORE constructing the fixture's immutable original input.
	// Canonical content still has real Registry receipt logs; its later
	// missing trace selection must refuse. Nothing is patched after setup.
	for i := range in.Blocks {
		in.Blocks[i].Executions = nil
	}
	c, in, methods := historyCollectionContentInputFixture(t, 2, in, SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods)
	upper := historyEffectReadyQuote(t, c)
	m := c.semantic.observer
	budget := replayBudget{limits: c.selection.input.journal.Replay, observer: m}
	decoded, err := replayDecodeContent(context.Background(), in.Blocks[0], &budget)
	if err != nil {
		t.Fatal("canonical capacity prerequisite", err)
	}
	var logical uint64
	for _, receipt := range decoded.receipts {
		logical += 20*uint64(len(receipt.Logs)) + 1
	}
	wantHeader := journalTestHeader(t, in.Blocks[0])
	before := m.actual
	if c.retainEffectsObserved(decoded, wantHeader, m) {
		t.Fatal("zero original capacity admitted selected Registry effects")
	}
	delta := historyEventQueueDelta(t, m, before)
	if delta != (candidateWorkVector{10: logical}) || m.failed || m.clocks != 0 || c.traces != 0 || len(c.effects) != 0 || upper.selectedRows != 0 || upper.hashes != (candidateWorkVector{}) || upper.normal != upper.envelope {
		t.Fatal("capacity refusal entered array/hash or erased its comparison prefix", delta, logical, c.effects, upper)
	}
	historyEventQueueWithin(t, delta, upper.selection)
}

func TestProviderHistoryEffectReadyBlockedEntryScansEverySelectedEffect(t *testing.T) {
	c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods)
	c = historyCollectionContentStep(t, c, in, methods)
	if c.phase != "effects" || len(c.effects) != 40 || len(c.pending) != 2 || c.pendingApplied != 0 || c.effectNext != 0 {
		t.Fatal("ordinary original pending precursor")
	}
	upper := historyEffectReadyQuote(t, c)
	m := c.semantic.observer
	before := m.actual
	// One isolated private method entry tests its local event placement;
	// it is not another authenticated S or a witness for journey counts.
	if !c.applyReadyBlocksObserved(context.Background(), m) {
		t.Fatal("ordinary unavailable-traces wait")
	}
	delta := historyEventQueueDelta(t, m, before)
	if delta != (candidateWorkVector{10: 360}) || m.failed || m.clocks != 0 || c.pendingApplied != 0 || c.semantic.blocks != 0 || c.effectNext != 0 || c.traces != 0 || c.phase != "effects" {
		t.Fatal("blocked entry skipped its complete effect scan or applied an incomplete block", delta)
	}
	historyEventQueueWithin(t, delta, upper.readyScans)
}

func TestProviderHistoryEffectReadyDeclaresWholePendingArrayBeforeFirstDecodeFailure(t *testing.T) {
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 1
	c, in, methods := historyCollectionContentInputLimitsFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile, limits)
	c = historyCollectionContentStep(t, c, in, methods)
	if len(c.pending) != 0 || c.phase != "content" {
		t.Fatal("ordinary pre-content pending state")
	}
	_ = historyEffectReadyQuote(t, c)
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, in, table)
	m := c.semantic.observer
	before := m.actual
	if c.contentBoundaryObserved(context.Background(), results, m) {
		t.Fatal("one-byte evidence cap accepted the first original block")
	}
	delta := historyEventQueueDelta(t, m, before)
	if len(c.pending) != 2 || c.pendingApplied != 0 || c.semantic.blocks != 0 || len(c.effects) != 0 || m.failed || m.clocks != 0 || delta[2] == 0 || delta[3] != 0 || delta[10] < 2*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())+1 {
		t.Fatal("declared pending slots were replaced by successful decoder count", c.pending, delta)
	}
	// The failed original transition stops here. Its whole decoder/spine
	// work is outside this assigned pending-array row, with no retry or publish.
}
