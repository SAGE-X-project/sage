package registry011

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
)

func testLimits() ObservationLimits {
	return ObservationLimits{ClockID: "owned-clock", PolicyGeneration: "fixed-test-generation",
		MaxClockUncertaintyMS: 2, MaxAcquisitionMS: 10, MaxObservationAgeMS: 20,
		MaxFinalizedLagMS: 1000, MaxAncestryHeaders: 3}
}

func testChecks(t *testing.T) *ObservationChecks {
	t.Helper()
	c, err := NewObservationChecks(testLimits())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testHeader(number uint64) FinalizedHeader {
	return FinalizedHeader{Hash: fmt.Sprintf("0x%064x", number+1),
		ParentHash: fmt.Sprintf("0x%064x", number), Number: number, Timestamp: 100 + number}
}

func testClock(ms int64) ClockSample {
	return ClockSample{ClockID: "owned-clock", Epoch: "one-process", WallMS: ms, MonotonicMS: ms}
}

func TestObservationConfiguration(t *testing.T) {
	mutations := []struct {
		name string
		edit func(*ObservationLimits)
	}{
		{"clock", func(c *ObservationLimits) { c.ClockID = "" }},
		{"generation", func(c *ObservationLimits) { c.PolicyGeneration = "" }},
		{"uncertainty", func(c *ObservationLimits) { c.MaxClockUncertaintyMS = -1 }},
		{"acquisition-zero", func(c *ObservationLimits) { c.MaxAcquisitionMS = 0 }},
		{"acquisition-negative", func(c *ObservationLimits) { c.MaxAcquisitionMS = -1 }},
		{"age-zero", func(c *ObservationLimits) { c.MaxObservationAgeMS = 0 }},
		{"age-negative", func(c *ObservationLimits) { c.MaxObservationAgeMS = -1 }},
		{"lag-zero", func(c *ObservationLimits) { c.MaxFinalizedLagMS = 0 }},
		{"lag-negative", func(c *ObservationLimits) { c.MaxFinalizedLagMS = -1 }},
		{"ancestry", func(c *ObservationLimits) { c.MaxAncestryHeaders = 0 }},
	}
	for _, tt := range mutations {
		t.Run(tt.name, func(t *testing.T) {
			limits := testLimits()
			tt.edit(&limits)
			if _, err := NewObservationChecks(limits); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("got %v", err)
			}
		})
	}
	limits := testLimits()
	c, err := NewObservationChecks(limits)
	if err != nil {
		t.Fatal(err)
	}
	limits.ClockID = "changed-after-construction"
	limits.MaxAcquisitionMS = 1
	if err := c.CheckReadWindow(testClock(101000), testClock(101010), testHeader(1)); err != nil {
		t.Fatalf("configuration was not copied: %v", err)
	}
	for _, c := range []*ObservationChecks{nil, {}} {
		if !errors.Is(c.CheckReadWindow(ClockSample{}, ClockSample{}, FinalizedHeader{}), ErrConfiguration) ||
			!errors.Is(c.CheckPublication(ClockSample{}, ClockSample{}, FinalizedHeader{}), ErrConfiguration) ||
			!errors.Is(c.CheckLineage(FinalizedHeader{}, FinalizedHeader{}, nil), ErrConfiguration) {
			t.Fatal("unconfigured check accepted")
		}
	}
}

func TestObservationWindows(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ClockSample, *ClockSample, *FinalizedHeader)
		want error
	}{
		{"normal", func(*ClockSample, *ClockSample, *FinalizedHeader) {}, nil},
		{"wrong-start-clock", func(s, _ *ClockSample, _ *FinalizedHeader) { s.ClockID = "other" }, ErrClock},
		{"wrong-end-clock", func(_, e *ClockSample, _ *FinalizedHeader) { e.ClockID = "other" }, ErrClock},
		{"empty-epoch", func(s, _ *ClockSample, _ *FinalizedHeader) { s.Epoch = "" }, ErrClock},
		{"restarted-clock", func(_, e *ClockSample, _ *FinalizedHeader) { e.Epoch = "new-process" }, ErrClock},
		{"negative-wall", func(s, _ *ClockSample, _ *FinalizedHeader) { s.WallMS = -1 }, ErrClock},
		{"negative-monotonic", func(_, e *ClockSample, _ *FinalizedHeader) { e.MonotonicMS = -1 }, ErrClock},
		{"negative-uncertainty", func(_, e *ClockSample, _ *FinalizedHeader) { e.UncertaintyMS = -1 }, ErrClock},
		{"uncertain-clock", func(_, e *ClockSample, _ *FinalizedHeader) { e.UncertaintyMS = 3 }, ErrClock},
		{"wall-rollback", func(_, e *ClockSample, _ *FinalizedHeader) { e.WallMS = 100999 }, ErrClock},
		{"monotonic-rollback", func(_, e *ClockSample, _ *FinalizedHeader) { e.MonotonicMS = 100999 }, ErrClock},
		{"wall-jump", func(_, e *ClockSample, _ *FinalizedHeader) { e.WallMS++ }, ErrClock},
		{"monotonic-jump", func(_, e *ClockSample, _ *FinalizedHeader) { e.MonotonicMS++ }, ErrClock},
		{"bounded-wall-drift", func(s, e *ClockSample, _ *FinalizedHeader) { s.UncertaintyMS = 1; e.WallMS++ }, nil},
		{"bounded-monotonic-drift", func(s, e *ClockSample, _ *FinalizedHeader) { s.UncertaintyMS = 1; e.WallMS-- }, nil},
		{"acquisition-expired", func(_, e *ClockSample, _ *FinalizedHeader) { e.WallMS++; e.MonotonicMS++ }, ErrStale},
		{"hash-length", func(_, _ *ClockSample, h *FinalizedHeader) { h.Hash = "0x1" }, ErrAncestry},
		{"hash-prefix", func(_, _ *ClockSample, h *FinalizedHeader) { h.Hash = "1x" + strings.Repeat("1", 64) }, ErrAncestry},
		{"hash-uppercase", func(_, _ *ClockSample, h *FinalizedHeader) { h.Hash = "0x" + strings.Repeat("A", 64) }, ErrAncestry},
		{"parent-invalid", func(_, _ *ClockSample, h *FinalizedHeader) { h.ParentHash = "bad" }, ErrAncestry},
		{"timestamp-range", func(_, _ *ClockSample, h *FinalizedHeader) { h.Timestamp = 1 << 53 }, ErrAncestry},
		{"header-from-future", func(_, _ *ClockSample, h *FinalizedHeader) { h.Timestamp = 102 }, ErrClock},
		{"stale-header", func(_, _ *ClockSample, h *FinalizedHeader) { h.Timestamp = 100 }, ErrStale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, header := testClock(101000), testClock(101010), testHeader(1)
			tt.edit(&start, &end, &header)
			if err := testChecks(t).CheckReadWindow(start, end, header); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestPublicationKeepsOriginalAcquisitionStart(t *testing.T) {
	c := testChecks(t)
	start, end := testClock(101000), testClock(101020)
	header := testHeader(1)
	if err := c.CheckPublication(start, end, header); err != nil {
		t.Fatal(err)
	}
	end.WallMS++
	end.MonotonicMS++
	if !errors.Is(c.CheckPublication(start, end, header), ErrStale) {
		t.Fatal("expired publication accepted")
	}
	// The initial header-age boundary is inclusive; uncertainty is included.
	start, end = testClock(102000), testClock(102000)
	if err := c.CheckReadWindow(start, end, header); err != nil {
		t.Fatal(err)
	}
	end.UncertaintyMS = 1
	if !errors.Is(c.CheckReadWindow(start, end, header), ErrStale) {
		t.Fatal("uncertainty made an old header fresh")
	}
}

func TestClockArithmeticDoesNotWrap(t *testing.T) {
	limits := testLimits()
	limits.MaxClockUncertaintyMS = math.MaxInt64
	limits.MaxFinalizedLagMS = math.MaxInt64
	c, err := NewObservationChecks(limits)
	if err != nil {
		t.Fatal(err)
	}
	s := ClockSample{ClockID: limits.ClockID, Epoch: "extreme", WallMS: math.MaxInt64,
		MonotonicMS: math.MaxInt64, UncertaintyMS: math.MaxInt64}
	h := testHeader(0)
	h.Timestamp = 1<<53 - 1
	if !errors.Is(c.CheckReadWindow(s, s, h), ErrStale) {
		t.Fatal("large unsigned age wrapped into a fresh result")
	}
	s.UncertaintyMS = 0
	if err := c.CheckReadWindow(s, s, h); err != nil {
		t.Fatalf("valid large boundary rejected: %v", err)
	}
}

func TestFinalizedLineage(t *testing.T) {
	tests := []struct {
		name string
		edit func(*FinalizedHeader, *FinalizedHeader, *[]FinalizedHeader)
		want error
	}{
		{"normal", func(*FinalizedHeader, *FinalizedHeader, *[]FinalizedHeader) {}, nil},
		{"previous-format", func(p, _ *FinalizedHeader, _ *[]FinalizedHeader) { p.Hash = "bad" }, ErrAncestry},
		{"current-format", func(_, c *FinalizedHeader, _ *[]FinalizedHeader) { c.Hash = "bad" }, ErrAncestry},
		{"rollback", func(p, _ *FinalizedHeader, _ *[]FinalizedHeader) { *p = testHeader(4) }, ErrAncestry},
		{"budget", func(_, c *FinalizedHeader, _ *[]FinalizedHeader) { *c = testHeader(5) }, ErrAncestry},
		{"missing-parent", func(_, _ *FinalizedHeader, p *[]FinalizedHeader) { *p = (*p)[:1] }, ErrAncestry},
		{"extra-parent", func(_, _ *FinalizedHeader, p *[]FinalizedHeader) { *p = append(*p, testHeader(0)) }, ErrAncestry},
		{"parent-format", func(_, _ *FinalizedHeader, p *[]FinalizedHeader) { (*p)[0].Hash = "bad" }, ErrAncestry},
		{"parent-hash", func(_, _ *FinalizedHeader, p *[]FinalizedHeader) { (*p)[0].Hash = testHeader(8).Hash }, ErrAncestry},
		{"parent-number", func(_, _ *FinalizedHeader, p *[]FinalizedHeader) { (*p)[0].Number++ }, ErrAncestry},
		{"parent-time-equal", func(_, c *FinalizedHeader, p *[]FinalizedHeader) { (*p)[0].Timestamp = c.Timestamp }, ErrAncestry},
		{"parent-time-future", func(_, c *FinalizedHeader, p *[]FinalizedHeader) { (*p)[0].Timestamp = c.Timestamp + 1 }, ErrAncestry},
		{"repeated-hash", func(_, c *FinalizedHeader, p *[]FinalizedHeader) { c.ParentHash = c.Hash; (*p)[0].Hash = c.Hash }, ErrAncestry},
		{"different-retained-anchor", func(p, _ *FinalizedHeader, _ *[]FinalizedHeader) { p.Timestamp-- }, ErrAncestry},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous, current := testHeader(1), testHeader(3)
			parents := []FinalizedHeader{testHeader(2), previous}
			tt.edit(&previous, &current, &parents)
			if err := testChecks(t).CheckLineage(previous, current, parents); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
	c := testChecks(t)
	base := testHeader(0)
	if err := c.CheckLineage(base, testHeader(3), []FinalizedHeader{testHeader(2), testHeader(1), base}); err != nil {
		t.Fatalf("inclusive work limit or genesis anchor rejected: %v", err)
	}
	if err := c.CheckLineage(base, base, nil); err != nil {
		t.Fatal(err)
	}
	other := base
	other.Hash = testHeader(4).Hash
	if !errors.Is(c.CheckLineage(base, other, nil), ErrAncestry) {
		t.Fatal("same-height fork accepted")
	}
}

func TestOfficialSepoliaHeaderFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/official-sepolia-headers.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Endpoint string
		Headers  []FinalizedHeader
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Endpoint != "https://rpc.sepolia.ethpandaops.io" || len(fixture.Headers) != 4 ||
		fixture.Headers[0].Number != 11976934 || fixture.Headers[3].Number != 11976931 {
		t.Fatal("unexpected historical fixture identity")
	}
	// This only checks historical parent links, not a fresh RPC response,
	// authenticated enrollment, consensus proof or operating Source readiness.
	if err := testChecks(t).CheckLineage(fixture.Headers[3], fixture.Headers[0], fixture.Headers[1:]); err != nil {
		t.Fatal(err)
	}
}

func TestObservationChecksConcurrent(t *testing.T) {
	c := testChecks(t)
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := c.CheckReadWindow(testClock(101000), testClock(101010), testHeader(1)); err != nil {
				t.Error(err)
			}
			if err := c.CheckLineage(testHeader(1), testHeader(2), []FinalizedHeader{testHeader(1)}); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
}

func ExampleObservationChecks() {
	checks, err := NewObservationChecks(ObservationLimits{
		ClockID: "example-clock", PolicyGeneration: "example-only",
		MaxAcquisitionMS: 10, MaxObservationAgeMS: 20, MaxFinalizedLagMS: 1000, MaxAncestryHeaders: 1,
	})
	if err != nil {
		panic(err)
	}
	start := ClockSample{ClockID: "example-clock", Epoch: "one-run", WallMS: 101000, MonotonicMS: 1}
	end := start
	end.WallMS += 10
	end.MonotonicMS += 10
	header := FinalizedHeader{Hash: "0x" + strings.Repeat("1", 64), ParentHash: "0x" + strings.Repeat("0", 64), Timestamp: 101}
	fmt.Println(checks.CheckReadWindow(start, end, header))
	// Output: <nil>
}
