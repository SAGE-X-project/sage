package guard010_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/execution010"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type gateFixture struct {
	guardFixture
	keys, policies, clocks      int
	failKey, failPolicy, expire int
}

func (f *gateFixture) ActiveKey(ctx context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	f.keys++
	if f.keys == f.failKey {
		return nil, g.ErrInvalid
	}
	return f.guardFixture.ActiveKey(ctx, issuer, kid)
}
func (f *gateFixture) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	f.policies++
	if f.policies == f.failPolicy {
		return g.ErrInvalid
	}
	return f.guardFixture.Authorize(ctx, issuer, tool, args)
}
func (f *gateFixture) Now(ctx context.Context) (int64, error) {
	f.clocks++
	if f.clocks == f.expire {
		return 1700000300, nil
	}
	return f.guardFixture.Now(ctx)
}

type gateSink struct {
	f                       guardFixture
	path                    string
	checks, commits         int
	failCheck, panicCheck   int
	failCommit, panicCommit bool
	entered, release        chan struct{}
	cancel                  context.CancelFunc
	observed                *g.Invocation
}

func (s *gateSink) Check(_ context.Context, manifest, tool string) error {
	s.checks++
	if s.checks == s.panicCheck {
		panic("fixture check failure")
	}
	if s.checks == s.failCheck {
		return g.ErrInvalid
	}
	if s.checks == 2 && s.cancel != nil {
		s.cancel()
	}
	raw, _ := json.Marshal(s.f["approved_manifest"])
	digest, err := g.VerifyManifest(raw, []g.Artifact{{Path: "engine.bin", Bytes: []byte("public pinned evaluator")}, {Path: "rules.json", Bytes: []byte(`{"allow":["read"]}`)}})
	if err != nil || digest != manifest || tool != "read" {
		return g.ErrInvalid
	}
	return nil
}
func (s *gateSink) Commit(ctx context.Context, i *g.Invocation) error {
	if ctx.Err() != nil {
		return g.ErrInvalid
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	rows := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	var row map[string]any
	if json.Unmarshal(rows[len(rows)-1], &row) != nil || row["state"] != "EXECUTING" {
		return g.ErrInvalid
	}
	s.commits++
	s.observed = i
	if s.entered != nil {
		close(s.entered)
		<-s.release
	}
	if s.panicCommit {
		panic("fixture commit uncertainty")
	}
	if s.failCommit {
		return g.ErrInvalid
	}
	return nil
}
func openGate(t *testing.T) (*g.DispatchGate, *gateFixture, *gateSink, string) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	f := &gateFixture{guardFixture: bridgeFixture(t)}
	p := filepath.Join(t.TempDir(), "journal")
	s := &gateSink{f: f.guardFixture, path: p}
	gate, e := g.OpenDispatchGate(p, true, f.s("expected_recipient"), f, f, s)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = gate.Close() })
	return gate, f, s, p
}
func stateAt(t *testing.T, p string) string {
	t.Helper()
	rows := bytes.Split(bytes.TrimSpace(bridgeBytes(t, p)), []byte("\n"))
	if len(rows) == 1 {
		return ""
	}
	var r map[string]any
	if json.Unmarshal(rows[len(rows)-1], &r) != nil {
		t.Fatal("row")
	}
	return r["state"].(string)
}
func TestDispatchExactOnceAndRecovery(t *testing.T) {
	gate, f, s, p := openGate(t)
	raw := bridgeRaw(f.guardFixture)
	r, e := gate.Dispatch(context.Background(), raw)
	if e != nil || !r.Created() || !r.Committed() || r.State() != "EXECUTING" {
		t.Fatal(r, e)
	}
	if s.checks != 2 || s.commits != 1 || !bytes.Equal(s.observed.CanonicalIntent(), raw) || string(s.observed.Arguments()) != `{"path":"public.txt"}` {
		t.Fatal("binding")
	}
	args := s.observed.Arguments()
	args[0] = '!'
	env := s.observed.CanonicalIntent()
	env[0] = '!'
	if !bytes.Equal(s.observed.CanonicalIntent(), raw) || string(s.observed.Arguments()) != `{"path":"public.txt"}` || s.observed.Tool() != "read" || s.observed.IntentDigest() != r.IntentDigest() {
		t.Fatal("mutable invocation")
	}
	before := bridgeBytes(t, p)
	r, e = gate.Dispatch(context.Background(), raw)
	if e != nil || r.Committed() || r.Created() || s.commits != 1 || !bytes.Equal(before, bridgeBytes(t, p)) {
		t.Fatal("duplicate dispatch")
	}
	if gate.Close() != nil {
		t.Fatal("close")
	}
	f2 := &gateFixture{guardFixture: bridgeFixture(t)}
	s2 := &gateSink{f: f2.guardFixture, path: p}
	recovered, e := g.OpenDispatchGate(p, false, f2.s("expected_recipient"), f2, f2, s2)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = recovered.Close() }()
	r, e = recovered.Dispatch(context.Background(), raw)
	if e != nil || r.State() != "UNKNOWN" || r.Committed() || s2.commits != 0 {
		t.Fatal("recovery dispatched", e)
	}
}
func TestDispatchFinalDenials(t *testing.T) {
	for _, kind := range []string{"key", "policy", "expiry", "component", "cancel", "panic-check", "commit-error", "commit-panic"} {
		t.Run(kind, func(t *testing.T) {
			gate, f, s, p := openGate(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "key":
				f.failKey = 2
			case "policy":
				f.failPolicy = 2
			case "expiry":
				f.expire = 7
			case "component":
				s.failCheck = 2
			case "cancel":
				s.cancel = cancel
			case "panic-check":
				s.panicCheck = 2
			case "commit-error":
				s.failCommit = true
			case "commit-panic":
				s.panicCommit = true
			}
			r, e := gate.Dispatch(ctx, bridgeRaw(f.guardFixture))
			if e == nil || r != nil || stateAt(t, p) != "UNKNOWN" {
				t.Fatal("denial failed", kind, e, stateAt(t, p))
			}
			expected := 0
			if kind == "commit-error" || kind == "commit-panic" {
				expected = 1
			}
			if s.commits != expected {
				t.Fatal("unexpected handoff")
			}
			f.failKey = 0
			f.failPolicy = 0
			f.expire = 0
			s.failCheck = 0
			s.cancel = nil
			s.panicCheck = 0
			r, e = gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture))
			if kind == "panic-check" || kind == "commit-panic" {
				if e == nil {
					t.Fatal("panic did not retire")
				}
			} else if e != nil || r.State() != "UNKNOWN" || r.Committed() {
				t.Fatal("unknown retry", e)
			}
			if s.commits != expected {
				t.Fatal("retry after uncertainty")
			}
		})
	}
}
func TestDispatchRetireAndReplaceOrdering(t *testing.T) {
	for _, operation := range []string{"retire", "replace"} {
		t.Run(operation, func(t *testing.T) {
			gate, f, s, _ := openGate(t)
			s.entered = make(chan struct{})
			s.release = make(chan struct{})
			done := make(chan error, 1)
			go func() { _, e := gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture)); done <- e }()
			select {
			case <-s.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("commit missing")
			}
			update := make(chan error, 1)
			started := make(chan struct{})
			replacement := &gateSink{f: f.guardFixture, path: s.path}
			go func() {
				close(started)
				if operation == "retire" {
					update <- gate.Retire()
				} else {
					update <- gate.Replace(f, f, replacement)
				}
			}()
			<-started
			select {
			case <-update:
				close(s.release)
				t.Fatal("update crossed commit")
			case <-time.After(20 * time.Millisecond):
			}
			close(s.release)
			if e := <-done; e != nil {
				t.Fatal(e)
			}
			if e := <-update; e != nil {
				t.Fatal(e)
			}
			r, e := gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture))
			if operation == "retire" {
				if e == nil {
					t.Fatal("retired dispatch")
				}
				if gate.Replace(f, f, replacement) == nil {
					t.Fatal("retirement reversed")
				}
			} else if e != nil || r.Committed() {
				t.Fatal("replacement redispatched")
			}
			if s.commits != 1 || replacement.commits != 0 {
				t.Fatal("wrong instance")
			}
		})
	}
}
func TestDispatchRetirementAndCancellationBeforeReservation(t *testing.T) {
	for _, kind := range []string{"retire", "cancel", "component", "replace", "closed"} {
		t.Run(kind, func(t *testing.T) {
			gate, f, s, p := openGate(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "retire":
				_ = gate.Retire()
			case "cancel":
				cancel()
			case "component":
				s.failCheck = 1
			case "replace":
				other := &gateFixture{guardFixture: bridgeFixture(t), failPolicy: 1}
				if gate.Replace(other, other, s) != nil {
					t.Fatal("replace")
				}
			case "closed":
				_ = gate.Close()
			}
			before := bridgeBytes(t, p)
			if _, e := gate.Dispatch(ctx, bridgeRaw(f.guardFixture)); e == nil || s.commits != 0 || !bytes.Equal(before, bridgeBytes(t, p)) {
				t.Fatal("denial effects")
			}
		})
	}
}
func TestDispatchConcurrentDuplicates(t *testing.T) {
	gate, f, s, _ := openGate(t)
	raw := bridgeRaw(f.guardFixture)
	var wg sync.WaitGroup
	results := make(chan bool, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := gate.Dispatch(context.Background(), raw)
			results <- e == nil && r.Committed()
		}()
	}
	wg.Wait()
	close(results)
	count := 0
	for ok := range results {
		if ok {
			count++
		}
	}
	if count != 1 || s.commits != 1 {
		t.Fatal("multiple commits", count, s.commits)
	}
}

func TestDispatchIndependentIntentDenials(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	for _, c := range cases(t) {
		if c.Operation != "sage.guard.intent.verify" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			var f guardFixture
			if json.Unmarshal(c.Input, &f) != nil {
				t.Fatal("fixture")
			}
			p := filepath.Join(t.TempDir(), "journal")
			s := &gateSink{f: f, path: p}
			gate, e := g.OpenDispatchGate(p, true, f.s("expected_recipient"), f, f, s)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = gate.Close() }()
			before := bridgeBytes(t, p)
			r, e := gate.Dispatch(context.Background(), bridgeRaw(f))
			if c.Expected["verdict"] == "ACCEPT" {
				if e != nil || !r.Committed() || s.commits != 1 {
					t.Fatal("valid fixture denied", e)
				}
			} else if e == nil || s.commits != 0 || !bytes.Equal(before, bridgeBytes(t, p)) {
				t.Fatal("invalid fixture caused effects")
			}
		})
	}
}

func TestDispatchStorageCapacityRetiresGate(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	for _, rows := range []int{4094, 4095} {
		t.Run(fmt.Sprint(rows), func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "journal")
			var journal bytes.Buffer
			journal.WriteString("sage-execution-ledger|0.10.0\n")
			// Owned offline storage fixture. Opaque synthetic terminal bytes are not signed results.
			for n := 0; n < rows; n++ {
				e := execution010.Entry{Issuer: "fixture", Recipient: "fixture", CallID: fmt.Sprint(n), Nonce: fmt.Sprint(n), Expires: 1700000300, IntentHex: "7b7d", State: "REJECTED", ResultHex: "7b7d"}
				b, _ := json.Marshal(e)
				journal.Write(b)
				journal.WriteByte('\n')
			}
			if os.WriteFile(p, journal.Bytes(), 0600) != nil {
				t.Fatal("fixture")
			}
			f := &gateFixture{guardFixture: bridgeFixture(t)}
			s := &gateSink{f: f.guardFixture, path: p}
			if rows == 4094 {
				s.failCheck = 2
			}
			gate, e := g.OpenDispatchGate(p, false, f.s("expected_recipient"), f, f, s)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = gate.Close() }()
			if _, e = gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture)); e == nil || s.commits != 0 {
				t.Fatal("capacity allowed effects")
			}
			before := bridgeBytes(t, p)
			if gate.Replace(f, f, s) == nil {
				t.Fatal("failed gate reactivated")
			}
			if _, e = gate.Dispatch(context.Background(), bridgeRaw(f.guardFixture)); e == nil || s.commits != 0 || !bytes.Equal(before, bridgeBytes(t, p)) {
				t.Fatal("capacity retry")
			}
		})
	}
}
