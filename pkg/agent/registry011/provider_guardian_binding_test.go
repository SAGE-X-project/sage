package registry011

import "testing"

func providerGuardianBindingFixture(t *testing.T, rpc bool) (*providerScanCodec, providerGuardianAdmissionBinding) {
	t.Helper()
	x, _, a := providerGuardianRecordFixture(t)
	v := providerGuardianAdmissionBinding{ordinal: a.ordinal, claim: admissionTestDigest("a"), head: admissionTestDigest("b"), previous: a.previous, main: a.main, frontier: a.frontier, setup: a.setup, admission: a.admission, operation: a.operation, purpose: "bootstrap-admission", ticket: admissionTestDigest("c"), input: admissionTestDigest("d"), kind: "local-work", clock: a.clock}
	if rpc {
		v.kind, v.purpose, v.method, v.params = "rpc-capture", "rpc-admission", "eth_chainId", admissionTestDigest("e")
	}
	var err error
	v.target, err = x.guardianTicketTarget(v.purpose, v.admission, v.kind, v.input, v.method, v.params)
	if err != nil {
		t.Fatal(err)
	}
	return x, v
}

func TestProviderGuardianBindingPreservesExactTicketInputAndClosedRole(t *testing.T) {
	for _, rpc := range []bool{false, true} {
		x, v := providerGuardianBindingFixture(t, rpc)
		raw, err := x.guardianAdmissionBinding(v)
		parsed, parseErr := x.parseGuardianAdmissionBinding(raw)
		if err != nil || parseErr != nil || parsed != v || parsed.input != v.input || parsed.target == parsed.input || parsed.ticket == parsed.target {
			t.Fatal("binding rewrote TicketV2 input or introduced a cyclic target", err, parseErr)
		}
	}
}

func TestProviderGuardianBindingRejectsFiniteInvalidIdentityAndRoleData(t *testing.T) {
	for _, mode := range []string{"nil", "bounds", "ordinal", "attempts", "clock", "claim", "head", "previous", "main", "frontier", "setup", "admission", "operation", "target", "ticket", "input", "purpose", "kind", "method", "params", "different-target"} {
		t.Run(mode, func(t *testing.T) {
			x, v := providerGuardianBindingFixture(t, false)
			switch mode {
			case "nil":
				x = nil
			case "bounds":
				x.config.bounds.guardian = nil
			case "ordinal":
				v.ordinal = 0
			case "attempts":
				v.ordinal = x.config.bounds.guardian.attempts + 1
			case "clock":
				v.clock = CandidateClockDiagnostic{}
			case "claim":
				v.claim = "missing"
			case "head":
				v.head = "missing"
			case "previous":
				v.previous = "missing"
			case "main":
				v.main = "missing"
			case "frontier":
				v.frontier = "missing"
			case "setup":
				v.setup = "missing"
			case "admission":
				v.admission = "missing"
			case "operation":
				v.operation = "missing"
			case "target":
				v.target = "missing"
			case "ticket":
				v.ticket = "missing"
			case "input":
				v.input = "missing"
			case "purpose":
				v.purpose = "ledger-page-seal"
			case "kind":
				v.kind = "arbitrary-work"
			case "method":
				v.method = "eth_chainId"
			case "params":
				v.params = admissionTestDigest("a")
			case "different-target":
				v.target = admissionTestDigest("f")
			}
			if raw, err := x.guardianAdmissionBinding(v); err == nil || raw != nil {
				t.Fatal("invalid binding DATA encoded", mode)
			}
		})
	}
	for _, mode := range []string{"shape", "format", "extra", "identity", "clock", "ordinal", "null", "noncanonical", "rpc-method", "rpc-params", "rpc-purpose"} {
		t.Run(mode, func(t *testing.T) {
			x, v := providerGuardianBindingFixture(t, true)
			raw, err := x.guardianAdmissionBinding(v)
			if err != nil {
				t.Fatal(err)
			}
			m := contentJSON(raw).(map[string]any)
			switch mode {
			case "shape":
				delete(m, "original_clock")
			case "format":
				m["format"] = "unsupported-unit-format"
			case "extra":
				m["unknown"] = "unit-data"
			case "identity":
				m["source_id"] = admissionTestDigest("f")
			case "clock":
				m["original_clock"] = nil
			case "ordinal":
				m["ordinal"] = 1
			case "rpc-method":
				m["method"] = "eth_sendRawTransaction"
			case "rpc-params":
				m["params_digest"] = nil
			case "rpc-purpose":
				m["purpose"] = "bootstrap-admission"
			}
			raw = providerIntegrationJSON(m)
			switch mode {
			case "null":
				raw = []byte("null")
			case "noncanonical":
				raw = append(raw, '\n')
			}
			if _, err := x.parseGuardianAdmissionBinding(raw); err == nil {
				t.Fatal("invalid or noncanonical original accepted", mode)
			}
		})
	}
}

func TestProviderGuardianTargetsSeparateClosedPurposesAndInputs(t *testing.T) {
	x, v := providerGuardianBindingFixture(t, false)
	seen := map[string]bool{}
	for _, purpose := range []string{"bootstrap-admission", "local-admission", "publication-inspection"} {
		target, err := x.guardianTicketTarget(purpose, v.admission, v.kind, v.input, "", "")
		if err != nil || seen[target] {
			t.Fatal("closed purposes shared a target")
		}
		seen[target] = true
	}
	if _, err := x.guardianTicketTarget("rpc-admission", v.admission, "rpc-capture", v.input, "eth_sendRawTransaction", admissionTestDigest("a")); err == nil {
		t.Fatal("a write method became a candidate read target")
	}
	var missing *providerScanCodec
	if _, err := missing.guardianTicketTarget(v.purpose, v.admission, v.kind, v.input, "", ""); err == nil {
		t.Fatal("missing codec accepted a target")
	}
	if _, err := missing.parseGuardianAdmissionBinding(nil); err == nil {
		t.Fatal("missing codec accepted a binding")
	}
	if _, err := providerGuardianBindingForTicket(nil, nil, nil); err == nil {
		t.Fatal("missing originals supplied a binding")
	}
}
