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

package core

import (
	"errors"
	"testing"
	"time"
)

func TestDefaultLimitsAreTheSpecValues(t *testing.T) {
	l := DefaultLimits()
	want := Limits{
		MaxBodyBytes:           1 << 20,
		MaxDigestBodyBytes:     16 << 20,
		MaxSignatureFieldBytes: 8 << 10,
		MaxCoveredComponents:   32,
		MaxHeaderBytes:         64 << 10,
		MaxHPKEInitBytes:       4 << 10,
		MaxRecordBytes:         16<<20 + 36,
		MaxAADBytes:            4 << 10,
		MaxAge:                 300 * time.Second,
		MaxClockSkew:           60 * time.Second,
		MaxResolutionAge:       60 * time.Second,
	}
	if l != want {
		t.Fatalf("got %+v\nwant %+v", l, want)
	}
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLimitsMayBeLoweredNeverRaised(t *testing.T) {
	lowered := DefaultLimits()
	lowered.MaxBodyBytes = 1024
	lowered.MaxAge = time.Second
	lowered.MaxClockSkew = 0
	if err := lowered.Validate(); err != nil {
		t.Fatalf("lowered limits: %v", err)
	}
	for name, mut := range map[string]func(*Limits){
		"body above 1 MiB":         func(l *Limits) { l.MaxBodyBytes++ },
		"digest body above 16 MiB": func(l *Limits) { l.MaxDigestBodyBytes++ },
		"signature field":          func(l *Limits) { l.MaxSignatureFieldBytes++ },
		"components":               func(l *Limits) { l.MaxCoveredComponents++ },
		"header":                   func(l *Limits) { l.MaxHeaderBytes++ },
		"hpke init":                func(l *Limits) { l.MaxHPKEInitBytes++ },
		"record":                   func(l *Limits) { l.MaxRecordBytes++ },
		"aad":                      func(l *Limits) { l.MaxAADBytes++ },
		"max age above 300 s":      func(l *Limits) { l.MaxAge = 301 * time.Second },
		"max age zero":             func(l *Limits) { l.MaxAge = 0 },
		"skew above 300 s":         func(l *Limits) { l.MaxClockSkew = 300*time.Second + 1 },
		"negative skew":            func(l *Limits) { l.MaxClockSkew = -1 },
		"resolution age above":     func(l *Limits) { l.MaxResolutionAge = 301 * time.Second },
		"resolution age zero":      func(l *Limits) { l.MaxResolutionAge = 0 },
		"zero body":                func(l *Limits) { l.MaxBodyBytes = 0 },
		"negative aad":             func(l *Limits) { l.MaxAADBytes = -1 },
		"components below E03-10":  func(l *Limits) { l.MaxCoveredComponents = MinCoveredComponents - 1 },
	} {
		l := DefaultLimits()
		mut(&l)
		if err := l.Validate(); !errors.Is(err, ErrLimits) {
			t.Errorf("%s: %v", name, err)
		}
		if _, err := NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)), WithLimits(l)); !errors.Is(err, ErrLimits) {
			t.Errorf("%s: NewSettings accepted it: %v", name, err)
		}
	}
}

func TestSettingsCarryLimits(t *testing.T) {
	s, err := NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)))
	if err != nil {
		t.Fatal(err)
	}
	if s.Limits() != DefaultLimits() {
		t.Fatal("settings without WithLimits must hold the spec values")
	}
	l := DefaultLimits()
	l.MaxAge = time.Minute
	s, err = NewSettings(WithSpecVersion(string(V0_9_0_Draft1_1)), WithLimits(l))
	if err != nil || s.Limits().MaxAge != time.Minute {
		t.Fatalf("%v %v", s.Limits(), err)
	}
	if (Settings{}).Limits() != (Limits{}) {
		t.Fatal("a zero Settings has zero limits, which every component rejects")
	}
}

func TestSpecBoundariesAccepted(t *testing.T) {
	l := DefaultLimits()
	l.MaxClockSkew = SpecMaxClockSkew
	l.MaxResolutionAge = SpecMaxResolutionAge
	l.MaxCoveredComponents = MinCoveredComponents
	if err := l.Validate(); err != nil {
		t.Fatalf("300 s skew, 300 s resolution age and 7 components are within the spec: %v", err)
	}
}
