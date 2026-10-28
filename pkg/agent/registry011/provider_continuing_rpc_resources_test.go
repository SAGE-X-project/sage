package registry011

import "testing"

func TestProviderRPCResourceQuoteKeepsStorageWorkAndMetadataSeparate(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithPageFixture(t, 64<<20, 6)
	// Config is inert input DATA: no enrollment, ledger, selected owner or IO.
	for _, method := range []string{"eth_chainId", "web3_clientVersion", "eth_getBlockByNumber"} {
		params := []any{}
		if method == "eth_getBlockByNumber" {
			params = []any{"finalized", false}
		}
		request, err := newProviderIntegrationRequest(method, params)
		if err != nil {
			t.Fatal(err)
		}
		capture, err := config.bounds.methods.capture(request, config.capture)
		if err != nil {
			t.Fatal(err)
		}
		p, storage, maximum, ok := providerContinuingRPCResourceUpper(config.bounds, capture, uint64(len(request.wire)))
		if !ok || p.memory != 0 || maximum < 1<<20 || p.headroom == 0 || p.physical != storage.physical || p.reservation.live != p.physical || p.reservation.retained != p.physical || p.metadataPhysical != p.metadataRaw+22*config.bounds.fileOverheadBytes ||
			p.reservation.execution[0] != 1 || p.reservation.execution[8] != 1 || p.reservation.execution[1] != uint64(capture.WireBytes) || p.reservation.remaining[0] != 0 || p.reservation.remaining[1] != 0 || p.reservation.remaining[8] != 0 ||
			p.reservation.execution[9] != storage.consumption || p.reservation.remaining[9] != storage.settlement || storage.physical != storage.raw+32*config.bounds.fileOverheadBytes || storage.exchange > 4<<20 || storage.transport > 4<<20 {
			t.Fatal("quote mixed data families or granted memory/transport authority", method, p, ok)
		}
		for i, v := range p.reservation.work {
			if v != p.reservation.execution[i]+p.reservation.remaining[i] {
				t.Fatal("phase overlap", i)
			}
		}
	}
}

func TestProviderRPCResourceQuoteRefusesIncompleteAndOverflowingData(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithPageFixture(t, 64<<20, 6)
	request, err := newProviderIntegrationRequest("eth_chainId", []any{})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := config.bounds.methods.capture(request, config.capture)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"guardian", "first", "tls", "capture", "empty-request", "large-request", "wire", "object", "stored", "metadata", "file-overhead", "claim"} {
		t.Run(mode, func(t *testing.T) {
			b, capture, size := config.bounds, selected, uint64(len(request.wire))
			guardian := *b.guardian
			b.guardian = &guardian
			switch mode {
			case "guardian":
				b.guardian = nil
			case "first":
				b.firstAdmission = nil
			case "tls":
				b.tls = nil
			case "capture":
				capture.Timeout = 0
			case "empty-request":
				size = 0
			case "large-request":
				size = (16 << 10) + 1
			case "wire":
				size = uint64(capture.WireBytes)
			case "object":
				b.maximumObjectBytes = 1
			case "stored":
				b.maximumStoredBytes = 1
			case "metadata":
				b.maximumMetadataPerCommit = ^uint64(0)
			case "file-overhead":
				b.fileOverheadBytes = ^uint64(0)
			case "claim":
				guardian.claim.work = candidateWorkVector{}
			}
			p, storage, maximum, ok := providerContinuingRPCResourceUpper(b, capture, size)
			if ok || p != (providerContinuingPlan{}) || storage != (providerFirstOperationStorage{}) || maximum != 0 {
				t.Fatal("incomplete DATA produced resource quote", mode)
			}
		})
	}
}
