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
	"os/exec"
	"path/filepath"
	"runtime"
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
	return newAdmissionFixtureAtPath(t, capacity, "", true, beforeSetup)
}
func newAdmissionFixtureAtPath(t *testing.T, capacity int, path string, create bool, beforeSetup func(*admissionFixture)) *admissionFixture {
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
	if path == "" {
		path = filepath.Join(t.TempDir(), "execution")
	}
	g, err := openMCPAdmissionGate(path, create, setupBob, config, clock, mcpBounds{capacity: capacity, request: 20 * time.Second, claim: 10 * time.Second, worker: time.Second})
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
func (f *admissionFixture) wireWithID(t *testing.T) ([]byte, string) {
	t.Helper()
	id := guuid.NewString()
	raw, err := MCPRequest(MCPVersion, id, f.intent)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := f.client.session.SealRequest(context.Background(), raw, 30)
	if err != nil {
		t.Fatal(err)
	}
	return wire, id
}
func (f *admissionFixture) wire(t *testing.T) []byte {
	t.Helper()
	wire, _ := f.wireWithID(t)
	return wire
}
func (f *admissionFixture) admit(t *testing.T) (*DispatchReceipt, error) {
	t.Helper()
	return f.server.admitProtected(context.Background(), f.wire(t), f.gate)
}
func (f *admissionFixture) entry(t *testing.T) execution010.Entry {
	t.Helper()
	_, m, _, _ := intentEnvelope(f.intent)
	e, ok, err := f.gate.ledger.store.Lookup(str(m, "issuer"), str(m, "call_id"))
	if err != nil || !ok {
		t.Fatal("missing entry", err)
	}
	return e
}
func (f *admissionFixture) state(t *testing.T) string {
	t.Helper()
	return f.entry(t).State
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
			f.gate.mu.Lock()
			historyBefore := len(f.server.owner.seen)
			f.gate.mu.Unlock()
			wire, requestID := f.wireWithID(t)
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
			entry := f.entry(t)
			if entry.State != want {
				t.Fatal("invented fence/outcome", entry.State)
			}
			_, intent, _, err := intentEnvelope(f.intent)
			if err != nil || entry.Nonce == "" || entry.Nonce != str(intent, "nonce") {
				t.Fatal("reservation nonce history not retained", err)
			}
			f.gate.mu.Lock()
			_, retained := f.server.owner.seen[requestID]
			historyAfter := len(f.server.owner.seen)
			f.gate.mu.Unlock()
			if !retained || historyAfter != historyBefore+1 {
				t.Fatal("protected request history not retained", retained, historyBefore, historyAfter)
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
func TestMCPAdmissionCrashHelper(t *testing.T) {
	path := os.Getenv("SAGE_MCP_ADMISSION_CRASH_PATH")
	if path == "" {
		return
	}
	f := newAdmissionFixtureAtPath(t, 1, path, true, nil)
	r, err := f.admit(t)
	if err != nil || !r.Created() || !r.Committed() || f.state(t) != "EXECUTING" || f.executor.effects.Load() != 0 {
		t.Fatal("durable admission before crash", err)
	}
	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Fatal("missing live owner lock", err)
	}
	os.Exit(0)
}
func TestMCPAdmissionCrashRecoveryDoesNotExecuteAgain(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("local durable storage requires Linux/macOS")
	}
	path := filepath.Join(t.TempDir(), "execution")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "-test.run=^TestMCPAdmissionCrashHelper$", "-test.count=1")
	command.Env = append(os.Environ(), "SAGE_MCP_ADMISSION_CRASH_PATH="+path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("crash fixture: %v\n%s", err, output)
	}
	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Fatal("crash released owner lock", err)
	}
	if ledger, err := execution010.Open(path, false); err == nil {
		_ = ledger.Close()
		t.Fatal("reopened before trusted recovery")
	}
	// The owned child has exited; this models trusted administration proving
	// exclusive ownership before clearing the stale process lock.
	if err := os.Remove(path + ".lock"); err != nil {
		t.Fatal("trusted lock recovery", err)
	}
	f := newAdmissionFixtureAtPath(t, 1, path, false, nil)
	if f.state(t) != "UNKNOWN" || f.executor.effects.Load() != 0 {
		t.Fatal("unresolved admission recovery")
	}
	r, err := f.admit(t)
	if err != nil || r.Created() || r.Committed() || r.State() != "UNKNOWN" {
		t.Fatal("recovered admission was redispatched", err)
	}
	if ran, err := f.gate.runOne(context.Background()); err != nil || ran || f.executor.effects.Load() != 0 {
		t.Fatal("recovered work executed", err)
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

func TestMCPAdmissionReadySessionExpiryDeniesNewAdmission(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	f.clock.mono.Store(f.client.session.CreatedMonoMS() + 599999)
	wire, _ := f.wireWithID(t)
	var protected map[string]any
	if err := json.Unmarshal(wire, &protected); err != nil {
		t.Fatal(err)
	}
	f.clock.mono.Add(1)
	now, err := f.clock.Now()
	if err != nil || now.Unix >= int64(protected["expires"].(float64)) {
		t.Fatal("protected message expired before session", err)
	}
	if _, err := f.server.admitProtected(context.Background(), wire, f.gate); err == nil {
		t.Fatal("expired ready session admitted new work")
	}
	if f.server.owner.phase != mcpClosed || f.executor.effects.Load() != 0 {
		t.Fatal("expired ready session remained usable")
	}
	if _, err := f.server.session.LocalNow(); err == nil {
		t.Fatal("expired session retained keys")
	}
}

func TestMCPAdmissionProtectedDeadlineBeforeFinalAdmissionRetainsReservation(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	f.gate.mu.Lock()
	historyBefore := len(f.server.owner.seen)
	f.gate.mu.Unlock()
	wire, requestID := f.wireWithID(t)
	f.executor.checkHook = func(n int64) error {
		if n == 2 {
			f.clock.mono.Add(20000)
		}
		return nil
	}
	if _, err := f.server.admitProtected(context.Background(), wire, f.gate); err == nil {
		t.Fatal("expired protected request admitted")
	}
	entry := f.entry(t)
	_, intent, _, err := intentEnvelope(f.intent)
	if err != nil || entry.State != "UNKNOWN" || entry.CallID != str(intent, "call_id") || entry.Nonce != str(intent, "nonce") {
		t.Fatal("intent reservation not retained", err, entry)
	}
	f.gate.mu.Lock()
	_, retained := f.server.owner.seen[requestID]
	historyAfter := len(f.server.owner.seen)
	f.gate.mu.Unlock()
	if !retained || historyAfter != historyBefore+1 {
		t.Fatal("protected request history not retained", retained, historyBefore, historyAfter)
	}
	if f.server.owner.phase != mcpClosed {
		t.Fatal("expired owner remains ready")
	}
	if ran, runErr := f.gate.runOne(context.Background()); runErr != nil || ran || f.executor.effects.Load() != 0 {
		t.Fatal("expired reservation executed", runErr)
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
