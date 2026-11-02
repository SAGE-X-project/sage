package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// These complete portable literals come from separate exact-integer source
// equations. They do not execute a second copy of the cost constructor.
func TestProviderHistoryBlockProjectionMatchesIndependentPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	body := uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	for _, row := range []struct {
		limits      providerHistoryBlockProjectionLimits
		encoding    string
		diagnostics [13]uint64
		parts       [15][8]uint64 // decoded,blocks,txs,logs,logical,word,body,gas.
	}{
		{
			providerHistoryBlockProjectionLimits{1, 1, 1, 65536, 2048, 3072, 2, 2, 0}, EthereumJSONBlocks,
			[13]uint64{2048, 2048, 2048, 2048, 131072, 2, 2, 2048, 2048, 16, 16, 16, 69632},
			[15][8]uint64{
				{10240, 0, 0, 0, 42006984, 48, 0, 0},           // successfulCommon
				{0, 0, 0, 0, 268435458, 6, 0, 0},               // successfulMaterialization
				{0, 0, 0, 0, 4194306, 0, 1, 0},                 // successfulBuilder
				{113570, 1, 2, 2, 1629883855, 61644, 0, 0},     // successfulBranch
				{0, 0, 0, 0, 331022524, 0, 0, 0},               // successfulExtraTries
				{10240, 0, 0, 0, 42006984, 48, 0, 0},           // failedCommon
				{0, 0, 0, 0, 4194306, 6, 0, 0},                 // failedMaterialization
				{0, 0, 0, 0, 4194306, 0, 1, 0},                 // failedBuilder
				{113570, 1, 2, 2, 22727440847, 61644, 0, 0},    // failedBranch
				{0, 0, 0, 0, 62394556, 0, 0, 0},                // failedExtraTries
				{157696, 0, 0, 0, 14584107657, 0, 0, 0},        // failedJSONAdditional
				{0, 0, 0, 0, 393230, 20, 0, 0},                 // retention
				{196608, 2, 6, 6, 38682254622, 0, 0, 0},        // decoder
				{601924, 4, 10, 10, 78382529635, 123416, 2, 0}, // normal
				{601924, 4, 10, 10, 83231145060, 123416, 2, 0}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{2, 5, 2, 65536, 8192, 9216, 3, 5, 3}, EthereumJSONBlocks,
			[13]uint64{8192, 8192, 40960, 40960, 262144, 12, 20, 40960, 40960, 16, 16, 16, 81920},
			[15][8]uint64{
				{204800, 0, 0, 0, 3356008680, 240, 0, 0},           // successfulCommon
				{0, 0, 0, 0, 2147483660, 36, 0, 0},                 // successfulMaterialization
				{0, 0, 0, 0, 335544330, 0, 5, 0},                   // successfulBuilder
				{1678860, 5, 12, 20, 24929419743, 1230088, 0, 0},   // successfulBranch
				{0, 0, 0, 0, 2450461436, 0, 0, 0},                  // successfulExtraTries
				{40960, 0, 0, 0, 671201736, 48, 0, 0},              // failedCommon
				{0, 0, 0, 0, 67108867, 9, 0, 0},                    // failedMaterialization
				{0, 0, 0, 0, 67108866, 0, 1, 0},                    // failedBuilder
				{337203, 1, 3, 5, 31860466651, 246050, 0, 0},       // failedBranch
				{0, 0, 0, 0, 127223908, 0, 0, 0},                   // failedExtraTries
				{237568, 0, 0, 0, 20737483401, 0, 0, 0},            // failedJSONAdditional
				{0, 0, 0, 0, 786519, 120, 0, 3},                    // retention
				{393216, 10, 18, 30, 77384980878, 0, 0, 0},         // decoder
				{2892607, 16, 33, 55, 164135278675, 1476591, 6, 3}, // normal
				{2892607, 16, 33, 55, 170846165076, 1476591, 6, 3}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{1, 100, 10, 65536, 131072, 132096, 2, 2, 4}, EthereumJSONBlocks,
			[13]uint64{65536, 65536, 801006, 720896, 720896, 22, 22, 6553600, 0, 16, 16, 16, 196608},
			[15][8]uint64{
				{3924920, 0, 0, 0, 503982472426, 4800, 0, 0},                // successfulCommon
				{0, 0, 0, 0, 47244640278, 66, 0, 0},                         // successfulMaterialization
				{0, 0, 0, 0, 52494729416, 0, 100, 0},                        // successfulBuilder
				{160851766, 100, 22, 22, 21902492815840, 117978436, 0, 0},   // successfulBranch
				{0, 0, 0, 0, 52778563232, 0, 0, 0},                          // successfulExtraTries
				{327680, 0, 0, 0, 42950375880, 48, 0, 0},                    // failedCommon
				{0, 0, 0, 0, 4294967298, 6, 0, 0},                           // failedMaterialization
				{0, 0, 0, 0, 4294967298, 0, 1, 0},                           // failedBuilder
				{1612706, 1, 2, 2, 354959352271, 1179852, 0, 0},             // failedBranch
				{0, 0, 0, 0, 4355326140, 0, 0, 0},                           // failedExtraTries
				{983040, 0, 0, 0, 154622763657, 0, 0, 0},                    // failedJSONAdditional
				{0, 0, 0, 0, 2753212, 220, 0, 4},                            // retention
				{786432, 200, 24, 24, 154729045560, 0, 0, 0},                // decoder
				{168486544, 301, 48, 48, 23279202772508, 119163428, 101, 4}, // normal
				{168486544, 301, 48, 48, 23317857478173, 119163428, 101, 4}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{1, 1, 1, 65536, 2048, 3072, 2, 2, 0}, "debug-raw-rlp-v1",
			[13]uint64{2048, 2048, 2048, 2048, 131072, 2, 2, 2048, 2048, 16384, 409, 409, 65536},
			[15][8]uint64{
				{10240, 0, 0, 0, 81322920, 49152, 0, 0},     // successfulCommon
				{0, 0, 0, 0, 268435458, 6, 0, 0},            // successfulMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                    // successfulBuilder
				{10236, 0, 0, 0, 20959256, 18, 0, 0},        // successfulBranch
				{0, 0, 0, 0, 28016033796, 0, 0, 0},          // successfulExtraTries
				{10240, 0, 0, 0, 42950970, 1227, 0, 0},      // failedCommon
				{0, 0, 0, 0, 4194306, 6, 0, 0},              // failedMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                    // failedBuilder
				{10236, 0, 0, 0, 20959256, 18, 0, 0},        // failedBranch
				{0, 0, 0, 0, 699374871, 0, 0, 0},            // failedExtraTries
				{0, 0, 0, 0, 0, 0, 0, 0},                    // failedJSONAdditional
				{0, 0, 0, 0, 393230, 20, 0, 0},              // retention
				{196608, 2, 6, 6, 38682254622, 0, 0, 0},     // decoder
				{237560, 2, 6, 6, 67836878685, 50447, 0, 0}, // normal
				{237560, 2, 6, 6, 72131845982, 50447, 0, 0}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{2, 5, 2, 65536, 8192, 67108864, 3, 5, 3}, "debug-raw-rlp-v1",
			[13]uint64{8192, 8192, 40960, 40960, 262144, 12, 20, 40960, 40960, 16384, 1638, 1638, 65536},
			[15][8]uint64{
				{204800, 0, 0, 0, 3552588360, 245760, 0, 0},      // successfulCommon
				{0, 0, 0, 0, 2147483660, 36, 0, 0},               // successfulMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                         // successfulBuilder
				{204780, 0, 0, 0, 1677475967, 126, 0, 0},         // successfulBranch
				{0, 0, 0, 0, 140080168980, 0, 0, 0},              // successfulExtraTries
				{40960, 0, 0, 0, 675097780, 4914, 0, 0},          // failedCommon
				{0, 0, 0, 0, 67108867, 9, 0, 0},                  // failedMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                         // failedBuilder
				{40956, 0, 0, 0, 335495194, 27, 0, 0},            // failedBranch
				{0, 0, 0, 0, 2800919398, 0, 0, 0},                // failedExtraTries
				{0, 0, 0, 0, 0, 0, 0, 0},                         // failedJSONAdditional
				{0, 0, 0, 0, 786519, 120, 0, 3},                  // retention
				{393216, 10, 18, 30, 77384980878, 0, 0, 0},       // decoder
				{884712, 10, 18, 30, 228722105603, 250992, 0, 3}, // normal
				{884712, 10, 18, 30, 233017072900, 250992, 0, 3}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{1, 1, 10, 1, 4, 1028, 16, 1, 0}, EthereumJSONBlocks,
			[13]uint64{1, 1, 1, 1, 11, 16, 1, 1, 0, 16, 16, 16, 65538},
			[15][8]uint64{
				{5, 0, 0, 0, 47584, 48, 0, 0},                  // successfulCommon
				{0, 0, 0, 0, 27, 48, 0, 0},                     // successfulMaterialization
				{0, 0, 0, 0, 3, 0, 1, 0},                       // successfulBuilder
				{72088, 1, 16, 1, 106391223, 394, 0, 0},        // successfulBranch
				{0, 0, 0, 0, 82013549, 0, 0, 0},                // successfulExtraTries
				{5, 0, 0, 0, 47584, 48, 0, 0},                  // failedCommon
				{0, 0, 0, 0, 17, 48, 0, 0},                     // failedMaterialization
				{0, 0, 0, 0, 3, 0, 1, 0},                       // failedBuilder
				{72088, 1, 16, 1, 81917941551, 394, 0, 0},      // failedBranch
				{0, 0, 0, 0, 82013199, 0, 0, 0},                // failedExtraTries
				{131085, 0, 0, 0, 12886090452, 0, 0, 0},        // failedJSONAdditional
				{0, 0, 0, 0, 112, 160, 0, 0},                   // retention
				{12, 2, 192, 12, 655065966, 0, 0, 0},           // decoder
				{275283, 4, 224, 14, 95729611270, 1140, 2, 0},  // normal
				{275283, 4, 224, 14, 100024840715, 1140, 2, 0}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{3, 3, 10, 67108864, 8388608, 8389632, 16384, 65536, 3}, EthereumJSONBlocks,
			[13]uint64{8388608, 65536, 25165824, 196608, 872415232, 49152, 196608, 25165824, 25165824, 16, 16, 16, 16842752},
			[15][8]uint64{
				{100859904, 0, 0, 0, 1266689061367128, 144, 0, 0},                      // successfulCommon
				{0, 0, 0, 0, 7318349394526208, 147456, 0, 0},                           // successfulMaterialization
				{0, 0, 0, 0, 211106232532998, 0, 3, 0},                                 // successfulBuilder
				{1028807616, 3, 49152, 196608, 44720081997637465, 760381824, 0, 0},     // successfulBranch
				{0, 0, 0, 0, 7318463068951620, 0, 0, 0},                                // successfulExtraTries
				{33619968, 0, 0, 0, 422229687122376, 48, 0, 0},                         // failedCommon
				{0, 0, 0, 0, 70368744194048, 49152, 0, 0},                              // failedMaterialization
				{0, 0, 0, 0, 70368744177666, 0, 1, 0},                                  // failedBuilder
				{342935872, 1, 16384, 65536, 3415943210137203, 253460608, 0, 0},        // failedBranch
				{0, 0, 0, 0, 70397033509228, 0, 0, 0},                                  // failedExtraTries
				{109182976, 0, 0, 0, 1484354035786377, 0, 0, 0},                        // failedJSONAdditional
				{0, 0, 0, 0, 3087204379, 491520, 0, 3},                                 // retention
				{1073741824, 6, 262144, 1048576, 27022530801304398, 0, 0, 0},           // decoder
				{2689148160, 10, 327680, 1310720, 93390885098451094, 1014530752, 4, 3}, // normal
				{2689148160, 10, 327680, 1310720, 93674563393384599, 1014530752, 4, 3}, // envelope
			},
		},
		{
			providerHistoryBlockProjectionLimits{1, 1, 1, 67108864, 8388608, 8389632, 16384, 65536, 1}, "debug-raw-rlp-v1",
			[13]uint64{8388608, 65536, 8388608, 65536, 134217728, 16384, 65536, 8388608, 8388608, 16384, 65536, 16384, 8388608},
			[15][8]uint64{
				{33619968, 0, 0, 0, 422229726438312, 49152, 0, 0},             // successfulCommon
				{0, 0, 0, 0, 1125899906859008, 49152, 0, 0},                   // successfulMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                                      // successfulBuilder
				{41943036, 0, 0, 0, 351843670589460, 147456, 0, 0},            // successfulBranch
				{0, 0, 0, 0, 28016033796, 0, 0, 0},                            // successfulExtraTries
				{33619968, 0, 0, 0, 422229844501416, 196608, 0, 0},            // failedCommon
				{0, 0, 0, 0, 70368744194048, 49152, 0, 0},                     // failedMaterialization
				{0, 0, 0, 0, 0, 0, 0, 0},                                      // failedBuilder
				{41943036, 0, 0, 0, 351843670589460, 147456, 0, 0},            // failedBranch
				{0, 0, 0, 0, 28016033796, 0, 0, 0},                            // failedExtraTries
				{0, 0, 0, 0, 0, 0, 0, 0},                                      // failedJSONAdditional
				{0, 0, 0, 0, 402718729, 163840, 0, 1},                         // retention
				{201326592, 2, 49152, 196608, 5066724525244698, 0, 0, 0},      // decoder
				{352452600, 2, 49152, 196608, 7811196523202723, 802816, 0, 1}, // normal
				{352452600, 2, 49152, 196608, 7881565267380388, 802816, 0, 1}, // envelope
			},
		},
	} {
		got, ok := providerHistoryBlockProjectionUpper(row.limits, row.encoding)
		shape := [13]uint64{got.body, got.header, got.bodyPool, got.headerPool, got.canonicalPool, got.txCount, got.logCount, got.extremalBlock, got.extremalReceipt, got.successfulWithdrawals, got.failedWithdrawals, got.failedTrieWithdrawals, got.incomingMaximum}
		if !ok || shape != row.diagnostics {
			t.Fatal("independent whole projection source dimensions", got, row, ok)
		}
		parts := [15]candidateWorkVector{got.successfulCommon, got.successfulMaterialization, got.successfulBuilder, got.successfulBranch, got.successfulExtraTries, got.failedCommon, got.failedMaterialization, got.failedBuilder, got.failedBranch, got.failedExtraTries, got.failedJSONAdditional, got.retention, got.decoder, got.normal, got.envelope}
		for i, p := range row.parts {
			want := candidateWorkVector{2: p[0], 3: p[1], 4: p[2], 6: p[3], 10: p[4] + p[5]*w + p[6]*body + p[7]*gas}
			if parts[i] != want {
				t.Fatal("independent complete projection component", row.encoding, i, parts[i], want)
			}
		}
	}
}

func TestProviderHistoryBlockProjectionRejectsInvalidAndOverflowingData(t *testing.T) {
	valid := providerHistoryBlockProjectionLimits{2, 4, 2, 65536, 4096, 5120, 4, 8, 2}
	for _, mode := range []string{"blocks-zero", "blocks-large", "windows-zero", "windows-large", "pending-zero", "pending-large", "evidence-zero", "evidence-large", "body-small", "body-large", "native-small", "native-large", "transactions-zero", "transactions-large", "logs-zero", "logs-large", "native-blocks-large", "own-late-overflow", "child-late-overflow"} {
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
		case "native-small":
			p.TransactionBytes = p.BodyBytes + 1023
		case "native-large":
			p.TransactionBytes = (64 << 20) + 1
		case "transactions-zero":
			p.Transactions = 0
		case "transactions-large":
			p.Transactions = 16385
		case "logs-zero":
			p.Logs = 0
		case "logs-large":
			p.Logs = 65537
		case "native-blocks-large":
			p.NativeBlocks = p.Blocks + 1
		case "own-late-overflow":
			p = providerHistoryBlockProjectionLimits{1, 1 << 20, 1, 64 << 20, 8 << 20, 64 << 20, 16384, 65536, 0}
		case "child-late-overflow":
			p = providerHistoryBlockProjectionLimits{1 << 20, 1 << 20, 10, 64 << 20, 8 << 20, 64 << 20, 16384, 65536, 1 << 20}
		}
		if mode == "own-late-overflow" {
			if _, childOK := providerHistoryContentDecodePoolUpper(providerHistoryContentDecodePoolLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, min(p.BodyBytes, p.EvidenceBytes), p.Transactions, p.Logs}); !childOK {
				t.Fatal("own arithmetic overflow precursor must leave decoder child finite")
			}
		}
		if got, ok := providerHistoryBlockProjectionUpper(p, EthereumJSONBlocks); ok || got != (providerHistoryBlockProjectionCost{}) {
			t.Fatal("invalid or overflowing data leaked partial whole cost", mode, got, ok)
		}
	}
	for i := range 9 {
		p := valid
		fields := []*uint64{&p.Windows, &p.Blocks, &p.PendingBlocks, &p.EvidenceBytes, &p.BodyBytes, &p.TransactionBytes, &p.Transactions, &p.Logs, &p.NativeBlocks}
		*fields[i] = ^uint64(0)
		if got, ok := providerHistoryBlockProjectionUpper(p, EthereumJSONBlocks); ok || got != (providerHistoryBlockProjectionCost{}) {
			t.Fatal("MAX field leaked whole projection data", i, got, ok)
		}
	}
	if got, ok := providerHistoryBlockProjectionUpper(valid, "unselected"); ok || got != (providerHistoryBlockProjectionCost{}) {
		t.Fatal("unselected encoding gained a quote", got, ok)
	}
}

func TestProviderHistoryBlockProjectionKeepsOriginalDecoderAndIncomingScope(t *testing.T) {
	for _, encoding := range []string{EthereumJSONBlocks, "debug-raw-rlp-v1"} {
		p := providerHistoryBlockProjectionLimits{2, 5, 2, 65536, 8192, 64 << 20, 3, 5, 3}
		got, ok := providerHistoryBlockProjectionUpper(p, encoding)
		child, childOK := providerHistoryContentDecodePoolUpper(providerHistoryContentDecodePoolLimits{p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, min(p.BodyBytes, p.EvidenceBytes), p.Transactions, p.Logs})
		if !ok || !childOK || got.decoder != child.normal {
			t.Fatal("whole decoder assignment changed", got, child, ok, childOK)
		}
		room := uint64(4294967297)
		if encoding == EthereumJSONBlocks {
			room = 4503633987174401
		}
		if got.envelope[10]-got.normal[10] != room {
			t.Fatal("original native/JSON incoming preflight was replaced", got, room)
		}
		for i := range got.normal {
			if i != 10 && got.normal[i] != got.envelope[i] {
				t.Fatal("incoming padding fabricated actual coordinates", i, got)
			}
		}
	}
}

// Ordinary offline original checker DATA. No source, selected chain, gas or
// collection admission is claimed. All fixture encoding and settings precede
// the target observer, whose genuine older native selector remains64MiB.
func historyBlockProjectionInput(t *testing.T, encoding string, block, receipts []byte) (providerHistoryCollectionInput, providerHistoryBlockProjectionCost) {
	t.Helper()
	in := historyCollectionInput(t)
	in.encoding, in.journal.Replay = encoding, nodeJSONLimits().Replay
	in.capture.HeaderBytes, in.capture.BodyBytes, in.capture.WireBytes, in.capture.JSONDepth = 64<<10, max(4, len(block), len(receipts)), 16<<20, 128
	l := in.journal.Replay
	upper, ok := providerHistoryBlockProjectionUpper(providerHistoryBlockProjectionLimits{
		1, 1, 1, uint64(l.MaxEvidenceBytes), uint64(in.capture.BodyBytes), 64 << 20, uint64(l.MaxTransactions), uint64(l.MaxLogs), 1,
	}, encoding)
	if !ok || !in.capture.valid() {
		t.Fatal("ordinary offline projection prerequisites", upper, ok)
	}
	return in, upper
}

func historyBlockProjectionDebug(t *testing.T, block []byte, receipts [][]byte) ([]byte, []byte) {
	t.Helper()
	a, err := json.Marshal(nodeHex(block))
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]string, len(receipts))
	for i, raw := range receipts {
		rows[i] = nodeHex(raw)
	}
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

func TestProviderHistoryBlockProjectionOriginalCompleteJSONAndDebugRemainCanonical(t *testing.T) {
	block, blockJSON, receiptJSON, receipts := nodeJSONFixtureBytes(t)
	binary, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	header, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := replayHeader(block.Header())
	expectedTx := make([][]byte, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		expectedTx[i], err = tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, encoding := range []string{EthereumJSONBlocks, "debug-raw-rlp-v1"} {
		a, b := blockJSON, receiptJSON
		if encoding == "debug-raw-rlp-v1" {
			a, b = historyBlockProjectionDebug(t, binary, receipts)
		}
		in, upper := historyBlockProjectionInput(t, encoding, a, b)
		m := providerHistorySemanticUnitMeter()
		out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(a), string(b), wantHeader, in, m)
		if err != nil || m.failed || decoded.header == nil || !bytes.Equal(out.HeaderRLP, header) || len(out.Transactions) != len(block.Transactions()) || len(out.Receipts) != len(receipts) {
			t.Fatal("complete original projection", encoding, out, decoded, err, m.actual)
		}
		for i, raw := range expectedTx {
			if !bytes.Equal(out.Transactions[i], raw) || !bytes.Equal(out.Receipts[i], receipts[i]) {
				t.Fatal("original canonical transaction/receipt changed", encoding, i)
			}
		}
		if m.actual[2] == 0 || m.actual[3] == 0 || m.actual[4] == 0 || m.clocks != 0 {
			t.Fatal("complete original decoder was not observed", encoding, m.actual)
		}
		historyEventQueueWithin(t, m.actual, upper.normal)
	}
}

func TestProviderHistoryBlockProjectionCopiesBothOriginalsBeforeEvidenceRefusal(t *testing.T) {
	block, a, b, _ := nodeJSONFixtureBytes(t)
	wantHeader := replayHeader(block.Header())
	in, _ := historyBlockProjectionInput(t, EthereumJSONBlocks, a, b)
	// Select this ordinary small scalar cap BEFORE constructing the target;
	// neither observer.actual nor any live quota/budget is reset or edited.
	in.journal.Replay.MaxEvidenceBytes = 1
	upper, ok := providerHistoryBlockProjectionUpper(providerHistoryBlockProjectionLimits{1, 1, 1, 1, uint64(in.capture.BodyBytes), 64 << 20, uint64(in.journal.Replay.MaxTransactions), uint64(in.journal.Replay.MaxLogs), 1}, EthereumJSONBlocks)
	if !ok {
		t.Fatal("before-E source quote")
	}
	m := providerHistorySemanticUnitMeter()
	out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(a), string(b), wantHeader, in, m)
	want := candidateWorkVector{10: uint64(len(a)+len(b)) + 2}
	if err == nil || m.failed || out.HeaderRLP != nil || decoded.header != nil || m.actual != want || in.journal.Replay.MaxEvidenceBytes != 1 {
		t.Fatal("original copies were erased by later evidence guard", out, decoded, err, m.actual, want)
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
}

func TestProviderHistoryBlockProjectionKeepsBothEagerOverflowQuantities(t *testing.T) {
	// Two ordinary17-digit uint64 overflows, never an exhaustion payload.
	// Both original quantity calls and both hash-text encodes execute before
	// the combined bool returns false. The exact event sum is independent.
	fields := map[string]json.RawMessage{}
	nodeJSONPut(fields, "blockHash", (common.Hash{}).Hex())
	nodeJSONPut(fields, "transactionHash", (common.Hash{}).Hex())
	nodeJSONPut(fields, "blockNumber", "0x10000000000000000")
	nodeJSONPut(fields, "transactionIndex", "0x10000000000000000")
	m := providerHistorySemanticUnitMeter()
	if nodeJSONLocationObserved(fields, common.Hash{}, 0, common.Hash{}, 0, m) || m.failed || m.actual != (candidateWorkVector{2: 212, 10: 19650}) {
		t.Fatal("eager original location quantity prefix changed", m.actual)
	}
}

func TestProviderHistoryBlockProjectionRefusesOrdinaryJSONBoundaryMismatches(t *testing.T) {
	block, blockJSON, receiptJSON, _ := nodeJSONFixtureBytes(t)
	wantHeader := replayHeader(block.Header())
	for _, mode := range []string{"from", "quantities", "header-binding", "unknown-field", "receipt-location"} {
		a, b := bytes.Clone(blockJSON), bytes.Clone(receiptJSON)
		if mode == "header-binding" {
			hash := block.Hash()
			hash[31] ^= 1
			a = nodeJSONEditObject(a, func(m map[string]json.RawMessage) { nodeJSONPut(m, "hash", hash.Hex()) })
		} else if mode == "receipt-location" {
			var rows []json.RawMessage
			if json.Unmarshal(b, &rows) != nil || len(rows) == 0 {
				t.Fatal("ordinary receipt precursor")
			}
			hash := block.Hash()
			hash[31] ^= 1
			rows[len(rows)-1] = nodeJSONEditObject(rows[len(rows)-1], func(m map[string]json.RawMessage) { nodeJSONPut(m, "blockHash", hash.Hex()) })
			var err error
			b, err = json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			a = nodeJSONEditObject(a, func(object map[string]json.RawMessage) {
				var rows []json.RawMessage
				if json.Unmarshal(object["transactions"], &rows) != nil || len(rows) == 0 {
					t.Fatal("ordinary transaction precursor")
				}
				rows[len(rows)-1] = nodeJSONEditObject(rows[len(rows)-1], func(m map[string]json.RawMessage) {
					switch mode {
					case "from":
						var from string
						if json.Unmarshal(m["from"], &from) != nil || len(from) != 42 {
							t.Fatal("ordinary sender field")
						}
						last := byte('0')
						if from[len(from)-1] == '0' {
							last = '1'
						}
						nodeJSONPut(m, "from", from[:len(from)-1]+string(last))
					case "quantities":
						nodeJSONPut(m, "blockNumber", "0x10000000000000000")
						nodeJSONPut(m, "transactionIndex", "0x10000000000000000")
					case "unknown-field":
						nodeJSONPut(m, "unrecognizedField", "ordinary")
					}
				})
				nodeJSONPut(object, "transactions", rows)
			})
		}
		in, upper := historyBlockProjectionInput(t, EthereumJSONBlocks, a, b)
		m := providerHistorySemanticUnitMeter()
		out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(a), string(b), wantHeader, in, m)
		if err == nil || m.failed || out.HeaderRLP != nil || decoded.header != nil || m.actual[2] == 0 || m.actual[10] == 0 || m.clocks != 0 {
			t.Fatal("ordinary malformed DATA was accepted or lost its prefix", mode, out, decoded, err, m.actual)
		}
		if (mode == "from" || mode == "quantities") && m.actual[2] < 97 {
			t.Fatal("early native sender prefix missing", mode, m.actual)
		}
		historyEventQueueWithin(t, m.actual, upper.normal)
	}
}

func TestProviderHistoryBlockProjectionDebugRefusesWrongWithdrawalRootAfterBody(t *testing.T) {
	f := historyBinaryWrapperRecorded(t)
	header := f.block.Header()
	if header.WithdrawalsHash == nil {
		t.Fatal("ordinary post-fork withdrawal precursor")
	}
	wrong := *header.WithdrawalsHash
	wrong[31] ^= 1
	header.WithdrawalsHash = &wrong
	block := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: f.block.Transactions(), Uncles: f.block.Uncles(), Withdrawals: f.block.Withdrawals()})
	binary, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	a, b := historyBlockProjectionDebug(t, binary, f.raw.Receipts)
	in, upper := historyBlockProjectionInput(t, "debug-raw-rlp-v1", a, b)
	wantHeader := replayHeader(header)
	m := providerHistorySemanticUnitMeter()
	out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(a), string(b), wantHeader, in, m)
	if err != ErrReplay || m.failed || out.HeaderRLP != nil || decoded.header != nil || m.actual[2] == 0 || m.actual[3] != 0 || m.actual[10] == 0 {
		t.Fatal("wrong withdrawal root survived the original common boundary", out, decoded, err, m.actual)
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
	// Large65536/16384 count ceilings are covered by the portable integer
	// rows only. This test allocates no oversized withdrawal or trie array.
}

func TestProviderHistoryBlockProjectionRetainsAndRebuildsAfterFullDecode(t *testing.T) {
	block, a, b, _ := nodeJSONFixtureBytes(t)
	in, upper := historyBlockProjectionInput(t, EthereumJSONBlocks, a, b)
	want := replayHeader(block.Header())
	m := providerHistorySemanticUnitMeter()
	raw, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(a), string(b), want, in, m)
	if err != nil || decoded.header == nil {
		t.Fatal("complete content prerequisite", err)
	}
	before := bytes.Clone(raw.HeaderRLP)
	retained, ok := providerHistoryCollectionRetainBlockObserved(raw, want, m)
	if !ok {
		t.Fatal("original immutable retention")
	}
	copy, ok := retained.replayObserved(m)
	if !ok || m.failed || !bytes.Equal(copy.HeaderRLP, before) || len(copy.Transactions) != len(raw.Transactions) || len(copy.Receipts) != len(raw.Receipts) {
		t.Fatal("original rebuild", copy, ok, m.actual)
	}
	for i := range raw.Transactions {
		if !bytes.Equal(copy.Transactions[i], raw.Transactions[i]) || !bytes.Equal(copy.Receipts[i], raw.Receipts[i]) {
			t.Fatal("immutable arrays changed", i)
		}
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
	copy.HeaderRLP[0] ^= 1
	if !bytes.Equal(raw.HeaderRLP, before) || retained.headerRLP != string(before) {
		t.Fatal("rebuild borrowed immutable header storage")
	}
}
