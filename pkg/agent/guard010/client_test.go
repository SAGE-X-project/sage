package guard010_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

type clientFixture struct {
	sentID     string
	sentIntent []byte
	sendDelay  int64
	sendFail   bool
	sends      int

	f                     guardFixture
	utc, mono             int64
	clockOK, resultActive bool
	pub                   string
}

func (f *clientFixture) Sample(context.Context) (int64, int64, error) {
	if !f.clockOK {
		return 0, 0, g.ErrInvalid
	}
	return f.utc, f.mono, nil
}
func (f *clientFixture) Now(context.Context) (int64, error) {
	if !f.clockOK {
		return 0, g.ErrInvalid
	}
	return f.utc / 1000, nil
}
func (f *clientFixture) ActiveKey(ctx context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if issuer == "did:sage:web:agents.example.com:executor" {
		if !f.resultActive || kid != issuer+"#signing-1" {
			return nil, g.ErrInvalid
		}
		b, e := hex.DecodeString(f.pub)
		return ed25519.PublicKey(b), e
	}
	return f.f.ActiveKey(ctx, issuer, kid)
}
func (f *clientFixture) Bindings(ctx context.Context, i, r string) (string, []byte, []byte, error) {
	return f.f.Bindings(ctx, i, r)
}
func (f *clientFixture) Authorize(ctx context.Context, i, t string, a []byte) error {
	return f.f.Authorize(ctx, i, t, a)
}
func (f *clientFixture) Commit(_ context.Context, id string, raw []byte) error {
	f.sends++
	f.sentID = id
	f.sentIntent = append([]byte(nil), raw...)
	f.utc += f.sendDelay
	f.mono += f.sendDelay
	if f.sendFail {
		return g.ErrInvalid
	}
	return nil
}
func (f *clientFixture) services() g.ClientServices {
	return g.ClientServices{IntentAuthority: f, Policy: f, ResultAuthority: f, Clock: f, Sender: f}
}

type clientObservation struct {
	Handoffs int    `json:"handoffs"`
	OK       bool   `json:"ok"`
	ID       string `json:"id"`
	Intent   string `json:"intent_hex"`
	Status   string `json:"status"`
	First    bool   `json:"first"`
	Ignored  bool   `json:"ignored"`
	Output   string `json:"output_hex"`
}
type clientStep struct {
	Action, ID, Result, Field string
	UTC                       int64 `json:"utc"`
	Mono                      int64
	Value                     bool
	Expected                  clientObservation
}
type clientSuite struct {
	Input   guardFixture
	Public  string `json:"public_key_hex"`
	Results map[string]string
	Cases   []struct {
		ID    string
		Steps []clientStep
	}
}

func clientVectors(t *testing.T) clientSuite {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	b, e := os.ReadFile("testdata/guard-client.json")
	if e != nil {
		t.Fatal(e)
	}
	var s clientSuite
	if json.Unmarshal(b, &s) != nil {
		t.Fatal("fixture")
	}
	return s
}
func clientSetup(t *testing.T) (*g.Client, *clientFixture, string, clientSuite) {
	t.Helper()
	s := clientVectors(t)
	f := &clientFixture{f: s.Input, utc: 1700000000000, clockOK: true, resultActive: true, pub: s.Public}
	p := filepath.Join(t.TempDir(), "journal")
	c, e := g.OpenClient(context.Background(), p, true, bridgeRaw(f.f), f.services())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, f, p, s
}
func outer(n int) string                         { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
func clientRaw(s clientSuite, key string) []byte { b, _ := hex.DecodeString(s.Results[key]); return b }
func clientApply(c *g.Client, f *clientFixture, tickets map[string]*g.ClientInvocation, q clientStep, results map[string]string) clientObservation {
	o := clientObservation{OK: true}
	ctx := context.Background()
	switch q.Action {
	case "tick":
		f.utc = q.UTC
		f.mono = q.Mono
	case "set":
		switch q.Field {
		case "clock_ok":
			f.clockOK = q.Value
		case "result_active":
			f.resultActive = q.Value
		case "intent_active":
			f.f["active_key"], _ = json.Marshal(q.Value)
		case "policy_allow":
			f.f["policy_allow"], _ = json.Marshal(q.Value)
		}
	case "begin":
		v, e := c.Begin(ctx, q.ID)
		o.OK = e == nil
		if e == nil {
			tickets[q.ID] = v
			o.ID = f.sentID
			o.Intent = hex.EncodeToString(f.sentIntent)
		}
	case "accept":
		b, _ := hex.DecodeString(results[q.Result])
		d, e := c.Accept(ctx, tickets[q.ID], b)
		o.OK = e == nil
		if e == nil {
			o.Status = d.Status()
			o.First = d.FirstTerminal()
			o.Ignored = d.Ignored()
			o.Output = hex.EncodeToString(d.Output())
		}
	case "failed":
		o.OK = c.Failed(tickets[q.ID]) == nil
	case "close":
		o.OK = c.Close() == nil
	default:
		o.OK = false
	}
	o.Handoffs = f.sends
	return o
}
func TestClientIndependentScenarios(t *testing.T) {
	suite := clientVectors(t)
	for _, scenario := range suite.Cases {
		t.Run(scenario.ID, func(t *testing.T) {
			c, f, _, s := clientSetup(t)
			tickets := map[string]*g.ClientInvocation{}
			for n, q := range scenario.Steps {
				got := clientApply(c, f, tickets, q, s.Results)
				if !reflect.DeepEqual(got, q.Expected) {
					t.Fatalf("step %d: got %+v want %+v", n, got, q.Expected)
				}
			}
		})
	}
}
func TestClientConcurrentTerminalIsConsumedOnce(t *testing.T) {
	c, f, _, s := clientSetup(t)
	tickets := []*g.ClientInvocation{}
	for n := 1; n <= 8; n++ {
		f.utc = 1700000000000 + int64(n)*1000
		f.mono = int64(n) * 1000
		v, e := c.Begin(context.Background(), outer(n))
		if e != nil {
			t.Fatal(e)
		}
		tickets = append(tickets, v)
	}
	var wg sync.WaitGroup
	out := make(chan *g.ClientDelivery, 8)
	for _, v := range tickets {
		wg.Add(1)
		go func(v *g.ClientInvocation) {
			defer wg.Done()
			d, e := c.Accept(context.Background(), v, clientRaw(s, "completed"))
			if e != nil {
				t.Error(e)
			}
			out <- d
		}(v)
	}
	wg.Wait()
	close(out)
	first := 0
	for d := range out {
		if d == nil {
			continue
		}
		if d.FirstTerminal() {
			first++
			if len(d.Output()) == 0 {
				t.Fatal("lost first output")
			}
		} else if !d.Ignored() || len(d.Output()) != 0 {
			t.Fatal("duplicate output")
		}
	}
	if first != 1 {
		t.Fatal(first)
	}
}
func TestClientRestartPreservesPollingAndConsumption(t *testing.T) {
	c, f, p, s := clientSetup(t)
	old, e := c.Begin(context.Background(), outer(1))
	if e != nil {
		t.Fatal(e)
	}
	if c.Close() != nil {
		t.Fatal("close")
	}
	f.utc += 1000
	reopened, e := g.OpenClient(context.Background(), p, false, bridgeRaw(f.f), f.services())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = reopened.Close() }()
	if _, e = reopened.Begin(context.Background(), outer(2)); e == nil {
		t.Fatal("restart bypassed minimum elapsed time")
	}
	f.mono = 1000
	if _, e = reopened.Begin(context.Background(), outer(1)); e == nil {
		t.Fatal("outer identity reused")
	}
	if _, e = reopened.Accept(context.Background(), old, clientRaw(s, "completed")); e == nil {
		t.Fatal("foreign invocation accepted")
	}
	next, e := reopened.Begin(context.Background(), outer(2))
	if e != nil {
		t.Fatal(e)
	}
	d, e := reopened.Accept(context.Background(), next, clientRaw(s, "completed"))
	if e != nil || !d.FirstTerminal() {
		t.Fatal(e)
	}
	if reopened.Close() != nil {
		t.Fatal("close")
	}
	again, e := g.OpenClient(context.Background(), p, false, bridgeRaw(f.f), f.services())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = again.Close() }()
	f.utc += 1000
	f.mono += 1000
	if _, e = again.Begin(context.Background(), outer(3)); e == nil {
		t.Fatal("terminal reopened")
	}
	if _, e = again.Accept(context.Background(), next, clientRaw(s, "completed")); e == nil {
		t.Fatal("terminal redelivered")
	}
}
func TestClientMissingExclusiveAndCorruptStorage(t *testing.T) {
	c, f, p, _ := clientSetup(t)
	if _, e := g.OpenClient(context.Background(), p, false, bridgeRaw(f.f), f.services()); e == nil {
		t.Fatal("two writers")
	}
	if _, e := g.OpenClient(context.Background(), p+"missing", false, bridgeRaw(f.f), f.services()); e == nil {
		t.Fatal("missing journal recreated")
	}
	if c.Close() != nil {
		t.Fatal("close")
	}
	b := bridgeBytes(t, p)
	if os.WriteFile(p, b[:len(b)-1], 0600) != nil {
		t.Fatal("fixture")
	}
	if _, e := g.OpenClient(context.Background(), p, false, bridgeRaw(f.f), f.services()); e == nil {
		t.Fatal("partial journal accepted")
	}
}

func TestClientHandoffDurationAndUncertainTransmission(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			c, f, _, _ := clientSetup(t)
			f.sendDelay = 500
			f.sendFail = fail
			ticket, e := c.Begin(context.Background(), outer(1))
			if (e != nil) != fail {
				t.Fatal("handoff outcome")
			}
			if fail && ticket != nil {
				t.Fatal("failed handoff issued ticket")
			}
			f.sendDelay = 0
			f.sendFail = false
			f.utc = 1700000001499
			f.mono = 1499
			if _, e = c.Begin(context.Background(), outer(2)); e == nil {
				t.Fatal("interval measured before handoff")
			}
			if f.sends != 1 {
				t.Fatal("denied poll handed off")
			}
			f.utc++
			f.mono++
			if _, e = c.Begin(context.Background(), outer(2)); e != nil {
				t.Fatal(e)
			}
			if f.sends != 2 {
				t.Fatal("missing handoff")
			}
		})
	}
}
