package registry011

import (
	"context"
	"errors"
	"io"
	"os"
)

type providerHistoryRecoveryCatalogEntry struct {
	evidence bool
	name     string
	info     os.FileInfo
}

// Observed names are a bounded comparison subject. They never become file
// selectors or the allowed namespace. Only original closed roles open files.
type providerHistoryRecoveryCatalog struct {
	owner    *providerHistoryRecoveryCatalog
	recovery *providerHistoryRecoveryOwner
	entries  []providerHistoryRecoveryCatalogEntry
	roots    [2]os.FileInfo
	names    uint64
	complete bool
}

// A namespace closure witness only, not complete recovery, an ACK or a new
// execution allowance. Any later observed work makes this witness stale.
type providerHistoryRecoveryCatalogResult struct {
	owner          *providerHistoryRecoveryCatalogResult
	recovery       *providerHistoryRecoveryOwner
	before, after  *providerHistoryRecoveryCatalog
	files, bytes   uint64
	work           candidateWorkVector
	entries, names uint64
}

func (*providerHistoryRecoveryCatalog) UnmarshalJSON([]byte) error { return errProviderIntegration }
func (*providerHistoryRecoveryCatalogResult) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (c *providerHistoryRecoveryCatalog) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return c != nil && c.owner == c && c.recovery == r && r != nil && c.complete &&
		(c == r.catalogBefore || c == r.catalogAfter) && len(c.entries) > 0 && uint64(cap(c.entries)) == r.parent.bounds.catalogEntries &&
		uint64(len(c.entries)) <= r.parent.bounds.catalogEntries && c.names <= r.parent.bounds.catalogNameBytes && c.roots[0] != nil && c.roots[1] != nil
}

func (c *providerHistoryRecoveryCatalogResult) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return c != nil && c.owner == c && c.recovery == r && r != nil && r.catalogResult == c && c.before == r.catalogBefore && c.after == r.catalogAfter &&
		c.before.selectedLocked(r) && c.after.selectedLocked(r) && c.files == r.inputFiles && c.bytes == r.inputBytes && c.work == r.meter.actual
}

// Bound this component's four simultaneous metadata/name slot arrays before
// allocation. This profile slot allowance is not whole-program physical RAM.
func (r *providerHistoryRecoveryOwner) catalogSlotsLocked() bool {
	n := r.parent.bounds.catalogEntries
	entry, a := providerUpperSum(r.config.bounds.scan.filenameEntryOverhead, 255)
	slots, b := providerUpperProduct(n, 4)
	bytes, c := providerUpperProduct(slots, entry)
	return a && b && c && n > 0 && n <= uint64(^uint(0)>>1) && bytes <= r.parent.bounds.resident
}

func (r *providerHistoryRecoveryOwner) beginCatalogCandidate(ctx context.Context) (*providerHistoryRecoveryCatalog, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || r.catalogBefore != nil || r.catalogAfter != nil || r.catalogResult != nil || r.inputFiles != 0 || r.inputBytes != 0 ||
		r.inventoryWalks != ([2]*providerHistoryRecoveryInventoryWalk{}) || r.scanWalks != ([2]*providerHistoryRecoveryScanWalk{}) ||
		r.legacyPrefixes != ([2]*providerHistoryRecoveryLegacyPrefix{}) || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
		r.groupModels != ([2]*providerHistoryRecoveryGroupModel{}) || r.completedMemberModels != ([2]*providerHistoryRecoveryCompletedMembers{}) ||
		r.tailModels != ([2]*providerHistoryRecoveryTailPayloads{}) || r.tailSeals != ([2]providerHistoryRecoveryTailData{}) {
		return nil, errProviderIntegration
	}
	out, err := r.readCatalogLocked(ctx)
	if err != nil {
		r.stopLocked(true)
		return nil, errProviderIntegration
	}
	r.catalogBefore = out
	return out, nil
}

func (r *providerHistoryRecoveryOwner) readCatalogLocked(ctx context.Context) (*providerHistoryRecoveryCatalog, error) {
	if !r.inputReadyLocked(ctx) || !r.catalogSlotsLocked() || !r.meter.event(providerCopyEvent, r.parent.bounds.catalogEntries) {
		return nil, errProviderIntegration
	}
	c := &providerHistoryRecoveryCatalog{recovery: r, entries: make([]providerHistoryRecoveryCatalogEntry, 0, int(r.parent.bounds.catalogEntries))}
	c.owner = c
	for root, disk := range [2]*retentionDisk{r.custody, r.evidence} {
		if c.readRootLocked(ctx, root, disk) != nil {
			return nil, errProviderIntegration
		}
	}
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	c.complete = true
	return c, nil
}

// Cleanup remains native even after observation fails. Each next native step
// has its own prospective allowance; failure cannot refund or reopen the lane.
func (c *providerHistoryRecoveryCatalog) readRootLocked(ctx context.Context, root int, disk *retentionDisk) error {
	r := c.recovery
	var open candidateWorkVector
	open[10] = 4 // open, before/after Stat, close
	if !r.inputReadyLocked(ctx) || !r.inputIdentityLocked(disk) || !r.meter.preflightWork(open) || !r.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	f, err := disk.root.Open(".")
	if err != nil {
		return errProviderIntegration
	}
	closed := false
	defer func() {
		if !closed {
			_ = r.meter.event(providerReadEvent, 0)
			_ = f.Close()
		}
	}()
	if !r.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	before, err := f.Stat()
	if err != nil || !before.IsDir() || !os.SameFile(before, disk.directoryInfo) {
		return errProviderIntegration
	}
	for {
		// Reserve an entry's read, lstat, bounded name copy, and every
		// duplicate-name comparison, plus native close and directory Stat.
		comparisons, a := providerUpperProduct(uint64(len(c.entries)), 511)
		units, b := providerUpperSum(comparisons, 3*256, 8)
		var upper candidateWorkVector
		upper[10] = units
		if !a || !b || !r.inputReadyLocked(ctx) || !r.meter.preflightWork(upper) || !r.meter.event(providerReadEvent, 0) {
			return errProviderIntegration
		}
		entries, readErr := f.ReadDir(1)
		if readErr == io.EOF && len(entries) == 0 {
			break
		}
		if readErr != nil || len(entries) != 1 || uint64(len(c.entries)) >= r.parent.bounds.catalogEntries {
			return errProviderIntegration
		}
		name := entries[0].Name()
		if name == "" || len(name) > 255 || !candidateCapacity(0, c.names, uint64(len(name)), r.parent.bounds.catalogNameBytes) {
			return errProviderIntegration
		}
		if !r.meter.event(providerReadEvent, uint64(len(name))) {
			return errProviderIntegration
		}
		for _, prior := range c.entries {
			if !r.meter.event(providerCompareEvent, uint64(len(prior.name)+len(name))) || prior.evidence == (root == 1) && prior.name == name {
				return errProviderIntegration
			}
		}
		if !r.meter.event(providerReadEvent, 0) {
			return errProviderIntegration
		}
		info, err := disk.root.Lstat(name)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 {
			return errProviderIntegration
		}
		copy := providerHistoryRecoveryCloneObserved(name, r.meter)
		if copy == "" || !r.meter.event(providerCopyEvent, 1) {
			return errProviderIntegration
		}
		c.entries = append(c.entries, providerHistoryRecoveryCatalogEntry{evidence: root == 1, name: copy, info: info})
		c.names += uint64(len(name))
	}
	if !r.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	after, err := f.Stat()
	if err != nil || !providerHistoryRecoverySameMetadata(before, after) || !r.meter.event(providerReadEvent, 0) {
		return errProviderIntegration
	}
	err, closed = f.Close(), true
	if err != nil || !r.inputIdentityLocked(disk) || !r.inputReadyLocked(ctx) {
		return errProviderIntegration
	}
	c.roots[root] = after
	return nil
}

func providerHistoryRecoverySameMetadata(a, b os.FileInfo) bool {
	return a != nil && b != nil && os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (r *providerHistoryRecoveryOwner) catalogInputUpperLocked(maximum uint64) (candidateWorkVector, bool) {
	upper, ok := providerHistoryRecoveryInputUpper(maximum)
	if !ok || r.catalogBefore == nil {
		return upper, ok
	}
	if !r.catalogBefore.selectedLocked(r) {
		return candidateWorkVector{}, false
	}
	comparisons, a := providerUpperProduct(uint64(len(r.catalogBefore.entries)), 511)
	upper[10], ok = providerUpperSum(upper[10], comparisons, 1)
	return upper, a && ok
}

func (r *providerHistoryRecoveryOwner) catalogInputMatchesLocked(disk *retentionDisk, name string, info os.FileInfo, err error) bool {
	if r.catalogBefore == nil {
		return true
	}
	if !r.catalogBefore.selectedLocked(r) || disk != r.custody && disk != r.evidence || len(name) > 255 {
		return false
	}
	for _, entry := range r.catalogBefore.entries {
		if !r.meter.event(providerCompareEvent, uint64(len(entry.name)+len(name))) {
			return false
		}
		if entry.evidence == (disk == r.evidence) && entry.name == name {
			return err == nil && r.meter.event(providerCompareEvent, 0) && providerHistoryRecoverySameMetadata(entry.info, info)
		}
	}
	return errors.Is(err, os.ErrNotExist)
}
