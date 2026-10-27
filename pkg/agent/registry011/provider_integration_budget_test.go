package registry011

import "testing"

func providerInertLegacyBounds() (providerIntegrationBounds, providerPrimitiveBounds) {
	b, capture := providerFrontierBoundsFixture()
	b.continuation, b.methods, b.tls = nil, nil, nil
	b.maximumEntries = 64
	return b, providerPrimitiveBounds{capture: capture, blocks: 4, transactions: 16, frames: 64, logs: 64, proofs: 16, maximumCreatedObjects: 32}
}

// Every case is scalar arithmetic over inert bounds. No buffer, filesystem,
// transport or operating profile of the declared size is created or selected.
func TestProviderReservationRejectsCumulativeArithmeticBeforeWork(t *testing.T) {
	for _, mode := range []string{"request-zero", "request-wire", "local-wire", "prefix-cap", "objects-zero", "object-count", "disk-cap", "prefix-product", "prefix-scale", "work-sum", "remaining-sum", "retained-peak"} {
		t.Run(mode, func(t *testing.T) {
			b, primitive := providerInertLegacyBounds()
			bytes, prefix, rpc := uint64(512), uint64(0), true
			switch mode {
			case "request-zero":
				bytes = 0
			case "request-wire":
				bytes = uint64(primitive.capture.WireBytes)
			case "local-wire":
				rpc = false
			case "prefix-cap":
				prefix = b.maximumStoredBytes + 1
			case "objects-zero":
				primitive.maximumCreatedObjects = 0
			case "object-count":
				primitive.maximumCreatedObjects = uint64(b.maximumEntries) + 1
			case "disk-cap":
				b.maximumStoredBytes, b.setupLive, b.setupRetained = 1<<20, 512<<10, 256<<10
			case "prefix-product":
				b.maximumStoredBytes, b.maximumObjectBytes, prefix = 1<<40, 32<<20, 1<<40
			case "prefix-scale":
				b.maximumStoredBytes, b.maximumObjectBytes, prefix = 1<<40, 32<<20, 1<<37
			case "work-sum":
				b.maximumStoredBytes, b.maximumObjectBytes, prefix = 1<<40, 256<<10, 1<<40-1
			case "remaining-sum":
				b.maximumStoredBytes, b.maximumObjectBytes, prefix = 1<<40, 256<<10, 1<<39
			case "retained-peak":
				b.maximumStoredBytes = 2 << 20
			}
			if !b.valid() {
				t.Fatal("arithmetic boundary must start with structurally valid bounds")
			}
			before := b
			reservation, err := providerEstimateOperation(b, primitive, bytes, prefix, rpc, false)
			if err == nil || reservation != (providerOperationReservation{}) || b != before {
				t.Fatal("insufficient or overflowing reservation accepted or changed its input")
			}
		})
	}
}

func TestProviderReservationCannotMixContinuationAndLegacyProfiles(t *testing.T) {
	legacy, _ := providerInertLegacyBounds()
	continued, _ := providerFrontierBoundsFixture()
	if !legacy.valid() || !continued.valid() {
		t.Fatal("normal inert bounds")
	}
	legacy.methods = continued.methods
	if legacy.valid() {
		t.Fatal("legacy profile silently adopted private method limits")
	}
	continued.maximumCommits--
	if continued.valid() {
		t.Fatal("continuation profile disagreed with its lifetime limits")
	}
}
