package registry011

import (
	"encoding/json"
	"reflect"
	"unicode/utf8"
)

// This is an observed encoder preflight, not a new journal wire/storage limit.
// The original finite configuration has six objects, 36 fixed keys (387 raw
// key bytes), 12 string values (at most1498 raw UTF8 bytes) and19 numbers
// (at most20 decimal bytes each). JSON syntax uses174 bytes and escaping at
// most6 bytes per input byte:174+387+6*1498+19*20=9929 <65536.
// A nil observer preserves the original journal validation and bytes exactly.
const providerHistoryJournalEncodeLimit uint64 = 64 << 10

func journalConfigurationObserved(c JournalConfig, meter *providerWorkMeter) ([]byte, *ObservationChecks, error) {
	if meter != nil && !meter.event(providerCopyEvent, uint64(reflect.TypeFor[ObservationChecks]().Size())) {
		return nil, nil, ErrConfiguration
	}
	checks, err := NewObservationChecks(c.Observation)
	if err != nil || c.Registry != "eip155:11155111:"+SepoliaRegistryAddress || !rpcAgent(c.Agent) || c.GenesisHash != sepoliaGenesis || !lowerHash(c.CodeIdentity) || !lowerHash(c.ABIIdentity) ||
		len(c.SourceID) == 0 || len(c.SourceID) > 256 || len(c.PolicyGeneration) == 0 || len(c.PolicyGeneration) > 256 || c.PolicyGeneration != c.Observation.PolicyGeneration || len(c.Observation.ClockID) > 256 || !validHeader(c.Deployment) || !replayLimits(c.Replay) ||
		!utf8.ValidString(c.SourceID) || !utf8.ValidString(c.PolicyGeneration) || !utf8.ValidString(c.Observation.ClockID) ||
		c.Observation.MaxAncestryHeaders > 1<<20 || c.MaxBatches <= 0 || c.MaxBatches > 1<<20 || c.MaxClaims <= 0 || c.MaxClaims > 1<<20 || c.MaxIdentities <= 0 || c.MaxIdentities > 1<<20 || c.MaxStoredBytes <= 0 || c.MaxStoredBytes > 1<<40 {
		return nil, nil, ErrConfiguration
	}
	if meter != nil && !meter.preflight(providerEncodeEvent, providerHistoryJournalEncodeLimit) {
		return nil, nil, ErrConfiguration
	}
	b, err := json.Marshal(struct {
		Format string
		Config JournalConfig
	}{"sage-replay-journal-v1", c})
	// Share a reachable refusal with the original storage check. The closed
	// scalar schema cannot produce Marshal errors/oversize bytes, but those
	// guards remain without artificial faults or separate unreachable blocks.
	if err != nil || meter != nil && (uint64(len(b)) > providerHistoryJournalEncodeLimit || !meter.event(providerEncodeEvent, uint64(len(b)))) ||
		int64(len(b)) > c.MaxStoredBytes {
		return nil, nil, ErrConfiguration
	}
	return b, checks, nil
}
