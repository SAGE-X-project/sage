package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"filippo.io/edwards25519"
	"strconv"
	"strings"
)

// Authority is trusted host configuration, never decoded from the wire.
// Now must fail when the clock is untrusted. ActiveKey must freshly resolve the
// exact accepted, unrevoked, unexpired Ed25519 key and fail on stale observations.
// Implementations must honor context cancellation and bound resolution time.
type Authority interface {
	Now(context.Context) (int64, error)
	ActiveKey(context.Context, string, string) (ed25519.PublicKey, error)
}

// IntentPolicy owns protected original/policy/manifest bindings. Authorize must
// evaluate the closed tool schema and the complete final arguments against the
// locally provisioned policy; missing decisions and evaluator failures are errors.
// Arguments are canonical copies. Success is a verification snapshot, not dispatch.
type IntentPolicy interface {
	Bindings(context.Context, string, string) (original string, policy []byte, manifest []byte, err error)
	Authorize(context.Context, string, string, []byte) error
}

// Outstanding supplies exact locally stored, previously authorized intent bytes
// for an outstanding transport invocation. It must fail for unsolicited results.
// It is responsible for durable single terminal consumption outside this API.
type Outstanding interface {
	Intent(context.Context, string, string) ([]byte, error)
}

// VerifiedIntent owns the entire authenticated canonical envelope including proof.
// Its zero value is invalid. Only the private reservation bridge consumes it;
// it is never a dispatch capability.
type VerifiedIntent struct{ canonical []byte }

func (v *VerifiedIntent) Canonical() []byte {
	if v == nil {
		return nil
	}
	return append([]byte(nil), v.canonical...)
}
func (v *VerifiedIntent) Digest() string {
	if v == nil || len(v.canonical) == 0 {
		return ""
	}
	return hash(v.canonical)
}

// VerifiedResult owns authenticated output but does not consume a client call.
type VerifiedResult struct {
	canonical []byte
	status    string
	output    []byte
}

func (v *VerifiedResult) Canonical() []byte {
	if v == nil {
		return nil
	}
	return append([]byte(nil), v.canonical...)
}
func (v *VerifiedResult) Output() []byte {
	if v == nil {
		return nil
	}
	return append([]byte(nil), v.output...)
}
func (v *VerifiedResult) Status() string {
	if v == nil {
		return ""
	}
	return v.status
}

func number(m map[string]any, k string) (int64, bool) {
	n, ok := m[k].(json.Number)
	if !ok {
		return 0, false
	}
	return exactInteger(string(n))
}
func exactInteger(s string) (int64, bool) {
	if strings.HasPrefix(s, "-") {
		return 0, false
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == 'e' || r == 'E' })
	mantissa := parts[0]
	digits := strings.ReplaceAll(mantissa, ".", "")
	frac := 0
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		frac = len(mantissa) - i - 1
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return 0, true
	}
	exponent := int64(0)
	if len(parts) == 2 {
		n, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || n < -MaxBytes || n > MaxBytes {
			return 0, false
		}
		exponent = n
	}
	trimmed := strings.TrimRight(digits, "0")
	scale := exponent - int64(frac) + int64(len(digits)-len(trimmed))
	if scale < 0 || scale > 16 || int64(len(trimmed))+scale > 16 {
		return 0, false
	}
	n, e := strconv.ParseInt(trimmed+strings.Repeat("0", int(scale)), 10, 64)
	return n, e == nil && n <= 9007199254740991
}
func strongPoint(b []byte) bool {
	p, e := new(edwards25519.Point).SetBytes(b)
	return e == nil && bytes.Equal(p.Bytes(), b) && new(edwards25519.Point).MultByCofactor(p).Equal(edwards25519.NewIdentityPoint()) != 1
}

func times(m map[string]any, now int64) bool {
	c, ok := number(m, "created")
	x, ok2 := number(m, "expires")
	return ok && ok2 && now >= 0 && now <= 9007199254740991 && c <= now+30 && now < x && x > c && x-c <= 300
}
func b64(s string, n int) ([]byte, bool) {
	if len(s) > (n*4+2)/3 {
		return nil, false
	}
	b, e := base64.RawURLEncoding.Strict().DecodeString(s)
	return b, e == nil && len(b) == n && base64.RawURLEncoding.EncodeToString(b) == s
}
func common(m map[string]any) bool {
	issuer := str(m, "issuer")
	kid := str(m, "keyid")
	p := strings.Split(kid, "#")
	_, c := number(m, "created")
	_, x := number(m, "expires")
	return str(m, "version") == "0.10.0" && uuid.MatchString(str(m, "request_id")) && uuid.MatchString(str(m, "call_id")) && did(issuer) && did(str(m, "recipient")) && len(kid) <= 289 && len(p) == 2 && p[0] == issuer && keyName.MatchString(p[1]) && str(m, "alg") == "ed25519" && c && x
}
func intentEnvelope(raw []byte) (map[string]any, map[string]any, []byte, error) {
	e, b, err := object(raw)
	if err != nil || !closed(e, "intent proof") {
		return nil, nil, nil, ErrInvalid
	}
	m, ok := e["intent"].(map[string]any)
	if !ok || !closed(m, "version profile request_id call_id parent_call_id original_digest issuer recipient tool arguments policy_digest manifest_digest created expires nonce keyid alg") || !common(m) || str(m, "profile") != "sage-execution-guard" || !toolName.MatchString(str(m, "tool")) || str(m, "tool") == "sage_secure_call" {
		return nil, nil, nil, ErrInvalid
	}
	if m["parent_call_id"] != nil && !uuid.MatchString(str(m, "parent_call_id")) {
		return nil, nil, nil, ErrInvalid
	}
	if _, ok = m["arguments"].(map[string]any); !ok {
		return nil, nil, nil, ErrInvalid
	}
	for _, k := range split("original_digest policy_digest manifest_digest") {
		if !digest.MatchString(str(m, k)) {
			return nil, nil, nil, ErrInvalid
		}
	}
	if _, ok = b64(str(m, "nonce"), 16); !ok {
		return nil, nil, nil, ErrInvalid
	}
	if _, ok = b64(str(e, "proof"), 64); !ok {
		return nil, nil, nil, ErrInvalid
	}
	c, _ := number(m, "created")
	x, _ := number(m, "expires")
	if x <= c || x-c > 300 {
		return nil, nil, nil, ErrInvalid
	}
	return e, m, b, nil
}
func authenticate(ctx context.Context, a Authority, e, m map[string]any, domain string) error {
	if a == nil || ctx.Err() != nil {
		return ErrInvalid
	}
	now, err := a.Now(ctx)
	if err != nil || !times(m, now) {
		return ErrInvalid
	}
	key, err := a.ActiveKey(ctx, str(m, "issuer"), str(m, "keyid"))
	if err != nil || len(key) != 32 || !strongPoint(key) || ctx.Err() != nil {
		return ErrInvalid
	}
	proof, ok := b64(str(e, "proof"), 64)
	b, err := Canonicalize(encode(m))
	if !ok || err != nil || !strongPoint(proof[:32]) || !ed25519.Verify(key, append([]byte(domain+"|0.10.0\x00"), b...), proof) {
		return ErrInvalid
	}
	now, err = a.Now(ctx)
	if err != nil || !times(m, now) || ctx.Err() != nil {
		return ErrInvalid
	}
	return nil
}

// VerifyIntent authenticates Ed25519 and consults locally configured policy.
// Optional algorithms are rejected. Callers must recheck current authority at
// the serialized dispatch boundary; this API cannot certify host isolation.
func VerifyIntent(ctx context.Context, raw []byte, recipient string, a Authority, p IntentPolicy) (*VerifiedIntent, error) {
	e, m, b, err := intentEnvelope(raw)
	if err != nil || p == nil || str(m, "recipient") != recipient {
		return nil, ErrInvalid
	}
	if authenticate(ctx, a, e, m, "sage-execution-intent") != nil {
		return nil, ErrInvalid
	}
	original, policy, manifestRaw, err := p.Bindings(ctx, str(m, "issuer"), str(m, "request_id"))
	if err != nil || original != str(m, "original_digest") {
		return nil, ErrInvalid
	}
	pd, err := PolicyCommitment(policy)
	if err != nil || pd != str(m, "policy_digest") {
		return nil, ErrInvalid
	}
	pm, _, err := object(policy)
	if err != nil || str(pm, "issuer") != str(m, "issuer") {
		return nil, ErrInvalid
	}
	md, err := ManifestCommitment(manifestRaw)
	if err != nil || md != str(m, "manifest_digest") {
		return nil, ErrInvalid
	}
	args, err := Canonicalize(encode(m["arguments"]))
	if err != nil || p.Authorize(ctx, str(m, "issuer"), str(m, "tool"), args) != nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	now, err := a.Now(ctx)
	if err != nil || !times(m, now) {
		return nil, ErrInvalid
	}
	return &VerifiedIntent{canonical: b}, nil
}

// VerifyResult checks a fresh result against an already accepted invocation.
// It deliberately does not require that stored intent to remain unexpired.
func VerifyResult(ctx context.Context, raw []byte, a Authority, o Outstanding) (*VerifiedResult, error) {
	e, b, err := object(raw)
	if err != nil || o == nil || !closed(e, "result proof") {
		return nil, ErrInvalid
	}
	m, ok := e["result"].(map[string]any)
	if !ok || !closed(m, "version request_id call_id issuer recipient created expires keyid alg intent_digest status output") || !common(m) || !digest.MatchString(str(m, "intent_digest")) {
		return nil, ErrInvalid
	}
	status := str(m, "status")
	out, ok := m["output"].(map[string]any)
	if !ok {
		return nil, ErrInvalid
	}
	switch status {
	case "completed":
	case "pending", "rejected", "unknown":
		if len(out) != 0 {
			return nil, ErrInvalid
		}
	default:
		return nil, ErrInvalid
	}
	if authenticate(ctx, a, e, m, "sage-tool-result") != nil {
		return nil, ErrInvalid
	}
	stored, err := o.Intent(ctx, str(m, "request_id"), str(m, "call_id"))
	if err != nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	_, intent, canonical, err := intentEnvelope(stored)
	if err != nil || hash(canonical) != str(m, "intent_digest") || str(intent, "request_id") != str(m, "request_id") || str(intent, "call_id") != str(m, "call_id") || str(intent, "issuer") != str(m, "recipient") || str(intent, "recipient") != str(m, "issuer") {
		return nil, ErrInvalid
	}
	output, err := Canonicalize(encode(out))
	if err != nil {
		return nil, ErrInvalid
	}
	now, err := a.Now(ctx)
	if err != nil || !times(m, now) || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	return &VerifiedResult{canonical: b, status: status, output: output}, nil
}
