package registry010

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/netip"
	"strings"
	"time"
)

// CheckWebRegistryTLSOrigin010 establishes a new TLS connection to an exact,
// locally approved IP destination and authenticates the web DID's DNS name
// against an explicitly trusted certificate root. It does not send an HTTP
// request or validate a Registry response, so success is not an observation
// or authorization decision.
func CheckWebRegistryTLSOrigin010(ctx context.Context, did string, allowedOrigins []string, destination netip.AddrPort, allowedDestinations []netip.AddrPort, rootDER []byte) error {
	if ctx == nil {
		return ErrUnreachable
	}
	if _, err := WebRegistryRequestURL010(did, allowedOrigins); err != nil {
		return err
	}
	approved := false
	for _, candidate := range allowedDestinations {
		if candidate == destination {
			approved = true
			break
		}
	}
	if !destination.IsValid() || destination.Port() == 0 || destination.Addr().IsUnspecified() || !approved {
		return ErrUnreachable
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil || !root.IsCA {
		return ErrUnreachable
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	rest := strings.TrimPrefix(did, "did:sage:web:")
	domain, _, _ := strings.Cut(rest, ":")
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	dialer := &net.Dialer{}
	raw, err := dialer.DialContext(bounded, "tcp", destination.String())
	if err != nil {
		return ErrUnreachable
	}
	connection := tls.Client(raw, &tls.Config{
		ServerName: domain,
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	})
	defer func() { _ = connection.Close() }()
	if err := connection.HandshakeContext(bounded); err != nil || len(connection.ConnectionState().VerifiedChains) == 0 {
		return ErrUnreachable
	}
	return nil
}
