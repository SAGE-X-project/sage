package guard010

import (
	"context"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/execution010"
)

// Component is a trusted, pinned immutable loaded instance, not a peer-supplied
// identifier. Check must validate its protected baseline, measured instance and
// tool binding. Commit must atomically accept the exact invocation into that same
// instance's protected execution boundary before returning success. It must not
// resolve a path/name again, add defaults, run an unbounded tool, or reenter this
// gate. Callbacks must honor bounded deadlines and cancellation. Long-running
// work runs outside the gate after this bounded commitment. Host isolation and
// durable administrative baseline/policy updates remain integration duties.
type Component interface {
	Check(context.Context, string, string) error
	Commit(context.Context, *Invocation) error
}

// Invocation is privately created from an authenticated envelope. Accessors return
// copies; it contains no unsigned defaults. It is only delivered to the trusted
// component under the dispatch gate, never returned as a reusable capability.
type Invocation struct {
	canonical, arguments   []byte
	tool, manifest, digest string
	completion             *Completion
	parent                 HopParent
}

func (i *Invocation) CanonicalIntent() []byte { return append([]byte(nil), i.canonical...) }
func (i *Invocation) Arguments() []byte       { return append([]byte(nil), i.arguments...) }
func (i *Invocation) Tool() string            { return i.tool }
func (i *Invocation) ManifestDigest() string  { return i.manifest }
func (i *Invocation) IntentDigest() string    { return i.digest }

// ParentAdmission is available only while this MCP invocation is running in
// its admitted worker. It can be passed to B's trusted downstream HopServices.
// Ordinary dispatch invocations have no parent admission capability.
func (i *Invocation) ParentAdmission() HopParent {
	if i == nil {
		return nil
	}
	return i.parent
}

// DispatchReceipt is local storage/commit metadata, not a signed tool result.
// Committed reports only this invocation's bounded handoff, never tool completion.
type DispatchReceipt struct {
	reservation Reservation
	committed   bool
	reply       *replyPermit
}

func (r *DispatchReceipt) Created() bool { return r != nil && r.reservation.Created() }
func (r *DispatchReceipt) State() string {
	if r == nil {
		return ""
	}
	return r.reservation.State()
}
func (r *DispatchReceipt) IntentDigest() string {
	if r == nil {
		return ""
	}
	return r.reservation.IntentDigest()
}
func (r *DispatchReceipt) Committed() bool { return r != nil && r.committed }

// DispatchGate exclusively owns its store and serializes verification, current
// configuration, retirement and bounded handoff. Callbacks must not mutate policy
// or replace the instance outside Replace/Retire. Fresh registry resolution is
// repeated after durable EXECUTING storage; trusted provider freshness is required.
// A retired gate cannot be reactivated. Reopening requires current protected host
// configuration and stopped prior workers, following execution010 recovery rules.
type DispatchGate struct {
	mu        sync.Mutex
	store     *execution010.Ledger
	recipient string
	authority Authority
	policy    IntentPolicy
	component Component
	retired   bool
}

func OpenDispatchGate(path string, create bool, recipient string, a Authority, p IntentPolicy, c Component) (*DispatchGate, error) {
	if !did(recipient) || a == nil || p == nil || c == nil {
		return nil, ErrInvalid
	}
	s, err := execution010.Open(path, create)
	if err != nil {
		return nil, ErrInvalid
	}
	return &DispatchGate{store: s, recipient: recipient, authority: a, policy: p, component: c}, nil
}

// Replace is a trusted administrative operation, never driven by a wire digest.
// The host must persist approved old/new baseline and policy mappings externally.
// An earlier committed invocation retains its old pinned instance. No pending
// verification snapshot or local capability survives a configuration change.
func (g *DispatchGate) Replace(a Authority, p IntentPolicy, c Component) error {
	if g == nil || a == nil || p == nil || c == nil {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.store == nil || g.retired {
		return ErrInvalid
	}
	g.authority, g.policy, g.component = a, p, c
	return nil
}

// Retire permanently cancels all not-yet-committed work in this local gate. It
// serializes with Commit, not with completion of an already running tool. The host
// must enforce retirement across receivers and restarts; no rollback is promised.
func (g *DispatchGate) Retire() error {
	if g == nil {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.store == nil {
		return ErrInvalid
	}
	g.retired = true
	return nil
}
func (g *DispatchGate) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.store == nil {
		return nil
	}
	err := g.store.Close()
	g.store = nil
	return err
}
func invocation(v *VerifiedIntent) (*Invocation, error) {
	_, m, _, err := intentEnvelope(v.canonical)
	if err != nil {
		return nil, ErrInvalid
	}
	args, err := Canonicalize(encode(m["arguments"]))
	if err != nil {
		return nil, ErrInvalid
	}
	return &Invocation{canonical: v.Canonical(), arguments: args, tool: str(m, "tool"), manifest: str(m, "manifest_digest"), digest: v.Digest()}, nil
}
func (g *DispatchGate) unknown(e execution010.Entry) {
	e.State = "UNKNOWN"
	// Failure poisons storage and retains its administrative lock. Never publish
	// success or try another effect when this final denial cannot be persisted.
	if _, err := g.store.Commit(e); err != nil {
		g.retired = true
	}
}

// Dispatch verifies inside the same gate as retirement, reserves once, durably
// records EXECUTING, then freshly rechecks authorization, instance and time before
// a bounded commitment. An exact duplicate only returns existing state. A failed
// or panicking post-reservation callback leaves UNKNOWN or unavailable storage.
// No raw terminal output or completion claim is released by this method.
func (g *DispatchGate) Dispatch(ctx context.Context, raw []byte) (receipt *DispatchReceipt, err error) {
	if g == nil || ctx == nil {
		return nil, ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.store == nil || g.retired || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	var pending *execution010.Entry
	defer func() {
		if recover() != nil {
			g.retired = true
			if pending != nil {
				g.unknown(*pending)
			}
			receipt = nil
			err = ErrInvalid
		}
	}()
	v, err := VerifyIntent(ctx, raw, g.recipient, g.authority, g.policy)
	if err != nil {
		return nil, ErrInvalid
	}
	i, err := invocation(v)
	if err != nil || g.component.Check(ctx, i.manifest, i.tool) != nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	i.completion = &Completion{owner: g, canonical: v.Canonical()}
	e, err := reservationEntry(v)
	if err != nil {
		return nil, ErrInvalid
	}
	stored, created, err := g.store.Reserve(e)
	if err != nil {
		return nil, ErrInvalid
	}
	receipt = &DispatchReceipt{reservation: Reservation{created: created, state: stored.State, digest: v.Digest()}, reply: &replyPermit{owner: g, canonical: v.Canonical()}}
	if !created {
		_, m, _, _ := intentEnvelope(v.canonical)
		now, e := g.authority.Now(ctx)
		if e != nil || !times(m, now) || ctx.Err() != nil {
			return nil, ErrInvalid
		}
		return receipt, nil
	}
	pending = &e
	e.State = "EXECUTING"
	if changed, e2 := g.store.Commit(e); e2 != nil || !changed {
		g.retired = true
		return nil, ErrInvalid
	}
	if _, err = VerifyIntent(ctx, v.canonical, g.recipient, g.authority, g.policy); err != nil {
		g.unknown(e)
		return nil, ErrInvalid
	}
	if g.component.Check(ctx, i.manifest, i.tool) != nil {
		g.unknown(e)
		return nil, ErrInvalid
	}
	_, m, _, _ := intentEnvelope(v.canonical)
	now, e2 := g.authority.Now(ctx)
	if e2 != nil || !times(m, now) || ctx.Err() != nil {
		g.unknown(e)
		return nil, ErrInvalid
	}
	if g.component.Commit(ctx, i) != nil {
		g.unknown(e)
		return nil, ErrInvalid
	}
	receipt.reservation.state = "EXECUTING"
	receipt.committed = true
	return receipt, nil
}
