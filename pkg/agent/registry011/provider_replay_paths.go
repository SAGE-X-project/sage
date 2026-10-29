package registry011

// Byte and filename copies are distinct invocations from the later read,
// codec or hash. They spend the store's current phase, never a cached meter.
func providerReplayBytes(value string, meter *providerWorkMeter) []byte {
	if meter != nil && !meter.event(providerCopyEvent, uint64(len(value))) {
		return nil
	}
	return []byte(value)
}

func (s *providerCustodyStore) name(kind string, index uint64) string {
	if kind == "" || len(kind) > 64 {
		return ""
	}
	if meter := s.copyMeter(); meter != nil && !meter.event(providerCopyEvent, uint64(len(kind)+21)) {
		return ""
	}
	return providerCustodyName(kind, index)
}

func (s *providerCustodyStore) objectName(kind, digest string) string {
	if _, known := providerIntegrationDomain(kind); !known || !lowerHash(digest) {
		return ""
	}
	return providerReplayJoin([]string{"object-", kind, "-", digest[2:]}, s.copyMeter())
}

func (s *providerCustodyStore) inventoryName(digest string) string {
	if !lowerHash(digest) {
		return ""
	}
	return providerReplayJoin([]string{"inventory-", digest[2:]}, s.copyMeter())
}

// The new copy-accounting revision is committed before V4 H0. Earlier
// profiles keep their existing measurement behavior and exact filename bytes.
func (s *providerCustodyStore) copyMeter() *providerWorkMeter {
	if s.config.bounds.firstAdmission == nil {
		return nil
	}
	return s.activeMeter.Load()
}

// Shared legacy DATA codecs receive the extra copy events only when their
// concrete current owner belongs to the new pre-H0 V4 accounting revision.
func providerReplayPhaseBytes(value string, meter *providerWorkMeter) []byte {
	var store *providerCustodyStore
	if meter != nil {
		if meter.scan != nil {
			store = meter.scan.store
		} else if meter.integration != nil {
			store = meter.integration.store
		} else if meter.guardian != nil {
			store = meter.guardian.store
		} else if meter.historyGroup != nil {
			store = meter.historyGroup.store
		}
	}
	if store == nil || store.config.bounds.firstAdmission == nil {
		return []byte(value)
	}
	return providerReplayBytes(value, meter)
}

func providerReplayPhaseTextEqual(left, right string, meter *providerWorkMeter) bool {
	a := providerReplayPhaseBytes(left, meter)
	b := providerReplayPhaseBytes(right, meter)
	return providerReplayEqual(a, b, meter)
}

// A V4 original-byte/private-text comparison pays for the real text copy and
// comparison. Earlier callers preserve their existing zero-extra-event path.
func providerReplayPhaseRawTextEqual(raw []byte, text string, meter *providerWorkMeter) bool {
	var store *providerCustodyStore
	if meter != nil {
		if meter.scan != nil {
			store = meter.scan.store
		} else if meter.integration != nil {
			store = meter.integration.store
		} else if meter.guardian != nil {
			store = meter.guardian.store
		} else if meter.historyGroup != nil {
			store = meter.historyGroup.store
		}
	}
	if store == nil || store.config.bounds.firstAdmission == nil {
		return string(raw) == text
	}
	return providerReplayEqual(raw, providerReplayBytes(text, meter), meter)
}
