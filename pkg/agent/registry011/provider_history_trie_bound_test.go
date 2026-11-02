package registry011

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func TestProviderHistoryTrieUpperMatchesIndependentLiteralsAndSharedBytes(t *testing.T) {
	for _, tc := range []struct{ items, bytes, value, units uint64 }{
		{0, 0, 0, 4}, {1, 1, 1, 1705935}, {3, 100, 2, 5117908},
		{257, 65536, 256, 457420716}, {16384, 64 << 20, 64 << 20, 4503629858471940},
		{16, 784, 49, 27359412},
		{16384, 802816, 49, 28016033796},
	} {
		got, ok := providerHistoryContentTrieUpper(tc.items, tc.bytes, tc.value)
		if !ok || got != (candidateWorkVector{10: tc.units}) {
			t.Fatal("independent whole trie polynomial and byte-pool literal", tc, got, ok)
		}
	}
	// Derive the node coefficient independently from original source events.
	want := 256 + 6*532*532 + 7 + 360 + 126 + 66 + 119 + 6 + 6*17*33 + (16 + 6*532 + 7) + 7*33
	if want != 1705896 || providerHistoryTrieNodeLogical != want {
		t.Fatal("changed source stages require rederiving the coefficient")
	}
}

func TestProviderHistoryTrieUpperCoversOrdinaryRootsAndIndexWidths(t *testing.T) {
	for _, count := range []int{0, 1, 2, 17, 127, 128, 255, 256, 257} {
		values := make(providerHistoryContentReference, count)
		var total, maximum uint64
		for i := range values {
			length := []int{1, 31, 32, 56}[i%4]
			values[i] = bytes.Repeat([]byte{byte(i % 251)}, length)
			total += uint64(length)
			maximum = max(maximum, uint64(length))
		}
		upper, ok := providerHistoryContentTrieUpper(uint64(count), total, maximum)
		if !ok {
			t.Fatal("ordinary prospective trie shape")
		}
		m := providerHistoryContentUnitMeter(upper)
		got, err := providerHistoryContentRootObserved(context.Background(), values, m)
		want := types.DeriveSha(values, trie.NewStackTrie(nil))
		if err != nil || got != want || m.failed {
			t.Fatal("original root escaped its independently derived allowance", count, err, m.actual, upper)
		}
		for i, n := range m.actual {
			if n > upper[i] || i != 10 && n != 0 {
				t.Fatal("trie quote lost original work or invented other coordinates", m.actual, upper)
			}
		}
	}
}

func TestProviderHistoryTrieExactEmptyAndFinalHashShortageKeepTheOriginalPrefix(t *testing.T) {
	empty := providerHistoryContentUnitMeter(candidateWorkVector{10: 4})
	if root, err := providerHistoryContentRootObserved(context.Background(), nil, empty); err != nil || root != types.EmptyRootHash || empty.actual != (candidateWorkVector{10: 4}) {
		t.Fatal("empty encode and root hash each cost two", err, empty.actual)
	}
	// RLP(index0) path is2080; the leaf c482208001 has five bytes.
	// Key2 + path5 + encode26 + copies3/2 + final hash6 =44.
	wantRaw := []byte{0xc4, 0x82, 0x20, 0x80, 0x01}
	encoded, err := rlp.EncodeToBytes([]any{[]byte{0x20, 0x80}, []byte{1}})
	if err != nil || !bytes.Equal(encoded, wantRaw) {
		t.Fatal("independent ordinary leaf encoding", err, encoded)
	}
	for _, short := range []bool{false, true} {
		units := uint64(44)
		if short {
			units--
		}
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: units})
		root, err := providerHistoryContentRootObserved(context.Background(), [][]byte{{1}}, m)
		if !short {
			want := types.DeriveSha(providerHistoryContentReference{{1}}, trie.NewStackTrie(nil))
			if err != nil || root != want || m.actual != (candidateWorkVector{10: 44}) {
				t.Fatal("independent exact single-leaf source events", err, m.actual)
			}
			continue
		}
		if err == nil || root != (common.Hash{}) || !m.failed || m.actual != (candidateWorkVector{10: 38}) {
			t.Fatal("last root hash shortage lost its entered prefix", err, m.actual)
		}
		before := m.actual
		if _, retry := providerHistoryContentRootObserved(context.Background(), [][]byte{{1}}, m); retry == nil || m.actual != before {
			t.Fatal("failed original trie observer resumed or refunded work")
		}
	}
}

func TestProviderHistoryTrieWithdrawalWidthUsesThePinnedCanonicalFields(t *testing.T) {
	withdrawal := &types.Withdrawal{Index: ^uint64(0), Validator: ^uint64(0), Amount: ^uint64(0), Address: common.Address{19: 1}}
	raw, err := rlp.EncodeToBytes(withdrawal)
	if err != nil || len(raw) != 49 {
		t.Fatal("three uint64 strings and one address must fit the independently derived49 bytes", err, len(raw))
	}
	values := make(providerHistoryContentReference, 16)
	for i := range values {
		values[i] = raw
	}
	upper, ok := providerHistoryContentTrieUpper(16, 784, 49)
	m := providerHistoryContentUnitMeter(upper)
	root, rootErr := providerHistoryContentRootObserved(context.Background(), values, m)
	if !ok || rootErr != nil || root != types.DeriveSha(values, trie.NewStackTrie(nil)) || m.failed || m.actual[10] > upper[10] {
		t.Fatal("ordinary canonical withdrawal roots exceed their separate row", rootErr, m.actual, upper)
	}
}

func TestProviderHistoryCollectionTrieUpperPreservesIndependentRootJourneys(t *testing.T) {
	for _, tc := range []struct {
		g, r, blocks, windows, evidence, txs, value uint64
		journeys, roots, withdrawals, pool          uint64
		contentUnits, withdrawalUnits, units        uint64
		encoding                                    string
	}{
		{1, 1, 1, 1, 1024, 1, 1024, 15, 75, 30, 1024, 176695140, 820782360, 997477500, EthereumJSONBlocks},
		{2, 3, 5, 2, 4096, 2, 1024, 29, 725, 290, 4096, 1743463412, 7934229480, 9677692892, EthereumJSONBlocks},
		{1, 32, 3, 3, 64 << 20, 16384, 8 << 20, 15, 225, 90, 64 << 20, 76004840356578180, 2462347080, 76004842818925260, EthereumJSONBlocks},
		{1, 1, 1, 1, 1024, 1, 1024, 15, 60, 15, 1024, 134855760, 420240506940, 420375362700, "debug-raw-rlp-v1"},
		{2, 3, 5, 2, 4096, 2, 1024, 29, 580, 145, 4096, 1294231952, 4062324900420, 4063619132372, "debug-raw-rlp-v1"},
	} {
		got, ok := providerHistoryCollectionTrieUpper(providerHistoryCollectionSemanticLimits{
			Groups: tc.g, Requests: tc.r, Blocks: tc.blocks, Windows: tc.windows,
		}, tc.evidence, tc.txs, tc.value, tc.encoding)
		if !ok || got.journeys != tc.journeys || got.contentRoots != tc.roots || got.withdrawalRoots != tc.withdrawals || got.windowBytes != tc.pool ||
			got.content != (candidateWorkVector{10: tc.contentUnits}) || got.withdrawals != (candidateWorkVector{10: tc.withdrawalUnits}) ||
			got.total != (candidateWorkVector{10: tc.units}) {
			t.Fatal("independent whole repeated root counts and pooled polynomial", got, ok)
		}
	}
}

func TestProviderHistoryTrieUpperInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, tc := range [][3]uint64{{16385, 1, 1}, {1, 64<<20 + 1, 1}, {1, 1, 64<<20 + 1}, {0, 1, 0}, {0, 0, 1}, {1, 0, 1}, {1, 1, 0}} {
		if got, ok := providerHistoryContentTrieUpper(tc[0], tc[1], tc[2]); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid root domain returned a partial quote", tc, got, ok)
		}
	}
	base := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "evidence-zero", "evidence-wide", "tx-zero", "tx-wide", "value-zero", "value-wide", "encoding"} {
		l, evidence, txs, value := base, uint64(1024), uint64(1), uint64(1)
		encoding := EthereumJSONBlocks
		switch mode {
		case "groups":
			l.Groups = 0
		case "requests":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "evidence-zero":
			evidence = 0
		case "evidence-wide":
			evidence = 64<<20 + 1
		case "tx-zero":
			txs = 0
		case "tx-wide":
			txs = 16385
		case "value-zero":
			value = 0
		case "value-wide":
			value = evidence + 1
		case "encoding":
			encoding = "unselected"
		}
		if got, ok := providerHistoryCollectionTrieUpper(l, evidence, txs, value, encoding); ok || got != (providerHistoryCollectionTrieCost{}) {
			t.Fatal("invalid collection root domain returned a partial quote", mode, got, ok)
		}
	}
}

func TestProviderHistoryCollectionTrieUpperLateOverflowReturnsWholeZero(t *testing.T) {
	// Counts and the disjoint withdrawal row remain finite; the successful
	// content-row polynomial exceeds uint64. Numeric validity is not a live
	// manifest, original reservation, physical RAM or operating qualification.
	independent, ok := new(big.Int).SetString("1891533717484781261595600000", 10)
	if !ok || independent.BitLen() <= 64 {
		t.Fatal("independent overflow precursor")
	}
	got, admitted := providerHistoryCollectionTrieUpper(providerHistoryCollectionSemanticLimits{
		Groups: 100000, Requests: 100000, Blocks: 100000, Windows: 100000,
	}, 64<<20, 16384, 64<<20, EthereumJSONBlocks)
	if admitted || got != (providerHistoryCollectionTrieCost{}) {
		t.Fatal("late logical overflow leaked finite counts or withdrawal work", got, admitted)
	}
}
