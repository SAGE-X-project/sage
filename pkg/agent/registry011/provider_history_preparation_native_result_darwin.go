//go:build darwin

package registry011

import "encoding/binary"

// This fixed record carries only the diagnostic fields actually consumed by
// FinishDiagnostic. It is not a native image of the entire frozen source plan.
const nativeHistoryResultMagic = "SAGEHPR1"
const nativeHistoryInputDigestBytes = 64
const nativeHistoryRegistryDigestBytes = 66
const nativeHistoryResultBytes = 8 + nativeHistoryInputDigestBytes + 6*nativeHistoryRegistryDigestBytes + 11*8 + 4*8

func (n *nativeHistoryPreparation) storeResult(s *HistoryPreparationSession, plan *providerHistorySourcePlan, inputDigest string) error {
	if plan == nil || plan.clock == nil || plan.config.bounds.history == nil || plan.config.bounds.history.collection == nil || n == nil || s == nil || s.compiler == nil || n.owner != s || n.compiler != s.compiler {
		return ErrConfiguration
	}
	var record [nativeHistoryResultBytes]byte
	copy(record[:8], nativeHistoryResultMagic)
	identity := plan.config.identity
	// The input SHA is plain hex; Registry identities retain their original 0x
	// prefix. Keep both wire contracts intact rather than truncating either.
	if len(inputDigest) != nativeHistoryInputDigestBytes || !lowerHash("0x"+inputDigest) {
		return ErrConfiguration
	}
	copy(record[8:8+nativeHistoryInputDigestBytes], inputDigest)
	hashes := [6]string{identity.Policy, identity.Instance, identity.Provider, identity.Namespace, identity.SourceID, plan.config.bounds.history.collection.digest}
	offset := 8 + nativeHistoryInputDigestBytes
	for _, value := range hashes {
		if !lowerHash(value) {
			return ErrConfiguration
		}
		copy(record[offset:offset+nativeHistoryRegistryDigestBytes], value)
		offset += nativeHistoryRegistryDigestBytes
	}
	binary.LittleEndian.PutUint64(record[offset:], identity.Generation)
	offset += 8
	for _, value := range plan.source.assigned.work {
		binary.LittleEndian.PutUint64(record[offset:], value)
		offset += 8
	}
	binary.LittleEndian.PutUint64(record[offset:], plan.source.assigned.clocks)
	binary.LittleEndian.PutUint64(record[offset+8:], uint64(s.compiler.origin.UnixNano()))
	binary.LittleEndian.PutUint64(record[offset+16:], uint64(plan.compilerDeadline.UnixNano()))
	if err := n.acquireOwner(s, &n.result, len(record)); err != nil {
		return err
	}
	span, err := n.result.region.Store(record[:])
	if err != nil {
		return err
	}
	if err := n.result.region.Seal(); err != nil {
		return err
	}
	n.resultSpan, n.resultPlan = span, plan
	return nil
}

func (n *nativeHistoryPreparation) readResultLocked(s *HistoryPreparationSession) (HistoryPreparationSummary, error) {
	var empty HistoryPreparationSummary
	if n == nil || s == nil || s.compiler == nil || s.plan == nil || n.owner != s || n.compiler != s.compiler || n.resultPlan != s.plan || n.result.region == nil {
		return empty, ErrConfiguration
	}
	var record [nativeHistoryResultBytes]byte
	if err := n.result.region.Read(n.resultSpan, record[:]); err != nil {
		return empty, err
	}
	if string(record[:8]) != nativeHistoryResultMagic {
		return empty, ErrConfiguration
	}
	output := HistoryPreparationSummary{Format: historyPreparationResultFormat, SourceReady: false}
	output.InputSHA256 = string(record[8 : 8+nativeHistoryInputDigestBytes])
	if !lowerHash("0x" + output.InputSHA256) {
		return empty, ErrConfiguration
	}
	destinations := [6]*string{&output.Identity.Policy, &output.Identity.Instance, &output.Identity.Provider, &output.Identity.Namespace, &output.Identity.SourceID, &output.ManifestSHA256}
	offset := 8 + nativeHistoryInputDigestBytes
	for _, destination := range destinations {
		value := string(record[offset : offset+nativeHistoryRegistryDigestBytes])
		if !lowerHash(value) {
			return empty, ErrConfiguration
		}
		*destination = value
		offset += nativeHistoryRegistryDigestBytes
	}
	output.Identity.Generation = binary.LittleEndian.Uint64(record[offset:])
	offset += 8
	for index := range output.AssignedWork {
		output.AssignedWork[index] = binary.LittleEndian.Uint64(record[offset:])
		offset += 8
	}
	output.AssignedClockReads = binary.LittleEndian.Uint64(record[offset:])
	output.OriginUnixNano = int64(binary.LittleEndian.Uint64(record[offset+8:]))
	output.DeadlineUnixNano = int64(binary.LittleEndian.Uint64(record[offset+16:]))
	return output, nil
}
