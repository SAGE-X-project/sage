package registry011

import (
	"encoding/json"
	"unicode/utf8"
)

// Exact immutable plaintext HTTP bytes are retained before ticket admission.
// The executor sends these same bytes, not an HTTP client's reconstruction.
type providerIntegrationRequest struct {
	method, params, body, head, wire string
	inputDigest, paramsDigest        string
}

func newProviderIntegrationRequest(method string, params any) (*providerIntegrationRequest, error) {
	if !candidateAdmissionReadMethod(method) || !providerRequestData(params, 0) {
		return nil, errProviderIntegration
	}
	parameters, err := json.Marshal(params)
	if err != nil || len(parameters) == 0 || len(parameters) > 8<<10 || parameters[0] != '[' {
		return nil, errProviderIntegration
	}
	body, _ := json.Marshal(struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{"2.0", 1, method, params})
	head := providerRequestHead(len(body))
	wire := append(append([]byte(nil), head...), body...)
	input, _ := providerIntegrationObjectDigest("request", wire)
	// The inventory's ordinary raw-content hash and the existing versioned
	// params domain serve distinct purposes. Neither is a transport credential.
	return &providerIntegrationRequest{method, string(parameters), string(body), string(head), string(wire), input, providerCandidateHash("sage-provider-params-v1", parameters)}, nil
}

// Only finite JSON data used by closed internal builders is accepted. No
// Marshaler, function, custom transport or arbitrary object is called here.
func providerRequestData(value any, depth int) bool {
	budget := providerRequestBudget{}
	return providerRequestDataBounded(value, depth, &budget)
}

type providerRequestBudget struct{ nodes, bytes int }

func providerRequestDataBounded(value any, depth int, budget *providerRequestBudget) bool {
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
	return r != nil && r.method == "eth_getBlockByNumber" && r.params == `["finalized",false]` && r.body == `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["finalized",false]}` && r.head == string(providerRequestHead(len(r.body))) && r.wire == r.head+r.body
}
