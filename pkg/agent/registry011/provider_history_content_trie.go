package registry011

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// The observed reader uses the same Ethereum indexed Merkle Patricia
// commitment over the ALREADY canonical binary values. Each concrete key,
// node encoding, byte copy and Keccak call is observed where it executes.
// Wrapping DeriveSha or a StackTrie callback would miss internal encodings
// and inline nodes. The legacy nil-observer path keeps using DeriveSha.
// This private DATA computation supplies no admission or operating authority.
type providerHistoryContentTrie struct {
	ctx    context.Context
	values [][]byte
	meter  *providerWorkMeter
}

type providerHistoryTrieItem struct {
	raw    []byte
	inline bool // A child RLP list shorter than32 bytes, not an RLP string.
}

func providerHistoryContentRootObserved(ctx context.Context, values [][]byte, meter *providerWorkMeter) (common.Hash, error) {
	if ctx == nil || ctx.Err() != nil || meter == nil || len(values) > 16384 || !meter.preflight(providerHashEvent, 1) {
		return common.Hash{}, ErrReplayUnavailable
	}
	var size uint64
	for _, value := range values {
		if len(value) == 0 || !candidateCapacity(0, size, uint64(len(value)), 64<<20) {
			return common.Hash{}, ErrReplayUnavailable
		}
		size += uint64(len(value))
	}
	t := providerHistoryContentTrie{ctx: ctx, values: values, meter: meter}
	if len(values) == 0 {
		if !meter.event(providerEncodeEvent, 1) {
			return common.Hash{}, ErrReplayUnavailable
		}
		return t.hash([]byte{0x80}) // RLP empty string, not an empty list.
	}
	raw, err := t.node(0, len(values), 0)
	if err != nil {
		return common.Hash{}, err
	}
	return t.hash(raw) // The root is hashed even when its RLP is shorter than32.
}

func (t *providerHistoryContentTrie) hash(raw []byte) (common.Hash, error) {
	if t.ctx.Err() != nil || !t.meter.event(providerHashEvent, uint64(len(raw))) {
		return common.Hash{}, ErrReplayUnavailable
	}
	return crypto.Keccak256Hash(raw), nil
}

// Indexed-list keys have at most3 RLP bytes at the existing16384 item limit.
// Their lexical order is1..127,0,128..; this computes that order directly
// without a second retained/sorted key array or an unobserved sort callback.
func (t *providerHistoryContentTrie) key(rank int) ([3]byte, int, int, error) {
	var key [3]byte
	if rank < 0 || rank >= len(t.values) || t.ctx.Err() != nil {
		return key, 0, 0, ErrReplayUnavailable
	}
	first := min(127, len(t.values)-1)
	index := rank
	if rank < first {
		index = rank + 1
	} else if rank == first {
		index = 0
	}
	n := 1
	if index >= 256 {
		n = 3
	} else if index >= 128 {
		n = 2
	}
	if !t.meter.event(providerEncodeEvent, uint64(n)) {
		return key, 0, 0, ErrReplayUnavailable
	}
	switch n {
	case 1:
		key[0] = byte(index)
		if index == 0 {
			key[0] = 0x80
		}
	case 2:
		key[0], key[1] = 0x81, byte(index)
	case 3:
		key[0], key[1], key[2] = 0x82, byte(index>>8), byte(index)
	}
	return key, n, index, nil
}

func providerHistoryTrieNibble(key [3]byte, offset int) byte {
	if offset%2 == 0 {
		return key[offset/2] >> 4
	}
	return key[offset/2] & 15
}

func (t *providerHistoryContentTrie) node(begin, end, depth int) ([]byte, error) {
	if begin < 0 || begin >= end || end > len(t.values) || depth < 0 || depth > 6 || t.ctx.Err() != nil {
		return nil, ErrReplayUnavailable
	}
	first, firstBytes, index, err := t.key(begin)
	if err != nil || depth > 2*firstBytes {
		return nil, ErrReplayUnavailable
	}
	if end-begin == 1 {
		path, err := t.path(first, depth, 2*firstBytes-depth, true)
		if err != nil {
			return nil, err
		}
		items := [2]providerHistoryTrieItem{{raw: path}, {raw: t.values[index]}}
		return t.encode(items[:])
	}
	last, lastBytes, _, err := t.key(end - 1)
	if err != nil || depth >= min(2*firstBytes, 2*lastBytes) {
		return nil, ErrReplayUnavailable
	}
	shared := depth
	for shared < min(2*firstBytes, 2*lastBytes) {
		if !t.meter.event(providerCompareEvent, 2) {
			return nil, ErrReplayUnavailable
		}
		if providerHistoryTrieNibble(first, shared) != providerHistoryTrieNibble(last, shared) {
			break
		}
		shared++
	}
	if shared > depth {
		child, err := t.node(begin, end, shared)
		if err != nil {
			return nil, err
		}
		ref, err := t.reference(child)
		if err != nil {
			return nil, err
		}
		path, err := t.path(first, depth, shared-depth, false)
		if err != nil {
			return nil, err
		}
		items := [2]providerHistoryTrieItem{{raw: path}, ref}
		return t.encode(items[:])
	}
	var items [17]providerHistoryTrieItem
	cursor := begin
	for nibble := byte(0); nibble < 16; nibble++ {
		start := cursor
		for cursor < end {
			key, n, _, err := t.key(cursor)
			if err != nil || depth >= 2*n || !t.meter.event(providerCompareEvent, 2) {
				return nil, ErrReplayUnavailable
			}
			if providerHistoryTrieNibble(key, depth) != nibble {
				break
			}
			cursor++
		}
		if start == cursor {
			continue
		}
		child, err := t.node(start, cursor, depth+1)
		if err != nil {
			return nil, err
		}
		items[nibble], err = t.reference(child)
		if err != nil {
			return nil, err
		}
	}
	if cursor != end {
		return nil, ErrReplay
	}
	// RLP(index) keys are prefix-free, so branch value slot16 stays empty.
	return t.encode(items[:])
}

func (t *providerHistoryContentTrie) reference(raw []byte) (providerHistoryTrieItem, error) {
	if len(raw) < 32 {
		return providerHistoryTrieItem{raw: raw, inline: true}, nil
	}
	hash, err := t.hash(raw)
	if err != nil || !t.meter.event(providerCopyEvent, 32) {
		return providerHistoryTrieItem{}, ErrReplayUnavailable
	}
	ref := make([]byte, 32)
	copy(ref, hash[:])
	return providerHistoryTrieItem{raw: ref}, nil
}

func (t *providerHistoryContentTrie) path(key [3]byte, offset, count int, leaf bool) ([]byte, error) {
	if offset < 0 || count < 0 || offset+count > 6 {
		return nil, ErrReplay
	}
	n := count/2 + 1
	if !t.meter.event(providerEncodeEvent, uint64(n)) {
		return nil, ErrReplayUnavailable
	}
	out := make([]byte, n)
	if leaf {
		out[0] = 0x20
	}
	i := 0
	if count%2 != 0 {
		out[0] |= 0x10 | providerHistoryTrieNibble(key, offset)
		i = 1
	}
	for pos := 1; i < count; pos, i = pos+1, i+2 {
		out[pos] = providerHistoryTrieNibble(key, offset+i)<<4 | providerHistoryTrieNibble(key, offset+i+1)
	}
	return out, nil
}
