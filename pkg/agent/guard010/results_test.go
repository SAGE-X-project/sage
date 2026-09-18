package guard010_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/sage-x-project/sage/pkg/agent/execution010"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type resultSigner struct {
	now                     int64
	active, fail, corrupt   bool
	signs, clocks, expireAt int
	kid                     string
}

func signer() *resultSigner {
	return &resultSigner{now: 1700000000, active: true, kid: "did:sage:web:agents.example.com:executor#signing-1"}
}
func resultKey() ed25519.PrivateKey {
	s := sha256.Sum256([]byte("public Guard fixture executor"))
	return ed25519.NewKeyFromSeed(s[:])
}
func (s *resultSigner) Now(context.Context) (int64, error) {
	s.clocks++
	if s.clocks == s.expireAt {
		return s.now + 300, nil
	}
	return s.now, nil
}
func (s *resultSigner) ActiveKey(_ context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if !s.active || issuer != "did:sage:web:agents.example.com:executor" || kid != s.kid {
		return nil, g.ErrInvalid
	}
	return resultKey().Public().(ed25519.PublicKey), nil
}
func (s *resultSigner) KeyID(context.Context) (string, error) { return s.kid, nil }
func (s *resultSigner) Sign(_ context.Context, kid string, b []byte) ([]byte, error) {
	s.signs++
	if s.fail || kid != s.kid {
		return nil, g.ErrInvalid
	}
	p := ed25519.Sign(resultKey(), b)
	if s.corrupt {
		p[0] ^= 1
	}
	return p, nil
}
func resultStatus(t *testing.T, b []byte) string {
	t.Helper()
	var e map[string]any
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("result JSON")
	}
	return e["result"].(map[string]any)["status"].(string)
}
func call(t *testing.T, gate *g.DispatchGate, f *gateFixture) *g.DispatchReceipt {
	t.Helper()
	r, e := gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestPendingFinishSingleReplyAndExactReuse(t *testing.T) {
	gate, f, s, p := openGate(t)
	ctx := context.Background()
	r := call(t, gate, f)
	copyReceipt := *r
	key := signer()
	pending, e := gate.Reply(ctx, r, key)
	if e != nil || resultStatus(t, pending) != "pending" {
		t.Fatal(e)
	}
	if _, e = gate.Reply(ctx, &copyReceipt, key); e == nil {
		t.Fatal("copied receipt replied twice")
	}
	if e = gate.Finish(ctx, s.observed.Completion(), []byte(`{"value":"ok"}`), key); e != nil {
		t.Fatal(e)
	}
	if _, e = gate.Reply(ctx, r, key); e == nil {
		t.Fatal("completion pushed after pending")
	}
	fresh := call(t, gate, f)
	terminal, e := gate.Reply(ctx, fresh, key)
	if e != nil || resultStatus(t, terminal) != "completed" {
		t.Fatal(e)
	}
	before := bridgeBytes(t, p)
	signs := key.signs
	if e = gate.Finish(ctx, s.observed.Completion(), []byte("{\n\"value\":\"ok\"}"), key); e != nil {
		t.Fatal(e)
	}
	again, e := gate.Reply(ctx, call(t, gate, f), key)
	if e != nil || !bytes.Equal(terminal, again) || key.signs != signs || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("terminal refreshed")
	}
	if gate.Finish(ctx, s.observed.Completion(), []byte(`{"value":"other"}`), key) == nil {
		t.Fatal("terminal conflict")
	}
	if s.commits != 1 {
		t.Fatal("redispatched")
	}
}
func TestAcceptedLateResultAndExpiredRetrieval(t *testing.T) {
	gate, f, s, _ := openGate(t)
	ctx := context.Background()
	r := call(t, gate, f)
	key := signer()
	key.now = 1700000301
	f.guardFixture["now"] = json.RawMessage(`1700000301`)
	if gate.Finish(ctx, s.observed.Completion(), []byte(`{"late":true}`), key) != nil {
		t.Fatal("accepted late result")
	}
	b, e := gate.Reply(ctx, r, key)
	if e != nil || resultStatus(t, b) != "completed" {
		t.Fatal("late response", e)
	}
	if _, e = gate.Dispatch(ctx, bridgeRaw(f.guardFixture)); e == nil {
		t.Fatal("expired retrieval")
	}
}
func TestResultSignerFailuresAndNoRefresh(t *testing.T) {
	for _, kind := range []string{"unavailable", "wrong-key", "corrupt", "revoked", "expired"} {
		t.Run(kind, func(t *testing.T) {
			gate, f, s, p := openGate(t)
			ctx := context.Background()
			r := call(t, gate, f)
			key := signer()
			if kind == "revoked" || kind == "expired" {
				if gate.Finish(ctx, s.observed.Completion(), []byte(`{}`), key) != nil {
					t.Fatal("finish")
				}
				before := bridgeBytes(t, p)
				signs := key.signs
				if kind == "revoked" {
					key.active = false
				} else {
					key.now += 300
				}
				if _, e := gate.Reply(ctx, r, key); e == nil {
					t.Fatal("invalid stored result published")
				}
				if key.signs != signs || !bytes.Equal(before, bridgeBytes(t, p)) {
					t.Fatal("invalid result refreshed")
				}
			} else {
				switch kind {
				case "unavailable":
					key.fail = true
				case "wrong-key":
					key.kid = "did:sage:web:agents.example.com:other#signing-1"
				case "corrupt":
					key.corrupt = true
				}
				before := bridgeBytes(t, p)
				if gate.Finish(ctx, s.observed.Completion(), []byte(`{}`), key) == nil || !bytes.Equal(before, bridgeBytes(t, p)) {
					t.Fatal("bad signature persisted")
				}
				key.fail = false
				key.corrupt = false
				key.kid = signer().kid
				if gate.Finish(ctx, s.observed.Completion(), []byte(`{}`), key) != nil {
					t.Fatal("signer recovery")
				}
			}
		})
	}
}
func TestTerminalExpiryDuringStorageRetainsFirstBytes(t *testing.T) {
	gate, f, s, p := openGate(t)
	r := call(t, gate, f)
	key := signer()
	key.expireAt = 5
	if gate.Finish(context.Background(), s.observed.Completion(), []byte(`{}`), key) == nil {
		t.Fatal("late validity exposed")
	}
	if stateAt(t, p) != "COMPLETED" {
		t.Fatal("durable terminal lost")
	}
	before := bridgeBytes(t, p)
	key.now += 300
	key.expireAt = 0
	if _, e := gate.Reply(context.Background(), r, key); e == nil || key.signs != 1 || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("stored expired result refreshed")
	}
}
func TestRejectReservesAndCannotOverwriteExecution(t *testing.T) {
	gate, f, s, p := openGate(t)
	ctx := context.Background()
	key := signer()
	r, e := gate.Reject(ctx, bridgeRaw(f.guardFixture), key)
	if e != nil {
		t.Fatal(e)
	}
	b, e := gate.Reply(ctx, r, key)
	if e != nil || resultStatus(t, b) != "rejected" || stateAt(t, p) != "REJECTED" {
		t.Fatal(e)
	}
	before := bridgeBytes(t, p)
	r = call(t, gate, f)
	again, e := gate.Reply(ctx, r, key)
	if e != nil || !bytes.Equal(b, again) || s.commits != 0 || key.signs != 1 || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("rejection replaced or executed")
	}
	other, ff, ss, pp := openGate(t)
	_ = call(t, other, ff)
	before = bridgeBytes(t, pp)
	if _, e = other.Reject(ctx, bridgeRaw(ff.guardFixture), key); e == nil || !bytes.Equal(before, bridgeBytes(t, pp)) || ss.commits != 1 {
		t.Fatal("rejection overwrote execution")
	}
}
func TestRecoveryUnknownAndGateBoundTokens(t *testing.T) {
	gate, f, s, p := openGate(t)
	r := call(t, gate, f)
	token := s.observed.Completion()
	if gate.Close() != nil {
		t.Fatal("close")
	}
	f2 := &gateFixture{guardFixture: bridgeFixture(t)}
	sink := &gateSink{f: f2.guardFixture, path: p}
	reopened, e := g.OpenDispatchGate(p, false, f2.s("expected_recipient"), f2, f2, sink)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = reopened.Close() }()
	key := signer()
	ctx := context.Background()
	if reopened.Finish(ctx, token, []byte(`{}`), key) == nil {
		t.Fatal("old token completed recovery")
	}
	if _, e = reopened.Reply(ctx, r, key); e == nil {
		t.Fatal("foreign reply token")
	}
	receipt := call(t, reopened, f2)
	key.fail = true
	before := bridgeBytes(t, p)
	if _, e = reopened.Reply(ctx, receipt, key); e == nil || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("unavailable unknown signer")
	}
	key.fail = false
	b, e := reopened.Reply(ctx, call(t, reopened, f2), key)
	if e != nil || resultStatus(t, b) != "unknown" {
		t.Fatal(e)
	}
	signs := key.signs
	again, e := reopened.Reply(ctx, call(t, reopened, f2), key)
	if e != nil || !bytes.Equal(b, again) || key.signs != signs || sink.commits != 0 {
		t.Fatal("unknown changed")
	}
}
func TestConcurrentFinishAndReply(t *testing.T) {
	gate, f, s, _ := openGate(t)
	r := call(t, gate, f)
	token := s.observed.Completion()
	key := signer()
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- gate.Finish(context.Background(), token, []byte(`{"value":"same"}`), key)
		}()
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		if e != nil {
			t.Fatal(e)
		}
	}
	if key.signs != 1 {
		t.Fatal("duplicate terminal signing")
	}
	replies := make(chan bool, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := gate.Reply(context.Background(), r, key); replies <- e == nil }()
	}
	wg.Wait()
	close(replies)
	n := 0
	for ok := range replies {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatal("more than one reply", n)
	}
}

func TestFinishStorageCapacityDoesNotPublishCompletion(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	p := filepath.Join(t.TempDir(), "journal")
	var b bytes.Buffer
	b.WriteString("sage-execution-ledger|0.10.0\n")
	// Owned offline capacity fixture; opaque filler is not cryptographic evidence.
	for n := 0; n < 4094; n++ {
		e := execution010.Entry{Issuer: "fixture", Recipient: "fixture", CallID: fmt.Sprint(n), Nonce: fmt.Sprint(n), Expires: 1700000300, IntentHex: "7b7d", State: "REJECTED", ResultHex: "7b7d"}
		raw, _ := json.Marshal(e)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if os.WriteFile(p, b.Bytes(), 0600) != nil {
		t.Fatal("fixture")
	}
	f := &gateFixture{guardFixture: bridgeFixture(t)}
	sink := &gateSink{f: f.guardFixture, path: p}
	gate, e := g.OpenDispatchGate(p, false, f.s("expected_recipient"), f, f, sink)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = gate.Close() }()
	r := call(t, gate, f)
	before := bridgeBytes(t, p)
	key := signer()
	if gate.Finish(context.Background(), sink.observed.Completion(), []byte(`{}`), key) == nil {
		t.Fatal("capacity accepted completion")
	}
	if !bytes.Equal(before, bridgeBytes(t, p)) || stateAt(t, p) != "EXECUTING" {
		t.Fatal("failed completion changed journal")
	}
	raw, e := gate.Reply(context.Background(), r, key)
	if e == nil && resultStatus(t, raw) != "pending" {
		t.Fatal("unpersisted completed result escaped")
	}
	if gate.Replace(f, f, sink) == nil {
		t.Fatal("failed gate reactivated")
	}
}
func TestMalformedOutputAndZeroTokenDoNotWrite(t *testing.T) {
	gate, f, s, p := openGate(t)
	_ = call(t, gate, f)
	key := signer()
	before := bridgeBytes(t, p)
	for _, output := range []string{`[]`, `null`, `{"x":1,"x":2}`, `{"x":-0}`} {
		if gate.Finish(context.Background(), s.observed.Completion(), []byte(output), key) == nil {
			t.Fatal("bad output accepted", output)
		}
	}
	if gate.Finish(context.Background(), &g.Completion{}, []byte(`{}`), key) == nil || key.signs != 0 || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("invalid input signed or persisted")
	}
}

func TestUnknownAndRejectCapacityNeverReleaseTerminal(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "journal")
			f := &gateFixture{guardFixture: bridgeFixture(t)}
			var b bytes.Buffer
			b.WriteString("sage-execution-ledger|0.10.0\n")
			write := func(e execution010.Entry) { raw, _ := json.Marshal(e); b.Write(raw); b.WriteByte('\n') }
			count := 4096
			if unknown {
				count = 4094
			}
			for n := 0; n < count; n++ {
				write(execution010.Entry{Issuer: "fixture", Recipient: "fixture", CallID: fmt.Sprint(n), Nonce: fmt.Sprint(n), Expires: 1700000300, IntentHex: "7b7d", State: "REJECTED", ResultHex: "7b7d"})
			}
			if unknown {
				raw := bridgeRaw(f.guardFixture)
				var env struct {
					Intent struct {
						Issuer, Recipient string
						CallID            string `json:"call_id"`
						Nonce             string
						Expires           int64
					}
				}
				if json.Unmarshal(raw, &env) != nil {
					t.Fatal("fixture")
				}
				i := env.Intent
				e := execution010.Entry{Issuer: i.Issuer, Recipient: i.Recipient, CallID: i.CallID, Nonce: i.Nonce, Expires: i.Expires, IntentHex: fmt.Sprintf("%x", raw), State: "RESERVED"}
				write(e)
				e.State = "UNKNOWN"
				write(e)
			}
			if os.WriteFile(p, b.Bytes(), 0600) != nil {
				t.Fatal("fixture")
			}
			sink := &gateSink{f: f.guardFixture, path: p}
			gate, e := g.OpenDispatchGate(p, false, f.s("expected_recipient"), f, f, sink)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = gate.Close() }()
			before := bridgeBytes(t, p)
			key := signer()
			if unknown {
				r := call(t, gate, f)
				if raw, e := gate.Reply(context.Background(), r, key); e == nil || len(raw) != 0 {
					t.Fatal("unpersisted unknown published")
				}
			} else {
				if r, e := gate.Reject(context.Background(), bridgeRaw(f.guardFixture), key); e == nil || r != nil {
					t.Fatal("unpersisted rejection accepted")
				}
			}
			if !bytes.Equal(before, bridgeBytes(t, p)) || sink.commits != 0 {
				t.Fatal("failed storage changed state or executed")
			}
		})
	}
}
