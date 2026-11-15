package wire011

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Chapter 08 §1 lists role as an optional common member. A verifier must
// accept it when it matches the sender's role and refuse any other value.
func TestOptionalRoleMember(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	env := envelopeOf(t, req)
	env.Role = "initiator"
	if _, err := f.receiver.VerifyRequest(context.Background(), signedRequest(t, f, env, true)); err != nil {
		t.Fatalf("request with role initiator refused: %s", f.trace.lastReason())
	}
	f = newFixture(t)
	req, _ = f.request(t)
	env = envelopeOf(t, req)
	env.Role = "responder"
	f.expectRefusal(t, signedRequest(t, f, env, true), "role")

	var out Request
	if decodeClosed([]byte(`{"role":""}`), requestMembers, &out) == nil {
		t.Fatal("present but empty role accepted")
	}
}

func TestResponseRole(t *testing.T) {
	for role, accept := range map[string]bool{"responder": true, "initiator": false} {
		t.Run(role, func(t *testing.T) {
			f := newFixture(t)
			req, sent := f.request(t)
			v, err := f.receiver.VerifyRequest(context.Background(), *req)
			if err != nil {
				t.Fatal(f.trace.lastReason())
			}
			ok, _ := f.server.SignResponse(v, 200, true, []byte("x"), "")
			var env Response
			if err := json.Unmarshal(ok.Body, &env); err != nil {
				t.Fatal(err)
			}
			env.Role = role
			_, err = f.responses.VerifyResponse(context.Background(), sent, *resign(t, f, v, env, 200))
			if accept && err != nil {
				t.Fatalf("response with role responder refused: %s", f.trace.lastReason())
			}
			if !accept && (err == nil || !strings.Contains(f.trace.lastReason(), "role")) {
				t.Fatalf("response with role initiator: %v (%s)", err, f.trace.lastReason())
			}
		})
	}
}
