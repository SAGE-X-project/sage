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

package clock

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSystem(t *testing.T) {
	before := time.Now()
	got, err := System().Now()
	if err != nil || got.Before(before) || time.Since(got) > time.Second {
		t.Fatalf("%v %v", got, err)
	}
}

func TestFixed(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFixed(start)
	if got, err := c.Now(); err != nil || !got.Equal(start) {
		t.Fatalf("%v %v", got, err)
	}
	c.Advance(90 * time.Second)
	if got, _ := c.Now(); !got.Equal(start.Add(90 * time.Second)) {
		t.Fatalf("advance: %v", got)
	}
	c.Set(start)
	if got, _ := c.Now(); !got.Equal(start) {
		t.Fatalf("set: %v", got)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); c.Advance(time.Millisecond) }()
		go func() { defer wg.Done(); _, _ = c.Now() }()
	}
	wg.Wait()
	if got, _ := c.Now(); !got.Equal(start.Add(16 * time.Millisecond)) {
		t.Fatalf("concurrent advance: %v", got)
	}
}

func TestFailing(t *testing.T) {
	boom := errors.New("clock down")
	if _, err := Failing(boom).Now(); !errors.Is(err, boom) {
		t.Fatalf("%v", err)
	}
}

func TestRead(t *testing.T) {
	now := time.Unix(1788609600, 0)
	if got, err := Read(NewFixed(now)); err != nil || !got.Equal(now) {
		t.Fatalf("%v %v", got, err)
	}
	boom := errors.New("boom")
	for name, c := range map[string]Clock{"nil": nil, "zero": NewFixed(time.Time{}), "failing": Failing(boom)} {
		if _, err := Read(c); err == nil {
			t.Fatalf("%s: no error", name)
		}
	}
	if _, err := Read(Failing(boom)); !errors.Is(err, boom) {
		t.Fatal("the clock's error is not returned")
	}
	if _, err := Read(NewFixed(time.Time{})); !errors.Is(err, ErrNoTime) {
		t.Fatal("zero time")
	}
	if _, err := Read(System()); err != nil {
		t.Fatal(err)
	}
}
