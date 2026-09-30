package registry010

import (
	"errors"
	"testing"
)

func TestWebRegistryRequestURL010(t *testing.T) {
	did := "did:sage:web:agents.example.com:billing-bot"
	want := "https://agents.example.com/.well-known/sage/agents/billing-bot"
	got, err := WebRegistryRequestURL010(did, []string{"https://other.example.com", "https://agents.example.com"})
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	for _, origin := range []string{"http://agents.example.com", "https://agents.example.com:443", "https://agents.example.com/", "https://other.example.com"} {
		if _, err := WebRegistryRequestURL010(did, []string{origin}); !errors.Is(err, ErrUnreachable) {
			t.Errorf("origin %q: got %v; want unreachable", origin, err)
		}
	}
	for _, invalidDID := range []string{"did:sage:web:127.0.0.1:a", "did:sage:web:Agents.example.com:a", "did:sage:web:agents.example.com:..", "did:sage:chain:x:a"} {
		if _, err := WebRegistryRequestURL010(invalidDID, []string{"https://agents.example.com"}); !errors.Is(err, ErrInvalidRecord010) {
			t.Errorf("DID %q: got %v; want invalid", invalidDID, err)
		}
	}
}

func TestWebRegistryResponsePolicy010(t *testing.T) {
	header := []HeaderField010{{Name: "Content-Type", Value: "application/json"}, {Name: "Cache-Control", Value: "private, NO-STORE"}}
	if err := CheckWebRegistryResponsePolicy010(200, header, nil); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{0, 301, 304, 404} {
		if err := CheckWebRegistryResponsePolicy010(status, header, nil); !errors.Is(err, ErrUnreachable) {
			t.Errorf("status %d: got %v; want unreachable", status, err)
		}
	}
	for _, bad := range [][]HeaderField010{
		{{Name: "Content-Type", Value: "application/json"}},
		{{Name: "Content-Type", Value: "application/json"}, {Name: "Cache-Control", Value: "no-store=1"}},
		{{Name: "Content-Type", Value: "text/plain"}, {Name: "Cache-Control", Value: "no-store"}},
		{{Name: "Content-Type", Value: "application/json"}, {Name: "Content-Encoding", Value: "gzip"}, {Name: "Cache-Control", Value: "no-store"}},
	} {
		if err := CheckWebRegistryResponsePolicy010(200, bad, nil); !errors.Is(err, ErrInvalidRecord010) {
			t.Errorf("headers %+v: got %v; want invalid", bad, err)
		}
	}
	if err := CheckWebRegistryResponsePolicy010(200, header, []HeaderField010{{Name: "Content-Type", Value: "application/json"}}); !errors.Is(err, ErrInvalidRecord010) {
		t.Fatalf("trailer: got %v; want invalid", err)
	}
}
