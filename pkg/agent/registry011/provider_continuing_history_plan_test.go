package registry011

import (
	"encoding/json"
	"math/big"
	"testing"
)

func historyScheduleProposal(blocks, window uint64) providerHistorySchedule {
	return providerHistorySchedule{providerHistoryBatchObservedCandidate, sepoliaDeploymentBlock + blocks - 1, window, 300000, 1 << 20, 1 << 20}
}

func TestProviderHistoryScheduleKnownCompleteSpanFloors(t *testing.T) {
	for _, row := range []struct {
		window, batches, rpc, clocks, records uint64
	}{
		{16, 2138, 126136, 1009088, 384822},
		{32, 1069, 114377, 915016, 346338},
	} {
		p := historyScheduleProposal(34206, row.window)
		d, err := p.diagnostic()
		if err != nil || d.blocks != 34206 || d.batches != row.batches || d.rpc != row.rpc || d.clocks != row.clocks || d.records != row.records || d.oldRPC != 410471 || d.oldClocks != 3283768 || d.clockExceeds || d.recordsExceed {
			t.Fatalf("unexpected complete-span floor for window %d: %+v, %v", row.window, d, err)
		}
		raw, err := p.report()
		var fields map[string]any
		if err != nil || json.Unmarshal(raw, &fields) != nil || fields["complete_costs_qualified"] != false || fields["operating_ready"] != false {
			t.Fatal("a lower-bound report must not qualify the complete history")
		}
		if fields["rpc_floor"] != map[uint64]string{16: "126136", 32: "114377"}[row.window] || fields["mode"] != providerHistoryBatchObservedCandidate {
			t.Fatal("report changed numeric or temporal-contract identity")
		}
	}
}

func TestProviderHistoryScheduleConsecutiveRanges(t *testing.T) {
	for _, blocks := range []uint64{1, 15, 16, 17, 31, 32, 33, 65} {
		for _, window := range []uint64{1, 16, 32} {
			p := historyScheduleProposal(blocks, window)
			d, err := p.diagnostic()
			if err != nil {
				t.Fatal(err)
			}
			next := uint64(sepoliaDeploymentBlock)
			var counted uint64
			for index := uint64(0); index < d.batches; index++ {
				first, last, err := p.rangeAt(index)
				if err != nil || first != next || last < first || last-first+1 > window || last > p.maximumSelected {
					t.Fatal("range omitted, duplicated or exceeded a block")
				}
				counted += last - first + 1
				next = last + 1
			}
			if counted != blocks || next != p.maximumSelected+1 {
				t.Fatal("range list did not cover the complete deployment span")
			}
			if _, _, err := p.rangeAt(d.batches); err == nil {
				t.Fatal("out-of-range index accepted")
			}
		}
	}
}

func TestProviderHistoryScheduleExactLimitsAndRefusalOnly(t *testing.T) {
	p := historyScheduleProposal(1, 1)
	p.clockLimit, p.recordLimit = 112, 45
	d, err := p.diagnostic()
	if err != nil || d.clocks != 112 || d.records != 45 || d.clockExceeds || d.recordsExceed {
		t.Fatal("exact lower-bound limits changed")
	}
	p.clockLimit, p.recordLimit = 111, 44
	d, err = p.diagnostic()
	if err != nil || !d.clockExceeds || !d.recordsExceed {
		t.Fatal("one-below lower-bound limits must be reported")
	}
	raw, err := p.report()
	var fields map[string]any
	if err != nil || json.Unmarshal(raw, &fields) != nil || fields["clock_floor_exceeds_limit"] != true || fields["record_floor_exceeds_limit"] != true || fields["operating_ready"] != false {
		t.Fatal("unsupported schedule lost its refusal diagnostics")
	}
}

func TestProviderHistoryScheduleRejectsIncompleteSelection(t *testing.T) {
	base := historyScheduleProposal(17, 16)
	mutations := []func(*providerHistorySchedule){
		func(p *providerHistorySchedule) { p.mode = "" },
		func(p *providerHistorySchedule) { p.mode = "block-observed-v1" },
		func(p *providerHistorySchedule) { p.mode = "batch-observed-v2" },
		func(p *providerHistorySchedule) { p.maximumSelected = sepoliaDeploymentBlock - 1 },
		func(p *providerHistorySchedule) { p.maximumSelected = 1 << 53 },
		func(p *providerHistorySchedule) { p.blocksPerBatch = 0 },
		func(p *providerHistorySchedule) { p.blocksPerBatch = 4097 },
		func(p *providerHistorySchedule) { p.windowMS = 0 },
		func(p *providerHistorySchedule) { p.windowMS = 300001 },
		func(p *providerHistorySchedule) { p.clockLimit = 0 },
		func(p *providerHistorySchedule) { p.clockLimit = 1<<20 + 1 },
		func(p *providerHistorySchedule) { p.recordLimit = 0 },
		func(p *providerHistorySchedule) { p.recordLimit = 1<<20 + 1 },
	}
	for index, mutate := range mutations {
		p := base
		mutate(&p)
		if p.valid() || p.binding() != "" {
			t.Fatalf("incomplete selection accepted at %d", index)
		}
		if _, err := p.diagnostic(); err == nil {
			t.Fatal("invalid proposal produced a diagnostic")
		}
		if _, _, err := p.rangeAt(0); err == nil {
			t.Fatal("invalid proposal produced a range")
		}
		if _, err := p.report(); err == nil {
			t.Fatal("invalid proposal produced a report")
		}
	}
	if err := json.Unmarshal([]byte(`{}`), &base); err == nil {
		t.Fatal("JSON must not reconstruct this explicit proposal")
	}
}

func TestProviderHistoryScheduleBindingAndLargestArithmetic(t *testing.T) {
	base := historyScheduleProposal(34206, 16)
	if !lowerHash(base.binding()) {
		t.Fatal("missing diagnostic binding")
	}
	for _, mutate := range []func(*providerHistorySchedule){
		func(p *providerHistorySchedule) { p.maximumSelected++ },
		func(p *providerHistorySchedule) { p.blocksPerBatch++ },
		func(p *providerHistorySchedule) { p.windowMS-- },
		func(p *providerHistorySchedule) { p.clockLimit-- },
		func(p *providerHistorySchedule) { p.recordLimit-- },
	} {
		p := base
		mutate(&p)
		if !p.valid() || p.binding() == base.binding() {
			t.Fatal("changed trusted bound reused a proposal binding")
		}
	}
	large := base
	large.maximumSelected, large.blocksPerBatch = 1<<53-1, 1
	d, err := large.diagnostic()
	if err != nil || !d.clockExceeds || !d.recordsExceed {
		t.Fatal("largest numeric input must refuse the finite capacities")
	}
	// Independently calculate with arbitrary precision; no platform word-size
	// overflow or whole-span allocation may make a huge proposal look small.
	blocks := new(big.Int).SetUint64(large.maximumSelected - sepoliaDeploymentBlock + 1)
	want := new(big.Int).Mul(blocks, big.NewInt(112))
	if !want.IsUint64() || d.clocks != want.Uint64() {
		t.Fatal("large-span clock lower bound overflowed")
	}
	first, last, err := large.rangeAt(d.batches - 1)
	if err != nil || first != large.maximumSelected || last != first {
		t.Fatal("largest last range overflowed")
	}
}
