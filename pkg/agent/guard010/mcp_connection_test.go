package guard010

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func connectionCount(h *mcpHost) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.connections {
		if c != nil {
			n++
		}
	}
	return n
}
func TestMCPHostConnectionRuntime(t *testing.T) {
	f, h := hostFixture(t)
	a, b, _ := setupEndpoints(t)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	serverDone := make(chan error, 1)
	serverConfig := mcpConnectionConfig{endpoint: b, name: "server", version: "1", ttl: 300, timeout: 3 * time.Second}
	go func() {
		serverDone <- h.serve(context.Background(), listener, 1, serverConfig, func(context.Context) error { return nil }, func(ctx context.Context, s *mcpSetupSession, transport mcpSetupIO) error {
			wire, e := transport.Receive(ctx)
			if e != nil {
				return e
			}
			if _, e = s.admitProtected(ctx, wire, h.gate); e != nil {
				return e
			}
			_, m, _, _ := intentEnvelope(f.intent)
			for {
				entry, ok, e := h.gate.ledger.store.Lookup(str(m, "issuer"), str(m, "call_id"))
				if e != nil {
					return e
				}
				if ok && entry.State == "COMPLETED" {
					break
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Millisecond):
				}
			}
			return s.replyProtected(ctx, transport)
		})
	}()
	conn, e := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	cfg := mcpConnectionConfig{endpoint: a, initiator: true, recipient: setupBob, recipientKey: setupBob + "#signing-1", name: "client", version: "1", ttl: 300, timeout: 3 * time.Second}
	path := filepath.Join(t.TempDir(), "client")
	var delivery *ClientDelivery
	e = h.connection(context.Background(), conn, cfg, nil, func(ctx context.Context, s *mcpSetupSession, transport mcpSetupIO) error {
		client, e := openMCPOwnedClient(ctx, s, h.clients, path, true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
		if e != nil {
			return e
		}
		delivery, e = client.exchange(ctx, transport)
		return e
	})
	if e != nil || delivery == nil || !delivery.FirstTerminal() || string(delivery.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("owned connection delivery", e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = h.stop(ctx); e != nil {
		t.Fatal("stop", e)
	}
	if e = <-serverDone; !errors.Is(e, context.Canceled) {
		t.Fatal("listener shutdown", e)
	}
	if connectionCount(h) != 0 {
		t.Fatal("connection slots leaked")
	}
}
func TestMCPHostConnectionQuotaAndShutdown(t *testing.T) {
	_, h := hostFixture(t)
	_, endpoint, _ := setupEndpoints(t)
	cfg := mcpConnectionConfig{endpoint: endpoint, name: "server", version: "1", ttl: 300, timeout: 3 * time.Second}
	done := make(chan error, len(h.connections))
	for range len(h.connections) {
		x, y := net.Pipe()
		defer y.Close()
		go func() {
			done <- h.connection(context.Background(), x, cfg, func(context.Context) error { return nil }, func(context.Context, *mcpSetupSession, mcpSetupIO) error {
				t.Error("silent peer reached handler")
				return nil
			})
		}()
	}
	awaitHost(t, func() bool { return connectionCount(h) == len(h.connections) })
	rejected := &streamFixture{}
	e := h.connection(context.Background(), rejected, cfg, func(context.Context) error { return nil }, func(context.Context, *mcpSetupSession, mcpSetupIO) error { return nil })
	if e == nil || rejected.closes.Load() != 1 || rejected.readBytes != 0 {
		t.Fatal("quota did not reject before input")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = h.stop(ctx); e != nil {
		t.Fatal(e)
	}
	for range len(h.connections) {
		if e = <-done; e == nil {
			t.Fatal("pending connection survived shutdown")
		}
	}
	if connectionCount(h) != 0 {
		t.Fatal("pending slots retained after cleanup")
	}
}
func TestMCPHostConnectionRetainsBlockedHandshake(t *testing.T) {
	_, h := hostFixture(t)
	endpoint, _, clock := setupEndpoints(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	clock.readHook = func(string) { once.Do(func() { close(entered); <-release }) }
	cfg := mcpConnectionConfig{endpoint: endpoint, initiator: true, recipient: setupBob, recipientKey: setupBob + "#signing-1", name: "client", version: "1", ttl: 300, timeout: 40 * time.Millisecond}
	x, y := net.Pipe()
	defer y.Close()
	done := make(chan error, 1)
	go func() {
		done <- h.connection(context.Background(), x, cfg, nil, func(context.Context, *mcpSetupSession, mcpSetupIO) error {
			t.Error("expired handshake reached handler")
			return nil
		})
	}()
	<-entered
	_ = y.SetReadDeadline(time.Now().Add(time.Second))
	if _, e := y.Read(make([]byte, 1)); e == nil {
		t.Fatal("deadline did not close socket")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if e := h.stop(ctx); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("stop released blocked provider", e)
	}
	cancel()
	if connectionCount(h) != 1 {
		t.Fatal("blocked provider freed connection capacity")
	}
	close(release)
	if e := <-done; e == nil {
		t.Fatal("expired handshake accepted")
	}
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e := h.stop(ctx); e != nil {
		t.Fatal(e)
	}
}

type delayedCloseConn struct {
	net.Conn
	entered, release chan struct{}
	once             sync.Once
	closed           atomic.Bool
}

func (c *delayedCloseConn) Close() error {
	c.once.Do(func() { close(c.entered); <-c.release; _ = c.Conn.Close(); c.closed.Store(true) })
	return nil
}
func TestMCPHostConnectionRetainsSocketCleanup(t *testing.T) {
	_, h := hostFixture(t)
	_, endpoint, _ := setupEndpoints(t)
	x, y := net.Pipe()
	defer y.Close()
	conn := &delayedCloseConn{Conn: x, entered: make(chan struct{}), release: make(chan struct{})}
	cfg := mcpConnectionConfig{endpoint: endpoint, name: "server", version: "1", ttl: 300, timeout: 40 * time.Millisecond}
	done := make(chan error, 1)
	go func() {
		done <- h.connection(context.Background(), conn, cfg, func(context.Context) error { return nil }, func(context.Context, *mcpSetupSession, mcpSetupIO) error { return nil })
	}()
	<-conn.entered
	if connectionCount(h) != 1 {
		t.Fatal("cleanup freed connection early")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if e := h.stop(ctx); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("shutdown hid cleanup", e)
	}
	cancel()
	close(conn.release)
	if e := <-done; e == nil || !conn.closed.Load() {
		t.Fatal("cleanup failed", e)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e := h.stop(ctx); e != nil {
		t.Fatal(e)
	}
}

type panicAcceptListener struct {
	net.Listener
	closed atomic.Bool
}

func (*panicAcceptListener) Accept() (net.Conn, error) { panic("accept provider") }
func (l *panicAcceptListener) Close() error            { l.closed.Store(true); return nil }
func TestMCPHostListenerPanic(t *testing.T) {
	_, h := hostFixture(t)
	l := &panicAcceptListener{}
	if e := h.serve(context.Background(), l, 1, mcpConnectionConfig{}, nil, nil); !errors.Is(e, ErrInvalid) {
		t.Fatal("listener panic", e)
	}
	if !l.closed.Load() {
		t.Fatal("listener not closed")
	}
	h.mu.Lock()
	retained := h.listenerCancel != nil
	h.mu.Unlock()
	if retained {
		t.Fatal("listener slot retained")
	}
}

func TestMCPConnectionRejectsTransportSubstitution(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	p, e := newMCPClientPool(f.clock, 1, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	client, e := openMCPOwnedClient(context.Background(), f.client, p, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	f.client.connection = &mcpHostConnection{stream: f.streams[0], cancel: func() {}}
	f.server.connection = &mcpHostConnection{stream: f.streams[1], cancel: func() {}}
	calls := 0
	substitute := replyIO{send: func(context.Context, []byte) error { calls++; return nil }}
	if d, e := client.exchange(context.Background(), substitute); e == nil || d != nil {
		t.Fatal("substituted client transport accepted")
	}
	if e := f.server.replyProtected(context.Background(), substitute); e == nil {
		t.Fatal("substituted response transport accepted")
	}
	if calls != 0 || f.executor.effects.Load() != 0 {
		t.Fatal("substitution had effects")
	}
}

type delayedListenerCleanup struct {
	net.Listener
	first            atomic.Bool
	entered, release chan struct{}
}

func (l *delayedListenerCleanup) Close() error {
	if l.first.CompareAndSwap(false, true) {
		_ = l.Listener.Close()
		close(l.entered)
		<-l.release
	}
	return nil
}
func TestMCPHostRetainsListenerCleanup(t *testing.T) {
	_, h := hostFixture(t)
	native, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	l := &delayedListenerCleanup{Listener: native, entered: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- h.serve(context.Background(), l, 1, mcpConnectionConfig{}, nil, nil) }()
	awaitHost(t, func() bool { h.mu.Lock(); defer h.mu.Unlock(); return h.listenerCancel != nil })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if e = h.stop(ctx); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("listener cleanup was not retained", e)
	}
	cancel()
	<-l.entered
	h.mu.Lock()
	retained := h.listenerCancel != nil
	h.mu.Unlock()
	if !retained {
		t.Fatal("listener released while Close still running")
	}
	close(l.release)
	if e = <-done; !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = h.stop(ctx); e != nil {
		t.Fatal(e)
	}
}

func TestMCPHostConnectionSetupFailure(t *testing.T) {
	_, h := hostFixture(t)
	a, b, _ := setupEndpoints(t)
	left := mcpConnectionConfig{endpoint: a, initiator: true, recipient: setupBob, recipientKey: setupBob + "#signing-1", name: "client", version: "1", ttl: 300, timeout: time.Second}
	right := mcpConnectionConfig{endpoint: b, name: "server", version: "1", ttl: 300, timeout: time.Second}
	x, y := net.Pipe()
	var calls atomic.Int64
	handler := func(context.Context, *mcpSetupSession, mcpSetupIO) error { calls.Add(1); return nil }
	done := make(chan error, 1)
	go func() {
		done <- h.connection(context.Background(), y, right, func(context.Context) error { return ErrInvalid }, handler)
	}()
	if e := h.connection(context.Background(), x, left, nil, handler); e == nil {
		t.Fatal("failed preparation established client")
	}
	if e := <-done; e == nil {
		t.Fatal("failed preparation established server")
	}
	if calls.Load() != 0 || connectionCount(h) != 0 {
		t.Fatal("setup failure published handler or leaked slot")
	}
}
