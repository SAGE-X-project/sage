package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	guuid "github.com/google/uuid"
)

type replyIO struct {
	send func(context.Context, []byte) error
}

func (i replyIO) Send(ctx context.Context, b []byte) error {
	if i.send != nil {
		return i.send(ctx, b)
	}
	return nil
}
func (i replyIO) Receive(context.Context) ([]byte, error) { return nil, ErrInvalid }

type replyClientSender struct {
	fixture *admissionFixture
	wire    []byte
}

func (s *replyClientSender) Commit(ctx context.Context, id string, intent []byte) error {
	f := s.fixture
	if err := f.client.owner.reserveProtected(id, true); err != nil {
		return err
	}
	raw, err := MCPRequest(MCPVersion, id, intent)
	if err != nil {
		return err
	}
	s.wire, err = f.client.session.SealRequest(ctx, raw, 30)
	return err
}

type replyClientClock struct{ fixture *admissionFixture }

func (c replyClientClock) Sample(context.Context) (int64, int64, error) {
	s, e := c.fixture.clock.Now()
	return s.Unix * 1000, s.MonoMS, e
}

func TestMCPProtectedReplyRuntimeConsumption(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	sender := &replyClientSender{fixture: f}
	services := ClientServices{IntentAuthority: f.config.authority, Policy: f.config.policy, ResultAuthority: f.config.signer, Clock: replyClientClock{f}, Sender: sender}
	path := filepath.Join(t.TempDir(), "client")
	c, err := OpenClient(context.Background(), path, true, f.intent, services)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	ticket, err := c.Begin(context.Background(), guuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.server.admitProtected(context.Background(), sender.wire, f.gate); err != nil {
		t.Fatal(err)
	}
	if _, err = f.gate.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	x, y := net.Pipe()
	defer x.Close()
	defer y.Close()
	done := make(chan error, 1)
	go func() { done <- f.server.replyProtected(context.Background(), &setupPipe{Conn: x}) }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wire, err := (&setupPipe{Conn: y}).Receive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	response, err := f.client.session.OpenResponse(ctx, wire)
	if err != nil || !response.Success || response.Error != "" || response.MessageID != wireField(sender.wire, "id") {
		t.Fatal("outer correlation", err)
	}
	success, code, err := sessionResult(MCPVersion, ticket.ID(), response.Data, f.intent, setupBob, setupAlice)
	if err != nil || !success || code != "" {
		t.Fatal("inner correlation", err)
	}
	d, err := c.AcceptMCPResponse(ctx, ticket, MCPVersion, response.Data)
	if err != nil || !d.FirstTerminal() || string(d.Output()) != `{"ok":true}` {
		t.Fatal("durable delivery", err)
	}
	if _, err = c.AcceptMCPResponse(ctx, ticket, MCPVersion, response.Data); err == nil {
		t.Fatal("duplicate delivery")
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenClient(ctx, path, false, f.intent, services)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	if _, err = reopened.Begin(ctx, guuid.NewString()); err == nil {
		t.Fatal("restart redelivered terminal")
	}
	if f.executor.effects.Load() != 1 || f.state(t) != "COMPLETED" {
		t.Fatal("execution outcome changed")
	}
}

func TestMCPProtectedReplyPendingAndTerminal(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	var wire []byte
	io := replyIO{send: func(_ context.Context, b []byte) error { wire = append([]byte(nil), b...); return nil }}
	if err := f.server.replyProtected(context.Background(), io); err != nil {
		t.Fatal(err)
	}
	pending, err := f.client.session.OpenResponse(context.Background(), wire)
	if err != nil || pending.Success || pending.Error != "unavailable" || f.executor.effects.Load() != 0 {
		t.Fatal("pending mapping", err)
	}
	if _, err = f.gate.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, m, _, _ := intentEnvelope(f.intent)
	entry, _, _ := f.gate.ledger.store.Lookup(str(m, "issuer"), str(m, "call_id"))
	expected, _ := hex.DecodeString(entry.ResultHex)
	if _, err = f.admit(t); err != nil {
		t.Fatal(err)
	}
	if err = f.server.replyProtected(context.Background(), io); err != nil {
		t.Fatal(err)
	}
	terminal, err := f.client.session.OpenResponse(context.Background(), wire)
	if err != nil || !terminal.Success {
		t.Fatal("terminal mapping", err)
	}
	envelope, err := parseMCPResponse(MCPVersion, wireField(terminal.Data, "id"), terminal.Data)
	if err != nil || !bytes.Equal(expected, envelope) || f.executor.effects.Load() != 1 {
		t.Fatal("stored terminal changed", err)
	}
	if err = f.server.replyProtected(context.Background(), io); err == nil {
		t.Fatal("response permit reused")
	}
}

func TestMCPProtectedReplyFailurePreservesCompletion(t *testing.T) {
	for _, kind := range []string{"partial-send", "close", "deadline", "generation", "revocation", "oversize", "deferred-overflow"} {
		t.Run(kind, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			if _, err := f.admit(t); err != nil {
				t.Fatal(err)
			}
			if kind == "oversize" {
				f.executor.output = []byte(`{"data":"` + strings.Repeat("x", 9000) + `"}`)
			}
			if _, err := f.gate.runOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			sends := 0
			err := f.server.replyProtected(context.Background(), replyIO{send: func(_ context.Context, _ []byte) error {
				sends++
				switch kind {
				case "partial-send":
					return ErrInvalid
				case "close":
					f.server.close()
				case "deadline":
					f.clock.mono.Add(20000)
				case "generation":
					return f.gate.replace(f.config)
				case "revocation":
					f.clock.bobKEM.Store(true)
				case "deferred-overflow":
					if e := f.server.owner.deferFrame([]byte("bounded wire"), true); e != nil {
						return e
					}
					return f.server.owner.deferFrame([]byte("second frame"), true)
				}
				return nil
			}})
			if err == nil || f.server.owner.phase != mcpClosed || f.state(t) != "COMPLETED" || f.executor.effects.Load() != 1 {
				t.Fatal("transport failure changed execution", err)
			}
			if _, err = f.server.session.LocalNow(); err == nil {
				t.Fatal("failed session remains live")
			}
			if kind == "oversize" && sends != 0 {
				t.Fatal("oversized output sent")
			}
			for _, p := range f.gate.outputs {
				if p != nil {
					t.Fatal("terminated output retained quota")
				}
			}
		})
	}
}

func TestMCPProtectedReplyCloseDoesNotReleaseBlockedOutput(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	if _, err := f.gate.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- f.server.replyProtected(context.Background(), replyIO{send: func(context.Context, []byte) error { close(entered); <-release; return nil }})
	}()
	<-entered
	f.server.close()
	f.gate.mu.Lock()
	occupied := f.gate.outputs[0] != nil
	f.gate.mu.Unlock()
	if !occupied {
		t.Fatal("blocked output abandoned its quota")
	}
	close(release)
	if err := <-done; err == nil {
		t.Fatal("late callback published")
	}
	if f.state(t) != "COMPLETED" || f.server.owner.pending != nil || f.server.owner.response != nil {
		t.Fatal("late completion changed state")
	}
}

type alternateReplySigner struct{ *RegistryAuthority }

func (s *alternateReplySigner) KeyID(context.Context) (string, error) {
	return setupBob + "#signing-2", nil
}
func (s *alternateReplySigner) Sign(_ context.Context, kid string, raw []byte) ([]byte, error) {
	if kid != setupBob+"#signing-2" {
		return nil, ErrInvalid
	}
	return ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{4}, 32)), raw), nil
}

func TestMCPProtectedReplyResultAuthorityBoundaries(t *testing.T) {
	for _, kind := range []string{"fresh", "stale", "key-expiry"} {
		t.Run(kind, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			if kind == "key-expiry" {
				authority, err := NewRegistryAuthority(f.config.authority.gate, setupBob, setupBob+"#signing-2")
				if err != nil {
					t.Fatal(err)
				}
				config := *f.config
				config.resultAuthority = authority
				config.signer = &alternateReplySigner{authority}
				if err = f.gate.replace(&config); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.admit(t); err != nil {
				t.Fatal(err)
			}
			if _, err := f.gate.runOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			var observedOutput *mcpOutput
			err := f.server.replyProtected(context.Background(), replyIO{send: func(context.Context, []byte) error {
				observedOutput = f.server.owner.pending
				bobReads := 0
				f.clock.readHook = func(did string) {
					if did != setupBob {
						return
					}
					bobReads++
					if kind == "key-expiry" && bobReads == 6 {
						f.clock.mono.Add(5000)
					}
					if kind != "key-expiry" && bobReads == 5 {
						if kind == "fresh" {
							f.clock.mono.Add(5000)
						} else {
							f.clock.mono.Add(5001)
						}
					}
				}
				return nil
			}})
			if kind == "fresh" {
				if err != nil {
					t.Fatal("5000ms result observation rejected", err)
				}
			} else if err == nil || f.server.owner.phase != mcpClosed {
				t.Fatal("stale or expired result authority published")
			}
			if kind == "key-expiry" && (observedOutput == nil || observedOutput.keyExpires == nil) {
				t.Fatal("key expiry was not checked at publication")
			}
			if f.state(t) != "COMPLETED" {
				t.Fatal("result publication changed durable outcome")
			}
		})
	}
}

func TestMCPProtectedReplyPendingEndsOnlyTransportInvocation(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	f.gate.bounds.claim = 30 * time.Second
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	if err := f.server.replyProtected(context.Background(), replyIO{}); err != nil {
		t.Fatal(err)
	}
	if f.server.owner.active != nil {
		t.Fatal("completed transport retained active invocation")
	}
	if r, err := f.admit(t); err != nil || r.Committed() || r.State() != "EXECUTING" {
		t.Fatal("poll during retained execution", err)
	}
	if err := f.server.replyProtected(context.Background(), replyIO{}); err != nil {
		t.Fatal(err)
	}
	if f.executor.effects.Load() != 0 {
		t.Fatal("poll executed effect")
	}
	occupied := 0
	for _, w := range f.gate.slots {
		if w != nil {
			occupied++
		}
	}
	if occupied != 1 {
		t.Fatal("pending response released execution capacity")
	}
	f.clock.mono.Add(20000)
	if ran, err := f.gate.runOne(context.Background()); !ran || err != nil {
		t.Fatal("historical execution lost", err)
	}
	if f.server.owner.phase != mcpReady || f.state(t) != "COMPLETED" || f.executor.effects.Load() != 1 {
		t.Fatal("completed transport deadline closed live owner")
	}
}

func TestMCPProtectedReplyDeferredFrameRemainsUnaccepted(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	frame := []byte("not yet authenticated")
	if err := f.server.replyProtected(context.Background(), replyIO{send: func(context.Context, []byte) error { return f.server.owner.deferFrame(frame, true) }}); err != nil {
		t.Fatal(err)
	}
	frame[0] = 'x'
	if got := f.server.owner.takeDeferred(); string(got) != "not yet authenticated" {
		t.Fatal("deferred snapshot changed")
	}
	if f.executor.effects.Load() != 0 {
		t.Fatal("deferred input executed")
	}
}

type replyPanicSigner struct{ ResultSigner }

func (s replyPanicSigner) Sign(context.Context, string, []byte) ([]byte, error) {
	panic("bounded provider failure")
}
func TestMCPProtectedReplyLedgerRetirementStopsQueuedWork(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	config := *f.config
	config.signer = replyPanicSigner{config.signer}
	if err := f.gate.replace(&config); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	if err := f.server.replyProtected(context.Background(), replyIO{}); err == nil || !f.gate.retired {
		t.Fatal("reply storage scope stayed available")
	}
	if ran, _ := f.gate.runOne(context.Background()); ran || f.executor.effects.Load() != 0 || f.state(t) != "UNKNOWN" {
		t.Fatal("retired scope dispatched work")
	}
}

func TestMCPProtectedReplyQuotaSharedAcrossOwners(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	a, b, _ := setupSessions(t)
	client, err := newMCPSetupSession(a, "second-client", "1")
	if err != nil {
		t.Fatal(err)
	}
	server, err := newMCPGuardSetup(b, "second-server", "1", f.gate)
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()
	defer server.close()
	x, y := net.Pipe()
	ea, eb := runSetupPair(t, client, server, &setupPipe{Conn: x}, &setupPipe{Conn: y}, func(context.Context) error { return nil })
	if ea != nil || eb != nil {
		t.Fatal(ea, eb)
	}
	if _, err = f.admit(t); err != nil {
		t.Fatal(err)
	}
	if _, err = f.gate.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- f.server.replyProtected(context.Background(), replyIO{send: func(context.Context, []byte) error { close(entered); <-release; return nil }})
	}()
	<-entered
	raw, err := MCPRequest(MCPVersion, guuid.NewString(), f.intent)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := client.session.SealRequest(context.Background(), raw, 30)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = server.admitProtected(context.Background(), wire, f.gate); err != nil {
		t.Fatal(err)
	}
	sends := 0
	err = server.replyProtected(context.Background(), replyIO{send: func(context.Context, []byte) error { sends++; return nil }})
	close(release)
	firstErr := <-done
	if err == nil || sends != 0 || firstErr != nil || f.server.owner.phase != mcpReady || server.owner.phase != mcpClosed {
		t.Fatal("output capacity or close isolation", err, firstErr)
	}
	if f.executor.effects.Load() != 1 || f.state(t) != "COMPLETED" {
		t.Fatal("quota changed durable execution")
	}
}
