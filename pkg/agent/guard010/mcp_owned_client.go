package guard010

import (
	"context"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// A host shares this bounded pool across client owners and reconnects. Clocks
// share the session's trusted origin. No provider runs under either coordinator;
// nested locking is always pool -> owner, never the reverse.
type mcpClientPool struct {
	mu        sync.Mutex
	scheduler *mcpHost
	clock     registry010.Clock
	last      registry010.Stamp
	timeout   time.Duration
	slots     []*mcpClientExchange
	retired   bool
}

func newMCPClientPool(clock registry010.Clock, capacity int, timeout time.Duration) (*mcpClientPool, error) {
	if clock == nil || capacity < 1 || capacity > 128 || timeout <= 0 || timeout > 5*time.Minute {
		return nil, ErrInvalid
	}
	return &mcpClientPool{clock: clock, timeout: timeout, slots: make([]*mcpClientExchange, capacity)}, nil
}
func (p *mcpClientPool) sampleLocked() (s registry010.Stamp, err error) {
	defer func() {
		if recover() != nil {
			p.retired = true
			err = ErrInvalid
		}
	}()
	s, err = p.clock.Now()
	if err != nil || s.MonoMS < 0 || s.Unix < 0 || s.MonoMS < p.last.MonoMS || s.Unix < p.last.Unix || s.MonoMS > int64((1<<63-1)/time.Millisecond) || s.Unix > 9007199254740691 {
		p.retired = true
		return s, ErrInvalid
	}
	p.last = s
	return s, nil
}
func (p *mcpClientPool) retire() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retired = true
	for _, w := range p.slots {
		if w != nil {
			w.cancel()
		}
	}
}

type mcpOwnedClient struct {
	session                    *mcpSetupSession
	pool                       *mcpClientPool
	client                     *Client
	intent                     []byte
	authority, resultAuthority *RegistryAuthority
	policy                     IntentPolicy
	active                     *mcpClientExchange // protected by pool.mu
}
type mcpClientExchange struct {
	binding         *mcpOwnedClient
	session         *mcpSetupSession
	ctx             context.Context
	cancel          context.CancelFunc
	io              mcpSetupIO
	start, deadline time.Duration
	id, outerID     string
	ticket          *ClientInvocation
	accepting       bool
}
type mcpClientEvidence struct {
	start, session time.Duration
	authority      *registryObservation
	expires        int64
}

// openMCPOwnedClient pins an existing exclusive initiator owner to a private
// durable client. The sender cannot be supplied by a plugin or swapped later.
// Creation is trusted bounded host preparation, serialized with session closure.
func openMCPOwnedClient(ctx context.Context, s *mcpSetupSession, p *mcpClientPool, path string, create bool, intent []byte, a, result *RegistryAuthority, policy IntentPolicy, clock ClientClock) (b *mcpOwnedClient, err error) {
	if ctx == nil || s == nil || p == nil || a == nil || result == nil || policy == nil || clock == nil || !s.session.Initiator() {
		return nil, ErrInvalid
	}
	// Size failure before persistence consumes neither a request ID nor a sequence.
	_, _, canonical, canonicalErr := intentEnvelope(append([]byte(nil), intent...))
	if canonicalErr != nil {
		return nil, ErrInvalid
	}
	intent = canonical
	raw, e := MCPRequest(MCPVersion, "00000000-0000-4000-8000-000000000001", intent)
	if e != nil || len(raw) > 16348 {
		return nil, ErrInvalid
	}
	s.mu.Lock()
	if s.running || s.closed || s.client != nil {
		s.mu.Unlock()
		return nil, ErrInvalid
	}
	work, cancel := context.WithTimeout(ctx, p.timeout)
	s.running, s.cancel = true, cancel
	s.mu.Unlock()
	var preparation *mcpClientExchange
	defer func() {
		cancel()
		if recover() != nil {
			err = ErrInvalid
		}
		b, err = finishMCPClientPreparation(s, p, preparation, b, err)
	}()
	p.mu.Lock()
	s.owner.mu.Lock()
	registered := s.owner.scheduler == p.scheduler
	s.owner.mu.Unlock()
	stamp, clockErr := p.sampleLocked()
	now := time.Duration(stamp.MonoMS) * time.Millisecond
	slot := -1
	for n, entry := range p.slots {
		if entry == nil {
			slot = n
			break
		}
	}
	if !registered || p.retired || clockErr != nil || slot < 0 || now > time.Duration(1<<63-1)-p.timeout || work.Err() != nil {
		p.mu.Unlock()
		return nil, ErrInvalid
	}
	preparation = &mcpClientExchange{session: s, ctx: work, cancel: cancel, start: now, deadline: now + p.timeout}
	p.slots[slot] = preparation
	p.mu.Unlock()
	local, peer, e := s.session.Participants()
	if e != nil {
		return nil, ErrInvalid
	}
	if _, e = sessionRequest(MCPVersion, wireField(raw, "id"), raw, local, peer); e != nil {
		return nil, ErrInvalid
	}
	b = &mcpOwnedClient{session: s, pool: p, intent: intent, authority: a, resultAuthority: result, policy: policy}
	b.client, e = OpenClient(work, path, create, intent, ClientServices{IntentAuthority: a, ResultAuthority: result, Policy: policy, Clock: mcpSafeClientClock{clock}, Sender: b, ExpectedIssuer: local, ExpectedRecipient: peer})
	if e != nil {
		return b, ErrInvalid
	}
	p.mu.Lock()
	s.owner.mu.Lock()
	live := s.owner.validLocked(true)
	final, clockErr := p.sampleLocked()
	finalMono := time.Duration(final.MonoMS) * time.Millisecond
	valid := live && !p.retired && clockErr == nil && work.Err() == nil && finalMono >= s.owner.last && finalMono >= preparation.start && finalMono < preparation.deadline
	s.owner.mu.Unlock()
	p.mu.Unlock()
	if !valid {
		return b, ErrInvalid
	}
	s.mu.Lock()
	s.client = b
	s.mu.Unlock()
	return b, nil
}

// exchange encompasses one protected handoff, correlated receive, durable
// consumption and final publication. Only its final return exposes output.
func (b *mcpOwnedClient) exchange(ctx context.Context, io mcpSetupIO) (delivery *ClientDelivery, err error) {
	if b == nil || ctx == nil || io == nil {
		return nil, ErrInvalid
	}
	s, p := b.session, b.pool
	s.mu.Lock()
	if s.running || s.closed || s.client != b || (s.connection != nil && io != s.connection.stream) {
		s.mu.Unlock()
		return nil, ErrInvalid
	}
	work, cancel := context.WithTimeout(ctx, p.timeout)
	s.running, s.cancel = true, cancel
	s.mu.Unlock()
	var w *mcpClientExchange
	defer func() {
		if recover() != nil {
			err = ErrInvalid
			delivery = nil
		}
		cancel()
		if err != nil && w != nil && w.ticket != nil && !w.accepting {
			_ = b.client.Failed(w.ticket)
		}
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
			_ = b.client.Close()
		}
		p.mu.Lock()
		for n, entry := range p.slots {
			if w != nil && entry == w {
				p.slots[n] = nil
			}
		}
		if b.active == w {
			b.active = nil
		}
		p.mu.Unlock()
	}()
	p.mu.Lock()
	s.owner.mu.Lock()
	valid := s.owner.scheduler == p.scheduler && !p.retired && b.active == nil && s.owner.phase == mcpReady && s.owner.pending == nil && s.owner.deferred == nil && s.owner.validLocked(true)
	stamp, e := p.sampleLocked()
	now := time.Duration(stamp.MonoMS) * time.Millisecond
	slot := -1
	for n, entry := range p.slots {
		if entry == nil {
			slot = n
			break
		}
	}
	if !valid || e != nil || slot < 0 || now < s.owner.last || now > time.Duration(1<<63-1)-p.timeout || work.Err() != nil {
		s.owner.mu.Unlock()
		p.mu.Unlock()
		return nil, ErrInvalid
	}
	w = &mcpClientExchange{binding: b, session: s, ctx: work, cancel: cancel, io: io, start: now, deadline: now + p.timeout, id: guuid.NewString()}
	b.active = w
	p.slots[slot] = w
	s.owner.mu.Unlock()
	p.mu.Unlock()
	w.ticket, e = b.client.Begin(work, w.id)
	if e != nil {
		return nil, ErrInvalid
	}
	wire := s.owner.takeDeferred()
	if wire == nil {
		wire, e = s.receive(work, io)
		if e != nil {
			return nil, ErrInvalid
		}
	}
	if len(wire) == 0 || len(wire) > 32768 {
		return nil, ErrInvalid
	}
	response, e := s.session.OpenResponse(work, wire)
	if e != nil || response.MessageID != w.outerID || wireField(wire, "id") == w.id {
		return nil, ErrInvalid
	}
	local, peer, e := s.session.Participants()
	if e != nil {
		return nil, ErrInvalid
	}
	success, code, e := sessionResult(MCPVersion, w.id, response.Data, b.intent, peer, local)
	if e != nil || response.Success != success || response.Error != code {
		return nil, ErrInvalid
	}
	result, e := parseMCPResponse(MCPVersion, w.id, response.Data)
	if e != nil {
		return nil, ErrInvalid
	}
	// Client acceptance persists first. A later failure suppresses output without
	// restoring consumption or claiming that an executed remote effect was undone.
	w.accepting = true
	delivery, e = b.client.AcceptMCPResponse(work, w.ticket, MCPVersion, response.Data)
	if e != nil {
		return nil, ErrInvalid
	}
	evidence, e := b.observe(w, result, true)
	if e != nil {
		return nil, ErrInvalid
	}
	if e = b.publish(w, nil, evidence); e != nil {
		return nil, ErrInvalid
	}
	return delivery, nil
}

// Commit is reachable only while the owned durable client is in Begin. It never
// returns raw session bytes or an invocation token to an untrusted caller.
func (b *mcpOwnedClient) Commit(ctx context.Context, id string, intent []byte) error {
	p, s := b.pool, b.session
	p.mu.Lock()
	w := b.active
	p.mu.Unlock()
	if w == nil || ctx != w.ctx || id != w.id || hash(intent) != hash(b.intent) {
		return ErrInvalid
	}
	if err := s.owner.reserveProtected(id, true); err != nil {
		return err
	}
	output, err := s.owner.transition(mcpProtectedOutput, "", w.deadline, true)
	if err != nil {
		return err
	}
	raw, err := MCPRequest(MCPVersion, id, intent)
	if err != nil || len(raw) > 16348 {
		return ErrInvalid
	}
	wire, err := s.session.SealRequest(ctx, raw, 30)
	if err != nil || len(wire) > 32768 || wireField(wire, "id") == id {
		return ErrInvalid
	}
	w.outerID = wireField(wire, "id")
	if err = w.io.Send(ctx, append([]byte(nil), wire...)); err != nil {
		return ErrInvalid
	}
	evidence, err := b.observe(w, intent, false)
	if err != nil {
		return err
	}
	return b.publish(w, output, evidence)
}
func (b *mcpOwnedClient) observe(w *mcpClientExchange, raw []byte, result bool) (*mcpClientEvidence, error) {
	p := b.pool
	p.mu.Lock()
	stamp, e := p.sampleLocked()
	p.mu.Unlock()
	if e != nil {
		return nil, ErrInvalid
	}
	start := time.Duration(stamp.MonoMS) * time.Millisecond
	a := b.authority
	var expires int64
	if result {
		a = b.resultAuthority
		if _, e = VerifyResult(w.ctx, raw, a, acceptedInvocation(b.intent)); e != nil {
			return nil, ErrInvalid
		}
		env, _, _ := object(raw)
		m, _ := env["result"].(map[string]any)
		expires, _ = number(m, "expires")
	} else {
		_, m, _, e := intentEnvelope(raw)
		if e != nil {
			return nil, ErrInvalid
		}
		if _, e = VerifyIntent(w.ctx, raw, str(m, "recipient"), a, b.policy); e != nil {
			return nil, ErrInvalid
		}
		expires, _ = number(m, "expires")
	}
	authority, e := a.observe(w.ctx)
	if e != nil {
		return nil, ErrInvalid
	}
	session, e := b.session.session.Observe(w.ctx)
	if e != nil {
		return nil, ErrInvalid
	}
	return &mcpClientEvidence{start: start, session: session, authority: authority, expires: expires}, nil
}
func (b *mcpOwnedClient) publish(w *mcpClientExchange, output *mcpOutput, e *mcpClientEvidence) error {
	p, o := b.pool, b.session.owner
	p.mu.Lock()
	defer p.mu.Unlock()
	o.mu.Lock()
	defer o.mu.Unlock()
	valid := o.validLocked(true)
	stamp, err := p.sampleLocked()
	now := time.Duration(stamp.MonoMS) * time.Millisecond
	if err != nil || p.retired || !valid || b.active != w || o.phase != mcpReady || w.ctx.Err() != nil || now < o.last || now < w.start || now >= w.deadline || e == nil || e.start < w.start || now < e.start || now-e.start > 5*time.Second || e.session < w.start || now < e.session || now-e.session > 5*time.Second || stamp.MonoMS < e.authority.stamp.MonoMS || stamp.Unix < e.authority.stamp.Unix || stamp.Unix >= e.expires || (e.authority.expires != nil && stamp.Unix >= *e.authority.expires) {
		o.closeLocked()
		return ErrInvalid
	}
	if output != nil {
		if output.owner != o || o.pending != output || output.deadline != w.deadline {
			o.closeLocked()
			return ErrInvalid
		}
		o.pending = nil
	} else if o.pending != nil || o.deferred != nil {
		o.closeLocked()
		return ErrInvalid
	}
	return nil
}

// Convert trusted clock panics to an ordinary failure while OpenClient still
// owns its descriptor and can execute its own conservative cleanup.
type mcpSafeClientClock struct{ ClientClock }

func (c mcpSafeClientClock) Sample(ctx context.Context) (utc, mono int64, err error) {
	defer func() {
		if recover() != nil {
			utc, mono, err = 0, 0, ErrInvalid
		}
	}()
	return c.ClientClock.Sample(ctx)
}

// Failed construction may own a journal not yet attached to the session. Keep
// cleanup ownership active so the host cannot release this owner's slot early.
func finishMCPClientPreparation(s *mcpSetupSession, p *mcpClientPool, w *mcpClientExchange, b *mcpOwnedClient, err error) (*mcpOwnedClient, error) {
	s.mu.Lock()
	cleanup := s.closed || err != nil
	if cleanup {
		s.closed = true
	} else {
		s.running = false
	}
	s.mu.Unlock()
	if cleanup {
		s.owner.close()
		s.session.Close()
		if b != nil && b.client != nil {
			_ = b.client.Close()
		}
		b = nil
		err = ErrInvalid
	}
	p.mu.Lock()
	for n, entry := range p.slots {
		if w != nil && entry == w {
			p.slots[n] = nil
		}
	}
	p.mu.Unlock()
	if cleanup {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}
	return b, err
}
