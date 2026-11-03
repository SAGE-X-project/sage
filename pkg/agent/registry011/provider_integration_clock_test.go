package registry011

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestProviderIntegrationOwnedClockIsFiniteAndNonserializable(t *testing.T) {
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	clock, err := newProviderOwnedClock(policy, 2, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	first, err := clock.sample()
	if err != nil || first.Epoch == "" || first.ClockID != "fixture-clock" || first.UncertaintyMS != 200 || first.MonotonicMS < 0 {
		t.Fatal("owned first sample", err)
	}
	second, err := clock.sample()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(first), providerClockDiagnostic(second)) || clock.read != 2 {
		t.Fatal("owned sample order", err)
	}
	if _, err := clock.sample(); err == nil || clock.read != 2 {
		t.Fatal("exhausted clock replenished its allowance")
	}
	var decoded providerOwnedClock
	if json.Unmarshal([]byte(`{}`), &decoded) == nil {
		t.Fatal("JSON recreated an owned clock")
	}
	var absent *providerOwnedClock
	if _, err := absent.sample(); err == nil {
		t.Fatal("nil clock sampled")
	}
}

func TestProviderIntegrationClockConfigurationRefusals(t *testing.T) {
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		policy       *ProviderPolicyCandidate
		reads        uint64
		base, window int64
	}{
		{nil, 1, 1, 1}, {&ProviderPolicyCandidate{}, 1, 1, 1},
		{policy, 0, 1, 1}, {policy, 1<<20 + 1, 1, 1},
		{policy, 1, 0, 1}, {policy, 1, 1, 0},
		{policy, 1, 1, math.MaxInt64}, {policy, 1, math.MaxInt64, 1},
		{policy, 1, 1000, 1}, {policy, 1, 1, 1001},
	} {
		if value, err := newProviderOwnedClock(fixture.policy, fixture.reads, fixture.base, fixture.window); err == nil || value != nil {
			t.Fatal("unsupported owned clock configuration")
		}
	}
}

func TestProviderIntegrationClockInvalidStateNeverProducesASample(t *testing.T) {
	policy, _ := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	for _, name := range []string{"owner", "failed", "zero-origin", "future-origin", "epoch", "id", "uncertainty", "window", "backward-sample"} {
		t.Run(name, func(t *testing.T) {
			clock, err := newProviderOwnedClock(policy, 4, 100, 100)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "owner":
				clock.owner = &providerOwnedClock{}
			case "failed":
				clock.failed = true
			case "zero-origin":
				clock.origin = time.Time{}
			case "future-origin":
				clock.origin = time.Now().Add(time.Hour)
			case "epoch":
				clock.epoch = "invalid"
			case "id":
				clock.id = ""
			case "uncertainty":
				clock.uncertainty = 0
			case "window":
				clock.readWindow = 0
			case "backward-sample":
				clock.read = 1
				clock.last = ClockSample{ClockID: clock.id, Epoch: clock.epoch, WallMS: time.Now().Add(time.Hour).UnixMilli(), MonotonicMS: 0, UncertaintyMS: 200}
			}
			if _, err := clock.sample(); err == nil {
				t.Fatal("invalid state acquired clock authority")
			}
			if name == "future-origin" || name == "backward-sample" {
				want := providerOwnedClockReadBracketRefusal
				if name == "backward-sample" {
					want = providerOwnedClockSampleOrderRefusal
				}
				if !clock.failed || clock.nativeRefusal != want {
					t.Fatal("clock uncertainty or its original refusal was not sticky", clock.nativeRefusal)
				}
				reads := clock.read
				if _, err := clock.sample(); err == nil || clock.read != reads || clock.nativeRefusal != want {
					t.Fatal("a later refused call replaced the first native failure or sampled again")
				}
			} else if clock.nativeRefusal != 0 {
				t.Fatal("pre-native shape refusal claimed a native read failure")
			}
		})
	}
}
