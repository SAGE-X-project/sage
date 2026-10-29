package registry011

import (
	"bytes"
	"strconv"
)

const providerHistoryGroupTicketDomain = "sage-provider-history-group-ticket-v1"
const providerHistoryGroupConsumptionDomain = "sage-provider-history-group-consumption-v1"
const providerHistoryGroupSettlementDomain = "sage-provider-history-group-settlement-v1"

// This is the new group's typed reservation DATA, not an old TicketV2 or a
// live whole-program hold. The future owner must derive every bound from its
// concrete program before H0. Neither a positive vector nor parsing this data
// supplies that derivation, retained originals, an ACK or a dispatch permit.
type providerHistoryGroupUpper struct {
	execution, custody                       candidateWorkVector
	physical, retainedRaw, payload, metadata uint64
	objects, files, clocks                   uint64
}

func (u providerHistoryGroupUpper) work() (candidateWorkVector, bool) {
	var total candidateWorkVector
	for i, n := range u.execution {
		value, ok := providerUpperSum(n, u.custody[i])
		if !ok {
			return candidateWorkVector{}, false
		}
		total[i] = value
	}
	return total, true
}

func (u providerHistoryGroupUpper) valid(count int) bool {
	_, ok := u.work()
	return ok && count > 0 && count <= providerHistoryGroupMembers &&
		u.execution[0] == uint64(count) && u.execution[1] > 0 && u.execution[10] > 0 &&
		u.custody[0] == 0 && u.custody[1] == 0 && u.custody[10] > 0 &&
		u.physical > 0 && u.physical <= 1<<40 && u.retainedRaw > 0 && u.retainedRaw <= u.physical &&
		u.payload > 0 && u.payload <= 16<<30 && u.payload <= u.retainedRaw &&
		u.metadata > 0 && u.metadata <= 64<<30 && u.metadata <= u.retainedRaw &&
		u.objects > 0 && u.objects <= 1<<20 && u.files >= u.objects && u.files <= u.physical &&
		u.clocks > 0 && u.clocks <= 1<<20
}

func (u providerHistoryGroupUpper) object() map[string]any {
	return map[string]any{
		"execution": candidateLedgerWorkObject(u.execution), "custody": candidateLedgerWorkObject(u.custody),
		"physical_bytes": strconv.FormatUint(u.physical, 10), "retained_raw_bytes": strconv.FormatUint(u.retainedRaw, 10),
		"payload_bytes": strconv.FormatUint(u.payload, 10), "objects": strconv.FormatUint(u.objects, 10),
		"metadata_bytes": strconv.FormatUint(u.metadata, 10),
		"files":          strconv.FormatUint(u.files, 10), "clock_reads": strconv.FormatUint(u.clocks, 10),
	}
}

// All records bind a separate private program/profile/cost revision. Request
// cardinality is the exact table count; the old single-request ticket contract
// is not broadened. The retained inventory digest is DATA to be independently
// reconstructed by the real custodian, never accepted as an ACK by this codec.
type providerHistoryGroupTicket struct {
	program, profile, costs, previous, inventory string
	table                                        providerHistoryGroupTable
	upper                                        providerHistoryGroupUpper
	raw, digest                                  string
}

func (*providerHistoryGroupTicket) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryGroupTicket(program, profile, costs, previous, inventory string, table *providerHistoryGroupTable, upper providerHistoryGroupUpper) (*providerHistoryGroupTicket, error) {
	if !lowerHash(program) || !lowerHash(profile) || !lowerHash(costs) || !lowerHash(previous) || !lowerHash(inventory) ||
		table == nil || table.program != program || table.stage == 0 || table.stage > 1<<53 || !lowerHash(table.digest) || !upper.valid(table.count) {
		return nil, ErrConfiguration
	}
	switch table.purpose {
	case "selection", "identity", "ancestry", "content", "effects", "state", "history", "final":
	default:
		return nil, ErrConfiguration
	}
	// Reconstruct the table from the exact immutable requests. A changed table
	// field with its former digest cannot enter a new typed ticket.
	// Table construction fixes capture per member. Its raw/digest alone is not
	// a substitute for reconstruction; compare the closed representation here.
	rawTable := providerIntegrationJSON(table.object())
	if len(rawTable) == 0 || len(rawTable) > providerHistoryGroupTableBytes ||
		!bytes.Equal(rawTable, []byte(table.raw)) || providerCandidateHash(providerHistoryGroupTableDomain, rawTable) != table.digest {
		return nil, ErrConfiguration
	}
	for i := range table.count {
		request := &table.members[i].request
		copy, err := providerHistoryGroupRequestCopy(request)
		class, classErr := providerHistoryGroupRequestClass(request)
		if err != nil || classErr != nil || copy != *request || class != table.members[i].class || !table.members[i].capture.valid() {
			return nil, ErrConfiguration
		}
	}
	t := &providerHistoryGroupTicket{program: program, profile: profile, costs: costs, previous: previous, inventory: inventory, table: *table, upper: upper}
	raw := providerIntegrationJSON(t.object())
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, ErrConfiguration
	}
	t.raw, t.digest = string(raw), providerCandidateHash(providerHistoryGroupTicketDomain, raw)
	return t, nil
}

func providerHistoryGroupTicketValid(t *providerHistoryGroupTicket) bool {
	if t == nil {
		return false
	}
	copy, err := newProviderHistoryGroupTicket(t.program, t.profile, t.costs, t.previous, t.inventory, &t.table, t.upper)
	return err == nil && *copy == *t
}

func (t *providerHistoryGroupTicket) object() map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"format": providerHistoryGroupTicketDomain, "program": t.program, "profile": t.profile,
		"cost_revision": t.costs, "previous": t.previous, "inventory": t.inventory,
		"stage": strconv.FormatUint(t.table.stage, 10), "table": t.table.digest,
		"members": strconv.Itoa(t.table.count), "upper": t.upper.object(),
	}
}

// The independent caller supplies its own original-derived table and bound.
// Exact byte equality rejects missing/extra/duplicate keys and noncanonical
// numbers without loading authority or trusting producer-side decoded state.
func parseProviderHistoryGroupTicket(raw []byte, program, profile, costs, previous, inventory string, table *providerHistoryGroupTable, upper providerHistoryGroupUpper) (*providerHistoryGroupTicket, error) {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, ErrConfiguration
	}
	t, err := newProviderHistoryGroupTicket(program, profile, costs, previous, inventory, table, upper)
	if err != nil || !bytes.Equal(raw, []byte(t.raw)) {
		return nil, ErrConfiguration
	}
	return t, nil
}

type providerHistoryGroupConsumption struct {
	ticket, previous, inventory string
	consumed                    uint32
	raw, digest                 string
}

func (*providerHistoryGroupConsumption) UnmarshalJSON([]byte) error { return ErrConfiguration }

// This encodes the irreversible all-member debit. It is not a retained C or
// an independently acknowledged consumption, and cannot dispatch a member.
func newProviderHistoryGroupConsumption(ticket *providerHistoryGroupTicket, previous, inventory string) (*providerHistoryGroupConsumption, error) {
	if !providerHistoryGroupTicketValid(ticket) || !lowerHash(previous) || !lowerHash(inventory) {
		return nil, ErrConfiguration
	}
	all, ok := providerHistoryMemberPrefix(ticket.table.count)
	if !ok {
		return nil, ErrConfiguration
	}
	raw := providerIntegrationJSON(map[string]any{
		"format": providerHistoryGroupConsumptionDomain, "program": ticket.program,
		"stage": strconv.FormatUint(ticket.table.stage, 10), "table": ticket.table.digest,
		"ticket": ticket.digest, "previous": previous, "inventory": inventory,
		"all_consumed": strconv.FormatUint(uint64(all), 10),
	})
	return &providerHistoryGroupConsumption{ticket.digest, previous, inventory, all, string(raw), providerCandidateHash(providerHistoryGroupConsumptionDomain, raw)}, nil
}

func parseProviderHistoryGroupConsumption(raw []byte, ticket *providerHistoryGroupTicket, previous, inventory string) (*providerHistoryGroupConsumption, error) {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, ErrConfiguration
	}
	c, err := newProviderHistoryGroupConsumption(ticket, previous, inventory)
	if err != nil || !bytes.Equal(raw, []byte(c.raw)) {
		return nil, ErrConfiguration
	}
	return c, nil
}

func providerHistoryGroupConsumptionValid(c *providerHistoryGroupConsumption, ticket *providerHistoryGroupTicket) bool {
	if c == nil {
		return false
	}
	copy, err := newProviderHistoryGroupConsumption(ticket, c.previous, c.inventory)
	return err == nil && *copy == *c
}

type providerHistoryGroupSettlement struct {
	ticket, consumption, previous, inventory string
	outcome, failure                         string
	count                                    int
	results                                  [providerHistoryGroupMembers]string
	raw, digest                              string
}

func (*providerHistoryGroupSettlement) UnmarshalJSON([]byte) error { return ErrConfiguration }

// Result digests identify an ordered prefix of retained member originals.
// Every successful member is required for complete. Failure may follow any
// prefix, including failure of final retention after all members completed.
// The actual writer must independently reconstruct every referenced original;
// passing these shape checks does not establish result semantics or closure.
func newProviderHistoryGroupSettlement(ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, previous, inventory, outcome, failure string, results []string) (*providerHistoryGroupSettlement, error) {
	if !providerHistoryGroupTicketValid(ticket) || !providerHistoryGroupConsumptionValid(consumption, ticket) ||
		!lowerHash(previous) || !lowerHash(inventory) || len(results) > ticket.table.count {
		return nil, ErrConfiguration
	}
	all, _ := providerHistoryMemberPrefix(ticket.table.count)
	if consumption.consumed != all {
		return nil, ErrConfiguration
	}
	switch outcome {
	case "complete":
		if failure != "" || len(results) != ticket.table.count {
			return nil, ErrConfiguration
		}
	case "failed":
		if !lowerHash(failure) {
			return nil, ErrConfiguration
		}
	default:
		return nil, ErrConfiguration
	}
	s := &providerHistoryGroupSettlement{ticket: ticket.digest, consumption: consumption.digest, previous: previous, inventory: inventory, outcome: outcome, failure: failure, count: len(results)}
	for i, result := range results {
		if !lowerHash(result) {
			return nil, ErrConfiguration
		}
		s.results[i] = result
	}
	// Copy before encoding; a caller slice cannot later replace the ordered
	// retained-result commitment or turn a failed prefix into complete DATA.
	raw := providerIntegrationJSON(s.object(ticket))
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, ErrConfiguration
	}
	s.raw, s.digest = string(raw), providerCandidateHash(providerHistoryGroupSettlementDomain, raw)
	return s, nil
}

func (s *providerHistoryGroupSettlement) object(t *providerHistoryGroupTicket) map[string]any {
	results := make([]string, s.count)
	copy(results, s.results[:s.count])
	return map[string]any{
		"format": providerHistoryGroupSettlementDomain, "program": t.program,
		"stage": strconv.FormatUint(t.table.stage, 10), "table": t.table.digest,
		"ticket": s.ticket, "consumption": s.consumption, "previous": s.previous,
		"inventory": s.inventory, "outcome": s.outcome, "failure": s.failure, "results": results,
	}
}

func parseProviderHistoryGroupSettlement(raw []byte, ticket *providerHistoryGroupTicket, consumption *providerHistoryGroupConsumption, previous, inventory, outcome, failure string, results []string) (*providerHistoryGroupSettlement, error) {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit {
		return nil, ErrConfiguration
	}
	s, err := newProviderHistoryGroupSettlement(ticket, consumption, previous, inventory, outcome, failure, results)
	if err != nil || !bytes.Equal(raw, []byte(s.raw)) {
		return nil, ErrConfiguration
	}
	return s, nil
}
