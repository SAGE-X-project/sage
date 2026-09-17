package hpke

import (
	"context"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// BindHTTP permanently selects HTTP carriage before any handshake attempt.
// The target is trusted local configuration. Bare handshake methods then reject.
func (e *CompletionEndpoint010) BindHTTP(target string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	authority, x := httpEndpoint010(target)
	if x != nil || e.used || e.httpTarget != "" || len(e.signing) != 64 {
		return errCompletion010
	}
	e.httpTarget, e.httpAuthority = target, authority
	return nil
}
func (e *CompletionEndpoint010) httpEnd010(start registry010.Stamp, expires int64, a, b *registry010.Pinned) error {
	end, x := e.sample()
	if x != nil || end.MonoMS-start.MonoMS > 5000 || end.Unix >= expires || !pinnedLive010(end.Unix, a, b) {
		return errCompletion010
	}
	return nil
}

// StartHTTP retains the signed HTTP request in one-shot pending state.
func (e *CompletionEndpoint010) StartHTTP(ctx context.Context, recipient, kid string, ttl int64) (*PendingCompletion010, HTTPMessage010, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || e.httpTarget == "" {
		return nil, HTTPMessage010{}, errCompletion010
	}
	p, body, x := e.start010(ctx, recipient, kid, ttl)
	if x != nil {
		return nil, HTTPMessage010{}, x
	}
	m, x := signHTTP010(e, e.httpTarget, e.httpAuthority, body, 0, nil)
	if x != nil || e.httpEnd010(start, p.expires, p.a, p.b) != nil {
		p.destroy()
		return nil, HTTPMessage010{}, errCompletion010
	}
	h, x := httpHeaders010(m)
	if x != nil {
		p.destroy()
		return nil, HTTPMessage010{}, x
	}
	p.http = contextHTTP010(m, h)
	return p, m, nil
}

// RespondHTTP verifies HTTP and envelope authentication before the single
// handshake replay reservation. Returned session records remain HTTP-bound.
func (e *CompletionEndpoint010) RespondHTTP(ctx context.Context, m HTTPMessage010, ttl int64) (*AuthenticatedCompletion010, HTTPMessage010, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil {
		return nil, HTTPMessage010{}, errCompletion010
	}
	proof, x := prepareHTTP010(e.httpTarget, e.httpAuthority, m, false, start)
	if x != nil {
		return nil, HTTPMessage010{}, x
	}
	s, body, x := e.respond010(ctx, m.Body, ttl, proof)
	if x != nil {
		return nil, HTTPMessage010{}, x
	}
	result, x := signHTTP010(e, e.httpTarget, e.httpAuthority, body, 200, contextHTTP010(m, proof.headers))
	if x != nil || e.httpEnd010(start, s.expires, s.a, s.b) != nil {
		s.destroy()
		return nil, HTTPMessage010{}, errCompletion010
	}
	s.httpTarget, s.httpAuthority = e.httpTarget, e.httpAuthority
	return s, result, nil
}

// CompleteHTTP consumes pending state on any attempted HTTP completion failure,
// as Complete does. It never falls back to bare completion or unsigned errors.
func (p *PendingCompletion010) CompleteHTTP(ctx context.Context, m HTTPMessage010) (*AuthenticatedCompletion010, error) {
	e := p.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || p.http == nil {
		p.destroy()
		return nil, errCompletion010
	}
	proof, x := prepareHTTP010(e.httpTarget, e.httpAuthority, m, true, start)
	if x != nil {
		p.destroy()
		return nil, x
	}
	s, x := p.complete010(ctx, m.Body, proof)
	if x != nil {
		return nil, x
	}
	s.httpTarget, s.httpAuthority = e.httpTarget, e.httpAuthority
	return s, nil
}

// ParseHTTP010 parses exactly one bounded HTTP/1.1 message from authenticated TLS
// content. It does not establish TLS itself. No pipelining, chunking or upgrade.
func ParseHTTP010(raw []byte, target string, response bool) (HTTPMessage010, error) {
	return parseHTTPBytes010(raw, target, response)
}

// EncodeHTTP010 emits one close-delimited connection's Content-Length-framed
// message without changing the content or signed field values.
func EncodeHTTP010(m HTTPMessage010, target string) ([]byte, error) {
	return encodeHTTPBytes010(m, target)
}
