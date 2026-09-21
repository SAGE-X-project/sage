package guard010

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// mcpHost provides a fixed worker set, independent deadline sweep and bounded
// owner cleanup. It is private trusted host plumbing, not a wire capability.
// The runtime must schedule its timer and providers within their configured bounds.
// If they stall, occupied slots stay retained and later admissions still recheck
// absolute deadlines. This is not a hard real-time or whole-host isolation claim.
type mcpHost struct {
	mu             sync.Mutex
	gate           *mcpAdmissionGate
	clients        *mcpClientPool
	owners         []*mcpHostOwner
	interval       time.Duration
	wake, cleanup  chan struct{}
	stopping, done chan struct{}
	once           sync.Once
	workers        atomic.Int64
}
type mcpHostOwner struct {
	session  *mcpSetupSession
	cleaning bool
}

func newMCPHost(g *mcpAdmissionGate, p *mcpClientPool, owners, workers int, interval time.Duration) (*mcpHost, error) {
	if g == nil || owners < 1 || owners > 256 || workers < 1 || workers > len(g.slots) || interval <= 0 || interval > time.Second || interval >= g.bounds.claim || interval >= g.bounds.request || (p != nil && interval >= p.timeout) {
		return nil, ErrInvalid
	}
	h := &mcpHost{gate: g, clients: p, owners: make([]*mcpHostOwner, owners), interval: interval, wake: make(chan struct{}, workers), cleanup: make(chan struct{}, 1), stopping: make(chan struct{}), done: make(chan struct{})}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.scheduler != nil || g.retired {
		return nil, ErrInvalid
	}
	for _, w := range g.slots {
		if w != nil {
			return nil, ErrInvalid
		}
	}
	for _, w := range g.outputs {
		if w != nil {
			return nil, ErrInvalid
		}
	}
	if p != nil {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.scheduler != nil || p.retired {
			return nil, ErrInvalid
		}
		for _, w := range p.slots {
			if w != nil {
				return nil, ErrInvalid
			}
		}
		p.scheduler = h
	}
	g.scheduler = h
	h.workers.Store(int64(workers))
	for range workers {
		go h.worker()
	}
	go h.cleaner()
	go h.watch()
	return h, nil
}
func (h *mcpHost) notify() {
	for range cap(h.wake) {
		select {
		case h.wake <- struct{}{}:
		default:
		}
	}
}
func (h *mcpHost) cleanupSoon() {
	select {
	case h.cleanup <- struct{}{}:
	default:
	}
}

// Registration precedes setup and makes owner/reconnect capacity explicit. Failed
// registration never transfers ownership; the trusted creator must close its owner.
func (h *mcpHost) register(s *mcpSetupSession) error {
	if s == nil {
		return ErrInvalid
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.stopping:
		return ErrInvalid
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.running || s.closed || s.host != nil || (s.client != nil && s.client.pool != h.clients) || (!s.session.Initiator() && s.admission != h.gate) {
		return ErrInvalid
	}
	for n, entry := range h.owners {
		if entry == nil {
			s.owner.mu.Lock()
			s.owner.scheduler = h
			s.owner.mu.Unlock()
			s.host = h
			h.owners[n] = &mcpHostOwner{session: s}
			return nil
		}
	}
	return ErrInvalid
}
func (h *mcpHost) snapshot() []*mcpHostOwner {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*mcpHostOwner(nil), h.owners...)
}
func (h *mcpHost) worker() {
	defer h.workers.Add(-1)
	for {
		for {
			ran, err := h.gate.runOne(context.Background())
			if !ran && err == nil {
				break
			}
		}
		select {
		case <-h.stopping:
			return
		default:
		}
		select {
		case <-h.wake:
		case <-h.stopping:
			return
		}
	}
}

// sweep only samples trusted local clocks, retires rights and requests cancellation.
// No registry, journal, tool or key-cleanup work may block this timer path.
func (h *mcpHost) sweep() {
	expired := make(map[*mcpSetupSession]bool)
	g := h.gate
	g.mu.Lock()
	stamp, err := g.sampleLocked()
	now := time.Duration(stamp.MonoMS) * time.Millisecond
	gateFailed := err != nil || g.retired
	for _, w := range g.slots {
		if w != nil {
			if gateFailed || (!w.claimed && w.queued && (now < w.enqueued || now-w.enqueued >= g.bounds.claim)) {
				w.cancelled = true
			}
			if gateFailed || (w.claimed && now >= w.workerEnd) {
				if w.cancel != nil {
					w.cancel()
				}
			}
			if gateFailed || ((w.owner.active == w || w.owner.response == w.response) && now >= w.deadline) {
				w.owner.closeLocked()
				expired[w.adapter] = true
			}
		}
	}
	for _, p := range g.outputs {
		if p != nil && (gateFailed || now >= p.deadline) {
			p.owner.closeLocked()
			expired[p.response.owner] = true
			if p.cancel != nil {
				p.cancel()
			}
		}
	}
	g.mu.Unlock()
	clientFailed := false
	if h.clients != nil {
		p := h.clients
		p.mu.Lock()
		stamp, e := p.sampleLocked()
		clientFailed = e != nil || p.retired
		now := time.Duration(stamp.MonoMS) * time.Millisecond
		for _, w := range p.slots {
			if w != nil && (e != nil || p.retired || now >= w.deadline) {
				w.session.owner.mu.Lock()
				w.session.owner.closeLocked()
				w.session.owner.mu.Unlock()
				expired[w.session] = true
				w.cancel()
			}
		}
		p.mu.Unlock()
	}
	for _, entry := range h.snapshot() {
		if entry != nil {
			s := entry.session
			o := s.owner
			o.mu.Lock()
			live := o.validLocked(true)
			deadline := time.Duration(0)
			if o.pending != nil {
				deadline = o.pending.deadline
			}
			if o.response != nil && (deadline == 0 || o.response.deadline < deadline) {
				deadline = o.response.deadline
			}
			fail := !live || (s.session.Initiator() && clientFailed) || expired[s] || (s.admission == g && gateFailed) || (deadline != 0 && o.last >= deadline)
			if fail {
				o.closeLocked()
			}
			o.mu.Unlock()
			if fail {
				s.retireOnly()
			}
		}
	}
	h.notify()
	h.cleanupSoon()
}
func (s *mcpSetupSession) retireOnly() {
	s.owner.close()
	s.mu.Lock()
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
}
func (h *mcpHost) cleaner() {
	for {
		select {
		case <-h.cleanup:
		case <-h.done:
			return
		}
		for {
			var selected *mcpHostOwner
			index := -1
			h.mu.Lock()
			for n, entry := range h.owners {
				if entry != nil && !entry.cleaning {
					entry.session.mu.Lock()
					ready := entry.session.closed && !entry.session.running
					entry.session.mu.Unlock()
					if ready {
						entry.cleaning = true
						selected = entry
						index = n
						break
					}
				}
			}
			h.mu.Unlock()
			if selected == nil {
				break
			}
			selected.session.close()
			h.mu.Lock()
			if h.owners[index] == selected {
				h.owners[index] = nil
			}
			h.mu.Unlock()
		}
	}
}
func (h *mcpHost) idle() bool {
	if h.workers.Load() != 0 {
		return false
	}
	h.mu.Lock()
	for _, entry := range h.owners {
		if entry != nil {
			h.mu.Unlock()
			return false
		}
	}
	h.mu.Unlock()
	h.gate.mu.Lock()
	for _, w := range h.gate.slots {
		if w != nil {
			h.gate.mu.Unlock()
			return false
		}
	}
	for _, p := range h.gate.outputs {
		if p != nil {
			h.gate.mu.Unlock()
			return false
		}
	}
	h.gate.mu.Unlock()
	if h.clients != nil {
		h.clients.mu.Lock()
		defer h.clients.mu.Unlock()
		for _, w := range h.clients.slots {
			if w != nil {
				return false
			}
		}
	}
	return true
}
func (h *mcpHost) watch() {
	timer := time.NewTicker(h.interval)
	defer timer.Stop()
	for range timer.C {
		h.sweep()
		select {
		case <-h.stopping:
			if h.idle() {
				close(h.done)
				return
			}
		default:
		}
	}
}

// stop first revokes rights, then waits only as long as the caller allows. A
// timeout does not release slots, close active storage or claim providers stopped.
// The same host remains responsible for cleanup and may be waited on again.
func (h *mcpHost) stop(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	h.once.Do(func() {
		h.gate.retire()
		if h.clients != nil {
			h.clients.retire()
		}
		close(h.stopping)
		for _, entry := range h.snapshot() {
			if entry != nil {
				entry.session.retireOnly()
			}
		}
		h.notify()
		h.cleanupSoon()
	})
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
