package guard010

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const setupAlice = "did:sage:web:agent.example:alice"
const setupBob = "did:sage:web:agent.example:bob"

type setupRegistry struct {
	mono       atomic.Int64
	revoked    atomic.Bool
	panicClock atomic.Bool
	readHook   func(string)
	bobSigning atomic.Bool
	bobKEM     atomic.Bool
}

func (c *setupRegistry) Now() (registry010.Stamp, error) {
	if c.panicClock.Load() {
		panic("unavailable clock")
	}
	mono := c.mono.Load()
	return registry010.Stamp{MonoMS: mono, Unix: 100 + mono/1000}, nil
}
func (c *setupRegistry) Read(_ context.Context, did string) (registry010.Snapshot, error) {
	if c.readHook != nil {
		c.readHook(did)
	}
	seed := byte(1)
	if did == setupBob {
		seed = 2
	}
	keys := []registry010.Key{{Name: "signing-1", Alg: "ed25519", Material: hex.EncodeToString(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32)).Public().(ed25519.PublicKey)), State: "accepted"}}
	if did == setupBob {
		sk, _ := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{3}, 32))
		keys = append(keys, registry010.Key{Name: "kem-1", Alg: "x25519", Material: hex.EncodeToString(sk.PublicKey().Bytes()), State: "accepted"})
	}
	version := "1"
	if c.revoked.Load() {
		keys[0].State = "revoked"
		version = "2"
	}
	if did == setupBob {
		if c.bobSigning.Load() {
			keys[0].State = "revoked"
			version = "2"
		}
		if c.bobKEM.Load() {
			keys[1].State = "revoked"
			version = "2"
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name < keys[j].Name })
	raw, _ := Canonicalize(encode(keys))
	h := sha256.Sum256(raw)
	return registry010.Snapshot{Source: "setup-test", Registry: "web:agent.example", Network: "local", DID: did, Version: version, State: "active", Digest: hex.EncodeToString(h[:]), Ready: true, Validated: true, Finalized: true, AcquiredMS: c.mono.Load(), Keys: keys}, nil
}
func setupSessions(t *testing.T) (*hpke.AuthenticatedCompletion010, *hpke.AuthenticatedCompletion010, *setupRegistry) {
	t.Helper()
	c := &setupRegistry{}
	endpoint := func(did string, n byte) *hpke.CompletionEndpoint010 {
		j, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = j.Close() })
		g, err := registry010.NewGate(registry010.Config{Source: "setup-test", Registry: "web:agent.example", Network: "local"}, c, c, j)
		if err != nil {
			t.Fatal(err)
		}
		r, err := hpke.OpenReplayJournal010(filepath.Join(t.TempDir(), "replay"), true, c)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = r.Close() })
		var kem []byte
		if n == 2 {
			kem = bytes.Repeat([]byte{3}, 32)
		}
		e, err := hpke.NewCompletionEndpoint010(did, did+"#signing-1", bytes.Repeat([]byte{n}, 32), kem, g, c, r)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(e.Close)
		return e
	}
	a, b := endpoint(setupAlice, 1), endpoint(setupBob, 2)
	// Honor the new durable replay journal's startup quarantine before handshake.
	c.mono.Store(360000)
	pending, request, err := a.Start(context.Background(), setupBob, setupBob+"#signing-1", 300)
	if err != nil {
		t.Fatal("handshake start", err)
	}
	responder, response, err := b.Respond(context.Background(), request, 300)
	if err != nil {
		t.Fatal("handshake respond", err)
	}
	initiator, err := pending.Complete(context.Background(), response)
	if err != nil {
		t.Fatal("handshake complete", err)
	}
	t.Cleanup(initiator.Close)
	t.Cleanup(responder.Close)
	return initiator, responder, c
}
func setupAdapters(t *testing.T) (*mcpSetupSession, *mcpSetupSession, *setupRegistry) {
	t.Helper()
	a, b, c := setupSessions(t)
	left, err := newMCPSetupSession(a, "client", "1")
	if err != nil {
		t.Fatal(err)
	}
	right, err := newMCPSetupSession(b, "server", "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { left.close(); right.close() })
	return left, right, c
}

// This test-only length-prefixed net.Pipe transport is a bounded framing fixture,
// not a new wire profile. Production framing remains a selected host dependency.
type setupPipe struct {
	net.Conn
	sent int
	hook func(int, []byte) error
}

func (p *setupPipe) Send(ctx context.Context, b []byte) error {
	if len(b) > 32768 {
		return ErrInvalid
	}
	p.sent++
	if p.hook != nil {
		if err := p.hook(p.sent, b); err != nil {
			return err
		}
	}
	end, _ := ctx.Deadline()
	_ = p.SetWriteDeadline(end)
	stop := context.AfterFunc(ctx, func() { _ = p.Close() })
	defer stop()
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(b)))
	if _, err := io.Copy(p.Conn, bytes.NewReader(append(header[:], b...))); err != nil {
		return err
	}
	return nil
}
func (p *setupPipe) Receive(ctx context.Context) ([]byte, error) {
	end, _ := ctx.Deadline()
	_ = p.SetReadDeadline(end)
	stop := context.AfterFunc(ctx, func() { _ = p.Close() })
	defer stop()
	var header [4]byte
	if _, err := io.ReadFull(p.Conn, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > 32768 {
		return nil, ErrInvalid
	}
	b := make([]byte, int(size))
	_, err := io.ReadFull(p.Conn, b)
	return b, err
}
func runSetupPair(t *testing.T, a, b *mcpSetupSession, left, right *setupPipe, prepare func(context.Context) error) (error, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer left.Close()
	defer right.Close()
	result := make(chan error, 1)
	go func() {
		err := b.run(ctx, right, prepare)
		if err != nil {
			cancel()
		}
		result <- err
	}()
	err := a.run(ctx, left, nil)
	if err != nil {
		cancel()
	}
	return err, <-result
}
func TestMCPAuthenticatedSetupRuntime(t *testing.T) {
	a, b, _ := setupAdapters(t)
	x, y := net.Pipe()
	left, right := &setupPipe{Conn: x}, &setupPipe{Conn: y}
	prepared := 0
	errA, errB := runSetupPair(t, a, b, left, right, func(context.Context) error { prepared++; return nil })
	if errA != nil || errB != nil {
		t.Fatal(errA, errB)
	}
	if a.owner.phase != mcpReady || b.owner.phase != mcpReady || prepared != 1 || left.sent != 3 || right.sent != 3 || len(a.owner.seen) != 2 || len(b.owner.seen) != 2 {
		t.Fatal("setup boundaries")
	}
}
func TestMCPAuthenticatedSetupFailureSchedules(t *testing.T) {
	for _, kind := range []string{"send-failure", "prepare-failure", "revoke-before-ack", "deadline-before-ready", "close-before-ready"} {
		t.Run(kind, func(t *testing.T) {
			a, b, c := setupAdapters(t)
			x, y := net.Pipe()
			left, right := &setupPipe{Conn: x}, &setupPipe{Conn: y}
			if kind == "send-failure" {
				left.hook = func(int, []byte) error { return errors.New("uncertain send") }
			}
			if kind == "deadline-before-ready" || kind == "close-before-ready" {
				right.hook = func(n int, _ []byte) error {
					if n == 3 {
						if kind == "deadline-before-ready" {
							c.mono.Add(30000)
						} else {
							b.close()
						}
					}
					return nil
				}
			}
			prepare := func(context.Context) error {
				if kind == "prepare-failure" {
					return ErrInvalid
				}
				if kind == "revoke-before-ack" {
					c.revoked.Store(true)
				}
				return nil
			}
			ea, eb := runSetupPair(t, a, b, left, right, prepare)
			if ea == nil && eb == nil {
				t.Fatal("failure accepted")
			}
			if b.owner.phase == mcpReady || (kind != "close-before-ready" && a.owner.phase == mcpReady) {
				t.Fatal("failed setup advertised ready")
			}
		})
	}
}
func TestMCPAuthenticatedSetupCloseDuringSend(t *testing.T) {
	a, b, _ := setupAdapters(t)
	x, y := net.Pipe()
	left, right := &setupPipe{Conn: x}, &setupPipe{Conn: y}
	entered, release := make(chan struct{}), make(chan struct{})
	left.hook = func(int, []byte) error { close(entered); <-release; return nil }
	done := make(chan struct{})
	var ea, eb error
	go func() {
		ea, eb = runSetupPair(t, a, b, left, right, func(context.Context) error { return nil })
		close(done)
	}()
	<-entered
	closed := make(chan struct{})
	go func() { a.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("close waited for send")
	}
	close(release)
	<-done
	if ea == nil || eb == nil || a.owner.phase != mcpClosed {
		t.Fatal("late send reopened owner")
	}
}
func TestMCPAuthenticatedSetupKeepsProvisionalCreationTime(t *testing.T) {
	a, b, c := setupSessions(t)
	c.mono.Add(30000)
	for _, s := range []*hpke.AuthenticatedCompletion010{a, b} {
		if _, err := newMCPSetupSession(s, "test", "1"); err == nil {
			t.Fatal("deadline restarted")
		}
	}
}

// A scripted authenticated peer supplies malformed application plaintext only
// inside this unit scenario. No attack program or host-bypass path is provided.
func TestMCPAuthenticatedSetupRejectsFalseOuterSuccess(t *testing.T) {
	rawClient, rawServer, _ := setupSessions(t)
	a, err := newMCPSetupSession(rawClient, "client", "1")
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()

	x, y := net.Pipe()
	left, right := &setupPipe{Conn: x}, &setupPipe{Conn: y}
	defer left.Close()
	defer right.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wire, e := right.Receive(ctx)
		if e != nil {
			return
		}
		plain, e := rawServer.OpenRequest(ctx, wire)
		if e != nil {
			return
		}
		var m map[string]any
		_ = json.Unmarshal(setupInitialize(true), &m)
		m["id"] = wireField(plain, "id")
		reply, e := rawServer.SealResponse(ctx, wireField(wire, "id"), encode(m), false, "unavailable", 30)
		if e == nil {
			_ = right.Send(ctx, reply)
		}
	}()
	if err = a.run(ctx, left, nil); err == nil || a.owner.phase != mcpClosed {
		t.Fatal("false carrier accepted")
	}
	wg.Wait()
}

func TestMCPSetupCloseCleansIdleAndCompletedSessions(t *testing.T) {
	for _, completed := range []bool{false, true} {
		a, b, _ := setupAdapters(t)
		if completed {
			x, y := net.Pipe()
			ea, eb := runSetupPair(t, a, b, &setupPipe{Conn: x}, &setupPipe{Conn: y}, func(context.Context) error { return nil })
			if ea != nil || eb != nil {
				t.Fatal(ea, eb)
			}
		}
		a.close()
		b.close()
		if _, err := a.session.LocalNow(); err == nil {
			t.Fatal("client keys still live")
		}
		if _, err := b.session.LocalNow(); err == nil {
			t.Fatal("server keys still live")
		}
	}
}
func TestMCPSetupStartupClockPanicCleansUp(t *testing.T) {
	a, _, c := setupAdapters(t)
	c.panicClock.Store(true)
	x, y := net.Pipe()
	defer x.Close()
	defer y.Close()
	if err := a.run(context.Background(), &setupPipe{Conn: x}, nil); err == nil {
		t.Fatal("clock panic accepted")
	}
	c.panicClock.Store(false)
	done := make(chan struct{})
	go func() { a.close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("clock panic retained adapter lock")
	}
	if _, err := a.session.LocalNow(); err == nil {
		t.Fatal("failed startup retained keys")
	}
}
