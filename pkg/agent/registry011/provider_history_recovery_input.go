package registry011

import (
	"context"
	"errors"
	"io"
	"os"
)

type providerHistoryRecoveryFileKind uint8

const (
	providerHistoryRecoveryHead providerHistoryRecoveryFileKind = iota + 1
	providerHistoryRecoveryHighwater
	providerHistoryRecoveryIntent
	providerHistoryRecoveryCommit
	providerHistoryRecoveryFrontier
	providerHistoryRecoveryPending
	providerHistoryRecoveryInventory
	providerHistoryRecoveryObject
)

type providerHistoryRecoveryInputState uint8

const (
	providerHistoryRecoveryWholeInput providerHistoryRecoveryInputState = iota + 1
	providerHistoryRecoveryMissingInput
	providerHistoryRecoveryPartialInput
	providerHistoryRecoveryChangedInput
)

// Inspection DATA, never an ACK, semantic reconstruction or proof of absence
// from the complete catalog. The caller must independently replay the bytes.
type providerHistoryRecoveryInput struct {
	state providerHistoryRecoveryInputState
	raw   []byte
}

func (*providerHistoryRecoveryInput) UnmarshalJSON([]byte) error { return errProviderIntegration }

// An input has one exact original checkpoint role or one selected walk role.
// Neither branch accepts a caller path, root or digest.
type providerHistoryRecoveryInputSelection struct {
	checkpoint *providerHistoryRecoveryCheckpoint
	kind       providerHistoryRecoveryFileKind
	index      int
	inventory  *providerHistoryRecoveryInventoryWalk
	scan       *providerHistoryRecoveryScanWalk
	object     bool
}

func (r *providerHistoryRecoveryOwner) inputPlanLocked(p providerHistoryRecoveryInputSelection) (*retentionDisk, uint64, bool) {
	if p.scan != nil {
		if p.inventory != nil || p.checkpoint != nil || p.kind != 0 || p.index != 0 || p.object {
			return nil, 0, false
		}
		return p.scan.inputPlanLocked(r)
	}
	if p.inventory != nil {
		if p.checkpoint != nil || p.kind != 0 || p.index != 0 {
			return nil, 0, false
		}
		return p.inventory.inputPlanLocked(r, p.object)
	}
	if p.object {
		return nil, 0, false
	}
	return r.checkpointInputPlanLocked(p.checkpoint, p.kind, p.index)
}

func (r *providerHistoryRecoveryOwner) inputNameLocked(p providerHistoryRecoveryInputSelection) string {
	if p.scan != nil {
		return p.scan.inputNameLocked(r)
	}
	if p.inventory != nil {
		return p.inventory.inputNameLocked(r, p.object)
	}
	return r.checkpointInputNameLocked(p.checkpoint, p.kind, p.index)
}

// Fourteen metadata primitives, an allocation of n=maximum+1 bytes and at
// most n bounded content reads. Every read must progress, finish at EOF or
// refuse. Content bytes<=n, read calls<=n and allocation work=n+1, so 3*n+15
// admits this closed file primitive. It is not a whole-recovery/RAM bound.
func providerHistoryRecoveryInputUpper(maximum uint64) (candidateWorkVector, bool) {
	var work candidateWorkVector
	if maximum == 0 || maximum > providerHistoryOriginalPackBytes {
		return work, false
	}
	n, a := providerUpperSum(maximum, 1)
	units, b := providerUpperProduct(n, 3)
	units, c := providerUpperSum(units, 15)
	if !a || !b || !c {
		return work, false
	}
	work[10] = units
	return work, true
}

// No path or root supplied by the caller is accepted. A selected checkpoint
// must be the original pending object or a member of the exact confirmed chain.
func (r *providerHistoryRecoveryOwner) selectedCheckpointLocked(c *providerHistoryRecoveryCheckpoint) bool {
	if r == nil || c == nil || c.owner != c || c.recovery != r || c.group != r.originalGroup || !r.checkpointSelectionLocked(r.originalGroup) {
		return false
	}
	if c == r.pending {
		return true
	}
	for n, count := r.confirmed, uint64(0); n != nil && count <= r.parent.bounds.records; n, count = n.previous, count+1 {
		if n == c {
			return true
		}
		if n == r.initial {
			break
		}
	}
	return false
}

func (r *providerHistoryRecoveryOwner) checkpointInputPlanLocked(c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) (*retentionDisk, uint64, bool) {
	if !r.selectedCheckpointLocked(c) || kind < providerHistoryRecoveryHead || kind > providerHistoryRecoveryHeadPending || index < 0 {
		return nil, 0, false
	}
	if kind > providerHistoryRecoveryObject {
		return r.namespaceInputPlanLocked(c, kind, index)
	}
	maximum := r.config.bounds.maximumMetadataPerCommit
	if kind == providerHistoryRecoveryHead || kind == providerHistoryRecoveryHighwater {
		return r.custody, maximum, index == 0
	}
	if c == r.initial || c.claim == nil || c.frame.after.sequence == 0 {
		return nil, 0, false
	}
	if kind <= providerHistoryRecoveryFrontier {
		return r.custody, maximum, index == 0
	}
	if kind <= providerHistoryRecoveryInventory {
		return r.evidence, maximum, index == 0
	}
	if c.inventory.count <= 0 || c.inventory.count > len(c.inventory.entries) || index >= c.inventory.count {
		return nil, 0, false
	}
	ref := c.inventory.entries[index]
	if _, known := providerHistoryStoredDomain(ref.kind); !known || !lowerHash(ref.digest) || ref.bytes == 0 || ref.bytes > providerHistoryOriginalPackBytes {
		return nil, 0, false
	}
	return r.evidence, ref.bytes, true
}

func (r *providerHistoryRecoveryOwner) checkpointInputNameLocked(c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) string {
	if kind > providerHistoryRecoveryObject {
		return r.namespaceInputNameLocked(c, kind, index)
	}
	if kind == providerHistoryRecoveryHead {
		if !r.meter.event(providerCopyEvent, 4) {
			return ""
		}
		return "HEAD"
	}
	if kind == providerHistoryRecoveryInventory {
		return providerHistoryInventoryName(c.inventory.digest, r.meter)
	}
	if kind == providerHistoryRecoveryObject {
		return providerHistoryStoredName(c.inventory.entries[index], r.meter)
	}
	var name string
	ordinal := c.frame.after.sequence
	switch kind {
	case providerHistoryRecoveryHighwater:
		name, ordinal = "highwater", c.pin.ordinal
	case providerHistoryRecoveryIntent:
		name = "intent"
	case providerHistoryRecoveryCommit:
		name = "commit"
	case providerHistoryRecoveryFrontier:
		name = "frontier"
	case providerHistoryRecoveryPending:
		name = "history-pending"
	default:
		return ""
	}
	// Decimal uint64 always fits twenty places. Charge before fmt allocation.
	if !r.meter.event(providerCopyEvent, uint64(len(name)+21)) {
		return ""
	}
	return providerCustodyName(name, ordinal)
}

func (r *providerHistoryRecoveryOwner) inputReadyLocked(ctx context.Context) bool {
	if r == nil || ctx == nil || ctx.Err() != nil || !r.activeLocked(r.meter) || !r.checkpointSelectionLocked(r.group) || !r.memberSelectionLocked(r.group) ||
		r.inputFiles > r.bounds.maximumFiles || r.inputBytes > r.bounds.maximumBytes {
		return false
	}
	c := r.parent.clock
	c.mu.Lock()
	ok := c.owner == c && c.history == r.parent && c.historyRevoked && r.parent.clockSelectionMatches(c) && c.read >= r.parent.initialRead && c.read <= r.parent.clockCeiling
	c.mu.Unlock()
	return ok
}

func (r *providerHistoryRecoveryOwner) inputIdentityLocked(disk *retentionDisk) bool {
	if disk != r.custody && disk != r.evidence {
		return false
	}
	for range 4 {
		if !r.meter.event(providerReadEvent, 0) {
			return false
		}
	}
	return disk.checkIdentity() == nil
}

// This actual file primitive is reachable only through the original one-use
// read-only lane. It does not install store.activeMeter, use store.read, invoke
// a failed owner, sample native time, reopen a root, write, sync or rename.
func (r *providerHistoryRecoveryOwner) readCheckpointInputCandidate(ctx context.Context, c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) (*providerHistoryRecoveryInput, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	return r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: c, kind: kind, index: index})
}

// Caller holds the original store mutex. Walkers use the same primitive without
// recursively acquiring it or injecting store.activeMeter after failure.
func (r *providerHistoryRecoveryOwner) readSelectedInputLocked(ctx context.Context, selection providerHistoryRecoveryInputSelection) (*providerHistoryRecoveryInput, error) {
	if !r.activeLocked(r.meter) {
		return nil, errProviderIntegration
	}
	disk, maximum, valid := r.inputPlanLocked(selection)
	if !valid {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	upper, ok := r.catalogInputUpperLocked(maximum)
	if !ok || !r.inputReadyLocked(ctx) || !candidateCapacity(0, r.inputFiles, 1, r.bounds.maximumFiles) ||
		!candidateCapacity(0, r.inputBytes, maximum+1, r.bounds.maximumBytes) || !r.meter.preflightWork(upper) {
		return nil, errProviderIntegration
	}
	name := r.inputNameLocked(selection)
	// Filename work also consumes the same cumulative row. Re-admit the whole
	// remaining primitive after that work and before the first native read.
	if name == "" || !r.meter.preflightWork(upper) {
		return nil, errProviderIntegration
	}
	r.inputFiles++ // Missing/partial inputs consume this same finite attempt.
	if !r.inputIdentityLocked(disk) || !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	info, err := disk.root.Lstat(name)
	if !r.catalogInputMatchesLocked(disk, name, info, err) {
		return nil, errProviderIntegration
	}
	var out *providerHistoryRecoveryInput
	switch {
	case errors.Is(err, os.ErrNotExist):
		out = &providerHistoryRecoveryInput{state: providerHistoryRecoveryMissingInput}
	case err != nil:
		return nil, errProviderIntegration
	case !info.Mode().IsRegular() || info.Size() < 0 || uint64(info.Size()) > maximum:
		out = &providerHistoryRecoveryInput{state: providerHistoryRecoveryChangedInput}
	default:
		out, err = r.readOpenedInputLocked(ctx, disk, name, info)
		if err != nil {
			return nil, errProviderIntegration
		}
	}
	if !r.inputIdentityLocked(disk) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	accepted = true
	return out, nil
}

// Native cleanup always closes an opened FD, including a poisoned observer.
// Such failure returns no observation/result; the prospective envelope already
// included cleanup and cannot be refunded or re-created after that failure.
func (r *providerHistoryRecoveryOwner) readOpenedInputLocked(ctx context.Context, disk *retentionDisk, name string, before os.FileInfo) (*providerHistoryRecoveryInput, error) {
	if !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	f, err := disk.root.Open(name)
	if err != nil {
		return nil, errProviderIntegration
	}
	closed := false
	defer func() {
		if !closed {
			_ = r.meter.event(providerReadEvent, 0)
			_ = f.Close()
		}
	}()
	if !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() != before.Size() || !opened.ModTime().Equal(before.ModTime()) {
		return nil, errProviderIntegration
	}
	next := uint64(before.Size()) + 1
	if !r.meter.event(providerCopyEvent, next) {
		return nil, errProviderIntegration
	}
	buffer, used, eof := make([]byte, int(next)), 0, false
	for used < len(buffer) {
		if !r.inputReadyLocked(ctx) || !r.meter.preflight(providerReadEvent, uint64(len(buffer)-used)) {
			return nil, errProviderIntegration
		}
		n, readErr := f.Read(buffer[used:])
		if !r.meter.event(providerReadEvent, uint64(n)) {
			return nil, errProviderIntegration
		}
		r.inputBytes += uint64(n)
		used += n
		if readErr == io.EOF {
			eof = true
			break
		}
		if readErr != nil || n == 0 {
			return nil, errProviderIntegration
		}
	}
	state := providerHistoryRecoveryWholeInput
	if before.Size() == 0 || used < int(before.Size()) {
		state = providerHistoryRecoveryPartialInput
	} else if used != int(before.Size()) || !eof {
		state = providerHistoryRecoveryChangedInput
	}
	if !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	after, err := f.Stat()
	if err != nil || !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	path, err := disk.root.Lstat(name)
	state, err = providerHistoryRecoveryPostPathState(state, before, after, path, err)
	if err != nil {
		return nil, errProviderIntegration
	}
	if !r.meter.event(providerReadEvent, 0) {
		return nil, errProviderIntegration
	}
	err, closed = f.Close(), true
	if err != nil {
		return nil, errProviderIntegration
	}
	return &providerHistoryRecoveryInput{state: state, raw: buffer[:used]}, nil
}

// A disappeared post-read path is uncertain/changed DATA, never proof of an
// attack or complete-catalog absence. Other OS errors are operational failure;
// the caller closes the lane without publishing even an inspection result.
func providerHistoryRecoveryPostPathState(state providerHistoryRecoveryInputState, before, after, path os.FileInfo, err error) (providerHistoryRecoveryInputState, error) {
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return providerHistoryRecoveryChangedInput, nil
		}
		return 0, errProviderIntegration
	}
	if before == nil || after == nil || path == nil {
		return 0, errProviderIntegration
	}
	if !after.Mode().IsRegular() || !path.Mode().IsRegular() || !os.SameFile(before, after) || !os.SameFile(before, path) ||
		after.Size() != before.Size() || path.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !path.ModTime().Equal(before.ModTime()) {
		return providerHistoryRecoveryChangedInput, nil
	}
	return state, nil
}
