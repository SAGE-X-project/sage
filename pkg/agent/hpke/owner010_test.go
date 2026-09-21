package hpke

import (
	"context"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"testing"
	"time"
)

func TestNonHTTPOwnerTransferInvalidatesAliases(t *testing.T) {
	a, b, _ := recordPair010(t, 0)
	copied := *a
	owned, err := a.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owned.Close)
	if !owned.Initiator() || owned.CreatedMonoMS() != 0 {
		t.Fatal("lost origin/role")
	}
	for _, old := range []*AuthenticatedCompletion010{a, &copied} {
		if _, err := old.TakeNonHTTP(); err == nil {
			t.Fatal("reclaimed session")
		}
		if _, err := old.SealRequest(context.Background(), []byte("old"), 30); err == nil {
			t.Fatal("old alias sent")
		}
		if old.Check(context.Background()) == nil {
			t.Fatal("old alias checked")
		}
		if old.BindHTTP("https://example.com/mcp") == nil {
			t.Fatal("old alias rebound")
		}
		if _, _, err := old.Participants(); err == nil {
			t.Fatal("old alias participants")
		}
		old.Close()
	}
	wire, err := owned.SealRequest(context.Background(), []byte("owned"), 30)
	if err != nil {
		t.Fatal("old close erased moved keys", err)
	}
	plain, err := b.OpenRequest(context.Background(), wire)
	if err != nil || string(plain) != "owned" {
		t.Fatal("owned exchange", err)
	}
	if _, err := b.TakeNonHTTP(); err == nil {
		t.Fatal("used session transferred")
	}
}
func TestNonHTTPOwnerOriginalCreationDeadline(t *testing.T) {
	a, b, c := recordPair010(t, 0)
	c.mono = 29000
	owned, err := b.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owned.Close)
	if owned.Initiator() || owned.CreatedMonoMS() != 0 {
		t.Fatal("provisional clock restarted")
	}
	now, err := owned.LocalNow()
	if err != nil || now != 29*time.Second {
		t.Fatal(now, err)
	}
	a.Close()
	c.mono = 3600000
	if _, err := owned.LocalNow(); err == nil {
		t.Fatal("session lifetime")
	}
}
func TestNonHTTPOwnerHTTPAndEndpointClosure(t *testing.T) {
	a, b, _ := recordPair010(t, 0)
	if err := a.BindHTTP("https://example.com/mcp"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TakeNonHTTP(); err == nil {
		t.Fatal("HTTP transfer")
	}
	owned, err := b.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owned.Close)
	b.endpoint.Close()
	if _, err := owned.LocalNow(); err == nil {
		t.Fatal("endpoint closure ignored")
	}
}

func TestNonHTTPOwnerRejectsStaleCopiesAfterUseOrBinding(t *testing.T) {
	for _, kind := range []string{"send", "receive", "http"} {
		t.Run(kind, func(t *testing.T) {
			a, b, _ := recordPair010(t, 0)
			target := a
			if kind == "receive" {
				target = b
			}
			copy := *target
			switch kind {
			case "send":
				if _, err := a.SealRequest(context.Background(), []byte("used"), 30); err != nil {
					t.Fatal(err)
				}
			case "receive":
				wire, err := a.SealRequest(context.Background(), []byte("used"), 30)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = b.OpenRequest(context.Background(), wire); err != nil {
					t.Fatal(err)
				}
			case "http":
				if err := a.BindHTTP("https://example.com/mcp"); err != nil {
					t.Fatal(err)
				}
			}
			if owner, err := copy.TakeNonHTTP(); err == nil {
				owner.Close()
				t.Fatal("stale copy transferred shared used state")
			}
		})
	}
}
func TestNonHTTPOwnerRejectsLocalClockRollback(t *testing.T) {
	a, _, c := recordPair010(t, 0)
	o, err := a.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	c.utc = 101
	c.mono = 1000
	if _, err = o.LocalNow(); err != nil {
		t.Fatal(err)
	}
	c.utc = 100
	c.mono = 1001
	if _, err = o.LocalNow(); err == nil {
		t.Fatal("wall rollback")
	}
	c.utc = 102
	c.mono = 999
	if _, err = o.LocalNow(); err == nil {
		t.Fatal("monotonic rollback")
	}
}

type ownerSequenceClock struct {
	control *completionControl
	samples []int64
}

func (c *ownerSequenceClock) Now() (registry010.Stamp, error) {
	if len(c.samples) == 0 {
		return registry010.Stamp{}, errCompletion010
	}
	c.control.mono = c.samples[0]
	c.samples = c.samples[1:]
	return c.control.Now()
}
func TestNonHTTPOwnerRejectsRollbackAfterObservation(t *testing.T) {
	a, _, c := recordPair010(t, 0)
	o, err := a.TakeNonHTTP()
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	c.mono = 1000
	if _, err = o.LocalNow(); err != nil {
		t.Fatal(err)
	}
	a.endpoint.clock = &ownerSequenceClock{control: c, samples: []int64{2000, 3000, 2500}}
	at, err := o.Observe(context.Background())
	if err != nil || at != 2*time.Second {
		t.Fatal(at, err)
	}
	if _, err = o.LocalNow(); err == nil {
		t.Fatal("rollback after validation end accepted")
	}
	if o.session.lifetime.active.Load() != 0 {
		t.Fatal("observation extended idle lifetime")
	}
}
