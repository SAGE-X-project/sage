package registry011

import (
	"bytes"
	"context"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// ReplayBatch contains EVERY consecutive block following the retained cursor
// (including deployment in the first batch), and only the new selected history.
// Parents descends from SelectedFinalized's parent through the preceding cursor
// inclusive; the first baseline is the separately authenticated Deployment.
// Ancestor metadata, selected finality, node/code identity and state-read
// provenance must be qualified separately. These fields cannot authenticate RPC.
type ReplayBatch struct {
	SelectedFinalized, Anchor FinalizedHeader
	Parents                   []FinalizedHeader
	Blocks                    []ReplayBlock
	History                   []HistoryBytes
	Record, Grants            []byte
}

type replayStream struct {
	state    replayState
	claims   map[replayClaimID]replayClaimState
	created  map[string]bool
	selected FinalizedHeader
}

func (s replayStream) apply(ctx context.Context, c JournalConfig, tip JournalCheckpoint, in ReplayBatch) (replayStream, error) {
	if err := ctx.Err(); err != nil {
		return replayStream{}, err
	}
	l := c.Replay
	if len(in.Blocks) == 0 || len(in.Blocks) > l.MaxBlocks || len(in.History) > int(l.History.MaxItems) || uint64(len(in.Parents)) > c.Observation.MaxAncestryHeaders {
		return replayStream{}, ErrReplayUnavailable
	}
	b := replayBudget{limits: l}
	for _, p := range in.Parents {
		if !b.charge(len(p.Hash) + len(p.ParentHash) + 16) {
			return replayStream{}, ErrReplayUnavailable
		}
	}
	remaining := l.History.MaxBytes
	charge := func(raw []byte, maximum int) bool {
		if len(raw) > maximum || len(raw) > remaining || !b.charge(len(raw)) {
			return false
		}
		remaining -= len(raw)
		return true
	}
	if !charge(in.Record, 65536) || !charge(in.Grants, 262144) {
		return replayStream{}, ErrReplayUnavailable
	}
	for _, h := range in.History {
		if len(h.Entry) == 0 || len(h.Command) == 0 || !charge(h.Entry, 262144) || !charge(h.Command, 262144) {
			return replayStream{}, ErrReplayUnavailable
		}
	}
	base, selected := tip.Anchor, s.selected
	if tip.Sequence == 0 {
		base, selected = c.Deployment, c.Deployment
	}
	checks, _ := NewObservationChecks(c.Observation)
	if err := checks.CheckLineage(base, in.SelectedFinalized, in.Parents); err != nil {
		return replayStream{}, err
	}
	at := func(number uint64) FinalizedHeader {
		if number == in.SelectedFinalized.Number {
			return in.SelectedFinalized
		}
		return in.Parents[in.SelectedFinalized.Number-number-1]
	}
	if selected.Number < base.Number || selected.Number > in.SelectedFinalized.Number || at(selected.Number) != selected {
		return replayStream{}, ErrAncestry
	}
	a, outcomes, err := replayABIs()
	if err != nil {
		return replayStream{}, ErrConfiguration
	}
	// Work on private copies. Failed batches never mutate previously committed
	// record/grants, lifetime duplicate indexes, or the externally retained tip.
	next := replayStream{state: s.state, claims: make(map[replayClaimID]replayClaimState, len(s.claims)), created: make(map[string]bool, len(s.created)), selected: in.SelectedFinalized}
	next.state.record.Keys = append([]replayKey{}, s.state.record.Keys...)
	for i, k := range next.state.record.Keys {
		if k.Expires != nil {
			x := *k.Expires
			next.state.record.Keys[i].Expires = &x
		}
	}
	next.state.record.Services = append([]replayService{}, s.state.record.Services...)
	next.state.grants = append([]replayGrant{}, s.state.grants...)
	for k, v := range s.claims {
		next.claims[k] = v
	}
	for k, v := range s.created {
		next.created[k] = v
	}
	r := replayRun{registry: c.Registry, agent: c.Agent, contract: a, outcomes: outcomes, claims: next.claims, created: next.created, state: next.state, baseVersion: next.state.version, budget: &b, history: in.History, claimLimit: c.MaxClaims, createdLimit: c.MaxIdentities}
	previous := base
	for i, raw := range in.Blocks {
		decoded, err := replayDecodeBlock(ctx, raw, &b)
		if err != nil {
			return replayStream{}, err
		}
		h := replayHeader(decoded.header)
		if tip.Sequence == 0 && i == 0 {
			if h != c.Deployment {
				return replayStream{}, ErrAncestry
			}
		} else if previous.Number == ^uint64(0) || h.Number != previous.Number+1 || h.ParentHash != previous.Hash || h.Timestamp <= previous.Timestamp {
			return replayStream{}, ErrAncestry
		}
		if h.Number > in.SelectedFinalized.Number || at(h.Number) != h {
			return replayStream{}, ErrAncestry
		}
		if err := r.block(ctx, raw, decoded); err != nil {
			return replayStream{}, err
		}
		previous = h
	}
	if previous != in.Anchor || len(r.steps) != len(in.History) {
		return replayStream{}, ErrReplay
	}
	for _, step := range r.steps {
		// event already charged these derived bytes to the replay work budget.
		if len(step.Record)+len(step.Grants) > remaining {
			return replayStream{}, ErrReplayUnavailable
		}
		remaining -= len(step.Record) + len(step.Grants)
	}
	next.state = r.state
	if !next.matches(in.Record, in.Grants) {
		return replayStream{}, ErrReplay
	}
	if err := ctx.Err(); err != nil {
		return replayStream{}, err
	}
	return next, nil
}

func (s replayStream) matches(record, grants []byte) bool {
	if s.state.version == 0 {
		return len(record) == 0 && len(grants) == 0
	}
	r, g := s.stateBytes()
	return bytes.Equal(record, r) && bytes.Equal(grants, g)
}

func (s replayStream) stateBytes() ([]byte, []byte) {
	if s.state.version == 0 {
		return nil, nil
	}
	// Only replayState.apply constructs these bounded ASCII typed schemas.
	r, _ := rfc8785.Marshal(s.state.record)
	g, _ := rfc8785.Marshal(s.state.grants)
	return r, g
}
