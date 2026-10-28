package registry011

import "testing"

func TestProviderFirstAdmissionPlanKeepsWeightedWorkAndPhysicalCoordinatesSeparate(t *testing.T) {
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	p, ok := providerFirstAdmissionPlanUpper(s.config.bounds, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	if !ok || p.owner != nil || p.used || p.stored != 0 || p.memory == 0 || p.memory > s.config.bounds.scan.maximumResidentBytes || p.clocks != 18 {
		t.Fatal("initial-only prospective candidate plan", ok, p)
	}
	n := max(s.config.bounds.maximumMetadataPerCommit, uint64(64<<10))
	events := uint64(611 + 2*(24+26*f.scanPin.ordinal) + 2*(5+29*f.records))
	q := uint64(2*273+538) + events
	b := uint64(2*1252+2930+3) + events
	want := q*(n*n+1) + b*(2*n+101) + 84998 + 44 + p.clocks + (1<<20)*(1<<20) + 1 + s.config.bounds.maximumObjectBytes + 1
	if p.work[2] != q*n || p.work[10] != want || p.work[9] != p.raw || p.physical != p.raw+21*s.config.bounds.fileOverheadBytes || p.work[0] != 0 || p.work[1] != 0 || p.work[8] != 0 || p.raw > f.limits.work[9] || p.work[10] <= 611 {
		t.Fatal("events were used as work or old physical files were charged again")
	}
	more, ok := providerFirstAdmissionPlanUpper(s.config.bounds, f.scanPin.ordinal+1, f.records, s.retainedRaw, f.readerOutput)
	if !ok || more.work[10] <= p.work[10] || more.work[2] <= p.work[2] || more.raw != p.raw || more.physical != p.physical {
		t.Fatal("original control closure was omitted or counted as new stored files")
	}
}

func TestProviderFirstAdmissionPlanRefusesOverflowOrIncompleteInitialShapes(t *testing.T) {
	s, f := providerFirstWriterFixture(t, providerFirstWriterBounds())
	for _, mode := range []string{"legacy", "ordinal", "attempts", "records", "retained", "output", "overflow", "resident", "entries"} {
		t.Run(mode, func(t *testing.T) {
			b, ordinal, records, retained, output := s.config.bounds, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput
			switch mode {
			case "legacy":
				b.firstAdmission = nil
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
			case "resident":
				output = b.scan.maximumResidentBytes
			case "entries":
				b.maximumEntries = 13
			}
			if _, ok := providerFirstAdmissionPlanUpper(b, ordinal, records, retained, output); ok {
				t.Fatal("incomplete or unbounded first-E1 plan", mode)
			}
		})
	}
}
