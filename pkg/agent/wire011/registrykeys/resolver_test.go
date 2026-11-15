package registrykeys

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

const (
	registryID = "eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6"
	agentDID   = "did:sage:" + registryID + ":payment-agent-2"
)

type clock struct{}

func (clock) Now() (time.Time, error) { return time.Unix(1_800_000_000, 0), nil }

type source struct {
	present bool
	record  []byte
	err     error
	asked   []string
}

func (s *source) Record(_ context.Context, agentID string) (bool, []byte, error) {
	s.asked = append(s.asked, agentID)
	return s.present, s.record, s.err
}

// The fixture is the canonical record the chain Registry returned on Anvil
// after the lifecycle scenario revoked the leaked sig-1 (sage-contracts #39).
func fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/record-after-revoke.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestResolveAcceptedSigningKey(t *testing.T) {
	src := &source{present: true, record: fixture(t)}
	r := &Resolver{Source: src, RegistryID: registryID, Clock: clock{}}
	key, err := r.ResolveSigningKey(context.Background(), agentDID+"#sig-3")
	if err != nil || len(key) != 32 {
		t.Fatalf("sig-3 refused: %v", err)
	}
	if len(src.asked) != 1 || src.asked[0] != "payment-agent-2" {
		t.Fatalf("asked for %v", src.asked)
	}
}

func TestResolveRefusals(t *testing.T) {
	rec := fixture(t)
	cases := []struct {
		name, kid, reason string
		src               *source
	}{
		{"revoked leaked key", agentDID + "#sig-1", "revoked", &source{present: true, record: rec}},
		{"KEM key used for signing", agentDID + "#kem-1", "not a signing key", &source{present: true, record: rec}},
		{"unknown key", agentDID + "#sig-9", "no key", &source{present: true, record: rec}},
		{"other Registry", "did:sage:eip155:1:0x0000000000000000000000000000000000000001:payment-agent-2#sig-3", "not a key of this Registry", &source{}},
		{"bare DID", agentDID, "not a key of this Registry", &source{}},
		{"absent record", agentDID + "#sig-3", "no record", &source{}},
		{"source failure", agentDID + "#sig-3", "read record", &source{err: errors.New("rpc down")}},
		{"deactivated", agentDID + "#sig-3", "record is deactivated", &source{present: true,
			record: []byte(strings.Replace(string(rec), `"state":"active"`, `"state":"deactivated"`, 1))}},
		{"record for another DID", agentDID + "#sig-3", "another DID", &source{present: true,
			record: []byte(strings.Replace(string(rec), `payment-agent-2","keys"`, `other","keys"`, 1))}},
		{"expired key", agentDID + "#sig-3", "expired", &source{present: true,
			record: []byte(strings.Replace(string(rec), `"name":"sig-3"`, `"expires":1700000000,"name":"sig-3"`, 1))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Resolver{Source: tc.src, RegistryID: registryID, Clock: clock{}}
			_, err := r.ResolveSigningKey(context.Background(), tc.kid)
			if err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("got %v, want %q", err, tc.reason)
			}
		})
	}
	var nilResolver *Resolver
	if _, err := nilResolver.ResolveSigningKey(context.Background(), agentDID+"#sig-3"); err == nil {
		t.Fatal("nil resolver resolved")
	}
	if _, _, err := (rpcSource{}).Record(context.Background(), "x"); err == nil {
		t.Fatal("source without RPC read")
	}
	if _, err := SelectSigningKey([]byte("{"), agentDID, "sig-3", 0); err == nil {
		t.Fatal("malformed record accepted")
	}
	bad := strings.Replace(string(rec), "NJyxNB1YscqEjhKU02_ijkJpmkEnlQqHAuSDT3RY2pk", "AAAA", 1)
	if _, err := SelectSigningKey([]byte(bad), agentDID, "sig-3", 0); err == nil {
		t.Fatal("short key accepted")
	}
}

type badClock struct{}

func (badClock) Now() (time.Time, error) { return time.Time{}, errors.New("no time") }

func TestResolverConstructionAndClock(t *testing.T) {
	network := registry011.SepoliaNetwork()
	r := NewRegistryResolver(nil, network, clock{})
	if r.RegistryID != network.RegistryID() {
		t.Fatal("registry id not taken from the network")
	}
	if _, err := r.ResolveSigningKey(context.Background(), "did:sage:"+network.RegistryID()+":agent#sig-1"); err == nil ||
		!strings.Contains(err.Error(), "read record") {
		t.Fatalf("resolver without RPC: %v", err)
	}
	withoutTime := &Resolver{Source: &source{present: true, record: fixture(t)}, RegistryID: registryID, Clock: badClock{}}
	if _, err := withoutTime.ResolveSigningKey(context.Background(), agentDID+"#sig-3"); err == nil || !strings.Contains(err.Error(), "clock") {
		t.Fatalf("resolver without trusted time: %v", err)
	}
}
