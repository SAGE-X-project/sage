package registry011

import (
	"testing"
	"time"
)

func TestProviderHistoryCollectionCaptureBoundsMatchHandwrittenClassCosts(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		blocks, window, content, native uint64
		effects, versions, present      uint64
		gas                             string
		classes                         [8]uint64
		requests, wire, decoded, units  uint64
	}{
		{"one-absent", 1, 1, 1, 0, 0, 0, 0, SepoliaAmsterdamRootExecutionGasProfile,
			[8]uint64{4, 6, 13, 3, 1, 1, 0, 0}, 28, 2621468, 2621412, 316753174512},
		{"two-root-native", 2, 1, 1, 2, 0, 0, 0, SepoliaAmsterdamRootRegularGasProfile,
			[8]uint64{12, 19, 39, 8, 2, 2, 0, 2}, 84, 8028244, 8028076, 1037232447440},
		{"short-last-root-window", 13, 4, 2, 13, 4, 4, 4, SepoliaAmsterdamRootExecutionGasProfile,
			[8]uint64{28, 54, 91, 26, 13, 13, 0, 17}, 242, 25428210, 25427726, 3690442142728},
		{"short-last-state-gas-window", 13, 4, 2, 13, 4, 4, 4, SepoliaAmsterdamGasProfile,
			[8]uint64{28, 54, 91, 26, 13, 13, 13, 4}, 242, 22872306, 22871822, 2852926075912},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := historyCollectionInput(t)
			in.blocks, in.maximumSelected = tc.blocks, sepoliaDeploymentBlock+tc.blocks-1
			in.windowBlocks, in.contentBlocks, in.native = tc.window, tc.content, tc.native
			in.effects, in.versions, in.presentStops, in.gas = tc.effects, tc.versions, tc.present, tc.gas
			manifest := historyCollectionManifest(t, in) // genuine offline DATA, no H0
			bounds, _ := providerFrontierBoundsFixture()
			got, ok := providerHistoryCollectionCaptureUpper(manifest.input, *bounds.methods, *bounds.tls)
			want := providerHistoryCollectionCaptureCost{tc.classes, candidateWorkVector{0: tc.requests, 1: tc.wire, 2: tc.decoded, 10: tc.units}}
			if !ok || got != want || manifest.counts.requests != tc.requests {
				t.Fatal("capture quote differs from handwritten grammar and event rows", got, want, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionCaptureBoundsFollowActualGasTracerClasses(t *testing.T) {
	for _, tc := range []struct {
		profile, class string
		wire           int
		gas, execution uint64
	}{
		{SepoliaAmsterdamGasProfile, "gas", 64 << 10, 13, 4},
		{SepoliaAmsterdamNethermindGasProfile, "gas", 64 << 10, 13, 4},
		{SepoliaAmsterdamRootRegularGasProfile, "execution", 256 << 10, 0, 17},
		{SepoliaAmsterdamRootExecutionGasProfile, "execution", 256 << 10, 0, 17},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			in := historyCollectionInput(t)
			in.gas = tc.profile
			manifest := historyCollectionManifest(t, in)
			bounds, _ := providerFrontierBoundsFixture()
			got, ok := providerHistoryCollectionCaptureUpper(manifest.input, *bounds.methods, *bounds.tls)
			if !ok || got.classes[6] != tc.gas || got.classes[7] != tc.execution {
				t.Fatal("gas evidence charged to a different tracer class", got, ok)
			}
			// Inert grammar scenario: use the real planner and selector for
			// one post-fork block. This is not a reconstructed S or authority.
			selected := historyFirstFinalizedHeader()
			selected.Number = in.maximumSelected
			selected.Timestamp = max(selected.Timestamp, sepoliaAmsterdamTime)
			cursor, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, selected, providerReplayUnitMeter())
			if err != nil {
				t.Fatal("offline cursor precursor", err)
			}
			cursor.phase, cursor.mode = "content", "content"
			cursor.low, cursor.high, cursor.next = selected.Number, selected.Number, selected.Number
			table, err := cursor.nextTableObserved(1, *bounds.methods, in.capture, providerReplayUnitMeter())
			if err != nil || table == nil || table.count != 3 || table.members[2].request.method != "debug_traceBlockByHash" ||
				table.members[2].class != tc.class || table.members[2].capture.WireBytes != tc.wire {
				t.Fatal("actual native-evidence request no longer uses the quoted class", table, err)
			}
		})
	}
}

func TestProviderHistoryCollectionCaptureEnvelopeCoversPermittedRequestSizes(t *testing.T) {
	// Numeric request lengths only, not fabricated HTTP or a TLS connection.
	// Independent explicit-event oracle: copies 2r+3, comparisons 2r+2,
	// read response+2, codec response^2+1, and four TLS identities total69912.
	for _, width := range []int{2, 3, 64, 16 << 10, 256 << 10, 16 << 20} {
		limit := providerMethodLimit{1, width}
		methods := providerMethodBounds{limit, limit, limit, limit, limit, limit, limit, limit}
		in := providerHistoryCollectionInput{blocks: 1, windowBlocks: 1, contentBlocks: 1, encoding: EthereumJSONBlocks, gas: SepoliaAmsterdamRootExecutionGasProfile,
			capture: ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: 1, BodyBytes: 1, WireBytes: width, JSONDepth: 1}}
		tls := providerTLSBounds{1, 4, 8 << 10}
		got, ok := providerHistoryCollectionCaptureUpper(in, methods, tls)
		w := uint64(width)
		unit := w*w - w + 12 + 69912
		want := candidateWorkVector{0: 28, 1: 28 * (w + 1), 2: 28 * (w - 1), 10: 28 * unit}
		if !ok || got.work != want {
			t.Fatal("numeric endpoint differs from independent event oracle", width, got, ok)
		}
		for request := uint64(1); request <= min(uint64(16<<10), w-1); request++ {
			actual, valid := providerHistoryCaptureUpper(request, in.capture, tls)
			response := w - request
			oracle := candidateWorkVector{1: w + 1, 2: response, 10: response*response + response + 4*request + 8 + 69912}
			if !valid || actual != oracle || actual[1] > w+1 || actual[2] > w-1 || actual[10] > unit {
				t.Fatal("a permitted request exceeds the prospective endpoint", width, request, actual, valid)
			}
		}
	}
}

func TestProviderHistoryCollectionCaptureBoundsRefuseInvalidData(t *testing.T) {
	for _, mode := range []string{"zero-blocks", "block-domain", "native", "window", "content", "present", "versions", "request-overflow", "encoding", "gas-profile", "capture", "method-zero", "method-global", "method-header", "tls"} {
		t.Run(mode, func(t *testing.T) {
			in := historyCollectionInput(t)
			bounds, _ := providerFrontierBoundsFixture()
			methods, certificates := *bounds.methods, *bounds.tls
			switch mode {
			case "zero-blocks":
				in.blocks = 0
			case "block-domain":
				in.blocks = 1<<20 + 1
			case "native":
				in.native = in.blocks + 1
			case "window":
				in.windowBlocks = 0
			case "content":
				in.contentBlocks = 11
			case "present":
				in.presentStops = 5
			case "versions":
				in.versions = 1<<20 + 1
			case "request-overflow":
				in.effects = ^uint64(0)
			case "encoding":
				in.encoding = "implicit-block-format"
			case "gas-profile":
				in.gas = "implicit-tracer"
			case "capture":
				in.capture.Timeout = 0
			case "method-zero":
				methods.scalar.body = 0
			case "method-global":
				methods.execution.wire = in.capture.WireBytes + 1
			case "method-header":
				methods.scalar.wire = in.capture.HeaderBytes
			case "tls":
				certificates.chains = 0
			}
			if got, ok := providerHistoryCollectionCaptureUpper(in, methods, certificates); ok || got != (providerHistoryCollectionCaptureCost{}) {
				t.Fatal("invalid input returned a partial capture row", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionCaptureBoundsRefuseProductAndSumOverflow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		blocks uint64
	}{
		{"sum", 2000}, {"product", 10000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := historyCollectionInput(t)
			in.blocks, in.maximumSelected, in.windowBlocks, in.contentBlocks = tc.blocks, sepoliaDeploymentBlock+tc.blocks-1, 1, 1
			in.native, in.effects, in.versions, in.presentStops = 0, 0, 0, 0
			in.capture.BodyBytes, in.capture.WireBytes = 8<<20, 16<<20
			manifest := historyCollectionManifest(t, in) // valid grammar, not a sufficient cost certificate
			limit := providerMethodLimit{8 << 20, 16 << 20}
			methods := providerMethodBounds{limit, limit, limit, limit, limit, limit, limit, limit}
			unit := uint64(16<<20)*uint64(16<<20) - uint64(16<<20) + 12 + 69912
			if tc.name == "sum" {
				// Every class product fits, but their total work does not.
				if manifest.counts.requests != 107974 {
					t.Fatal("sum-overflow grammar precursor changed")
				}
				for _, count := range [8]uint64{15996, 25993, 51987, 9998, 2000, 2000, 0, 0} {
					if count > ^uint64(0)/unit {
						t.Fatal("sum case already overflows a class product")
					}
				}
			} else if uint64(259987) <= ^uint64(0)/unit {
				t.Fatal("product-overflow code-class precursor changed")
			}
			if got, ok := providerHistoryCollectionCaptureUpper(manifest.input, methods, providerTLSBounds{1, 4, 8 << 10}); ok || got != (providerHistoryCollectionCaptureCost{}) {
				t.Fatal("overflow returned a wrapped or partial row", got, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionCaptureBoundsCoverSmallerOptionalTotals(t *testing.T) {
	in := historyCollectionInput(t)
	bounds, _ := providerFrontierBoundsFixture()
	upper, ok := providerHistoryCollectionCaptureUpper(in, *bounds.methods, *bounds.tls)
	if !ok {
		t.Fatal("prospective capture precursor")
	}
	for _, blocks := range []uint64{1, 4, 8, 13} {
		smaller := in
		smaller.blocks, smaller.maximumSelected = blocks, sepoliaDeploymentBlock+blocks-1
		smaller.native, smaller.effects, smaller.versions, smaller.presentStops = 0, 0, 0, 0
		manifest := historyCollectionManifest(t, smaller)
		got, ok := providerHistoryCollectionCaptureUpper(manifest.input, *bounds.methods, *bounds.tls)
		if !ok {
			t.Fatal("smaller valid grammar refused", blocks)
		}
		for i, count := range got.classes {
			if count > upper.classes[i] {
				t.Fatal("smaller optional grammar exceeds class upper", blocks, i)
			}
		}
		for i, units := range got.work {
			if units > upper.work[i] {
				t.Fatal("smaller optional grammar exceeds work upper", blocks, i)
			}
		}
	}
}
