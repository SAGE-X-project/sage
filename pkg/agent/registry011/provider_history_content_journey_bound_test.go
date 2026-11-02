package registry011

import (
	"context"
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func historyContentJourneySizes() [12]uint64 {
	return [12]uint64{
		uint64(strconv.IntSize / 8), uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size()),
		uint64(reflect.TypeFor[ReplayGasEvidence]().Size()),
		uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2,
		uint64(reflect.TypeFor[big.Int]().Size()), uint64(reflect.TypeFor[replayLog]().Size()),
		uint64(reflect.TypeFor[types.Log]().Size()), uint64(reflect.TypeFor[ReplayExecution]().Size()),
		uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size()), uint64(reflect.TypeFor[string]().Size()),
		uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()), uint64(reflect.TypeFor[FinalizedHeader]().Size()),
	}
}

func TestProviderHistoryContentJourneyMatchesIndependentPortableMechanicalRows(t *testing.T) {
	sizes := historyContentJourneySizes()
	for _, size := range sizes {
		if size > 4096 {
			t.Fatal("changed logical targets require a new portable source oracle")
		}
	}
	for _, row := range []struct {
		limits            providerHistoryContentJourneyLimits
		encoding, profile string
		vectors           [8]candidateWorkVector
		logical           [8][12]uint64
	}{
		{providerHistoryContentJourneyLimits{1, 1, 1, 65536, 2048, 1024, 3072, 2, 16, 2, 1, 0}, EthereumJSONBlocks, SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 601924, 3: 4, 4: 10, 6: 10, 10: 78382529635}, // projection
			{},                                       // gas
			{2: 285425, 5: 32, 6: 4, 10: 4878618573}, // trace
			{10: 1054067},                            // effects
			{10: 142},                                // control
			{2: 887349, 3: 4, 4: 10, 5: 32, 6: 14, 10: 83262202417}, // normal
			{10: 4848615425}, // incoming
			{2: 887349, 3: 4, 4: 10, 5: 32, 6: 14, 10: 88110817842}, // envelope
		}, [8][12]uint64{
			{123416, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{846212, 0, 0, 1, 2, 30, 2, 1, 0, 0, 0, 0},
			{0, 0, 0, 1, 0, 0, 0, 0, 1, 3, 2, 0},
			{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			{969629, 2, 0, 2, 2, 30, 2, 1, 1, 3, 2, 1},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{969629, 2, 0, 2, 2, 30, 2, 1, 1, 3, 2, 1},
		}},
		{providerHistoryContentJourneyLimits{2, 5, 2, 65536, 8192, 1024, 9216, 3, 32, 5, 10, 3}, EthereumJSONBlocks, SepoliaAmsterdamNethermindGasProfile, [8]candidateWorkVector{
			{2: 2892607, 3: 16, 4: 33, 6: 55, 10: 164135278675}, // projection
			{2: 370386, 10: 28840151352},                        // gas
			{2: 11317078, 5: 128, 6: 20, 10: 118464365565},      // trace
			{10: 10540724}, // effects
			{10: 710},      // control
			{2: 14580071, 3: 16, 4: 33, 5: 128, 6: 75, 10: 311450337026}, // normal
			{10: 6710886401}, // incoming
			{2: 14580071, 3: 16, 4: 33, 5: 128, 6: 75, 10: 318161223427}, // envelope
		}, [8][12]uint64{
			{1476591, 6, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{162, 0, 3, 18, 18, 0, 0, 0, 0, 0, 0, 0},
			{33834596, 0, 0, 6, 12, 166, 10, 10, 0, 0, 0, 0},
			{0, 0, 0, 10, 0, 0, 0, 0, 20, 30, 10, 0},
			{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5},
			{35311354, 6, 6, 34, 30, 166, 10, 10, 20, 30, 10, 5},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{35311354, 6, 6, 34, 30, 166, 10, 10, 20, 30, 10, 5},
		}},
		{providerHistoryContentJourneyLimits{1, 100, 10, 65536, 131072, 1024, 132096, 2, 16, 2, 40, 4}, EthereumJSONBlocks, SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 168486544, 3: 301, 4: 48, 6: 48, 10: 23279202772508}, // projection
			{2: 8915736, 10: 1271406542128},                          // gas
			{2: 153750418, 5: 32, 6: 4, 10: 20161480632965},          // trace
			{10: 23192462}, // effects
			{10: 14200},    // control
			{2: 331152698, 3: 301, 4: 48, 5: 32, 6: 52, 10: 44712113154263}, // normal
			{10: 38654705665}, // incoming
			{2: 331152698, 3: 301, 4: 48, 5: 32, 6: 52, 10: 44750767859928}, // envelope
		}, [8][12]uint64{
			{119163428, 101, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{192, 0, 4, 24, 16, 0, 0, 0, 0, 0, 0, 0},
			{460128644, 0, 0, 2, 4, 46, 2, 22, 0, 0, 0, 0},
			{0, 0, 0, 22, 0, 0, 0, 0, 400, 120, 200, 0},
			{100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100},
			{579292364, 101, 8, 48, 20, 46, 2, 22, 400, 120, 200, 100},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{579292364, 101, 8, 48, 20, 46, 2, 22, 400, 120, 200, 100},
		}},
		{providerHistoryContentJourneyLimits{1, 1, 1, 65536, 2048, 1024, 3072, 2, 16, 2, 0, 0}, "debug-raw-rlp-v1", SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 237560, 3: 2, 4: 6, 6: 6, 10: 67836878685}, // projection
			{},        // gas
			{},        // trace
			{10: 88},  // effects
			{10: 142}, // control
			{2: 237560, 3: 2, 4: 6, 6: 6, 10: 67836878915}, // normal
			{10: 4294967297}, // incoming
			{2: 237560, 3: 2, 4: 6, 6: 6, 10: 72131846212}, // envelope
		}, [8][12]uint64{
			{50447, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
			{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			{50448, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 1},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{50448, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 1},
		}},
		{providerHistoryContentJourneyLimits{2, 5, 2, 65536, 8192, 1024, 67108864, 3, 32, 5, 33, 3}, "debug-raw-rlp-v1", SepoliaAmsterdamNethermindGasProfile, [8]candidateWorkVector{
			{2: 884712, 3: 10, 4: 18, 6: 30, 10: 228722105603}, // projection
			{2: 370386, 10: 30047945016},                       // gas
			{2: 37318486, 5: 128, 6: 20, 10: 332424582256},     // trace
			{10: 12649309}, // effects
			{10: 710},      // control
			{2: 38573584, 3: 10, 4: 18, 5: 128, 6: 50, 10: 591207282894}, // normal
			{10: 4503633987174401}, // incoming
			{2: 38573584, 3: 10, 4: 18, 5: 128, 6: 50, 10: 4504225194457295}, // envelope
		}, [8][12]uint64{
			{250992, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{162, 0, 3, 18, 18, 0, 0, 0, 0, 0, 0, 0},
			{111650404, 0, 0, 6, 12, 166, 10, 12, 0, 0, 0, 0},
			{0, 0, 0, 12, 0, 0, 0, 0, 66, 99, 10, 0},
			{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5},
			{111901563, 0, 6, 36, 30, 166, 10, 12, 66, 99, 10, 5},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{111901563, 0, 6, 36, 30, 166, 10, 12, 66, 99, 10, 5},
		}},
		{providerHistoryContentJourneyLimits{1, 1, 10, 1, 4, 1, 1028, 16, 1, 1, 0, 0}, EthereumJSONBlocks, SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 275283, 3: 4, 4: 224, 6: 14, 10: 95729611270}, // projection
			{},        // gas
			{},        // trace
			{10: 76},  // effects
			{10: 142}, // control
			{2: 275283, 3: 4, 4: 224, 6: 14, 10: 95729611488}, // normal
			{10: 4295229445}, // incoming
			{2: 275283, 3: 4, 4: 224, 6: 14, 10: 100024840933}, // envelope
		}, [8][12]uint64{
			{1140, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
			{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			{1141, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 1},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{1141, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 1},
		}},
		{providerHistoryContentJourneyLimits{3, 3, 10, 67108864, 8388608, 1024, 8389632, 16384, 64, 65536, 10, 3}, EthereumJSONBlocks, SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 2689148160, 3: 10, 4: 327680, 6: 1310720, 10: 93390885098451094}, // projection
			{2: 520765440, 10: 4434309514914408},                                 // gas
			{2: 11634501194, 5: 384, 6: 393216, 10: 97108982038239411},           // trace
			{10: 18503236}, // effects
			{10: 426},      // control
			{2: 14844414794, 3: 10, 4: 327680, 5: 384, 6: 1703936, 10: 194934176670108575}, // normal
			{10: 283678294933505}, // incoming
			{2: 14844414794, 3: 10, 4: 327680, 5: 384, 6: 1703936, 10: 195217854965042080}, // envelope
		}, [8][12]uint64{
			{1014530752, 4, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{1179648, 0, 3, 147456, 98304, 0, 0, 0, 0, 0, 0, 0},
			{34653212224, 0, 0, 10, 20, 1376416, 196608, 10, 0, 0, 0, 0},
			{0, 0, 0, 10, 0, 0, 0, 0, 100, 30, 6, 0},
			{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3},
			{35668922627, 4, 6, 147476, 98324, 1376416, 196608, 10, 100, 30, 6, 3},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{35668922627, 4, 6, 147476, 98324, 1376416, 196608, 10, 100, 30, 6, 3},
		}},
		{providerHistoryContentJourneyLimits{1, 1, 1, 67108864, 8388608, 1024, 8389632, 16384, 128, 65536, 1, 1}, "debug-raw-rlp-v1", SepoliaAmsterdamRootExecutionGasProfile, [8]candidateWorkVector{
			{2: 352452600, 3: 2, 4: 49152, 6: 196608, 10: 7811196523202723}, // projection
			{2: 173588480, 10: 1478103171638136},                            // gas
			{2: 1177043105, 5: 256, 6: 131072, 10: 9710899192319449},        // trace
			{10: 3708191}, // effects
			{10: 142},     // control
			{2: 1703084185, 3: 2, 4: 49152, 5: 256, 6: 327680, 10: 19000198890868641}, // normal
			{10: 70390220652545}, // incoming
			{2: 1703084185, 3: 2, 4: 49152, 5: 256, 6: 327680, 10: 19070589111521186}, // envelope
		}, [8][12]uint64{
			{802816, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{393216, 0, 1, 49152, 32768, 0, 0, 0, 0, 0, 0, 0},
			{3467250048, 0, 0, 1, 2, 458768, 65536, 1, 0, 0, 0, 0},
			{0, 0, 0, 1, 0, 0, 0, 0, 1, 3, 2, 0},
			{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			{3468446081, 0, 2, 49154, 32770, 458768, 65536, 1, 1, 3, 2, 1},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{3468446081, 0, 2, 49154, 32770, 458768, 65536, 1, 1, 3, 2, 1},
		}},
	} {
		got, ok := providerHistoryContentJourneyUpper(row.limits, row.encoding, row.profile)
		if !ok {
			t.Fatal("ordinary portable mechanical domain", row.limits)
		}
		parts := [8]candidateWorkVector{got.projection, got.gas, got.trace, got.effects, got.control, got.normal, got.incoming, got.envelope}
		for i, want := range row.vectors {
			for j, coefficient := range row.logical[i] {
				want[10] += coefficient * sizes[j]
			}
			if parts[i] != want {
				t.Fatal("independent disjoint mechanical component", row.limits, i, parts[i], want)
			}
		}
	}
}

func TestProviderHistoryContentJourneyRefusesInvalidDataWithoutPartialRows(t *testing.T) {
	valid := providerHistoryContentJourneyLimits{2, 4, 2, 65536, 4096, 1024, 64 << 20, 4, 8, 8, 10, 1}
	for _, mode := range []string{"blocks-zero", "blocks-large", "windows-zero", "windows-large", "pending-zero", "pending-large", "evidence-zero", "evidence-large", "body-small", "body-large", "value-zero", "value-evidence", "value-body", "value-native", "native-zero", "native-small", "native-large", "transactions-zero", "transactions-large", "frames-zero", "frames-large", "logs-zero", "logs-large", "effects-large", "native-blocks-large", "encoding", "profile", "child-overflow"} {
		p := valid
		encoding, profile := EthereumJSONBlocks, SepoliaAmsterdamRootExecutionGasProfile
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
		case "value-zero":
			p.ValueBytes = 0
		case "value-evidence":
			p.EvidenceBytes = p.ValueBytes - 1
		case "value-body":
			p.BodyBytes = p.ValueBytes - 1
		case "value-native":
			p.TransactionBytes = p.ValueBytes - 1
		case "native-zero":
			p.TransactionBytes = 0
		case "native-small":
			p.TransactionBytes = p.BodyBytes + 1023
		case "native-large":
			p.TransactionBytes = (64 << 20) + 1
		case "transactions-zero":
			p.Transactions = 0
		case "transactions-large":
			p.Transactions = 16385
		case "frames-zero":
			p.Frames = 0
		case "frames-large":
			p.Frames = 65537
		case "logs-zero":
			p.Logs = 0
		case "logs-large":
			p.Logs = 65537
		case "effects-large":
			p.Effects = (1 << 20) + 1
		case "native-blocks-large":
			p.NativeBlocks = p.Blocks + 1
		case "encoding":
			encoding = "unselected"
		case "profile":
			profile = "unselected"
		case "child-overflow":
			p = providerHistoryContentJourneyLimits{1 << 20, 1 << 20, 10, 64 << 20, 8 << 20, 8 << 20, 64 << 20, 16384, 65536, 65536, 1 << 20, 1 << 20}
		}
		if got, ok := providerHistoryContentJourneyUpper(p, encoding, profile); ok || got != (providerHistoryContentJourneyCost{}) {
			t.Fatal("invalid mechanical join retained partial data", mode, got, ok)
		}
	}
	for i := range 12 {
		p := valid
		fields := []*uint64{&p.Windows, &p.Blocks, &p.PendingBlocks, &p.EvidenceBytes, &p.BodyBytes, &p.ValueBytes, &p.TransactionBytes, &p.Transactions, &p.Frames, &p.Logs, &p.Effects, &p.NativeBlocks}
		*fields[i] = ^uint64(0)
		if got, ok := providerHistoryContentJourneyUpper(p, EthereumJSONBlocks, SepoliaAmsterdamGasProfile); ok || got != (providerHistoryContentJourneyCost{}) {
			t.Fatal("MAX mechanical field retained partial rows", i, got, ok)
		}
	}
}

func TestProviderHistoryContentJourneySelectsOneOriginalIncomingRoom(t *testing.T) {
	for _, row := range []struct {
		encoding                  string
		effects, native, incoming uint64
	}{
		{EthereumJSONBlocks, 0, 0, 4503633987174401},
		{"debug-raw-rlp-v1", 0, 0, 4294967297},
		{"debug-raw-rlp-v1", 1, 0, 4503633987174401},
		{"debug-raw-rlp-v1", 0, 1, 4503633987174401},
	} {
		p := providerHistoryContentJourneyLimits{1, 1, 1, 65536, 2048, 1024, 64 << 20, 2, 16, 2, row.effects, row.native}
		got, ok := providerHistoryContentJourneyUpper(p, row.encoding, SepoliaAmsterdamRootRegularGasProfile)
		if !ok || got.incoming != (candidateWorkVector{10: row.incoming}) {
			t.Fatal("lost original wide incoming ceiling or summed independent pending preflights", row, got, ok)
		}
		for c, n := range got.normal {
			if got.envelope[c] != n+got.incoming[c] {
				t.Fatal("incoming changed an actual component", c, got)
			}
		}
	}
}

func TestProviderHistoryContentJourneyRefusesParentOverflowAfterFiniteChildren(t *testing.T) {
	// All large counts are finite scalar DATA: no corresponding arrays, native
	// calls, source records, time windows or resource workload are constructed.
	for _, size := range historyContentJourneySizes() {
		if size > 4096 {
			t.Fatal("changed logical target requires a new portable overflow witness")
		}
	}
	p := providerHistoryContentJourneyLimits{1150, 1150, 10, 64 << 20, 8 << 20, 8 << 20, 64 << 20, 16384, 65536, 65536, 1150, 1150}
	_, projectionOK := providerHistoryBlockProjectionUpper(providerHistoryBlockProjectionLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes, p.TransactionBytes, p.Transactions, p.Logs, p.NativeBlocks}, "debug-raw-rlp-v1")
	_, gasOK := providerHistoryGasJourneyUpper(providerHistoryGasJourneyLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes, p.ValueBytes, p.TransactionBytes, p.Transactions, p.NativeBlocks}, SepoliaAmsterdamRootExecutionGasProfile)
	_, traceOK := providerHistoryTraceJourneyUpper(providerHistoryTraceJourneyLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes, p.ValueBytes, p.TransactionBytes, p.Transactions, p.Frames, p.Logs, p.Effects})
	_, effectsOK := providerHistoryEffectReadyUpper(providerHistoryEffectReadyLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes, p.TransactionBytes, p.ValueBytes, p.Transactions, p.Logs, p.Effects})
	if !projectionOK || !gasOK || !traceOK || !effectsOK {
		t.Fatal("parent-only overflow witness stopped in a child", projectionOK, gasOK, traceOK, effectsOK)
	}
	if got, ok := providerHistoryContentJourneyUpper(p, "debug-raw-rlp-v1", SepoliaAmsterdamRootExecutionGasProfile); ok || got != (providerHistoryContentJourneyCost{}) {
		t.Fatal("late parent overflow leaked components or pending room", got, ok)
	}
}

// All caps come from the original selected fixture BEFORE the target call.
// This private deployment-zero cursor cannot authenticate a Source or chain.
func historyContentJourneyQuote(t *testing.T, c *providerHistoryCollectionCursor) providerHistoryContentJourneyCost {
	t.Helper()
	in := c.selection.input
	l := in.journal.Replay
	bound, original := providerHistoryNativeInputBoundObserved(c.semantic.observer)
	if !original || bound.collection || bound.transaction != 64<<20 {
		t.Fatal("original genuine wide inert native selector changed")
	}
	p := providerHistoryContentJourneyLimits{(in.blocks + in.windowBlocks - 1) / in.windowBlocks, in.blocks, in.contentBlocks,
		uint64(l.MaxEvidenceBytes), uint64(in.capture.BodyBytes), min(uint64(l.MaxEvidenceBytes), uint64(in.capture.BodyBytes)), bound.transaction,
		uint64(l.MaxTransactions), uint64(l.MaxFrames), uint64(l.MaxLogs), in.effects, in.native}
	got, ok := providerHistoryContentJourneyUpper(p, in.encoding, in.gas)
	if !ok {
		t.Fatal("original source caps did not admit finite mechanical DATA", p)
	}
	return got
}

func historyContentJourneyReplayLimits() ReplayLimits {
	limits := replayTestLimits()
	limits.MaxEvidenceBytes = 1 << 20
	limits.MaxTransactions = 64
	limits.MaxFrames = 512
	limits.MaxLogs = 256
	return limits
}

func TestProviderHistoryContentJourneyInitialContentKeepsAssignedMechanicalScope(t *testing.T) {
	c, in, methods := historyCollectionContentInputLimitsFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile, historyContentJourneyReplayLimits())
	c = historyCollectionContentStep(t, c, in, methods) // Original open S.
	upper := historyContentJourneyQuote(t, c)
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, in, table)
	m := c.semantic.observer
	before := m.actual
	if !c.contentBoundaryObserved(context.Background(), results, m) {
		t.Fatal("ordinary initial private content boundary")
	}
	delta := historyEventQueueDelta(t, m, before)
	if c.phase != "effects" || len(c.pending) != 2 || len(c.effects) != 40 || c.effectNext != 0 || c.traces != 0 || c.semantic.blocks != 0 || m.failed || m.clocks != 0 || delta[2] == 0 || delta[3] != 2 || delta[4] != 40 || delta[5] != 0 || delta[6] != 40 {
		t.Fatal("initial mechanical prefix entered semantic event callbacks or lost canonical counts", delta)
	}
	historyEventQueueWithin(t, delta, upper.normal)
	// No trace results are supplied: this invocation stops at the initial wait,
	// BEFORE any blockDeriving command/claim/state/queue callback. A later S
	// requires those separate rows, rather than borrowing this quote's slack.
}

func TestProviderHistoryContentJourneyZeroTraceCapacityKeepsEnteredPrefix(t *testing.T) {
	in := historyCollectionStreamingInput(t)
	for i := range in.Blocks {
		in.Blocks[i].Executions = nil
	} // Before immutable fixture.
	c, in, methods := historyCollectionContentInputLimitsFixture(t, 2, in, SepoliaAmsterdamRootExecutionGasProfile, historyContentJourneyReplayLimits())
	c = historyCollectionContentStep(t, c, in, methods)
	upper := historyContentJourneyQuote(t, c)
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, in, table)
	m := c.semantic.observer
	before := m.actual
	if c.contentBoundaryObserved(context.Background(), results, m) {
		t.Fatal("zero trace capacity accepted ordinary selected effects")
	}
	delta := historyEventQueueDelta(t, m, before)
	if c.selection.input.capture.BodyBytes != 128<<10 || len(c.pending) != 2 || len(c.effects) != 0 || c.effectNext != 0 || c.traces != 0 || c.semantic.blocks != 0 || m.failed || m.clocks != 0 || delta[3] != 1 || delta[4] != 2 || delta[6] != 2 || upper.trace != (candidateWorkVector{}) || upper.incoming != (candidateWorkVector{10: 17179869185}) {
		t.Fatal("zero-cap refusal lost its completed first-block prefix", delta, upper)
	}
	historyEventQueueWithin(t, delta, upper.normal)
	// Stop after the first ordinary refusal. No failed S is resumed or retried.
}

func TestProviderHistoryContentJourneyDeclaresPendingBeforeEvidenceFailure(t *testing.T) {
	limits := historyContentJourneyReplayLimits()
	limits.MaxEvidenceBytes = 1
	c, in, methods := historyCollectionContentInputLimitsFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile, limits)
	c = historyCollectionContentStep(t, c, in, methods)
	upper := historyContentJourneyQuote(t, c)
	table := historyCollectionBoundaryPlan(t, c, methods)
	results := historyCollectionContentResults(t, c, in, table)
	m := c.semantic.observer
	before := m.actual
	if c.contentBoundaryObserved(context.Background(), results, m) {
		t.Fatal("one-byte cap accepted original content")
	}
	delta := historyEventQueueDelta(t, m, before)
	if len(c.pending) != 2 || len(c.effects) != 0 || c.semantic.blocks != 0 || m.failed || m.clocks != 0 || delta[2] == 0 || delta[3] != 0 || delta[4] != 0 || delta[5] != 0 || delta[6] != 0 || delta[10] < 2*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())+1 {
		t.Fatal("first decode refusal erased declared pending or entered later blocks", delta)
	}
	historyEventQueueWithin(t, delta, upper.normal)
}
