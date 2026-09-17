package hpke

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"net/url"
	"strconv"
	"strings"
)

// HTTPMessage010 carries exact content bytes and uncombined field occurrences.
// Method/Target/Authority and Status MUST come from the trusted HTTP transport,
// never Forwarded fields or a peer-supplied JSON representation. TLS authentication,
// HTTP framing and routing remain the transport's responsibility.
type HTTPMessage010 struct {
	Method    string      `json:"method"`
	Target    string      `json:"target"`
	Authority string      `json:"authority"`
	Status    int         `json:"status"`
	Headers   [][2]string `json:"headers"`
	Body      []byte      `json:"body"`
}
type httpContext010 struct{ method, target, authority, digest, signature, version string }
type httpProof010 struct {
	message         HTTPMessage010
	headers, params map[string]string
	input           string
	signature       []byte
	start           registry010.Stamp
}

const httpRequestComponents010 = `"@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "x-sage-version"`
const httpResponseComponents010 = `"@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "x-sage-version";req "content-type" "content-digest" "x-sage-did" "x-sage-version"`

// BindHTTP permanently selects the bounded HTTPS POST session-message profile.
// It must precede all session records; bare record APIs then reject calls. The
// configured canonical endpoint is trusted application configuration, not input.
func (s *AuthenticatedCompletion010) BindHTTP(target string) error {
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	u, x := url.Parse(target)
	if x != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.Path == "" || strings.HasSuffix(u.Host, ":443") || u.Host != strings.ToLower(u.Host) || u.String() != target || !httpASCII010(target) || strings.ContainsAny(target, "\"\\ ") || s.closed || s.httpTarget != "" || len(s.sent)+len(s.received) != 0 {
		return errCompletion010
	}
	s.httpTarget, s.httpAuthority = target, u.Host
	return nil
}
func httpASCII010(v string) bool {
	for _, c := range []byte(v) {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
func httpToken010(v string) bool {
	if v == "" {
		return false
	}
	for _, c := range v {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", c) {
			continue
		}
		return false
	}
	return true
}
func httpDigest010(b []byte) string {
	h := sha256.Sum256(b)
	return "sha-256=:" + base64.StdEncoding.EncodeToString(h[:]) + ":"
}
func httpHeaders010(m HTTPMessage010) (map[string]string, error) {
	if len(m.Body) > 32768 || len(m.Headers) > 256 {
		return nil, errCompletion010
	}
	h := map[string]string{}
	total := 0
	for _, pair := range m.Headers {
		k, v := strings.ToLower(pair[0]), pair[1]
		total += len(pair[0]) + len(v) + 4
		if !httpToken010(k) || !httpASCII010(v) || strings.TrimSpace(v) != v || total > 32768 {
			return nil, errCompletion010
		}
		critical := k == "signature" || k == "signature-input" || k == "content-digest" || k == "content-type" || k == "host" || k == "content-length" || strings.HasPrefix(k, "x-sage-")
		if _, ok := h[k]; ok && critical {
			return nil, errCompletion010
		}
		if k == "content-encoding" || k == "transfer-encoding" || k == "trailer" || strings.HasPrefix(k, "x-sage-meta-") {
			return nil, errCompletion010
		}
		if (k == "signature" || k == "signature-input") && len(v) > 8192 {
			return nil, errCompletion010
		}
		h[k] = v
	}
	if h["content-type"] != "application/json" || h["x-sage-version"] != "0.10.0" || h["x-sage-did"] == "" || h["content-digest"] != httpDigest010(m.Body) {
		return nil, errCompletion010
	}
	if v, ok := h["content-length"]; ok && v != strconv.Itoa(len(m.Body)) {
		return nil, errCompletion010
	}
	if v, ok := h["host"]; ok && v != m.Authority {
		return nil, errCompletion010
	}
	return h, nil
}

// The admitted Structured Fields subset is canonically serialized. Parameter
// order may vary and is retained verbatim; duplicate/unknown parameters fail.
func httpInput010(input string, response bool) (map[string]string, error) {
	components := httpRequestComponents010
	if response {
		components = httpResponseComponents010
	}
	prefix := "sig1=(" + components + ")"
	if !strings.HasPrefix(input, prefix) {
		return nil, errCompletion010
	}
	rest := input[len(prefix):]
	out := map[string]string{}
	for rest != "" {
		if rest[0] != ';' {
			return nil, errCompletion010
		}
		rest = rest[1:]
		n := strings.IndexByte(rest, '=')
		if n < 1 {
			return nil, errCompletion010
		}
		key := rest[:n]
		rest = rest[n+1:]
		if _, ok := out[key]; ok {
			return nil, errCompletion010
		}
		var value string
		switch key {
		case "keyid", "alg", "nonce", "tag":
			if len(rest) < 2 || rest[0] != '"' {
				return nil, errCompletion010
			}
			n = strings.IndexByte(rest[1:], '"')
			if n < 0 {
				return nil, errCompletion010
			}
			n++
			value = rest[1:n]
			if !httpASCII010(value) || strings.ContainsRune(value, '\\') {
				return nil, errCompletion010
			}
			rest = rest[n+1:]
		case "created", "expires":
			n = strings.IndexByte(rest, ';')
			if n < 0 {
				n = len(rest)
			}
			value = rest[:n]
			rest = rest[n:]
			i, x := strconv.ParseInt(value, 10, 64)
			if x != nil || i < 0 || len(value) > 15 || strconv.FormatInt(i, 10) != value {
				return nil, errCompletion010
			}
		default:
			return nil, errCompletion010
		}
		out[key] = value
	}
	if len(out) != 6 || out["alg"] != "ed25519" || out["tag"] != "sage-0.10.0" {
		return nil, errCompletion010
	}
	return out, nil
}
func (s *AuthenticatedCompletion010) prepareHTTP010(m HTTPMessage010, response bool, start registry010.Stamp) (*httpProof010, error) {
	if s.httpTarget == "" || (!response && (m.Method != "POST" || m.Target != s.httpTarget || m.Authority != s.httpAuthority || m.Status != 0)) || (response && (m.Method != "" || m.Target != "" || m.Authority != "" || m.Status < 200 || m.Status > 599 || m.Status == 204)) {
		return nil, errCompletion010
	}
	h, x := httpHeaders010(m)
	if x != nil {
		return nil, x
	}
	p, x := httpInput010(h["signature-input"], response)
	if x != nil {
		return nil, x
	}
	v := h["signature"]
	if !strings.HasPrefix(v, "sig1=:") || !strings.HasSuffix(v, ":") {
		return nil, errCompletion010
	}
	sig, x := base64.StdEncoding.Strict().DecodeString(v[6 : len(v)-1])
	if x != nil || len(sig) != 64 || "sig1=:"+base64.StdEncoding.EncodeToString(sig)+":" != v {
		return nil, errCompletion010
	}
	return &httpProof010{message: m, headers: h, params: p, input: h["signature-input"][5:], signature: sig, start: start}, nil
}
func httpBase010(m HTTPMessage010, h map[string]string, input string, q *httpContext010) []byte {
	lines := []string{}
	add := func(k, v string) { lines = append(lines, k+": "+v) }
	if q == nil {
		add(`"@method"`, m.Method)
		add(`"@target-uri"`, m.Target)
		add(`"@authority"`, m.Authority)
	} else {
		add(`"@status"`, strconv.Itoa(m.Status))
		add(`"@method";req`, q.method)
		add(`"@target-uri";req`, q.target)
		add(`"@authority";req`, q.authority)
		add(`"content-digest";req`, q.digest)
		add(`"signature";req`, q.signature)
		add(`"x-sage-version";req`, q.version)
	}
	for _, k := range []string{"content-type", "content-digest", "x-sage-did", "x-sage-version"} {
		add(`"`+k+`"`, h[k])
	}
	add(`"@signature-params"`, input)
	return []byte(strings.Join(lines, "\n"))
}
func contextHTTP010(m HTTPMessage010, h map[string]string) *httpContext010 {
	return &httpContext010{m.Method, m.Target, m.Authority, h["content-digest"], h["signature"], h["x-sage-version"]}
}
func (s *AuthenticatedCompletion010) verifyHTTP010(p *httpProof010, w map[string]json.RawMessage) error {
	for param, field := range map[string]string{"keyid": "kid", "created": "created", "expires": "expires", "nonce": "nonce"} {
		value := str010(w, field)
		if field == "created" || field == "expires" {
			value = string(w[field])
		}
		if p.params[param] != value {
			return errCompletion010
		}
	}
	if p.headers["x-sage-did"] != str010(w, "did") {
		return errCompletion010
	}
	for k, f := range map[string]string{"x-sage-message-id": "id", "x-sage-context-id": "context_id", "x-sage-task-id": "task_id"} {
		if v, ok := p.headers[k]; ok && (str010(w, f) == "" || v != str010(w, f)) {
			return errCompletion010
		}
	}
	var q *httpContext010
	if p.message.Status != 0 {
		r := s.sent[str010(w, "message_id")]
		if r == nil || r.http == nil {
			return errCompletion010
		}
		q = r.http
	}
	key := s.a.Signing()
	if s.initiator {
		key = s.b.Signing()
	}
	pub, x := hex.DecodeString(key.Material)
	if x != nil || len(pub) != 32 || key.Alg != "ed25519" {
		return errCompletion010
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), httpBase010(p.message, p.headers, p.input, q), p.signature) {
		return errCompletion010
	}
	return nil
}
func (s *AuthenticatedCompletion010) signHTTP010(body []byte, status int, q *httpContext010) (HTTPMessage010, error) {
	var w map[string]json.RawMessage
	if json.Unmarshal(body, &w) != nil {
		return HTTPMessage010{}, errCompletion010
	}
	m := HTTPMessage010{Body: body, Status: status}
	components := httpResponseComponents010
	if q == nil {
		m.Method = "POST"
		m.Target = s.httpTarget
		m.Authority = s.httpAuthority
		components = httpRequestComponents010
	}
	input := "(" + components + ");keyid=" + strconv.Quote(str010(w, "kid")) + `;alg="ed25519";created=` + string(w["created"]) + ";expires=" + string(w["expires"]) + ";nonce=" + strconv.Quote(str010(w, "nonce")) + `;tag="sage-0.10.0"`
	h := map[string]string{"content-type": "application/json", "content-digest": httpDigest010(body), "x-sage-did": str010(w, "did"), "x-sage-version": "0.10.0", "signature-input": "sig1=" + input}
	h["signature"] = "sig1=:" + base64.StdEncoding.EncodeToString(ed25519.Sign(s.endpoint.signing, httpBase010(m, h, input, q))) + ":"
	for _, k := range []string{"content-type", "content-digest", "x-sage-did", "x-sage-version", "signature-input", "signature"} {
		m.Headers = append(m.Headers, [2]string{k, h[k]})
	}
	return m, nil
}
func (s *AuthenticatedCompletion010) SealHTTPRequest(ctx context.Context, plaintext []byte, ttl int64) (HTTPMessage010, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return HTTPMessage010{}, errCompletion010
	}
	if s.httpTarget == "" {
		return HTTPMessage010{}, errCompletion010
	}
	b, x := s.sealRequest010(ctx, plaintext, ttl)
	if x != nil {
		return HTTPMessage010{}, x
	}
	m, x := s.signHTTP010(b, 0, nil)
	if x != nil {
		s.destroy()
		return HTTPMessage010{}, x
	}
	if _, x = s.recordGate(start, start.Unix+ttl); x != nil {
		return HTTPMessage010{}, x
	}
	var w map[string]json.RawMessage
	_ = json.Unmarshal(b, &w)
	h, _ := httpHeaders010(m)
	s.sent[str010(w, "id")].http = contextHTTP010(m, h)
	return m, nil
}
func (s *AuthenticatedCompletion010) OpenHTTPRequest(ctx context.Context, m HTTPMessage010) ([]byte, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil {
		s.destroy()
		return nil, errCompletion010
	}
	p, x := s.prepareHTTP010(m, false, start)
	if x != nil {
		return nil, x
	}
	b, x := s.openRequest010(ctx, m.Body, p)
	if x != nil {
		return nil, x
	}
	var w map[string]json.RawMessage
	_ = json.Unmarshal(m.Body, &w)
	s.received[str010(w, "id")].http = contextHTTP010(m, p.headers)
	return b, nil
}
func (s *AuthenticatedCompletion010) SealHTTPResponse(ctx context.Context, messageID string, data []byte, success bool, code string, ttl int64, status int) (HTTPMessage010, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	r := s.received[messageID]
	if x != nil || s.recordLive(start) != nil {
		s.destroy()
		return HTTPMessage010{}, errCompletion010
	}
	if s.httpTarget == "" || r == nil || r.http == nil || status < 200 || status > 599 || status == 204 {
		return HTTPMessage010{}, errCompletion010
	}
	b, x := s.sealResponse010(ctx, messageID, data, success, code, ttl)
	if x != nil {
		return HTTPMessage010{}, x
	}
	m, x := s.signHTTP010(b, status, r.http)
	if x != nil {
		s.destroy()
		return HTTPMessage010{}, x
	}
	if _, x = s.recordGate(start, start.Unix+ttl); x != nil {
		return HTTPMessage010{}, x
	}
	return m, nil
}
func (s *AuthenticatedCompletion010) OpenHTTPResponse(ctx context.Context, m HTTPMessage010) (*SessionResponse010, error) {
	e := s.endpoint
	e.mu.Lock()
	defer e.mu.Unlock()
	start, x := e.sample()
	if x != nil {
		s.destroy()
		return nil, errCompletion010
	}
	p, x := s.prepareHTTP010(m, true, start)
	if x != nil {
		return nil, x
	}
	return s.openResponse010(ctx, m.Body, p)
}
