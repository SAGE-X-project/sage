package registry011

import (
	"context"
	"reflect"
	"strconv"
)

// These rows record the actual ordered file inputs of ONE complete original
// replay. Duplicates and original read maxima are retained. They are DATA, not
// permissions, an independent replay, a cost certificate or a replacement for
// interpreting originals. A separately registered same-C member binding may
// reuse the full-pass result only after fresh ordered original comparisons.
type providerHistoryReplayRead struct {
	root           uint8
	name, digest   string
	maximum, bytes uint64
}

type providerHistoryReplayReads struct {
	owner            *providerHistoryReplayReads
	group            *providerHistoryGroupOwner
	meter            *providerWorkMeter
	context          context.Context
	role             uint8
	config           providerCustodyConfig
	program, costs   string
	initial, current providerFrontierHighwater
	cutoffPin        providerFrontierHighwater
	cutoff           *providerHistoryRecoveryCheckpoint
	ack              *providerHistoryGroupAcknowledgment
	maximum          uint64
	rows             []providerHistoryReplayRead
	model, root      string
	complete, failed bool
}

func (*providerHistoryReplayReads) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Limits below cap recording DATA in the already held lifetime. They do not
// assert that every possible full replay fits this many inputs or that Go
// backing allocations fit resident RAM. Those are whole-source compiler rows.
func newProviderHistoryReplayReadsLocked(ctx context.Context, o *providerHistoryGroupOwner, role uint8) (*providerHistoryReplayReads, error) {
	if ctx == nil || ctx.Err() != nil || o == nil || role > 1 || o.phase != "preparing-dispatch" ||
		!o.activeLocked(o.meter) || !o.committedConsumptionShapeLocked() || o.meter.replayReads != nil ||
		o.store.config != o.parent.config {
		return nil, errProviderIntegration
	}
	first := uint64(0)
	if o.parent.bounds.collection != nil {
		first = o.parent.collectionInput.firstRead.objects
	}
	maximum, ok := providerUpperSum(o.parent.bounds.catalogEntries, first, 128)
	if !ok || maximum == 0 || maximum > uint64(^uint(0)>>1) ||
		!o.meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryReplayReads]().Size())) {
		return nil, errProviderIntegration
	}
	p := &providerHistoryReplayReads{group: o, meter: o.meter, context: ctx, role: role,
		config: o.store.config, initial: o.start.main, current: o.pin,
		program: o.parent.bounds.program, costs: o.parent.bounds.costs, cutoffPin: o.stageBefore.pin,
		cutoff: o.stageBefore, ack: o.last, maximum: maximum}
	p.owner = p
	return p, nil
}

func (p *providerHistoryReplayReads) recordingLocked(s *providerCustodyStore) bool {
	return p != nil && s != nil && p.owner == p && !p.failed && !p.complete && p.context != nil && p.context.Err() == nil &&
		p.group != nil && p.group.store == s && p.group.phase == "preparing-dispatch" &&
		p.group.meter == p.meter && p.meter != nil && p.meter.replayReads == p && s.activeMeter.Load() == p.meter &&
		p.group.activeLocked(p.meter) && p.group.committedConsumptionShapeLocked() &&
		p.config == s.config && p.config == p.group.parent.config && p.initial == p.group.start.main &&
		p.current == p.group.pin && p.cutoff == p.group.stageBefore && p.cutoff != nil && p.cutoffPin == p.cutoff.pin &&
		p.ack == p.group.last && p.program == p.group.parent.bounds.program && p.costs == p.group.parent.bounds.costs && p.role <= 1 &&
		p.maximum > 0 && uint64(len(p.rows)) <= p.maximum
}

func (p *providerHistoryReplayReads) refuse() bool {
	if p != nil && p.owner == p {
		p.failed = true
	}
	return false
}

// Called after the actual store.read, including every readExact. No callback,
// producer list or reconstructed filename list supplies a row. A read/hash or
// copy refusal retains the original meter's consumed prefix and invalidates
// the incomplete recording; it never completes a seal from a failed replay.
func (p *providerHistoryReplayReads) recordLocked(s *providerCustodyStore, disk *retentionDisk, name string, maximum int64, raw []byte) bool {
	if !p.recordingLocked(s) || maximum <= 0 || len(name) == 0 || len(name) > 255 ||
		len(raw) == 0 || uint64(len(raw)) > uint64(maximum) || uint64(len(p.rows)) >= p.maximum {
		return p.refuse()
	}
	root := uint8(0)
	switch disk {
	case s.custody:
	case s.evidence:
		root = 1
	default:
		return p.refuse()
	}
	m := p.meter
	digest := providerReplayDigest("", raw, m)
	owned := providerReplayString(providerReplayBytes(name, m), m)
	// This is an explicit logical descriptor-copy event, not sizeof, malloc
	// accounting or a native resident-memory bound for append/backing arrays.
	if digest == "" || owned == "" || !m.event(providerCopyEvent, 128) || !p.recordingLocked(s) {
		return p.refuse()
	}
	p.rows = append(p.rows, providerHistoryReplayRead{root: root, name: owned, digest: digest,
		maximum: uint64(maximum), bytes: uint64(len(raw))})
	return true
}

// A rolling commitment keeps each canonical input bounded; a large ordered
// transcript is never submitted as one unbounded JSON/encoder allocation.
// Nil observation is allowed only for ordinary DATA inspection, not capture.
func (p *providerHistoryReplayReads) dataRootObserved(m *providerWorkMeter) string {
	if p == nil || p.owner != p || p.group == nil || p.group.parent == nil || p.cutoff == nil || p.ack == nil ||
		p.role > 1 || p.maximum == 0 || len(p.rows) == 0 || uint64(len(p.rows)) > p.maximum || !lowerHash(p.model) {
		return ""
	}
	v := map[string]any{"format": "sage-provider-history-replay-reads-v1", "role": strconv.Itoa(int(p.role)),
		"program": p.program, "profile": p.config.profile, "cost_revision": p.costs,
		"initial_frontier": p.initial.frontier, "initial_commit": p.initial.commit,
		"current_frontier": p.current.frontier, "current_commit": p.current.commit,
		"cutoff_frontier": p.cutoffPin.frontier, "cutoff_commit": p.cutoffPin.commit,
		"model": p.model, "rows": strconv.Itoa(len(p.rows)), "maximum_rows": strconv.FormatUint(p.maximum, 10)}
	raw := providerReplayCanonical(v, candidateLedgerMetadataLimit, m)
	root := providerReplayDigest("sage-provider-history-replay-reads-v1", raw, m)
	if len(raw) == 0 || root == "" {
		return ""
	}
	for ordinal, row := range p.rows {
		if row.root > 1 || row.name == "" || len(row.name) > 255 || row.maximum == 0 || row.bytes == 0 ||
			row.bytes > row.maximum || !lowerHash(row.digest) {
			return ""
		}
		raw = providerReplayCanonical(map[string]any{"previous": root, "ordinal": strconv.Itoa(ordinal),
			"root": strconv.Itoa(int(row.root)), "name": row.name, "maximum": strconv.FormatUint(row.maximum, 10),
			"bytes": strconv.FormatUint(row.bytes, 10), "digest": row.digest}, candidateLedgerMetadataLimit, m)
		root = providerReplayDigest("sage-provider-history-replay-read-v1", raw, m)
		if len(raw) == 0 || root == "" {
			return ""
		}
	}
	return root
}

func (o *providerHistoryGroupOwner) replayRecordedConsumptionLocked(ctx context.Context, role uint8) (*providerHistoryGroupLedger, *providerHistoryReplayReads, error) {
	p, err := newProviderHistoryReplayReadsLocked(ctx, o, role)
	if err != nil {
		return nil, nil, errProviderIntegration
	}
	m := o.meter
	m.replayReads = p
	defer func() { m.replayReads = nil }()
	model, err := o.replayCommittedConsumptionLocked(ctx)
	if err != nil || model == nil || !p.recordingLocked(o.store) || !model.validObserved(m) ||
		model.head != o.coordinator.head || model.head != o.custodian.head || len(p.rows) == 0 {
		p.refuse()
		return nil, nil, errProviderIntegration
	}
	p.model = model.head
	p.root = p.dataRootObserved(m)
	if p.root == "" || !p.recordingLocked(o.store) {
		p.refuse()
		return nil, nil, errProviderIntegration
	}
	p.context = nil // A completed DATA transcript does not retain a caller context graph.
	p.complete = true
	return model, p, nil
}
