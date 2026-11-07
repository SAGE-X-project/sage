package registry011

// Failure-record BYTE events, excluding the result contexts already owned by
// the record-control component. A constructor validates one such context; a
// parser validates two because it rebuilds the record. The writer and its two
// readers therefore retain five existing contexts, not five new charges here.
// Native CPU, backing/RAM, outer objects, I/O and complete S remain separate.
type providerHistoryFailureBodyCost struct {
	minimumObjectBytes           uint64
	built, parsed, failedClosure candidateWorkVector
}

// Conditional DATA for the original failed-S body. requestBytes bounds the
// same valid T/C request; wireBytes and maximum are its original selected caps.
// The parser row includes ONE caller text-to-byte copy bounded by maximum.
// Earlier unverified reads/copies and arbitrary damaged private fields are not
// qualified by these scalars. No source witness, clock or allowance is created.
func providerHistoryFailureBodyUpper(requestBytes, wireBytes, maximum uint64, certificates providerTLSBounds) (providerHistoryFailureBodyCost, bool) {
	var empty providerHistoryFailureBodyCost
	if requestBytes == 0 || requestBytes > 16<<10 || requestBytes >= wireBytes || wireBytes > 16<<20 ||
		maximum == 0 || maximum > providerHistoryOriginalPackBytes || !certificates.valid() {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const header uint64 = providerHistoryFailureHeaderBytes
	magic, domain := uint64(len(providerHistoryFailureMagic)), uint64(len(providerHistoryFailureDomain))
	// Preserve failureOriginalFitsLocked's pre-dial funding of the rejected
	// extra wire byte and the FULL fixed header, even for a dial failure.
	minimum := a.sum(magic, 4, header, wireBytes, 1)
	if a.failed || minimum > maximum {
		return empty, false
	}
	count := a.product(uint64(certificates.chains), uint64(certificates.certificates))
	quadratic := a.product(header, header)
	var out providerHistoryFailureBodyCost
	out.minimumObjectBytes = minimum
	// Request/TLS match 3R+4; copies/hashes of the written and read prefixes
	// 4w+3v, where w<=R and v<=W+1; header copies2H; raw copy/hash2M;
	// magic/domain and all event sentinels remain explicit. No HTTP parse
	// or successful result-pack operation is hidden inside this body row.
	out.built[10] = a.sum(quadratic, a.product(7, requestBytes), a.product(3, a.sum(wireBytes, 1)),
		a.product(2, header), a.product(2, maximum), magic, domain, 35, a.product(67, count))
	// Caller M+1, magic3g+2, decode4=17, JSON H^2+1, prefix3R+2,
	// request copies R+2, digest compare133, raw rebuild comparison3M+2.
	// The rebuilt constructor body is one actual additional invocation.
	out.parsed[2] = a.sum(header, 4)
	out.parsed[10] = a.sum(out.built[10], quadratic, a.product(4, maximum), a.product(4, requestBytes), a.product(3, magic), 160)
	for coordinate := range out.failedClosure {
		out.failedClosure[coordinate] = a.sum(out.built[coordinate], a.product(2, out.parsed[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
