package registry011

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// Only a normal owned TLS server is used at runtime. Its test certificate and
// connection never enter a production constructor or operating selection.
// Invalid/fault cases are finite unit data, not network attack producers.
func providerIntegrationOwnedTLS(t *testing.T, request, response string) *tls.Conn {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{SerialNumber: big.NewInt(9), Subject: pkix.Name{CommonName: "owned integration fixture"}, DNSNames: []string{"rpc.sepolia.ethpandaops.io"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateDER}))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
		if err != nil || string(body) != request || r.Method != "POST" || r.Host != "rpc.sepolia.ethpandaops.io" || r.URL.Path != "/" {
			t.Error("owned request bytes changed")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("X-Trace", "a")
		w.Header().Add("X-Trace", "b")
		_, _ = io.WriteString(w, response)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}}
	server.StartTLS()
	t.Cleanup(server.Close)
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})) {
		t.Fatal("test root")
	}
	conn, err := tls.Dial("tcp", server.Listener.Addr().String(), &tls.Config{RootCAs: pool, ServerName: "rpc.sepolia.ethpandaops.io", MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func providerRetainedFirstExchangeFixture(t *testing.T) (*providerIntegration, *providerCustodyStore) {
	t.Helper()
	store, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	header := `{"hash":"` + admissionTestDigest("a") + `","parentHash":"` + admissionTestDigest("b") + `","number":"0x` + strconv.FormatUint(sepoliaDeploymentBlock+1, 16) + `","timestamp":"0x` + strconv.FormatInt(time.Now().Unix(), 16) + `"}`
	response := `{"jsonrpc":"2.0","id":1,"result":` + header + `}`
	c.mu.Lock()
	permit, err := c.consume(c.firstTicket, c.first)
	if err != nil {
		c.mu.Unlock()
		t.Fatal(err)
	}
	if !c.use(permit) {
		c.mu.Unlock()
		t.Fatal("consumption ACK did not authorize its one private transition")
	}
	conn := providerIntegrationOwnedTLS(t, c.first.body, response)
	ctx, cancel := context.WithTimeout(context.Background(), primitive.capture.Timeout)
	defer cancel()
	sample, err := providerIntegrationCapture(ctx, conn, primitive.capture, c.first, permit.meter)
	_ = conn.Close()
	if err != nil {
		c.mu.Unlock()
		t.Fatal(err)
	}
	if !sample.Complete || sample.WireBytes != len(c.first.wire)+len(sample.ResponseWire) || permit.meter.actual[1] != uint64(sample.WireBytes) {
		c.mu.Unlock()
		t.Fatal("both wire directions were not measured")
	}
	_, result, err := c.finishRPC(permit, sample, nil)
	c.restoreMeter(permit.meter)
	c.mu.Unlock()
	if err != nil || string(result) != header {
		t.Fatal("normal owned exchange refused", err)
	}
	if c.firstExchange == "" || c.bootstrap != "" || store.replay.firstExchange != c.firstExchange || c.use(permit) {
		t.Fatal("exchange manufactured a start or a second dispatch")
	}
	return c, store
}

func TestProviderIntegrationOwnedBootstrapConnectsCustodyAndOneExchange(t *testing.T) {
	c, store := providerRetainedFirstExchangeFixture(t)
	if err := c.installBootstrap(); err != nil {
		t.Fatal(err)
	}
	if c.bootstrap == "" || c.selected.Hash != admissionTestDigest("a") || store.replay.start != c.bootstrap || c.ledger.providerStart != c.bootstrap || store.enrollment != c.root.digest || c.root.status != "enrolling" || c.root.checkpoint != "" {
		t.Fatal("bootstrap was not the exact retained first exchange")
	}
	if c.lastExchange != c.firstExchange || c.ledger.current.counts != [3]uint64{2, 2, 2} || store.sequence != 7 {
		t.Fatal("unexpected hidden/free work or duplicate execution")
	}
	// Fixture success remains enrolling, with no actual operating selection,
	// complete chain replay, active Source or message authorization credential.
}
