package registry011

import (
	"strings"
	"testing"
)

func TestProviderHistoryReconstructionScanKeepsBothOriginalControlAndReceiptJourneys(t *testing.T) {
	// Independent scalar rows for the original call graph: fixed constructor/
	// binding/control work, each historical attempt, and both original copies
	// of each chunk and command. These are DATA, not executed scan results.
	for _, row := range []struct {
		metadata, object, attempts, chunks, policy uint64
		counts                                     [5]uint64
		decoded, copies, logical, incoming         uint64
	}{
		{4096, 4096, 1, 2, 1024, [5]uint64{56, 26, 36, 50, 100}, 106496, 43034, 1376346710, 16777217},
		{65536, 1048576, 4, 32, 8192, [5]uint64{452, 332, 300, 500, 1300}, 21757952, 10502470, 3367350245818, 4294967297},
		{1048576, 33554432, 64, 1024, 65536, [5]uint64{13076, 10372, 8716, 14988, 40980}, 10875830272, 4362217478, 25781393587657850, 1099511627777},
	} {
		limits := providerHistoryLegacyControlLimits{row.metadata, row.object, 4, 64 << 20, 256, row.attempts, row.chunks}
		before := limits
		got, ok := providerHistoryReconstructionScanUpper(limits, row.policy)
		if !ok || limits != before || [5]uint64{got.encodes, got.decodes, got.hashes, got.equalities, got.textEqualities} != row.counts ||
			got.copies != (candidateWorkVector{10: row.copies}) || got.normal != (candidateWorkVector{2: row.decoded, 10: row.logical}) ||
			got.incoming != (candidateWorkVector{2: row.metadata, 10: row.incoming}) ||
			got.envelope != (candidateWorkVector{2: row.decoded + row.metadata, 10: row.logical + row.incoming}) {
			t.Fatal("initial scan omitted a fresh original or borrowed an input/producer row", row, got, ok)
		}
	}
}

func TestProviderHistoryReconstructionScanRefusesIncompleteOrOverflowedSelections(t *testing.T) {
	base := historyLegacyControlLimits()
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "first-objects", "first-raw", "entries", "attempts-zero", "attempts-large", "chunks-zero", "chunks-overflow", "policy-zero", "policy-large"} {
		t.Run(mode, func(t *testing.T) {
			limits, policy := base, uint64(8192)
			switch mode {
			case "metadata-zero":
				limits.metadata = 0
			case "metadata-large":
				limits.metadata = 1<<20 + 1
			case "object-zero":
				limits.objectBytes = 0
			case "object-large":
				limits.objectBytes = 32<<20 + 1
			case "first-objects":
				limits.firstObjects = 0
			case "first-raw":
				limits.firstRaw = 0
			case "entries":
				limits.entries = 0
			case "attempts-zero":
				limits.attempts = 0
			case "attempts-large":
				limits.attempts = 65
			case "chunks-zero":
				limits.chunks = 0
			case "chunks-overflow":
				limits.chunks = ^uint64(0)
			case "policy-zero":
				policy = 0
			case "policy-large":
				policy = 64<<10 + 1
			}
			if got, ok := providerHistoryReconstructionScanUpper(limits, policy); ok || got != (providerHistoryReconstructionScanCost{}) {
				t.Fatal("unfunded original returned a partial scan row", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionScanFundsFreshOriginalBindingCodecsWithoutAuthority(t *testing.T) {
	s, f, originalCodec, binding := providerScanBindingFixture(t)
	raw, err := originalCodec.admissionBinding(binding)
	if err != nil {
		t.Fatal("original closed binding", err)
	}
	config, actual, held, stored := s.config, f.meter.actual, s.scanControl.held, s.stored
	limits := providerHistoryLegacyControlLimits{config.bounds.maximumMetadataPerCommit, config.bounds.maximumObjectBytes,
		1, config.bounds.maximumObjectBytes, uint64(config.bounds.maximumEntries), config.bounds.scan.maximumAttempts, config.bounds.scan.maximumChunks}
	quote, ok := providerHistoryReconstructionScanUpper(limits, uint64(len(config.policy.raw)))
	if !ok {
		t.Fatal("original finite scan limits")
	}
	// Component UNIT only. This is not a recovery/final owner and does not
	// replace the store observer or mint the still-missing full derivation.
	meter := providerHistoryContentUnitMeter(quote.envelope)
	var previous *providerScanCodec
	for pass := 0; pass < 2; pass++ {
		codec, codecErr := newProviderScanCodec(config, meter)
		if codecErr != nil || codec == nil || codec == previous || codec.meter != meter {
			t.Fatal("independent original codec", pass, codecErr)
		}
		got, parseErr := codec.parseAdmissionBinding(providerReplayPhaseBytes(string(raw), meter))
		if parseErr != nil || got != binding {
			t.Fatal("fresh original binding decode/canonical comparison", pass, parseErr)
		}
		previous = codec
	}
	if meter.failed || meter.actual[2] == 0 || meter.actual[10] == 0 || meter.authority || meter.clocks != 0 ||
		s.config != config || f.meter.actual != actual || s.scanControl.held != held || s.stored != stored || s.activeMeter.Load() != nil {
		t.Fatal("component quote renewed an owner, changed originals or acquired execution authority")
	}
	// Exactly one logical unit below the ORIGINAL metadata encode preflight.
	// No encode, hash, replacement allowance or owner is reached after refusal.
	short := providerHistoryContentUnitMeter(candidateWorkVector{2: quote.envelope[2], 10: limits.metadata * limits.metadata})
	if codec, codecErr := newProviderScanCodec(config, short); codecErr == nil || codec != nil || !short.failed || short.actual != (candidateWorkVector{}) || short.clocks != 0 || short.authority {
		t.Fatal("short original constructor guard was bypassed", codecErr)
	}
}

func TestProviderHistoryReconstructionScanRejectsMissingOrUnboundedSelectedPolicy(t *testing.T) {
	config := historyReconstructionSourceConfig(t)
	original := *config.policy
	for _, mode := range []string{"missing", "empty", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			changed := config
			policy := *config.policy
			changed.policy = &policy
			switch mode {
			case "missing":
				changed.policy = nil
			case "empty":
				policy.raw = ""
			case "oversized":
				policy.raw = strings.Repeat("x", 64<<10+1)
			}
			if got, ok := providerHistoryCompileReconstructionSource(changed, false); ok || got != (providerHistoryReconstructionSourceCost{}) || *config.policy != original {
				t.Fatal("unbounded policy copy returned a partial source quote or changed the selected original", mode, ok)
			}
		})
	}
}
