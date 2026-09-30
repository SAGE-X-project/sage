package registry010

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// WebRegistryAdminMTLSSession010 ties an authenticated administrator to the
// same TLS connection from which a deployment reads one administrative request.
// It permits controller writes only; delegated management state is not defined
// by this reference binding. The deployment must frame and validate the exact
// request bytes before passing them to ApplyWebRegistryWrite010.
type WebRegistryAdminMTLSSession010 struct {
	connection *tls.Conn
	actor      string
	closed     atomic.Bool
}

// AcceptWebRegistryAdminMTLS010 performs a fresh server-side mTLS handshake.
// Both the client CA and leaf-certificate-to-actor pins are trusted deployment
// configuration. An unknown certificate, missing client certificate, or an
// actor outside the Registry authorization identifier bounds is rejected.
func AcceptWebRegistryAdminMTLS010(ctx context.Context, raw net.Conn, serverCertificate tls.Certificate, clientRootDER []byte, actorPins map[[32]byte]string) (*WebRegistryAdminMTLSSession010, error) {
	if ctx == nil || ctx.Err() != nil || raw == nil || len(serverCertificate.Certificate) == 0 || serverCertificate.PrivateKey == nil || len(actorPins) == 0 {
		if raw != nil {
			_ = raw.Close()
		}
		return nil, ErrRejected
	}
	root, err := x509.ParseCertificate(clientRootDER)
	if err != nil || !root.IsCA {
		_ = raw.Close()
		return nil, ErrRejected
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection := tls.Server(raw, &tls.Config{
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
		MinVersion:   tls.VersionTLS12,
	})
	if err := connection.HandshakeContext(bounded); err != nil {
		_ = raw.Close()
		return nil, ErrRejected
	}
	state := connection.ConnectionState()
	if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
		_ = raw.Close()
		return nil, ErrRejected
	}
	actor, found := actorPins[sha256.Sum256(state.PeerCertificates[0].Raw)]
	if !found || !webASCII010(actor, 1, 256) {
		_ = raw.Close()
		return nil, ErrRejected
	}
	return &WebRegistryAdminMTLSSession010{connection: connection, actor: actor}, nil
}

// Read consumes bytes only from the authenticated TLS channel.
func (s *WebRegistryAdminMTLSSession010) Read(p []byte) (int, error) {
	if s == nil || s.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	return s.connection.Read(p)
}

// Write sends a response on the authenticated TLS channel.
func (s *WebRegistryAdminMTLSSession010) Write(p []byte) (int, error) {
	if s == nil || s.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	return s.connection.Write(p)
}

// SetDeadline bounds subsequent administrative request and response I/O.
func (s *WebRegistryAdminMTLSSession010) SetDeadline(deadline time.Time) error {
	if s == nil || s.closed.Load() {
		return io.ErrClosedPipe
	}
	return s.connection.SetDeadline(deadline)
}

// Close retires the authority with its connection.
func (s *WebRegistryAdminMTLSSession010) Close() error {
	if s == nil || s.closed.Swap(true) {
		return nil
	}
	_ = s.connection.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
	return s.connection.Close()
}

// AuthenticatedActor returns the actor pinned to the verified client leaf.
func (s *WebRegistryAdminMTLSSession010) AuthenticatedActor(ctx context.Context) (string, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || s.closed.Load() {
		return "", ErrRejected
	}
	return s.actor, nil
}

// Delegated denies all operator writes until a deployment binds its own
// controller-authorized, version-scoped management state.
func (s *WebRegistryAdminMTLSSession010) Delegated(context.Context, string, string, string, string, string) (bool, error) {
	return false, nil
}

var _ WebRegistryAdminAuthority010 = (*WebRegistryAdminMTLSSession010)(nil)
var _ io.ReadWriteCloser = (*WebRegistryAdminMTLSSession010)(nil)
