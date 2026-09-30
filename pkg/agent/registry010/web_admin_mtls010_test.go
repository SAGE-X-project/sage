package registry010

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"testing"
	"time"
)

func testWebAdminCertificates010(t *testing.T) (tls.Certificate, tls.Certificate, tls.Certificate, []byte, []byte) {
	t.Helper()
	caPublic, caPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "local admin CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true, IsCA: true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, ca, ca, caPublic, caPrivate)
	if err != nil {
		t.Fatal(err)
	}
	issue := func(serial int64, name string, usage x509.ExtKeyUsage) (tls.Certificate, []byte) {
		t.Helper()
		public, private, keyErr := ed25519.GenerateKey(rand.Reader)
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		leaf := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: name},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
			DNSNames: []string{name}, KeyUsage: x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{usage}, BasicConstraintsValid: true,
		}
		der, certErr := x509.CreateCertificate(rand.Reader, leaf, ca, public, caPrivate)
		if certErr != nil {
			t.Fatal(certErr)
		}
		return tls.Certificate{Certificate: [][]byte{der, rootDER}, PrivateKey: private}, der
	}
	server, _ := issue(2, "admin.example.com", x509.ExtKeyUsageServerAuth)
	client, clientDER := issue(3, "operator.example.com", x509.ExtKeyUsageClientAuth)
	other, _ := issue(4, "other.example.com", x509.ExtKeyUsageClientAuth)
	return server, client, other, rootDER, clientDER
}

func testWebAdminHandshake010(t *testing.T, server, client tls.Certificate, rootDER []byte, pins map[[32]byte]string) (*WebRegistryAdminMTLSSession010, *tls.Conn, error) {
	t.Helper()
	serverRaw, clientRaw := net.Pipe()
	t.Cleanup(func() { _ = serverRaw.Close(); _ = clientRaw.Close() })
	result := make(chan struct {
		session *WebRegistryAdminMTLSSession010
		err     error
	}, 1)
	go func() {
		session, err := AcceptWebRegistryAdminMTLS010(context.Background(), serverRaw, server, rootDER, pins)
		if err != nil {
			_ = serverRaw.Close()
		}
		result <- struct {
			session *WebRegistryAdminMTLSSession010
			err     error
		}{session, err}
	}()
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	clientTLS := tls.Client(clientRaw, &tls.Config{
		RootCAs: pool, ServerName: "admin.example.com", MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{client},
	})
	_ = clientTLS.Handshake()
	select {
	case outcome := <-result:
		return outcome.session, clientTLS, outcome.err
	case <-time.After(6 * time.Second):
		t.Fatal("mTLS handshake exceeded deadline")
		return nil, nil, nil
	}
}

func TestWebRegistryAdminMTLS010(t *testing.T) {
	server, client, other, rootDER, clientDER := testWebAdminCertificates010(t)
	pins := map[[32]byte]string{sha256.Sum256(clientDER): "operator"}
	session, peer, err := testWebAdminHandshake010(t, server, client, rootDER, pins)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close(); _ = peer.Close() })
	actor, err := session.AuthenticatedActor(context.Background())
	if err != nil || actor != "operator" {
		t.Fatalf("authenticated actor: %q, %v", actor, err)
	}
	go func() { _, _ = peer.Write([]byte("write")) }()
	message := make([]byte, 5)
	if _, err := io.ReadFull(session, message); err != nil || string(message) != "write" {
		t.Fatalf("TLS request bytes: %q, %v", message, err)
	}
	if delegated, err := session.Delegated(context.Background(), "operator", "assistant", "did", "activate", "1"); err != nil || delegated {
		t.Fatal("unconfigured delegation was accepted")
	}
	_ = session.Close()
	if _, err := session.AuthenticatedActor(context.Background()); err == nil {
		t.Fatal("closed session retained authority")
	}

	if _, _, err := testWebAdminHandshake010(t, server, client, rootDER, nil); err == nil {
		t.Fatal("missing certificate pin accepted")
	}
	if _, _, err := testWebAdminHandshake010(t, server, client, rootDER,
		map[[32]byte]string{sha256.Sum256(clientDER): ""}); err == nil {
		t.Fatal("empty actor accepted")
	}
	if _, _, err := testWebAdminHandshake010(t, server, tls.Certificate{}, rootDER, pins); err == nil {
		t.Fatal("missing client certificate accepted")
	}
	if _, _, err := testWebAdminHandshake010(t, server, other, rootDER, pins); err == nil {
		t.Fatal("unrecognized client certificate accepted")
	}
	if _, err := AcceptWebRegistryAdminMTLS010(context.Background(), nil, server, rootDER, pins); err == nil {
		t.Fatal("missing transport accepted")
	}
}
