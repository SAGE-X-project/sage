package registry011

import "strconv"

type providerMethodLimit struct{ body, wire int }

// Every class is explicitly selected before the first request. Exceeding a
// smaller class refuses; it never retries with the global or another class.
type providerMethodBounds struct {
	scalar, header, code, state, block, receipts, gas, execution providerMethodLimit
}

type providerTLSBounds struct{ chains, certificates, certificateBytes int }

func (b providerTLSBounds) valid() bool {
	return b.chains > 0 && b.chains <= 8 && b.certificates > 0 && b.certificates <= 16 && b.certificateBytes > 0 && b.certificateBytes <= 64<<10
}

func (b providerTLSBounds) upper() (uint64, bool) {
	if !b.valid() {
		return 0, false
	}
	certificates, ok := providerUpperProduct(uint64(b.chains), uint64(b.certificates))
	if !ok {
		return 0, false
	}
	return providerUpperProduct(certificates, uint64(b.certificateBytes))
}

func (b providerTLSBounds) object() map[string]string {
	return map[string]string{"chains": strconv.Itoa(b.chains), "certificates_per_chain": strconv.Itoa(b.certificates), "certificate_bytes": strconv.Itoa(b.certificateBytes)}
}

func (b providerMethodBounds) valid() bool {
	for _, limit := range []providerMethodLimit{b.scalar, b.header, b.code, b.state, b.block, b.receipts, b.gas, b.execution} {
		if limit.body <= 0 || limit.body > 8<<20 || limit.wire <= 0 || limit.wire > 16<<20 || limit.body >= limit.wire {
			return false
		}
	}
	return true
}

func (b providerMethodBounds) fits(global ProviderCaptureLimits) bool {
	if !b.valid() || !global.valid() {
		return false
	}
	for _, limit := range []providerMethodLimit{b.scalar, b.header, b.code, b.state, b.block, b.receipts, b.gas, b.execution} {
		if limit.body > global.BodyBytes || limit.wire > global.WireBytes || limit.wire <= global.HeaderBytes {
			return false
		}
	}
	return true
}

func (b providerMethodBounds) object() map[string]any {
	result := make(map[string]any, 8)
	for name, limit := range map[string]providerMethodLimit{"scalar": b.scalar, "header": b.header, "code": b.code, "state": b.state, "block": b.block, "receipts": b.receipts, "gas": b.gas, "execution": b.execution} {
		result[name] = map[string]string{"body_bytes": strconv.Itoa(limit.body), "wire_bytes": strconv.Itoa(limit.wire)}
	}
	return result
}

func (b providerMethodBounds) capture(request *providerIntegrationRequest, global ProviderCaptureLimits) (ProviderCaptureLimits, error) {
	if !b.fits(global) || request == nil || len(request.params) > 8<<10 || len(request.wire) > 16<<10 {
		return ProviderCaptureLimits{}, errProviderIntegration
	}
	var limit providerMethodLimit
	switch request.method {
	case "eth_chainId", "web3_clientVersion":
		limit = b.scalar
	case "eth_getCode":
		limit = b.code
	case "eth_call":
		limit = b.state
	case "debug_getRawBlock":
		limit = b.block
	case "eth_getBlockReceipts", "debug_getRawReceipts":
		limit = b.receipts
	case "eth_getBlockByHash", "eth_getBlockByNumber":
		params, ok := contentJSON([]byte(request.params)).([]any)
		if !ok || len(params) != 2 {
			return ProviderCaptureLimits{}, errProviderIntegration
		}
		full, ok := params[1].(bool)
		if !ok {
			return ProviderCaptureLimits{}, errProviderIntegration
		}
		limit = b.header
		if full {
			limit = b.block
		}
	case "debug_traceTransaction", "debug_traceBlockByHash":
		params, ok := contentJSON([]byte(request.params)).([]any)
		if !ok || len(params) != 2 {
			return ProviderCaptureLimits{}, errProviderIntegration
		}
		options, ok := params[1].(map[string]any)
		if !ok {
			return ProviderCaptureLimits{}, errProviderIntegration
		}
		switch options["tracer"] {
		case "stateGasTracer":
			limit = b.gas
		case "callTracer":
			limit = b.execution
		default:
			return ProviderCaptureLimits{}, errProviderIntegration
		}
	default:
		return ProviderCaptureLimits{}, errProviderIntegration
	}
	return limit.capture(global, len(request.wire))
}

// Keep the final capture validation at the selected-class boundary as well.
func (limit providerMethodLimit) capture(global ProviderCaptureLimits, requestBytes int) (ProviderCaptureLimits, error) {
	if limit.body > global.BodyBytes || limit.wire > global.WireBytes || requestBytes >= limit.wire {
		return ProviderCaptureLimits{}, errProviderIntegration
	}
	global.BodyBytes, global.WireBytes = limit.body, limit.wire
	if !global.valid() {
		return ProviderCaptureLimits{}, errProviderIntegration
	}
	return global, nil
}

func (c *providerIntegration) operationPrimitive(request *providerIntegrationRequest) (providerPrimitiveBounds, error) {
	if !c.valid() || request == nil {
		return providerPrimitiveBounds{}, errProviderIntegration
	}
	return c.store.config.bounds.operationPrimitive(request, c.primitive)
}

// This selector returns bounded DATA, never an admitted operation or permit.
func (b providerIntegrationBounds) operationPrimitive(request *providerIntegrationRequest, primitive providerPrimitiveBounds) (providerPrimitiveBounds, error) {
	if b.continuation == nil {
		return primitive, nil
	}
	if b.methods == nil || b.tls == nil || !b.tls.valid() {
		return providerPrimitiveBounds{}, errProviderIntegration
	}
	capture, err := b.methods.capture(request, primitive.capture)
	if err != nil {
		return providerPrimitiveBounds{}, err
	}
	primitive.capture = capture
	return primitive, nil
}

// The authenticated frontier validates a bounded new delta, while full scans
// are separately admitted commands. This estimate never charges a fictitious
// whole-prefix reread to every small RPC or silently switches ACK semantics.
func providerEstimateFrontierOperation(store providerIntegrationBounds, primitive providerPrimitiveBounds, requestBytes uint64, rpc, newSegment bool) (providerOperationReservation, error) {
	if !store.valid() || store.continuation == nil || store.tls == nil || !primitive.capture.valid() || primitive.maximumCreatedObjects == 0 || primitive.maximumCreatedObjects > uint64(store.maximumEntries) || rpc && (requestBytes == 0 || requestBytes >= uint64(primitive.capture.WireBytes)) || !rpc && requestBytes != 0 {
		return providerOperationReservation{}, errProviderIntegration
	}
	tls, ok := store.tls.upper()
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	// DER is stored in the closed TLS identity with bounded encoding overhead.
	tls, ok = providerUpperProduct(tls, 4)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	metadata, ok := providerUpperProduct(store.maximumMetadataPerCommit, 20)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	files, ok := providerUpperSum(primitive.maximumCreatedObjects, 40)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	objects, ok := providerUpperProduct(files, store.fileOverheadBytes)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	payload, ok := providerUpperSum(requestBytes, uint64(primitive.capture.WireBytes), uint64(primitive.capture.BodyBytes), tls)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	payload, ok = providerUpperProduct(payload, 4)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	disk, ok := providerUpperSum(payload, metadata, objects)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	decoded, ok := providerUpperSum(payload, metadata)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	units, ok := providerUpperProduct(decoded, decoded)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	units, ok = providerUpperSum(units, disk, 65536)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	execution := candidateWorkVector{0, 0, decoded, primitive.blocks, primitive.transactions, primitive.frames, primitive.logs, primitive.proofs, 0, disk, units}
	if rpc {
		execution[0], execution[1] = 1, uint64(primitive.capture.WireBytes)
		for i := 3; i <= 7; i++ {
			execution[i] = 0
		}
	}
	if newSegment {
		execution[8] = 1
	}
	remaining := candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, disk, units}
	return providerFrontierReservation(execution, remaining, disk, store.maximumStoredBytes)
}

// Execution and authority work share every coordinate without wraparound.
func providerFrontierWorkTotal(execution, remaining candidateWorkVector) (candidateWorkVector, error) {
	var total candidateWorkVector
	for i := range total {
		var ok bool
		total[i], ok = providerUpperSum(execution[i], remaining[i])
		if !ok {
			return candidateWorkVector{}, errProviderIntegration
		}
	}
	return total, nil
}

// Combine resource coordinates and physical holding before returning a quote.
func providerFrontierReservation(execution, remaining candidateWorkVector, disk, maximumStoredBytes uint64) (providerOperationReservation, error) {
	total, err := providerFrontierWorkTotal(execution, remaining)
	if err != nil {
		return providerOperationReservation{}, err
	}
	capacity, ok := providerUpperProduct(disk, 2)
	if !ok || capacity > maximumStoredBytes {
		return providerOperationReservation{}, errProviderIntegration
	}
	return providerOperationReservation{work: total, execution: execution, remaining: remaining, live: capacity, retained: capacity}, nil
}
