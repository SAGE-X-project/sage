package gatesource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

const (
	registryID = "eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6"
	did        = "did:sage:" + registryID + ":payment-agent-2"
	blockHex   = "0x" + "ab12000000000000000000000000000000000000000000000000000000000000"
)

// The fixture is the canonical record the chain Registry returned on Anvil
// after the lifecycle scenario revoked the leaked sig-1 (sage-contracts #39).
func fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/record.json")
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(raw)
}

type clock struct {
	mu   sync.Mutex
	mono int64
	err  error
}

func (c *clock) Now() (registry010.Stamp, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return registry010.Stamp{MonoMS: c.mono, Unix: 1_800_000_000 + c.mono/1000}, c.err
}

func (c *clock) advance(ms int64) { c.mu.Lock(); c.mono += ms; c.mu.Unlock() }

type reader struct {
	read  Read
	err   error
	agent string
	delay func()
}

func (r *reader) Read(_ context.Context, agentID string) (Read, error) {
	r.agent = agentID
	if r.delay != nil {
		r.delay()
	}
	return r.read, r.err
}

type store struct{}

func (store) Advance(registry010.Scope, uint64, string, bool) error { return nil }

func newGate(t *testing.T, rd *reader, c *clock) (*registry010.Gate, *Source) {
	t.Helper()
	src, err := newWithReader("anvil-dev", registryID, "dev", rd, c, limits)
	if err != nil {
		t.Fatal(err)
	}
	g, err := registry010.NewGate(src.Config(), src, c, store{})
	if err != nil {
		t.Fatal(err)
	}
	return g, src
}

var limits = Limits{MaxFinalizedAge: 600}

// header is a finalized header 10 seconds older than the test clock's start.
func header(number uint64, hash string) registry011.FinalizedHeader {
	return registry011.FinalizedHeader{Hash: hash, Number: number, Timestamp: 1_800_000_000 - 10}
}

func present(raw []byte) *reader {
	return &reader{read: Read{Header: header(90, blockHex), Present: true, Record: raw}}
}

func TestGateSelectsKeysOfA011Record(t *testing.T) {
	c := &clock{mono: 1000}
	rd := present(fixture(t))
	g, _ := newGate(t, rd, c)
	p, err := g.Select(context.Background(), did, did+"#sig-2", true)
	if err != nil {
		t.Fatalf("select sig-2 with KEM: %v", err)
	}
	if rd.agent != "payment-agent-2" || p.Signing().Name != "sig-2" || p.KEM() == nil || p.KEM().Name != "kem-1" {
		t.Fatalf("selected %+v kem %+v", p.Signing(), p.KEM())
	}
	if len(p.Signing().Material) != 64 || strings.ToLower(p.Signing().Material) != p.Signing().Material {
		t.Fatalf("material is not lowercase hex32: %q", p.Signing().Material)
	}
	if err := g.CheckPinned(context.Background(), p); err != nil {
		t.Fatalf("recheck: %v", err)
	}
}

func TestGateRefusesTheRevokedKey(t *testing.T) {
	g, _ := newGate(t, present(fixture(t)), &clock{mono: 1000})
	if _, err := g.Select(context.Background(), did, did+"#sig-1", false); !errors.Is(err, registry010.ErrRejected) {
		t.Fatalf("revoked sig-1 selected: %v", err)
	}
}

func TestSnapshotCarriesBlockAndDigest(t *testing.T) {
	raw := fixture(t)
	c := &clock{mono: 1000}
	_, src := newGate(t, present(raw), c)
	s, err := src.Read(context.Background(), did)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if s.Digest != hex.EncodeToString(sum[:]) || s.BlockHash != blockHex[2:] || s.KeysBlockHash != s.BlockHash ||
		s.Version != "6" || s.State != "active" || !s.Ready || !s.Validated || !s.Finalized || s.AcquiredMS != 1000 {
		t.Fatalf("snapshot %+v", s)
	}
}

func TestMalformedRecordsAreRefused(t *testing.T) {
	raw := fixture(t)
	cases := map[string][]byte{
		"not canonical":    append([]byte(" "), raw...),
		"extra member":     bytes.Replace(raw, []byte(`"version":"6"}`), []byte(`"version":"6","x":1}`), 1),
		"missing member":   bytes.Replace(raw, []byte(`"state":"active",`), nil, 1),
		"key extra member": bytes.Replace(raw, []byte(`"name":"kem-1",`), []byte(`"name":"kem-1","note":"x",`), 1),
		"bad key encoding": bytes.Replace(raw, []byte(`"key":"D_3H`), []byte(`"key":"D_3H=`), 1),
		"another DID":      bytes.ReplaceAll(raw, []byte(":payment-agent-2"), []byte(":payment-agent-3")),
		"duplicate member": bytes.Replace(raw, []byte(`"version":"6"}`), []byte(`"version":"6","version":"6"}`), 1),
		"member case":      bytes.Replace(raw, []byte(`"id":`), []byte(`"ID":`), 1),
		"proof tampered":   bytes.Replace(raw, []byte(`"value":"pbZT`), []byte(`"value":"qbZT`), 1),
		"proof extra":      bytes.Replace(raw, []byte(`"proof":{"signer"`), []byte(`"proof":{"extra":1,"signer"`), 1),
		"services null":    bytes.Replace(raw, []byte(`"services":[`), []byte(`"services":null,"x":[`), 1),
		"controller":       bytes.Replace(raw, []byte(`"controller":"0x70997970c51812dc3a010c7d01b50e0d17dc79c8"`), []byte(`"controller":"ANY THING"`), 1),
	}
	for name, mutated := range cases {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(mutated, raw) {
				t.Fatal("mutation did not apply")
			}
			g, _ := newGate(t, present(mutated), &clock{mono: 1000})
			if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err == nil {
				t.Fatal("malformed record accepted")
			}
		})
	}
}

func TestReadFailuresAreRefused(t *testing.T) {
	raw := fixture(t)
	for name, rd := range map[string]*reader{
		"absent":       {read: Read{Header: header(90, blockHex)}},
		"rpc error":    {err: errors.New("connection refused")},
		"no header":    {read: Read{Present: true, Record: raw}},
		"short hash":   {read: Read{Header: header(90, "0x12"), Present: true, Record: raw}},
		"bare hash":    {read: Read{Header: header(90, blockHex[2:]+"00"), Present: true, Record: raw}},
		"upper hash":   {read: Read{Header: header(90, strings.ToUpper(blockHex)), Present: true, Record: raw}},
		"zero hash":    {read: Read{Header: header(90, "0x"+strings.Repeat("0", 64)), Present: true, Record: raw}},
		"stale header": {read: Read{Header: registry011.FinalizedHeader{Hash: blockHex, Number: 90, Timestamp: 1_800_000_000 - 601}, Present: true, Record: raw}},
		"future block": {read: Read{Header: registry011.FinalizedHeader{Hash: blockHex, Number: 90, Timestamp: 1_800_000_000 + 60}, Present: true, Record: raw}},
		"huge stamp":   {read: Read{Header: registry011.FinalizedHeader{Hash: blockHex, Number: 90, Timestamp: 1 << 63}, Present: true, Record: raw}},
		"huge stamp+5": {read: Read{Header: registry011.FinalizedHeader{Hash: blockHex, Number: 90, Timestamp: 1<<63 + 5}, Present: true, Record: raw}},
	} {
		t.Run(name, func(t *testing.T) {
			g, _ := newGate(t, rd, &clock{mono: 1000})
			if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err == nil {
				t.Fatal("failed read accepted")
			}
		})
	}
}

// The acquisition time is sampled before the read, so a read that takes
// longer than the Gate's 5-second window is stale, however recent its block.
func TestSlowReadIsStale(t *testing.T) {
	c := &clock{mono: 1000}
	rd := present(fixture(t))
	g, _ := newGate(t, rd, c)
	rd.delay = func() { c.advance(5001) }
	if _, err := g.Select(context.Background(), did, did+"#sig-2", false); !errors.Is(err, registry010.ErrStale) {
		t.Fatalf("slow read accepted: %v", err)
	}
	rd.delay = func() { c.advance(4000) }
	if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err != nil {
		t.Fatalf("read within the window refused: %v", err)
	}
}

func TestFinalizedBlockNeverGoesBack(t *testing.T) {
	c := &clock{mono: 1000}
	rd := present(fixture(t))
	g, _ := newGate(t, rd, c)
	if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err != nil {
		t.Fatal(err)
	}
	for name, h := range map[string]registry011.FinalizedHeader{
		"lower block":         header(89, blockHex),
		"same height, forked": header(90, "0x"+strings.Repeat("cd", 32)),
	} {
		rd.read.Header = h
		if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	rd.read.Header = header(91, "0x"+strings.Repeat("ef", 32))
	if _, err := g.Select(context.Background(), did, did+"#sig-2", false); err != nil {
		t.Fatalf("next block refused: %v", err)
	}
}

func TestOtherRegistriesAndBadConfiguration(t *testing.T) {
	c := &clock{mono: 1000}
	rd := present(fixture(t))
	_, src := newGate(t, rd, c)
	for _, other := range []string{"did:sage:eip155:1:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6:payment-agent-2", "did:sage:" + registryID + ":", "did:sage:" + registryID + ":a:b"} {
		if _, err := src.Read(context.Background(), other); err == nil {
			t.Fatalf("%s read", other)
		}
	}
	c.err = errors.New("clock untrusted")
	if _, err := src.Read(context.Background(), did); err == nil {
		t.Fatal("clock failure hidden")
	}
	if _, err := newWithReader("", registryID, "dev", rd, c, limits); err == nil {
		t.Fatal("incomplete configuration accepted")
	}
	if _, err := newWithReader("anvil-dev", registryID, "dev", rd, c, Limits{}); err == nil {
		t.Fatal("missing finalized-age limit accepted")
	}
	if _, err := NewDevelopment("sepolia", nil, registry011.SepoliaNetwork(), c, limits); err == nil || !strings.Contains(err.Error(), "qualifying") {
		t.Fatalf("qualifying profile accepted: %v", err)
	}
}
