package hpke

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func httpMutate010(m HTTPMessage010, kind string) HTTPMessage010 {
	m.Body = append([]byte{}, m.Body...)
	m.Headers = append([][2]string{}, m.Headers...)
	set := func(k, v string) {
		for i := range m.Headers {
			if m.Headers[i][0] == k {
				m.Headers[i][1] = v
				return
			}
		}
		m.Headers = append(m.Headers, [2]string{k, v})
	}
	switch kind {
	case "request-method":
		m.Method = "GET"
	case "request-target":
		m.Target += "?other=1"
	case "request-authority":
		m.Authority = "other.example"
	case "response-status":
		m.Status = 204
	case "response-status-tamper":
		m.Status = 201
	case "body-digest":
		m.Body = append(m.Body, ' ')
	case "duplicate-signature", "duplicate-input", "duplicate-digest", "duplicate-type", "duplicate-sage":
		k := map[string]string{"duplicate-signature": "signature", "duplicate-input": "signature-input", "duplicate-digest": "content-digest", "duplicate-type": "content-type", "duplicate-sage": "x-sage-did"}[kind]
		for _, p := range m.Headers {
			if p[0] == k {
				m.Headers = append(m.Headers, [2]string{strings.ToUpper(k), p[1]})
				break
			}
		}
	case "content-type":
		set("content-type", "application/json; charset=utf-8")
	case "content-encoding":
		set("content-encoding", "identity")
	case "transfer-encoding":
		set("transfer-encoding", "chunked")
	case "trailer":
		set("trailer", "signature")
	case "content-length":
		set("content-length", "1")
	case "header-control":
		set("extra", "a\nb")
	case "version":
		set("x-sage-version", "1.0")
	case "did":
		set("x-sage-did", "did:sage:web:agent.example:other")
	case "projection":
		set("x-sage-context-id", "other")
	case "metadata":
		set("x-sage-meta-key", "value")
	case "signature":
		set("signature", "sig1=:"+strings.Repeat("A", 86)+"==:")
	case "multiple-signatures":
		for i := range m.Headers {
			if m.Headers[i][0] == "signature" {
				m.Headers[i][1] += ", " + m.Headers[i][1]
			}
		}
	case "noncanonical-base64":
		for i := range m.Headers {
			if m.Headers[i][0] == "signature" {
				m.Headers[i][1] = strings.ReplaceAll(m.Headers[i][1], "=:", ":")
			}
		}
	default:
		for i := range m.Headers {
			if m.Headers[i][0] != "signature-input" {
				continue
			}
			v := m.Headers[i][1]
			switch kind {
			case "unknown-parameter":
				v += ";extra=1"
			case "duplicate-parameter":
				v += `;tag="sage-0.10.0"`
			case "wrong-tag":
				v = strings.Replace(v, "sage-0.10.0", "sage-1.0", 1)
			case "wrong-keyid":
				v = strings.Replace(v, "#signing-1", "#other", 1)
			case "wrong-nonce":
				v = strings.Replace(v, `;nonce="`, `;nonce="A`, 1)
			case "wrong-created":
				v = strings.Replace(v, ";created=100", ";created=101", 1)
			case "wrong-alg":
				v = strings.Replace(v, "ed25519", "rsa-pss-sha512", 1)
			case "coverage":
				v = strings.Replace(v, `"@method"`, `"@path"`, 1)
			}
			m.Headers[i][1] = v
		}
	}
	return m
}
func TestHTTPBinding010(t *testing.T) {
	raw, x := os.ReadFile("testdata/http-session010.json")
	if x != nil {
		t.Fatal(x)
	}
	var f struct{ Cases []string }
	if json.Unmarshal(raw, &f) != nil {
		t.Fatal("fixture")
	}
	for _, kind := range f.Cases {
		t.Run(kind, func(t *testing.T) {
			a, b, c := recordPair010(t, 0)
			ctx := context.Background()
			for _, s := range []*AuthenticatedCompletion010{a, b} {
				if s.BindHTTP("https://agent.example/messages") != nil {
					t.Fatal("bind")
				}
			}
			q, x := a.SealHTTPRequest(ctx, []byte("request"), 300)
			if x != nil {
				t.Fatal(x)
			}
			badRequest := strings.HasPrefix(kind, "request-") || kind == "body-digest" || strings.HasPrefix(kind, "duplicate-") || strings.HasPrefix(kind, "wrong-") || strings.HasPrefix(kind, "content-") || kind == "header-control" || kind == "unknown-parameter" || kind == "transfer-encoding" || kind == "trailer" || kind == "version" || kind == "did" || kind == "projection" || kind == "metadata" || kind == "signature" || kind == "multiple-signatures" || kind == "noncanonical-base64" || kind == "coverage"
			if badRequest {
				if _, x = b.OpenHTTPRequest(ctx, httpMutate010(q, kind)); x == nil {
					t.Fatal("bad HTTP accepted")
				}
				if b.State() != "RESPONSE_SENT" {
					t.Fatal("premature confirmation")
				}
			}
			if kind == "bare-bypass" {
				if _, x = b.OpenRequest(ctx, q.Body); x == nil {
					t.Fatal("bare bypass")
				}
				if _, x = a.SealRequest(ctx, nil, 300); x == nil {
					t.Fatal("bare emission")
				}
			}
			if got, x := b.OpenHTTPRequest(ctx, q); x != nil || !bytes.Equal(got, []byte("request")) {
				t.Fatal("request", x)
			}
			if _, x = b.OpenHTTPRequest(ctx, q); x == nil {
				t.Fatal("replay")
			}
			if kind == "reverse" {
				a, b = b, a
				q, x = a.SealHTTPRequest(ctx, []byte("request"), 300)
				if x != nil {
					t.Fatal(x)
				}
				if _, x = b.OpenHTTPRequest(ctx, q); x != nil {
					t.Fatal(x)
				}
			}
			var w map[string]json.RawMessage
			_ = json.Unmarshal(q.Body, &w)
			id := str010(w, "id")
			data := []byte("result")
			if kind == "empty" {
				data = nil
			}
			code := ""
			if kind == "error" {
				code = "policy_denied"
			}
			r, x := b.SealHTTPResponse(ctx, id, data, code == "", code, 300, 200)
			if x != nil {
				t.Fatal(x)
			}
			if _, x = b.SealHTTPResponse(ctx, id, data, true, "", 300, 200); x == nil {
				t.Fatal("second response")
			}
			if kind == "response-status" || kind == "response-status-tamper" {
				if _, x = a.OpenHTTPResponse(ctx, httpMutate010(r, kind)); x == nil {
					t.Fatal("status")
				}
			}
			if kind == "response-request-signature" {
				saved := a.sent[id].http.signature
				a.sent[id].http.signature += "wrong"
				if _, x = a.OpenHTTPResponse(ctx, r); x == nil {
					t.Fatal("request signature")
				}
				a.sent[id].http.signature = saved
			}
			if kind == "inner-signature" {
				var v map[string]any
				_ = json.Unmarshal(r.Body, &v)
				v["signature"] = strings.Repeat("A", 86)
				bad, x := b.signHTTP010(canon010(v), 200, b.received[id].http)
				if x != nil {
					t.Fatal(x)
				}
				if _, x = a.OpenHTTPResponse(ctx, bad); x == nil {
					t.Fatal("inner signature")
				}
			}
			if kind == "reordered-parameters" {
				for i := range r.Headers {
					if r.Headers[i][0] == "signature-input" {
						v := r.Headers[i][1]
						at := strings.Index(v, ");")
						fields := strings.Split(v[at+2:], ";")
						for l, h := 0, len(fields)-1; l < h; l, h = l+1, h-1 {
							fields[l], fields[h] = fields[h], fields[l]
						}
						r.Headers[i][1] = v[:at+2] + strings.Join(fields, ";")
					}
				}
				h, _ := httpHeaders010(r)
				for i := range r.Headers {
					if r.Headers[i][0] == "signature" {
						r.Headers[i][1] = "sig1=:" + base64.StdEncoding.EncodeToString(ed25519.Sign(b.endpoint.signing, httpBase010(r, h, h["signature-input"][5:], b.received[id].http))) + ":"
					}
				}
			}
			if kind == "store-error" || kind == "store-delay" || kind == "revoke-resp" {
				c.mode = kind
				if _, x = a.OpenHTTPResponse(ctx, r); x == nil {
					t.Fatal("fault accepted")
				}
				c.mode = ""
				if kind != "store-error" {
					if a.State() != "CLOSED" {
						t.Fatal("not closed")
					}
					return
				}
			}
			if kind == "expired" {
				c.utc = 400
				if _, x = a.OpenHTTPResponse(ctx, r); x == nil {
					t.Fatal("expired")
				}
				return
			}
			got, x := a.OpenHTTPResponse(ctx, r)
			if x != nil || got.MessageID != id || got.Success != (code == "") || got.Error != code || !bytes.Equal(got.Data, data) {
				t.Fatal("response", x)
			}
			if _, x = a.OpenHTTPResponse(ctx, r); x == nil {
				t.Fatal("terminal replay")
			}
		})
	}
}

func TestHTTPAdmission010(t *testing.T) {
	a, _, _ := recordPair010(t, 0)
	for _, u := range []string{"http://agent.example/messages", "https://AGENT.example/messages", "https://agent.example:443/messages", "https://user@agent.example/messages", "https://agent.example/messages#fragment", "https://agent.example"} {
		if a.BindHTTP(u) == nil {
			t.Fatalf("accepted endpoint %q", u)
		}
	}
	if a.BindHTTP("https://agent.example/messages") != nil {
		t.Fatal("bind")
	}
	if a.BindHTTP("https://other.example/messages") == nil {
		t.Fatal("rebind")
	}
	q, x := a.SealHTTPRequest(context.Background(), nil, 300)
	if x != nil {
		t.Fatal(x)
	}
	size := 0
	for _, h := range q.Headers {
		size += len(h[0]) + len(h[1]) + 4
	}
	padded := q
	padded.Headers = append(append([][2]string{}, q.Headers...), [2]string{"padding", strings.Repeat("p", 32768-size-len("padding")-4)})
	if _, x = httpHeaders010(padded); x != nil {
		t.Fatal("exact field bound")
	}
	padded.Headers[len(padded.Headers)-1][1] += "p"
	if _, x = httpHeaders010(padded); x == nil {
		t.Fatal("field bound")
	}
	for _, k := range []string{"signature", "signature-input"} {
		m := q
		m.Headers = append([][2]string{}, q.Headers...)
		for i := range m.Headers {
			if m.Headers[i][0] == k {
				m.Headers[i][1] = strings.Repeat("a", 8192)
			}
		}
		if _, x = httpHeaders010(m); x != nil {
			t.Fatal("field size admission")
		}
		for i := range m.Headers {
			if m.Headers[i][0] == k {
				m.Headers[i][1] += "a"
			}
		}
		if _, x = httpHeaders010(m); x == nil {
			t.Fatal("signature field bound")
		}
	}
	h, _ := httpHeaders010(q)
	input := h["signature-input"]
	for _, v := range []string{input + ";created=100", input + ", sig2=()", strings.Replace(input, ";created=100", ";created=0100", 1), strings.Replace(input, ";created=100", ";created=-1", 1), strings.Replace(input, ";created=100", ";created=1.0", 1), strings.Replace(input, `;tag="sage-0.10.0"`, "", 1), strings.Replace(input, `;nonce="`, `;nonce="\`, 1), strings.Replace(input, ";alg=", ";\talg=", 1)} {
		if _, x = httpInput010(v, false); x == nil {
			t.Fatal("non-profile structured fields accepted")
		}
	}
	q.Body = bytes.Repeat([]byte("x"), 32768)
	for i := range q.Headers {
		if q.Headers[i][0] == "content-digest" {
			q.Headers[i][1] = httpDigest010(q.Body)
		}
	}
	if _, x = httpHeaders010(q); x != nil {
		t.Fatal("body admission bound")
	}
	q.Body = append(q.Body, 'x')
	if _, x = httpHeaders010(q); x == nil {
		t.Fatal("body limit")
	}
}

func TestHTTPSerialization010(t *testing.T) {
	raw, err := os.ReadFile("testdata/http-serialization010.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		StructuredFields []struct {
			ID               string
			Response         bool
			Input, Canonical string
			Accept           bool
		} `json:"structured_fields"`
		URIs []struct {
			Target, Authority string
			Accept            bool
		} `json:"uris"`
	}
	if json.Unmarshal(raw, &fixture) != nil {
		t.Fatal("fixture")
	}
	for _, c := range fixture.StructuredFields {
		t.Run(c.ID, func(t *testing.T) {
			_, err := httpInput010(c.Input, c.Response)
			if (err == nil) != c.Accept {
				t.Fatalf("admission: %v", err)
			}
			if c.Accept {
				v, err := canonicalHTTPInput010(c.Input, c.Response)
				if err != nil || v != c.Canonical {
					t.Fatalf("serialization: %q %v", v, err)
				}
			}
		})
	}
	for _, c := range fixture.URIs {
		t.Run(c.Target, func(t *testing.T) {
			authority, err := httpEndpoint010(c.Target)
			if (err == nil) != c.Accept || (c.Accept && authority != c.Authority) {
				t.Fatalf("endpoint: %q %v", authority, err)
			}
		})
	}
}
