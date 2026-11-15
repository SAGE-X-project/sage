package registry011

import (
	"encoding/hex"
	"reflect"
	"strconv"
)

type providerHistorySemanticIndexEntry struct {
	key   string
	block uint64
	flag  bool
}

// Every root is reconstructed from the owned actual entries. Sorting is an
// observed, fixed-storage merge, not Go map iteration order or a cached root.
// The complete repeated-prefix/time/RAM envelope remains a separate gate.
func providerHistorySemanticIndexRootObserved(s replayStream, claims bool, maximum int, meter *providerWorkMeter) string {
	count, kind := len(s.created), "created"
	if claims {
		count, kind = len(s.claims), "claims"
	}
	if meter == nil || maximum < 1 || maximum > 1<<20 || count > maximum || !meter.event(providerCopyEvent, uint64(count)*2*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size())) {
		return ""
	}
	entries := make([]providerHistorySemanticIndexEntry, count)
	other := make([]providerHistorySemanticIndexEntry, count)
	pos := 0
	if claims {
		for id, value := range s.claims {
			actor, ok := replayStateTextObserved(id.actor, 42, meter)
			if !ok || !meter.event(providerEncodeEvent, 64) {
				return ""
			}
			key := providerReplayJoin([]string{actor, ":", hex.EncodeToString(id.digest[:])}, meter)
			if key == "" || !providerHistorySemanticIndexPutObserved(entries, pos, providerHistorySemanticIndexEntry{key, value.block, value.consumed}, meter) {
				return ""
			}
			pos++
		}
	} else {
		for agent, value := range s.created {
			key, ok := replayStateTextObserved(agent, 64, meter)
			if !ok || !providerHistorySemanticIndexPutObserved(entries, pos, providerHistorySemanticIndexEntry{key: key, flag: value}, meter) {
				return ""
			}
			pos++
		}
	}
	for width := 1; width < count; width *= 2 {
		for first := 0; first < count; first += 2 * width {
			middle, end := min(first+width, count), min(first+2*width, count)
			left, right := first, middle
			for dest := first; dest < end; dest++ {
				chooseLeft := right == end
				if left < middle && right < end {
					if !meter.event(providerCompareEvent, uint64(len(entries[left].key)+len(entries[right].key))) {
						return ""
					}
					chooseLeft = entries[left].key <= entries[right].key
				}
				source := right
				if chooseLeft {
					source = left
					left++
				} else {
					right++
				}
				if !providerHistorySemanticIndexPutObserved(other, dest, entries[source], meter) {
					return ""
				}
			}
		}
		if !meter.event(providerCopyEvent, 2*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())) {
			return ""
		}
		entries, other = other, entries
	}
	root := providerHistoryCollectionValueDigestObserved("semantic-index", map[string]any{"index": kind, "count": count}, meter)
	for i, entry := range entries {
		block, ok := providerHistorySemanticDecimalObserved(entry.block, meter)
		if root == "" || !ok || i > 0 && (!meter.event(providerCompareEvent, uint64(len(entries[i-1].key)+len(entry.key))) || entries[i-1].key >= entry.key) {
			return ""
		}
		root = providerHistoryCollectionValueDigestObserved("semantic-index-entry", map[string]any{"index": kind, "previous": root, "ordinal": i, "key": entry.key, "block": block, "flag": entry.flag}, meter)
	}
	return root
}

func providerHistorySemanticIndexPutObserved(out []providerHistorySemanticIndexEntry, i int, value providerHistorySemanticIndexEntry, meter *providerWorkMeter) bool {
	if meter == nil || i < 0 || i >= len(out) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size())) {
		return false
	}
	out[i] = value
	return true
}

func providerHistorySemanticDecimalObserved(value uint64, meter *providerWorkMeter) (string, bool) {
	if meter == nil || !meter.preflight(providerEncodeEvent, 20) {
		return "", false
	}
	text := strconv.FormatUint(value, 10)
	return text, meter.event(providerEncodeEvent, uint64(len(text)))
}
