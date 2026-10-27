package registry011

import "strconv"

const (
	providerScanCommandV2 = "sage-provider-scan-command-v2"
	providerScanChunkV2   = "sage-provider-scan-chunk-v2"
)

// This live, nonserializable owner retains pre-I inspection DATA only.
// Neither its bytes nor its terminal cursor restores a scan or an ACK.
type providerScanReceipts struct {
	body                     *providerScanBody
	head, highwater, attempt string
	commands, chunks         []string
}

func (*providerScanReceipts) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderScanReceipts(b *providerScanBody, x *providerScanCodec) (*providerScanReceipts, error) {
	if !b.valid() || b.receipts != nil || b.control.prepared == nil || b.control.store.config.bounds.firstAdmission == nil || x == nil {
		return nil, errProviderIntegration
	}
	c := b.control
	highwater, a := x.highwater(c.pin)
	head, d := x.head(c.pin)
	attempt, e := x.attempt(*c.prepared)
	r := &providerScanReceipts{body: b, head: x.hash(providerScanHeadDomain, head), highwater: x.hash(providerScanHighwaterDomain, highwater), attempt: x.hash(providerScanIntentDomain, attempt)}
	if a != nil || d != nil || e != nil || !lowerHash(r.head) || !lowerHash(r.highwater) || !lowerHash(r.attempt) || r.attempt != c.pin.intent {
		return nil, errProviderIntegration
	}
	return r, nil
}

func (r *providerScanReceipts) valid() bool {
	return r != nil && r.body != nil && r.body.valid() && r.body.receipts == r
}

func (r *providerScanReceipts) bind(m map[string]any, format string) {
	m["format"], m["scan_head"], m["scan_highwater"], m["scan_attempt"] = format, r.head, r.highwater, r.attempt
}

func providerScanReceiptName(kind, digest string, meter *providerWorkMeter) string {
	if (kind != "command" && kind != "chunk") || !lowerHash(digest) {
		return ""
	}
	return providerReplayJoin([]string{"scan-", kind, "-v2-", digest[2:]}, meter)
}

func (r *providerScanReceipts) command(x *providerScanCodec, target string, index uint64, previous string, cursor providerScanCursor) ([]byte, error) {
	raw, err := x.command(target, index, previous, cursor)
	if err != nil {
		return nil, errProviderIntegration
	}
	m, ok := x.decode(raw).(map[string]any)
	if !ok {
		return nil, errProviderIntegration
	}
	r.bind(m, providerScanCommandV2)
	return x.encode(m)
}

func (r *providerScanReceipts) decode(x *providerScanCodec, kind string, raw []byte) map[string]any {
	fields := []string{"format", "target", "index", "previous_chunk", "cursor_start", "scan_head", "scan_highwater", "scan_attempt"}
	switch kind {
	case "command":
		fields = append(fields, "maximum_records", "maximum_bytes")
	case "chunk":
		fields = append(fields, "command_digest", "cursor_end", "records", "objects", "bytes", "cumulative_records", "cumulative_objects", "cumulative_bytes", "replay_state_digest", "ticket_index_root", "operation_index_root", "segment_totals_root", "clock_start", "clock_end", "work", "cumulative_work")
	default:
		return nil
	}
	return contentObject(x.decode(raw), fields...)
}

// Only exact privately derived names are classified as control DATA. Every
// corresponding original still has to be written/read back before success.
func (r *providerScanReceipts) classify(name string) bool {
	b, s := r.body, r.body.control.store
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		key, ok := b.key(disk, name)
		if !ok || b.required[key] != 0 || uint64(len(b.required)) >= s.config.bounds.scan.maximumCatalogEntries || !candidateCapacity(0, b.requiredNameBytes, uint64(len(name)), s.config.bounds.scan.maximumCatalogNameBytes) {
			return false
		}
		b.required[key] = 2
		b.requiredNameBytes += uint64(len(name))
	}
	return true
}

func (r *providerScanReceipts) retain(x *providerScanCodec, kind, digest string, raw []byte) bool {
	if !r.valid() {
		return false
	}
	b, s := r.body, r.body.control.store
	name := providerScanReceiptName(kind, digest, b.meter)
	if name == "" || len(raw) == 0 || uint64(len(raw)) > s.config.bounds.maximumMetadataPerCommit || uint64(len(r.commands)) > s.config.bounds.scan.maximumChunks || kind == "command" && (len(r.commands) != len(r.chunks) || uint64(len(r.commands)) == s.config.bounds.scan.maximumChunks) || kind == "chunk" && len(r.commands) != len(r.chunks)+1 {
		return false
	}
	previousEncoding := b.encoding
	b.encoding = true
	defer func() { b.encoding = previousEncoding }()
	domain, index := providerScanCommandV2, len(r.commands)
	if kind == "chunk" {
		domain, index = providerScanChunkV2, len(r.chunks)
	}
	previous := ""
	if index != 0 {
		previous = r.chunks[index-1]
	}
	m := r.decode(x, kind, raw)
	canonical, err := x.encode(m)
	if m == nil || err != nil || !x.equal(raw, canonical) || x.hash(domain, raw) != digest || m["format"] != domain || m["scan_head"] != r.head || m["scan_highwater"] != r.highwater || m["scan_attempt"] != r.attempt || m["target"] != b.control.prepared.target || m["index"] != strconv.Itoa(index) || m["previous_chunk"] != providerNullableDigest(previous) || kind == "chunk" && m["command_digest"] != r.commands[index] || !r.classify(name) {
		return false
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if s.write(disk, name, raw) != nil || !b.control.readExact(disk, name, raw) {
			return false
		}
	}
	actual, ok := providerUpperSum(s.retainedRaw, uint64(2*len(raw)))
	if !ok || actual > s.stored {
		return false
	}
	s.retainedRaw = actual
	if kind == "command" {
		r.commands = append(r.commands, digest)
	} else {
		r.chunks = append(r.chunks, digest)
	}
	return true
}

// Re-read earlier originals under the same prepaid body before the next
// nonempty record. The selected H_k/attempt is never replaced by a later HEAD.
func (r *providerScanReceipts) verifyPrevious(x *providerScanCodec, target string, nextIndex uint64) bool {
	if !r.valid() {
		return false
	}
	b, s := r.body, r.body.control.store
	if !b.valid() || uint64(len(r.commands)) != nextIndex || len(r.chunks) != len(r.commands) {
		return false
	}
	previous := ""
	for n := uint64(0); n < nextIndex; n++ {
		for _, pair := range []struct{ kind, format, digest string }{{"command", providerScanCommandV2, r.commands[n]}, {"chunk", providerScanChunkV2, r.chunks[n]}} {
			name := providerScanReceiptName(pair.kind, pair.digest, b.meter)
			if name == "" {
				return false
			}
			var first []byte
			for _, disk := range []*retentionDisk{s.custody, s.evidence} {
				raw, err := s.read(disk, name, int64(s.config.bounds.maximumMetadataPerCommit))
				m := r.decode(x, pair.kind, raw)
				canonical, canonicalErr := x.encode(m)
				if err != nil || m == nil || canonicalErr != nil || !x.equal(raw, canonical) || x.hash(pair.format, raw) != pair.digest || m["format"] != pair.format || m["scan_head"] != r.head || m["scan_highwater"] != r.highwater || m["scan_attempt"] != r.attempt || m["target"] != target || m["index"] != strconv.FormatUint(n, 10) || m["previous_chunk"] != providerNullableDigest(previous) || pair.kind == "chunk" && m["command_digest"] != r.commands[n] || first != nil && !x.equal(first, raw) {
					return false
				}
				first = raw
			}
		}
		previous = r.chunks[n]
	}
	return true
}
