package registry010

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestWebRegistryJournalPublicEnvelope010(t *testing.T) {
	did, source := proofDID010, "https://agents.example.com"
	authority := &webTestAuthority010{actor: "operator"}
	journal, err := OpenWebRegistryWriteJournal010(filepath.Join(t.TempDir(), "writes.log"),
		did, source, authority, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.PublicEnvelope010(did, source, 100); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("empty journal served a record: %v", err)
	}
	created := proofFixture010(t)
	created["state"] = "created"
	if err := ApplyWebRegistryWrite010(context.Background(), journal, source, did,
		webJournalEnvelopeAt010(t, created, 100), 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []struct{ did, source string }{
		{did, "other-origin"}, {"did:sage:web:other.example.com:billing-bot", source},
	} {
		if _, err := journal.PublicEnvelope010(invalid.did, invalid.source, 101); err == nil {
			t.Fatal("wrong public binding received a record")
		}
	}
	response, err := journal.PublicEnvelope010(did, source, 101)
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		Record  json.RawMessage `json:"record"`
		Issued  int64           `json:"issued"`
		Expires int64           `json:"expires"`
	}
	if json.Unmarshal(response, &value) != nil || value.Issued != 101 || value.Expires != 106 ||
		CheckWebRegistryProofs010(response, did, 101) != nil {
		t.Fatal("public response was not fresh and valid")
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.PublicEnvelope010(did, source, 102); !errors.Is(err, ErrUnreachable) {
		t.Fatal("closed journal served a record")
	}
}
