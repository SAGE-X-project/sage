package guard010

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type ownedHopAuthority struct {
	issuer string
	now    int64
	key    ed25519.PublicKey
}

func (a ownedHopAuthority) Now(context.Context) (int64, error) { return a.now, nil }
func (a ownedHopAuthority) ActiveKey(_ context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if issuer != a.issuer || kid != issuer+"#signing-1" {
		return nil, ErrInvalid
	}
	return a.key, nil
}

type ownedHopPolicy struct {
	issuer, original string
	policy, manifest []byte
}

func (p ownedHopPolicy) Bindings(_ context.Context, issuer, _ string) (string, []byte, []byte, error) {
	if issuer != p.issuer {
		return "", nil, nil, ErrInvalid
	}
	return p.original, p.policy, p.manifest, nil
}
func (p ownedHopPolicy) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	if issuer != p.issuer || tool != "read" || string(args) != `{"path":"public.txt"}` {
		return ErrInvalid
	}
	return nil
}

type ownedHopParent struct {
	incoming []byte
	allowed  bool
}

func (p *ownedHopParent) Authorized(_ context.Context, incoming []byte) error {
	if !p.allowed || !bytes.Equal(incoming, p.incoming) {
		return ErrInvalid
	}
	return nil
}

func prepareOwnedHop(t *testing.T, f *admissionFixture) ([]byte, HopServices, *ownedHopParent) {
	t.Helper()
	const origin = "did:sage:web:agent.example:origin"
	var env map[string]any
	if json.Unmarshal(f.intent, &env) != nil {
		t.Fatal("outgoing")
	}
	parent := map[string]any{}
	for k, v := range env["intent"].(map[string]any) {
		parent[k] = v
	}
	parent["issuer"], parent["recipient"], parent["keyid"] = origin, setupAlice, origin+"#signing-1"
	parent["request_id"] = "00000000-0000-4000-8000-000000000031"
	parent["call_id"] = "00000000-0000-4000-8000-000000000032"
	var descriptor map[string]any
	policy := f.config.policy.(*admissionPolicy)
	if json.Unmarshal(policy.policy, &descriptor) != nil {
		t.Fatal("policy")
	}
	descriptor["issuer"] = origin
	parentPolicy := encode(descriptor)
	digest, err := PolicyCommitment(parentPolicy)
	if err != nil {
		t.Fatal(err)
	}
	parent["policy_digest"] = digest
	seed := bytes.Repeat([]byte{3}, 32)
	key := ed25519.NewKeyFromSeed(seed)
	canonical, err := Canonicalize(encode(parent))
	if err != nil {
		t.Fatal(err)
	}
	incoming, err := Canonicalize(encode(map[string]any{"intent": parent,
		"proof": base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, append([]byte("sage-execution-intent|0.10.0\x00"), canonical...)))}))
	if err != nil {
		t.Fatal(err)
	}
	original, err := OriginalCommitment([][]byte{incoming})
	if err != nil {
		t.Fatal(err)
	}
	policy.original = original
	child := env["intent"].(map[string]any)
	child["original_digest"] = original
	canonical, err = Canonicalize(encode(child))
	if err != nil {
		t.Fatal(err)
	}
	f.intent, err = Canonicalize(encode(map[string]any{"intent": child,
		"proof": base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)), append([]byte("sage-execution-intent|0.10.0\x00"), canonical...)))}))
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := f.clock.Now()
	if err != nil {
		t.Fatal(err)
	}
	admitted := &ownedHopParent{incoming: incoming, allowed: true}
	return incoming, HopServices{Authority: ownedHopAuthority{issuer: origin, now: stamp.Unix, key: key.Public().(ed25519.PublicKey)},
		Policy: ownedHopPolicy{issuer: origin, original: parent["original_digest"].(string), policy: parentPolicy, manifest: policy.manifest}, Parent: admitted}, admitted
}

func TestMCPOwnedHopUsesParentGateBeforeRuntimeSend(t *testing.T) {
	for _, mode := range []string{"denied at open", "revoked before send", "allowed"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdmissionFixture(t, 2)
			incoming, hop, parent := prepareOwnedHop(t, f)
			pool, err := newMCPClientPool(f.clock, 2, 20*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "client")
			parent.allowed = mode != "denied at open"
			b, err := openMCPOwnedHopClient(context.Background(), f.client, pool, path, true,
				incoming, f.intent, hop, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
			if mode == "denied at open" {
				if err == nil || b != nil {
					t.Fatal("unadmitted parent opened MCP client")
				}
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatal("unadmitted parent created journal")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			parent.allowed = mode == "allowed"
			io := &ownedClientIO{f: f, execute: true}
			delivery, err := b.exchange(context.Background(), io)
			if mode == "allowed" {
				if err != nil || delivery == nil || delivery.Status() != "completed" || io.sent != 1 || f.executor.effects.Load() != 1 {
					t.Fatalf("authorized hop did not complete: %v", err)
				}
			} else if err == nil || delivery != nil || io.sent != 0 || f.executor.effects.Load() != 0 {
				t.Fatal("revoked parent reached MCP transport")
			}
		})
	}
}

type ownedClientIO struct {
	f             *admissionFixture
	execute       bool
	response      []byte
	sent          int
	afterSend     func() error
	beforeReceive func()
}

func (i *ownedClientIO) Send(ctx context.Context, wire []byte) error {
	i.sent++
	if _, e := i.f.server.admitProtected(ctx, wire, i.f.gate); e != nil {
		return e
	}
	if i.execute {
		if _, e := i.f.gate.runOne(ctx); e != nil {
			return e
		}
	}
	if e := i.f.server.replyProtected(ctx, replyIO{send: func(_ context.Context, b []byte) error { i.response = append([]byte(nil), b...); return nil }}); e != nil {
		return e
	}
	if i.afterSend != nil {
		return i.afterSend()
	}
	return nil
}
func (i *ownedClientIO) Receive(context.Context) ([]byte, error) {
	if i.beforeReceive != nil {
		i.beforeReceive()
	}
	return append([]byte(nil), i.response...), nil
}
func ownedClientFixture(t *testing.T) (*admissionFixture, *mcpOwnedClient, string) {
	t.Helper()
	f := newAdmissionFixture(t, 2)
	p, e := newMCPClientPool(f.clock, 2, 20*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "owned-client")
	b, e := openMCPOwnedClient(context.Background(), f.client, p, path, true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	return f, b, path
}
func TestMCPOwnedClientRuntimeExchange(t *testing.T) {
	f, b, _ := ownedClientFixture(t)
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
			_, e = f.gate.runOne(ctx)
		}
		if e == nil {
			e = f.server.replyProtected(ctx, io)
		}
		done <- e
	}()
	delivery, e := b.exchange(ctx, &setupPipe{Conn: x})
	if e != nil {
		t.Fatal(e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	if !delivery.FirstTerminal() || delivery.Status() != "completed" || string(delivery.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("verified delivery")
	}
	io := &ownedClientIO{f: f, execute: true}
	if d, e := b.exchange(ctx, io); e == nil || d != nil || io.sent != 0 {
		t.Fatal("terminal retransmitted")
	}
}
func TestMCPOwnedClientPendingAndDeferredReply(t *testing.T) {
	f, b, _ := ownedClientFixture(t)
	io := &ownedClientIO{f: f}
	io.afterSend = func() error { return f.client.owner.deferFrame(io.response, true) }
	d, e := b.exchange(context.Background(), io)
	if e != nil || d.Status() != "pending" || len(d.Output()) != 0 || f.executor.effects.Load() != 0 {
		t.Fatal("pending", e)
	}
	if _, e = f.gate.runOne(context.Background()); e != nil {
		t.Fatal(e)
	}
	f.clock.mono.Add(1000)
	io.afterSend = nil
	d, e = b.exchange(context.Background(), io)
	if e != nil || !d.FirstTerminal() || string(d.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("poll", e)
	}
}
func TestMCPOwnedClientCloseAfterConsumptionWithholdsOutput(t *testing.T) {
	f, b, path := ownedClientFixture(t)
	io := &ownedClientIO{f: f, execute: true}
	io.beforeReceive = func() {
		f.clock.readHook = func(string) {
			if len(b.client.terminal) > 0 {
				f.client.close()
			}
		}
	}
	d, e := b.exchange(context.Background(), io)
	if e == nil || d != nil || len(b.client.terminal) == 0 {
		t.Fatal("closed delivery published or consumption lost")
	}
	f.clock.readHook = nil
	if f.client.owner.phase != mcpClosed {
		t.Fatal("owner remains open")
	}
	reopened, e := OpenClient(context.Background(), path, false, f.intent, ClientServices{IntentAuthority: f.config.authority, ResultAuthority: f.config.resultAuthority, Policy: f.config.policy, Clock: replyClientClock{f}, Sender: &replyClientSender{fixture: f}, ExpectedIssuer: setupAlice, ExpectedRecipient: setupBob})
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = reopened.Close() }()
	if len(reopened.terminal) == 0 {
		t.Fatal("restart lost durable consumption")
	}
	if _, e = reopened.Begin(context.Background(), "00000000-0000-4000-8000-000000000099"); e == nil {
		t.Fatal("restart repeated terminal")
	}
}
func TestMCPOwnedClientFailureBoundaries(t *testing.T) {
	for _, kind := range []string{"send-failure", "close", "deadline", "pool-retired", "revoked-session", "bad-correlation", "deferred-overflow"} {
		t.Run(kind, func(t *testing.T) {
			f, b, _ := ownedClientFixture(t)
			io := &ownedClientIO{f: f, execute: true}
			io.afterSend = func() error {
				switch kind {
				case "send-failure":
					return ErrInvalid
				case "close":
					f.client.close()
				case "deadline":
					f.clock.mono.Add(20000)
				case "pool-retired":
					b.pool.retire()
				case "revoked-session":
					f.clock.bobKEM.Store(true)
				case "bad-correlation":
					io.response = []byte(`{"message_id":"different"}`)
				case "deferred-overflow":
					if e := f.client.owner.deferFrame(io.response, true); e != nil {
						return e
					}
					return f.client.owner.deferFrame(io.response, true)
				}
				return nil
			}
			d, e := b.exchange(context.Background(), io)
			if e == nil || d != nil || f.client.owner.phase != mcpClosed || f.state(t) != "COMPLETED" {
				t.Fatal("failed exchange exposed result", e)
			}
			if _, e = f.client.session.LocalNow(); e == nil {
				t.Fatal("session not cleaned up")
			}
			for _, w := range b.pool.slots {
				if w != nil {
					t.Fatal("terminated exchange leaked quota")
				}
			}
		})
	}
}
func TestMCPOwnedClientBlockedSendRetainsCapacity(t *testing.T) {
	f, b, _ := ownedClientFixture(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, e := b.exchange(context.Background(), replyIO{send: func(context.Context, []byte) error { close(entered); <-release; return nil }})
		done <- e
	}()
	<-entered
	f.client.close()
	b.pool.mu.Lock()
	occupied := b.pool.slots[0] != nil
	b.pool.mu.Unlock()
	if !occupied {
		t.Fatal("blocked provider abandoned quota")
	}
	close(release)
	if e := <-done; e == nil {
		t.Fatal("late send published")
	}
	if f.executor.effects.Load() != 0 {
		t.Fatal("test caused remote effect")
	}
}

func TestMCPOwnedClientCanonicalEnvelope(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	p, _ := newMCPClientPool(f.clock, 1, 20*time.Second)
	var pretty bytes.Buffer
	if e := json.Indent(&pretty, f.intent, "", "  "); e != nil {
		t.Fatal(e)
	}
	b, e := openMCPOwnedClient(context.Background(), f.client, p, filepath.Join(t.TempDir(), "client"), true, pretty.Bytes(), f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(b.intent, f.intent) {
		t.Fatal("binding changed signed canonical intent")
	}
	d, e := b.exchange(context.Background(), &ownedClientIO{f: f, execute: true})
	if e != nil || !d.FirstTerminal() {
		t.Fatal("valid formatted envelope rejected", e)
	}
}

type blockedOwnedClock struct {
	ClientClock
	entered, release chan struct{}
	once             sync.Once
	panicSample      bool
}

func (c *blockedOwnedClock) Sample(ctx context.Context) (int64, int64, error) {
	if c.panicSample {
		panic("bounded clock failure")
	}
	c.once.Do(func() { close(c.entered); <-c.release })
	return c.ClientClock.Sample(ctx)
}
func TestMCPOwnedClientPreparationSharesCapacity(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	other := newAdmissionFixture(t, 1)
	p, _ := newMCPClientPool(f.clock, 1, 20*time.Second)
	clock := &blockedOwnedClock{ClientClock: replyClientClock{f}, entered: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	path := filepath.Join(t.TempDir(), "first")
	go func() {
		_, e := openMCPOwnedClient(context.Background(), f.client, p, path, true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, clock)
		done <- e
	}()
	<-clock.entered
	f.client.close()
	second := filepath.Join(t.TempDir(), "second")
	_, e := openMCPOwnedClient(context.Background(), other.client, p, second, true, other.intent, other.config.authority, other.config.resultAuthority, other.config.policy, replyClientClock{other})
	if e == nil {
		t.Fatal("reconnect bypassed preparation quota")
	}
	if _, e = os.Stat(second); !os.IsNotExist(e) {
		t.Fatal("exhausted preparation created journal")
	}
	p.mu.Lock()
	occupied := p.slots[0] != nil
	p.mu.Unlock()
	if !occupied {
		t.Fatal("closed blocked preparation released capacity")
	}
	close(clock.release)
	if e = <-done; e == nil {
		t.Fatal("closed preparation published")
	}
	if p.slots[0] != nil {
		t.Fatal("terminated preparation leaked capacity")
	}
}
func TestMCPOwnedClientPreparationFailures(t *testing.T) {
	for _, kind := range []string{"retired", "clock-panic", "expired"} {
		t.Run(kind, func(t *testing.T) {
			f := newAdmissionFixture(t, 1)
			p, _ := newMCPClientPool(f.clock, 1, 20*time.Second)
			var clock ClientClock = replyClientClock{f}
			if kind == "retired" {
				p.retire()
			}
			if kind == "clock-panic" {
				clock = &blockedOwnedClock{panicSample: true}
			}
			if kind == "expired" {
				clock = ownedClockFunc(func(ctx context.Context) (int64, int64, error) {
					f.clock.mono.Add(20000)
					return (replyClientClock{f}).Sample(ctx)
				})
			}
			path := filepath.Join(t.TempDir(), "client")
			if b, e := openMCPOwnedClient(context.Background(), f.client, p, path, true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, clock); e == nil || b != nil {
				t.Fatal("invalid preparation published")
			}
			if p.slots[0] != nil || f.client.owner.phase != mcpClosed {
				t.Fatal("preparation cleanup")
			}
			if kind == "retired" {
				if _, e := os.Stat(path); !os.IsNotExist(e) {
					t.Fatal("retired pool created journal")
				}
			}
		})
	}
}

type ownedClockFunc func(context.Context) (int64, int64, error)

func (c ownedClockFunc) Sample(ctx context.Context) (int64, int64, error) { return c(ctx) }

func TestMCPOwnedClientResultObservationBoundaries(t *testing.T) {
	for _, age := range []int64{5000, 5001} {
		t.Run(time.Duration(age*int64(time.Millisecond)).String(), func(t *testing.T) {
			f, b, _ := ownedClientFixture(t)
			io := &ownedClientIO{f: f, execute: true}
			io.beforeReceive = func() {
				reads := 0
				f.clock.readHook = func(did string) {
					if did == setupBob && len(b.client.terminal) > 0 {
						reads++
						if reads == 9 {
							f.clock.mono.Add(age)
						}
					}
				}
			}
			d, e := b.exchange(context.Background(), io)
			if age == 5000 {
				if e != nil || d == nil {
					t.Fatal("fresh observation rejected", e)
				}
			} else if e == nil || d != nil || len(b.client.terminal) == 0 {
				t.Fatal("stale authority published or lost consumption")
			}
		})
	}
}

func TestMCPOwnedClientSetupFailureClosesJournal(t *testing.T) {
	f := newAdmissionFixture(t, 1)
	a, _, _ := setupSessions(t)
	s, e := newMCPSetupSession(a, "unready-client", "1")
	if e != nil {
		t.Fatal(e)
	}
	defer s.close()
	p, _ := newMCPClientPool(f.clock, 1, 20*time.Second)
	b, e := openMCPOwnedClient(context.Background(), s, p, filepath.Join(t.TempDir(), "client"), true, f.intent, f.config.authority, f.config.resultAuthority, f.config.policy, replyClientClock{f})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.run(context.Background(), replyIO{}, nil); e == nil {
		t.Fatal("missing setup response accepted")
	}
	if b.client.file != nil || s.owner.phase != mcpClosed {
		t.Fatal("setup failure retained client journal")
	}
}
