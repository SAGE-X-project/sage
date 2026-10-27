package registry011

import (
	"context"
	"testing"
)

// Absent/incomplete owned components are finite unit data. None of these cases
// opens a transport, changes a file, supplies a callback or constructs authority.
func TestProviderIntegrationAbsentOwnersCannotPrepareAdmitOrDispatch(t *testing.T) {
	for _, c := range []*providerIntegration{nil, {}} {
		if c.valid() || c.prepare() == nil {
			t.Fatal("absent owner prepared an operation")
		}
		if _, err := c.admitRPC(nil, false); err == nil {
			t.Fatal("absent owner admitted RPC work")
		}
		if _, err := c.admit("local-work", admissionTestDigest("a"), "", "", providerOperationReservation{}, nil); err == nil {
			t.Fatal("absent owner admitted local work")
		}
		if _, err := c.commitChange("admit", admissionTestDigest("a"), nil, nil); err == nil {
			t.Fatal("absent owner committed a model")
		}
		if _, err := c.consume(nil, nil); err == nil || c.use(nil) {
			t.Fatal("absent owner acquired a dispatch permit")
		}
		if _, err := c.attachMeter(providerOperationReservation{}); err == nil {
			t.Fatal("absent owner attached a measurement")
		}
		if _, err := c.readClient(); err == nil {
			t.Fatal("absent owner acquired a network client")
		}
		if _, _, err := c.execute(context.Background(), "eth_chainId", []any{}, 1024); err == nil {
			t.Fatal("absent owner dispatched a request")
		}
		if _, _, err := c.finishRPC(nil, nil, nil); err == nil {
			t.Fatal("absent owner published an RPC result")
		}
		if _, err := c.finishLocalResult(nil, "pin", nil); err == nil {
			t.Fatal("absent owner published a local result")
		}
		if _, err := c.beginLocal(admissionTestDigest("a")); err == nil {
			t.Fatal("absent owner began local work")
		}
		if _, err := c.localPrimitive(admissionTestDigest("a"), nil); err == nil {
			t.Fatal("absent owner selected a narrower algorithm")
		}
		if c.installBootstrap() == nil {
			t.Fatal("absent owner installed a bootstrap")
		}
		if _, err := c.pinAgent(context.Background(), "normal-agent"); err == nil {
			t.Fatal("absent owner pinned a Registry")
		}
		if _, err := c.operationPrimitive(nil); err == nil || c.beginRemaining(nil) == nil {
			t.Fatal("absent owner resumed measured work")
		}
	}
	var s *providerCustodyStore
	if _, err := s.commit("enroll", admissionTestDigest("a"), nil, nil); err == nil || s.confirms(nil, "", nil) {
		t.Fatal("absent store acknowledged custody")
	}
	var f *providerFrontierFiles
	if _, err := f.initialize(); err == nil {
		t.Fatal("absent frontier initialized")
	}
	if _, err := f.advance(providerFrontierHighwater{}, "main", nil, nil, nil); err == nil {
		t.Fatal("absent frontier advanced")
	}
	if _, err := newProviderFrontierFiles(nil); err == nil {
		t.Fatal("absent store selected a frontier owner")
	}
}
