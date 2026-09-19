package guard010_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type rpcCase struct {
	ID, Version   string
	Wire          string `json:"wire_hex"`
	Parse, Accept bool
}
type rpcSuite struct {
	ID                  string
	Requests, Responses []rpcCase
}

func rpcVectors(t *testing.T) rpcSuite {
	t.Helper()
	raw, e := os.ReadFile("testdata/guard-rpc.json")
	if e != nil {
		t.Fatal(e)
	}
	var v rpcSuite
	if json.Unmarshal(raw, &v) != nil || len(v.Requests) != 27 || len(v.Responses) != 22 {
		t.Fatal("fixture")
	}
	return v
}
func TestRPCRequestBoundary(t *testing.T) {
	v := rpcVectors(t)
	ctx := context.Background()
	for _, q := range v.Requests {
		t.Run(q.ID, func(t *testing.T) {
			gate, f, s, _ := openGate(t)
			raw, _ := hex.DecodeString(q.Wire)
			_, err := g.ParseMCPRequest(q.Version, v.ID, raw)
			if (err == nil) != q.Parse {
				t.Fatal("parse", err)
			}
			ep, err := g.NewMCPEndpoint(q.Version, gate)
			if err != nil {
				if q.Version == g.MCPVersion {
					t.Fatal(err)
				}
				return
			}
			receipt, err := ep.Dispatch(ctx, v.ID, raw)
			if (err == nil) != q.Accept {
				t.Fatal("dispatch", err)
			}
			if q.Accept && (!receipt.Committed() || s.commits != 1) {
				t.Fatal("missing commitment")
			}
			if !q.Accept && s.commits != 0 {
				t.Fatal("denied request reached tool")
			}
			valid, _ := g.MCPRequest(g.MCPVersion, v.ID, bridgeRaw(f.guardFixture))
			if _, err = ep.Dispatch(ctx, v.ID, valid); err == nil {
				t.Fatal("attempt ID reused")
			}
		})
	}
}
func TestRPCResponseConsumesInvocation(t *testing.T) {
	v := rpcVectors(t)
	ctx := context.Background()
	for _, q := range v.Responses {
		t.Run(q.ID, func(t *testing.T) {
			c, _, _, s := clientSetup(t)
			ticket, e := c.Begin(ctx, v.ID)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := hex.DecodeString(q.Wire)
			d, e := c.AcceptMCPResponse(ctx, ticket, q.Version, raw)
			if (e == nil) != q.Accept {
				t.Fatal("response", e)
			}
			if q.Accept && (!d.FirstTerminal() || d.Status() != "completed" || !bytes.Equal(d.Output(), []byte(`{"value":"ok"}`))) {
				t.Fatal("output")
			}
			if _, e = c.Accept(ctx, ticket, clientRaw(s, "completed")); e == nil {
				t.Fatal("ticket reused")
			}
		})
	}
}
func TestRPCSingleDispatchAndReply(t *testing.T) {
	gate, f, s, _ := openGate(t)
	ep, _ := g.NewMCPEndpoint(g.MCPVersion, gate)
	ctx := context.Background()
	id := outer(101)
	raw, _ := g.MCPRequest(g.MCPVersion, id, bridgeRaw(f.guardFixture))
	var wg sync.WaitGroup
	results := make(chan *g.MCPReceipt, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, e := ep.Dispatch(ctx, id, raw); e == nil {
				results <- r
			}
		}()
	}
	wg.Wait()
	close(results)
	if len(results) != 1 || s.commits != 1 {
		t.Fatal("duplicate dispatch")
	}
	receipt := <-results
	copyReceipt := *receipt
	other, _ := g.NewMCPEndpoint(g.MCPVersion, gate)
	if _, e := other.Reply(ctx, receipt, signer()); e == nil {
		t.Fatal("foreign receipt")
	}
	pending, e := ep.Reply(ctx, receipt, signer())
	if e != nil {
		t.Fatal(e)
	}
	var p map[string]any
	_ = json.Unmarshal(pending, &p)
	if p["id"] != id {
		t.Fatal("reply ID")
	}
	if _, e = ep.Reply(ctx, &copyReceipt, signer()); e == nil {
		t.Fatal("copied reply permit")
	}
	if e = gate.Finish(ctx, s.observed.Completion(), []byte(`{"value":"ok"}`), signer()); e != nil {
		t.Fatal(e)
	}
	id = outer(102)
	raw, _ = g.MCPRequest(g.MCPVersion, id, bridgeRaw(f.guardFixture))
	r, e := ep.Dispatch(ctx, id, raw)
	if e != nil || r.Committed() || r.State() != "COMPLETED" || s.commits != 1 {
		t.Fatal("poll redispatched", e)
	}
	ep.Close()
	if _, e = ep.Reply(ctx, r, signer()); e == nil {
		t.Fatal("closed reply")
	}
}
func TestRPCSessionCapacity(t *testing.T) {
	gate, f, s, _ := openGate(t)
	ep, _ := g.NewMCPEndpoint(g.MCPVersion, gate)
	ctx := context.Background()
	for n := 0; n < 1024; n++ {
		if _, e := ep.Dispatch(ctx, outer(n+1000), []byte(`{}`)); e == nil {
			t.Fatal("invalid accepted")
		}
	}
	raw, _ := g.MCPRequest(g.MCPVersion, outer(3000), bridgeRaw(f.guardFixture))
	if _, e := ep.Dispatch(ctx, outer(3000), raw); e == nil || s.commits != 0 {
		t.Fatal("capacity bypass")
	}
}

type rpcWire struct {
	f   *clientFixture
	raw []byte
}

func (w *rpcWire) Send(ctx context.Context, id string, raw []byte) error {
	w.raw = append([]byte(nil), raw...)
	intent, e := g.ParseMCPRequest(g.MCPVersion, id, raw)
	if e != nil {
		return e
	}
	return w.f.Commit(ctx, id, intent)
}
func TestRPCProtectedClientHandoffAndSchema(t *testing.T) {
	v := clientVectors(t)
	f := &clientFixture{f: v.Input, utc: 1700000000000, clockOK: true, resultActive: true, pub: v.Public}
	w := &rpcWire{f: f}
	sender, e := g.NewMCPClientSender(g.MCPVersion, w)
	if e != nil {
		t.Fatal(e)
	}
	services := f.services()
	services.Sender = sender
	c, e := g.OpenClient(context.Background(), filepath.Join(t.TempDir(), "journal"), true, bridgeRaw(f.f), services)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = c.Close() }()
	if _, e = c.Begin(context.Background(), outer(101)); e != nil {
		t.Fatal(e)
	}
	expected, _ := g.MCPRequest(g.MCPVersion, outer(101), bridgeRaw(f.f))
	if !bytes.Equal(w.raw, expected) || f.sends != 1 {
		t.Fatal("actual handoff")
	}
	if _, e = g.NewMCPClientSender("2024-11-05", w); e == nil {
		t.Fatal("setup")
	}
	raw, e := g.MCPTool(g.MCPVersion)
	if e != nil {
		t.Fatal(e)
	}
	var tool map[string]any
	_ = json.Unmarshal(raw, &tool)
	schema := tool["inputSchema"].(map[string]any)
	if tool["name"] != "sage_secure_call" || schema["additionalProperties"] != false || len(schema["properties"].(map[string]any)) != 1 {
		t.Fatal("schema")
	}
	raw[0] = '!'
	fresh, _ := g.MCPTool(g.MCPVersion)
	if fresh[0] != '{' {
		t.Fatal("mutable schema")
	}
}

func TestRPCZeroEndpointDenies(t *testing.T) {
	var e g.MCPEndpoint
	if _, err := e.Dispatch(context.Background(), outer(1), []byte(`{}`)); err == nil {
		t.Fatal("zero endpoint")
	}
}

func TestRPCLargeSignedOutput(t *testing.T) {
	for _, text := range []string{strings.Repeat("<>&", 75000), strings.Repeat("\u2028\u2029", 100000)} {
		gate, _, sink, _ := openGate(t)
		ep, e := g.NewMCPEndpoint(g.MCPVersion, gate)
		if e != nil {
			t.Fatal(e)
		}
		client, _, _, _ := clientSetup(t)
		ctx := context.Background()
		ticket, e := client.Begin(ctx, outer(101))
		if e != nil {
			t.Fatal(e)
		}
		request, e := g.MCPRequest(g.MCPVersion, ticket.ID(), ticket.Intent())
		if e != nil {
			t.Fatal(e)
		}
		receipt, e := ep.Dispatch(ctx, ticket.ID(), request)
		if e != nil {
			t.Fatal(e)
		}
		output := []byte(`{"text":"` + text + `"}`)
		if e = gate.Finish(ctx, sink.observed.Completion(), output, signer()); e != nil {
			t.Fatal("large signed completion", e)
		}
		response, e := ep.Reply(ctx, receipt, signer())
		if e != nil {
			t.Fatal("large signed response", e)
		}
		delivery, e := client.AcceptMCPResponse(ctx, ticket, g.MCPVersion, response)
		if e != nil || !delivery.FirstTerminal() || !bytes.Equal(delivery.Output(), output) {
			t.Fatal("large authenticated delivery", e)
		}
	}
}
