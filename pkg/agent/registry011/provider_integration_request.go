package registry011

import "unicode/utf8"

// Exact immutable plaintext HTTP bytes are retained before ticket admission.
// The executor sends these same bytes, not an HTTP client's reconstruction.
type providerIntegrationRequest struct {
	method, params, body, head, wire string
	inputDigest, paramsDigest        string
}

func newProviderIntegrationRequest(method string, params any) (*providerIntegrationRequest, error) {
	return newProviderIntegrationRequestObserved(method, params, nil)
}

func newProviderIntegrationRequestObserved(method string, params any, meter *providerWorkMeter) (*providerIntegrationRequest, error) {
	if !candidateAdmissionReadMethod(method) || !providerRequestDataObserved(params, 0, meter) {
		return nil, errProviderIntegration
	}
	parameters := providerReplayMarshal(params, 8<<10, meter)
	if len(parameters) == 0 || len(parameters) > 8<<10 || parameters[0] != '[' {
		return nil, errProviderIntegration
	}
	body := providerReplayMarshal(struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{"2.0", 1, method, params}, 16<<10, meter)
	if len(body) == 0 || meter != nil && !meter.preflight(providerEncodeEvent, 512) {
		return nil, errProviderIntegration
	}
	head := providerRequestHead(len(body))
	if meter != nil && (!meter.event(providerEncodeEvent, uint64(len(head))) || !meter.event(providerCopyEvent, uint64(len(head)))) {
		return nil, errProviderIntegration
	}
	wire := make([]byte, 0, len(head)+len(body))
	wire = append(wire, head...)
	if meter != nil && !meter.event(providerCopyEvent, uint64(len(body))) {
		return nil, errProviderIntegration
	}
	wire = append(wire, body...)
	input, valid := providerIntegrationObjectDigestObserved("request", wire, meter)
	// The inventory's ordinary raw-content hash and the existing versioned
	// params domain serve distinct purposes. Neither is a transport credential.
	parameterText, bodyText := providerReplayString(parameters, meter), providerReplayString(body, meter)
	headText, wireText := providerReplayString(head, meter), providerReplayString(wire, meter)
	paramsDigest := providerReplayDigest("sage-provider-params-v1", parameters, meter)
	if !valid || input == "" || paramsDigest == "" || parameterText == "" || bodyText == "" || headText == "" || wireText == "" {
		return nil, errProviderIntegration
	}
	return &providerIntegrationRequest{method, parameterText, bodyText, headText, wireText, input, paramsDigest}, nil
}

// Only finite JSON data used by closed internal builders is accepted. No
// Marshaler, function, custom transport or arbitrary object is called here.
func providerRequestData(value any, depth int) bool {
	return providerRequestDataObserved(value, depth, nil)
}

func providerRequestDataObserved(value any, depth int, meter *providerWorkMeter) bool {
	budget := providerRequestBudget{meter: meter}
	return providerRequestDataBounded(value, depth, &budget)
}

type providerRequestBudget struct {
	nodes, bytes int
	meter        *providerWorkMeter
}

func providerRequestDataBounded(value any, depth int, budget *providerRequestBudget) bool {
	if budget.meter != nil && !budget.meter.event(providerCompareEvent, 1) {
		return false
	}
	budget.nodes++
	if budget.nodes > 512 {
		return false
	}
	if depth > 8 {
		return false
	}
	switch v := value.(type) {
	case bool:
		return true
	case int:
		return v == 0 // The fixed tracer option reexec is exactly zero.
	case string:
		if len(v) > (8<<10)-budget.bytes {
			return false
		}
		if budget.meter != nil && !budget.meter.event(providerCompareEvent, uint64(len(v))) {
			return false
		}
		budget.bytes += len(v)
		return utf8.ValidString(v)
	case []any:
		if len(v) > 16 {
			return false
		}
		for _, item := range v {
			if !providerRequestDataBounded(item, depth+1, budget) {
				return false
			}
		}
		return true
	case map[string]string:
		if len(v) > 16 {
			return false
		}
		for name, value := range v {
			if !providerRequestDataBounded(name, depth+1, budget) || !providerRequestDataBounded(value, depth+1, budget) {
				return false
			}
		}
		return true
	case map[string]bool:
		if len(v) > 16 {
			return false
		}
		for name := range v {
			if !providerRequestDataBounded(name, depth+1, budget) {
				return false
			}
		}
		return true
	case map[string]any:
		if len(v) > 16 {
			return false
		}
		for name, item := range v {
			if !providerRequestDataBounded(name, depth+1, budget) || !providerRequestDataBounded(item, depth+1, budget) {
				return false
			}
		}
		return true
	}
	return false
}

func (r *providerIntegrationRequest) firstFinalized() bool {
	return r.firstFinalizedObserved(nil)
}

func (r *providerIntegrationRequest) firstFinalizedObserved(meter *providerWorkMeter) bool {
	if r == nil || !providerReplayTextEqual(r.method, "eth_getBlockByNumber", meter) || !providerReplayTextEqual(r.params, `["finalized",false]`, meter) || !providerReplayTextEqual(r.body, `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["finalized",false]}`, meter) || meter != nil && !meter.preflight(providerEncodeEvent, 512) {
		return false
	}
	head := providerRequestHead(len(r.body))
	if meter != nil && !meter.event(providerEncodeEvent, uint64(len(head))) {
		return false
	}
	expected := providerReplayString(head, meter)
	if expected == "" || !providerReplayTextEqual(r.head, expected, meter) {
		return false
	}
	wire := providerReplayJoin([]string{r.head, r.body}, meter)
	return wire != "" && providerReplayTextEqual(r.wire, wire, meter)
}
