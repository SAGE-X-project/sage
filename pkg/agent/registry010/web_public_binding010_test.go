package registry010

import (
	"context"
	"errors"
	"net/netip"
	"path/filepath"
	"testing"
)

func TestObserveWebRegistryJournalRejectsUnboundSource010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writes.log")
	authority := &webTestAuthority010{actor: "operator"}
	journal, err := OpenWebRegistryWriteJournal010(path, proofDID010, "other-origin", authority, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = journal.Close() }()
	destination := netip.MustParseAddrPort("127.0.0.1:443")
	observe := func() error {
		_, observationErr := ObserveWebRegistryJournal010(context.Background(), journal,
			[]string{"https://agents.example.com"}, destination, []netip.AddrPort{destination}, nil, 100)
		return observationErr
	}
	if !errors.Is(observe(), ErrUnreachable) {
		t.Fatal("empty journal acquired public authority")
	}
	created := proofFixture010(t)
	created["state"] = "created"
	if err := ApplyWebRegistryWrite010(context.Background(), journal, "other-origin", proofDID010,
		webJournalEnvelopeAt010(t, created, 100), 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(observe(), ErrUnreachable) {
		t.Fatal("unbound source reached public fetch")
	}
}
