package registry011

import (
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryTypedSpineKeepsIndependentCanonicalBytesAndFreshWork(t *testing.T) {
	for _, ordinal := range []int{0, 1, 9, 10, (1 << 20) - 1} {
		for _, header := range []FinalizedHeader{
			{Hash: admissionTestDigest("a"), ParentHash: admissionTestDigest("b")},
			historyCollectionSpineTestHeader(),
		} {
			previous := ""
			if ordinal != 0 {
				previous = admissionTestDigest("d")
			}
			before := header
			value := historyCollectionSpineTestValue(previous, ordinal, header)
			value["format"], value["kind"] = providerHistoryCollectionStateDomain, "spine"
			raw := providerIntegrationJSON(value)
			want := providerCandidateHash(providerHistoryCollectionStateDomain, raw)
			m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
			got := providerHistoryCollectionTypedSpineDigestObserved(previous, ordinal, header, m)
			// An independent byte-length equation covers all fifteen copies,
			// three actual decimal encodes, original kind copy and final hash.
			n, d := uint64(len(raw)), uint64(len(providerHistoryCollectionStateDomain))
			x, y, z := uint64(len(strconv.FormatUint(header.Number, 10))), uint64(len(strconv.FormatUint(header.Timestamp, 10))), uint64(len(strconv.Itoa(ordinal)))
			units := 6 + n + 15 + x*x + y*y + z*z + 3 + n + d + 2
			if got != want || m.failed || m.actual != (candidateWorkVector{10: units}) || units > 1576 || header != before {
				t.Fatal("closed encoder changed independent canonical hash or actual work", ordinal, header, got, want, m.actual, units)
			}
			if providerHistoryCollectionTypedSpineDigestObserved(previous, ordinal, header, m) != want || m.actual != (candidateWorkVector{10: 2 * units}) {
				t.Fatal("repeated encoding cached work or reset its original observer")
			}
			if ordinal == (1<<20)-1 && header.Number == ^uint64(0) && (len(raw) != 402 || units != 1576) {
				t.Fatal("maximum fixed-schema bytes/work literal changed", len(raw), units)
			}
			header.Hash = admissionTestDigest("c")
			value = historyCollectionSpineTestValue(previous, ordinal, header)
			value["format"], value["kind"] = providerHistoryCollectionStateDomain, "spine"
			changed := providerCandidateHash(providerHistoryCollectionStateDomain, providerIntegrationJSON(value))
			if got := providerHistoryCollectionTypedSpineDigestObserved(previous, ordinal, header, m); got == want || got != changed || m.failed {
				t.Fatal("fresh original header change was hidden by an old root")
			}
		}
	}
}

func TestProviderHistoryTypedSpinePreservesUnenteredDataAndStickyPreflightRefusal(t *testing.T) {
	for _, mode := range []string{"nil", "negative", "large", "first-previous", "later-empty", "later-invalid", "hash", "parent", "timestamp"} {
		header, previous, ordinal := historyCollectionSpineTestHeader(), "", 0
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
		switch mode {
		case "nil":
			m = nil
		case "negative":
			ordinal = -1
		case "large":
			ordinal = 1 << 20
		case "first-previous":
			previous = admissionTestDigest("d")
		case "later-empty":
			ordinal = 1
		case "later-invalid":
			ordinal, previous = 1, "0x"+strings.Repeat("D", 64)
		case "hash":
			header.Hash = admissionTestDigest("A")
		case "parent":
			header.ParentHash = "short"
		case "timestamp":
			header.Timestamp = 1 << 53
		}
		if got := providerHistoryCollectionTypedSpineDigestObserved(previous, ordinal, header, m); got != "" || m != nil && (m.actual != (candidateWorkVector{}) || m.failed) {
			t.Fatal("invalid detached typed DATA entered work", mode, got)
		}
	}
	for _, capacity := range []uint64{0, 5, 262150, 262151} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: capacity})
		got := providerHistoryCollectionTypedSpineDigestObserved(admissionTestDigest("d"), (1<<20)-1, historyCollectionSpineTestHeader(), m)
		if capacity == 262151 {
			if !lowerHash(got) || m.failed || m.actual != (candidateWorkVector{10: 1576}) {
				t.Fatal("old exact512 preflight failed the closed encoder", got, m.actual)
			}
			continue
		}
		want := candidateWorkVector{}
		if capacity >= 6 {
			want[10] = 6
		}
		if got != "" || !m.failed || m.actual != want {
			t.Fatal("original copy/preflight boundary lost its failure prefix", capacity, got, m.actual, want)
		}
		if retry := providerHistoryCollectionTypedSpineDigestObserved("", 0, historyCollectionSpineTestHeader(), m); retry != "" || m.actual != want {
			t.Fatal("sticky failed observer was revived", capacity, retry, m.actual)
		}
	}
}
