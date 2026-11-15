package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryTraceNormalizationEnvelopeMatchesPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, row := range []struct {
		limits                    providerHistoryTraceNormalizationLimits
		decoded, frames, logs     uint64
		constant, wordCoefficient uint64
	}{
		{providerHistoryTraceNormalizationLimits{4, 1, 1, 65536, 65536}, 458, 1, 1, 32588, 474},
		{providerHistoryTraceNormalizationLimits{1024, 1024, 1, 4, 6}, 143104, 4, 6, 145687014, 423240},
		{providerHistoryTraceNormalizationLimits{8192, 327680, 40, 80, 40}, 45235200, 80, 40, 370705828120, 135335040},
		{providerHistoryTraceNormalizationLimits{8388608, 335544320, 40, 80, 40}, 46305131520, 80, 40, 388435737304566040, 138579807360},
		{providerHistoryTraceNormalizationLimits{65536, 196608, 3, 2, 1}, 27132288, 2, 1, 1778274381071, 81199184},
	} {
		got, ok := providerHistoryTraceNormalizationRefusalUpper(row.limits)
		want := candidateWorkVector{2: row.decoded, 5: row.frames, 6: row.logs, 10: row.constant + row.wordCoefficient*w}
		if !ok || got != want {
			t.Fatal("portable whole-normalization row", row.limits, got, want, ok)
		}
	}
}

func TestProviderHistoryTraceNormalizationEnvelopeRejectsInvalidOrOverflowingRows(t *testing.T) {
	for _, mode := range []string{"body-zero", "body-small", "body-wide", "frames-zero", "frames-wide", "logs-zero", "logs-wide", "attempts-wide", "raw-small", "raw-wide", "raw-max", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			p := providerHistoryTraceNormalizationLimits{1024, 1024, 1, 4, 6}
			switch mode {
			case "body-zero":
				p.BodyBytes = 0
			case "body-small":
				p.BodyBytes = 3
			case "body-wide":
				p.BodyBytes = (8 << 20) + 1
			case "frames-zero":
				p.Frames = 0
			case "frames-wide":
				p.Frames = 65537
			case "logs-zero":
				p.Logs = 0
			case "logs-wide":
				p.Logs = 65537
			case "attempts-wide":
				p.Attempts = p.Frames + 2
			case "raw-small":
				p.RawBytes = 0
			case "raw-wide":
				p.RawBytes = p.BodyBytes + 1
			case "raw-max":
				p.RawBytes = ^uint64(0)
			case "overflow":
				p = providerHistoryTraceNormalizationLimits{8 << 20, (8 << 20) * 65537, 65537, 65536, 65536}
			}
			if got, ok := providerHistoryTraceNormalizationRefusalUpper(p); ok || got != (candidateWorkVector{}) {
				t.Fatal("failed arithmetic/domain published a partial quote", p, got, ok)
			}
		})
	}
}

func TestProviderHistoryTraceNormalizationEnvelopeZeroAttemptsHaveNoWork(t *testing.T) {
	p := providerHistoryTraceNormalizationLimits{BodyBytes: 1024, Frames: 4, Logs: 6}
	if got, ok := providerHistoryTraceNormalizationRefusalUpper(p); !ok || got != (candidateWorkVector{}) {
		t.Fatal("no normalization invented frame/log work", got, ok)
	}
	p.RawBytes = 1
	if got, ok := providerHistoryTraceNormalizationRefusalUpper(p); ok || got != (candidateWorkVector{}) {
		t.Fatal("raw bytes without an original attempt accepted", got, ok)
	}
}

func TestProviderHistoryTraceNormalizationEnvelopeFundsSharedCapacityRefusals(t *testing.T) {
	for _, mode := range []string{"frames", "logs"} {
		t.Run(mode, func(t *testing.T) {
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = 2, 1
			if mode == "logs" {
				limits.MaxFrames, limits.MaxLogs = 3, 2
			}
			raw := historyNormalizationTrace(t, historyNormalizationLeaf(mode == "logs"))
			original := bytes.Clone(raw)
			p := providerHistoryTraceNormalizationLimits{uint64(len(raw)), 3 * uint64(len(raw)), 3, uint64(limits.MaxFrames), uint64(limits.MaxLogs)}
			upper, ok := providerHistoryTraceNormalizationRefusalUpper(p)
			if !ok {
				t.Fatal("finite three-root quote")
			}
			m := providerHistoryContentUnitMeter(upper)
			b := replayBudget{limits: limits, observer: m}
			var beforeLast candidateWorkVector
			for attempt := 0; attempt < 3; attempt++ {
				if attempt == 2 {
					beforeLast = m.actual
				}
				_, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b)
				if (err == nil) != (attempt < 2) || m.failed {
					t.Fatal("shared quota refusal was confused with underfunded work", attempt, err, m.actual)
				}
			}
			if b.frames != limits.MaxFrames || b.traceLogs != map[string]int{"frames": 0, "logs": 2}[mode] || !bytes.Equal(raw, original) {
				t.Fatal("original raw bytes or cumulative parser counters changed", b.frames, b.traceLogs)
			}
			if mode == "frames" && (m.actual[2]-beforeLast[2] != 2*uint64(len(raw)) || m.actual[5] != beforeLast[5]) {
				t.Fatal("exhausted frame cap lost the final raw preflight", m.actual, beforeLast)
			}
			historyEventQueueWithin(t, m.actual, upper)
		})
	}
}

func TestProviderHistoryTraceNormalizationEnvelopeFundsRefusingScalarAndFieldPrefixes(t *testing.T) {
	for _, mode := range []string{"quantity-range", "unknown-name"} {
		t.Run(mode, func(t *testing.T) {
			object := nodeTestFrame(historyNormalizationLeaf(false))
			if mode == "quantity-range" {
				object["gas"] = "0x" + strings.Repeat("f", 64)
			} else {
				object[strings.Repeat("z", 4096)] = true
			}
			raw, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			upper, ok := providerHistoryTraceNormalizationRefusalUpper(providerHistoryTraceNormalizationLimits{uint64(len(raw)), uint64(len(raw)), 1, 1, 1})
			if !ok {
				t.Fatal("finite fixed DATA prefix quote")
			}
			m := providerHistoryContentUnitMeter(upper)
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = 1, 1
			b := replayBudget{limits: limits, observer: m}
			got, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b)
			if err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || m.failed || b.frames != 1 || b.traceLogs != 0 || m.actual[2] == 0 || m.actual[10] == 0 {
				t.Fatal("ordinary semantic refusal exceeded its complete entered envelope", err, b.frames, m.actual)
			}
			historyEventQueueWithin(t, m.actual, upper)
		})
	}
}

func TestProviderHistoryTraceNormalizationEnvelopeFundsUnvisitedSiblingStorage(t *testing.T) {
	// Root reserves two children; its first child reserves a grandchild. All
	// three frames fit, then the unvisited sibling hits the frame cap. Three
	// declared child slots already exist although only two child frames entered.
	grandchild := historyNormalizationLeaf(false)
	grandchild.From = SepoliaRegistryAddress
	child := historyNormalizationLeaf(false)
	child.From, child.To, child.Calls = replayTestWallet, SepoliaRegistryAddress, []ReplayFrame{grandchild}
	root := historyNormalizationLeaf(false)
	object := nodeTestFrame(root)
	object["calls"] = []any{nodeTestFrame(child), map[string]any{}}
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(raw)
	upper, ok := providerHistoryTraceNormalizationRefusalUpper(providerHistoryTraceNormalizationLimits{uint64(len(raw)), uint64(len(raw)), 1, 3, 1})
	if !ok {
		t.Fatal("finite declared-sibling prefix quote")
	}
	m := providerHistoryContentUnitMeter(upper)
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 3, 1
	b := replayBudget{limits: limits, observer: m}
	got, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b)
	if err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || m.failed || b.frames != 3 || b.traceLogs != 0 || m.actual[5] != 3 || m.actual[6] != 0 ||
		m.actual[10] <= 5*nodeFrameCopyBytes || !bytes.Equal(raw, original) {
		t.Fatal("preallocated sibling work was hidden behind the visited-frame count", err, b.frames, m.actual)
	}
	historyEventQueueWithin(t, m.actual, upper)
}
