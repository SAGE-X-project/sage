package registry011

// This small closed encoder accepts only private derived trie-node fields.
// All lengths are computed before allocation. Inline child lists keep their
// RLP, while leaf values and32-byte child hashes are encoded as strings.
func (t *providerHistoryContentTrie) encode(items []providerHistoryTrieItem) ([]byte, error) {
	if len(items) != 2 && len(items) != 17 || t.ctx.Err() != nil {
		return nil, ErrReplayUnavailable
	}
	const maximum = (64 << 20) + 1024
	body := 0
	for _, item := range items {
		n := len(item.raw)
		if n > 64<<20 || item.inline && (n == 0 || n >= 32 || item.raw[0] < 0xc0) {
			return nil, ErrReplay
		}
		if !item.inline && (n != 1 || item.raw[0] >= 0x80) {
			n += providerHistoryRLPPrefixSize(n)
		}
		if n > maximum-body {
			return nil, ErrReplayUnavailable
		}
		body += n
	}
	size := body + providerHistoryRLPPrefixSize(body)
	if size > maximum || !t.meter.event(providerEncodeEvent, uint64(size)) {
		return nil, ErrReplayUnavailable
	}
	out := make([]byte, size)
	pos := providerHistoryRLPWritePrefix(out, 0, 0xc0, body)
	for _, item := range items {
		n := len(item.raw)
		if !item.inline && (n != 1 || item.raw[0] >= 0x80) {
			pos = providerHistoryRLPWritePrefix(out, pos, 0x80, n)
		}
		if n != 0 {
			if !t.meter.event(providerCopyEvent, uint64(n)) {
				return nil, ErrReplayUnavailable
			}
			pos += copy(out[pos:], item.raw)
		}
	}
	if pos != size {
		return nil, ErrReplay
	}
	return out, nil
}

func providerHistoryRLPPrefixSize(size int) int {
	if size <= 55 {
		return 1
	}
	n := 1
	for size != 0 {
		n++
		size >>= 8
	}
	return n
}

func providerHistoryRLPWritePrefix(out []byte, pos int, base byte, size int) int {
	if size <= 55 {
		out[pos] = base + byte(size)
		return pos + 1
	}
	width := providerHistoryRLPPrefixSize(size) - 1
	out[pos] = base + 55 + byte(width)
	for i := width; i > 0; i-- {
		out[pos+i] = byte(size)
		size >>= 8
	}
	return pos + width + 1
}
