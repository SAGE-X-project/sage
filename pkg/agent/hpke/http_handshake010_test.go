package hpke

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const handshakeTarget010 = "https://localhost:8443/messages"

func TestHTTPHandshake010(t *testing.T) {
	raw, x := os.ReadFile("testdata/http-handshake010.json")
	if x != nil {
		t.Fatal(x)
	}
	var f struct{ Cases []string }
	if json.Unmarshal(raw, &f) != nil {
		t.Fatal("fixture")
	}
	for _, kind := range f.Cases {
		t.Run(kind, func(t *testing.T) {
			a, b, c := completionPair(t)
			ctx := context.Background()
			if a.BindHTTP(handshakeTarget010) != nil || b.BindHTTP(handshakeTarget010) != nil {
				t.Fatal("bind")
			}
			p, q, x := a.StartHTTP(ctx, completionBob, completionBob+"#signing-1", 300)
			if x != nil {
				t.Fatal(x)
			}
			defer p.Close()
			if kind == "bare-bypass" {
				if _, _, x = a.Start(ctx, completionBob, completionBob+"#signing-1", 300); x == nil {
					t.Fatal("bare start")
				}
				if _, _, x = b.Respond(ctx, q.Body, 300); x == nil {
					t.Fatal("bare respond")
				}
			}
			if strings.HasPrefix(kind, "request-") {
				name := map[string]string{"request-signature": "signature", "request-digest": "body-digest", "request-duplicate": "duplicate-signature"}[kind]
				if name == "" {
					name = kind
				}
				if _, _, x = b.RespondHTTP(ctx, httpMutate010(q, name), 300); x == nil {
					t.Fatal("invalid request")
				}
			}
			wire, x := EncodeHTTP010(q, handshakeTarget010)
			if x != nil {
				t.Fatal(x)
			}
			parsed, x := ParseHTTP010(wire, handshakeTarget010, false)
			if x != nil {
				t.Fatal(x)
			}
			server, r, x := b.RespondHTTP(ctx, parsed, 300)
			if x != nil {
				t.Fatal(x)
			}
			defer server.Close()
			if kind == "first-record" {
				if _, x := server.SealHTTPRequest(ctx, nil, 300); x == nil {
					t.Fatal("provisional emission")
				}
			}
			if kind == "duplicate-initiation" {
				if s, _, x := b.RespondHTTP(ctx, q, 300); x == nil {
					s.Close()
					t.Fatal("duplicate initiation")
				}
			}
			if kind == "bare-bypass" {
				if _, x = p.Complete(ctx, r.Body); x == nil {
					t.Fatal("bare complete")
				}
			}
			reject := false
			switch kind {
			case "response-signature":
				r = httpMutate010(r, "signature")
				reject = true
			case "response-status":
				r = httpMutate010(r, "response-status-tamper")
				reject = true
			case "response-digest":
				r = httpMutate010(r, "body-digest")
				reject = true
			case "response-request-signature":
				p.http.signature += "different"
				reject = true
			case "inner-completion":
				r.Body = changedCompletion(t, q.Body, r.Body, "inner-signature")
				r, x = signHTTP010(b, b.httpTarget, b.httpAuthority, r.Body, 200, p.http)
				if x != nil {
					t.Fatal(x)
				}
				reject = true
			case "revoke-resp", "store-delay", "store-error":
				c.mode = kind
				reject = true
			case "expired":
				c.utc = 400
				reject = true
			case "closed-pending":
				p.Close()
				reject = true
			}
			client, x := p.CompleteHTTP(ctx, r)
			if reject {
				if x == nil {
					client.Close()
					t.Fatal("invalid completion")
				}
				if p.State() != "CLOSED" {
					t.Fatal("pending retained")
				}
				return
			}
			if x != nil {
				t.Fatal(x)
			}
			defer client.Close()
			if _, x = p.CompleteHTTP(ctx, r); x == nil {
				t.Fatal("duplicate completion")
			}
			if _, x = client.SealRequest(ctx, nil, 300); x == nil {
				t.Fatal("record downgrade")
			}
			request, x := client.SealHTTPRequest(ctx, []byte("request"), 300)
			if x != nil {
				t.Fatal(x)
			}
			if _, x = server.OpenRequest(ctx, request.Body); x == nil {
				t.Fatal("receive downgrade")
			}
			data, x := server.OpenHTTPRequest(ctx, request)
			if x != nil || !bytes.Equal(data, []byte("request")) || server.State() != "ESTABLISHED" {
				t.Fatal("first record", x)
			}
			var w map[string]json.RawMessage
			_ = json.Unmarshal(request.Body, &w)
			code := ""
			if kind == "error-response" {
				code = "operation_failed"
			}
			response, x := server.SealHTTPResponse(ctx, str010(w, "id"), []byte("result"), code == "", code, 300, 200)
			if x != nil {
				t.Fatal(x)
			}
			wire, x = EncodeHTTP010(response, handshakeTarget010)
			if x != nil {
				t.Fatal(x)
			}
			parsed, x = ParseHTTP010(wire, handshakeTarget010, true)
			if x != nil {
				t.Fatal(x)
			}
			result, x := client.OpenHTTPResponse(ctx, parsed)
			if x != nil || result.Success != (code == "") || string(result.Data) != "result" {
				t.Fatal("response", x)
			}
		})
	}
}
func TestHTTPFraming010(t *testing.T) {
	a, _, _ := completionPair(t)
	if a.BindHTTP(handshakeTarget010) != nil {
		t.Fatal("bind")
	}
	p, m, x := a.StartHTTP(context.Background(), completionBob, completionBob+"#signing-1", 300)
	if x != nil {
		t.Fatal(x)
	}
	defer p.Close()
	raw, x := EncodeHTTP010(m, handshakeTarget010)
	if x != nil {
		t.Fatal(x)
	}
	original := string(raw)
	parts := strings.SplitN(original, "\r\n\r\n", 2)
	head, body := parts[0], parts[1]
	var length string
	for _, l := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(l, "Content-Length:") {
			length = l
		}
	}
	first := strings.SplitN(head, "\r\n", 2)[0]
	padding := 32768 - (len(head) - len(first)) - len("Padding:") - 2
	exact := head + "\r\nPadding:" + strings.Repeat(" ", padding)
	if _, x := ParseHTTP010([]byte(exact+"\r\n\r\n"+body), handshakeTarget010, false); x != nil {
		t.Fatal("exact raw field bound", x)
	}
	if _, x := ParseHTTP010([]byte(exact+" "+"\r\n\r\n"+body), handshakeTarget010, false); x == nil {
		t.Fatal("raw whitespace limit")
	}
	cases := map[string]string{"duplicate-length": head + "\r\n" + length, "conflicting-length": head + "\r\nContent-Length: 1", "missing-length": strings.Replace(head, "\r\n"+length, "", 1), "noncanonical-length": strings.Replace(head, "Content-Length: ", "Content-Length: 0", 1), "transfer-encoding": head + "\r\nTransfer-Encoding: chunked", "trailer": head + "\r\nTrailer: signature", "content-encoding": head + "\r\nContent-Encoding: identity", "duplicate-host": head + "\r\nHOST: localhost:8443", "missing-host": strings.Replace(head, "\r\nHost: localhost:8443", "", 1), "wrong-host": strings.Replace(head, "Host: localhost:8443", "Host: other.example", 1), "absolute-target": strings.Replace(head, "POST /messages", "POST "+handshakeTarget010, 1), "wrong-path": strings.Replace(head, "POST /messages", "POST /other", 1), "http-version": strings.Replace(head, "HTTP/1.1", "HTTP/1.0", 1), "folded-field": head + "\r\n continued", "space-name": head + "\r\nName : value", "control-value": head + "\r\nName: a\tb", "upgrade": head + "\r\nUpgrade: websocket", "expect": head + "\r\nExpect: 100-continue", "keep-alive": strings.Replace(head, "Connection: close", "Connection: keep-alive", 1)}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			if _, x := ParseHTTP010([]byte(h+"\r\n\r\n"+body), handshakeTarget010, false); x == nil {
				t.Fatal("framing accepted")
			}
		})
	}
	for _, bad := range []string{original[:len(original)-1], original + "x", strings.ReplaceAll(original, "\r\n", "\n")} {
		if _, x := ParseHTTP010([]byte(bad), handshakeTarget010, false); x == nil {
			t.Fatal("length/line failure")
		}
	}
	duplicate := m
	duplicate.Headers = append(append([][2]string{}, m.Headers...), m.Headers[5])
	if _, x = EncodeHTTP010(duplicate, handshakeTarget010); x == nil {
		t.Fatal("duplicate signature")
	}
	injection := m
	injection.Headers = append(append([][2]string{}, m.Headers...), [2]string{"extra", "value\r\nInjected: yes"})
	if _, x = EncodeHTTP010(injection, handshakeTarget010); x == nil {
		t.Fatal("header injection")
	}
}
