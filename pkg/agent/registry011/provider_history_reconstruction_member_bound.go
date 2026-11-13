package registry011

// Both fresh original readers, not the producer's repeated prefix readers.
// Each per-call child keeps its FULL original byte/codec envelope. These
// numbers therefore cover entered guards too, without assuming small successful
// JSON or subtracting another child's unused allowance. I/O, generic S/ledger,
// content boundaries, final joining, native work and physical RAM are separate.
type providerHistoryReconstructionMemberCost struct {
	packParses, settled, completed, attempts, failureParses, contexts uint64
	failureMinimum                                                    uint64
	packs, context, tls, response, custody, policy, failure, tail     candidateWorkVector
	normal, incoming, envelope                                        candidateWorkVector
}

func providerHistoryReconstructionMembersUpper(groups, requests, metadata, maximum, policyBytes uint64, capture ProviderCaptureLimits, methods providerMethodBounds, certificates providerTLSBounds, failedInspection bool) (providerHistoryReconstructionMemberCost, bool) {
	var empty providerHistoryReconstructionMemberCost
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok || metadata == 0 || metadata > 1<<20 ||
		policyBytes == 0 || policyBytes > 64<<10 || !methods.fits(capture) {
		return empty, false
	}
	pack, packOK := providerHistoryOriginalPackControlUpper(2*providerHistoryGroupMembers, maximum)
	record, recordOK := providerHistoryRecordControlUpper(providerHistoryGroupMembers)
	tls, tlsOK := providerHistoryTLSControlUpper(providerHistoryGroupMembers, certificates)
	response, responseOK := providerHistoryResponseControlUpper(capture)
	if !packOK || !recordOK || !tlsOK || !responseOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	members, attempt, failed := min(requests, uint64(providerHistoryGroupMembers)), uint64(0), uint64(0)
	if failedInspection {
		attempt, failed = 1, 1
	}
	out := providerHistoryReconstructionMemberCost{
		packParses: a.product(2, groups), settled: a.product(2, requests), completed: a.product(2, members),
		attempts: a.product(2, attempt), failureParses: a.product(2, failed),
		contexts: a.product(2, a.sum(groups, members, attempt, a.product(2, failed))),
	}
	parses := a.sum(out.settled, out.completed, out.attempts)
	for coordinate := range out.normal {
		out.packs[coordinate] = a.product(out.packParses, pack.parsed[coordinate])
		out.context[coordinate] = a.product(out.contexts, record.context[coordinate])
		// parseResultPack shares only its just-checked T/C context. Completed
		// and unfinished standalone members need a fresh context EACH. Keep
		// those above, and remove only the same context from the TLS child.
		suffix := a.difference(tls.parsed[coordinate], record.context[coordinate])
		out.tls[coordinate] = a.product(parses, suffix)
		out.response[coordinate] = a.product(parses, response.member[coordinate])
	}
	// The opaque pack parser checks providerHistoryPackOriginalSize for every
	// TLS entry BEFORE its string reaches the TLS caller copy. Thus the old
	// 64KiB TLS envelope applies to that copy even for a P-sized result pack.
	// Independent spool input has the same smaller P/64KiB cap; its reads are
	// already in reconstructionIO, not a second network/file charge here.
	wire, diagnostic := uint64(capture.WireBytes), min(maximum, uint64(64<<10))
	// Seven original request fields and the closed class (at most9 bytes).
	requestEqual := a.sum(2*32+1, 2*8192+1, 2*16384+1, 2*512+1, 2*16384+1, 2*133, 2*9+1)
	// Settled: spool hashes, capture hash comparisons, verbatim pack copies
	// and comparisons, result equality. Completed: class/request match,
	// spool hashes, caller byte-to-text copies, result equality and clone.
	settled := a.sum(a.product(4, a.sum(wire, diagnostic)), 405)
	completed := a.sum(requestEqual, a.product(2, a.sum(wire, diagnostic)), 470)
	// A whole unfinished pair needs caller text copies and its result clone.
	// The alternative response-only HTTP parser is componentwise <= the
	// assigned full response.member row, with TLS/context unused, not borrowed.
	unfinished := a.sum(requestEqual, wire, diagnostic, 69)
	out.custody[10] = a.sum(a.product(out.settled, settled), a.product(out.completed, completed),
		a.product(out.attempts, unfinished), a.product(2, a.sum(requests, groups, members, 2)))
	// One policy per S, one current completed prefix, one attempted capture.
	// The completed allowance remains even when that current prefix is empty.
	policies := a.product(2, a.sum(groups, 1, attempt))
	out.policy[2] = a.product(policies, policyBytes)
	out.policy[10] = a.product(policies, a.sum(a.product(policyBytes, policyBytes), policyBytes, 2))
	// Even an empty tail creates and seals two fixed18-slot DATA copies/pass.
	out.tail[10] = 2 * (2 * (providerHistoryRecoveryTailInputs + 2))
	if failedInspection {
		// Failure admission checks the actual selected method wire, not an
		// unrelated larger global capture ceiling. Never shrink a method cap.
		failureWire := uint64(max(methods.scalar.wire, methods.header.wire, methods.code.wire, methods.state.wire,
			methods.block.wire, methods.receipts.wire, methods.gas.wire, methods.execution.wire))
		body, ok := providerHistoryFailureBodyUpper(min(uint64(16<<10), failureWire-1), failureWire, maximum, certificates)
		if !ok {
			return empty, false // Preserve the original full-header+wire+1 funding.
		}
		out.failureMinimum = body.minimumObjectBytes
		for coordinate := range out.failure {
			out.failure[coordinate] = a.product(out.failureParses, body.parsed[coordinate])
		}
		// The failure parser's two contexts are already in out.contexts.
		out.failure[10] = a.sum(out.failure[10], a.product(out.failureParses, 2*(16<<10)+1))
		domain := uint64(max(len(providerHistoryOriginalPackDomain), len(providerHistoryGroupTicketDomain),
			len(providerHistoryGroupConsumptionDomain), len(providerHistoryGroupSettlementDomain), len(providerHistoryFailureDomain)))
		// All8 pending object hashes/equalities,7 control hashes/raw-prefix
		// copies/equalities,2 attempted spool hashes/equalities and17 appended
		// entries. Missing/partial states still visit their own planned slots.
		pending := a.sum(a.product(8, a.sum(maximum, domain, 135)), a.product(7, a.sum(a.product(4, metadata), 3)),
			wire, diagnostic, 268, 2*providerHistoryRecoveryTailInputs)
		out.tail[10] = a.sum(out.tail[10], a.product(2, pending))
	}
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.packs[coordinate], out.context[coordinate], out.tls[coordinate], out.response[coordinate],
			out.custody[coordinate], out.policy[coordinate], out.failure[coordinate], out.tail[coordinate])
		// All children above are full per-call envelopes, not planned smaller
		// successful rows. No second outstanding primitive is added to them.
		out.envelope[coordinate] = out.normal[coordinate]
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
