// Package registry011 contains explicit SAGE 0.11.0 Source checks. These checks
// do not enroll a Source, authenticate RPC results, or grant message authority.
package registry011

import "errors"

var (
	ErrConfiguration = errors.New("registry011: explicit Source configuration required")
	ErrClock         = errors.New("registry011: inconsistent trusted clock")
	ErrStale         = errors.New("registry011: observation exceeds its configured time budget")
	ErrAncestry      = errors.New("registry011: finalized ancestry is unavailable or inconsistent")
)

// ObservationLimits contains operator-supplied millisecond budgets, without
// defaults. Qualification of these values and the clock is external to this
// package. Existing 0.10.0 freshness rules are not changed.
type ObservationLimits struct {
	ClockID               string
	PolicyGeneration      string
	MaxClockUncertaintyMS int64
	MaxAcquisitionMS      int64
	MaxObservationAgeMS   int64
	MaxFinalizedLagMS     int64
	MaxAncestryHeaders    uint64
}

// ClockSample must come from the configured trusted local clock, never peer
// JSON or an RPC timestamp. Epoch identifies one monotonic-clock lifetime;
// samples from different process/boot epochs cannot be combined.
type ClockSample struct {
	ClockID, Epoch                     string
	WallMS, MonotonicMS, UncertaintyMS int64
}

// FinalizedHeader carries a complete header reference from the enrolled RPC.
// The caller must separately establish its chain/genesis, policy generation,
// finalized provenance and header authenticity. This is not a consensus proof.
type FinalizedHeader struct {
	Hash, ParentHash string
	Number           uint64
	Timestamp        uint64 // Unix seconds, at most 2^53-1.
}

// ObservationChecks retains a copied configuration. It has no network, storage,
// signing, reconfiguration or readiness path. A zero value rejects every check.
type ObservationChecks struct{ limits ObservationLimits }

func NewObservationChecks(limits ObservationLimits) (*ObservationChecks, error) {
	if limits.ClockID == "" || limits.PolicyGeneration == "" ||
		limits.MaxClockUncertaintyMS < 0 || limits.MaxAcquisitionMS <= 0 ||
		limits.MaxObservationAgeMS <= 0 || limits.MaxFinalizedLagMS <= 0 ||
		limits.MaxAncestryHeaders == 0 {
		return nil, ErrConfiguration
	}
	return &ObservationChecks{limits: limits}, nil
}

// CheckReadWindow bounds the whole acquisition using trusted local samples.
// Equality with a configured maximum is allowed. Header age includes the
// completion clock's uncertainty, so an uncertain stale header cannot pass.
func (c *ObservationChecks) CheckReadWindow(start, completed ClockSample, header FinalizedHeader) error {
	if c == nil || c.limits.PolicyGeneration == "" {
		return ErrConfiguration
	}
	return c.checkWindow(start, completed, header, c.limits.MaxAcquisitionMS)
}

// CheckPublication must run again after durable commit, before returning any
// observation. Use the original acquisition-start sample, never a replacement
// completion timestamp that makes old reads appear fresh. Failure does not undo
// committed state. Both samples must belong to the same trusted clock epoch.
func (c *ObservationChecks) CheckPublication(start, published ClockSample, header FinalizedHeader) error {
	if c == nil || c.limits.PolicyGeneration == "" {
		return ErrConfiguration
	}
	return c.checkWindow(start, published, header, c.limits.MaxObservationAgeMS)
}

func (c *ObservationChecks) checkWindow(start, end ClockSample, header FinalizedHeader, maximum int64) error {
	for _, sample := range []ClockSample{start, end} {
		if sample.ClockID != c.limits.ClockID || sample.Epoch == "" || sample.WallMS < 0 ||
			sample.MonotonicMS < 0 || sample.UncertaintyMS < 0 ||
			sample.UncertaintyMS > c.limits.MaxClockUncertaintyMS {
			return ErrClock
		}
	}
	if start.Epoch != end.Epoch || end.WallMS < start.WallMS || end.MonotonicMS < start.MonotonicMS {
		return ErrClock
	}
	wall := uint64(end.WallMS - start.WallMS)
	elapsed := uint64(end.MonotonicMS - start.MonotonicMS)
	uncertainty := uint64(start.UncertaintyMS) + uint64(end.UncertaintyMS)
	var difference uint64
	if wall >= elapsed {
		difference = wall - elapsed
	} else {
		difference = elapsed - wall
	}
	if difference > uncertainty {
		return ErrClock
	}
	if elapsed > uint64(maximum) {
		return ErrStale
	}
	if !validHeader(header) {
		return ErrAncestry
	}
	// 2^53-1 seconds times 1000 fits uint64 and int64. Two nonnegative
	// int64 values also fit uint64 when added; do not add them in int64.
	blockMS := header.Timestamp * 1000
	upperNow := uint64(end.WallMS) + uint64(end.UncertaintyMS)
	if blockMS > upperNow {
		return ErrClock
	}
	if upperNow-blockMS > uint64(c.limits.MaxFinalizedLagMS) {
		return ErrStale
	}
	return nil
}

// CheckLineage compares a durably retained anchor with a newly selected header.
// Parents are in descending order: current's parent through previous inclusive.
// No height-only shortcut, omitted parent, extra item or budget overflow passes.
// Enrollment of an initial anchor and authenticated policy changes are separate
// operations; a missing previous anchor is never created by this method.
func (c *ObservationChecks) CheckLineage(previous, current FinalizedHeader, parents []FinalizedHeader) error {
	if c == nil || c.limits.PolicyGeneration == "" {
		return ErrConfiguration
	}
	if !validHeader(previous) || !validHeader(current) || current.Number < previous.Number {
		return ErrAncestry
	}
	distance := current.Number - previous.Number
	if distance > c.limits.MaxAncestryHeaders || uint64(len(parents)) != distance {
		return ErrAncestry
	}
	if distance == 0 {
		if current != previous {
			return ErrAncestry
		}
		return nil
	}
	child := current
	seen := map[string]bool{current.Hash: true}
	for _, parent := range parents {
		if !validHeader(parent) || seen[parent.Hash] || child.Number == 0 ||
			parent.Number != child.Number-1 || parent.Hash != child.ParentHash ||
			parent.Timestamp >= child.Timestamp {
			return ErrAncestry
		}
		seen[parent.Hash] = true
		child = parent
	}
	if child != previous {
		return ErrAncestry
	}
	return nil
}

func validHeader(header FinalizedHeader) bool {
	return lowerHash(header.Hash) && lowerHash(header.ParentHash) && header.Timestamp <= 1<<53-1
}

func lowerHash(value string) bool {
	if len(value) != 66 || value[:2] != "0x" {
		return false
	}
	for _, b := range value[2:] {
		if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
			return false
		}
	}
	return true
}
