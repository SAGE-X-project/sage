package registry011

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

type providerClockPolicyTestReader struct {
	fields providerOwnedClockPolicyFields
	err    error
	calls  int
}

func (r *providerClockPolicyTestReader) readOwnedClockPolicy(*ProviderPolicyCandidate) (providerOwnedClockPolicyFields, error) {
	r.calls++
	return r.fields, r.err
}

func TestProviderIntegrationClockPolicyReaderKeepsOriginalValidation(t *testing.T) {
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	origin := time.Now()
	fields, err := (providerOwnedClockGoPolicyReader{}).readOwnedClockPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	reader := &providerClockPolicyTestReader{fields: fields}
	clock, err := newProviderOwnedClockFromPolicyReader(policy, reader, 2, 1, 1, origin)
	if err != nil || clock == nil || reader.calls != 1 || clock.owner != clock || clock.origin != origin ||
		clock.id != fields.id || clock.policy != policy.policy || !lowerHash(clock.epoch) || clock.read != 0 || clock.uncertainty != 2 {
		t.Fatal("shared original clock construction", err)
	}
	for _, fixture := range []struct {
		name   string
		fields providerOwnedClockPolicyFields
	}{
		{"absent-id", providerOwnedClockPolicyFields{maximumUncertainty: fields.maximumUncertainty, maximumWindow: fields.maximumWindow}},
		{"long-id", providerOwnedClockPolicyFields{strings.Repeat("x", providerHistorySourceFixedTextLimit+1), fields.maximumUncertainty, fields.maximumWindow}},
		{"zero-uncertainty", providerOwnedClockPolicyFields{fields.id, 0, fields.maximumWindow}},
		{"zero-window", providerOwnedClockPolicyFields{fields.id, fields.maximumUncertainty, 0}},
		{"uncertainty", providerOwnedClockPolicyFields{fields.id, 1, fields.maximumWindow}},
		{"window", providerOwnedClockPolicyFields{fields.id, fields.maximumUncertainty, 1}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			reader := &providerClockPolicyTestReader{fields: fixture.fields}
			clock, err := newProviderOwnedClockFromPolicyReader(policy, reader, 2, 1, 2, origin)
			if !errors.Is(err, ErrConfiguration) || clock != nil || reader.calls != 1 {
				t.Fatal("invalid reader fields produced a clock", err)
			}
		})
	}
	for index, fixture := range []struct {
		policy       *ProviderPolicyCandidate
		reader       providerOwnedClockPolicyReader
		maximum      uint64
		base, window int64
		origin       time.Time
	}{
		{nil, reader, 1, 1, 1, origin}, {&ProviderPolicyCandidate{}, reader, 1, 1, 1, origin},
		{policy, nil, 1, 1, 1, origin}, {policy, reader, 0, 1, 1, origin},
		{policy, reader, 1<<20 + 1, 1, 1, origin}, {policy, reader, 1, 0, 1, origin},
		{policy, reader, 1, 1, 0, origin}, {policy, reader, 1, 1, math.MaxInt64, origin},
		{policy, reader, 1, math.MaxInt64, 1, origin}, {policy, reader, 1, 1, 1, time.Time{}},
	} {
		before := reader.calls
		clock, err := newProviderOwnedClockFromPolicyReader(fixture.policy, fixture.reader, fixture.maximum, fixture.base, fixture.window, fixture.origin)
		if !errors.Is(err, ErrConfiguration) || clock != nil || reader.calls != before {
			t.Fatalf("invalid envelope %d entered the reader: %v", index, err)
		}
	}
	reader = &providerClockPolicyTestReader{err: context.Canceled}
	if clock, err := newProviderOwnedClockFromPolicyReader(policy, reader, 2, 1, 1, origin); !errors.Is(err, context.Canceled) || clock != nil || reader.calls != 1 {
		t.Fatal("reader failure lost its original cause", err)
	}
}
