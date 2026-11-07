package registry011

import "strconv"

// Scalar DATA for ONE original-window trace-normalization phase. RawBytes is
// the sum of complete original raw result lengths, not semantic payload bytes.
// Attempts includes a possible final raw preflight after frame capacity is
// exhausted. These values do not authenticate their source or grant dispatch.
// The enclosing collector stops at its first decoder error; repeated direct
// DATA retries do not establish the Attempts bound.
type providerHistoryTraceNormalizationLimits struct {
	BodyBytes, RawBytes, Attempts, Frames, Logs uint64
}

// RefusalUpper includes raw validation, every entered frame/log prefix and
// declared but unvisited child storage. It requires the original strict Unicode
// result route, distinct raw field/array spans and the collector's shared-window
// normalization budget. Arbitrary private maps or fresh per-trace budgets do
// not establish these premises.
//
// Frame depth0..32 gives sum(frame raw lengths)<=33R and sum(squares)<=33CR;
// nonrecursive log raw spans give <=R and <=CR. Declared child slots can exceed
// visited children on refusal: their allocation is bounded by19w*(33R/2),
// rounded upward to314wR. Each visited frame and reserved log also needs one
// distinct nonempty raw value, so their counts are <=min(original cap,R).
//
// This is a logical whole-normalization E, not an add-on to an overlapping
// successful normalization row. Both receipt decoders, semantic frame walking,
// outer original-string copies, native CPU/backing/peak/lifetime RAM, resources,
// caller multiplicities, factory and operating authority remain separate.
func providerHistoryTraceNormalizationRefusalUpper(p providerHistoryTraceNormalizationLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.Frames == 0 || p.Frames > 65536 ||
		p.Logs == 0 || p.Logs > 65536 || p.Attempts > p.Frames+1 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	maximumRaw := a.product(p.BodyBytes, p.Attempts)
	if a.failed || p.RawBytes > maximumRaw || p.RawBytes < p.Attempts {
		return empty, false
	}
	if p.Attempts == 0 {
		return empty, true
	}
	f, logs := min(p.Frames, p.RawBytes), min(p.Logs, p.RawBytes)
	w := uint64(strconv.IntSize / 8)
	var out candidateWorkVector
	out[2] = a.sum(a.product(138, p.RawBytes), a.product(64, f), a.product(256, logs))
	out[5], out[6] = f, logs
	perRaw := a.sum(a.product(138, p.BodyBytes), a.product(413, w), 803)
	perFrame := a.sum(13062, a.product(19, w))
	perLog := a.sum(18167, a.product(42, w))
	out[10] = a.sum(a.product(perRaw, p.RawBytes), a.product(perFrame, f),
		a.product(perLog, logs), a.product(4, p.Attempts))
	if a.failed {
		return empty, false
	}
	return out, true
}
