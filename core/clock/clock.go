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

// Package clock is the time source of the SAGE core. Every check that needs
// time takes a Clock, so tests fix the clock instead of disabling a check
// (DP-4), and a clock that fails makes every such check fail.
package clock

import (
	"errors"
	"sync"
	"time"
)

// ErrNoTime is returned by Read for a nil clock and for a clock that
// returns the zero time, which dates nothing.
var ErrNoTime = errors.New("clock: no time")

// Clock returns the current time, or an error when it cannot.
type Clock interface {
	Now() (time.Time, error)
}

// Read returns c.Now() and fails for a nil clock, a clock error and the zero
// time, so every check that reads the time fails closed the same way.
func Read(c Clock) (time.Time, error) {
	if c == nil {
		return time.Time{}, ErrNoTime
	}
	t, err := c.Now()
	if err != nil {
		return time.Time{}, err
	}
	if t.IsZero() {
		return time.Time{}, ErrNoTime
	}
	return t, nil
}

// System returns the system clock.
func System() Clock { return systemClock{} }

type systemClock struct{}

func (systemClock) Now() (time.Time, error) { return time.Now(), nil }

// Fixed is a clock for tests and vector checks. It returns the time it was
// set to until Set or Advance changes it. It is safe for concurrent use.
type Fixed struct {
	mu sync.Mutex
	t  time.Time
}

// NewFixed returns a clock fixed at t.
func NewFixed(t time.Time) *Fixed { return &Fixed{t: t} }

// Now returns the fixed time.
func (f *Fixed) Now() (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t, nil
}

// Set fixes the clock at t.
func (f *Fixed) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = t
}

// Advance moves the clock forward by d.
func (f *Fixed) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

// Failing returns a clock whose Now always fails with err, for tests of the
// fail-closed paths.
func Failing(err error) Clock { return failing{err: err} }

type failing struct{ err error }

func (f failing) Now() (time.Time, error) { return time.Time{}, f.err }
