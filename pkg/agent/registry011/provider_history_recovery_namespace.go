package registry011

import "strconv"

const (
	providerHistoryRecoveryAnchorCustody providerHistoryRecoveryFileKind = providerHistoryRecoveryObject + 1 + iota
	providerHistoryRecoveryAnchorEvidence
	providerHistoryRecoveryLegacyIntent
	providerHistoryRecoveryLegacyCommit
	providerHistoryRecoveryLegacyFrontier
	providerHistoryRecoveryLegacyPending
	providerHistoryRecoveryLegacyHighwater
	providerHistoryRecoverySetupIntent
	providerHistoryRecoverySetupCommit
	providerHistoryRecoverySetupClaim
	providerHistoryRecoveryScanHeadCustody
	providerHistoryRecoveryScanHeadEvidence
	providerHistoryRecoveryScanHighwaterCustody
	providerHistoryRecoveryScanHighwaterEvidence
	providerHistoryRecoveryScanIntent
	providerHistoryRecoveryScanCommit
	providerHistoryRecoveryScanClaim
	providerHistoryRecoveryMemberResponse
	providerHistoryRecoveryMemberTLS
	providerHistoryRecoveryScanLastChunkCustody
	providerHistoryRecoveryScanLastChunkEvidence
	providerHistoryRecoveryLockCustody
	providerHistoryRecoveryLockEvidence
	providerHistoryRecoveryInitialInventory
	providerHistoryRecoveryHeadPending
)

// Only an original sealed attempt, including an attempted but unfinished last
// ordinal, may select a member spool. No live coordinator/table cache is used
// to derive its name after failure, and no connection is acquired or replayed.
func (r *providerHistoryRecoveryOwner) inputMemberIntentLocked(c *providerHistoryRecoveryCheckpoint, index int) *providerHistoryRecoveryMemberAttempt {
	if r == nil || c == nil || index < 0 {
		return nil
	}
	var n *providerHistoryRecoveryMemberAttempt
	if c.frame.phase == "settle" {
		sealed := r.settledMemberLocked(c, index)
		if sealed == nil {
			return nil
		}
		n = sealed.capture.intent
	} else {
		if c != r.consumption || !r.memberSelectionLocked(r.originalGroup) {
			return nil
		}
		if index < r.completed {
			n = r.members[index].capture.intent
		} else if index == r.completed {
			n = r.memberAttempt
		}
	}
	if n == nil || n.ordinal != index || n.count <= index || n.stage != c.frame.stage || !n.member.capture.valid() || len(n.member.request.wire) >= n.member.capture.WireBytes {
		return nil
	}
	return n
}

func (r *providerHistoryRecoveryOwner) namespaceInputPlanLocked(c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) (*retentionDisk, uint64, bool) {
	maximum := r.config.bounds.maximumMetadataPerCommit
	if kind == providerHistoryRecoveryHeadPending {
		return r.custody, maximum, c != nil && c == r.pending && index == 0
	}
	if kind == providerHistoryRecoveryMemberResponse || kind == providerHistoryRecoveryMemberTLS {
		n := r.inputMemberIntentLocked(c, index)
		if n == nil {
			return nil, 0, false
		}
		maximum = uint64(n.member.capture.WireBytes - len(n.member.request.wire))
		if kind == providerHistoryRecoveryMemberTLS {
			maximum = min(r.config.bounds.maximumObjectBytes, uint64(64<<10))
		}
		return r.evidence, maximum, maximum > 0 && maximum <= providerHistoryOriginalPackBytes
	}
	if c != r.initial || !r.initialDescriptorShapeLocked(r.originalGroup) || index < 0 {
		return nil, 0, false
	}
	n, value := r.initialDescriptor, uint64(index)
	disk, valid := r.custody, false
	switch kind {
	case providerHistoryRecoveryAnchorCustody, providerHistoryRecoveryScanHeadCustody, providerHistoryRecoveryScanLastChunkCustody:
		valid = index == 0
	case providerHistoryRecoveryAnchorEvidence, providerHistoryRecoveryScanHeadEvidence, providerHistoryRecoveryScanLastChunkEvidence:
		disk, valid = r.evidence, index == 0
	case providerHistoryRecoveryLegacyIntent, providerHistoryRecoveryLegacyCommit, providerHistoryRecoveryLegacyFrontier:
		valid = value > 0 && value <= n.main.sequence
	case providerHistoryRecoveryLegacyPending:
		disk, valid = r.evidence, value > 0 && value <= n.main.sequence
	case providerHistoryRecoveryLegacyHighwater:
		valid = value <= n.main.ordinal
	case providerHistoryRecoverySetupIntent, providerHistoryRecoverySetupCommit:
		valid = value < n.setups
	case providerHistoryRecoverySetupClaim:
		disk, valid = r.evidence, value < n.setups
	case providerHistoryRecoveryScanHighwaterCustody:
		valid = value <= n.scan.ordinal
	case providerHistoryRecoveryScanHighwaterEvidence:
		disk, valid = r.evidence, value <= n.scan.ordinal
	case providerHistoryRecoveryScanIntent, providerHistoryRecoveryScanCommit:
		valid = value > 0 && value <= n.scan.ordinal
	case providerHistoryRecoveryScanClaim:
		disk, valid = r.evidence, value > 0 && value <= n.scan.ordinal
	case providerHistoryRecoveryInitialInventory:
		disk, maximum, valid = r.evidence, 64<<10, index == 0 && lowerHash(n.objects)
	case providerHistoryRecoveryLockCustody:
		maximum, valid = 1, index == 0
	case providerHistoryRecoveryLockEvidence:
		disk, maximum, valid = r.evidence, 1, index == 0
	}
	return disk, maximum, valid
}

func (r *providerHistoryRecoveryOwner) namespaceInputNameLocked(c *providerHistoryRecoveryCheckpoint, kind providerHistoryRecoveryFileKind, index int) string {
	if kind == providerHistoryRecoveryHeadPending {
		if c == nil || c != r.pending || index != 0 {
			return ""
		}
		return r.catalogOrdinalNameLocked("HEAD.pending", c.pin.ordinal)
	}
	if kind == providerHistoryRecoveryMemberResponse || kind == providerHistoryRecoveryMemberTLS {
		n := r.inputMemberIntentLocked(c, index)
		if n == nil || n.consumption == nil || !lowerHash(n.consumption.pin.commit) {
			return ""
		}
		part, expected := "response", n.responseName
		if kind == providerHistoryRecoveryMemberTLS {
			part, expected = "tls", n.tlsName
		}
		name := providerReplayJoin([]string{"history-original-", n.consumption.pin.commit[2:], "-", strconv.FormatUint(n.stage, 10), "-", strconv.Itoa(index), "-", part}, r.meter)
		if name == "" || !providerReplayTextEqual(name, expected, r.meter) {
			return ""
		}
		return name
	}
	n := r.initialDescriptor
	if kind == providerHistoryRecoveryScanLastChunkCustody || kind == providerHistoryRecoveryScanLastChunkEvidence {
		return providerScanReceiptName("chunk", n.lastChunk, r.meter)
	}
	if kind == providerHistoryRecoveryInitialInventory {
		return providerReplayJoin([]string{"inventory-", n.objects[2:]}, r.meter)
	}
	var fixed, prefix string
	switch kind {
	case providerHistoryRecoveryAnchorCustody, providerHistoryRecoveryAnchorEvidence:
		fixed = "ANCHOR"
	case providerHistoryRecoveryScanHeadCustody, providerHistoryRecoveryScanHeadEvidence:
		fixed = "SCAN-HEAD"
	case providerHistoryRecoveryLockCustody, providerHistoryRecoveryLockEvidence:
		fixed = "AUTHORITY.LOCK"
	case providerHistoryRecoveryLegacyIntent:
		prefix = "intent"
	case providerHistoryRecoveryLegacyCommit:
		prefix = "commit"
	case providerHistoryRecoveryLegacyFrontier:
		prefix = "frontier"
	case providerHistoryRecoveryLegacyPending:
		prefix = "pending"
	case providerHistoryRecoveryLegacyHighwater:
		prefix = "highwater"
	case providerHistoryRecoverySetupIntent:
		prefix = "setup-intent"
	case providerHistoryRecoverySetupCommit:
		prefix = "setup-commit"
	case providerHistoryRecoverySetupClaim:
		prefix = "setup-claim"
	case providerHistoryRecoveryScanHighwaterCustody, providerHistoryRecoveryScanHighwaterEvidence:
		prefix = "scan-highwater"
	case providerHistoryRecoveryScanIntent:
		prefix = "scan-intent"
	case providerHistoryRecoveryScanCommit:
		prefix = "scan-commit"
	case providerHistoryRecoveryScanClaim:
		prefix = "scan-claim"
	default:
		return ""
	}
	if fixed != "" {
		if !r.meter.event(providerCopyEvent, uint64(len(fixed))) {
			return ""
		}
		return fixed
	}
	if !r.meter.event(providerCopyEvent, uint64(len(prefix)+21)) {
		return ""
	}
	return providerCustodyName(prefix, uint64(index))
}
