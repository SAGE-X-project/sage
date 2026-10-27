package registry011

import (
	"math"
	"testing"
)

func providerLedgerBoundsFixture() providerLedgerBounds {
	return providerLedgerBounds{maximumRecords: 8192, pageRecords: 4096, maximumPages: 2, maximumPayloadBytes: 16 << 20, maximumMetadataBytes: 64 << 20, pagePayloadBytes: 16 << 20, pageMetadataBytes: 32 << 20, maximumResidentBytes: 128 << 20, codecMemoryMultiplier: 4, ticketMemoryOverhead: 512, operationMemoryOverhead: 128, segmentMemoryOverhead: 128, objectMemoryOverhead: 128, maximumCommits: 8196, maximumObjects: 65536}
}

func TestProviderPrivateLedgerProfileHasExplicitFiniteBounds(t *testing.T) {
	b := providerLedgerBoundsFixture()
	if !b.valid() {
		t.Fatal("bounded inert profile")
	}
	object := b.object()
	if object["format"] != "sage-provider-private-ledger-profile-v1" || len(object) != 16 {
		t.Fatal("closed private ledger profile")
	}
	for key, value := range object {
		if key != "format" {
			if _, ok := providerCandidateUint(value, true); !ok {
				t.Fatal("implicit/nondecimal resource value")
			}
		}
	}
	for _, name := range []string{"records-zero", "records-large", "page-zero", "page-large", "page-over-lifetime", "pages-zero", "pages-insufficient", "payload-zero", "payload-large", "metadata-zero", "metadata-large", "page-payload-zero", "page-payload-large", "page-metadata-zero", "page-metadata-large", "resident-zero", "resident-large", "codec-zero", "codec-large", "ticket-overhead", "operation-overhead", "segment-overhead", "object-overhead", "commits-zero", "commits-large", "commits-insufficient", "objects-zero", "objects-large"} {
		t.Run(name, func(t *testing.T) {
			copy := b
			switch name {
			case "records-zero":
				copy.maximumRecords = 0
			case "records-large":
				copy.maximumRecords = 1<<20 + 1
			case "page-zero":
				copy.pageRecords = 0
			case "page-large":
				copy.pageRecords = 4097
			case "page-over-lifetime":
				copy.maximumRecords = 1
			case "pages-zero":
				copy.maximumPages = 0
			case "pages-insufficient":
				copy.maximumPages = 1
			case "payload-zero":
				copy.maximumPayloadBytes = 0
			case "payload-large":
				copy.maximumPayloadBytes = 16<<30 + 1
			case "metadata-zero":
				copy.maximumMetadataBytes = 0
			case "metadata-large":
				copy.maximumMetadataBytes = 64<<30 + 1
			case "page-payload-zero":
				copy.pagePayloadBytes = 0
			case "page-payload-large":
				copy.pagePayloadBytes = copy.maximumPayloadBytes + 1
			case "page-metadata-zero":
				copy.pageMetadataBytes = 0
			case "page-metadata-large":
				copy.pageMetadataBytes = copy.maximumMetadataBytes + 1
			case "resident-zero":
				copy.maximumResidentBytes = 0
			case "resident-large":
				copy.maximumResidentBytes = 16<<30 + 1
			case "codec-zero":
				copy.codecMemoryMultiplier = 0
			case "codec-large":
				copy.codecMemoryMultiplier = 257
			case "ticket-overhead":
				copy.ticketMemoryOverhead = 0
			case "operation-overhead":
				copy.operationMemoryOverhead = 1<<20 + 1
			case "segment-overhead":
				copy.segmentMemoryOverhead = 0
			case "object-overhead":
				copy.objectMemoryOverhead = 1<<20 + 1
			case "commits-zero":
				copy.maximumCommits = 0
			case "commits-large":
				copy.maximumCommits = 2<<20 + 1
			case "commits-insufficient":
				copy.maximumCommits = copy.maximumRecords
			case "objects-zero":
				copy.maximumObjects = 0
			case "objects-large":
				copy.maximumObjects = 16<<20 + 1
			}
			if copy.valid() {
				t.Fatal("unfunded or unsupported private profile")
			}
		})
	}
}

func TestProviderPrivateLedgerResidentUpperIncludesCopiesAndScratch(t *testing.T) {
	b := providerLedgerBoundsFixture()
	if upper, ok := b.residentUpper(1, 2, 1, 3, 100, 200); !ok || upper != 3560 {
		t.Fatal("two copies and scratch not conserved", upper, ok)
	}
	for _, fixture := range [][6]uint64{
		{b.maximumRecords + 1, 0, 0, 0, 0, 0}, {0, b.maximumCommits + 1, 0, 0, 0, 0},
		{0, 0, b.maximumRecords + 1, 0, 0, 0}, {0, 0, 0, b.maximumObjects + 1, 0, 0},
		{0, 0, 0, 0, math.MaxUint64, 0}, {0, 0, 0, 0, 1, math.MaxUint64},
		{0, 0, 0, 0, b.maximumResidentBytes, 0},
	} {
		if _, ok := b.residentUpper(fixture[0], fixture[1], fixture[2], fixture[3], fixture[4], fixture[5]); ok {
			t.Fatal("resident growth exceeded its declared bound")
		}
	}
	if _, ok := (providerLedgerBounds{}).residentUpper(0, 0, 0, 0, 0, 0); ok {
		t.Fatal("implicit resident profile")
	}
}
