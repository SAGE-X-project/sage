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

type providerSourceClockPolicyTestReader struct {
	fields      providerOwnedClockPolicyFields
	secondID    string
	secondError error
	calls       int
}

func (r *providerSourceClockPolicyTestReader) readOwnedClockPolicy(*ProviderPolicyCandidate) (providerOwnedClockPolicyFields, error) {
	r.calls++
	fields := r.fields
	if r.calls > 1 {
		if r.secondError != nil {
			return providerOwnedClockPolicyFields{}, r.secondError
		}
		if r.secondID != "" {
			fields.id = r.secondID
		}
	}
	return fields, nil
}

func TestProviderHistorySourceCompilerUsesItsPolicyReaderThroughSelection(t *testing.T) {
	for _, name := range []string{"original", "changed-id", "lost-reader"} {
		t.Run(name, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			fields, err := (providerOwnedClockGoPolicyReader{}).readOwnedClockPolicy(config.policy)
			if err != nil {
				t.Fatal(err)
			}
			reader := &providerSourceClockPolicyTestReader{fields: fields}
			switch name {
			case "changed-id":
				reader.secondID = fields.id + "-changed"
			case "lost-reader":
				reader.secondError = context.Canceled
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			compiler, err := newProviderHistorySourceCompiler(ctx)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := compiler.compileWithClockPolicy(config, reader, 1<<20, 1, 1)
			if reader.calls != 2 || !compiler.used.Load() {
				t.Fatal("source selection bypassed its reader or refunded the attempt", reader.calls)
			}
			if name == "original" {
				if err != nil || plan == nil || plan.clock == nil || plan.clock.origin != compiler.origin ||
					plan.compilerDeadline != compiler.deadline || plan.source.clock != providerHistoryClockSelectionOf(plan.clock) || plan.clock.read != 0 {
					t.Fatal("reader changed the original plan or clock", err)
				}
				numeric, ok := providerHistoryCompileSourcePlan(plan.config, plan.source.clock)
				if !ok || numeric.source != plan.source {
					t.Fatal("reader changed the original source costs")
				}
			} else if plan != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatal("changed reader published a partial plan or fell back to Go", err)
			}
			if retry, err := compiler.compileWithClockPolicy(config, reader, 1<<20, 1, 1); retry != nil || !errors.Is(err, ErrConfiguration) || reader.calls != 2 {
				t.Fatal("source selection reused the compiler or reader", err)
			}
		})
	}
}

func TestProviderHistorySourcePolicyReaderDoesNotReparseOrFallBack(t *testing.T) {
	config := historySelectedSourceConfig(t)
	clock := historySelectedSourceClock(config)
	policy := *config.policy
	// This isolated reader unit observes which representation was consumed.
	// The production native reader independently binds the validated raw bytes.
	policy.raw = "reader-only-unit-data"
	reader := &providerClockPolicyTestReader{fields: providerOwnedClockPolicyFields{id: clock.id}}
	if !providerHistorySourceClockMatchesPolicy(&policy, clock, reader) || reader.calls != 1 {
		t.Fatal("selected policy reader was replaced by a Go parser")
	}
	if providerHistorySourceClockMatchesPolicy(&policy, clock, nil) {
		t.Fatal("Go compatibility matching accepted non-JSON policy data")
	}
	reader.err = context.Canceled
	if providerHistorySourceClockMatchesPolicy(config.policy, clock, reader) || reader.calls != 2 {
		t.Fatal("failed reader fell back to valid Go policy data")
	}
	reader.err = nil
	clock.epoch = "invalid"
	if providerHistorySourceClockMatchesPolicy(config.policy, clock, reader) || reader.calls != 2 {
		t.Fatal("invalid source clock entered its reader")
	}
}
