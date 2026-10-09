package guard010

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

func clientHostBounds() MCPClientHostBounds {
	b := publicBounds()
	return MCPClientHostBounds{Clients: b.Clients, Owners: b.Owners, Client: b.Client, Tick: b.Tick}
}

// An initiator-only host completes a root call against a full receiver host
// without opening any gate, ledger, executor, policy or result signer.
func TestMCPClientHostRootExchange(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	intent, capture := publicInputs(t, f)
	server, e := OpenMCPHost(filepath.Join(t.TempDir(), "gate"), true, setupBob, publicServices(f), publicBounds())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { closePublic(t, server) })
	caller, e := OpenMCPClientHost(publicServices(f).Clock, clientHostBounds())
	if e != nil {
		t.Fatal(e)
	}
	if caller.host.gate != nil {
		t.Fatal("client host opened an admission gate")
	}
	a, b, endpointClock := setupEndpoints(t)
	tcp, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		done <- server.Serve(context.Background(), tcp, 1, publicConfig(false), &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return b, nil }, prepare: func(context.Context) error { return nil }, handle: func(_ context.Context, c *MCPConnection) error {
			for {
				if e := c.ServeOne(); e != nil {
					return e
				}
			}
		}})
	}()
	conn, e := net.DialTimeout("tcp", tcp.Addr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	var terminal *ClientDelivery
	client := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return a, nil }, handle: func(_ context.Context, c *MCPConnection) error {
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
	if e = caller.Connect(context.Background(), conn, publicConfig(true), client); e != nil {
		t.Fatal(e)
	}
	if terminal == nil || !terminal.FirstTerminal() || string(terminal.Output()) != `{"ok":true}` || f.executor.effects.Load() != 1 {
		t.Fatal("client host delivery or effect count")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = caller.Close(ctx); e != nil {
		t.Fatal(e)
	}
	if e = caller.Close(ctx); e != nil {
		t.Fatal("second close", e)
	}
	closePublic(t, server)
	if e = <-done; !errors.Is(e, context.Canceled) {
		t.Fatal("listener did not terminate", e)
	}
}

func TestMCPClientHostRefusesReceiverRoles(t *testing.T) {
	f := newAdmissionFixture(t, 2)
	caller, e := OpenMCPClientHost(publicServices(f).Clock, clientHostBounds())
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = caller.Close(ctx)
	}()
	tcp, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	handler := &publicHandler{endpoint: func(context.Context) (*hpke.CompletionEndpoint010, error) { return nil, ErrInvalid }, prepare: func(context.Context) error { return nil }, handle: func(context.Context, *MCPConnection) error { return nil }}
	if e = caller.Serve(context.Background(), tcp, 1, publicConfig(false), handler); e == nil {
		t.Fatal("client host served")
	}
	left, right := net.Pipe()
	defer func() { _ = right.Close() }()
	if e = caller.Connect(context.Background(), left, publicConfig(false), handler); e == nil {
		t.Fatal("client host accepted a responder connection")
	}
	clock := publicServices(f).Clock
	for name, b := range map[string]MCPClientHostBounds{
		"no-clients": {Owners: 1, Client: time.Second, Tick: time.Millisecond},
		"no-owners":  {Clients: 1, Client: time.Second, Tick: time.Millisecond},
		"slow-tick":  {Clients: 1, Owners: 1, Client: time.Second, Tick: time.Second},
		"sub-ms":     {Clients: 1, Owners: 1, Client: time.Second + time.Microsecond, Tick: time.Millisecond},
	} {
		if h, e := OpenMCPClientHost(clock, b); e == nil || h != nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if h, e := OpenMCPClientHost(nil, clientHostBounds()); e == nil || h != nil {
		t.Fatal("nil clock accepted")
	}
}
