package guard010

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"

	"github.com/sage-x-project/sage/pkg/agent/execution010"
)

// ResultSigner is a trusted executor signer and fresh result-key authority.
// KeyID must identify this executor. Sign signs only the supplied domain-separated
// bytes for that exact key ID. Callbacks must honor bounded deadlines. The core
// verifies the returned proof against the current active key before storing it.
type ResultSigner interface {
	Authority
	KeyID(context.Context) (string, error)
	Sign(context.Context, string, []byte) ([]byte, error)
}

// Completion is an opaque, gate-bound accepted execution token. Only a committed
// Invocation supplies one. It cannot complete a different call or recovered UNKNOWN.
// The trusted worker must keep it outside plugin/model write capabilities.
type Completion struct {
	owner     *DispatchGate
	canonical []byte
}

// Completion returns the token to a trusted worker. It grants result recording,
// not another execution. Finish persists an outcome but never pushes a response.
func (i *Invocation) Completion() *Completion {
	if i == nil {
		return nil
	}
	return i.completion
}

type replyPermit struct {
	owner     *DispatchGate
	canonical []byte
	used      bool
}
type acceptedInvocation []byte

func (o acceptedInvocation) Intent(context.Context, string, string) ([]byte, error) {
	return append([]byte(nil), o...), nil
}
func matchesIntent(e execution010.Entry, raw []byte) bool {
	expected, err := reservationEntry(&VerifiedIntent{canonical: raw})
	if err != nil {
		return false
	}
	e.State = ""
	e.ResultHex = ""
	expected.State = ""
	return e == expected
}
func (g *DispatchGate) resultEntry(raw []byte) (execution010.Entry, error) {
	_, i, _, err := intentEnvelope(raw)
	if err != nil {
		return execution010.Entry{}, ErrInvalid
	}
	e, ok, err := g.store.Lookup(str(i, "issuer"), str(i, "call_id"))
	if err != nil || !ok || !matchesIntent(e, raw) {
		return execution010.Entry{}, ErrInvalid
	}
	return e, nil
}
func checkResult(ctx context.Context, raw, intent []byte, state string, s ResultSigner) (*VerifiedResult, error) {
	if s == nil {
		return nil, ErrInvalid
	}
	v, e := VerifyResult(ctx, raw, s, acceptedInvocation(intent))
	if e != nil || !bytes.Equal(v.Canonical(), raw) {
		return nil, ErrInvalid
	}
	want := map[string]string{"RESERVED": "pending", "EXECUTING": "pending", "COMPLETED": "completed", "REJECTED": "rejected", "UNKNOWN": "unknown"}[state]
	if v.Status() != want {
		return nil, ErrInvalid
	}
	return v, nil
}
func signResult(ctx context.Context, intent []byte, status string, output []byte, s ResultSigner) ([]byte, error) {
	if s == nil {
		return nil, ErrInvalid
	}
	_, i, _, err := intentEnvelope(intent)
	if err != nil {
		return nil, ErrInvalid
	}
	out, _, err := object(output)
	if err != nil {
		return nil, ErrInvalid
	}
	now, err := s.Now(ctx)
	if err != nil || now < 0 || now > 9007199254740691 {
		return nil, ErrInvalid
	}
	kid, err := s.KeyID(ctx)
	if err != nil {
		return nil, ErrInvalid
	}
	r := map[string]any{"version": "0.10.0", "request_id": str(i, "request_id"), "call_id": str(i, "call_id"), "issuer": str(i, "recipient"), "recipient": str(i, "issuer"), "created": now, "expires": now + 300, "keyid": kid, "alg": "ed25519", "intent_digest": hash(intent), "status": status, "output": out}
	body, err := Canonicalize(encode(r))
	if err != nil {
		return nil, ErrInvalid
	}
	// Validate closed fields and key ownership before calling the signer.
	probe := encode(map[string]any{"result": r, "proof": base64.RawURLEncoding.EncodeToString(make([]byte, 64))})
	env, _, err := object(probe)
	m, ok := env["result"].(map[string]any)
	if err != nil || !ok || !common(m) || str(m, "issuer") != str(i, "recipient") || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	proof, err := s.Sign(ctx, kid, append([]byte("sage-tool-result|0.10.0\x00"), body...))
	if err != nil || len(proof) != 64 {
		return nil, ErrInvalid
	}
	raw, err := Canonicalize(encode(map[string]any{"result": r, "proof": base64.RawURLEncoding.EncodeToString(proof)}))
	if err != nil {
		return nil, ErrInvalid
	}
	state := map[string]string{"pending": "EXECUTING", "completed": "COMPLETED", "rejected": "REJECTED", "unknown": "UNKNOWN"}[status]
	if _, err = checkResult(ctx, raw, intent, state, s); err != nil {
		return nil, ErrInvalid
	}
	return raw, nil
}

// Finish records the first completed result atomically with its exact signed bytes.
// It never sends a response. Accepted work may finish after intent expiry or local
// retirement; only current result-key/time validity is required at this boundary.
// Identical output retries reuse existing bytes; conflicting outcomes are denied.
func (g *DispatchGate) Finish(ctx context.Context, c *Completion, output []byte, s ResultSigner) (err error) {
	if g == nil || ctx == nil || c == nil || c.owner != g {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	defer func() {
		if recover() != nil {
			g.retired = true
			err = ErrInvalid
		}
	}()
	if g.store == nil || ctx.Err() != nil {
		return ErrInvalid
	}
	_, out, err := object(output)
	if err != nil {
		return ErrInvalid
	}
	e, err := g.resultEntry(c.canonical)
	if err != nil {
		return ErrInvalid
	}
	if e.State == "COMPLETED" {
		raw, er := hex.DecodeString(e.ResultHex)
		if er != nil {
			return ErrInvalid
		}
		v, er := checkResult(ctx, raw, c.canonical, e.State, s)
		if er != nil || !bytes.Equal(v.Output(), out) {
			return ErrInvalid
		}
		return nil
	}
	if e.State != "EXECUTING" || e.ResultHex != "" {
		return ErrInvalid
	}
	raw, err := signResult(ctx, c.canonical, "completed", out, s)
	if err != nil {
		return ErrInvalid
	}
	e.State = "COMPLETED"
	e.ResultHex = hex.EncodeToString(raw)
	if changed, er := g.store.Commit(e); er != nil || !changed {
		g.retired = true
		return ErrInvalid
	}
	_, err = checkResult(ctx, raw, c.canonical, e.State, s)
	return err
}

// Reply consumes one local response permit from an accepted Dispatch or Reject.
// Copying the receipt does not duplicate this permit. Failure consumes it as an
// unverified local transport failure; retry requires a newly authenticated Dispatch.
// The host must bind each receipt to one fresh outer transport invocation and must
// not call Dispatch again for the same invocation. Client polling/consumption is
// separate. A late accepted response may outlive intent expiry, never result expiry.
func (g *DispatchGate) Reply(ctx context.Context, r *DispatchReceipt, s ResultSigner) (raw []byte, err error) {
	if g == nil || ctx == nil || r == nil || r.reply == nil || r.reply.owner != g {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	p := r.reply
	if p.used {
		return nil, ErrInvalid
	}
	p.used = true
	defer func() {
		if recover() != nil {
			g.retired = true
			raw = nil
			err = ErrInvalid
		}
	}()
	if g.store == nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	e, err := g.resultEntry(p.canonical)
	if err != nil {
		return nil, ErrInvalid
	}
	if e.ResultHex != "" {
		raw, err = hex.DecodeString(e.ResultHex)
		if err != nil {
			return nil, ErrInvalid
		}
	} else {
		status := "pending"
		if e.State == "UNKNOWN" {
			status = "unknown"
		} else if e.State != "RESERVED" && e.State != "EXECUTING" {
			return nil, ErrInvalid
		}
		raw, err = signResult(ctx, p.canonical, status, []byte("{}"), s)
		if err != nil {
			return nil, ErrInvalid
		}
		if e.State == "UNKNOWN" {
			e.ResultHex = hex.EncodeToString(raw)
			if changed, er := g.store.Commit(e); er != nil || !changed {
				g.retired = true
				return nil, ErrInvalid
			}
		}
	}
	if _, err = checkResult(ctx, raw, p.canonical, e.State, s); err != nil {
		return nil, ErrInvalid
	}
	return append([]byte(nil), raw...), nil
}

// Reject is an explicit trusted decision not to execute an otherwise authenticated
// and currently policy-authorized intent. Verification failures are not converted
// into signed verdicts. It atomically reserves a previously absent call and nonce
// with its first signed rejection; an existing execution cannot be overwritten.
func (g *DispatchGate) Reject(ctx context.Context, raw []byte, s ResultSigner) (r *DispatchReceipt, err error) {
	if g == nil || ctx == nil {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	defer func() {
		if recover() != nil {
			g.retired = true
			r = nil
			err = ErrInvalid
		}
	}()
	if g.store == nil || g.retired || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	v, err := VerifyIntent(ctx, raw, g.recipient, g.authority, g.policy)
	if err != nil {
		return nil, ErrInvalid
	}
	e, err := reservationEntry(v)
	if err != nil {
		return nil, ErrInvalid
	}
	old, exists, err := g.store.Lookup(e.Issuer, e.CallID)
	if err != nil {
		return nil, ErrInvalid
	}
	if exists {
		if old.State != "REJECTED" || !matchesIntent(old, v.canonical) {
			return nil, ErrInvalid
		}
	} else {
		signed, er := signResult(ctx, v.canonical, "rejected", []byte("{}"), s)
		if er != nil {
			return nil, ErrInvalid
		}
		e.State = "REJECTED"
		e.ResultHex = hex.EncodeToString(signed)
		if changed, er := g.store.Commit(e); er != nil || !changed {
			return nil, ErrInvalid
		}
	}
	return &DispatchReceipt{reservation: Reservation{created: !exists, state: "REJECTED", digest: v.Digest()}, reply: &replyPermit{owner: g, canonical: v.Canonical()}}, nil
}
