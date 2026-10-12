package didsource010

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/consumer"
	"github.com/sage-x-project/did-go/producer"
	"github.com/sage-x-project/did-go/resolution"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const eipReg = "eip155:11155111:0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const eipID = "did:sage:" + eipReg + ":billing-bot"

type didCompleteReader struct {
	snapshot registry010.RecordSnapshot
	err      error
	calls    int
}

func (r *didCompleteReader) ReadRecord(context.Context, string) (registry010.RecordSnapshot, error) {
	r.calls++
	return r.snapshot, r.err
}

type didCompleteClock struct{ err error }

func (c *didCompleteClock) Now() (registry010.Stamp, error) {
	return registry010.Stamp{MonoMS: 100, Unix: 101}, c.err
}
func eipBody(t *testing.T) []byte {
	t.Helper()
	r := signedRecord(t)
	r["id"] = eipID
	r["controller"] = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	priv := ed25519.NewKeyFromSeed(make([]byte, 32))
	for _, value := range r["keys"].([]any) {
		k := value.(map[string]any)
		pub, err := base64.RawURLEncoding.DecodeString(k["key"].(string))
		if err != nil {
			t.Fatal(err)
		}
		challenge, err := registry010.PoPChallenge010(eipReg, "billing-bot", k["name"].(string), k["alg"].(string), pub)
		if err != nil {
			t.Fatal(err)
		}
		k["proof"] = map[string]any{"signer": eipID + "#a-ed", "value": base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, challenge))}
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func newDIDComplete(t *testing.T) (*EIP155Source, *didCompleteReader, *didCompleteClock) {
	t.Helper()
	hash := strings.Repeat("a", 64)
	r := &didCompleteReader{snapshot: registry010.RecordSnapshot{Source: "trusted-local", Registry: eipReg, Network: "sepolia", Record: eipBody(t), BlockHash: hash, RecordBlockHash: hash, KeysBlockHash: hash, ServicesBlockHash: hash, AcquiredMS: 100, Ready: true, Validated: true, Finalized: true}}
	j, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "history"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})
	c := &didCompleteClock{}
	s, err := NewEIP155Source(registry010.Config{Source: "trusted-local", Registry: eipReg, Network: "sepolia", Blockchain: true}, r, c, j)
	if err != nil {
		t.Fatal(err)
	}
	return s, r, c
}
func TestEIP155SourceRecord(t *testing.T) {
	s, r, _ := newDIDComplete(t)
	rec, err := s.Observe(t.Context(), mustDID(t, eipID))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Controller != "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" || rec.Version != "3" || len(rec.Keys) != 2 || len(rec.Services) != 1 || rec.Observation.Issued != nil || rec.Observation.BlockHash != r.snapshot.BlockHash || rec.Observation.Source != "trusted-local" || rec.Observation.Time != 101 {
		t.Fatalf("bad record %+v", rec)
	}
	rec.Keys[0].Material[0] ^= 1
	again, err := s.Observe(t.Context(), mustDID(t, eipID))
	if err != nil {
		t.Fatal(err)
	}
	if r.calls != 2 || again.Keys[0].Material[0] == rec.Keys[0].Material[0] {
		t.Fatal("aliased or cached record")
	}
}
func TestEIP155SourceRefusals(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		edit     func(*didCompleteReader, *didCompleteClock)
		code     sage.Code
	}{
		{"unsupported", webDID, func(*didCompleteReader, *didCompleteClock) {}, sage.CodeIDUnknownKind},
		{"malformed", eipID + ":extra", func(*didCompleteReader, *didCompleteClock) {}, sage.CodeIDMalformed},
		{"unknown", "did:sage:dns:example.com:a", func(*didCompleteReader, *didCompleteClock) {}, sage.CodeIDUnknownKind},
		{"source unavailable", eipID, func(r *didCompleteReader, _ *didCompleteClock) { r.err = errors.New("offline") }, sage.CodeRecordUnreachable},
		{"clock unavailable", eipID, func(_ *didCompleteReader, c *didCompleteClock) { c.err = errors.New("lost") }, sage.CodeRecordUnreachable},
		{"mixed block", eipID, func(r *didCompleteReader, _ *didCompleteClock) { r.snapshot.ServicesBlockHash = "" }, sage.CodeRecordStale},
		{"bad record", eipID, func(r *didCompleteReader, _ *didCompleteClock) { r.snapshot.Record = []byte("{") }, sage.CodeRecordInvalid},
		{"too large", eipID, func(r *didCompleteReader, _ *didCompleteClock) { r.snapshot.Record = make([]byte, 65537) }, sage.CodeSizeExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r, c := newDIDComplete(t)
			tc.edit(r, c)
			rec, err := s.Observe(t.Context(), mustDID(t, tc.id))
			var se *sage.Error
			if !errors.As(err, &se) || se.Code != tc.code || se.Detail != "" || rec.DID.String() != "" {
				t.Fatalf("got %+v %v", rec, err)
			}
			if tc.id != eipID && r.calls != 0 {
				t.Fatal("preflight performed read")
			}
		})
	}
	if _, err := NewEIP155Source(registry010.Config{}, nil, nil, nil); err == nil {
		t.Fatal("missing configuration accepted")
	}
}
func TestEIP155SafeTLSRuntime(t *testing.T) {
	s, r, _ := newDIDComplete(t)
	resolver, err := sage.NewResolver(s, consumer.New(), producer.New())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(sage.NewHandler(resolver, "/identifiers/"))
	defer server.Close()
	for _, mt := range []string{"application/did+json", "application/did"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/identifiers/"+eipID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept", mt)
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("read %v close %v", err, closeErr)
		}
		if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != mt || !strings.Contains(string(data), `"id":"`+eipID+`"`) || !strings.Contains(string(data), `"authentication":["`+eipID+`#a-ed"]`) || !strings.Contains(string(data), `"keyAgreement":["`+eipID+`#b-kem"]`) || !strings.Contains(string(data), `"type":"A2AInterface"`) {
			t.Fatalf("%s status %d: %s", mt, resp.StatusCode, data)
		}
	}
	if r.calls != 2 {
		t.Fatalf("want two fresh reads, got %d", r.calls)
	}
	// Registry failure produces a diagnostic and no document through did-go.
	r.snapshot.Finalized = false
	res, code := resolver.ResolveWithCode(t.Context(), mustDID(t, eipID), resolution.ResolutionOptions{})
	if code != sage.CodeRecordStale || res.Document != nil || res.ResolutionMetadata.Error == nil {
		t.Fatalf("failure %+v %q", res, code)
	}
}
func TestEIP155ExpiredAndInactiveProjection(t *testing.T) {
	for _, state := range []string{"active", "created", "deactivated"} {
		t.Run(state, func(t *testing.T) {
			s, r, _ := newDIDComplete(t)
			var rec map[string]any
			if err := json.Unmarshal(r.snapshot.Record, &rec); err != nil {
				t.Fatal(err)
			}
			rec["state"] = state
			for _, k := range rec["keys"].([]any) {
				k.(map[string]any)["expires"] = 100
			}
			raw, err := json.Marshal(rec)
			if err != nil {
				t.Fatal(err)
			}
			r.snapshot.Record = raw
			value, err := s.Observe(t.Context(), mustDID(t, eipID))
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []sage.Projection{sage.Core10, sage.DID11} {
				doc, se := sage.Document(value, p)
				if se != nil {
					t.Fatal(se)
				}
				data, se := sage.Produce(value, p, producer.New())
				if se != nil || doc == nil {
					t.Fatalf("projection %v", se)
				}
				for _, field := range []string{`"authentication"`, `"assertionMethod"`, `"keyAgreement"`} {
					if strings.Contains(string(data), field) {
						t.Fatalf("expired/inactive relationship: %s", data)
					}
				}
			}
		})
	}
}

func TestCompleteRecordParsing(t *testing.T) {
	if _, err := parseRecordBody([]byte("{"), mustDID(t, eipID)); err == nil {
		t.Fatal("bad JSON accepted")
	}
	for _, bad := range []string{"bad", "1.5", "-1", "9007199254740992", "NaN", "+Inf", "-0", "-0.0", "100.0000000000000000001", "1e100001", "1e-100001", "1e999999999999999999999", "100 1"} {
		if _, err := recordInteger(bad); err == nil {
			t.Fatalf("expiry %s accepted", bad)
		}
	}
	for _, literal := range []string{"100.0", "1e2", "100"} {
		t.Run(literal, func(t *testing.T) {
			s, r, _ := newDIDComplete(t)
			r.snapshot.Record = []byte(strings.Replace(string(r.snapshot.Record), `"expires":4000000000`, `"expires":`+literal, 1))
			rec, err := s.Observe(t.Context(), mustDID(t, eipID))
			if err != nil || rec.Keys[1].Expires == nil || *rec.Keys[1].Expires != 100 {
				t.Fatalf("integer %s: %+v %v", literal, rec, err)
			}
		})
	}
}

func TestWebRecordIntegerSpellings(t *testing.T) {
	for _, literal := range []string{"100.0", "1e2", "100"} {
		t.Run(literal, func(t *testing.T) {
			raw := body(t, signedRecord(t))
			raw = []byte(strings.ReplaceAll(string(raw), `"issued":100`, `"issued":`+literal))
			raw = []byte(strings.ReplaceAll(string(raw), `"expires":4000000000`, `"expires":`+literal))
			s := newTestSource(t, raw, nil, clock(101, 102))
			rec, err := s.Observe(t.Context(), mustDID(t, webDID))
			if err != nil || rec.Observation.Issued == nil || *rec.Observation.Issued != 100 || rec.Keys[1].Expires == nil || *rec.Keys[1].Expires != 100 {
				t.Fatalf("web integer %s: %+v %v", literal, rec, err)
			}
		})
	}
}
