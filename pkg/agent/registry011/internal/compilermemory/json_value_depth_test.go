package compilermemory

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func valueDepthTestInput(containers int, leaf, kind string) string {
	for index := 0; index < containers; index++ {
		if kind == "object" || (kind == "mixed" && index%2 == 0) {
			leaf = "{\"k\":" + leaf + "}"
		} else {
			leaf = "[" + leaf + "]"
		}
	}
	return leaf
}

func TestRegionJSONValueDepthPreservesRootZeroBoundaries(t *testing.T) {
	for _, kind := range []string{"array", "object", "mixed"} {
		for _, tc := range []struct {
			depth int
			leaf  string
			valid bool
		}{
			{63, "[]", true}, {64, "[]", true}, {65, "[]", false},
			{64, "0", true}, {65, "0", false},
			{64, "\"x\"", true}, {65, "\"x\"", false},
			{64, "true", true}, {65, "false", false},
			{64, "null", true}, {65, "null", false},
			{64, "{ }", true}, {64, "{\"x\":0}", false},
		} {
			name := kind+"/"+strconv.Itoa(tc.depth)+"/"+tc.leaf
			t.Run(name, func(t *testing.T) {
				action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				t.Cleanup(cancel)
				region := jsonTestRegion(t, action)
				span, err := region.Store([]byte(valueDepthTestInput(tc.depth, tc.leaf, kind)))
				if err != nil {
					t.Fatal(err)
				}
				document, err := region.ParseJSONAtValueDepth(span, JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: 512})
				if (err == nil) != tc.valid {
					t.Fatalf("value-depth acceptance=%v, expected=%v, error=%v", err == nil, tc.valid, err)
				}
				if tc.valid {
					if _, err := document.Root().Kind(); err != nil {
						t.Fatal(err)
					}
				} else if _, err := region.Store(nil); !errors.Is(err, ErrClosed) {
					t.Fatalf("entered depth refusal did not poison owner: %v", err)
				}
			})
		}
	}
	for _, raw := range []string{"0", "\"x\"", "true", "null", "[]", "{}"} {
		action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		t.Cleanup(cancel)
		region := jsonTestRegion(t, action)
		span, err := region.Store([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := region.ParseJSONAtValueDepth(span, JSONValueDepthLimits{MaximumValueDepth: 0, MaximumNodes: 4}); err != nil {
			t.Fatalf("root depth zero %q: %v", raw, err)
		}
	}
}

func TestRegionJSONValueDepthDoesNotChangeExistingStackLimits(t *testing.T) {
	for _, raw := range []string{
		strings.Repeat("[", 65) + strings.Repeat("]", 65),
		valueDepthTestInput(64, "{}", "object"),
	} {
		action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		t.Cleanup(cancel)
		region := jsonTestRegion(t, action)
		span, err := region.Store([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := region.ParseJSON(span, JSONLimits{MaximumDepth: 64, MaximumNodes: 512}); !errors.Is(err, ErrRange) {
			t.Fatalf("original stack64 accepted65 containers: %v", err)
		}
	}
	for _, raw := range []string{"[0]", "{\"k\":0}"} {
		action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		t.Cleanup(cancel)
		region := jsonTestRegion(t, action)
		span, err := region.Store([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := region.ParseJSONAtValueDepth(span, JSONValueDepthLimits{MaximumValueDepth: 0, MaximumNodes: 4}); err == nil {
			t.Fatal("value depth zero accepted a child")
		}
	}
}

func TestRegionJSONStorageCapacitiesRefuseOverflowAndInvalidLimits(t *testing.T) {
	for _, tc := range []struct {
		bytes int
		nodes int
		depth int
	}{
		{0, 1, 1}, {-1, 1, 1}, {1, 0, 1}, {1, -1, 1},
		{1, 1, -1}, {1, 1, 65}, {math.MaxInt, 1, 1}, {1, math.MaxInt, 1},
	} {
		if capacity, err := JSONCapacity(tc.bytes, JSONLimits{MaximumDepth: tc.depth, MaximumNodes: tc.nodes}); capacity != 0 || err == nil {
			t.Fatalf("invalid original capacity: %d, %v", capacity, err)
		}
		if capacity, err := JSONValueCapacity(tc.bytes, JSONValueDepthLimits{MaximumValueDepth: tc.depth, MaximumNodes: tc.nodes}); capacity != 0 || err == nil {
			t.Fatalf("invalid value-depth capacity: %d, %v", capacity, err)
		}
	}
	for _, legacyValues := range []bool{false, true} {
		raw := []byte("[0]")
		limits := JSONLimits{MaximumDepth: 2, MaximumNodes: 2}
		valueLimits := JSONValueDepthLimits{MaximumValueDepth: 1, MaximumNodes: 2}
		capacity, err := JSONCapacity(len(raw), limits)
		if legacyValues {
			capacity, err = JSONValueCapacity(len(raw), valueLimits)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, missing := range []int{0, 1} {
			action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cleanup, stopCleanup := context.WithTimeout(context.Background(), time.Minute)
			region, err := acquireWithOperations(action, cleanup, capacity-missing, &simulatedRegionOperations{})
			if err != nil {
				cancel()
				stopCleanup()
				t.Fatal(err)
			}
			span, err := region.Store(raw)
			if err == nil {
				if legacyValues {
					_, err = region.ParseJSONAtValueDepth(span, valueLimits)
				} else {
					_, err = region.ParseJSON(span, limits)
				}
			}
			if (err == nil) != (missing == 0) {
				t.Fatalf("capacity boundary missing=%d: %v", missing, err)
			}
			if err := region.Close(); err != nil {
				t.Fatal(err)
			}
			cancel()
			stopCleanup()
		}
	}
	if _, err := JSONCapacity(1, JSONLimits{MaximumDepth: 0, MaximumNodes: 1}); err == nil {
		t.Fatal("existing depth zero became valid")
	}
}
