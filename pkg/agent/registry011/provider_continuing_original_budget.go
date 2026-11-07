package registry011

import "context"

// Only the original reader's read/hash input rows. Metadata calls, path/digest
// construction, native work, qualified clocks, memory, retention and semantic
// replay remain separate whole-program obligations, not zero-cost rows here.
func providerHistoryOriginalInputClaim(p providerHistoryOriginalPlan) (providerHistoryInputUses, error) {
	var claim providerHistoryInputUses
	upper, err := p.upper()
	if err != nil {
		return claim, err
	}
	claim[providerReadEvent] = providerHistoryInputUse{upper.readBytes, upper.readCalls}
	claim[providerHashEvent] = providerHistoryInputUse{upper.hashBytes, upper.hashWrites}
	return claim, nil
}

// Known retained lengths allow the entire read/hash upper to be consumed
// before the first filesystem operation. Every underlying ReadAt/Hash.Write
// has input <=8192, including the separate one-byte EOF probe. Failed attempts
// retain their prior charge and poison the instance; replacing an original
// or receiving a shorter read cannot refund it or create another permission.
// This returns integrity DATA only. It is not group C/S or Source admission.
func (r *providerHistoryOriginalReader) scanWithInputBudget(ctx context.Context, p providerHistoryOriginalPlan, b *providerHistoryInputBudget) (string, error) {
	claim, err := providerHistoryOriginalInputClaim(p)
	if err != nil || ctx == nil || r == nil || b == nil {
		return "", ErrConfiguration
	}
	// Lock order is budget then reader; scan never calls back into a budget.
	b.mu.Lock()
	defer b.mu.Unlock()
	// Narrow widths are not inferred from total bytes or invocation count.
	// The concrete scan implementation supplies this disjoint per-input bound.
	if b.envelope[providerReadEvent].width < providerHistoryOriginalChunk ||
		b.envelope[providerHashEvent].width < providerHistoryOriginalChunk || !b.consumeLocked(claim) {
		b.failed = true
		return "", ErrConfiguration
	}
	digest, err := r.scan(ctx, p)
	if err != nil {
		b.failed = true
		return "", err
	}
	return digest, nil
}
