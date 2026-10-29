package registry011

import "testing"

func TestProviderFirstOperationPlanKeepsFuturePhysicalAndMeasuredWorkSeparate(t *testing.T) {
	s, f := providerFirstOperationPreparedFixture(t)
	p, ok := providerFirstOperationPlanUpper(s.config.bounds, s.config.capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	if !ok || p.memory == 0 || p.memory > s.config.bounds.scan.maximumResidentBytes || p.storage.physical != p.storage.raw+32*s.config.bounds.fileOverheadBytes || p.reservation.live != p.storage.physical || p.reservation.retained != p.storage.physical || p.reservation.execution[9] != p.storage.consumption || p.reservation.remaining[9] != p.storage.settlement || p.reservation.execution[0] != 1 || p.reservation.execution[8] != 1 || p.reservation.remaining[0] != 0 || p.reservation.remaining[1] != 0 || p.reservation.remaining[8] != 0 || p.reservation.execution[1] != uint64(p.capture.WireBytes) || p.firstExtra[2] == 0 || p.firstExtra[10] == 0 || p.firstExtra[9] != 0 || p.firstExtra[0] != 0 {
		t.Fatal("future candidate quote borrowed E1, doubled its closed physical quote or invented actuals", ok, p)
	}
	for coordinate, total := range p.reservation.work {
		if total != p.reservation.execution[coordinate]+p.reservation.remaining[coordinate] {
			t.Fatal("execution and separately held remaining work overlap", coordinate)
		}
	}
	if p.storage.exchange > s.config.bounds.maximumObjectBytes || p.reservation.work[2] > 256<<20 || s.firstLimits.operation[2] > 256<<20 || s.config.bounds.maximumObjectBytes != 128<<10 {
		t.Fatal("fresh candidate profile evaded the existing operation policy ceiling")
	}
	maximum := max(uint64(1<<20), s.config.bounds.maximumMetadataPerCommit, p.storage.transport, p.storage.exchange)
	if p.headroom != maximum*maximum+1+s.config.bounds.maximumObjectBytes+1 || p.reservation.execution[10] <= p.headroom || p.reservation.remaining[10] <= p.headroom || f.replayStarted || f.custodianStarted || f.plan != nil {
		t.Fatal("largest declared serializer preflight was lost or a scalar quote performed IO")
	}
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil || request.wire != providerFirstOperationRequestWire || request.head != providerFirstOperationRequestHead || request.body != providerFirstOperationRequestBody || len(request.body) != 85 || len(request.wire) != 262 {
		t.Fatal("pre-I scalar wire length differs from the existing exact formatter", err)
	}
}

func TestProviderFirstOperationPlanRefusesIncompleteOrOverflowingData(t *testing.T) {
	s, f := providerFirstOperationPreparedFixture(t)
	for _, mode := range []string{"legacy", "entries", "objects", "ordinal", "attempts", "records", "retained", "output", "overflow", "capture", "memory"} {
		t.Run(mode, func(t *testing.T) {
			b, global := s.config.bounds, s.config.capture
			ordinal, records, retained, output := f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput
			switch mode {
			case "legacy":
				b.firstAdmission = nil
			case "entries":
				b.maximumEntries = 13
			case "objects":
				b.maximumObjects = 37
			case "ordinal":
				ordinal = 0
			case "attempts":
				ordinal = b.scan.maximumAttempts + 1
			case "records":
				records = 3
			case "retained":
				retained = b.maximumStoredBytes + 1
			case "output":
				output = 0
			case "overflow":
				output = ^uint64(0)
			case "capture":
				global.Timeout = 0
			case "memory":
				output = b.scan.maximumResidentBytes
			}
			if p, ok := providerFirstOperationPlanUpper(b, global, ordinal, records, retained, output); ok || p != (providerFirstOperationPlan{}) {
				t.Fatal("incomplete scalar data produced a quote", mode)
			}
		})
	}
	for _, args := range [][7]uint64{{^uint64(0), 1, 1, 0, 0, 0, 0}, {1, ^uint64(0), 1, 0, 0, 0, 0}, {1, 1, ^uint64(0), 0, 0, 0, 0}, {1, 1, 1, ^uint64(0), 0, 0, 0}, {1, 1, 1, 0, ^uint64(0), 0, 0}, {1, 1, 1, 0, 0, 0, ^uint64(0)}} {
		if value, ok := providerFirstOperationPhaseUpper(args[0], args[1], args[2], args[3], args[4], args[5], args[6]); ok || value != (candidateWorkVector{}) {
			t.Fatal("prospective scalar arithmetic wrapped", args)
		}
	}
}
