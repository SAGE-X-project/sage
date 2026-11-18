// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package secret

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const material = "0123456789abcdef-SECRET"

// leaks returns the forms in which the material could appear in output:
// raw, lower and upper hex, and the space-separated decimals of %d or of a
// byte slice printed by fmt.
func leaks() []string {
	h := fmt.Sprintf("%x", material)
	parts := make([]string, len(material))
	for i := 0; i < len(material); i++ {
		parts[i] = strconv.Itoa(int(material[i]))
	}
	dec := strings.Join(parts, " ") // the form fmt prints a []byte in
	return []string{material, "SECRET", h, strings.ToUpper(h), dec, dec[:20]}
}

type exported struct {
	Key  Bytes
	Keys []Bytes
	Ptr  *Bytes
	Map  map[string]Bytes
}

type unexported struct {
	name string
	key  Bytes
}

type nested struct {
	Key   Bytes
	Inner unexported
	inner *unexported
}

func TestNothingPrintsTheSecret(t *testing.T) {
	s := From([]byte(material))
	values := []any{
		s, &s,
		exported{Key: s, Keys: []Bytes{s}, Ptr: &s, Map: map[string]Bytes{"k": s}},
		unexported{name: "n", key: s},
		&unexported{name: "n", key: s},
		nested{Key: s, Inner: unexported{key: s}, inner: &unexported{key: s}},
		[]unexported{{key: s}},
		map[string]unexported{"k": {key: s}},
	}
	verbs := []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%t", "%c", "%e", "%U", "%b", "%o", "%p", "%T"}
	var outputs []string
	for _, v := range values {
		for _, verb := range verbs {
			outputs = append(outputs, fmt.Sprintf(verb, v))
		}
		outputs = append(outputs, fmt.Sprint(v), fmt.Sprintln(v))
	}
	j, err := json.Marshal(exported{Key: s, Keys: []Bytes{s}, Ptr: &s, Map: map[string]Bytes{"k": s}})
	if err != nil {
		t.Fatal(err)
	}
	outputs = append(outputs, string(j))
	txt, err := s.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	outputs = append(outputs, string(txt))
	var logBuf bytes.Buffer
	for _, h := range []slog.Handler{slog.NewJSONHandler(&logBuf, nil), slog.NewTextHandler(&logBuf, nil)} {
		for _, v := range values {
			slog.New(h).Info("x", "v", v, slog.Group("g", "k", s))
		}
	}
	outputs = append(outputs, logBuf.String())
	for _, out := range outputs {
		for _, leak := range leaks() {
			if strings.Contains(out, leak) {
				t.Fatalf("output leaks the secret as %q: %q", leak, out)
			}
		}
	}
	for _, verb := range verbs[:len(verbs)-2] { // every verb but %p and %T
		if got := fmt.Sprintf(verb, s); got != Redacted {
			t.Errorf("Sprintf(%q, secret) = %q, want %q", verb, got, Redacted)
		}
	}
	if !strings.Contains(string(j), Redacted) || !strings.Contains(logBuf.String(), Redacted) {
		t.Fatalf("json or log lacks %s: %s %s", Redacted, j, logBuf.String())
	}
	if err := gob.NewEncoder(&bytes.Buffer{}).Encode(exported{Key: s}); err == nil {
		t.Fatal("gob encoded a secret")
	}
}

func TestDestroyZeroesTheBorrowedSlice(t *testing.T) {
	s := From([]byte(material))
	copyOf := s
	var borrowed []byte
	if err := s.Use(func(b []byte) error {
		borrowed = b // kept only to observe the erasure
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	copyOf.Destroy()
	if !bytes.Equal(borrowed, make([]byte, len(material))) {
		t.Fatalf("backing array not zeroed: %q", borrowed)
	}
	if err := s.Use(func([]byte) error { return nil }); !errors.Is(err, ErrDestroyed) {
		t.Fatalf("Use after Destroy: %v", err)
	}
	if !s.Destroyed() || s.Len() != 0 || s.Equal(s) {
		t.Fatal("destroyed secret still looks alive")
	}
	s.Destroy() // twice is fine
	var zero Bytes
	zero.Destroy()
	if err := zero.Use(func([]byte) error { return nil }); !errors.Is(err, ErrDestroyed) || !zero.Destroyed() {
		t.Fatal("zero value must be destroyed")
	}
}

func TestFromCopies(t *testing.T) {
	in := []byte(material)
	s := From(in)
	clear(in)
	_ = s.Use(func(b []byte) error {
		if string(b) != material {
			t.Fatal("From did not copy")
		}
		return nil
	})
}

func TestFromEmptyIsDestroyed(t *testing.T) {
	for _, b := range [][]byte{nil, {}} {
		s := From(b)
		if !s.Destroyed() || s.Use(func([]byte) error { return nil }) == nil {
			t.Fatal("an empty secret must not be usable")
		}
	}
}

func TestDestroyWaitsForUse(t *testing.T) {
	s := From([]byte(material))
	inUse := make(chan struct{})
	release := make(chan struct{})
	seen := make(chan string, 1)
	go func() {
		_ = s.Use(func(b []byte) error {
			close(inUse)
			<-release
			seen <- string(b)
			return nil
		})
	}()
	<-inUse
	destroyed := make(chan struct{})
	go func() { s.Destroy(); close(destroyed) }()
	select {
	case <-destroyed:
		t.Fatal("Destroy returned while a Use was running")
	case <-time.After(50 * time.Millisecond):
	}
	if !s.Destroyed() || s.Len() != 0 {
		t.Fatal("Len and Destroyed must report the pending destroy without blocking")
	}
	close(release)
	<-destroyed
	if got := <-seen; got != material {
		t.Fatalf("Use saw %q, want the intact secret", got)
	}
}

func TestEqual(t *testing.T) {
	a, b, c := From([]byte("abc")), From([]byte("abc")), From([]byte("abd"))
	if !a.Equal(b) || !a.Equal(a) || a.Equal(c) || a.Equal(From([]byte("ab"))) || a.Equal(Bytes{}) {
		t.Fatal("Equal is wrong")
	}
	b.Destroy()
	if a.Equal(b) || b.Equal(a) {
		t.Fatal("destroyed secret compared equal")
	}
}

type fixedReader byte

func (f fixedReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(f)
	}
	return len(p), nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

func TestRandom(t *testing.T) {
	s, err := Random(fixedReader(7), 32)
	if err != nil || s.Len() != 32 {
		t.Fatalf("Random: %v %d", err, s.Len())
	}
	if _, err := Random(failingReader{}, 32); err == nil {
		t.Fatal("reader failure ignored")
	}
	if _, err := Random(fixedReader(1), 0); err == nil {
		t.Fatal("zero length accepted")
	}
}

func TestConcurrentUseEqualAndDestroy(t *testing.T) {
	a, b := From([]byte(material)), From([]byte(material))
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); _ = a.Equal(b) }()
		go func() { defer wg.Done(); _ = b.Equal(a) }()
		go func() { defer wg.Done(); _ = a.Use(func([]byte) error { return nil }) }()
	}
	wg.Add(1)
	go func() { defer wg.Done(); b.Destroy() }()
	wg.Wait()
	if !b.Destroyed() {
		t.Fatal("not destroyed")
	}
}
