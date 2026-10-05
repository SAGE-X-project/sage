package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// IntentSigner is protected host key custody, never a model-facing signing tool.
// Sign must use only the exact locally selected key ID and honor cancellation.
type IntentSigner interface {
	Sign(context.Context, string, []byte) ([]byte, error)
}

// IssuancePolicy evaluates the entire canonical unsigned intent, including its
// locally selected recipient, epoch, capture and exact arguments. Success is
// consumed through an opaque one-use token; it is not a reusable wire permission.
type IssuancePolicy interface {
	IntentPolicy
	ApproveIntent(context.Context, []byte) error
}

// IntentMeasurement checks the same pinned loaded instance and its dependencies.
// Hashes alone do not establish loading or isolation. Callbacks are bounded.
type IntentMeasurement interface {
	Check(context.Context, string, string) error
}

// IssuerServices is trusted immutable host configuration. All callbacks must be
// bounded and non-reentrant. Administration must serialize updates with Retire;
// model/plugin code must not possess these services or their backing capabilities.
type IssuerServices struct {
	Client      ClientServices
	Policy      IssuancePolicy
	Signer      IntentSigner
	Measurement IntentMeasurement
	KeyID       string
}

// IntentProposal contains only a proposed tool and fully resolved JSON arguments.
// Identity, key, capture, commitments, time, call ID and nonce are core/host owned.
type IntentProposal struct {
	Tool            string
	Arguments       []byte
	LifetimeSeconds int64
}

// AuthorizedIntent is an opaque issuer-bound, one-use decision. Its zero value
// is invalid. Keep it in the trusted host; it cannot be serialized as authority.
type AuthorizedIntent struct {
	owner *IntentIssuer
	body  []byte
	used  bool
}

// IntentIssuer serializes approval, retirement and protected issuance. Each stable
// operation path gets a permanent .issuance fence before key use. A failed or
// interrupted attempt must be reconciled, never automatically re-signed at a new
// path. The host protects both that fence and the Client journal against rollback.
type IntentIssuer struct {
	mu          sync.Mutex
	capture     RootCapture
	services    IssuerServices
	incoming    []byte
	hop         *HopServices
	retired     bool
	clientTaken bool
}

const issuanceHeader = "sage-intent-issuance|0.10.0\n"

// NewIntentIssuer constructs one root operation issuer without key use.
// Issue/Reopen transfers its Client services once; resuming uses a fresh issuer
// with the same protected configuration and stable operation path.
func NewIntentIssuer(capture *RootCapture, s IssuerServices) (*IntentIssuer, error) {
	if capture == nil || !uuid.MatchString(capture.requestID) || !digest.MatchString(capture.digest) ||
		s.Policy == nil || s.Signer == nil || s.Measurement == nil || s.Client.IntentAuthority == nil || s.Client.Policy == nil ||
		s.Client.ResultAuthority == nil || s.Client.Clock == nil || s.Client.Sender == nil ||
		!did(s.Client.ExpectedIssuer) || !did(s.Client.ExpectedRecipient) ||
		(runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		return nil, ErrInvalid
	}
	probe := map[string]any{"version": "0.10.0", "request_id": capture.requestID, "call_id": capture.requestID,
		"issuer": s.Client.ExpectedIssuer, "recipient": s.Client.ExpectedRecipient, "keyid": s.KeyID, "alg": "ed25519", "created": int64(0), "expires": int64(1)}
	parsed, _, parseErr := object(encode(probe))
	if parseErr != nil || !common(parsed) {
		return nil, ErrInvalid
	}
	return &IntentIssuer{capture: *capture, services: s}, nil
}

// NewHopIntentIssuer requires a fresh local capture of the exact authenticated
// inbound envelope. Upstream verification and admitted parent authority are
// rechecked before approval, key use and each downstream Client handoff.
func NewHopIntentIssuer(capture *RootCapture, s IssuerServices, incoming []byte, h HopServices) (*IntentIssuer, error) {
	g, err := NewIntentIssuer(capture, s)
	if err != nil || h.Authority == nil || h.Policy == nil || h.Parent == nil {
		return nil, ErrInvalid
	}
	_, p, canonical, err := intentEnvelope(incoming)
	commitment, e := OriginalCommitment([][]byte{incoming})
	if err != nil || e != nil || !bytes.Equal(canonical, incoming) || str(p, "recipient") != s.Client.ExpectedIssuer ||
		capture.requestID == str(p, "request_id") || capture.digest != commitment {
		return nil, ErrInvalid
	}
	g.incoming = append([]byte(nil), incoming...)
	g.hop = &h
	return g, nil
}

func issuanceUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", ErrInvalid
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func unsignedEnvelope(body []byte) []byte {
	return []byte(`{"intent":` + string(body) + `,"proof":"` + base64.RawURLEncoding.EncodeToString(make([]byte, 64)) + `"}`)
}
func unsignedPolicy(ctx context.Context, p IntentPolicy, i map[string]any) error {
	original, policy, manifestRaw, err := p.Bindings(ctx, str(i, "issuer"), str(i, "request_id"))
	pd, e := PolicyCommitment(policy)
	md, m := ManifestCommitment(manifestRaw)
	pm, _, pe := object(policy)
	if err != nil || e != nil || m != nil || pe != nil || original != str(i, "original_digest") ||
		pd != str(i, "policy_digest") || md != str(i, "manifest_digest") || str(pm, "issuer") != str(i, "issuer") {
		return ErrInvalid
	}
	args, e := Canonicalize(encode(i["arguments"]))
	if e != nil || p.Authorize(ctx, str(i, "issuer"), str(i, "tool"), args) != nil || ctx.Err() != nil {
		return ErrInvalid
	}
	return nil
}
func (g *IntentIssuer) check(ctx context.Context, body []byte) (ed25519.PublicKey, error) {
	if g.retired || ctx == nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	_, i, _, err := intentEnvelope(unsignedEnvelope(body))
	s := g.services
	if err != nil || str(i, "issuer") != s.Client.ExpectedIssuer || str(i, "recipient") != s.Client.ExpectedRecipient ||
		str(i, "request_id") != g.capture.requestID || str(i, "original_digest") != g.capture.digest || str(i, "keyid") != s.KeyID {
		return nil, ErrInvalid
	}
	if g.hop != nil {
		if checkHop(ctx, g.incoming, unsignedEnvelope(body), s.Client, *g.hop) != nil {
			return nil, ErrInvalid
		}
	} else if i["parent_call_id"] != nil {
		return nil, ErrInvalid
	}
	if unsignedPolicy(ctx, s.Policy, i) != nil || unsignedPolicy(ctx, s.Client.Policy, i) != nil ||
		s.Policy.ApproveIntent(ctx, append([]byte(nil), body...)) != nil ||
		s.Measurement.Check(ctx, str(i, "manifest_digest"), str(i, "tool")) != nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	key, err := s.Client.IntentAuthority.ActiveKey(ctx, s.Client.ExpectedIssuer, s.KeyID)
	if err != nil || len(key) != 32 || !strongPoint(key) || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	now, err := s.Client.IntentAuthority.Now(ctx)
	if err != nil || !times(i, now) || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	return append(ed25519.PublicKey(nil), key...), nil
}

// Authorize resolves a closed proposal to immutable canonical intent bytes. It
// does not sign, create storage, send traffic or dispatch an effect.
func (g *IntentIssuer) Authorize(ctx context.Context, p IntentProposal) (token *AuthorizedIntent, err error) {
	if g == nil || ctx == nil {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	defer func() {
		if recover() != nil {
			g.retired = true
			token = nil
			err = ErrInvalid
		}
	}()
	if g.retired || g.clientTaken || ctx.Err() != nil || p.LifetimeSeconds < 1 || p.LifetimeSeconds > 300 {
		return nil, ErrInvalid
	}
	args, _, err := object(p.Arguments)
	if err != nil {
		return nil, ErrInvalid
	}
	now, err := g.services.Client.IntentAuthority.Now(ctx)
	if err != nil || now < 0 || now > 9007199254740691 {
		return nil, ErrInvalid
	}
	original, policy, manifestRaw, err := g.services.Policy.Bindings(ctx, g.services.Client.ExpectedIssuer, g.capture.requestID)
	pd, e := PolicyCommitment(policy)
	md, m := ManifestCommitment(manifestRaw)
	if err != nil || e != nil || m != nil || original != g.capture.digest {
		return nil, ErrInvalid
	}
	call, err := issuanceUUID()
	if err != nil {
		return nil, ErrInvalid
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, ErrInvalid
	}
	var parent any
	if g.hop != nil {
		_, i, _, e := intentEnvelope(g.incoming)
		if e != nil {
			return nil, ErrInvalid
		}
		parent = str(i, "call_id")
	}
	i := map[string]any{"version": "0.10.0", "profile": "sage-execution-guard", "request_id": g.capture.requestID, "call_id": call,
		"parent_call_id": parent, "original_digest": original, "issuer": g.services.Client.ExpectedIssuer, "recipient": g.services.Client.ExpectedRecipient,
		"tool": p.Tool, "arguments": args, "policy_digest": pd, "manifest_digest": md, "created": now, "expires": now + p.LifetimeSeconds,
		"nonce": base64.RawURLEncoding.EncodeToString(nonce[:]), "keyid": g.services.KeyID, "alg": "ed25519"}
	body, err := Canonicalize(encode(i))
	if err != nil {
		return nil, ErrInvalid
	}
	if _, err = g.check(ctx, body); err != nil {
		return nil, ErrInvalid
	}
	return &AuthorizedIntent{owner: g, body: body}, nil
}

func issuanceFence(path string, body []byte) error {
	// #nosec G304 -- stable protected host operation path, never a peer argument.
	file, err := os.OpenFile(path+".issuance", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return ErrInvalid
	}
	_, we := file.WriteString(issuanceHeader + hash(body) + "\n")
	se := file.Sync()
	ce := file.Close()
	// #nosec G304 -- sync trusted host storage directory before private-key use.
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		return ErrInvalid
	}
	de := dir.Sync()
	dc := dir.Close()
	if we != nil || se != nil || ce != nil || de != nil || dc != nil {
		return ErrInvalid
	}
	return nil
}

// Issue consumes the decision before any callback, reserves a durable issuance
// fence, rechecks authority and signs once. Only a journaled Client is returned;
// no arbitrary message/signature route or transport send is exposed. Failure
// leaves the token consumed and any durable fence intact for reconciliation.
func (g *IntentIssuer) Issue(ctx context.Context, path string, t *AuthorizedIntent) (client *Client, err error) {
	if g == nil || ctx == nil {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	defer func() {
		if recover() != nil {
			g.retired = true
			client = nil
			err = ErrInvalid
		}
	}()
	if t == nil || t.owner != g || t.used || g.retired || g.clientTaken {
		return nil, ErrInvalid
	}
	t.used = true
	if _, err = g.check(ctx, t.body); err != nil {
		return nil, ErrInvalid
	}
	// Existing Client state cannot be replaced by a new signature.
	if _, e := os.Lstat(path); !os.IsNotExist(e) {
		return nil, ErrInvalid
	}
	if issuanceFence(path, t.body) != nil {
		return nil, ErrInvalid
	}
	key, err := g.check(ctx, t.body)
	if err != nil {
		return nil, ErrInvalid
	}
	proof, err := g.services.Signer.Sign(ctx, g.services.KeyID, append([]byte("sage-execution-intent|0.10.0\x00"), t.body...))
	if err != nil || ctx.Err() != nil || len(proof) != 64 || !strongPoint(proof[:32]) ||
		!ed25519.Verify(key, append([]byte("sage-execution-intent|0.10.0\x00"), t.body...), proof) {
		return nil, ErrInvalid
	}
	raw, err := Canonicalize([]byte(`{"intent":` + string(t.body) + `,"proof":"` + base64.RawURLEncoding.EncodeToString(proof) + `"}`))
	if err != nil {
		return nil, ErrInvalid
	}
	if _, err = g.check(ctx, t.body); err != nil {
		return nil, ErrInvalid
	}
	g.clientTaken = true
	if g.hop != nil {
		return OpenHopClient(ctx, path, true, g.incoming, raw, g.services.Client, *g.hop)
	}
	return OpenCapturedClient(ctx, path, true, raw, g.services.Client, &g.capture)
}

// Retire invalidates all outstanding decisions. Construct a new issuer only
// after trusted configuration updates; a retired issuer cannot be reset.
func (g *IntentIssuer) Retire() error {
	if g == nil {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.retired = true
	return nil
}

// Reopen resumes exact journaled bytes without signing. A missing/partial fence
// or journal fails closed. Host-owned reconciliation is required after a crash
// before journal creation; never silently remove the issuance fence.
func (g *IntentIssuer) Reopen(ctx context.Context, path string) (client *Client, err error) {
	if g == nil || ctx == nil {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	defer func() {
		if recover() != nil {
			g.retired = true
			client = nil
			err = ErrInvalid
		}
	}()
	if g.retired || g.clientTaken || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	// #nosec G304 -- protected operation journal, bounded before decoding.
	file, e := os.Open(path)
	if e != nil {
		return nil, ErrInvalid
	}
	raw, e := io.ReadAll(io.LimitReader(file, clientMaxSize+1))
	ce := file.Close()
	if e != nil || ce != nil || len(raw) > clientMaxSize || !bytes.HasPrefix(raw, []byte(clientHeader)) {
		return nil, ErrInvalid
	}
	lines := bytes.SplitN(raw[len(clientHeader):], []byte("\n"), 2)
	if len(lines) != 2 {
		return nil, ErrInvalid
	}
	var row clientEvent
	if json.Unmarshal(lines[0], &row) != nil || row.Kind != "open" {
		return nil, ErrInvalid
	}
	envelope, e := hex.DecodeString(row.IntentHex)
	if e != nil {
		return nil, ErrInvalid
	}
	_, i, _, e := intentEnvelope(envelope)
	if e != nil {
		return nil, ErrInvalid
	}
	body, e := Canonicalize(encode(i))
	if e != nil {
		return nil, ErrInvalid
	}
	// #nosec G304 -- protected permanent issuance fence, bounded before comparison.
	fence, e := os.Open(path + ".issuance")
	if e != nil {
		return nil, ErrInvalid
	}
	marker, e := io.ReadAll(io.LimitReader(fence, 128))
	ce = fence.Close()
	if e != nil || ce != nil || string(marker) != issuanceHeader+hash(body)+"\n" {
		return nil, ErrInvalid
	}
	if str(i, "keyid") != g.services.KeyID || str(i, "request_id") != g.capture.requestID || str(i, "original_digest") != g.capture.digest {
		return nil, ErrInvalid
	}
	g.clientTaken = true
	if g.hop != nil {
		return OpenHopClient(ctx, path, false, g.incoming, envelope, g.services.Client, *g.hop)
	}
	return OpenCapturedClient(ctx, path, false, envelope, g.services.Client, &g.capture)
}
