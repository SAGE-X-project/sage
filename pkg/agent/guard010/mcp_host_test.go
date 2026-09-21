package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	guuid "github.com/google/uuid"
)

func hostFixture(t *testing.T) (*admissionFixture, *mcpHost) {
	t.Helper()
	var host *mcpHost
	f := newAdmissionFixtureWithSetup(t, 2, func(f *admissionFixture) {
		p, e := newMCPClientPool(f.clock, 2, 20*time.Second)
		if e != nil {
			t.Fatal(e)
		}
		host, e = newMCPHost(f.gate, p, 4, 1, time.Millisecond)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if e := host.stop(ctx); e != nil {
				t.Error("host did not drain", e)
			}
		})
		if e = host.register(f.client); e != nil {
			t.Fatal(e)
		}
		if e = host.register(f.server); e != nil {
			t.Fatal(e)
		}
	})
	return f, host
}
func awaitHost(t *testing.T, condition func() bool) {
	t.Helper()
	end := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(end) {
			t.Fatal("host condition timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
func ownerClosed(s *mcpSetupSession) bool {
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	return s.owner.phase == mcpClosed
}
func TestMCPHostRuntimeExchange(t *testing.T) {
	f, h := hostFixture(t)
	b, e := openMCPOwnedClient(context.Background(), f.client, h.clients, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	x, y := net.Pipe()
	defer x.Close()
	defer y.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		io := &setupPipe{Conn: y}
		wire, e := io.Receive(ctx)
		if e == nil {
			_, e = f.server.admitProtected(ctx, wire, f.gate)
		}
		if e == nil {
			_, m, _, _ := intentEnvelope(f.intent)
			for {
				entry, ok, lookupErr := f.gate.ledger.store.Lookup(str(m, "issuer"), str(m, "call_id"))
				if lookupErr != nil {
					e = lookupErr
					break
				}
				if ok && entry.State == "COMPLETED" {
					break
				}
				select {
				case <-ctx.Done():
					e = ctx.Err()
				case <-time.After(time.Millisecond):
				}
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			e = f.server.replyProtected(ctx, io)
		}
		done <- e
	}()
	d, e := b.exchange(ctx, &setupPipe{Conn: x})
	if e != nil {
		t.Fatal(e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	if !d.FirstTerminal() || string(d.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("scheduler delivery")
	}
}
func TestMCPHostExpiresBlockedAuthentication(t *testing.T) {
	f, h := hostFixture(t)
	wire := f.wire(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	first := true
	f.clock.readHook = func(string) {
		if first {
			first = false
			close(entered)
			<-release
		}
	}
	go func() { _, e := f.server.admitProtected(context.Background(), wire, f.gate); done <- e }()
	<-entered
	f.clock.mono.Add(20000)
	awaitHost(t, func() bool { return ownerClosed(f.server) })
	h.gate.mu.Lock()
	occupied := h.gate.slots[0] != nil
	h.gate.mu.Unlock()
	if !occupied {
		t.Fatal("blocked authentication freed capacity")
	}
	close(release)
	if e := <-done; e == nil || f.executor.effects.Load() != 0 {
		t.Fatal("expired authentication admitted")
	}
}
func TestMCPHostExpiresBlockedClient(t *testing.T) {
	f, h := hostFixture(t)
	b, e := openMCPOwnedClient(context.Background(), f.client, h.clients, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, e := b.exchange(context.Background(), replyIO{send: func(context.Context, []byte) error { close(entered); <-release; return nil }})
		done <- e
	}()
	<-entered
	f.clock.mono.Add(20000)
	awaitHost(t, func() bool { return ownerClosed(f.client) })
	h.clients.mu.Lock()
	occupied := h.clients.slots[0] != nil
	h.clients.mu.Unlock()
	if !occupied {
		t.Fatal("stalled client freed capacity")
	}
	close(release)
	if e := <-done; e == nil {
		t.Fatal("expired client published")
	}
}
func TestMCPHostQueuedCancellationAndBoundedStop(t *testing.T) {
	f, h := hostFixture(t)
	f.executor.entered = make(chan struct{})
	f.executor.release = make(chan struct{})
	if _, e := f.admit(t); e != nil {
		t.Fatal(e)
	}
	<-f.executor.entered
	if e := f.server.replyProtected(context.Background(), replyIO{}); e != nil {
		t.Fatal(e)
	}
	var env map[string]any
	if e := json.Unmarshal(f.intent, &env); e != nil {
		t.Fatal(e)
	}
	intent := env["intent"].(map[string]any)
	intent["call_id"] = guuid.NewString()
	intent["nonce"] = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 16))
	canonical, _ := Canonicalize(encode(intent))
	env["proof"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)), append([]byte("sage-execution-intent|0.10.0\x00"), canonical...)))
	f.intent, _ = Canonicalize(encode(env))
	if _, e := f.admit(t); e != nil {
		t.Fatal(e)
	}
	f.clock.mono.Add(10000)
	awaitHost(t, func() bool {
		h.gate.mu.Lock()
		defer h.gate.mu.Unlock()
		for _, w := range h.gate.slots {
			if w != nil && w.queued && !w.claimed && w.cancelled {
				return true
			}
		}
		return false
	})
	if f.executor.effects.Load() != 1 {
		t.Fatal("expired queued work executed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if e := h.stop(ctx); e == nil {
		t.Fatal("stop claimed stalled worker had terminated")
	}
	h.gate.mu.Lock()
	occupied := 0
	for _, w := range h.gate.slots {
		if w != nil {
			occupied++
		}
	}
	h.gate.mu.Unlock()
	if occupied != 2 {
		t.Fatal("shutdown abandoned work")
	}
	close(f.executor.release)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	if e := h.stop(ctx2); e != nil {
		t.Fatal(e)
	}
	if f.executor.effects.Load() != 1 || f.state(t) != "UNKNOWN" {
		t.Fatal("canceled queue executed or fabricated outcome")
	}
}
func TestMCPHostSetupDeadlineAndClockFailure(t *testing.T) {
	for _, kind := range []string{"setup-expiry", "clock-failure"} {
		t.Run(kind, func(t *testing.T) {
			f, h := hostFixture(t)
			a, _, _ := setupSessions(t)
			s, e := newMCPSetupSession(a, "idle-setup", "1")
			if e != nil {
				t.Fatal(e)
			}
			defer s.close()
			if e = h.register(s); e != nil {
				t.Fatal(e)
			}
			if kind == "setup-expiry" {
				// The registered owner's own trusted clock reaches its original setup bound.
				s.owner.mu.Lock()
				clock := s.owner.clock
				s.owner.clock = func() (time.Duration, error) { n, e := clock(); return n + 30*time.Second, e }
				s.owner.mu.Unlock()
			} else {
				f.clock.panicClock.Store(true)
			}
			awaitHost(t, func() bool { return ownerClosed(s) })
			if kind == "clock-failure" {
				awaitHost(t, func() bool { return ownerClosed(f.server) && ownerClosed(f.client) })
			}
		})
	}
}

func TestMCPHostIdleResponseExpiryPreservesCompletion(t *testing.T) {
	f, _ := hostFixture(t)
	if _, e := f.admit(t); e != nil {
		t.Fatal(e)
	}
	awaitHost(t, func() bool { return f.state(t) == "COMPLETED" })
	f.clock.mono.Add(20000)
	awaitHost(t, func() bool { return ownerClosed(f.server) })
	awaitHost(t, func() bool { _, e := f.server.session.LocalNow(); return e != nil })
	if f.state(t) != "COMPLETED" || f.executor.effects.Load() != 1 {
		t.Fatal("response expiry changed execution")
	}
}
func TestMCPHostCleanupStallDoesNotBlockDeadlinesOrReleaseOwners(t *testing.T) {
	f, h := hostFixture(t)
	a, _, _ := setupSessions(t)
	b, _, bClock := setupSessions(t)
	c, _, _ := setupSessions(t)
	first, e := newMCPSetupSession(a, "cleanup-stall", "1")
	if e != nil {
		t.Fatal(e)
	}
	second, e := newMCPSetupSession(b, "expiring-setup", "1")
	if e != nil {
		t.Fatal(e)
	}
	third, e := newMCPSetupSession(c, "replacement", "1")
	if e != nil {
		t.Fatal(e)
	}
	defer first.close()
	defer second.close()
	defer third.close()
	if e = h.register(first); e != nil {
		t.Fatal(e)
	}
	if e = h.register(second); e != nil {
		t.Fatal(e)
	}
	client, e := openMCPOwnedClient(context.Background(), first, h.clients, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	client.client.mu.Lock()
	first.retireOnly()
	h.cleanupSoon()
	awaitHost(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		for _, entry := range h.owners {
			if entry != nil && entry.session == first && entry.cleaning {
				return true
			}
		}
		return false
	})
	bClock.mono.Add(30000)
	awaitHost(t, func() bool { return ownerClosed(second) })
	if e = h.register(third); e == nil {
		client.client.mu.Unlock()
		t.Fatal("blocked cleanup freed owner quota")
	}
	client.client.mu.Unlock()
	awaitHost(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		count := 0
		for _, entry := range h.owners {
			if entry != nil {
				count++
			}
		}
		return count == 2
	})
	if e = h.register(third); e != nil {
		t.Fatal("terminated cleanup retained owner quota", e)
	}
}

func TestMCPHostRejectsUnmanagedBindings(t *testing.T) {
	f, h := hostFixture(t)
	if _, e := newMCPHost(f.gate, h.clients, 4, 1, time.Millisecond); e == nil {
		t.Fatal("duplicate scheduler accepted")
	}
	if e := h.register(f.server); e == nil {
		t.Fatal("setup owner registered twice")
	}
	other, e := newMCPClientPool(f.clock, 1, 20*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = openMCPOwnedClient(context.Background(), f.client, other, filepath.Join(t.TempDir(), "wrong-pool"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f}); e == nil {
		t.Fatal("registered owner escaped its host pool")
	}
}

func TestMCPHostRejectsInitiatorWithServerCoordinator(t *testing.T) {
	f, _ := hostFixture(t)
	initiator, _, _ := setupSessions(t)
	if s, e := newMCPGuardSetup(initiator, "wrong-role", "1", f.gate); e == nil || s != nil {
		t.Fatal("initiator acquired server coordinator")
	}
}

func TestMCPHostRetainsOwnerDuringUnpublishedPreparationCleanup(t *testing.T) {
	f, h := hostFixture(t)
	a, _, _ := setupSessions(t)
	b, _, _ := setupSessions(t)
	c, _, _ := setupSessions(t)
	first, e := newMCPSetupSession(a, "failed-preparation", "1")
	if e != nil {
		t.Fatal(e)
	}
	second, e := newMCPSetupSession(b, "occupied-owner", "1")
	if e != nil {
		t.Fatal(e)
	}
	replacement, e := newMCPSetupSession(c, "replacement", "1")
	if e != nil {
		t.Fatal(e)
	}
	defer first.close()
	defer second.close()
	defer replacement.close()
	if e = h.register(first); e != nil {
		t.Fatal(e)
	}
	if e = h.register(second); e != nil {
		t.Fatal(e)
	}
	client, e := openMCPOwnedClient(context.Background(), first, h.clients, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	// Offline lifecycle schedule: OpenClient has returned a real journal, but
	// final preparation failed before attaching it. Exercise the actual cleanup
	// helper with that unpublished resource and its retained preparation slot.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	preparation := &mcpClientExchange{session: first, ctx: ctx, cancel: cancel, deadline: time.Hour}
	h.clients.mu.Lock()
	h.clients.slots[0] = preparation
	h.clients.mu.Unlock()
	first.mu.Lock()
	first.client = nil
	first.running = true
	first.closed = true
	first.mu.Unlock()
	client.client.mu.Lock()
	done := make(chan struct{})
	go func() {
		_, _ = finishMCPClientPreparation(first, h.clients, preparation, client, ErrInvalid)
		close(done)
	}()
	awaitHost(t, func() bool { return ownerClosed(first) })
	h.sweep()
	if e = h.register(replacement); e == nil {
		client.client.mu.Unlock()
		t.Fatal("unpublished cleanup released owner capacity")
	}
	first.mu.Lock()
	running := first.running
	first.mu.Unlock()
	if !running {
		client.client.mu.Unlock()
		t.Fatal("blocked cleanup lost ownership")
	}
	client.client.mu.Unlock()
	<-done
	awaitHost(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		for _, entry := range h.owners {
			if entry != nil && entry.session == first {
				return false
			}
		}
		return true
	})
	if e = h.register(replacement); e != nil {
		t.Fatal(e)
	}
}
