package guard010

import (
	"context"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// MCPExecutor is the trusted immutable effect owner. Run returns only after the
// actual effect and dependent cleanup end; poll ctx and bound every callback.
// Never resolve the authenticated tool/arguments into a different loaded instance.
type MCPExecutor interface {
	Check(context.Context, string, string) error
	Run(context.Context, *Invocation) ([]byte, error)
}

// MCPHostServices are protected local capabilities, never model/plugin inputs.
// Clock, registry endpoints and Client clocks share the same monotonic origin.
// Providers and shared signers must be bounded, non-reentrant and concurrency-safe.
type MCPHostServices struct {
	IntentAuthority, ResultAuthority *RegistryAuthority
	Policy                           IntentPolicy
	Executor                         MCPExecutor
	Signer                           ResultSigner
	Clock                            registry010.Clock
}

// MCPHostBounds explicitly selects finite quotas and deadlines. Durations are
// whole milliseconds. Tick is shorter than every logical operation deadline.
type MCPHostBounds struct {
	Capacity, Preparations, Clients, Owners, Workers int
	Request, Claim, Worker, Client, Tick             time.Duration
}

func (b MCPHostBounds) valid() bool {
	if b.Capacity < 1 || b.Capacity > 128 || b.Preparations < b.Capacity || b.Preparations > 128 ||
		b.Clients < 1 || b.Clients > 128 || b.Owners < 1 || b.Owners > 256 || b.Workers < 1 || b.Workers > b.Capacity {
		return false
	}
	for _, d := range []time.Duration{b.Request, b.Claim, b.Worker, b.Client} {
		if d < time.Millisecond || d > 5*time.Minute || d%time.Millisecond != 0 || b.Tick >= d {
			return false
		}
	}
	return b.Tick >= time.Millisecond && b.Tick <= time.Second && b.Tick%time.Millisecond == 0
}

// MCPHost owns admission, the exclusive execution ledger, fixed workers, owner
// supervision and private stream carriage. It exports no gate, READY setter,
// session key, raw transport or worker/reservation token. Close must succeed
// before replacing configuration or reopening the same protected storage.
type MCPHost struct {
	host    *mcpHost
	closeMu sync.Mutex
	closed  bool
}

// OpenMCPHost constructs all coordinators before accepting any peer input.
// create is only for a new isolated scope; recovery never resets lost history.
func OpenMCPHost(path string, create bool, recipient string, s MCPHostServices, b MCPHostBounds) (*MCPHost, error) {
	if !b.valid() || (runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		return nil, ErrInvalid
	}
	g, e := openMCPAdmissionGate(path, create, recipient, &mcpAdmissionConfig{authority: s.IntentAuthority, resultAuthority: s.ResultAuthority, policy: s.Policy, executor: s.Executor, signer: s.Signer}, s.Clock,
		mcpBounds{capacity: b.Capacity, preparations: b.Preparations, request: b.Request, claim: b.Claim, worker: b.Worker})
	if e != nil {
		return nil, e
	}
	p, e := newMCPClientPool(s.Clock, b.Clients, b.Client)
	if e != nil {
		_ = g.close()
		return nil, e
	}
	h, e := newMCPHost(g, p, b.Owners, b.Workers, b.Tick)
	if e != nil {
		p.retire()
		_ = g.close()
		return nil, e
	}
	return &MCPHost{host: h}, nil
}

// MCPClientHostBounds selects finite Client quotas for an initiator-only host.
// Durations are whole milliseconds; Tick is shorter than Client.
type MCPClientHostBounds struct {
	Clients, Owners int
	Client, Tick    time.Duration
}

func (b MCPClientHostBounds) valid() bool {
	return b.Clients >= 1 && b.Clients <= 128 && b.Owners >= 1 && b.Owners <= 256 &&
		b.Client >= time.Millisecond && b.Client <= 5*time.Minute && b.Client%time.Millisecond == 0 &&
		b.Tick >= time.Millisecond && b.Tick <= time.Second && b.Tick%time.Millisecond == 0 && b.Tick < b.Client
}

// OpenMCPClientHost constructs a host that only initiates root Client calls.
// It opens no admission gate, execution ledger, executor, policy or result
// signer, and refuses Serve and responder connections. Use OpenMCPHost for a
// participant that also receives calls, including every hop participant.
func OpenMCPClientHost(clock registry010.Clock, b MCPClientHostBounds) (*MCPHost, error) {
	if !b.valid() || clock == nil || (runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		return nil, ErrInvalid
	}
	p, e := newMCPClientPool(clock, b.Clients, b.Client)
	if e != nil {
		return nil, e
	}
	h, e := newMCPHost(nil, p, b.Owners, 0, b.Tick)
	if e != nil {
		p.retire()
		return nil, e
	}
	return &MCPHost{host: h}, nil
}

// Close permanently retires all rights first, then drains connection/provider,
// effect and cleanup lifetimes. A timeout retains quotas and the ledger lock;
// call Close again after bounded providers finish. Success releases storage.
func (h *MCPHost) Close(ctx context.Context) error {
	if h == nil || h.host == nil || ctx == nil {
		return ErrInvalid
	}
	h.closeMu.Lock()
	defer h.closeMu.Unlock()
	if h.closed {
		return nil
	}
	if e := h.host.stop(ctx); e != nil {
		return e
	}
	if h.host.gate != nil {
		if e := h.host.gate.close(); e != nil {
			return e
		}
	}
	h.closed = true
	return nil
}

// MCPRole is selected by the trusted host, never negotiated from peer bytes.
type MCPRole uint8

const (
	MCPResponder MCPRole = 1
	MCPInitiator MCPRole = 2
)

// MCPConnectionConfig pins local identity presentation and the initiator's exact
// recipient/key tuple. Timeout bounds the complete authenticated setup. Native
// carriage is a deployment-selected four-byte length framing, not MCP stdio.
type MCPConnectionConfig struct {
	Role                                   MCPRole
	Recipient, RecipientKey, Name, Version string
	TTLSeconds                             int64
	Timeout                                time.Duration
}

func (c MCPConnectionConfig) valid() bool {
	return (c.Role == MCPResponder || c.Role == MCPInitiator) && c.TTLSeconds >= 1 && c.TTLSeconds <= 300 &&
		c.Timeout >= time.Millisecond && c.Timeout <= 30*time.Second && c.Timeout%time.Millisecond == 0 && len(c.Name)+len(c.Version) <= 16000 && utf8.ValidString(c.Name) && utf8.ValidString(c.Version) &&
		((c.Role == MCPResponder && c.Recipient == "" && c.RecipientKey == "") ||
			(c.Role == MCPInitiator && did(c.Recipient) && mcpPublicKey(c.RecipientKey, c.Recipient)))
}

func mcpPublicKey(key, principal string) bool {
	p := strings.Split(key, "#")
	return len(p) == 2 && p[0] == principal && keyName.MatchString(p[1])
}

// MCPConnectionHandler is trusted host code. Endpoint constructs a fresh,
// exclusively transferred endpoint after connection quota reservation. Prepare
// binds actual local readiness before initialized acknowledgement. Handle runs
// only after authenticated setup; it must finish all operations before returning.
// Serve calls shared handlers concurrently on its fixed connection workers.
type MCPConnectionHandler interface {
	Endpoint(context.Context) (*hpke.CompletionEndpoint010, error)
	Prepare(context.Context) error
	Handle(context.Context, *MCPConnection) error
}

func (h *MCPHost) config(c MCPConnectionConfig, handler MCPConnectionHandler) (mcpConnectionConfig, error) {
	if h == nil || h.host == nil || !c.valid() || handler == nil {
		return mcpConnectionConfig{}, ErrInvalid
	}
	return mcpConnectionConfig{endpointFactory: handler.Endpoint, initiator: c.Role == MCPInitiator, recipient: c.Recipient, recipientKey: c.RecipientKey, name: c.Name, version: c.Version, ttl: c.TTLSeconds, timeout: c.Timeout}, nil
}
func (h *MCPHost) handle(handler MCPConnectionHandler) func(context.Context, *mcpSetupSession, mcpSetupIO) error {
	return func(ctx context.Context, s *mcpSetupSession, io mcpSetupIO) error {
		state := &mcpPublicConnection{host: h.host, session: s, io: io, ctx: ctx, active: true}
		c := &MCPConnection{state: state}
		defer func() { state.mu.Lock(); state.active = false; state.mu.Unlock() }()
		return handler.Handle(ctx, c)
	}
}

// Connect transfers exclusive socket ownership even on rejection. It creates no
// connection worker: call synchronously from a bounded host worker. Keep no alias.
func (h *MCPHost) Connect(ctx context.Context, conn net.Conn, c MCPConnectionConfig, handler MCPConnectionHandler) error {
	cfg, e := h.config(c, handler)
	if e != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return e
	}
	return h.host.connection(ctx, conn, cfg, handler.Prepare, h.handle(handler))
}

// Serve owns the listener and exactly workers accept/connection workers. All
// endpoints come from the charged handler. Cancellation or Close stops acceptance.
func (h *MCPHost) Serve(ctx context.Context, listener net.Listener, workers int, c MCPConnectionConfig, handler MCPConnectionHandler) error {
	cfg, e := h.config(c, handler)
	if e != nil {
		if listener != nil {
			_ = listener.Close()
		}
		return e
	}
	return h.host.serve(ctx, listener, workers, cfg, handler.Prepare, h.handle(handler))
}

// MCPClientServices pins the trusted outbound Guard services. The sender and
// local/peer tuple are always taken from this connection's authenticated owner.
type MCPClientServices struct {
	IntentAuthority, ResultAuthority *RegistryAuthority
	Policy                           IntentPolicy
	Clock                            ClientClock
}

// MCPHopServices supplies the current admitted parent and its trusted verifier.
// A caller-created ordinary DispatchGate invocation has no MCP parent authority.
type MCPHopServices struct {
	Parent    *Invocation
	Authority Authority
	Policy    IntentPolicy
}

// MCPConnection is an opaque callback-scoped owner. Methods serialize and use
// the original connection context and sole private stream. A retained handle is
// invalid after Handle returns. It cannot replace the sender or force readiness.
type MCPConnection struct{ state *mcpPublicConnection }

type mcpPublicConnection struct {
	mu      sync.Mutex
	host    *mcpHost
	session *mcpSetupSession
	io      mcpSetupIO
	ctx     context.Context
	active  bool
}

func (c *mcpPublicConnection) live() bool {
	return c != nil && c.active && c.host != nil && c.session != nil && c.io != nil && c.ctx != nil && c.ctx.Err() == nil
}

// ServeOne receives exactly one protected request, crosses owner-aware durable
// admission and publishes one verified signed response. A response may be pending;
// it is not a promise that the effect has completed or that the peer accepted it.
func (c *MCPConnection) ServeOne() (err error) {
	if c == nil || c.state == nil {
		return ErrInvalid
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live() {
		return ErrInvalid
	}
	defer func() {
		if err != nil {
			s.session.retireOnly()
		}
	}()
	if s.session.session.Initiator() {
		return ErrInvalid
	}
	wire, e := s.io.Receive(s.ctx)
	if e != nil {
		return e
	}
	if _, e = s.session.admitProtected(s.ctx, wire, s.host.gate); e != nil {
		return e
	}
	return s.session.replyProtected(s.ctx, s.io)
}

// OpenRootClient binds a signed intent to an independently captured original
// before journaling. Verified bytes are not new authorization; host policy remains
// mandatory. One owner gets one Client; recovery keeps the same journal identity.
func (c *MCPConnection) OpenRootClient(path string, create bool, intent []byte, services MCPClientServices, capture *RootCapture) (err error) {
	if c == nil || c.state == nil {
		return ErrInvalid
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live() {
		return ErrInvalid
	}
	_, err = openMCPOwnedRootClient(s.ctx, s.session, s.host.clients, path, create, intent, services.IntentAuthority, services.ResultAuthority, services.Policy, services.Clock, capture)
	if err != nil {
		s.session.retireOnly()
	}
	return err
}

// OpenHopClient uses only the parent minted by the actually admitted live worker.
// Root input cannot be substituted for authenticated downstream provenance.
func (c *MCPConnection) OpenHopClient(path string, create bool, intent []byte, services MCPClientServices, parent MCPHopServices) (err error) {
	if c == nil || c.state == nil {
		return ErrInvalid
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live() {
		return ErrInvalid
	}
	_, err = openMCPOwnedHopClient(s.ctx, s.session, s.host.clients, path, create, parent.Parent, intent, parent.Authority, parent.Policy, services.IntentAuthority, services.ResultAuthority, services.Policy, services.Clock)
	if err != nil {
		s.session.retireOnly()
	}
	return err
}

// Exchange uses only this owner's previously opened Client and authenticated
// stream. It returns journaled verified delivery, never an untrusted raw response.
func (c *MCPConnection) Exchange() (*ClientDelivery, error) {
	if c == nil || c.state == nil {
		return nil, ErrInvalid
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live() {
		return nil, ErrInvalid
	}
	s.session.mu.Lock()
	client := s.session.client
	s.session.mu.Unlock()
	if client == nil {
		s.session.retireOnly()
		return nil, ErrInvalid
	}
	return client.exchange(s.ctx, s.io)
}
