package registry011

import "bytes"

type providerHistoryExpectedItem struct {
	version        uint64
	entry, command string
}

// The new derivation is explicit, private DATA. nil history in an existing
// strict API NEVER selects it. Only its own run and original fold may append.
// It keeps exact expected entry/command bytes, not every intermediate record.
type providerHistoryExpectedWindow struct {
	owner                      *providerHistoryExpectedWindow
	fold                       *providerHistorySemanticFold
	run                        *replayRun
	budget                     replayBudget
	baseVersion                uint64
	queue                      []providerHistoryExpectedItem
	historyBytes               int
	blocks, compared           int
	pendingVersion             uint64
	pendingCommand             string
	pending, comparing, closed bool
}

func (*providerHistoryExpectedWindow) UnmarshalJSON([]byte) error { return ErrReplay }

func (w *providerHistoryExpectedWindow) selected(r *replayRun) bool {
	return w != nil && w.owner == w && !w.closed && w.fold != nil && w.fold.valid() && w.fold.window == w && r != nil && w.run == r &&
		r.budget == &w.budget && r.budget.observer == w.fold.observer && r.baseVersion == w.baseVersion && r.registry == w.fold.selection.config.Registry && r.agent == w.fold.selection.config.Agent &&
		r.budget.limits == w.fold.selection.config.Replay && r.claimLimit == w.fold.selection.config.MaxClaims && r.createdLimit == w.fold.selection.config.MaxIdentities &&
		r.history == nil && len(r.steps) == 0
}

func (w *providerHistoryExpectedWindow) prepareSubject(r *replayRun, command replayCommand) bool {
	if !w.selected(r) || w.comparing || w.pending || w.blocks >= w.budget.limits.MaxBlocks ||
		uint64(len(w.queue)) >= w.budget.limits.History.MaxItems || w.fold.items >= w.fold.selection.maximumItems ||
		w.baseVersion > ^uint64(0)-uint64(len(w.queue)) || r.state.version != w.baseVersion+uint64(len(w.queue)) ||
		command.expected != r.state.version || command.expected == ^uint64(0) || len(command.raw) == 0 || len(command.raw) > 262144 {
		return false
	}
	// These are existing closed entry/record/grants byte ceilings. Ensure the
	// next bounded state/crypto step and queue write fit BEFORE deriving it.
	need := 262144 + 65536 + 262144 + len(command.raw)
	if w.historyBytes < 0 || need > w.budget.limits.History.MaxBytes-w.historyBytes || need > w.budget.limits.MaxEvidenceBytes-w.budget.bytes {
		return false
	}
	if w.budget.observer != nil {
		return w.prepareSubjectObserved(command)
	}
	w.pendingVersion, w.pendingCommand, w.pending = command.expected+1, string(command.raw), true
	return true
}

func (w *providerHistoryExpectedWindow) retainExpected(r *replayRun, step HistoryContentStep) bool {
	if w != nil && w.budget.observer != nil {
		return w.retainExpectedObserved(r, step)
	}
	if !w.selected(r) || !w.pending || step.History.Version != w.pendingVersion || r.state.version != w.pendingVersion ||
		len(step.History.Entry) == 0 || len(step.History.Entry) > 262144 || len(step.History.Command) == 0 || len(step.History.Command) > 262144 ||
		!bytes.Equal(step.History.Command, []byte(w.pendingCommand)) ||
		!w.chargeHistory(len(step.Record)+len(step.Grants)+len(step.History.Entry)+len(step.History.Command)) {
		return false
	}
	w.queue = append(w.queue, providerHistoryExpectedItem{step.History.Version, string(step.History.Entry), string(step.History.Command)})
	w.fold.items++ // Whole-program derived history count is never reset per window.
	w.pending, w.pendingCommand = false, ""
	return true
}

func (w *providerHistoryExpectedWindow) chargeHistory(n int) bool {
	if w == nil || n < 0 || w.historyBytes < 0 || n > w.budget.limits.History.MaxBytes-w.historyBytes ||
		w.budget.observer != nil && !w.budget.observer.event(providerCopyEvent, 16) || !w.budget.charge(n) {
		return false
	}
	w.historyBytes += n
	return true
}
