package registry010

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/netip"
	"testing"
	"time"
)

func testWebTLSServer010(t *testing.T, hostname string) (netip.AddrPort, []byte) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: hostname},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{hostname}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true,
		IsCA: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, private)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: private}},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				_ = conn.(*tls.Conn).Handshake()
			}()
		}
	}()
	return netip.MustParseAddrPort(listener.Addr().(*net.TCPAddr).AddrPort().String()), der
}

func TestWebRegistryTLSOrigin010(t *testing.T) {
	destination, root := testWebTLSServer010(t, "agents.example.com")
	did := "did:sage:web:agents.example.com:billing-bot"
	allowed := []string{"https://agents.example.com"}
	approved := []netip.AddrPort{destination}
	check := func(name, subject string, origins []string, target netip.AddrPort, destinations []netip.AddrPort, cert []byte, want error) {
		t.Helper()
		err := CheckWebRegistryTLSOrigin010(context.Background(), subject, origins, target, destinations, cert)
		if !errors.Is(err, want) {
			t.Errorf("%s: got %v; want %v", name, err, want)
		}
	}
	check("valid TLS", did, allowed, destination, approved, root, nil)
	check("unapproved origin", did, nil, destination, approved, root, ErrUnreachable)
	check("unapproved destination", did, allowed, destination, nil, root, ErrUnreachable)
	check("unknown root", did, allowed, destination, approved, []byte{1, 2, 3}, ErrUnreachable)
	check("invalid DID", "did:sage:web:127.0.0.1:a", allowed, destination, approved, root, ErrInvalidRecord010)
	wrongDestination, wrongRoot := testWebTLSServer010(t, "other.example.com")
	check("wrong certificate name", did, allowed, wrongDestination, []netip.AddrPort{wrongDestination}, wrongRoot, ErrUnreachable)
}
