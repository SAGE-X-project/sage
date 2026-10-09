package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

func publicBounds() MCPHostBounds {
	return MCPHostBounds{Capacity: 2, Preparations: 2, Clients: 2, Owners: 4, Workers: 1, Request: 20 * time.Second, Claim: 10 * time.Second, Worker: time.Second, Client: 20 * time.Second, Tick: time.Millisecond}
}
func publicServices(f *admissionFixture) MCPHostServices {
	return MCPHostServices{IntentAuthority: f.config.authority, ResultAuthority: f.config.resultAuthority, Policy: f.config.policy, Executor: f.executor, Signer: f.config.signer, Clock: f.clock}
}
func publicInputs(t *testing.T, f *admissionFixture) ([]byte, *RootCapture) {
	t.Helper()
	var env map[string]any
	if e := json.Unmarshal(f.intent, &env); e != nil {
		t.Fatal(e)
	}
	i := env["intent"].(map[string]any)
	capture, e := NewRootCapture([][]byte{[]byte("trusted root input")}, str(i, "request_id"))
	if e != nil {
		t.Fatal(e)
	}
	i["original_digest"] = capture.digest
	f.config.policy.(*admissionPolicy).original = capture.digest
	canonical, e := Canonicalize(encode(i))
	if e != nil {
		t.Fatal(e)
	}
	env["proof"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)), append([]byte("sage-execution-intent|0.10.0\x00"), canonical...)))
	raw, e := Canonicalize(encode(env))
	if e != nil {
		t.Fatal(e)
	}
	return raw, capture
}
func publicConfig(initiator bool) MCPConnectionConfig {
	c := MCPConnectionConfig{Role: MCPResponder, Name: "public fixture", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}
	if initiator {
		c.Role = MCPInitiator
		c.Recipient = setupBob
		c.RecipientKey = setupBob + "#signing-1"
	}
	return c
}

type publicHandler struct {
	endpoint func(context.Context) (*hpke.CompletionEndpoint010, error)
	prepare  func(context.Context) error
	handle   func(context.Context, *MCPConnection) error
}

func (h *publicHandler) Endpoint(ctx context.Context) (*hpke.CompletionEndpoint010, error) {
	return h.endpoint(ctx)
}
func (h *publicHandler) Prepare(ctx context.Context) error {
	if h.prepare != nil {
		return h.prepare(ctx)
	}
	return nil
}
func (h *publicHandler) Handle(ctx context.Context, c *MCPConnection) error { return h.handle(ctx, c) }
func closePublic(t *testing.T, h *MCPHost) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e := h.Close(ctx); e != nil {
		t.Error(e)
	}
}
func TestMCPPublicValidationBeforeStorage(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	for _, mode := range []string{"capacity", "preparations", "owners", "workers", "deadline", "fractional", "tick", "missing service"} {
		t.Run(mode, func(t *testing.T) {
			b := publicBounds()
			s := publicServices(f)
			switch mode {
			case "capacity":
				b.Capacity = 0
			case "preparations":
				b.Preparations = 1
			case "owners":
				b.Owners = 257
			case "workers":
				b.Workers = 3
			case "deadline":
				b.Worker = 6 * time.Minute
			case "fractional":
				b.Client++
			case "tick":
				b.Tick = b.Worker
			case "missing service":
				s.Executor = nil
			}
			path := filepath.Join(t.TempDir(), "gate")
			if h, e := OpenMCPHost(path, true, setupBob, s, b); e == nil {
				closePublic(t, h)
				t.Fatal("invalid host configuration accepted")
			}
			if _, e := os.Stat(path); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("invalid configuration created protected storage")
			}
		})
	}
	var zero MCPHost
	if e := zero.Close(context.Background()); e == nil {
		t.Fatal("zero host")
	}
	var c MCPConnection
	if _, e := c.Exchange(); e == nil {
		t.Fatal("zero connection")
	}
	if e := c.ServeOne(); e == nil {
		t.Fatal("zero connection served")
	}
}
func TestMCPPublicTCPRuntime(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	intent, capture := publicInputs(t, f)
	h, e := OpenMCPHost(filepath.Join(t.TempDir(), "public-gate"), true, setupBob, publicServices(f), publicBounds())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { closePublic(t, h) })
	a, b, endpointClock := setupEndpoints(t)
	tcp, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	var prepared atomic.Bool
	server := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return b, nil }, prepare: func(context.Context) error { prepared.Store(true); return nil }, handle: func(_ context.Context, c *MCPConnection) error {
		if !prepared.Load() {
			return ErrInvalid
		}
		for {
			if e := c.ServeOne(); e != nil {
				return e
			}
		}
	}}
	go func() { done <- h.Serve(context.Background(), tcp, 1, publicConfig(false), server) }()
	conn, e := net.DialTimeout("tcp", tcp.Addr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	var retained *MCPConnection
	var terminal *ClientDelivery
	client := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return a, nil }, handle: func(_ context.Context, c *MCPConnection) error {
		copied := *c
		retained = &copied

		if e := c.OpenRootClient(filepath.Join(t.TempDir(), "client"), true, intent, MCPClientServices{IntentAuthority: f.config.authority, ResultAuthority: f.config.resultAuthority, Policy: f.config.policy, Clock: replyClientClock{f}}, capture); e != nil {
			return e
		}
		for range 20 {
			d, e := c.Exchange()
			if e != nil {
				return e
			}
			if d.Status() == "completed" {
				terminal = d
				return nil
			}
			time.Sleep(20 * time.Millisecond)
			f.clock.mono.Add(1000)
			endpointClock.mono.Add(1000)
		}
		return ErrInvalid
	}}
	if e = h.Connect(context.Background(), conn, publicConfig(true), client); e != nil {
		t.Fatal(e)
	}
	if terminal == nil || !terminal.FirstTerminal() || string(terminal.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("public delivery or effect count")
	}
	if _, e = retained.Exchange(); e == nil {
		t.Fatal("escaped callback scope")
	}
	if e = retained.OpenRootClient("unused", true, intent, MCPClientServices{}, capture); e == nil {
		t.Fatal("retired handle opened client")
	}
	closePublic(t, h)
	if e = <-done; !errors.Is(e, context.Canceled) {
		t.Fatal("listener did not terminate", e)
	}
	if _, _, e = a.Start(context.Background(), setupBob, setupBob+"#signing-1", 300); e == nil {
		t.Fatal("owned endpoint survived callback")
	}
}
func TestMCPPublicChargedFactoryAndBoundedClose(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	bounds := publicBounds()
	bounds.Owners = 1
	path := filepath.Join(t.TempDir(), "gate")
	h, e := OpenMCPHost(path, true, setupBob, publicServices(f), bounds)
	if e != nil {
		t.Fatal(e)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var factories atomic.Int64
	handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) {
		factories.Add(1)
		once.Do(func() { close(entered) })
		<-release
		return nil, ErrInvalid
	}, handle: func(context.Context, *MCPConnection) error { t.Error("blocked factory published handler"); return nil }}
	x, y := net.Pipe()
	defer y.Close()
	done := make(chan error, 1)
	go func() { done <- h.Connect(context.Background(), x, publicConfig(true), handler) }()
	<-entered
	rejected, peer := net.Pipe()
	defer peer.Close()
	if e = h.Connect(context.Background(), rejected, publicConfig(true), handler); e == nil || factories.Load() != 1 {
		t.Fatal("factory before quota")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if e = h.Close(ctx); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("blocked factory was released", e)
	}
	cancel()
	if other, e := OpenMCPHost(path, false, setupBob, publicServices(f), bounds); e == nil {
		closePublic(t, other)
		t.Fatal("timeout released ledger lock")
	}
	close(release)
	if e = <-done; e == nil {
		t.Fatal("retired factory succeeded")
	}
	closePublic(t, h)
	reopened, e := OpenMCPHost(path, false, setupBob, publicServices(f), bounds)
	if e != nil {
		t.Fatal("drained ledger could not reopen", e)
	}
	closePublic(t, reopened)
	if e = h.Connect(context.Background(), rejected, publicConfig(true), handler); e == nil || factories.Load() != 1 {
		t.Fatal("closed host invoked provider")
	}
}
func TestMCPPublicConfigurationRejectsBeforeFactory(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	h, e := OpenMCPHost(filepath.Join(t.TempDir(), "gate"), true, setupBob, publicServices(f), publicBounds())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { closePublic(t, h) })
	var count atomic.Int64
	handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { count.Add(1); return nil, ErrInvalid }, handle: func(context.Context, *MCPConnection) error { return nil }}
	for _, mode := range []string{"role", "ttl", "timeout", "fractional", "key", "responder key"} {
		cfg := publicConfig(true)
		switch mode {
		case "role":
			cfg.Role = 0
		case "ttl":
			cfg.TTLSeconds = 301
		case "timeout":
			cfg.Timeout = 0
		case "fractional":
			cfg.Timeout++
		case "key":
			cfg.RecipientKey = setupAlice + "#signing-1"
		case "responder key":
			cfg.Role = MCPResponder
		}
		a, b := net.Pipe()
		if e = h.Connect(context.Background(), a, cfg, handler); e == nil {
			t.Fatal(mode)
		}
		_ = b.SetReadDeadline(time.Now().Add(time.Second))
		if _, e = b.Read(make([]byte, 1)); e == nil {
			t.Fatal("socket not closed")
		}
		_ = b.Close()
	}
	if count.Load() != 0 {
		t.Fatal("invalid configuration invoked provider")
	}
}

func TestMCPPublicRejectsReadinessAndOriginalBeforeEffect(t *testing.T) {
	for _, mode := range []string{"prepare denied", "changed capture"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdmissionFixture(t, 2)
			raw, capture := publicInputs(t, f)
			h, e := OpenMCPHost(filepath.Join(t.TempDir(), "gate"), true, setupBob, publicServices(f), publicBounds())
			if e != nil {
				t.Fatal(e)
			}
			defer closePublic(t, h)
			a, b, _ := setupEndpoints(t)
			tcp, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return b, nil }, prepare: func(context.Context) error {
				if mode == "prepare denied" {
					return ErrInvalid
				}
				return nil
			}, handle: func(_ context.Context, c *MCPConnection) error { return c.ServeOne() }}
			go func() { done <- h.Serve(context.Background(), tcp, 1, publicConfig(false), handler) }()
			conn, e := net.DialTimeout("tcp", tcp.Addr().String(), time.Second)
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "client")
			var called bool
			if mode == "changed capture" {
				capture, e = NewRootCapture([][]byte{[]byte("changed root")}, capture.requestID)
				if e != nil {
					t.Fatal(e)
				}
			}
			client := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return a, nil }, handle: func(_ context.Context, c *MCPConnection) error {
				called = true
				return c.OpenRootClient(path, true, raw, MCPClientServices{IntentAuthority: f.config.authority, ResultAuthority: f.config.resultAuthority, Policy: f.config.policy, Clock: replyClientClock{f}}, capture)
			}}
			if e = h.Connect(context.Background(), conn, publicConfig(true), client); e == nil {
				t.Fatal("invalid readiness or original accepted")
			}
			if mode == "prepare denied" && called {
				t.Fatal("handler before readiness")
			}
			if _, e = os.Stat(path); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("invalid request persisted")
			}
			if f.executor.effects.Load() != 0 {
				t.Fatal("effect before readiness or capture")
			}
			closePublic(t, h)
			if e = <-done; !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
		})
	}
}

// Fixed public fixture process: only loopback and inert exact read arguments.
type publicProcessClock struct{ *setupRegistry }

func (c publicProcessClock) Sample(context.Context) (int64, int64, error) {
	m := c.mono.Load()
	return 100000 + m, m, nil
}
func publicProcessEndpoint(t *testing.T, c publicProcessClock, initiator bool) *hpke.CompletionEndpoint010 {
	t.Helper()
	did, seed := setupBob, byte(2)
	if initiator {
		did, seed = setupAlice, 1
	}
	j, e := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = j.Close() })
	gate, e := registry010.NewGate(registry010.Config{Source: "setup-test", Registry: "web:agent.example", Network: "local"}, c, c, j)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := hpke.OpenReplayJournal010(filepath.Join(t.TempDir(), "replay"), true, c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = replay.Close() })
	var kem []byte
	if !initiator {
		kem = bytes.Repeat([]byte{3}, 32)
	}
	endpoint, e := hpke.NewCompletionEndpoint010(did, did+"#signing-1", bytes.Repeat([]byte{seed}, 32), kem, gate, c, replay)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(endpoint.Close)
	return endpoint
}
func publicProcessAuthority(t *testing.T, c publicProcessClock, did string) *RegistryAuthority {
	t.Helper()
	j, e := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = j.Close() })
	gate, e := registry010.NewGate(registry010.Config{Source: "setup-test", Registry: "web:agent.example", Network: "local"}, c, c, j)
	if e != nil {
		t.Fatal(e)
	}
	authority, e := NewRegistryAuthority(gate, did, did+"#signing-1")
	if e != nil {
		t.Fatal(e)
	}
	return authority
}

// publicMapping is receiver administration for the fixture's one provisioned
// commitment; it never holds the original request.
type publicMapping struct {
	policy, manifest []byte
	authorize        func(context.Context, string, string, []byte) error
}

func (m *publicMapping) Approved(_ context.Context, issuer, digest string) ([]byte, []byte, error) {
	canonical, e := Canonicalize(m.policy)
	if e != nil {
		return nil, nil, e
	}
	if d, e := PolicyCommitment(canonical); e != nil || issuer != setupAlice || d != digest {
		return nil, nil, ErrInvalid
	}
	return m.policy, m.manifest, nil
}
func (m *publicMapping) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	return m.authorize(ctx, issuer, tool, args)
}

func TestMCPPublicProcessHelper(t *testing.T) {
	root := os.Getenv("SAGE_MCP_PUBLIC_TEST_ROOT")
	mode := os.Getenv("SAGE_MCP_PUBLIC_TEST_MODE")
	if root == "" {
		t.Skip("bounded fixture helper")
	}
	if mode != "server" && mode != "client" {
		t.Fatal("fixed fixture mode")
	}
	f := newAdmissionFixture(t, 2)
	raw, capture := publicInputs(t, f)
	clock := publicProcessClock{&setupRegistry{}}
	endpoint := publicProcessEndpoint(t, clock, mode == "client")
	clock.mono.Store(360000)
	intentAuthority := publicProcessAuthority(t, clock, setupAlice)
	resultAuthority := publicProcessAuthority(t, clock, setupBob)
	var env map[string]any
	if json.Unmarshal(raw, &env) != nil {
		t.Fatal("fixture")
	}
	intent := env["intent"].(map[string]any)
	intent["created"], intent["expires"] = int64(460), int64(760)
	body, _ := Canonicalize(encode(intent))
	env["proof"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)), append([]byte("sage-execution-intent|0.10.0\x00"), body...)))
	raw, _ = Canonicalize(encode(env))
	signer := &admissionSigner{RegistryAuthority: resultAuthority, key: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, 32))}
	gatePath := filepath.Join(root, mode+"-gate")
	policy := f.config.policy
	if mode == "server" && os.Getenv("SAGE_MCP_PUBLIC_RECEIVER_MAPPING") == "1" {
		// A separate receiver: it holds no original, so the fixture's original
		// is replaced and verification must use the provisioned mapping.
		fixed := f.config.policy.(*admissionPolicy)
		mapped, e := NewReceiverPolicy(&publicMapping{policy: fixed.policy, manifest: fixed.manifest, authorize: fixed.Authorize})
		if e != nil {
			t.Fatal(e)
		}
		fixed.original = strings.Repeat("0", 64)
		policy = mapped
	}
	h, e := OpenMCPHost(gatePath, true, setupBob, MCPHostServices{IntentAuthority: intentAuthority, ResultAuthority: resultAuthority, Policy: policy, Executor: f.executor, Signer: signer, Clock: clock}, publicBounds())
	if e != nil {
		t.Fatal(e)
	}
	defer closePublic(t, h)
	var status string
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if mode == "server" {
		tcp, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "address"), []byte(tcp.Addr().String()), 0600); e != nil {
			t.Fatal(e)
		}
		watchDone := make(chan struct{})
		go func() {
			defer close(watchDone)
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, e := os.Stat(filepath.Join(root, "finished")); e == nil {
						cancel()
						return
					}
				}
			}
		}()
		handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) {
			return endpoint, nil
		}, handle: func(_ context.Context, c *MCPConnection) error {
			for {
				if e := c.ServeOne(); e != nil {
					return e
				}
			}
		}}
		e = h.Serve(ctx, tcp, 1, publicConfig(false), handler)
		cancel()
		<-watchDone
		if !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
		if f.executor.effects.Load() != 1 {
			t.Fatal("fixed effect count")
		}
		status = "completed"
	} else {
		address, e := os.ReadFile(filepath.Join(root, "address"))
		if e != nil {
			t.Fatal(e)
		}
		host, port, e := net.SplitHostPort(string(address))
		n, pe := strconv.Atoi(port)
		if e != nil || pe != nil || host != "127.0.0.1" || n < 1 || n > 65535 {
			t.Fatal("only fixed loopback")
		}
		tcp, e := net.DialTimeout("tcp", string(address), time.Second)
		if e != nil {
			t.Fatal(e)
		}
		handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) {
			return endpoint, nil
		}, handle: func(_ context.Context, c *MCPConnection) error {
			if e := c.OpenRootClient(filepath.Join(root, "client-journal"), true, raw, MCPClientServices{IntentAuthority: intentAuthority, ResultAuthority: resultAuthority, Policy: f.config.policy, Clock: clock}, capture); e != nil {
				return e
			}
			for range 20 {
				d, e := c.Exchange()
				if e != nil {
					return e
				}
				if d.Status() == "completed" {
					if !d.FirstTerminal() || string(d.Output()) != `{"ok":true}` {
						return ErrInvalid
					}
					status = d.Status()
					return nil
				}
				time.Sleep(20 * time.Millisecond)
				clock.mono.Add(1000)
			}
			return ErrInvalid
		}}
		if e = h.Connect(ctx, tcp, publicConfig(true), handler); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "finished"), []byte("done"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	closePublic(t, h)
	ledger, e := os.ReadFile(gatePath)
	if e != nil {
		t.Fatal(e)
	}
	record := map[string]any{"mode": mode, "status": status, "effects": f.executor.effects.Load(), "ledger_hex": hex.EncodeToString(ledger)}
	if mode == "client" {
		journal, e := os.ReadFile(filepath.Join(root, "client-journal"))
		if e != nil {
			t.Fatal(e)
		}
		record["journal_hex"] = hex.EncodeToString(journal)
	}
	bytes, e := json.Marshal(record)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, mode+".json"), bytes, 0600); e != nil {
		t.Fatal(e)
	}
}
