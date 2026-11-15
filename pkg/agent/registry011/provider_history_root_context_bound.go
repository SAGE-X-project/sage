package registry011

// Scalar DATA for ONE nodeRootContextObserved invocation on an original strict
// JSON root. The ten scalar RawMessage spans must be distinct parts of at most
// bodyBytes original bytes. transactionBytes is the ceiling selected by the
// original native-input owner, not a credential supplied by a caller. Inert
// DATA and old roles keep64MiB; a collection owner selects capture+1024.
//
// The scalar JSON/inner pools contribute decoded2C+64 and logical2C²+2C+5320.
// Sender recovery plus its continued address conversion contributes decoded97
// and logicalM²+15557, with M=H+256. Sender failure does not skip that address
// attempt. The CREATE destination branch retains all three calls before its
// combined guard, including a bad destination scalar. Its maximum4183, the
// destination address1765 and three binding comparisons give the row below.
// Initial seven-call, sender, destination and reservoir incoming preflights
// are included; an unconsumed preflight is not added again as actual work.
//
// This replaces an overlapping root-context prefix/envelope in a matched
// composition. It does not include outer object parsing, dimensions, execution
// header/receipt binding, role repetitions, native CPU/accessor allocations,
// backing/peak/lifetime RAM, production factories or operating authority.
func providerHistoryRootContextRefusalUpper(bodyBytes, transactionBytes uint64) (candidateWorkVector, bool) {
	if bodyBytes < 4 || bodyBytes > 8<<20 || transactionBytes == 0 || transactionBytes > 64<<20 {
		return candidateWorkVector{}, false
	}
	// The largest accepted row is4644371584608544, below2^53. Validation
	// precedes H+256 and every product, including on32-bit Go systems.
	c, h, m := bodyBytes, transactionBytes, transactionBytes+256
	return candidateWorkVector{2: 2*c + 161, 10: 2*c*c + 5*c + m*m + h + 26912}, true
}
