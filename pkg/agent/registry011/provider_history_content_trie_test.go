package registry011

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// These are finite in-memory commitment values, not transactions or replies
// sent to a peer. The independent reference uses the pinned Ethereum library.
type providerHistoryContentReference [][]byte

func (v providerHistoryContentReference) Len() int { return len(v) }
func (v providerHistoryContentReference) EncodeIndex(i int, out *bytes.Buffer) {
	out.Write(v[i])
}

func providerHistoryContentUnitMeter(upper candidateWorkVector) *providerWorkMeter {
	m := &providerWorkMeter{integration: &providerIntegration{}, upper: upper}
	m.owner = m
	return m // Inert arithmetic subject; no store, inspection or native owner.
}

func TestProviderHistoryContentTrieMatchesIndependentCommitment(t *testing.T) {
	for _, count := range []int{0, 1, 2, 15, 16, 17, 126, 127, 128, 129, 255, 256, 257, 4096, 16384} {
		t.Run(fmt.Sprintf("items-%d", count), func(t *testing.T) {
			values := make(providerHistoryContentReference, count)
			lengths := []int{1, 2, 29, 30, 31, 32, 55, 56, 255, 256}
			for i := range values {
				values[i] = bytes.Repeat([]byte{byte(i % 253)}, lengths[i%len(lengths)])
			}
			want := types.DeriveSha(values, trie.NewStackTrie(nil))
			m := providerReplayUnitMeter()
			got, err := providerHistoryContentRootObserved(context.Background(), values, m)
			if err != nil || got != want || m.failed || m.actual[10] == 0 {
				t.Fatal("independent indexed commitment", got, want, err, m.actual)
			}
			for i, n := range m.actual {
				if i != 10 && n != 0 {
					t.Fatal("trie processing invented requests, semantic objects or decoded bytes", m.actual)
				}
			}
		})
	}
}

func TestProviderHistoryContentTrieEmptyAndSingleValueHaveExplicitRoots(t *testing.T) {
	empty, err := providerHistoryContentRootObserved(context.Background(), nil, providerReplayUnitMeter())
	if err != nil || empty != types.EmptyRootHash {
		t.Fatal("empty list commitment is not Keccak(RLP empty string)", empty, err)
	}
	leaf, err := rlp.EncodeToBytes([]any{[]byte{0x20, 0x80}, []byte{1}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := providerHistoryContentRootObserved(context.Background(), [][]byte{{1}}, providerReplayUnitMeter())
	if err != nil || got != crypto.Keccak256Hash(leaf) {
		t.Fatal("root leaf was inlined or its RLP(index0) compact path changed", got, err)
	}
}

func TestProviderHistoryContentTrieOriginalMeterHasExactAndLateBoundaries(t *testing.T) {
	values := [][]byte{bytes.Repeat([]byte{1}, 56), {2}, bytes.Repeat([]byte{3}, 31)}
	probe := providerReplayUnitMeter()
	want, err := providerHistoryContentRootObserved(context.Background(), values, probe)
	if err != nil || probe.actual[10] <= 2 {
		t.Fatal("independent positive precursor", err)
	}
	upper := probe.actual
	exact := providerHistoryContentUnitMeter(upper)
	got, err := providerHistoryContentRootObserved(context.Background(), values, exact)
	if err != nil || got != want || exact.actual != probe.actual || exact.failed {
		t.Fatal("exact original allowance", got, err, exact.actual, probe.actual)
	}
	upper[10]--
	late := providerHistoryContentUnitMeter(upper)
	got, err = providerHistoryContentRootObserved(context.Background(), values, late)
	if err == nil || got != (common.Hash{}) || !late.failed || late.actual[10] == 0 || late.actual[10] >= probe.actual[10] {
		t.Fatal("last root hash was not refused with the prior work retained", got, err, late.actual)
	}
	before := late.actual
	if _, err := providerHistoryContentRootObserved(context.Background(), values, late); err == nil || late.actual != before {
		t.Fatal("failed original meter was reused or refunded")
	}
}

func TestProviderHistoryContentTrieRefusesUnavailableInputs(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		values [][]byte
		meter  *providerWorkMeter
	}{
		{"context", nil, nil, providerReplayUnitMeter()},
		{"cancelled", cancelled, [][]byte{{1}}, providerReplayUnitMeter()},
		{"observer", context.Background(), [][]byte{{1}}, nil},
		{"count", context.Background(), make([][]byte, 16385), providerReplayUnitMeter()},
		{"empty-value", context.Background(), [][]byte{{}}, providerReplayUnitMeter()},
		{"unfunded", context.Background(), [][]byte{{1}}, providerHistoryContentUnitMeter(candidateWorkVector{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := providerHistoryContentRootObserved(tc.ctx, tc.values, tc.meter)
			if err == nil || root != (common.Hash{}) {
				t.Fatal("unavailable input acquired a root", root, err)
			}
		})
	}
	// Reusing a finite1MiB DATA value checks aggregate limits without creating
	// a65MiB payload or any network traffic.
	value := bytes.Repeat([]byte{1}, 1<<20)
	values := make([][]byte, 65)
	for i := range values {
		values[i] = value
	}
	m := providerReplayUnitMeter()
	if root, err := providerHistoryContentRootObserved(context.Background(), values, m); err == nil || root != (common.Hash{}) || m.actual != (candidateWorkVector{}) {
		t.Fatal("aggregate evidence limit was not checked before construction", err, m.actual)
	}
}

func TestProviderHistoryContentTrieClosedNodeEncodingMatchesRLP(t *testing.T) {
	for _, length := range []int{0, 1, 31, 32, 54, 55, 56, 255, 256, 65535, 65536} {
		t.Run(fmt.Sprintf("value-bytes-%d", length), func(t *testing.T) {
			value := bytes.Repeat([]byte{0xa5}, length)
			items := [2]providerHistoryTrieItem{{raw: []byte{0x20}}, {raw: value}}
			m := providerReplayUnitMeter()
			tr := providerHistoryContentTrie{ctx: context.Background(), meter: m}
			got, err := tr.encode(items[:])
			want, referenceErr := rlp.EncodeToBytes([][]byte{{0x20}, value})
			if err != nil || referenceErr != nil || !bytes.Equal(got, want) || m.actual[2] != 0 {
				t.Fatal("closed node length/copy encoding", err, referenceErr)
			}
		})
	}
	inline, _ := rlp.EncodeToBytes([][]byte{{0x20}, {1}})
	items := [17]providerHistoryTrieItem{}
	items[3] = providerHistoryTrieItem{raw: inline, inline: true}
	reference := make([]any, 17)
	for i := range reference {
		reference[i] = []byte{}
	}
	reference[3] = rlp.RawValue(inline)
	tr := providerHistoryContentTrie{ctx: context.Background(), meter: providerReplayUnitMeter()}
	got, err := tr.encode(items[:])
	want, referenceErr := rlp.EncodeToBytes(reference)
	if err != nil || referenceErr != nil || !bytes.Equal(got, want) {
		t.Fatal("inline branch child became an RLP string", err, referenceErr)
	}
}

func TestProviderHistoryContentTrieEverySmallOriginalAllowanceRefusesPartialRoots(t *testing.T) {
	// Exhaust the small finite arithmetic subject, including shortages after
	// leaf construction, inline-child copies and immediately before root hash.
	// Each attempt gets its own preselected original meter; none is replenished.
	values := [][]byte{{1}, {2}, {3}}
	probe := providerReplayUnitMeter()
	if _, err := providerHistoryContentRootObserved(context.Background(), values, probe); err != nil || probe.actual[10] > 10000 {
		t.Fatal("finite positive precursor", err, probe.actual)
	}
	for maximum := uint64(0); maximum < probe.actual[10]; maximum++ {
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: maximum})
		if root, err := providerHistoryContentRootObserved(context.Background(), values, m); err == nil || root != (common.Hash{}) || !m.failed || m.actual[10] > maximum {
			t.Fatal("partial root escaped a primitive allowance", maximum, root, err, m.actual)
		}
	}
}

func TestProviderHistoryContentTriePrivateBoundsRemainClosed(t *testing.T) {
	tr := providerHistoryContentTrie{ctx: context.Background(), values: [][]byte{{1}}, meter: providerReplayUnitMeter()}
	for _, rank := range []int{-1, 1} {
		if _, _, _, err := tr.key(rank); err == nil {
			t.Fatal("rank outside the selected complete list", rank)
		}
	}
	for _, bounds := range [][3]int{{-1, 1, 0}, {0, 0, 0}, {0, 2, 0}, {0, 1, -1}, {0, 1, 7}, {0, 1, 3}} {
		if _, err := tr.node(bounds[0], bounds[1], bounds[2]); err == nil {
			t.Fatal("unavailable selected node range", bounds)
		}
	}
	for _, bounds := range [][2]int{{-1, 1}, {0, -1}, {0, 7}} {
		if _, err := tr.path([3]byte{}, bounds[0], bounds[1], true); err == nil {
			t.Fatal("compact path escaped the indexed key", bounds)
		}
	}
	for _, items := range [][]providerHistoryTrieItem{
		nil,
		{{raw: []byte{1}}},
		{{inline: true}, {}},
		{{inline: true, raw: []byte{1}}, {}},
		{{inline: true, raw: bytes.Repeat([]byte{0xc0}, 32)}, {}},
	} {
		if _, err := tr.encode(items); err == nil {
			t.Fatal("a non-node or unavailable inline list was encoded")
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	stopped := providerHistoryContentTrie{ctx: cancelled, values: [][]byte{{1}}, meter: providerReplayUnitMeter()}
	if _, err := stopped.hash([]byte{1}); err == nil {
		t.Fatal("cancelled trie hashed a value")
	}
	if _, _, _, err := stopped.key(0); err == nil {
		t.Fatal("cancelled trie constructed a key")
	}
	if _, err := stopped.node(0, 1, 0); err == nil {
		t.Fatal("cancelled trie constructed a node")
	}
	if _, err := stopped.encode(make([]providerHistoryTrieItem, 2)); err == nil {
		t.Fatal("cancelled trie encoded a node")
	}
}
