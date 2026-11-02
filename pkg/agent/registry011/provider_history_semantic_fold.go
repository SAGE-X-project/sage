package registry011

import (
	"context"
	"reflect"
)

// This is reconstructed semantic DATA, not a Source, execution owner, retained
// ACK or production cost witness. Its inputs must be selected from original
// T/C/S by BOTH independent collector readers. Connecting those readers and a
// observed constructor binds one original provider observer across all windows;
// the legacy constructor leaves it nil. Complete reader/cost/native/factory
// qualification remains mandatory. No exported API opens this DATA fold.
type providerHistorySemanticSelection struct {
	program                     string
	config                      JournalConfig
	selected                    FinalizedHeader
	maximumBlocks, maximumItems uint64
}

type providerHistorySemanticFold struct {
	owner                    *providerHistorySemanticFold
	selection, sealed        providerHistorySemanticSelection
	confirmed                replayStream
	last                     FinalizedHeader
	window                   *providerHistoryExpectedWindow
	blocks, items            uint64
	windows                  int
	failed                   bool
	observer, sealedObserver *providerWorkMeter
}

func (*providerHistorySemanticFold) UnmarshalJSON([]byte) error { return ErrReplay }

func newProviderHistorySemanticFold(program string, config JournalConfig, selected FinalizedHeader, maximumBlocks, maximumItems uint64) (*providerHistorySemanticFold, error) {
	return newProviderHistorySemanticFoldWithObserver(program, config, selected, maximumBlocks, maximumItems, nil)
}

func newProviderHistorySemanticFoldWithObserver(program string, config JournalConfig, selected FinalizedHeader, maximumBlocks, maximumItems uint64, meter *providerWorkMeter) (*providerHistorySemanticFold, error) {
	if !lowerHash(program) || !validHeader(selected) || selected.Number < config.Deployment.Number ||
		maximumBlocks == 0 || maximumBlocks > 1<<20 || maximumItems > 1<<20 {
		return nil, ErrConfiguration
	}
	span := selected.Number - config.Deployment.Number
	if span >= maximumBlocks || span > config.Observation.MaxAncestryHeaders {
		return nil, ErrConfiguration
	}
	if _, _, err := journalConfigurationObserved(config, meter); err != nil {
		return nil, err
	}
	s := providerHistorySemanticSelection{program, config, selected, maximumBlocks, maximumItems}
	f := &providerHistorySemanticFold{selection: s, sealed: s, last: config.Deployment}
	f.owner = f
	return f, nil
}

func (f *providerHistorySemanticFold) valid() bool {
	return f != nil && f.owner == f && !f.failed && f.observer == f.sealedObserver && f.selection == f.sealed &&
		f.blocks <= f.selection.maximumBlocks && f.items <= f.selection.maximumItems
}

func (f *providerHistorySemanticFold) refuse() error {
	if f != nil && f.owner == f {
		f.failed = true // An entered failure never resets shared indexes or quotas.
	}
	return ErrReplay
}

func (f *providerHistorySemanticFold) beginWindowCandidate(ctx context.Context) error {
	if !f.valid() || ctx == nil || ctx.Err() != nil || f.window != nil || f.atEnd() || f.windows >= f.selection.config.MaxBatches {
		return f.refuse()
	}
	a, outcomes, err := replayABIsObserved(f.observer)
	if err != nil || ctx.Err() != nil {
		return f.refuse()
	}
	next, cloned := f.cloneConfirmedObserved()
	if !cloned || ctx.Err() != nil || !f.observedCopy(uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()+reflect.TypeFor[replayRun]().Size())) {
		return f.refuse()
	}
	c := f.selection.config
	w := &providerHistoryExpectedWindow{fold: f, baseVersion: next.state.version, budget: replayBudget{limits: c.Replay, observer: f.observer}}
	w.owner = w
	w.run = &replayRun{registry: c.Registry, agent: c.Agent, contract: a, outcomes: outcomes,
		claims: next.claims, created: next.created, state: next.state, baseVersion: next.state.version,
		budget: &w.budget, claimLimit: c.MaxClaims, createdLimit: c.MaxIdentities}
	f.window = w
	return nil
}

// The outer closed collector must additionally select want from its own fresh
// complete spine and qualify chain/code/finality/native gas provenance. Matching
// a caller's header here proves only contiguous semantic DATA, never RPC trust.
func (f *providerHistorySemanticFold) applyBlockCandidate(ctx context.Context, raw ReplayBlock, want FinalizedHeader) error {
	if !f.valid() || ctx == nil || ctx.Err() != nil || f.window == nil || !f.window.selected(f.window.run) || f.window.comparing ||
		f.window.blocks >= f.selection.config.Replay.MaxBlocks || f.blocks >= f.selection.maximumBlocks ||
		!validHeader(want) || want.Number > f.selection.selected.Number || want.Number == f.selection.selected.Number && want != f.selection.selected {
		return f.refuse()
	}
	if f.blocks == 0 {
		if want != f.selection.config.Deployment {
			return f.refuse()
		}
	} else if f.last.Number == ^uint64(0) || want.Number != f.last.Number+1 || want.ParentHash != f.last.Hash || want.Timestamp <= f.last.Timestamp {
		return f.refuse()
	}
	w := f.window
	decoded, err := replayDecodeBlock(ctx, raw, &w.budget)
	if err != nil || replayHeader(decoded.header) != want || w.run.blockDeriving(ctx, raw, decoded, w) != nil || ctx.Err() != nil {
		return f.refuse()
	}
	if !f.observedCopy(uint64(reflect.TypeFor[FinalizedHeader]().Size() + 8 + reflect.TypeFor[int]().Size())) {
		return f.refuse()
	}
	f.last, f.blocks, w.blocks = want, f.blocks+1, w.blocks+1
	return nil
}

// The stop read must agree with independently derived state BEFORE any
// history request is selected. A deactivated record remains present/versioned.
// No present=false result produces an authenticated-absence credential.
func (f *providerHistorySemanticFold) compareStateCandidate(ctx context.Context, present bool, record, grants []byte) error {
	if !f.valid() || ctx == nil || ctx.Err() != nil || f.window == nil || !f.window.selected(f.window.run) ||
		f.window.blocks == 0 || f.window.comparing || f.window.pending || len(record) > 65536 || len(grants) > 262144 {
		return f.refuse()
	}
	w := f.window
	if present != (w.run.state.version != 0) {
		return f.refuse()
	}
	wantRecord, wantGrants, encoded := (replayStream{state: w.run.state}).stateBytesObserved(f.observer)
	if !encoded || !providerReplayEqual(record, wantRecord, f.observer) || !providerReplayEqual(grants, wantGrants, f.observer) || !w.chargeHistory(len(record)+len(grants)) || ctx.Err() != nil || !f.observedCopy(1) {
		return f.refuse()
	}
	w.comparing = true
	return nil
}

func (f *providerHistorySemanticFold) compareHistoryItemCandidate(ctx context.Context, version uint64, entry, command []byte) error {
	if !f.valid() || ctx == nil || ctx.Err() != nil || f.window == nil || !f.window.selected(f.window.run) ||
		!f.window.comparing || f.window.compared >= len(f.window.queue) || len(entry) == 0 || len(entry) > 262144 || len(command) == 0 || len(command) > 262144 {
		return f.refuse()
	}
	w := f.window
	equal := false
	if f.observer != nil {
		equal = w.historyEqualObserved(version, entry, command)
	} else {
		want := w.queue[w.compared]
		equal = version == want.version && providerReplayEqual(entry, []byte(want.entry), nil) && providerReplayEqual(command, []byte(want.command), nil)
	}
	if !equal || !w.chargeHistory(len(entry)+len(command)) || ctx.Err() != nil || !f.observedCopy(uint64(reflect.TypeFor[int]().Size())) {
		return f.refuse()
	}
	w.compared++
	return nil
}

func (f *providerHistorySemanticFold) finishWindowCandidate(ctx context.Context) error {
	if !f.valid() || ctx == nil || ctx.Err() != nil || f.window == nil || !f.window.selected(f.window.run) ||
		!f.window.comparing || f.window.pending || f.window.blocks == 0 || f.window.compared != len(f.window.queue) {
		return f.refuse()
	}
	w := f.window
	if !f.observedCopy(uint64(reflect.TypeFor[replayStream]().Size() + reflect.TypeFor[int]().Size() + reflect.TypeFor[*providerHistoryExpectedWindow]().Size() + 1)) {
		return f.refuse()
	}
	// Only this private closed window owns these arrays/maps; no method returns
	// them or an expected queue to a producer. beginWindow clones every mutable
	// state/index so a later failed window cannot change this confirmed prefix.
	f.confirmed = replayStream{state: w.run.state, claims: w.run.claims, created: w.run.created, selected: f.selection.selected}
	w.closed, f.window, f.windows = true, nil, f.windows+1
	return nil
}

// DATA completeness only; final page, original C0/Dp/C1, elapsed/native
// qualification and operating/publication authorization remain outside it.
func (f *providerHistorySemanticFold) atEnd() bool {
	return f.valid() && f.window == nil && f.windows > 0 && f.last == f.selection.selected &&
		f.blocks == f.selection.selected.Number-f.selection.config.Deployment.Number+1
}
