package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRecord010Vectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/session-records-010.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Cases []struct {
			ID        string
			Operation string
			Input     struct {
				Seed      string `json:"seed_hex"`
				TH        string `json:"th_hex"`
				SID       string `json:"sid"`
				Direction string
				Seq       uint64
				Wire      string `json:"record_hex"`
				AAD       string `json:"caller_aad_hex"`
				Plaintext struct {
					Byte   byte
					Length int
				}
			}
			Expected struct {
				Verdict string
				Output  struct {
					Key       string `json:"key_hex"`
					Plaintext string `json:"plaintext_hex"`
					Hash      string `json:"record_sha256"`
					Size      int    `json:"record_bytes"`
				}
			}
		}
	}
	if err = json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	if len(suite.Cases) != 55 {
		t.Fatal("fixture inventory changed")
	}
	decode := func(s string) []byte {
		b, e := hex.DecodeString(s)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	for _, c := range suite.Cases {
		t.Run(c.ID, func(t *testing.T) {
			i := c.Input
			sending := c.Operation != "sage.session.record.open"
			s, e := NewRecordSession010(decode(i.Seed), decode(i.TH), (i.Direction == "c2s") == sending)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			var out []byte
			switch c.Operation {
			case "sage.session.key":
				d := byte(0)
				if i.Direction == "s2c" {
					d = 1
				}
				k, err := s.key(d, i.Seq)
				e = err
				out = k[:]
				if hex.EncodeToString(out) != c.Expected.Output.Key {
					t.Fatal("key differs")
				}
			case "sage.session.record.open":
				out, e = s.Open(decode(i.Wire), decode(i.AAD))
				if e == nil && hex.EncodeToString(out) != c.Expected.Output.Plaintext {
					t.Fatal("plaintext differs")
				}
			case "sage.session.record.seal":
				out, e = s.Seal(bytes.Repeat([]byte{i.Plaintext.Byte}, i.Plaintext.Length), decode(i.AAD))
				if e == nil {
					h := sha256.Sum256(out)
					if hex.EncodeToString(h[:]) != c.Expected.Output.Hash || len(out) != c.Expected.Output.Size {
						t.Fatal("wire differs")
					}
				}
			default:
				t.Fatal("unknown operation")
			}
			if (e == nil) != (c.Expected.Verdict == "ACCEPT") {
				t.Fatalf("unexpected verdict: %v", e)
			}
		})
	}
}

// A close may overlap an authenticated receive. The receive either completes
// before close or is denied; a later receive can never revive the session.
func TestRecord010CloseOpenOrder(t *testing.T) {
	seed, th := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	for attempt := 0; attempt < 32; attempt++ {
		sender, err := NewRecordSession010(seed, th, true)
		if err != nil {
			t.Fatal(err)
		}
		receiver, err := NewRecordSession010(seed, th, false)
		if err != nil {
			t.Fatal(err)
		}
		first, err := sender.Seal([]byte("first"), nil)
		if err != nil {
			t.Fatal(err)
		}
		second, err := sender.Seal([]byte("second"), nil)
		if err != nil {
			t.Fatal(err)
		}

		start := make(chan struct{})
		var done sync.WaitGroup
		var received []byte
		var receiveErr error
		done.Add(2)
		go func() {
			defer done.Done()
			<-start
			received, receiveErr = receiver.Open(first, nil)
		}()
		go func() {
			defer done.Done()
			<-start
			receiver.Close()
		}()
		close(start)
		done.Wait()
		if receiveErr == nil && !bytes.Equal(received, []byte("first")) {
			t.Fatalf("attempt %d: wrong authenticated plaintext", attempt)
		}
		if _, err := receiver.Open(second, nil); err == nil {
			t.Fatalf("attempt %d: closed session accepted a later record", attempt)
		}
		if receiver.seed != ([32]byte{}) || !receiver.closed {
			t.Fatalf("attempt %d: close did not retire the session", attempt)
		}
		sender.Close()
	}
}
func TestRecord010State(t *testing.T) {
	seed, th := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	sender, _ := NewRecordSession010(seed, th, true)
	receiver, _ := NewRecordSession010(seed, th, false)
	defer sender.Close()
	defer receiver.Close()
	if sender.ID() != receiver.ID() || len(sender.ID()) != 22 {
		t.Fatal("invalid identity")
	}
	first, _ := sender.Seal([]byte("first"), nil)
	second, _ := sender.Seal([]byte("second"), nil)
	bad := bytes.Clone(first)
	bad[len(bad)-1] ^= 1
	before := receiver.active
	if _, e := receiver.Open(bad, nil); e == nil || receiver.active != before {
		t.Fatal("invalid traffic changed state")
	}
	for _, w := range [][]byte{second, first} {
		if _, e := receiver.Open(w, nil); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := receiver.Open(first, nil); e == nil {
		t.Fatal("replay accepted")
	}
	receiver.Close()
	if _, e := receiver.Open(second, nil); e == nil {
		t.Fatal("closed accepted")
	}
	if receiver.seed != ([32]byte{}) {
		t.Fatal("retained seed")
	}
	for _, absolute := range []bool{false, true} {
		s, _ := NewRecordSession010(seed, th, true)
		if absolute {
			s.created = time.Now().Add(-time.Hour)
		} else {
			s.active = time.Now().Add(-10 * time.Minute)
		}
		if _, e := s.Seal(nil, nil); e == nil || !s.closed {
			t.Fatal("expiry failed")
		}
	}
	capped, _ := NewRecordSession010(seed, th, true)
	capped.next = 999
	if _, e := capped.Seal(nil, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := capped.Seal(nil, nil); e == nil || !capped.closed {
		t.Fatal("cap failed")
	}
	if _, e := NewRecordSession010(seed[:31], th, true); e == nil {
		t.Fatal("short seed accepted")
	}
	if _, e := NewRecordSession010(seed, th[:31], true); e == nil {
		t.Fatal("short transcript accepted")
	}
}

func TestRecord010RoundTrip(t *testing.T) {
	for _, initiator := range []bool{false, true} {
		a, _ := NewRecordSession010(make([]byte, 32), bytes.Repeat([]byte{9}, 32), initiator)
		b, _ := NewRecordSession010(make([]byte, 32), bytes.Repeat([]byte{9}, 32), !initiator)
		for seq := 0; seq < 1000; seq++ {
			w, e := a.Seal([]byte("message"), []byte("aad"))
			if e != nil {
				t.Fatal(e)
			}
			p, e := b.Open(w, []byte("aad"))
			if e != nil || string(p) != "message" {
				t.Fatalf("seq %d: %v", seq, e)
			}
		}
		if _, e := a.Seal(nil, nil); e == nil {
			t.Fatal("exhausted sender accepted")
		}
		a.Close()
		b.Close()
	}
}
