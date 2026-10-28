package registry011

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Preparation bounds for a new, private group representation. They do not
// enlarge any selected operating profile or prove a whole group fits custody.
const providerHistoryGroupMembers = 32
const providerHistoryGroupTableBytes = 64 << 10
const providerHistoryGroupTableDomain = "sage-provider-history-members-v1"

type providerHistoryGroupMember struct {
	request providerIntegrationRequest
	class   string
	capture ProviderCaptureLimits
}

// A closed ordered request table is DATA, never T/C/S, a hold or a permit.
// Requests are copied by value; neither caller slices nor request pointers are
// retained. The future program must independently validate the closed grammar
// against retained prior originals before a table can be admitted in T.
type providerHistoryGroupTable struct {
	program, purpose string
	stage            uint64
	count            int
	members          [providerHistoryGroupMembers]providerHistoryGroupMember
	raw, digest      string
}

func (*providerHistoryGroupTable) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryGroupTable(program, purpose string, stage uint64, requests []*providerIntegrationRequest, bounds providerMethodBounds, global ProviderCaptureLimits) (*providerHistoryGroupTable, error) {
	if !lowerHash(program) || stage == 0 || stage > 1<<53 || len(requests) == 0 || len(requests) > providerHistoryGroupMembers || !bounds.fits(global) {
		return nil, ErrConfiguration
	}
	switch purpose {
	case "selection", "identity", "ancestry", "content", "effects", "state", "history", "final":
	default:
		return nil, ErrConfiguration
	}
	t := &providerHistoryGroupTable{program: program, purpose: purpose, stage: stage, count: len(requests)}
	for i, input := range requests {
		request, err := providerHistoryGroupRequestCopy(input)
		if err != nil {
			return nil, err
		}
		class, err := providerHistoryGroupRequestClass(&request)
		if err != nil {
			return nil, err
		}
		capture, err := bounds.capture(&request, global)
		if err != nil {
			return nil, err
		}
		t.members[i] = providerHistoryGroupMember{request, class, capture}
	}
	raw := providerIntegrationJSON(t.object())
	if len(raw) == 0 || len(raw) > providerHistoryGroupTableBytes {
		return nil, ErrConfiguration
	}
	t.raw = string(raw)
	t.digest = providerCandidateHash(providerHistoryGroupTableDomain, raw)
	return t, nil
}

// Rebuild the whole request, including its original HTTP bytes and both
// different digest contracts. No caller-supplied digest alone is accepted.
func providerHistoryGroupRequestCopy(input *providerIntegrationRequest) (providerIntegrationRequest, error) {
	var empty providerIntegrationRequest
	if input == nil || len(input.params) == 0 || len(input.params) > 8<<10 || len(input.body) == 0 || len(input.body) > 16<<10 ||
		len(input.head) == 0 || len(input.head) > 512 || len(input.wire) == 0 || len(input.wire) > 16<<10 {
		return empty, ErrConfiguration
	}
	var params any
	if json.Unmarshal([]byte(input.params), &params) != nil {
		return empty, ErrConfiguration
	}
	copy, err := newProviderIntegrationRequest(input.method, providerRequestNumbers(params))
	if err != nil || *copy != *input {
		return empty, ErrConfiguration
	}
	return *copy, nil
}

// Symbolic class is derived from exact request DATA, not inferred by equal
// numeric ceilings: two explicitly selected classes may have identical caps.
// The existing selector separately verifies the chosen capture at construction.
func providerHistoryGroupRequestClass(r *providerIntegrationRequest) (string, error) {
	if r == nil || len(r.params) > 8<<10 {
		return "", ErrConfiguration
	}
	switch r.method {
	case "eth_chainId", "web3_clientVersion":
		return "scalar", nil
	case "eth_getCode":
		return "code", nil
	case "eth_call":
		return "state", nil
	case "debug_getRawBlock":
		return "block", nil
	case "eth_getBlockReceipts", "debug_getRawReceipts":
		return "receipts", nil
	}
	var params []any
	if json.Unmarshal([]byte(r.params), &params) != nil || len(params) != 2 {
		return "", ErrConfiguration
	}
	switch r.method {
	case "eth_getBlockByHash", "eth_getBlockByNumber":
		full, ok := params[1].(bool)
		if !ok {
			return "", ErrConfiguration
		}
		if full {
			return "block", nil
		}
		return "header", nil
	case "debug_traceTransaction", "debug_traceBlockByHash":
		options, ok := params[1].(map[string]any)
		if !ok {
			return "", ErrConfiguration
		}
		switch options["tracer"] {
		case "stateGasTracer":
			return "gas", nil
		case "callTracer":
			return "execution", nil
		}
	}
	return "", ErrConfiguration
}

func (t *providerHistoryGroupTable) object() map[string]any {
	if t == nil || t.count <= 0 || t.count > providerHistoryGroupMembers {
		return nil
	}
	members := make([]any, t.count)
	for i := range members {
		m := t.members[i]
		members[i] = map[string]any{
			"ordinal": strconv.Itoa(i), "method": m.request.method, "class": m.class,
			"request": m.request.inputDigest, "params": m.request.paramsDigest,
			"timeout_ns":   strconv.FormatInt(int64(m.capture.Timeout), 10),
			"header_bytes": strconv.Itoa(m.capture.HeaderBytes), "body_bytes": strconv.Itoa(m.capture.BodyBytes),
			"wire_bytes": strconv.Itoa(m.capture.WireBytes), "json_depth": strconv.Itoa(m.capture.JSONDepth),
		}
	}
	return map[string]any{"format": providerHistoryGroupTableDomain, "program": t.program,
		"purpose": t.purpose, "stage": strconv.FormatUint(t.stage, 10), "members": members}
}

// Explicit bounded parsing reads an original table and the separately retained
// original requests. It returns only newly reconstructed DATA. ACK provenance,
// program semantics, prior ledger state and admission remain separate checks.
func parseProviderHistoryGroupTable(raw []byte, program, purpose string, stage uint64, requests []*providerIntegrationRequest, bounds providerMethodBounds, global ProviderCaptureLimits) (*providerHistoryGroupTable, error) {
	if len(raw) == 0 || len(raw) > providerHistoryGroupTableBytes {
		return nil, ErrConfiguration
	}
	t, err := newProviderHistoryGroupTable(program, purpose, stage, requests, bounds, global)
	if err != nil || !bytes.Equal(raw, []byte(t.raw)) {
		return nil, ErrConfiguration
	}
	return t, nil
}
