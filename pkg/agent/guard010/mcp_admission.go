package guard010

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/execution010"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// The executor is a pinned immutable host instance. Run encompasses the actual
// effect and its termination, not an asynchronous handoff. Trusted providers must
// have finite completion/cancellation bounds; an unexpected stall retains capacity.
type mcpExecutor interface {
	Check(context.Context, string, string) error
	Run(context.Context, *Invocation) ([]byte, error)
}
type mcpAdmissionConfig struct {
	authority       *RegistryAuthority
	resultAuthority *RegistryAuthority
	policy          IntentPolicy
	executor        mcpExecutor
	signer          ResultSigner
}
type mcpBounds struct {
	capacity               int
	request, claim, worker time.Duration
}

// This gate is private until result carriage and host scheduling are integrated.
// mu coordinates all attached owners, queue insertion/claim and administration.
// The execution mutex is ledger.mu. Only execution -> coordinator nesting is used.
type mcpAdmissionGate struct {
	mu         sync.Mutex
	ledger     *DispatchGate
	clock      registry010.Clock
	last       registry010.Stamp
	config     *mcpAdmissionConfig
	bounds     mcpBounds
	generation uint64
	retired    bool
	slots      []*mcpWork
	outputs    []*mcpOutput
	commit     func(execution010.Entry) (bool, error)
}
type mcpWork struct {
	owner                      *mcpOwner
	session                    *hpke.NonHTTPOwner010
	adapter                    *mcpSetupSession
	config                     *mcpAdmissionConfig
	generation                 uint64
	start, deadline, enqueued  time.Duration
	wire                       []byte
	invocation                 *Invocation
	entry                      execution010.Entry
	queued, claimed, cancelled bool
	cancel                     context.CancelFunc
	response                   *mcpProtectedReply
}

// The underlying legacy gate cannot hand off an invocation through this adapter.
type mcpNoDirectCommit struct{ executor mcpExecutor }

func (c mcpNoDirectCommit) Check(ctx context.Context, m, t string) error {
	return c.executor.Check(ctx, m, t)
}
func (c mcpNoDirectCommit) Commit(context.Context, *Invocation) error { return ErrInvalid }
func validMCPConfig(c *mcpAdmissionConfig) bool {
	return c != nil && c.authority != nil && c.resultAuthority != nil && c.policy != nil && c.executor != nil && c.signer != nil
}
func openMCPAdmissionGate(path string, create bool, recipient string, c *mcpAdmissionConfig, clock registry010.Clock, b mcpBounds) (*mcpAdmissionGate, error) {
	if !validMCPConfig(c) || clock == nil || b.capacity < 1 || b.capacity > 128 || b.request <= 0 || b.request > 5*time.Minute || b.claim <= 0 || b.claim > 5*time.Minute || b.worker <= 0 || b.worker > time.Hour {
		return nil, ErrInvalid
	}
	d, err := OpenDispatchGate(path, create, recipient, c.authority, c.policy, mcpNoDirectCommit{c.executor})
	if err != nil {
		return nil, err
	}
	copy := *c
	g := &mcpAdmissionGate{ledger: d, clock: clock, config: &copy, bounds: b, generation: 1, slots: make([]*mcpWork, b.capacity), outputs: make([]*mcpOutput, b.capacity), commit: d.store.Commit}
	return g, nil
}
func newMCPGuardSetup(s *hpke.AuthenticatedCompletion010, name, version string, g *mcpAdmissionGate) (*mcpSetupSession, error) {
	if g == nil {
		return nil, ErrInvalid
	}
	a, err := newMCPSetupSession(s, name, version)
	if err != nil {
		return nil, err
	}
	// Construction has not published the owner to any callback or transport.
	a.owner.mu = &g.mu
	a.owner.admission = g
	a.admission = g
	return a, nil
}
func (g *mcpAdmissionGate) sampleLocked() (t registry010.Stamp, err error) {
	defer func() {
		if recover() != nil {
			err = ErrInvalid
			g.retired = true
		}
	}()
	t, err = g.clock.Now()
	if err != nil || t.MonoMS < g.last.MonoMS || t.Unix < g.last.Unix || t.MonoMS > int64((1<<63-1)/time.Millisecond) || t.Unix > 9007199254740691 {
		g.retired = true
		return t, ErrInvalid
	}
	g.last = t
	return t, nil
}
func (g *mcpAdmissionGate) begin(o *mcpOwner, wire []byte) (*mcpWork, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if o.mu != &g.mu || g.retired || o.phase != mcpReady || o.pending != nil || o.deferred != nil || o.response != nil || o.active != nil || len(wire) == 0 || len(wire) > 32768 || !o.validLocked(true) {
		return nil, ErrInvalid
	}
	t, err := g.sampleLocked()
	if err != nil {
		return nil, err
	}
	start := time.Duration(t.MonoMS) * time.Millisecond
	if start < o.last || start > time.Duration(1<<63-1)-g.bounds.request {
		return nil, ErrInvalid
	}
	for n, slot := range g.slots {
		if slot == nil {
			w := &mcpWork{owner: o, config: g.config, generation: g.generation, start: start, deadline: start + g.bounds.request, wire: append([]byte(nil), wire...)}
			g.slots[n] = w
			o.active = w
			return w, nil
		}
	}
	return nil, ErrInvalid
}
func (g *mcpAdmissionGate) release(w *mcpWork) {
	g.mu.Lock()
	defer func() {
		closed := w.owner.phase == mcpClosed
		g.mu.Unlock()
		if closed && w.adapter != nil {
			w.adapter.close()
		}
	}()
	if t, err := g.sampleLocked(); err != nil || ((w.owner.active == w || w.owner.response == w.response) && time.Duration(t.MonoMS)*time.Millisecond >= w.deadline) {
		w.owner.closeLocked()
	}
	for n, v := range g.slots {
		if v == w {
			g.slots[n] = nil
			break
		}
	}
	if w.owner.active == w {
		w.owner.active = nil
	}
}
func (g *mcpAdmissionGate) poison() { g.mu.Lock(); g.retired = true; g.mu.Unlock() }

// Called under the execution mutex, never under the coordinator.
func (g *mcpAdmissionGate) unknown(e execution010.Entry) {
	current, ok, err := g.ledger.store.Lookup(e.Issuer, e.CallID)
	if err != nil || !ok {
		g.ledger.retired = true
		g.poison()
		return
	}
	if current.State == "COMPLETED" || current.State == "REJECTED" || current.State == "UNKNOWN" {
		return
	}
	current.State = "UNKNOWN"
	if changed, err := g.commit(current); err != nil || !changed {
		g.ledger.retired = true
		g.poison()
	}
}

// admitProtected owns one bounded authenticated request through final queue
// insertion. The caller receives only storage metadata, never the private work.
func (s *mcpSetupSession) admitProtected(ctx context.Context, wire []byte, g *mcpAdmissionGate) (r *DispatchReceipt, err error) {
	if ctx == nil || g == nil || s.admission != g || s.session.Initiator() {
		return nil, ErrInvalid
	}
	w, err := g.begin(s.owner, wire)
	if err != nil {
		s.close()
		return nil, err
	}
	work, cancel := context.WithTimeout(ctx, g.bounds.request)
	s.mu.Lock()
	if s.running || s.closed {
		s.mu.Unlock()
		cancel()
		g.release(w)
		s.close()
		return nil, ErrInvalid
	}
	s.running, s.cancel = true, cancel
	s.mu.Unlock()
	w.session, w.adapter = s.session, s
	defer func() {
		if recover() != nil {
			g.poison()
			err = ErrInvalid
			r = nil
		}
		cancel()
		s.mu.Lock()
		s.running = false
		cleanup := s.closed || err != nil
		if cleanup {
			s.closed = true
		}
		s.mu.Unlock()
		if cleanup {
			s.owner.close()
			s.session.Close()
		}
		if r == nil || !r.Committed() {
			g.release(w)
		}
	}()
	raw, err := s.session.OpenRequest(work, w.wire)
	if err != nil {
		s.close()
		return nil, ErrInvalid
	}
	local, peer, err := s.session.Participants()
	if err != nil || local != g.ledger.recipient {
		s.close()
		return nil, ErrInvalid
	}
	id := wireField(raw, "id")
	g.mu.Lock()
	valid := s.owner.phase == mcpReady && s.owner.active == w && s.owner.validLocked(true) && s.owner.reserveLocked(id)
	g.mu.Unlock()
	if !valid || id == wireField(w.wire, "id") {
		s.close()
		return nil, ErrInvalid
	}
	intent, err := sessionRequest(MCPVersion, id, raw, peer, local)
	if err != nil {
		s.close()
		return nil, ErrInvalid
	}
	w.response = &mcpProtectedReply{owner: s, gate: g, config: w.config, generation: w.generation, start: w.start, deadline: w.deadline, innerID: id, outerID: wireField(w.wire, "id"), intent: append([]byte(nil), intent...)}
	return g.fenceAndAdmit(work, w, intent)
}
func (g *mcpAdmissionGate) fenceAndAdmit(ctx context.Context, w *mcpWork, raw []byte) (receipt *DispatchReceipt, err error) {
	d := g.ledger
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.store == nil || d.retired || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	var pending *execution010.Entry
	defer func() {
		if recover() != nil {
			g.poison()
			d.retired = true
			if pending != nil {
				g.unknown(*pending)
			}
			receipt = nil
			err = ErrInvalid
		}
	}()
	c := w.config
	v, err := VerifyIntent(ctx, raw, d.recipient, c.authority, c.policy)
	if err != nil {
		return nil, ErrInvalid
	}
	i, err := invocation(v)
	if err != nil || c.executor.Check(ctx, i.manifest, i.tool) != nil {
		return nil, ErrInvalid
	}
	e, err := reservationEntry(v)
	if err != nil {
		return nil, ErrInvalid
	}
	stored, created, err := d.store.Reserve(e)
	if err != nil {
		if !errors.Is(err, execution010.ErrDenied) {
			d.retired = true
			g.poison()
		}
		return nil, ErrInvalid
	}
	receipt = &DispatchReceipt{reservation: Reservation{created: created, state: stored.State, digest: v.Digest()}, reply: &replyPermit{owner: d, canonical: v.Canonical()}}
	if created {
		pending = &e
		e.State = "EXECUTING"
		if changed, x := g.commit(e); x != nil || !changed {
			d.retired = true
			g.poison()
			return nil, ErrInvalid
		}
	}
	// Capture the beginning of all final validation, not a later positive-cache time.
	g.mu.Lock()
	sample, x := g.sampleLocked()
	g.mu.Unlock()
	if x != nil {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	observed := time.Duration(sample.MonoMS) * time.Millisecond
	if _, err = VerifyIntent(ctx, v.canonical, d.recipient, c.authority, c.policy); err != nil {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	if c.executor.Check(ctx, i.manifest, i.tool) != nil {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	observation, err := c.authority.observe(ctx)
	if err != nil {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	sessionObserved, err := w.session.Observe(ctx)
	if err != nil {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	_, m, _, _ := intentEnvelope(v.canonical)
	g.mu.Lock()
	ownerLive := w.owner.validLocked(true)
	now, x := g.sampleLocked()
	mono := time.Duration(now.MonoMS) * time.Millisecond
	valid := mono >= w.owner.last && sessionObserved >= w.start && mono >= sessionObserved && mono-sessionObserved <= 5*time.Second && x == nil && !g.retired && w.generation == g.generation && w.owner.active == w && w.owner.phase == mcpReady && ownerLive && ctx.Err() == nil && mono >= w.start && mono < w.deadline && observed >= w.start && mono >= observed && mono-observed <= 5*time.Second && now.MonoMS >= observation.stamp.MonoMS && now.Unix >= observation.stamp.Unix && times(m, now.Unix) && (observation.expires == nil || now.Unix < *observation.expires)
	if valid {
		w.response.receipt = receipt
		w.owner.response = w.response
	}
	if valid && created {
		i.completion = &Completion{owner: d, canonical: v.Canonical()}
		w.invocation = i
		w.entry = e
		w.enqueued = mono
		w.queued = true
	}
	g.mu.Unlock()
	if !valid {
		if created {
			g.unknown(e)
		}
		return nil, ErrInvalid
	}
	if created {
		receipt.reservation.state = "EXECUTING"
		receipt.committed = true
	}
	return receipt, nil
}

// replace invalidates pending generation snapshots and queued, unclaimed work.
// Running work retains its pinned executor/signer and receives policy cancellation.
func (g *mcpAdmissionGate) replace(c *mcpAdmissionConfig) error {
	if !validMCPConfig(c) {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.retired || g.generation == ^uint64(0) {
		return ErrInvalid
	}
	copy := *c
	g.config = &copy
	g.generation++
	for _, p := range g.outputs {
		if p != nil && p.cancel != nil {
			p.cancel()
		}
	}
	for _, w := range g.slots {
		if w != nil {
			if w.queued && !w.claimed {
				w.cancelled = true
			}
			if w.claimed && w.cancel != nil {
				w.cancel()
			}
		}
	}
	return nil
}

// runOne is invoked by a bounded trusted host worker. It claims exactly one slot;
// no tool, signer, storage or extensible callback runs under the coordinator.
func (g *mcpAdmissionGate) runOne(ctx context.Context) (ran bool, err error) {
	if ctx == nil {
		return false, ErrInvalid
	}
	g.mu.Lock()
	var w *mcpWork
	for _, candidate := range g.slots {
		if candidate != nil && candidate.queued && !candidate.claimed {
			w = candidate
			break
		}
	}
	if w == nil {
		g.mu.Unlock()
		return false, nil
	}
	t, e := g.sampleLocked()
	now := time.Duration(t.MonoMS) * time.Millisecond
	if e != nil || ((w.owner.active == w || w.owner.response == w.response) && now >= w.deadline) {
		w.owner.closeLocked()
	}
	cancelled := e != nil || g.retired || w.cancelled || w.generation != g.generation || now < w.enqueued || now-w.enqueued >= g.bounds.claim || ctx.Err() != nil
	w.claimed = true
	work, cancel := context.WithTimeout(ctx, g.bounds.worker)
	w.cancel = cancel
	closed := w.owner.phase == mcpClosed
	g.mu.Unlock()
	if closed {
		w.adapter.close()
	}
	defer cancel()
	defer g.release(w)
	defer func() {
		if recover() != nil {
			g.ledger.mu.Lock()
			g.unknown(w.entry)
			g.ledger.mu.Unlock()
			err = ErrInvalid
		}
	}()
	if cancelled {
		g.ledger.mu.Lock()
		g.unknown(w.entry)
		g.ledger.mu.Unlock()
		return false, ErrInvalid
	}
	output, err := w.config.executor.Run(work, w.invocation)
	if err != nil {
		g.ledger.mu.Lock()
		g.unknown(w.entry)
		g.ledger.mu.Unlock()
		return true, ErrInvalid
	}
	// Completion persists exact signed output; transport delivery is separate.
	if err = g.ledger.Finish(work, w.invocation.Completion(), output, w.config.signer); err != nil {
		g.ledger.mu.Lock()
		if g.ledger.retired {
			g.poison()
		}
		g.unknown(w.entry)
		g.ledger.mu.Unlock()
		return true, ErrInvalid
	}
	return true, nil
}
func (g *mcpAdmissionGate) retire() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.retired = true
	for _, p := range g.outputs {
		if p != nil && p.cancel != nil {
			p.cancel()
		}
	}
	for _, w := range g.slots {
		if w != nil {
			if !w.claimed {
				w.cancelled = true
			}
			if w.cancel != nil {
				w.cancel()
			}
		}
	}
}
func (g *mcpAdmissionGate) close() error {
	g.retire()
	g.mu.Lock()
	busy := false
	for _, p := range g.outputs {
		busy = busy || p != nil
	}
	for _, w := range g.slots {
		busy = busy || w != nil
	}
	g.mu.Unlock()
	if busy {
		return ErrInvalid
	}
	return g.ledger.Close()
}
