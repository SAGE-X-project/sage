package registry011

import "strconv"

// A frozen target names only already committed data. It does not authenticate
// a current tip, select a retaining principal or grant an inspection/dispatch.
type providerScanTarget struct {
	setup, custody, frontier, objects, enrollment, admission, budget, checkpoint string
}

func (x *providerScanCodec) target(t providerScanTarget) ([]byte, error) {
	if !lowerHash(t.setup) || !lowerHash(t.custody) || !lowerHash(t.objects) || t.enrollment != "" && !lowerHash(t.enrollment) || t.frontier != "" && !lowerHash(t.frontier) || (t.enrollment == "") != (t.frontier == "") || (t.admission == "") != (t.budget == "") || t.admission != "" && (!lowerHash(t.admission) || !lowerHash(t.budget) || t.enrollment == "") || t.checkpoint != "" && (!lowerHash(t.checkpoint) || t.admission == "") {
		return nil, errProviderIntegration
	}
	m := x.identity("sage-provider-scan-target-v1")
	m["setup_head"], m["custody_commit"], m["objects_root"] = t.setup, t.custody, t.objects
	m["frontier"], m["enrollment_root"], m["admission_digest"], m["budget_root"], m["checkpoint"] = providerNullableDigest(t.frontier), providerNullableDigest(t.enrollment), providerNullableDigest(t.admission), providerNullableDigest(t.budget), providerNullableDigest(t.checkpoint)
	return x.encode(m)
}

func (x *providerScanCodec) parseTarget(raw []byte) (providerScanTarget, error) {
	m := contentObject(x.decode(raw), "format", "namespace", "instance_key", "generation", "fence", "setup_head", "custody_commit", "frontier", "objects_root", "enrollment_root", "admission_digest", "budget_root", "checkpoint", "profile")
	if m == nil {
		return providerScanTarget{}, errProviderIntegration
	}
	for _, key := range []string{"frontier", "enrollment_root", "admission_digest", "budget_root", "checkpoint"} {
		if !providerIndexNullableDigest(m[key]) {
			return providerScanTarget{}, errProviderIntegration
		}
	}
	t := providerScanTarget{contentString(m["setup_head"]), contentString(m["custody_commit"]), contentString(m["frontier"]), contentString(m["objects_root"]), contentString(m["enrollment_root"]), contentString(m["admission_digest"]), contentString(m["budget_root"]), contentString(m["checkpoint"])}
	want, err := x.target(t)
	if err != nil || !x.equal(raw, want) {
		return providerScanTarget{}, errProviderIntegration
	}
	return t, nil
}

type providerScanCursor struct {
	phase string
	next  uint64
}

func (p providerScanCursor) object() map[string]string {
	if p.phase != "setup" && p.phase != "main" && p.phase != "done" || p.phase == "done" && p.next != 0 || p.phase == "main" && p.next == 0 {
		return nil
	}
	return map[string]string{"phase": p.phase, "next_sequence": strconv.FormatUint(p.next, 10)}
}

func (x *providerScanCodec) command(target string, index uint64, previous string, cursor providerScanCursor) ([]byte, error) {
	if !lowerHash(target) || index >= x.config.bounds.scan.maximumChunks || (index == 0) != (previous == "") || previous != "" && !lowerHash(previous) || cursor.phase == "done" || cursor.object() == nil || index == 0 && cursor != (providerScanCursor{"setup", 0}) {
		return nil, errProviderIntegration
	}
	b := x.config.bounds.scan
	return x.encode(map[string]any{"format": "sage-provider-scan-command-v1", "target": target, "index": strconv.FormatUint(index, 10), "previous_chunk": providerNullableDigest(previous), "cursor_start": cursor.object(), "maximum_records": strconv.FormatUint(b.chunkRecords, 10), "maximum_bytes": strconv.FormatUint(b.chunkRawBytes, 10)})
}
