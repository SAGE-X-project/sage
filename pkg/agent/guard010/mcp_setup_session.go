package guard010

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

// mcpSetupIO belongs to the trusted host. Send reports full local handoff only;
// Receive returns one complete frame. Both honor cancellation with a finite bound,
// never reenter the adapter, and retain at most one bounded incoming frame while
// Send is pending. This adapter creates no goroutine per I/O or reconnect.
type mcpSetupIO interface {
	Send(context.Context, []byte) error
	Receive(context.Context) ([]byte, error)
}

// mcpSetupSession is private until owner-aware Guard admission is integrated.
// Negotiating this object alone grants no protected dispatch or public readiness.
type mcpSetupSession struct {
	admission       *mcpAdmissionGate
	client          *mcpOwnedClient
	host            *mcpHost
	connection      *mcpHostConnection
	mu              sync.Mutex
	running, closed bool
	started         bool
	cancel          context.CancelFunc
	owner           *mcpOwner
	session         *hpke.NonHTTPOwner010
	info            json.RawMessage
}

func newMCPSetupSession(s *hpke.AuthenticatedCompletion010, name, version string) (*mcpSetupSession, error) {
	info := encode(map[string]any{"name": name, "version": version})
	if len(info) > 16000 {
		return nil, ErrInvalid
	}
	owned, err := s.TakeNonHTTP()
	if err != nil {
		return nil, ErrInvalid
	}
	created := owned.CreatedMonoMS()
	if created < 0 || created > int64((1<<63-1)/time.Millisecond) {
		owned.Close()
		return nil, ErrInvalid
	}
	o, err := newMCPOwner(!owned.Initiator(), time.Duration(created)*time.Millisecond, 30*time.Second, owned.LocalNow)
	if err != nil {
		owned.Close()
		return nil, ErrInvalid
	}
	return &mcpSetupSession{owner: o, session: owned, info: info}, nil
}

// close retires publication immediately. Active work owns eventual key cleanup;
// idle cleanup runs outside all coordinator locks. Providers have finite bounds.
func (s *mcpSetupSession) close() {
	s.owner.close()
	s.mu.Lock()
	s.closed = true
	if s.connection != nil {
		s.connection.cancel()
	}
	if s.cancel != nil {
		s.cancel()
	}
	running := s.running
	client := s.client
	s.mu.Unlock()
	// Never hold the coordinator or adapter mutex while erasing session keys.
	// Active work owns cleanup on exit; idle/completed owners clean up here.
	if !running {
		s.session.Close()
		if client != nil {
			_ = client.client.Close()
		}
	}
}
func (s *mcpSetupSession) run(ctx context.Context, io mcpSetupIO, prepare func(context.Context) error) (err error) {
	if ctx == nil || io == nil {
		s.close()
		return ErrInvalid
	}
	s.mu.Lock()
	if s.started || s.closed || (s.connection != nil && io != s.connection.stream) {
		s.mu.Unlock()
		s.close()
		return ErrInvalid
	}
	s.started = true
	s.running = true
	work, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		if recover() != nil {
			err = ErrInvalid
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
			s.mu.Lock()
			client := s.client
			s.mu.Unlock()
			if client != nil {
				_ = client.client.Close()
			}
		}
	}()
	now, e := s.session.LocalNow()
	if e != nil || now >= s.owner.setupEnd {
		return ErrInvalid
	}
	work, deadlineCancel := context.WithTimeout(work, s.owner.setupEnd-now)
	defer deadlineCancel()
	if !s.session.Initiator() {
		if prepare == nil {
			return ErrInvalid
		}
		return s.serve(work, io, prepare)
	}
	return s.initiate(work, io)
}
func (s *mcpSetupSession) send(ctx context.Context, io mcpSetupIO, p *mcpOutput, wire []byte) error {
	if len(wire) == 0 || len(wire) > 32768 || ctx.Err() != nil {
		return ErrInvalid
	}
	// No owner mutex is held across crypto, registry, transport or preparation.
	if err := io.Send(ctx, append([]byte(nil), wire...)); err != nil {
		return ErrInvalid
	}
	observation, err := s.session.Observe(ctx)
	if ctx.Err() != nil || err != nil {
		return ErrInvalid
	}
	return s.owner.completeObserved(p, true, true, &observation, ctx)
}
func (s *mcpSetupSession) receive(ctx context.Context, io mcpSetupIO) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ErrInvalid
	}
	b, err := io.Receive(ctx)
	if err != nil || len(b) == 0 || len(b) > 32768 || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	return append([]byte(nil), b...), nil
}
func (s *mcpSetupSession) initiate(ctx context.Context, io mcpSetupIO) error {
	events := []mcpEvent{mcpSendInitialize, mcpSendInitialized, mcpSendList}
	for index, event := range events {
		id := ""
		var raw []byte
		switch index {
		case 0:
			id = guuid.NewString()
			raw = encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": "initialize", "params": map[string]any{"protocolVersion": MCPVersion, "capabilities": map[string]any{}, "clientInfo": s.info}})
		case 1:
			raw = []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
		case 2:
			id = guuid.NewString()
			raw = encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/list"})
		}
		if len(raw) > 16348 {
			return ErrInvalid
		}
		p, err := s.owner.transition(event, id, 0, true)
		if err != nil {
			return ErrInvalid
		}
		wire, err := s.session.SealRequest(ctx, raw, 30)
		if err != nil || (id != "" && wireField(wire, "id") == id) {
			return ErrInvalid
		}
		// The core retains the exact signed request for hash/ID correlation.
		outer := wireField(wire, "id")
		if err = s.send(ctx, io, p, wire); err != nil {
			return err
		}
		reply, err := s.receive(ctx, io)
		if err != nil {
			return err
		}
		r, err := s.session.OpenResponse(ctx, reply)
		if err != nil || r.MessageID != outer || !r.Success || r.Error != "" || (id != "" && wireField(reply, "id") == id) {
			return ErrInvalid
		}
		switch index {
		case 0:
			err = validateMCPInitialize(r.Data, id, true)
		case 1:
			err = validateMCPInitialized(r.Data, true)
		case 2:
			err = validateMCPDiscovery(r.Data, id, true)
		}
		if err != nil {
			return ErrInvalid
		}
		observation, err := s.session.Observe(ctx)
		if ctx.Err() != nil || err != nil {
			return ErrInvalid
		}
		if _, err = s.owner.transitionObserved(event+1, "", 0, true, &observation, ctx); err != nil {
			return ErrInvalid
		}
	}
	return nil
}
func (s *mcpSetupSession) serve(ctx context.Context, io mcpSetupIO, prepare func(context.Context) error) error {
	for index, event := range []mcpEvent{mcpReplyInitialize, mcpReplyInitialized, mcpReplyList} {
		wire, err := s.receive(ctx, io)
		if err != nil {
			return err
		}
		raw, err := s.session.OpenRequest(ctx, wire)
		if err != nil {
			return ErrInvalid
		}
		m, err := mcpSetupObject(raw)
		if err != nil {
			return ErrInvalid
		}
		id := rpcText(m, "id")
		if id != "" && id == wireField(wire, "id") {
			return ErrInvalid
		}
		if index == 1 && uuid.MatchString(id) {
			// Even a forbidden notification ID consumes authenticated lifetime history.
			s.owner.mu.Lock()
			s.owner.reserveLocked(id)
			s.owner.mu.Unlock()
		}
		// Reserve a valid authenticated request ID before setup message routing. Any
		// subsequent shape failure closes the pending output without releasing history.
		p, err := s.owner.transition(event, id, 0, true)
		if err != nil {
			return ErrInvalid
		}
		var response []byte
		switch index {
		case 0:
			err = validateMCPInitialize(raw, id, false)
			response = encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"protocolVersion": MCPVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": s.info}})
		case 1:
			err = validateMCPInitialized(raw, false)
			if err == nil {
				err = prepare(ctx)
			}
			response = []byte("{}")
		case 2:
			err = validateMCPDiscovery(raw, id, false)
			response = encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []json.RawMessage{mcpTool}}})
		}
		if err != nil || len(response) > 16348 || ctx.Err() != nil {
			return ErrInvalid
		}
		reply, err := s.session.SealResponse(ctx, wireField(wire, "id"), response, true, "", 30)
		if err != nil || (id != "" && wireField(reply, "id") == id) {
			return ErrInvalid
		}
		if err = s.send(ctx, io, p, reply); err != nil {
			return err
		}
	}
	return nil
}
