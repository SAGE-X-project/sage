package registry011

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

// Fixed in-memory refusal DATA only. None of these tests opens a connection,
// injects bytes into a peer, creates an attack driver or selects a live role.
func TestProviderHistoryFailureLayoutRejectsUnboundedMetadata(t *testing.T) {
	for _, mode := range []string{"short", "zero-header", "oversized-header", "missing-header-bytes"} {
		raw := append([]byte(providerHistoryFailureMagic), make([]byte, 4)...)
		switch mode {
		case "short":
			raw = raw[:len(raw)-1]
		case "oversized-header":
			binary.BigEndian.PutUint32(raw[len(providerHistoryFailureMagic):], providerHistoryFailureHeaderBytes+1)
		case "missing-header-bytes":
			binary.BigEndian.PutUint32(raw[len(providerHistoryFailureMagic):], 1)
		}
		if metadata, payload, ok := providerHistoryFailureLayout(string(raw)); ok || metadata != 0 || payload != 0 {
			t.Fatal("invalid finite failure layout accepted", mode)
		}
	}
}

func TestProviderHistoryFailureCapacityRequiresEveryMemberAndTLSBound(t *testing.T) {
	for _, mode := range []string{"same", "missing-ticket", "missing-parent", "invalid-tls", "small-object", "invalid-member"} {
		t.Run(mode, func(t *testing.T) {
			ticket, _, _, _, bounds := historyFailureDataFixture(t, "read")
			d := &providerHistoryGroupDispatch{coordinator: &providerHistoryGroupLedger{ticket: ticket}, order: &providerHistoryGroupOrder{count: 1},
				group: &providerHistoryGroupOwner{store: &providerCustodyStore{}, parent: &providerHistoryProgramOwner{tls: bounds}}}
			d.group.store.config.bounds.maximumObjectBytes = providerHistoryOriginalPackBytes
			switch mode {
			case "missing-ticket":
				d.coordinator.ticket = nil
			case "missing-parent":
				d.group.parent = nil
			case "invalid-tls":
				d.group.parent.tls.chains = 0
			case "small-object":
				d.group.store.config.bounds.maximumObjectBytes = 1
			case "invalid-member":
				ticket.table.members[0].capture.WireBytes = 0
			}
			if got := d.failureOriginalFitsLocked(); got != (mode == "same") || d.group.parent != nil && d.group.parent.group != nil || d.attempt != nil {
				t.Fatal("incomplete scalar failure capacity accepted or selected an attempt", mode, got)
			}
		})
	}
}

func TestProviderHistoryFailureTLSRejectsIncompleteEvidenceBeforePublication(t *testing.T) {
	for _, mode := range []string{"missing-sample", "unsupported-cipher", "empty-chain", "invalid-digest"} {
		t.Run(mode, func(t *testing.T) {
			ticket, _, data, _, bounds := historyFailureDataFixture(t, "read")
			switch mode {
			case "missing-sample":
				data.sample = nil
			case "unsupported-cipher":
				data.sample.CipherSuite = 0
			case "empty-chain":
				data.sample.VerifiedChains = [][]string{{}}
			case "invalid-digest":
				data.sample.VerifiedChains = [][]string{{"invalid-unit-digest"}}
			}
			m := historyGroupCodecMeter()
			if out, ok := providerHistoryFailureTLSObserved(data.sample, &ticket.table.members[0].request, bounds, m); ok || out != nil || m.failed ||
				m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 {
				t.Fatal("incomplete TLS failure DATA accepted or produced an effect", mode)
			}
		})
	}
}

func TestProviderHistoryFailureOriginalRejectsInconsistentCapturePrefixes(t *testing.T) {
	for _, mode := range []string{"negative-written", "dial-with-sample", "missing-sample", "incomplete-write", "invalid-tls", "unknown-stage", "small-object"} {
		t.Run(mode, func(t *testing.T) {
			ticket, consumption, data, roots, bounds := historyFailureDataFixture(t, "read")
			maximum := uint64(providerHistoryOriginalPackBytes)
			switch mode {
			case "negative-written":
				data.requestWritten = -1
			case "dial-with-sample":
				data.failureStage = "dial"
			case "missing-sample":
				data.sample = nil
			case "incomplete-write":
				data.requestWritten--
			case "invalid-tls":
				data.sample.CipherSuite = 0
			case "unknown-stage":
				data.failureStage = "unsupported-unit-stage"
			case "small-object":
				maximum = 1
			}
			m := historyGroupCodecMeter()
			if out, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, bounds, maximum, m); err == nil || out != nil ||
				m.failed || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 {
				t.Fatal("inconsistent private failure prefix accepted or produced an effect", mode, err)
			}
		})
	}
}

func TestProviderHistoryFailureParserRejectsIncompleteTLSMetadata(t *testing.T) {
	ticket, consumption, data, roots, bounds := historyFailureDataFixture(t, "read")
	original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, bounds, providerHistoryOriginalPackBytes, nil)
	if err != nil || original == nil {
		t.Fatal("ordinary finite failure precursor", err)
	}
	for _, mode := range []string{"no-chains", "empty-chain", "invalid-digest"} {
		t.Run(mode, func(t *testing.T) {
			raw := []byte(original.raw)
			start := len(providerHistoryFailureMagic) + 4
			size := int(binary.BigEndian.Uint32(raw[len(providerHistoryFailureMagic):start]))
			var header map[string]any
			if json.Unmarshal(raw[start:start+size], &header) != nil {
				t.Fatal("original bounded header precursor")
			}
			metadata, ok := header["tls"].(map[string]any)
			if !ok {
				t.Fatal("original TLS metadata precursor")
			}
			switch mode {
			case "no-chains":
				metadata["verified_chains"] = []any{}
			case "empty-chain":
				metadata["verified_chains"] = []any{[]any{}}
			case "invalid-digest":
				metadata["verified_chains"] = []any{[]any{"invalid-unit-digest"}}
			}
			encoded := providerReplayCanonical(header, providerHistoryFailureHeaderBytes, nil)
			if len(encoded) == 0 {
				t.Fatal("finite UNIT metadata encoding")
			}
			changed := append([]byte(providerHistoryFailureMagic), make([]byte, 4)...)
			binary.BigEndian.PutUint32(changed[len(providerHistoryFailureMagic):start], uint32(len(encoded)))
			changed = append(changed, encoded...)
			changed = append(changed, raw[start+size:]...)
			m := historyGroupCodecMeter()
			if out, err := parseProviderHistoryFailureOriginalObserved(changed, original.digest, ticket, consumption, 0, roots, bounds, providerHistoryOriginalPackBytes, m); err == nil || out != nil ||
				m.failed || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 {
				t.Fatal("incomplete private TLS metadata accepted or produced an effect", mode, err)
			}
		})
	}
}
