package registry011

import (
	"context"
	"reflect"
	"runtime"
	"testing"
)

// Only direct readiness calls of the selected reconstruction stage count.
// Native reader loops may make any number of additional context checks; they
// cannot move the selected cancellation into an unfinished file read. Runtime
// frames are test-only observations and never modify a production role.
type historyRecoveryStageCancelContext struct {
	context.Context
	cancel                     context.CancelFunc
	ready, method              string
	checks, stageChecks, limit int
}

func (c *historyRecoveryStageCancelContext) Err() error {
	c.checks++
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
	if c.ready != "" && c.method != "" && names[1] == c.ready && names[2] == c.method {
		c.stageChecks++
		if c.limit > 0 && c.stageChecks == c.limit {
			c.cancel()
		}
	}
	return c.Context.Err()
}

// A complete positive replay determines the last direct readiness check for
// the same stage in another owned UNIT fixture. Cancellation changes only its
// standard context; original readers, limits and cumulative work stay intact.
func TestProviderHistoryRecoveryFinalReadinessCannotPublishCancelledModels(t *testing.T) {
	for _, stage := range []string{"legacy", "initial", "group"} {
		t.Run(stage, func(t *testing.T) {
			var finalCheck int
			var wholeFiles uint64
			for _, mode := range []string{"whole", "final-cancel"} {
				t.Run(mode, func(t *testing.T) {
					o, _ := historyRecoveryGroupFixture(t, 1)
					r, want := historyRecoveryGroupStart(t, o)
					switch stage {
					case "legacy":
						historyRecoveryLegacySource(t, r, 0, want)
					case "initial":
						historyRecoveryModelPrefix(t, r, 0, want)
					case "group":
						historyRecoveryGroupInitial(t, r, 0, want)
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					method := map[string]string{"legacy": "replayLegacyPrefixCandidate", "initial": "replayInitialModelCandidate", "group": "replayGroupModelCandidate"}[stage]
					prefix := reflect.TypeFor[providerHistoryRecoveryOwner]().PkgPath() + ".(*providerHistoryRecoveryOwner)."
					probe := &historyRecoveryStageCancelContext{Context: ctx, cancel: cancel, ready: prefix + "inputReadyLocked", method: prefix + method}
					wantChecks := map[string]int{"legacy": 2, "initial": 10, "group": 3}[stage]
					if mode == "final-cancel" {
						if finalCheck <= 1 || wholeFiles == 0 {
							t.Fatal("complete positive stage must establish its final readiness boundary")
						}
						probe.limit = finalCheck
					}
					originals := providerHistoryProvisionedOriginalBytes(t, r.config)
					prefixes, models, groups := r.legacyPrefixes, r.initialModels, r.groupModels
					initial, confirmed, pending, bounds := r.initial, r.confirmed, r.pending, r.bounds
					work, upper, files, bytes := r.meter.actual, r.meter.upper, r.inputFiles, r.inputBytes
					old, reads, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
					var present, selected bool
					var err error
					switch stage {
					case "legacy":
						out, e := r.replayLegacyPrefixCandidate(probe, 0)
						err, present = e, out != nil
						selected = out != nil && out.selectedLocked(r)
					case "initial":
						out, e := r.replayInitialModelCandidate(probe, 0)
						err, present = e, out != nil
						selected = out != nil && out.selectedLocked(r)
					case "group":
						out, e := r.replayGroupModelCandidate(probe, 0)
						err, present = e, out != nil
						selected = out != nil && out.selectedLocked(r)
					}
					if mode == "whole" {
						if err != nil || !present || !selected || probe.Context.Err() != nil || probe.stageChecks != wantChecks || probe.checks < probe.stageChecks ||
							!r.activeLocked(r.meter) || r.failed || r.closed {
							t.Fatal("complete positive original reconstruction prerequisite", stage, err)
						}
						finalCheck, wholeFiles = probe.stageChecks, r.inputFiles-files
					} else if err == nil || present || selected || probe.stageChecks != finalCheck || probe.checks < probe.stageChecks || probe.Context.Err() != context.Canceled ||
						r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed ||
						r.legacyPrefixes != prefixes || r.initialModels != models || r.groupModels != groups ||
						r.inputFiles-files != wholeFiles {
						t.Fatal("final cancellation published a partial model or bypassed the complete observed input boundary", stage, err, probe.stageChecks, finalCheck)
					}
					if r.initial != initial || r.confirmed != confirmed || r.pending != pending || r.bounds != bounds || r.meter.upper != upper ||
						r.inputFiles < files || r.inputBytes < bytes || o.meter.actual != old || o.store.clock.read != reads ||
						o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil ||
						!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
						t.Fatal("reconstruction changed original execution, checkpoints, allowances or retained bytes", stage, mode)
					}
					for coordinate, before := range work {
						if r.meter.actual[coordinate] < before {
							t.Fatal("reconstruction refunded already observed work", stage, mode, coordinate)
						}
					}
				})
			}
		})
	}
}

func TestProviderHistoryRecoveryLegacyInvalidMainSequenceDoesNotReadOriginals(t *testing.T) {
	var absent *providerHistoryRecoveryOwner
	if out, err := absent.replayLegacyPrefixCandidate(context.Background(), 0); err == nil || out != nil {
		t.Fatal("missing original owner supplied a legacy prefix")
	}
	o, r := historyRecoveryInputFixture(t)
	p := &providerHistoryRecoveryLegacyPrefix{recovery: r}
	originals := providerHistoryProvisionedOriginalBytes(t, r.config)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, sequence := range []uint64{0, uint64(len(p.records)) + 1} {
		work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
		old, reads, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
		out, err := p.mainLocked(context.Background(), providerFrontierHighwater{}, providerFrontierHighwater{sequence: sequence}, "", "")
		if err == nil || out != (providerHistoryRecoveryLegacyRecord{}) || p.bytes != 0 || p.owner != nil ||
			r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.activeLocked(r.meter) ||
			r.legacyPrefixes != ([2]*providerHistoryRecoveryLegacyPrefix{}) || o.meter.actual != old || o.store.clock.read != reads ||
			o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil ||
			!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
			t.Fatal("invalid private record sequence reached original input or gained a prefix", sequence, err)
		}
	}
}
