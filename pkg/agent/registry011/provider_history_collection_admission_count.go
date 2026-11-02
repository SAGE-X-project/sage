package registry011

// A finite scalar proposal for reserving a T subwindow, NOT a validated next
// table or a phase transition. It does not parse bytes, claim a meter/clock,
// mutate a cursor, allocate a request or authorize a write/network effect.
// The two fresh original passes must confirm the exact count after entry.
func (c *providerHistoryCollectionCursor) admissionCountLocked() (int, bool) {
	if c == nil || c.owner != c || c.selection != c.sealed || c.selection.input.windowBlocks == 0 || c.selection.input.windowBlocks > 4096 ||
		c.selection.input.contentBlocks == 0 || c.selection.input.contentBlocks > 10 || c.low > c.high || c.high-c.low >= c.selection.input.windowBlocks ||
		c.high > c.selection.selected.Number || c.low < c.selection.input.journal.Deployment.Number ||
		(c.mode != "ancestry" && c.mode != "content") {
		return 0, false
	}
	count := uint64(0)
	switch c.phase {
	case "open":
		count = 5 + uint64(len(c.selection.input.codes))
	case "ancestry":
		if c.mode != "ancestry" || c.next < c.low || c.next > c.high {
			return 0, false
		}
		count = min(uint64(providerHistoryGroupMembers), c.next-c.low+1)
	case "content":
		if c.mode != "content" || c.next < c.low || c.next > c.high {
			return 0, false
		}
		blocks := min(c.selection.input.contentBlocks, c.high-c.next+1)
		for i := uint64(0); i < blocks; i++ {
			number, ok := providerUpperSum(c.next, i)
			if !ok || number > c.selection.selected.Number {
				return 0, false
			}
			index := c.selection.selected.Number - number
			if index >= uint64(len(c.spine)) || c.spine[index].Number != number {
				return 0, false
			}
			count += 2
			if c.spine[index].Timestamp >= sepoliaAmsterdamTime {
				count++
			}
		}
	case "effects":
		if c.mode != "content" || c.effectNext < 0 || c.effectNext >= len(c.effects) {
			return 0, false
		}
		count = uint64(min(providerHistoryGroupMembers, len(c.effects)-c.effectNext))
	case "state", "grants":
		if c.mode != "content" {
			return 0, false
		}
		count = 1
	case "history":
		if c.mode != "content" || c.historyNext == 0 || c.historyNext > c.stopVersion {
			return 0, false
		}
		count = min(uint64(providerHistoryGroupMembers), c.stopVersion-c.historyNext+1)
	case "close":
		count = 7
	default:
		return 0, false
	}
	if count == 0 || count > providerHistoryGroupMembers {
		return 0, false
	}
	return int(count), true
}
