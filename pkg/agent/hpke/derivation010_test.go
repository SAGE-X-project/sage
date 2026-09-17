package hpke

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type derivationFixture010 struct {
	Cases []struct {
		ID        string            `json:"id"`
		Operation string            `json:"operation"`
		Input     map[string]string `json:"input"`
		Expected  map[string]string `json:"expected"`
	} `json:"cases"`
}

func fixture010(t *testing.T) derivationFixture010 {
	t.Helper()
	b, e := os.ReadFile("testdata/derivation010.json")
	if e != nil {
		t.Fatal(e)
	}
	var f derivationFixture010
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	return f
}
func decode010(t *testing.T, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(s)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestDerivation010IndependentVectors(t *testing.T) {
	f := fixture010(t)
	if len(f.Cases) != 60 {
		t.Fatal("missing vectors")
	}
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			d := func(k string) []byte { return decode010(t, c.Input[k]) }
			var out map[string]string
			var err error
			if c.Operation == "domains" {
				var v Domains010
				v, err = BuildDomains010(d("binding_hex"))
				if err == nil {
					out = map[string]string{"binding_hex": hex.EncodeToString(v.Binding), "info_hex": hex.EncodeToString(v.Info), "export_context_hex": hex.EncodeToString(v.ExportContext)}
				}
			} else {
				var v *Derivation010
				v, err = DeriveResponder010(d("initiation_hex"), d("kem_private_hex"), d("e2e_private_hex"), c.Input["kid"])
				if err == nil {
					defer zeroBytes(v.Seed)
					out = map[string]string{"transcript_hex": hex.EncodeToString(v.Transcript), "th_hex": hex.EncodeToString(v.TH), "seed_hex": hex.EncodeToString(v.Seed), "ack_tag_hex": hex.EncodeToString(v.AckTag), "sid": v.SID}
				}
			}
			if c.Expected == nil {
				if err == nil || out != nil {
					t.Fatal("invalid input accepted")
				}
			} else if err != nil || !reflect.DeepEqual(out, c.Expected) {
				t.Fatal("independent vector mismatch", err)
			}
		})
	}
}
func TestDerivation010FreshExchangeAndConsumption(t *testing.T) {
	f := fixture010(t)
	b := decode010(t, f.Cases[0].Input["binding_hex"])
	kem, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for n := 0; n < 4; n++ {
		s, init, e := StartInitiator010(b, kem.PublicKey().Bytes())
		if e != nil {
			t.Fatal(e)
		}
		defer s.Close()
		var m map[string]string
		if e = json.Unmarshal(init, &m); e != nil {
			t.Fatal(e)
		}
		for _, key := range []string{m["enc"], m["ephC"]} {
			if seen[key] {
				t.Fatal("ephemeral key reused")
			}
			seen[key] = true
		}
		r, e := RespondFresh010(init, kem.Bytes())
		if e != nil {
			t.Fatal(e)
		}
		defer zeroBytes(r.Seed)
		l, e := s.Derive(r.Transcript)
		if e != nil {
			t.Fatal(e)
		}
		defer zeroBytes(l.Seed)
		if !bytes.Equal(l.Seed, r.Seed) || !bytes.Equal(l.TH, r.TH) || l.SID != r.SID || !VerifyAckTag010(l.Seed, l.TH, r.AckTag) {
			t.Fatal("peers disagree")
		}
		if _, e = s.Derive(r.Transcript); e == nil {
			t.Fatal("material reused")
		}
		if len(s.exporter) != 0 || len(s.private) != 0 {
			t.Fatal("material retained")
		}
	}
}
func TestDerivation010RejectsChangedTranscript(t *testing.T) {
	f := fixture010(t)
	b := decode010(t, f.Cases[0].Input["binding_hex"])
	kem, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	for _, field := range append(append([]string{}, bindingFields010...), "task", "enc", "ephC", "ephS", "kid", "extra") {
		t.Run(field, func(t *testing.T) {
			s, init, e := StartInitiator010(b, kem.PublicKey().Bytes())
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			r, e := RespondFresh010(init, kem.Bytes())
			if e != nil {
				t.Fatal(e)
			}
			defer zeroBytes(r.Seed)
			var m map[string]string
			if e = json.Unmarshal(r.Transcript, &m); e != nil {
				t.Fatal(e)
			}
			m[field] = "invalid"
			raw, e := json.Marshal(m)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.Derive(raw); e == nil {
				t.Fatal("changed transcript accepted")
			}
			if _, e = s.Derive(r.Transcript); e == nil {
				t.Fatal("failed state reused")
			}
		})
	}
	for _, bad := range [][]byte{make([]byte, 32), append([]byte{1}, make([]byte, 31)...), make([]byte, 31)} {
		if _, _, e := StartInitiator010(b, bad); e == nil {
			t.Fatal("invalid KEM key accepted")
		}
	}
	s, _, e := StartInitiator010(b, kem.PublicKey().Bytes())
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	if _, e = s.Derive(nil); e == nil {
		t.Fatal("closed material accepted")
	}
}
