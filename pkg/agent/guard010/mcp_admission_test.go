package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	guuid "github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/execution010"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type admissionPolicy struct {
	original         string
	policy, manifest []byte
}

func (p *admissionPolicy) Bindings(context.Context, string, string) (string, []byte, []byte, error) {
	return p.original, p.policy, p.manifest, nil
}
func (p *admissionPolicy) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	if issuer != setupAlice || tool != "read" || string(args) != `{"path":"public.txt"}` {
		return ErrInvalid
	}
	return nil
}

type admissionSigner struct {
	*RegistryAuthority
	key ed25519.PrivateKey
}

func (s *admissionSigner) KeyID(context.Context) (string, error) { return setupBob + "#signing-1", nil }
func (s *admissionSigner) Sign(_ context.Context, kid string, b []byte) ([]byte, error) {
	if kid != setupBob+"#signing-1" {
		return nil, ErrInvalid
	}
	return ed25519.Sign(s.key, b), nil
}

type admissionExecutor struct {
	manifest         string
	effects          atomic.Int64
	checks           atomic.Int64
	mu               sync.Mutex
	observed         *Invocation
	output           []byte
	entered, release chan struct{}
	checkHook        func(int64) error
}

func (e *admissionExecutor) Check(_ context.Context, manifest, tool string) error {
	n := e.checks.Add(1)
	if e.checkHook != nil {
		if err := e.checkHook(n); err != nil {
			return err
		}
	}
	if manifest != e.manifest || tool != "read" {
		return ErrInvalid
	}
	return nil
}
func (e *admissionExecutor) Run(ctx context.Context, i *Invocation) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ErrInvalid
	}
	e.effects.Add(1)
	e.mu.Lock()
	e.observed = i
	e.mu.Unlock()
	if e.entered != nil {
		close(e.entered)
		<-e.release
	}
	if e.output != nil {
		return append([]byte(nil), e.output...), nil
	}
	return []byte(`{"ok":true}`), nil
}

type admissionFixture struct {
	gate           *mcpAdmissionGate
	client, server *mcpSetupSession
	clock          *setupRegistry
	executor       *admissionExecutor
	config         *mcpAdmissionConfig
	intent         []byte
	path           string
	streams        [2]*mcpStream
}

func newAdmissionFixture(t *testing.T, capacity int) *admissionFixture {
	return newAdmissionFixtureWithSetup(t, capacity, nil)
}
func newAdmissionFixtureWithSetup(t *testing.T, capacity int, beforeSetup func(*admissionFixture)) *admissionFixture {
	t.Helper()
	a, b, clock := setupSessions(t)
	raw, err := os.ReadFile("testdata/guard-rpc.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct{ Input map[string]json.RawMessage }
	if json.Unmarshal(raw, &data) != nil {
		t.Fatal("fixture")
	}
	var envHex string
	_ = json.Unmarshal(data.Input["envelope_hex"], &envHex)
	envRaw, _ := hex.DecodeString(envHex)
	var env map[string]any
	_ = json.Unmarshal(envRaw, &env)
	intent := env["intent"].(map[string]any)
	var policy map[string]any
	_ = json.Unmarshal(data.Input["approved_policy"], &policy)
	policy["issuer"] = setupAlice
	policyRaw := encode(policy)
	pd, err := PolicyCommitment(policyRaw)
	if err != nil {
		t.Fatal(err)
	}
	intent["issuer"], intent["recipient"], intent["keyid"] = setupAlice, setupBob, setupAlice+"#signing-1"
	now, _ := clock.Now()
	intent["created"], intent["expires"], intent["policy_digest"] = now.Unix, now.Unix+300, pd
	canonical, err := Canonicalize(encode(intent))
	if err != nil {
		t.Fatal(err)
	}
	env["proof"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)), append([]byte("sage-execution-intent|0.10.0\x00"), canonical...)))
	intentRaw, err := Canonicalize(encode(env))
	if err != nil {
		t.Fatal(err)
	}
	j, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "authority"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	registry, err := registry010.NewGate(registry010.Config{Source: "setup-test", Registry: "web:agent.example", Network: "local"}, clock, clock, j)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := NewRegistryAuthority(registry, setupAlice, setupAlice+"#signing-1")
	if err != nil {
		t.Fatal(err)
	}
	resultAuthority, err := NewRegistryAuthority(registry, setupBob, setupBob+"#signing-1")
	if err != nil {
		t.Fatal(err)
	}
	executor := &admissionExecutor{manifest: intent["manifest_digest"].(string)}
	config := &mcpAdmissionConfig{authority: authority, resultAuthority: resultAuthority, policy: &admissionPolicy{original: intent["original_digest"].(string), policy: policyRaw, manifest: data.Input["approved_manifest"]}, executor: executor, signer: &admissionSigner{RegistryAuthority: resultAuthority, key: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, 32))}}
	path := filepath.Join(t.TempDir(), "execution")
	g, err := openMCPAdmissionGate(path, true, setupBob, config, clock, mcpBounds{capacity: capacity, request: 20 * time.Second, claim: 10 * time.Second, worker: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newMCPSetupSession(a, "client", "1")
	if err != nil {
		t.Fatal(err)
	}
	server, err := newMCPGuardSetup(b, "server", "1", g)
	if err != nil {
		t.Fatal(err)
	}
	f := &admissionFixture{gate: g, client: client, server: server, clock: clock, executor: executor, config: config, intent: intentRaw, path: path}
	t.Cleanup(func() {
		client.close()
		server.close()
		g.retire()
		for range g.bounds.capacity {
			_, _ = g.runOne(context.Background())
		}
		_ = g.close()
	})
	if beforeSetup != nil {
		beforeSetup(f)
	}
	x, y := net.Pipe()
	left, _ := newMCPStream(x, 3*time.Second)
	right, _ := newMCPStream(y, 3*time.Second)
	f.streams = [2]*mcpStream{left, right}
	t.Cleanup(func() { _ = left.Close(); _ = right.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		e := server.run(ctx, right, func(context.Context) error { return nil })
		if e != nil {
			cancel()
		}
		done <- e
	}()
	ea := client.run(ctx, left, nil)
	if ea != nil {
		cancel()
	}
	eb := <-done
	if ea != nil || eb != nil {
		t.Fatal(ea, eb)
	}
	return f
}
func (f *admissionFixture) wire(t *testing.T) []byte {
	t.Helper()
	raw, err := MCPRequest(MCPVersion, guuid.NewString(), f.intent)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := f.client.session.SealRequest(context.Background(), raw, 30)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
func (f *admissionFixture) admit(t *testing.T) (*DispatchReceipt, error) {
	t.Helper()
	return f.server.admitProtected(context.Background(), f.wire(t), f.gate)
}
func (f *admissionFixture) state(t *testing.T) string {
	t.Helper()
	_, m, _, _ := intentEnvelope(f.intent)
	e, ok, err := f.gate.ledger.store.Lookup(str(m, "issuer"), str(m, "call_id"))
	if err != nil || !ok {
		t.Fatal("missing entry", err)
	}
	return e.State
}
func TestMCPAdmissionAuthenticatedExecutionAndDuplicate(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	r, err := f.admit(t)
	if err != nil || !r.Committed() || f.executor.effects.Load() != 0 || f.state(t) != "EXECUTING" {
		t.Fatal("fence/queue", err)
	}
	ran, err := f.gate.runOne(context.Background())
	if err != nil || !ran || f.executor.effects.Load() != 1 || f.state(t) != "COMPLETED" {
		t.Fatal("worker completion", err)
	}
	if !bytes.Equal(f.executor.observed.CanonicalIntent(), f.intent) {
		t.Fatal("changed invocation")
	}
	if err := f.server.replyProtected(context.Background(), replyIO{}); err != nil {
		t.Fatal(err)
	}
	r, err = f.admit(t)
	if err != nil || r.Committed() || r.Created() || r.State() != "COMPLETED" {
		t.Fatal("duplicate", err)
	}
	if ran, _ := f.gate.runOne(context.Background()); ran || f.executor.effects.Load() != 1 {
		t.Fatal("redispatch")
	}
}
func TestMCPAdmissionCloseDuringFence(t *testing.T) {
	for _, mode := range []string{"success", "failure", "uncertain"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			entered, release := make(chan struct{}), make(chan struct{})
			commit := f.gate.commit
			f.gate.commit = func(e execution010.Entry) (bool, error) {
				if e.State == "EXECUTING" {
					close(entered)
					<-release
					if mode == "failure" {
						return false, ErrInvalid
					}
					changed, err := commit(e)
					if mode == "uncertain" {
						return false, errors.New("uncertain persistence")
					}
					return changed, err
				}
				return commit(e)
			}
			wire := f.wire(t)
			done := make(chan error, 1)
			go func() { _, err := f.server.admitProtected(context.Background(), wire, f.gate); done <- err }()
			<-entered
			closed := make(chan struct{})
			go func() { f.server.close(); close(closed) }()
			select {
			case <-closed:
			case <-time.After(time.Second):
				close(release)
				t.Fatal("close waited for fence")
			}
			close(release)
			if err := <-done; err == nil {
				t.Fatal("closed admission")
			}
			if ran, _ := f.gate.runOne(context.Background()); ran || f.executor.effects.Load() != 0 {
				t.Fatal("effect after failed admission")
			}
			want := "UNKNOWN"
			if mode == "failure" {
				want = "RESERVED"
			}
			if mode == "uncertain" {
				want = "EXECUTING"
			}
			if f.state(t) != want {
				t.Fatal("invented fence/outcome", f.state(t))
			}
			if mode != "success" && !f.gate.retired {
				t.Fatal("uncertain store remained available")
			}
		})
	}
}
func TestMCPAdmissionCloseAfterInsertionKeepsAdmission(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	f.server.close()
	if ran, err := f.gate.runOne(context.Background()); !ran || err != nil || f.executor.effects.Load() != 1 {
		t.Fatal("close rolled back admitted work", err)
	}
}
func TestMCPAdmissionReplacementAndClaimOrder(t *testing.T) {
	for _, claimed := range []bool{false, true} {
		t.Run(map[bool]string{false: "replace-first", true: "claim-first"}[claimed], func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			if _, err := f.admit(t); err != nil {
				t.Fatal(err)
			}
			replacement := *f.config
			replacement.executor = &admissionExecutor{manifest: f.executor.manifest}
			if !claimed {
				if err := f.gate.replace(&replacement); err != nil {
					t.Fatal(err)
				}
				ran, _ := f.gate.runOne(context.Background())
				if ran || f.executor.effects.Load() != 0 || f.state(t) != "UNKNOWN" {
					t.Fatal("cancelled entry executed")
				}
				return
			}
			f.executor.entered = make(chan struct{})
			f.executor.release = make(chan struct{})
			done := make(chan struct{})
			go func() { _, _ = f.gate.runOne(context.Background()); close(done) }()
			<-f.executor.entered
			if err := f.gate.replace(&replacement); err != nil {
				t.Fatal(err)
			}
			close(f.executor.release)
			<-done
			if f.executor.effects.Load() != 1 || replacement.executor.(*admissionExecutor).effects.Load() != 0 || f.state(t) != "UNKNOWN" {
				t.Fatal("running instance replaced or completion fabricated")
			}
		})
	}
}
func TestMCPAdmissionDeadlineGenerationAndScheduler(t *testing.T) {
	for _, kind := range []string{"request-expiry", "generation", "scheduler"} {
		t.Run(kind, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			if kind != "scheduler" {
				f.executor.checkHook = func(n int64) error {
					if n == 2 {
						if kind == "request-expiry" {
							f.clock.mono.Add(20000)
						} else {
							return f.gate.replace(f.config)
						}
					}
					return nil
				}
			}
			r, err := f.admit(t)
			if kind == "scheduler" {
				if err != nil || !r.Committed() {
					t.Fatal(err)
				}
				f.clock.mono.Add(10000)
				ran, _ := f.gate.runOne(context.Background())
				if ran {
					t.Fatal("late scheduler claim")
				}
			} else if err == nil {
				t.Fatal("expired/changed generation admitted")
			}
			if f.executor.effects.Load() != 0 || f.state(t) != "UNKNOWN" {
				t.Fatal("denial outcome")
			}
		})
	}
}
func TestMCPAdmissionCapacityRetainedUntilWorkerTerminates(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	if _, err := f.admit(t); err != nil {
		t.Fatal(err)
	}
	f.executor.entered = make(chan struct{})
	f.executor.release = make(chan struct{})
	done := make(chan struct{})
	go func() { _, _ = f.gate.runOne(context.Background()); close(done) }()
	<-f.executor.entered
	f.server.close()
	f.gate.mu.Lock()
	occupied := f.gate.slots[0] != nil
	f.gate.mu.Unlock()
	if !occupied {
		t.Fatal("slot released before termination")
	}
	close(f.executor.release)
	<-done
	f.gate.mu.Lock()
	occupied = f.gate.slots[0] != nil
	f.gate.mu.Unlock()
	if occupied {
		t.Fatal("slot not released after termination")
	}
}

func TestMCPAdmissionCloseDuringAuthentication(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	wire := f.wire(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.clock.readHook = func(string) { once.Do(func() { close(entered); <-release }) }
	go func() { defer close(done); _, _ = f.server.admitProtected(context.Background(), wire, f.gate) }()
	<-entered
	closed := make(chan struct{})
	go func() { f.server.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		close(release)
		<-done
		t.Fatal("close waited for registry authentication")
	}
	close(release)
	<-done
	if f.server.owner.phase != mcpClosed || f.executor.effects.Load() != 0 {
		t.Fatal("closed owner admitted work")
	}
}

func TestMCPAdmissionSessionRevocationDuringFence(t *testing.T) {
	for _, kind := range []string{"responder-signing", "session-kem"} {
		t.Run(kind, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			commit := f.gate.commit
			f.gate.commit = func(e execution010.Entry) (bool, error) {
				changed, err := commit(e)
				if e.State == "EXECUTING" {
					if kind == "responder-signing" {
						f.clock.bobSigning.Store(true)
					} else {
						f.clock.bobKEM.Store(true)
					}
				}
				return changed, err
			}
			if _, err := f.admit(t); err == nil {
				t.Fatal("revoked session admitted")
			}
			if f.state(t) != "UNKNOWN" || f.server.owner.phase != mcpClosed || f.executor.effects.Load() != 0 {
				t.Fatal("revocation outcome")
			}
		})
	}
}

func TestMCPAdmissionRequestExpiryClosesOwner(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-insertion", true: "after-insertion"}[after], func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			if !after {
				f.executor.checkHook = func(n int64) error {
					if n == 2 {
						f.clock.mono.Add(20000)
					}
					return nil
				}
			} else {
				f.gate.bounds.claim = 30 * time.Second
			}
			_, err := f.admit(t)
			if after {
				if err != nil {
					t.Fatal(err)
				}
				f.clock.mono.Add(20000)
				ran, err := f.gate.runOne(context.Background())
				if !ran || err != nil || f.state(t) != "COMPLETED" || f.executor.effects.Load() != 1 {
					t.Fatal("historical admission lost", err)
				}
			} else if err == nil || f.state(t) != "UNKNOWN" {
				t.Fatal("expired request admitted")
			}
			if f.server.owner.phase != mcpClosed {
				t.Fatal("expired owner remains ready")
			}
			if _, err := f.server.session.LocalNow(); err == nil {
				t.Fatal("expired session retains keys")
			}
		})
	}
}

type admissionClockFunc func() (registry010.Stamp, error)

func (f admissionClockFunc) Now() (registry010.Stamp, error) { return f() }

func TestMCPAdmissionFinalObservationBoundaries(t *testing.T) {
	for _, age := range []int64{4999, 5000, 5001} {
		t.Run(time.Duration(age*time.Millisecond.Nanoseconds()).String(), func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			samples := 0
			f.gate.clock = admissionClockFunc(func() (registry010.Stamp, error) {
				samples++
				if samples == 3 {
					f.clock.mono.Add(age)
				}
				return f.clock.Now()
			})
			r, err := f.admit(t)
			if age <= 5000 {
				if err != nil || !r.Committed() {
					t.Fatal("fresh observation denied", err)
				}
			} else if err == nil || f.state(t) != "UNKNOWN" {
				t.Fatal("stale observation admitted")
			}
		})
	}
}

func TestMCPAdmissionFinalClockCannotPrecedeOwner(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	f.executor.checkHook = func(n int64) error {
		if n == 2 {
			f.clock.mono.Add(2000)
		}
		return nil
	}
	ownerClock := f.server.owner.clock
	ownerSamples := 0
	f.server.owner.clock = func() (time.Duration, error) {
		ownerSamples++
		if ownerSamples == 3 {
			f.clock.mono.Add(1000)
		}
		return ownerClock()
	}
	gateSamples := 0
	f.gate.clock = admissionClockFunc(func() (registry010.Stamp, error) {
		gateSamples++
		stamp, err := f.clock.Now()
		if gateSamples == 3 {
			stamp.MonoMS -= 500
		}
		return stamp, err
	})
	if _, err := f.admit(t); err == nil || f.state(t) != "UNKNOWN" || f.executor.effects.Load() != 0 {
		t.Fatal("clock rollback admitted")
	}
}

func TestMCPAdmissionSharedOwners(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(map[bool]string{false: "close-isolation", true: "shared-capacity"}[full], func(t *testing.T) {
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
			t.Cleanup(client.close)
			t.Cleanup(server.close)
			x, y := net.Pipe()
			ea, eb := runSetupPair(t, client, server, &setupPipe{Conn: x}, &setupPipe{Conn: y}, func(context.Context) error { return nil })
			if ea != nil || eb != nil {
				t.Fatal(ea, eb)
			}
			if full {
				if _, err := f.admit(t); err != nil {
					t.Fatal(err)
				}
			} else {
				f.server.close()
			}
			raw, err := MCPRequest(MCPVersion, guuid.NewString(), f.intent)
			if err != nil {
				t.Fatal(err)
			}
			wire, err := client.session.SealRequest(context.Background(), raw, 30)
			if err != nil {
				t.Fatal(err)
			}
			r, err := server.admitProtected(context.Background(), wire, f.gate)
			if full {
				if err == nil || server.owner.phase != mcpClosed || f.server.owner.phase != mcpReady {
					t.Fatal("shared capacity or owner isolation failed")
				}
			} else if err != nil || !r.Committed() {
				t.Fatal("closed peer retired independent owner", err)
			}
			ran, err := f.gate.runOne(context.Background())
			if !ran || err != nil || f.executor.effects.Load() != 1 {
				t.Fatal("independent admission lost", err)
			}
		})
	}
}
