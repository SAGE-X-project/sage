package registry011

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func providerTestLimits() ProviderCaptureLimits {
	return ProviderCaptureLimits{time.Second * 5, 4096, 4096, 16384, 64}
}

func TestProviderCaptureConfiguration(t *testing.T) {
	good := providerTestLimits()
	if !good.valid() {
		t.Fatal("valid limits")
	}
	cases := []ProviderCaptureLimits{
		{}, {0, 4096, 4096, 16384, 64}, {6 * time.Minute, 4096, 4096, 16384, 64},
		{time.Second, 0, 4096, 16384, 64}, {time.Second, 65537, 4096, 1 << 20, 64},
		{time.Second, 4096, 0, 16384, 64}, {time.Second, 4096, 8<<20 + 1, 16 << 20, 64},
		{time.Second, 4096, 4096, 4096, 64}, {time.Second, 4096, 4096, 16<<20 + 1, 64},
		{time.Second, 4096, 4096, 16384, 0}, {time.Second, 4096, 4096, 16384, 129},
	}
	for _, l := range cases {
		if l.valid() {
			t.Fatal("invalid limits accepted", l)
		}
		if _, e := CaptureSepoliaProviderFinality(context.Background(), l); !errors.Is(e, ErrConfiguration) {
			t.Fatal(e)
		}
	}
	if _, e := CaptureSepoliaProviderFinality(nil, good); !errors.Is(e, ErrConfiguration) {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := CaptureSepoliaProviderFinality(ctx, good); !errors.Is(e, ErrProviderCapture) {
		t.Fatal(e)
	}
}

func providerTestTLSState() tls.ConnectionState {
	return tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13, ServerName: "rpc.sepolia.ethpandaops.io", NegotiatedProtocol: "http/1.1", VerifiedChains: [][]*x509.Certificate{{{Raw: []byte("owned fixed certificate bytes")}}}}
}

func TestProviderCaptureTLSMetadata(t *testing.T) {
	good := providerTestTLSState()
	a, e := providerTLSIdentity(good)
	if e != nil || len(a) != 1 || len(a[0]) != 1 || len(a[0][0]) != 66 {
		t.Fatal(a, e)
	}
	good.NegotiatedProtocol = ""
	if _, e = providerTLSIdentity(good); e != nil {
		t.Fatal(e)
	}
	for _, change := range []func(*tls.ConnectionState){
		func(s *tls.ConnectionState) { s.HandshakeComplete = false }, func(s *tls.ConnectionState) { s.Version = tls.VersionTLS11 },
		func(s *tls.ConnectionState) { s.ServerName = "" }, func(s *tls.ConnectionState) { s.NegotiatedProtocol = "h2" },
		func(s *tls.ConnectionState) { s.VerifiedChains = nil }, func(s *tls.ConnectionState) { s.VerifiedChains = [][]*x509.Certificate{{}} },
		func(s *tls.ConnectionState) { s.VerifiedChains = [][]*x509.Certificate{{nil}} }, func(s *tls.ConnectionState) { s.VerifiedChains = [][]*x509.Certificate{{{}}} },
	} {
		s := providerTestTLSState()
		change(&s)
		if _, e := providerTLSIdentity(s); !errors.Is(e, ErrProviderCapture) {
			t.Fatal(e)
		}
	}
}

// All negative HTTP cases below are finite inert byte fixtures. No malformed
// request/response is emitted over a network and no attack producer is defined.
func TestProviderCaptureFixedHTTP(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{"number":"0x1"}}`
	head := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nX-Trace: a\r\nX-Trace: b\r\nContent-Length: " + itoaProvider(len(body)) + "\r\n\r\n"
	parse := func(data string, l ProviderCaptureLimits) error {
		_, _, _, _, e := readProviderHTTP(bufio.NewReader(strings.NewReader(data)), l)
		return e
	}
	rh, h, rb, result, e := readProviderHTTP(bufio.NewReader(strings.NewReader(head+body)), providerTestLimits())
	if e != nil || string(rh) != head || string(rb) != body || len(h) != 4 || h[1].Value != "a" || h[2].Value != "b" || string(result) != `{"number":"0x1"}` {
		t.Fatal("raw preservation", e, h)
	}
	closeHead := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Encoding: identity\r\n\r\n"
	if e = parse(closeHead+body, providerTestLimits()); e != nil {
		t.Fatal(e)
	}
	chunkHead := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n"
	chunkBody := hexProvider(len(body)) + "\r\n" + body + "\r\n0\r\n\r\n"
	if e = parse(chunkHead+chunkBody, providerTestLimits()); e != nil {
		t.Fatal(e)
	}
	bad := []string{
		"", "HTTP/1.1 200 OK\r\n", strings.Replace(head+body, "200 OK", "302 Moved", 1), strings.Replace(head+body, "HTTP/1.1", "HTTP/1.0", 1),
		strings.Replace(head+body, "200 OK", "200 O\nK", 1), strings.Replace(head+body, "application/json", "text/plain", 1),
		strings.Replace(head+body, "X-Trace: a", "Content-Encoding: gzip", 1), strings.Replace(head+body, "X-Trace: a", "Content-Encoding:", 1),
		strings.Replace(head+body, "X-Trace: a", "Content-Type: application/json", 1), strings.Replace(head+body, "X-Trace: a", "content-length: 1", 1),
		strings.Replace(head+body, "X-Trace: a", "Transfer-Encoding: chunked", 1), strings.Replace(head+body, "X-Trace: a", "Trailer: X-Test", 1),
		strings.Replace(head+body, "X-Trace: a", " X-Trace: a", 1), strings.Replace(head+body, "X-Trace: a", "X-Trace a", 1),
		strings.Replace(head+body, "X-Trace: a", ": a", 1), strings.Replace(head+body, "X-Trace: a", "X-Trace: a\x00", 1),
		strings.Replace(head+body, "X-Trace: a", "X-Trace: a\x7f", 1), strings.Replace(head+body, "X-Trace: a", "X-Trace: \xff", 1),
		strings.Replace(head+body, "Content-Length: ", "Content-Length: +", 1), strings.Replace(head+body, "Content-Length: ", "Content-Length: 0", 1),
		strings.Replace(head+body, "Content-Length: "+itoaProvider(len(body)), "Content-Length: 4097", 1),
		head + body[:len(body)-1], head + body + "extra", chunkHead + chunkBody + "extra", strings.Replace(head+body, `"id":1`, `"id":2`, 1),
		strings.Replace(chunkHead+chunkBody, "chunked", "gzip", 1), strings.Replace(chunkHead+chunkBody, "0\r\n\r\n", "0\r\nX-Test: data\r\n\r\n", 1),
		closeHead + strings.Repeat(" ", 4097), "HTTP/1.1 200 " + strings.Repeat("a", 257) + "\r\n\r\n",
		"HTTP/1.1 200 OK\r\n" + strings.Repeat("X: a\r\n", 129) + "\r\n",
		"HTTP/1.1 200 OK\r\n" + strings.Repeat("a", 257) + ": a\r\n\r\n",
	}
	for i, raw := range bad {
		if e = parse(raw, providerTestLimits()); e == nil {
			t.Fatal("accepted fixed invalid fixture", i)
		}
	}
	l := providerTestLimits()
	l.HeaderBytes = len(head) - 1
	if e = parse(head+body, l); e == nil {
		t.Fatal("header cap")
	}
	l.HeaderBytes = len(head)
	if e = parse(head+body, l); e != nil {
		t.Fatal("exact header cap", e)
	}
	l = providerTestLimits()
	l.BodyBytes = len(body) - 1
	if e = parse(head+body, l); e == nil {
		t.Fatal("body cap")
	}
}

func TestProviderCaptureHeadAndInput(t *testing.T) {
	if _, e := providerHeaderFields([]byte("HTTP/1.1 200 OK\r\n")); e == nil {
		t.Fatal("head terminator")
	}
	if _, e := providerHeaderFields([]byte("\xff\r\n\r\n")); e == nil {
		t.Fatal("head UTF8")
	}
	if _, e := readProviderHead(bufio.NewReader(strings.NewReader("unfinished")), 4); e == nil {
		t.Fatal("bounded incomplete header")
	}
	if !strings.Contains(string(providerRequestHead(3)), "Content-Length: 3\r\n") {
		t.Fatal("request length")
	}
}

func itoaProvider(v int) string { return strconv.FormatInt(int64(v), 10) }
func hexProvider(v int) string  { return strconv.FormatInt(int64(v), 16) }

// This runtime owns both endpoints and a newly generated test-only certificate.
// It exercises ordinary TLS verification, not a production endpoint override.
func TestProviderCaptureOwnedTLS(t *testing.T) {
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "owned test only"}, DNSNames: []string{"rpc.sepolia.ethpandaops.io"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	keyDER, e := x509.MarshalECPrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	cert, e := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	if e != nil {
		t.Fatal(e)
	}
	var connections []*tls.Conn
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1024))
		if err != nil {
			t.Error(err)
		}
		if r.Method != "POST" || r.Host != "rpc.sepolia.ethpandaops.io" || r.URL.Path != "/" || !strings.Contains(string(b), `"finalized"`) {
			t.Error("unexpected fixed read")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("X-Trace", "a")
		w.Header().Add("X-Trace", "b")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"number":"0x1"}}`)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}}
	server.StartTLS()
	defer func() {
		for _, c := range connections {
			_ = c.Close()
		}
		server.Close()
	}()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	dial := func() *tls.Conn {
		c, err := tls.Dial("tcp", server.Listener.Addr().String(), &tls.Config{RootCAs: pool, ServerName: "rpc.sepolia.ethpandaops.io", MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}})
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, c)
		return c
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sample, e := captureProviderTLS(ctx, dial(), providerTestLimits())
	if e != nil {
		t.Fatal(e)
	}
	if !sample.Complete || sample.CaptureError != "" || sample.WireBytes != len(sample.RequestHead)+len(sample.RequestBody)+len(sample.ResponseWire) || sample.Endpoint != SepoliaRPCURL || sample.HTTPVersion != "HTTP/1.1" || len(sample.VerifiedChains) == 0 || sample.WireBytes <= len(sample.ResponseBody) || sample.Finished.Before(sample.Started) {
		t.Fatal("metadata", sample)
	}
	if string(sample.Result) != `{"number":"0x1"}` || !strings.Contains(string(sample.ResponseHead), "X-Trace: a\r\nX-Trace: b\r\n") {
		t.Fatal("original fields")
	}
	if _, e = captureProviderTLS(context.Background(), dial(), providerTestLimits()); e == nil {
		t.Fatal("no deadline")
	}
	l := providerTestLimits()
	l.WireBytes = 1
	if _, e = captureProviderTLS(ctx, dial(), l); e == nil {
		t.Fatal("request cap")
	}
	l = providerTestLimits()
	l.HeaderBytes = 256
	l.WireBytes = 330
	if _, e = captureProviderTLS(ctx, dial(), l); e == nil {
		t.Fatal("short header wire cap")
	}
	l = providerTestLimits()
	l.HeaderBytes = 256
	l.WireBytes = sample.WireBytes - 1
	if _, e = captureProviderTLS(ctx, dial(), l); e == nil {
		t.Fatal("aggregate wire cap")
	}
	c := dial()
	_ = c.Close()
	if _, e = captureProviderTLS(ctx, c, providerTestLimits()); e == nil {
		t.Fatal("closed connection")
	}
}
