package registry011

import (
	"bytes"
	"testing"
)

// Read normal owned originals into an independent DATA replay, stopping before
// the selected transition. It has no owner, ACK, permit or recovery authority.
func providerReplayBeforeUnit(t *testing.T, s *providerCustodyStore, target uint64) (*providerCustodyStore, *providerCustodyReplay, map[string]any) {
	t.Helper()
	v := providerReadDataView(t, s)
	for key, ref := range s.objects {
		v.objects[key] = ref
	}
	for key, value := range s.bindings {
		v.bindings[key] = value
	}
	v.setups = append([]providerSetupReference(nil), s.setups...)
	v.setupTip = s.setupTip
	r := &providerCustodyReplay{context: s.replay.context, policy: s.replay.policy}
	for sequence := uint64(1); sequence <= target; sequence++ {
		raw, err := v.read(v.custody, providerCustodyName("intent", sequence), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		m := contentJSON(raw).(map[string]any)
		if sequence == target {
			return v, r, m
		}
		if err := r.apply(v, m); err != nil {
			t.Fatal("normal independent data prefix", sequence, err)
		}
		commit, err := v.read(v.custody, providerCustodyName("commit", sequence), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		v.tip, v.sequence = providerCandidateHash("sage-provider-custody-commit-v1", commit), sequence
		v.enrollment, v.objectsRoot = contentString(m["new_enrollment_root"]), contentString(m["objects_root"])
	}
	t.Fatal("target not reached")
	return nil, nil, nil
}

func TestProviderIntegrationTypedReplayRejectsIncompleteTransitionClosure(t *testing.T) {
	_, s := providerRetainedFirstExchangeFixture(t)
	for _, item := range []struct {
		name     string
		sequence uint64
		mode     string
	}{
		{"missing-root", 1, "root"}, {"duplicate-enroll", 1, "anchor"}, {"operation", 1, "operation"}, {"setup-boundary", 1, "setup"}, {"setup-commit", 1, "setup-commit"},
		{"no-anchor", 2, "anchor"}, {"missing-budget", 2, "budget"}, {"missing-admission", 2, "admission"}, {"missing-ticket", 2, "ticket"}, {"missing-request", 2, "request"},
		{"consume-without-ledger", 3, "ledger"}, {"missing-consumption", 3, "consumption"}, {"consume-unknown-ticket", 3, "ticket-map"},
		{"settle-without-ledger", 4, "ledger"}, {"missing-settlement", 4, "settlement"}, {"settle-unknown-ticket", 4, "ticket-map"}, {"missing-exchange", 4, "exchange"}, {"missing-node", 4, "ledger-node"}, {"unknown-kind", 4, "kind"},
	} {
		t.Run(item.name, func(t *testing.T) {
			v, r, m := providerReplayBeforeUnit(t, s, item.sequence)
			switch item.mode {
			case "root":
				delete(v.objects, "enrollment:"+contentString(m["new_enrollment_root"]))
			case "anchor":
				if item.sequence == 1 {
					r.anchor = s.replay.anchor
				} else {
					r.anchor = nil
				}
			case "operation":
				m["operation"] = admissionTestDigest("f")
			case "setup":
				v.setups = nil
			case "setup-commit":
				delete(v.objects, "setup-commit:"+v.setups[0].commit)
			case "budget":
				raw, err := v.object("enrollment", contentString(m["new_enrollment_root"]))
				if err != nil {
					t.Fatal(err)
				}
				delete(v.objects, "budget:"+contentString(contentJSON(raw).(map[string]any)["budget_root"]))
			case "admission":
				delete(v.objects, "admission:"+s.replay.intent.digest)
			case "ticket", "consumption", "settlement":
				delete(v.objects, item.mode+":"+contentString(m["operation"]))
			case "request":
				delete(v.objects, "request:"+s.replay.ledger.lastTicket.input)
			case "ledger":
				r.ledger = nil
			case "ticket-map":
				r.ledger.tickets = make(map[string]candidateLedgerTicket)
			case "exchange":
				delete(v.objects, "exchange:"+s.replay.firstExchange)
			case "ledger-node":
				for key, ref := range v.objects {
					if ref.kind == "ledger-node" {
						delete(v.objects, key)
					}
				}
			case "kind":
				m["kind"] = "unsupported"
			}
			if r.apply(v, m) == nil {
				t.Fatal("incomplete independent typed replay passed", item.name)
			}
		})
	}
}

func TestProviderIntegrationTicketBytesRejectFiniteNoncanonicalRequests(t *testing.T) {
	c, s := providerRetainedFirstExchangeFixture(t)
	for _, input := range [][]byte{
		[]byte("missing request delimiter"),
		append([]byte("POST / HTTP/1.1\r\n\r\n"), 255),
		[]byte("POST / HTTP/1.1\r\n\r\n{\"x\":\"\\ud800\"}"),
		[]byte("POST / HTTP/1.1\r\n\r\n{"),
		[]byte("POST / HTTP/1.1\r\n\r\n{} {}"),
		[]byte("POST / HTTP/1.1\r\n\r\n{}"),
		[]byte("POST / HTTP/1.1\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getBlockByNumber\",\"params\":[\"finalized\",false]}"),
	} {
		v := providerReadDataView(t, s)
		object := providerRetainReplayUnitData(t, v, "request", input)
		ticket := *c.firstTicket
		ticket.input = object.digest
		if s.replay.checkTicketBytes(v, &ticket) == nil {
			t.Fatal("finite request data passed canonical retained request closure")
		}
	}
	v := providerReadDataView(t, s)
	for key, ref := range s.objects {
		v.objects[key] = ref
	}
	if s.replay.checkTicketBytes(v, c.firstTicket) != nil {
		t.Fatal("normal original request")
	}
	params, _ := providerIntegrationObjectDigest("params", []byte(c.first.params))
	delete(v.objects, "params:"+params)
	if s.replay.checkTicketBytes(v, c.firstTicket) == nil {
		t.Fatal("missing exact original params")
	}
	local := *c.firstTicket
	local.kind, local.input = "local-work", admissionTestDigest("f")
	if s.replay.checkTicketBytes(v, &local) == nil {
		t.Fatal("unretained local input")
	}
	if !bytes.Equal([]byte(c.first.params), []byte(`["finalized",false]`)) {
		t.Fatal("closed request fixture changed")
	}
}
