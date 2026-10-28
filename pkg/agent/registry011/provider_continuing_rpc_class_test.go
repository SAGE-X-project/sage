package registry011

import "testing"

func TestProviderRetainedMethodClassIsDerivedFromExactRequestData(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithPageFixture(t, 64<<20, 6)
	// This store-shaped value contains config DATA only: no owner, disks,
	// clock, guardian, read/write capability or execution authority exists.
	s := &providerCustodyStore{config: config}
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	ticket := &CandidateAdmissionTicket{kind: "rpc-capture", method: request.method, input: request.inputDigest, params: request.paramsDigest}
	wanted, err := config.bounds.methods.capture(request, config.capture)
	got, classErr := providerRetainedMethodCapture(s, ticket, []byte(request.wire), nil)
	if err != nil || classErr != nil || got != wanted || s.owner != nil || s.guardian != nil || s.custody != nil {
		t.Fatal("exact retained request class DATA", err, classErr)
	}
	for _, mode := range []string{"kind", "method", "input", "params", "missing-wire", "large-wire", "separator", "json", "version", "id", "unsupported-params", "different-head", "missing-guardian", "missing-methods"} {
		t.Run(mode, func(t *testing.T) {
			c, entry, wire := s.config, *ticket, request.wire
			switch mode {
			case "kind":
				entry.kind = "local-work"
			case "method":
				entry.method = "eth_chainId"
			case "input":
				entry.input = admissionTestDigest("f")
			case "params":
				entry.params = admissionTestDigest("f")
			case "missing-wire":
				wire = ""
			case "large-wire":
				wire = string(make([]byte, (16<<10)+1))
			case "separator":
				wire = "ordinary finite DATA"
			case "json":
				wire = "POST / HTTP/1.1\r\n\r\n{"
			case "version":
				wire = "POST / HTTP/1.1\r\n\r\n" + `{"jsonrpc":"1.0","id":1,"method":"eth_getBlockByNumber","params":["finalized",false]}`
			case "id":
				wire = "POST / HTTP/1.1\r\n\r\n" + `{"jsonrpc":"2.0","id":2,"method":"eth_getBlockByNumber","params":["finalized",false]}`
			case "unsupported-params":
				wire = "POST / HTTP/1.1\r\n\r\n" + `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":1}`
			case "different-head":
				wire = "X" + request.wire[1:]
			case "missing-guardian":
				c.bounds.guardian = nil
			case "missing-methods":
				c.bounds.methods = nil
			}
			input := &providerCustodyStore{config: c}
			if _, err := providerRetainedMethodCapture(input, &entry, []byte(wire), nil); err == nil {
				t.Fatal("inconsistent finite request DATA selected a class", mode)
			}
		})
	}
	// A syntactically admitted request still needs the closed tracer class.
	unsupported, err := newProviderIntegrationRequest("debug_traceTransaction", []any{admissionTestDigest("a"), map[string]any{"tracer": "unsupported-unit-class"}})
	if err != nil {
		t.Fatal(err)
	}
	entry := &CandidateAdmissionTicket{kind: "rpc-capture", method: unsupported.method, input: unsupported.inputDigest, params: unsupported.paramsDigest}
	if _, err := providerRetainedMethodCapture(s, entry, []byte(unsupported.wire), nil); err == nil {
		t.Fatal("unknown class gained a global fallback")
	}
}
