package guard010

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type clientInternalServices struct {
	input            map[string]json.RawMessage
	public           string
	results          bool
	clocks, expireAt int
}

func (s *clientInternalServices) Commit(context.Context, string, []byte) error { return nil }
func (s *clientInternalServices) Now(context.Context) (int64, error) {
	s.clocks++
	if s.expireAt == s.clocks {
		return 1700000300, nil
	}
	return 1700000000, nil
}
func (s *clientInternalServices) Sample(context.Context) (int64, int64, error) {
	return 1700000000000, 0, nil
}
func (s *clientInternalServices) ActiveKey(context.Context, string, string) (ed25519.PublicKey, error) {
	key := s.public
	if !s.results {
		_ = json.Unmarshal(s.input["public_key_hex"], &key)
	}
	b, e := hex.DecodeString(key)
	return ed25519.PublicKey(b), e
}
func (s *clientInternalServices) Bindings(context.Context, string, string) (string, []byte, []byte, error) {
	var original string
	_ = json.Unmarshal(s.input["original_digest"], &original)
	return original, s.input["approved_policy"], s.input["approved_manifest"], nil
}
func (s *clientInternalServices) Authorize(context.Context, string, string, []byte) error { return nil }
func TestClientStorageFailureAndExpiryBeforeRelease(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported storage")
	}
	for _, capacity := range []bool{false, true} {
		t.Run(map[bool]string{false: "expiry", true: "capacity"}[capacity], func(t *testing.T) {
			b, e := os.ReadFile("testdata/guard-client.json")
			if e != nil {
				t.Fatal(e)
			}
			var v struct {
				Input   map[string]json.RawMessage
				Public  string `json:"public_key_hex"`
				Results map[string]string
			}
			if json.Unmarshal(b, &v) != nil {
				t.Fatal("fixture")
			}
			a := &clientInternalServices{input: v.Input}
			r := &clientInternalServices{public: v.Public, results: true}
			if !capacity {
				r.expireAt = 4
			}
			var intent string
			_ = json.Unmarshal(v.Input["envelope_hex"], &intent)
			raw, _ := hex.DecodeString(intent)
			result, _ := hex.DecodeString(v.Results["completed"])
			p := filepath.Join(t.TempDir(), "journal")
			c, e := OpenClient(context.Background(), p, true, raw, ClientServices{IntentAuthority: a, Policy: a, ResultAuthority: r, Clock: a, Sender: a, ExpectedIssuer: "did:sage:web:agents.example.com:alice", ExpectedRecipient: "did:sage:web:agents.example.com:executor"})
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = c.Close() }()
			ticket, e := c.Begin(context.Background(), "00000000-0000-4000-8000-000000000001")
			if e != nil {
				t.Fatal(e)
			}
			if capacity {
				c.rows = 1023
			} // Offline capacity injection; no production bypass API.
			d, e := c.Accept(context.Background(), ticket, result)
			if e == nil || d != nil {
				t.Fatal("invalid delivery released")
			}
			if capacity {
				if !c.failed || len(c.terminal) != 0 {
					t.Fatal("unpersisted terminal")
				}
			} else {
				if len(c.terminal) == 0 {
					t.Fatal("durable consumed marker lost")
				}
				if c.Close() != nil {
					t.Fatal("close")
				}
				reopened, e := OpenClient(context.Background(), p, false, raw, ClientServices{IntentAuthority: a, Policy: a, ResultAuthority: r, Clock: a, Sender: a, ExpectedIssuer: "did:sage:web:agents.example.com:alice", ExpectedRecipient: "did:sage:web:agents.example.com:executor"})
				if e != nil {
					t.Fatal(e)
				}
				defer func() { _ = reopened.Close() }()
				if _, e = reopened.Begin(context.Background(), "00000000-0000-4000-8000-000000000002"); e == nil {
					t.Fatal("failed delivery caused redelivery")
				}
			}
		})
	}
}
