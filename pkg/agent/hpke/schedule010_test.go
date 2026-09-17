package hpke

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestSchedule010IndependentVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/schedule010.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []map[string]string `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 3 {
		t.Fatal("missing fixed vectors")
	}
	for _, c := range fixture.Cases {
		t.Run(c["id"], func(t *testing.T) {
			decode := func(k string) []byte {
				b, e := hex.DecodeString(c[k])
				if e != nil {
					t.Fatal(e)
				}
				return b
			}
			exporter, shared, th := decode("exporter_hex"), decode("ss_e2e_hex"), decode("th_hex")
			seed, err := CombineSecrets010(exporter, shared, th)
			if err != nil || !bytes.Equal(seed, decode("seed_hex")) {
				t.Fatal("seed differs from independent vector", err)
			}
			ack, err := MakeAckTag010(seed, th)
			if err != nil || !bytes.Equal(ack, decode("ack_tag_hex")) || !VerifyAckTag010(seed, th, ack) {
				t.Fatal("ACK differs from independent vector", err)
			}
			for i := range ack {
				changed := bytes.Clone(ack)
				changed[i] ^= 1
				if VerifyAckTag010(seed, th, changed) {
					t.Fatal("changed ACK accepted")
				}
			}
			changed := bytes.Clone(th)
			changed[0] ^= 1
			if VerifyAckTag010(seed, changed, ack) {
				t.Fatal("changed transcript accepted")
			}
			changedSeed, err := CombineSecrets010(exporter, shared, changed)
			if err != nil || bytes.Equal(seed, changedSeed) {
				t.Fatal("transcript did not change seed")
			}
			legacy, err := CombineSecrets(exporter, shared, th)
			if err != nil || bytes.Equal(seed, legacy) {
				t.Fatal("legacy label used")
			}
			// The primitive must not overwrite caller-owned input slices.
			if !bytes.Equal(exporter, decode("exporter_hex")) || !bytes.Equal(shared, decode("ss_e2e_hex")) || !bytes.Equal(th, decode("th_hex")) {
				t.Fatal("input changed")
			}
		})
	}
}

func TestSchedule010RejectsInvalidInputs(t *testing.T) {
	good := bytes.Repeat([]byte{1}, 32)
	for slot := 0; slot < 3; slot++ {
		for _, n := range []int{0, 1, 31, 33, 64} {
			args := [][]byte{good, good, good}
			args[slot] = bytes.Repeat([]byte{1}, n)
			out, err := CombineSecrets010(args[0], args[1], args[2])
			if err == nil || out != nil {
				t.Fatal("invalid combiner length accepted", slot, n)
			}
		}
	}
	if out, err := CombineSecrets010(good, make([]byte, 32), good); err == nil || out != nil {
		t.Fatal("zero DH result accepted")
	}
	for _, n := range []int{0, 1, 31, 33, 64} {
		bad := make([]byte, n)
		if out, err := MakeAckTag010(bad, good); err == nil || out != nil {
			t.Fatal("invalid seed accepted")
		}
		if out, err := MakeAckTag010(good, bad); err == nil || out != nil {
			t.Fatal("invalid hash accepted")
		}
		if VerifyAckTag010(good, good, bad) || VerifyAckTag010(bad, good, good) || VerifyAckTag010(good, bad, good) {
			t.Fatal("invalid ACK inputs accepted")
		}
	}
}
