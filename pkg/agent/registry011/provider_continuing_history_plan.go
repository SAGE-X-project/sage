package registry011

import "strconv"

// This is a proposed acquisition schedule, not an admitted program, a live
// cursor, a custody ACK or an operating profile. Existing collectors do not
// consume it. The named temporal mode has no implicit/default selection.
type providerHistorySchedule struct {
	mode                                      string
	maximumSelected, blocksPerBatch, windowMS uint64
	clockLimit, recordLimit                   uint64
}

const providerHistoryBatchObservedCandidate = "batch-observed-v1"

func (p providerHistorySchedule) valid() bool {
	return p.mode == providerHistoryBatchObservedCandidate &&
		p.maximumSelected >= sepoliaDeploymentBlock && p.maximumSelected < 1<<53 &&
		p.blocksPerBatch > 0 && p.blocksPerBatch <= 4096 &&
		p.windowMS > 0 && p.windowMS <= 300000 &&
		p.clockLimit > 0 && p.clockLimit <= 1<<20 &&
		p.recordLimit > 0 && p.recordLimit <= 1<<20
}

func (*providerHistorySchedule) UnmarshalJSON([]byte) error { return ErrConfiguration }

// A successful arithmetic calculation NEVER establishes full feasibility.
// Additional work/bytes/pages/catalogs/clock/native/four-model/failure rows
// are still required before H0; no selected response can enlarge this plan.
type providerHistoryScheduleDiagnostic struct {
	blocks, batches                        uint64
	contentRPC, boundaryRPC, comparisonOps uint64
	rpc, clocks, records                   uint64
	oldRPC, oldClocks                      uint64
	clockExceeds, recordsExceed            bool
}

func (p providerHistorySchedule) diagnostic() (providerHistoryScheduleDiagnostic, error) {
	if !p.valid() {
		return providerHistoryScheduleDiagnostic{}, ErrConfiguration
	}
	blocks := p.maximumSelected - sepoliaDeploymentBlock + 1
	batches := (blocks-1)/p.blocksPerBatch + 1
	// These are LOWER bounds, not reservations. Candidate content has one
	// full-block, one committed-receipt and one native-gas read per block.
	// Four start and seven end observations belong to EVERY bounded batch.
	content, boundary := 3*blocks, 11*batches
	rpc := content + boundary
	// Each RPC consumes its own T/C/S; every batch also needs a separate
	// comparison T/C/S after all eleven boundary originals have closed.
	records := 3 * (rpc + batches)
	// Eight backend samples are only the existing successful RPC floor.
	// This excludes separate local inspection/compare/G/page/final clocks.
	clocks := 8 * rpc
	// The old source_batch.go schedule includes eleven reads per block and
	// the separate selected-to-base path. It is NOT replaced by this API.
	oldRPC := 12*blocks - 1
	// blocks<=2^53 and batches<=blocks imply every expression above and
	// 8*oldRPC<=96*2^53 fits uint64. Work remains O(1) even for a huge span.
	return providerHistoryScheduleDiagnostic{
		blocks, batches, content, boundary, batches, rpc, clocks, records,
		oldRPC, 8 * oldRPC, clocks > p.clockLimit, records > p.recordLimit,
	}, nil
}

// rangeAt enumerates DATA only. Looking up range k does not accept k-1,
// advance either model or prove continuity. A live candidate cursor must
// independently bind and retain the complete comparison's originals/ACK.
func (p providerHistorySchedule) rangeAt(index uint64) (uint64, uint64, error) {
	d, err := p.diagnostic()
	if err != nil || index >= d.batches {
		return 0, 0, ErrConfiguration
	}
	first := uint64(sepoliaDeploymentBlock) + index*p.blocksPerBatch
	last := first + min(p.blocksPerBatch-1, p.maximumSelected-first)
	return first, last, nil
}

// The binding is a configuration/diagnostic digest only, with no operating
// role and no JSON loader. Exact mode, full span, interval and finite limits
// stay distinguishable before any future H0 or network operation.
func (p providerHistorySchedule) binding() string {
	if !p.valid() {
		return ""
	}
	raw := providerIntegrationJSON(map[string]any{
		"format": "sage-provider-history-schedule-proposal-v1",
		"mode":   p.mode, "deployment": strconv.FormatUint(sepoliaDeploymentBlock, 10),
		"maximum_selected": strconv.FormatUint(p.maximumSelected, 10),
		"blocks_per_batch": strconv.FormatUint(p.blocksPerBatch, 10),
		"window_ms":        strconv.FormatUint(p.windowMS, 10),
		"clock_limit":      strconv.FormatUint(p.clockLimit, 10),
		"record_limit":     strconv.FormatUint(p.recordLimit, 10),
	})
	return providerCandidateHash("sage-provider-history-schedule-proposal-v1", raw)
}

// Even an example whose floor fits has complete_costs_qualified=false and
// operating_ready=false. The report cannot be fed into an execution API.
func (p providerHistorySchedule) report() ([]byte, error) {
	d, err := p.diagnostic()
	if err != nil {
		return nil, err
	}
	fields := map[string]any{
		"format": "sage-provider-history-schedule-diagnostic-v1",
		"mode":   p.mode, "binding": p.binding(),
		"complete_costs_qualified": false, "operating_ready": false,
		"clock_floor_exceeds_limit":  d.clockExceeds,
		"record_floor_exceeds_limit": d.recordsExceed,
	}
	for key, value := range map[string]uint64{
		"blocks": d.blocks, "batches": d.batches,
		"content_rpc_floor": d.contentRPC, "boundary_rpc_floor": d.boundaryRPC,
		"batch_comparison_operations_floor": d.comparisonOps,
		"rpc_floor":                         d.rpc, "backend_clock_floor": d.clocks,
		"tcs_record_floor": d.records, "old_rpc_floor": d.oldRPC,
		"old_backend_clock_floor": d.oldClocks,
	} {
		fields[key] = strconv.FormatUint(value, 10)
	}
	return providerIntegrationJSON(fields), nil
}
