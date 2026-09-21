package guard010

import (
	"sync"
	"time"
)

// mcpOwner is the private lifecycle coordinator. It does not authenticate input
// or authorize effects. The eventual session adapter must authenticate and parse
// each event before calling it and must use a separate owner-aware dispatch gate.
// Only the trusted bounded local clock is sampled under mu. No storage,
// transport, registry lookup or other extensible work runs under this lock.
type mcpOwner struct {
	mu             sync.Mutex
	clock          func() (time.Duration, error)
	phase          mcpPhase
	last, setupEnd time.Duration
	seen           map[string]struct{}
	pending        *mcpOutput
	deferred       []byte
}
type mcpPhase uint8

const (
	mcpClientStart mcpPhase = iota
	mcpClientWaitInitialize
	mcpClientInitialized
	mcpClientWaitAck
	mcpClientNegotiated
	mcpClientWaitList
	mcpServerStart
	mcpServerWaitInitialized
	mcpServerDiscovery
	mcpReady
	mcpClosed
)

type mcpEvent uint8

const (
	mcpSendInitialize mcpEvent = iota
	mcpAcceptInitialize
	mcpSendInitialized
	mcpAcceptInitialized
	mcpSendList
	mcpAcceptList
	mcpReplyInitialize
	mcpReplyInitialized
	mcpReplyList
	mcpProtectedOutput
)

type mcpTransition struct {
	from, to        mcpPhase
	output, request bool
}

var mcpTransitions = [...]mcpTransition{
	{mcpClientStart, mcpClientWaitInitialize, true, true},
	{mcpClientWaitInitialize, mcpClientInitialized, false, false},
	{mcpClientInitialized, mcpClientWaitAck, true, false},
	{mcpClientWaitAck, mcpClientNegotiated, false, false},
	{mcpClientNegotiated, mcpClientWaitList, true, true},
	{mcpClientWaitList, mcpReady, false, false},
	{mcpServerStart, mcpServerWaitInitialized, true, true},
	{mcpServerWaitInitialized, mcpServerDiscovery, true, false},
	{mcpServerDiscovery, mcpReady, true, true},
	{mcpReady, mcpReady, true, false},
}

// mcpOutput is an unexported, owner-bound completion identity. A transport never
// receives it; the adapter retains it while sending an immutable byte snapshot.
type mcpOutput struct {
	owner    *mcpOwner
	next     mcpPhase
	deadline time.Duration
}

// Samples are elapsed monotonic durations from one trusted host clock origin.
// Construction cannot restart the deadline at connection confirmation time.
func newMCPOwner(server bool, created, setupLimit time.Duration, clock func() (time.Duration, error)) (*mcpOwner, error) {
	now, err := sampleMCPClock(clock)
	if err != nil {
		return nil, ErrInvalid
	}
	if created < 0 || now < created || setupLimit <= 0 || setupLimit > 30*time.Second || now-created >= setupLimit {
		return nil, ErrInvalid
	}
	end := created + setupLimit
	if end < created {
		return nil, ErrInvalid
	}
	phase := mcpClientStart
	if server {
		phase = mcpServerStart
	}
	return &mcpOwner{clock: clock, phase: phase, last: now, setupEnd: end, seen: make(map[string]struct{})}, nil
}
func (o *mcpOwner) closeLocked() { o.phase = mcpClosed; o.pending = nil; o.deferred = nil }
func (o *mcpOwner) close()       { o.mu.Lock(); defer o.mu.Unlock(); o.closeLocked() }

// The clock must be bounded, non-reentrant and purely local; RegistryAuthority.Now
// is not a compatible provider. Missing/failed/panicking samples fail closed.
func sampleMCPClock(clock func() (time.Duration, error)) (now time.Duration, err error) {
	defer func() {
		if recover() != nil {
			now = 0
			err = ErrInvalid
		}
	}()
	if clock == nil {
		return 0, ErrInvalid
	}
	return clock()
}
func (o *mcpOwner) validLocked(sessionValid bool) bool {
	now, err := sampleMCPClock(o.clock)
	if o.phase == mcpClosed {
		return false
	}
	if err != nil || !sessionValid || now < o.last || (o.phase != mcpReady && now >= o.setupEnd) {
		o.closeLocked()
		return false
	}
	o.last = now
	return true
}
func (o *mcpOwner) reserveLocked(id string) bool {
	if !uuid.MatchString(id) {
		o.closeLocked()
		return false
	}
	if _, ok := o.seen[id]; ok || len(o.seen) >= 1024 {
		o.closeLocked()
		return false
	}
	o.seen[id] = struct{}{}
	return true
}

// reserveProtected reserves a validated, authenticated protected request ID before
// Guard routing. Closing never releases it, even if later authorization fails.
func (o *mcpOwner) reserveProtected(id string, sessionValid bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.validLocked(sessionValid) {
		return ErrInvalid
	}
	if o.phase != mcpReady || o.pending != nil || o.deferred != nil {
		o.closeLocked()
		return ErrInvalid
	}
	if !o.reserveLocked(id) {
		return ErrInvalid
	}
	return nil
}

// transition accepts only a typed, already authenticated and validated event.
// Setup request IDs are reserved here; response IDs must have been correlated by
// the session adapter, and protected request IDs by reserveProtected. No arbitrary
// target phase or publicly importable readiness value exists.
func (o *mcpOwner) transition(event mcpEvent, id string, protectedEnd time.Duration, sessionValid bool) (*mcpOutput, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.validLocked(sessionValid) {
		return nil, ErrInvalid
	}
	if int(event) >= len(mcpTransitions) || o.pending != nil || o.deferred != nil {
		o.closeLocked()
		return nil, ErrInvalid
	}
	t := mcpTransitions[event]
	if o.phase != t.from || (!t.request && id != "") {
		o.closeLocked()
		return nil, ErrInvalid
	}
	if t.request && !o.reserveLocked(id) {
		return nil, ErrInvalid
	}
	if !t.output {
		o.phase = t.to
		return nil, nil
	}
	end := o.setupEnd
	if event == mcpProtectedOutput {
		end = protectedEnd
		if end <= o.last {
			o.closeLocked()
			return nil, ErrInvalid
		}
	}
	p := &mcpOutput{owner: o, next: t.to, deadline: end}
	o.pending = p
	return p, nil
}

// deferFrame keeps at most one bounded, owned wire snapshot during output. It
// deliberately does not authenticate it or consume replay/JSON-RPC state.
func (o *mcpOwner) deferFrame(wire []byte, sessionValid bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.validLocked(sessionValid) {
		return ErrInvalid
	}
	if o.pending == nil || o.last >= o.pending.deadline || o.deferred != nil || len(wire) == 0 || len(wire) > 32768 {
		o.closeLocked()
		return ErrInvalid
	}
	o.deferred = append([]byte(nil), wire...)
	return nil
}

// complete publishes only a full successful local send for the exact operation.
// Stale callbacks are inert; a failed active send closes without retry. Deferred
// bytes remain private until takeDeferred transfers them to the receive path.
func (o *mcpOwner) complete(p *mcpOutput, fullSend bool, sessionValid bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if p == nil || p.owner != o || o.pending != p {
		return ErrInvalid
	}
	if !o.validLocked(sessionValid) {
		return ErrInvalid
	}
	if !fullSend || o.last >= p.deadline {
		o.closeLocked()
		return ErrInvalid
	}
	o.phase = p.next
	o.pending = nil
	// Keep the deferred slot occupied until the serialized receive path takes it.
	return nil
}

// takeDeferred is used only by the serialized receive path after successful
// publication. It transfers the one owned frame, never a validation claim.
func (o *mcpOwner) takeDeferred() []byte {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.phase == mcpClosed || o.pending != nil {
		return nil
	}
	b := o.deferred
	o.deferred = nil
	return b
}
