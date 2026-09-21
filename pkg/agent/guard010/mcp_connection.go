package guard010

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

// Configuration is pinned by the trusted host before reading any peer bytes.
// Endpoints remain host-owned; one connection never closes a shared endpoint.
type mcpConnectionConfig struct {
	endpoint                *hpke.CompletionEndpoint010
	initiator               bool
	recipient, recipientKey string
	name, version           string
	ttl                     int64
	timeout                 time.Duration
}

type mcpHostConnection struct {
	stream *mcpStream
	cancel context.CancelFunc
}

// connection executes synchronously, reserving a connection slot before any
// authentication or provider work. The caller transfers exclusive socket ownership
// even on rejection. Fixed listener/dial workers must call this method directly;
// it does not create an accept loop or a goroutine per connection. Connection and
// registered-owner quotas are separate, each bounded by the configured owner count.
// The slot is retained through the entire trusted handler and actual cleanup.
func (h *mcpHost) connection(ctx context.Context, conn net.Conn, cfg mcpConnectionConfig, prepare func(context.Context) error, handle func(context.Context, *mcpSetupSession, mcpSetupIO) error) (err error) {
	if conn == nil {
		return ErrInvalid
	}
	stream, e := newMCPStream(conn, cfg.timeout)
	if e != nil {
		_ = conn.Close()
		return ErrInvalid
	}
	defer func() { _ = stream.Close() }()
	if ctx == nil || cfg.endpoint == nil || cfg.ttl < 1 || cfg.ttl > 300 || handle == nil || (!cfg.initiator && prepare == nil) {
		return ErrInvalid
	}
	life, cancel := context.WithCancel(ctx)
	entry := &mcpHostConnection{stream: stream, cancel: cancel}
	h.mu.Lock()
	slot := -1
	select {
	case <-h.stopping:
	default:
		for n, c := range h.connections {
			if c == nil {
				slot = n
				break
			}
		}
	}
	if slot >= 0 {
		h.connections[slot] = entry
	}
	h.mu.Unlock()
	if slot < 0 {
		cancel()
		return ErrInvalid
	}
	// Only one cancellation callback per admitted connection. Close must make finite
	// progress. It never runs on the deadline sweeper or under a coordinator lock.
	stop := context.AfterFunc(life, func() { _ = stream.Close() })
	var s *mcpSetupSession
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
		cancel()
		stop()
		_ = stream.Close()
		if s != nil {
			s.close()
		}
		h.mu.Lock()
		h.connections[slot] = nil
		h.mu.Unlock()
		h.cleanupSoon()
	}()
	setup, endSetup := context.WithTimeout(life, cfg.timeout)
	defer endSetup()
	stopSetup := context.AfterFunc(setup, cancel)
	defer stopSetup()
	authenticated, e := establishMCPConnection(setup, stream, cfg)
	if e != nil {
		return ErrInvalid
	}
	// The authenticated completion is consumed on successful conversion; Close is
	// still required for conversion failures, and stale aliases cannot erase owners.
	defer authenticated.Close()
	if cfg.initiator {
		s, e = newMCPSetupSession(authenticated, cfg.name, cfg.version)
	} else {
		s, e = newMCPGuardSetup(authenticated, cfg.name, cfg.version, h.gate)
	}
	if e != nil {
		return ErrInvalid
	}
	s.connection = entry
	if setup.Err() != nil || h.register(s) != nil {
		return ErrInvalid
	}
	if e = s.run(setup, stream, prepare); e != nil {
		return ErrInvalid
	}
	if !stopSetup() || setup.Err() != nil {
		return ErrInvalid
	}
	endSetup()
	if life.Err() != nil {
		return ErrInvalid
	}
	err = handle(life, s, stream)
	if life.Err() != nil {
		return ErrInvalid
	}
	return err
}

// establishMCPConnection retains pending/provisional secrets until transfer or
// cleanup. A partial response send cannot expose an authenticated owner to setup.
func establishMCPConnection(ctx context.Context, stream *mcpStream, cfg mcpConnectionConfig) (authenticated *hpke.AuthenticatedCompletion010, err error) {
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
		if err != nil && authenticated != nil {
			authenticated.Close()
			authenticated = nil
		}
	}()
	if cfg.initiator {
		pending, request, e := cfg.endpoint.Start(ctx, cfg.recipient, cfg.recipientKey, cfg.ttl)
		if e != nil {
			return nil, ErrInvalid
		}
		defer pending.Close()
		if e = stream.Send(ctx, request); e != nil {
			return nil, ErrInvalid
		}
		response, e := stream.Receive(ctx)
		if e != nil {
			return nil, ErrInvalid
		}
		authenticated, err = pending.Complete(ctx, response)
	} else {
		request, e := stream.Receive(ctx)
		if e != nil {
			return nil, ErrInvalid
		}
		var response []byte
		authenticated, response, err = cfg.endpoint.Respond(ctx, request, cfg.ttl)
		if err == nil {
			err = stream.Send(ctx, response)
		}
	}
	if err == nil && ctx.Err() != nil {
		err = ErrInvalid
	}
	return authenticated, err
}

func (h *mcpHost) cancelConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.listenerCancel != nil {
		h.listenerCancel()
	}
	for _, c := range h.connections {
		if c != nil {
			c.cancel()
		}
	}
}

// serve owns one listener and a fixed number of accept/connection workers. No
// goroutine is created for an accepted socket. Closing the listener must unblock
// Accept within a finite bound. Stop waits for these workers and socket cleanup.
func (h *mcpHost) serve(ctx context.Context, listener net.Listener, workers int, cfg mcpConnectionConfig, prepare func(context.Context) error, handle func(context.Context, *mcpSetupSession, mcpSetupIO) error) error {
	if listener == nil {
		return ErrInvalid
	}
	var closeOnce sync.Once
	closeListener := func() { closeOnce.Do(func() { _ = listener.Close() }) }
	defer closeListener()
	if ctx == nil || workers < 1 || workers > len(h.connections) || cfg.initiator {
		return ErrInvalid
	}
	life, cancel := context.WithCancel(ctx)
	defer cancel()
	h.mu.Lock()
	select {
	case <-h.stopping:
		h.mu.Unlock()
		return ErrInvalid
	default:
	}
	if h.listenerCancel != nil {
		h.mu.Unlock()
		return ErrInvalid
	}
	h.listenerCancel = cancel
	h.mu.Unlock()
	stop := context.AfterFunc(life, closeListener)
	var wg sync.WaitGroup
	var failed atomic.Bool
	defer func() {
		cancel()
		stop()
		closeListener()
		wg.Wait()
		h.mu.Lock()
		h.listenerCancel = nil
		h.mu.Unlock()
	}()
	for range workers {
		wg.Go(func() {
			defer func() {
				if recover() != nil {
					failed.Store(true)
				}
				cancel()
			}()
			for life.Err() == nil {
				conn, e := listener.Accept()
				if e != nil {
					return
				}
				_ = h.connection(life, conn, cfg, prepare, handle)
			}
		})
	}
	<-life.Done()
	wg.Wait()
	if failed.Load() {
		return ErrInvalid
	}
	return life.Err()
}
