package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// Sessions of different versions built from the same seed and transcript are
// unrelated: different identifiers, and neither opens the other's records.
func TestRecordSession011IsSeparateFrom010(t *testing.T) {
	seed, th := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	c11, err := NewRecordSession011(seed, th, true)
	if err != nil {
		t.Fatal(err)
	}
	s11, _ := NewRecordSession011(seed, th, false)
	s10, _ := NewRecordSession010(seed, th, false)
	sum := sha256.Sum256(append([]byte("sage-session|0.11.0"), th...))
	if c11.ID() != base64.RawURLEncoding.EncodeToString(sum[:16]) || c11.ID() == s10.ID() {
		t.Fatalf("0.11.0 session id %q", c11.ID())
	}
	wire, err := c11.Seal([]byte("hello"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s10.Open(wire, []byte("aad")); err == nil {
		t.Fatal("0.10.0 session opened a 0.11.0 record")
	}
	got, err := s11.Open(wire, []byte("aad"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("0.11.0 record not opened: %v", err)
	}
	if _, err := NewRecordSession011(seed[:31], th, true); err == nil {
		t.Fatal("short seed accepted")
	}
}
