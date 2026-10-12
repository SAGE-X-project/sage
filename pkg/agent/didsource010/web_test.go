package didsource010

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/consumer"
	"github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/did-go/producer"
	"github.com/sage-x-project/did-go/resolution"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const webDID = "did:sage:web:agents.example.com:billing-bot"

// signedRecord is a web record with an Ed25519 signing key, an X25519 key
// endorsed by it and one A2A interface service, each with a valid PoP proof
// (spec chapter 09 §4).
func signedRecord(t *testing.T) map[string]any {
	t.Helper()
	enc := base64.RawURLEncoding.EncodeToString
	edPriv := ed25519.NewKeyFromSeed(make([]byte, 32))
	edPub := edPriv.Public().(ed25519.PublicKey)
	challenge := func(name, alg string, pub []byte) []byte {
		m, err := registry010.PoPChallenge010("web:agents.example.com", "billing-bot", name, alg, pub)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	kemPriv, err := ecdh.X25519().NewPrivateKey(append([]byte{7}, make([]byte, 31)...))
	if err != nil {
		t.Fatal(err)
	}
	kemPub := kemPriv.PublicKey().Bytes()
	entry := func(name, alg string, pub []byte, expires any) map[string]any {
		e := map[string]any{"name": name, "alg": alg, "key": enc(pub), "state": "accepted",
			"proof": map[string]any{"signer": webDID + "#a-ed", "value": enc(ed25519.Sign(edPriv, challenge(name, alg, pub)))}}
		if expires != nil {
			e["expires"] = expires
		}
		return e
	}
	return map[string]any{
		"id": webDID, "controller": "operator", "state": "active", "version": "3",
		"keys":     []any{entry("a-ed", "ed25519", edPub, nil), entry("b-kem", "x25519", kemPub, 4000000000)},
		"services": []any{map[string]any{"name": "c-a2a", "type": "A2AInterface", "uri": "https://agents.example.com/a2a"}},
	}
}

func body(t *testing.T, record map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"record": record, "issued": 100, "expires": 105})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func clock(times ...int64) func() (int64, error) {
	i := 0
	return func() (int64, error) {
		if i >= len(times) {
			return 0, errors.New("clock lost")
		}
		i++
		return times[i-1], nil
	}
}

// newTestSource returns a source whose fetch returns raw after the same
// envelope, shape and proof checks the REG-08 adapter applies.
func newTestSource(t *testing.T, raw []byte, fetchErr error, now func() (int64, error)) *WebSource {
	t.Helper()
	s, err := NewWebSource(WebConfig{
		AllowedOrigins:      []string{"https://agents.example.com"},
		Destination:         netip.MustParseAddrPort("192.0.2.1:443"),
		AllowedDestinations: []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:443")},
		RootDER:             []byte{1},
		SourceID:            "web-registry-1",
		Now:                 now,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.fetch = func(_ context.Context, id string, _ []string, _ netip.AddrPort, _ []netip.AddrPort, _ []byte, now int64) ([]byte, error) {
		if fetchErr != nil {
			return nil, fetchErr
		}
		if err := registry010.CheckWebRegistryProofs010(raw, id, now); err != nil {
			return nil, err
		}
		return raw, nil
	}
	return s
}

func mustDID(t *testing.T, s string) did.DID {
	t.Helper()
	d, err := did.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestObserveWebRecord(t *testing.T) {
	rec, err := newTestSource(t, body(t, signedRecord(t)), nil, clock(101, 102)).Observe(t.Context(), mustDID(t, webDID))
	if err != nil {
		t.Fatal(err)
	}
	if rec.DID.String() != webDID || rec.Controller != "operator" || rec.State != sage.StateActive || rec.Version != "3" ||
		len(rec.Keys) != 2 || len(rec.Keys[0].Material) != 32 || rec.Keys[1].Expires == nil || *rec.Keys[1].Expires != 4000000000 ||
		len(rec.Services) != 1 || rec.Services[0].Type != "A2AInterface" {
		t.Fatalf("record %+v", rec)
	}
	// RESOLVE-03: observedAt is the issued integer, the source identity is
	// configured, and observedTime is when the observation completed.
	if rec.Observation.Issued == nil || *rec.Observation.Issued != 100 || rec.Observation.Source != "web-registry-1" || rec.Observation.Time != 102 {
		t.Errorf("observation %+v", rec.Observation)
	}
}

// End to end: did-go's DID Resolution interface resolves a verified web
// record into both DIDP-01 projections.
func TestResolveThroughDidGo(t *testing.T) {
	src := newTestSource(t, body(t, signedRecord(t)), nil, func() (int64, error) { return 101, nil })
	r, err := sage.NewResolver(src, consumer.New(), producer.New())
	if err != nil {
		t.Fatal(err)
	}
	for accept, mt := range map[string]string{"": "application/did+json", "application/did": "application/did"} {
		res, code := r.ResolveWithCode(t.Context(), mustDID(t, webDID), resolution.ResolutionOptions{Accept: accept})
		if code != "" || res.ResolutionMetadata.Error != nil || res.ResolutionMetadata.ContentType != mt {
			t.Fatalf("accept %q: %+v %q", accept, res.ResolutionMetadata, code)
		}
		raw := string(res.Raw)
		if !strings.Contains(raw, `"authentication":["`+webDID+`#a-ed"]`) || !strings.Contains(raw, `"keyAgreement":["`+webDID+`#b-kem"]`) ||
			!strings.Contains(raw, `"type":"A2AInterface"`) {
			t.Errorf("accept %q: %s", accept, raw)
		}
	}
}

func TestObserveErrors(t *testing.T) {
	good := body(t, signedRecord(t))
	tampered := signedRecord(t)
	tampered["keys"].([]any)[0].(map[string]any)["proof"].(map[string]any)["value"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
	cases := []struct {
		name  string
		id    string
		raw   []byte
		fetch error
		now   func() (int64, error)
		code  sage.Code
	}{
		{"other kind", "did:sage:eip155:1:0x" + strings.Repeat("a", 40) + ":0x1", good, nil, clock(101, 102), sage.CodeIDUnknownKind},
		{"unknown kind", "did:sage:dns:example.com:a", good, nil, clock(101, 102), sage.CodeIDUnknownKind},
		{"malformed", "did:sage:web:agents.example.com:a:b", good, nil, clock(101, 102), sage.CodeIDMalformed},
		{"clock before", webDID, good, nil, clock(), sage.CodeRecordUnreachable},
		{"clock after", webDID, good, nil, clock(101), sage.CodeRecordUnreachable},
		{"unreachable", webDID, good, registry010.ErrUnreachable, clock(101, 102), sage.CodeRecordUnreachable},
		{"oversized", webDID, good, registry010.ErrSizeExceeded010, clock(101, 102), sage.CodeSizeExceeded},
		{"tampered proof", webDID, body(t, tampered), nil, clock(101, 102), sage.CodeRecordInvalid},
		{"expired envelope", webDID, good, nil, clock(105, 106), sage.CodeRecordInvalid},
	}
	for _, c := range cases {
		_, err := newTestSource(t, c.raw, c.fetch, c.now).Observe(t.Context(), mustDID(t, c.id))
		var se *sage.Error
		if !errors.As(err, &se) || se.Code != c.code || se.Detail != "" || se.Cause == nil {
			t.Errorf("%s: %v, want %s", c.name, err, c.code)
		}
	}
}

func TestParseWebBody(t *testing.T) {
	d := mustDID(t, webDID)
	if _, err := parseWebBody([]byte(`{`), d); err == nil {
		t.Error("broken JSON accepted")
	}
	for _, raw := range []string{
		`{"record":{"id":"did:sage:web:other.example:x"},"issued":1}`,
		`{"record":{"id":"` + webDID + `"},"issued":1.5}`,
		`{"record":{"id":"` + webDID + `","keys":[{"name":"k","key":"!!"}]},"issued":1}`,
		`{"record":{"id":"` + webDID + `","keys":[{"name":"k","key":"AA","expires":1.5}]},"issued":1}`,
	} {
		if _, err := parseWebBody([]byte(raw), d); err == nil {
			t.Errorf("%s accepted", raw)
		}
	}
}

func TestNewWebSource(t *testing.T) {
	ok := WebConfig{
		AllowedOrigins:      []string{"https://a.example"},
		AllowedDestinations: []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:443")},
		RootDER:             []byte{1}, SourceID: "s", Now: clock(),
	}
	if _, err := NewWebSource(ok); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*WebConfig){
		"no origins":      func(c *WebConfig) { c.AllowedOrigins = nil },
		"no destinations": func(c *WebConfig) { c.AllowedDestinations = nil },
		"no root":         func(c *WebConfig) { c.RootDER = nil },
		"no clock":        func(c *WebConfig) { c.Now = nil },
		"no source id":    func(c *WebConfig) { c.SourceID = "" },
		"control byte":    func(c *WebConfig) { c.SourceID = "a\nb" },
	} {
		c := ok
		mutate(&c)
		if _, err := NewWebSource(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
