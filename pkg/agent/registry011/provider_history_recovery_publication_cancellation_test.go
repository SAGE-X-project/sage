package registry011

import (
	"context"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

// Select only direct guards of one original replay entry. Native file-read
// guards and nested semantic readers do not change this private UNIT ordinal.
// This context cannot install a role, replace an observer or renew an allowance.
type historyRecoveryPublicationCancelContext struct {
	context.Context
	cancel        context.CancelFunc
	guard, caller string
	limit, checks int
}

func (c *historyRecoveryPublicationCancelContext) Err() error {
	var pcs [4]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var names [3]string
	for i := range names {
		frame, more := frames.Next()
		names[i] = frame.Function
		if !more {
			break
		}
	}
	if names[1] == c.guard && names[2] == c.caller && c.Context.Err() == nil {
		c.checks++
		if c.checks == c.limit {
			c.cancel()
		}
	}
	return c.Context.Err()
}

func TestProviderHistoryRecoveryPublicationCancellationCannotSelectPartialReplayModels(t *testing.T) {
	for _, mode := range []string{"legacy", "initial", "group"} {
		limits := []int{2}
		caller := "replayLegacyPrefixCandidate"
		switch mode {
		case "initial":
			limits, caller = []int{2, 10}, "replayInitialModelCandidate"
		case "group":
			limits, caller = []int{2, 3}, "replayGroupModelCandidate"
		}
		for pass := range 2 {
			for _, limit := range limits {
				t.Run(mode+"-"+strconv.Itoa(pass)+"-"+strconv.Itoa(limit), func(t *testing.T) {
					var o *providerHistoryGroupOwner
					var r *providerHistoryRecoveryOwner
					var want map[string]providerIntegrationObjectRef
					if mode == "group" {
						o, _ = historyRecoveryGroupFixture(t, 1)
						r, want = historyRecoveryGroupStart(t, o)
					} else {
						o, r, want = historyRecoveryInventoryFixtureWithLegacy(t, true)
					}
					prepare := func(selected int) {
						if mode == "legacy" {
							historyRecoveryLegacySource(t, r, selected, want)
						} else {
							historyRecoveryModelPrefix(t, r, selected, want)
							if mode == "group" {
								if out, err := r.replayInitialModelCandidate(context.Background(), selected); err != nil || out == nil || !out.selectedLocked(r) {
									t.Fatal("genuine original initial model prerequisite", err)
								}
							}
						}
					}
					invoke := func(ctx context.Context, selected int) (bool, error) {
						switch mode {
						case "legacy":
							out, err := r.replayLegacyPrefixCandidate(ctx, selected)
							return out == nil, err
						case "initial":
							out, err := r.replayInitialModelCandidate(ctx, selected)
							return out == nil, err
						default:
							out, err := r.replayGroupModelCandidate(ctx, selected)
							return out == nil, err
						}
					}
					for previous := range pass {
						prepare(previous)
						if empty, err := invoke(context.Background(), previous); err != nil || empty {
							t.Fatal("independent prior original pass prerequisite", err)
						}
					}
					prepare(pass)
					s, m := o.store, r.meter
					if !o.failed || !o.meter.closed || !r.active || r.closed || r.failed || s.activeMeter.Load() != nil {
						t.Fatal("terminal producer and original healthy read-only lane required")
					}
					producer, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
					work, upper, clocks, files, bytes := m.actual, m.upper, m.clocks, r.inputFiles, r.inputBytes
					legacy, initial, groups := r.legacyPrefixes, r.initialModels, r.groupModels
					confirmed, pending, pendingAck := r.confirmed, r.pending, r.pendingAck
					before := providerHistoryProvisionedOriginalBytes(t, s.config)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					prefix := reflect.TypeFor[providerHistoryRecoveryOwner]().PkgPath() + ".(*providerHistoryRecoveryOwner)."
					input := &historyRecoveryPublicationCancelContext{Context: ctx, cancel: cancel, guard: prefix + "inputReadyLocked", caller: prefix + caller, limit: limit}
					empty, err := invoke(input, pass)
					if err == nil || !empty || ctx.Err() != context.Canceled || input.checks != limit ||
						r.active || !r.used || !r.closed || !r.failed || !m.closed || !m.failed ||
						m.actual[10] <= work[10] || m.actual[0] != work[0] || m.actual[1] != work[1] || m.upper != upper || m.clocks != clocks ||
						r.inputFiles < files || r.inputBytes < bytes || r.legacyPrefixes != legacy || r.initialModels != initial || r.groupModels != groups ||
						r.confirmed != confirmed || r.pending != pending || r.pendingAck != pendingAck ||
						o.meter.actual != producer || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
						!o.failed || !o.meter.closed || s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
						t.Fatal("cancelled original replay selected partial DATA, changed originals or renewed a role", mode, pass, limit, err, input.checks)
					}
					for coordinate, actual := range work {
						if m.actual[coordinate] < actual {
							t.Fatal("cancelled replay refunded observed input work", coordinate)
						}
					}
					spent, inputFiles, inputBytes := m.actual, r.inputFiles, r.inputBytes
					if empty, err := invoke(context.Background(), pass); err == nil || !empty || r.active || !r.closed || !r.failed ||
						m.actual != spent || m.upper != upper || m.clocks != clocks || r.inputFiles != inputFiles || r.inputBytes != inputBytes ||
						r.legacyPrefixes != legacy || r.initialModels != initial || r.groupModels != groups ||
						o.meter.actual != producer || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack {
						t.Fatal("closed original inspection was revived or retried", mode, pass, limit, err)
					}
				})
			}
		}
	}
}
