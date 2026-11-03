package registry011

import "testing"

func TestProviderHistoryTLSRootDataParsingPreservesTheSelectedDigestAndRefusesInvalidShapes(t *testing.T) {
	digest := admissionTestDigest("c")
	for _, raw := range []string{
		`{"provider":{"tls_roots_artifact":"` + digest + `"}}`,
		`{`, `[]`, `{}`, `{"provider":[]}`, `{"provider":{"tls_roots_artifact":7}}`,
		`{"provider":{"tls_roots_artifact":"not-a-digest"}}`,
	} {
		m := providerReplayUnitMeter()
		got := providerHistoryTLSRootsObserved(raw, m)
		want := ""
		if raw == `{"provider":{"tls_roots_artifact":"`+digest+`"}}` {
			want = digest
		}
		if got != want || m.failed || m.actual[2] != uint64(len(raw)) || m.actual[10] == 0 ||
			m.historyGroup != nil || m.historyRecovery != nil || m.authority {
			t.Fatal("private policy DATA parser accepted an invalid shape, changed the digest or acquired a role", got, want)
		}
	}
}

func TestProviderHistoryTLSRootDataParsingCannotBorrowFuelFromARefusedObserver(t *testing.T) {
	raw := `{"provider":{"tls_roots_artifact":"` + admissionTestDigest("c") + `"}}`
	for _, mode := range []string{"copy", "codec"} {
		m := providerReplayUnitMeter()
		// Each allowance belongs to a new private arithmetic subject and is
		// selected before its first observation. No live owner is resized.
		if mode == "copy" {
			m.upper[10] = 0
		} else {
			m.upper[2] = 0
		}
		if providerHistoryTLSRootsObserved(raw, m) != "" || !m.failed || m.actual[2] != 0 ||
			m.historyGroup != nil || m.historyRecovery != nil || m.authority {
			t.Fatal("unfunded private policy parser supplied a TLS root or execution role", mode)
		}
		charged := m.actual
		if providerHistoryTLSRootsObserved(raw, m) != "" || m.actual != charged {
			t.Fatal("refused private policy parser renewed its observer", mode)
		}
	}
}
