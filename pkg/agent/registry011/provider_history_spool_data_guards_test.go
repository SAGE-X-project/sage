package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryMissingSpoolOwnerCannotSelectOrReadOriginals(t *testing.T) {
	var d *providerHistoryGroupDispatch
	var a *providerHistoryDispatchAttempt
	if d.originalNameLocked(0, "response") != "" || d.rootsObservedLocked() != "" || d.memberCatalogLocked() || d.wholeResultPackFitsLocked() {
		t.Fatal("absent original owner selected a spool name, policy root or catalog")
	}
	if names, ok := d.originalCatalogNamesLocked(); ok || names[0] != nil || names[1] != nil {
		t.Fatal("absent original dispatch produced an allowed-name catalog")
	}
	if out, err := a.retainOriginalCandidateLocked(nil); err == nil || out != nil {
		t.Fatal("absent attempt acknowledged original retention")
	}
	if out, err := d.readMemberPrefixCandidateLocked(admissionTestDigest("a")); err == nil || out != nil {
		t.Fatal("absent dispatch read a live member prefix")
	}
	if out, err := d.readOriginalPrefixCandidateLocked(1, admissionTestDigest("a")); err == nil || out != nil {
		t.Fatal("absent dispatch read an archived original prefix")
	}
}

func TestProviderHistoryResultPackCapacityRemainsDataWithoutAnExecutionOwner(t *testing.T) {
	for _, mode := range []string{"same", "header-cap", "member-capture", "member-wire", "second-member-cap"} {
		t.Run(mode, func(t *testing.T) {
			ticket := *historyRecordTicket(t, 2, 1, admissionTestDigest("e"))
			tls := providerTLSBounds{2, 4, 8 << 10}
			store := &providerCustodyStore{config: providerCustodyConfig{bounds: providerIntegrationBounds{maximumObjectBytes: providerHistoryOriginalPackBytes}}}
			parent := &providerHistoryProgramOwner{tls: tls}
			group := &providerHistoryGroupOwner{store: store, parent: parent}
			d := &providerHistoryGroupDispatch{group: group, coordinator: &providerHistoryGroupLedger{ticket: &ticket}, order: &providerHistoryGroupOrder{count: 2}}
			if !d.wholeResultPackFitsLocked() || d.owner != nil || group.owner != nil || store.activeMeter.Load() != nil {
				t.Fatal("positive finite result-pack capacity DATA required")
			}
			// All values are private, unopened DATA. No original store, witness,
			// timer, dispatch row or selected execution allowance is changed.
			header := uint64(len(providerHistoryOriginalPackMagic) + 4 + providerHistoryOriginalPackHeaderBytes)
			switch mode {
			case "header-cap":
				store.config.bounds.maximumObjectBytes = header - 1
			case "member-capture":
				ticket.table.members[0].capture.Timeout = 0
			case "member-wire":
				ticket.table.members[0].capture.HeaderBytes = 1
				ticket.table.members[0].capture.WireBytes = len(ticket.table.members[0].request.wire)
				if !ticket.table.members[0].capture.valid() {
					t.Fatal("wire-only refusal must retain a valid numeric capture shape")
				}
			case "second-member-cap":
				member := ticket.table.members[0]
				response := uint64(member.capture.WireBytes - len(member.request.wire))
				identity := uint64(2048 + tls.chains*tls.certificates*70 + tls.chains*4)
				store.config.bounds.maximumObjectBytes = header + response + identity
			}
			before := ticket
			if got := d.wholeResultPackFitsLocked(); got != (mode == "same") || ticket != before || d.owner != nil || group.owner != nil ||
				group.meter != nil || parent.group != nil || store.activeMeter.Load() != nil || store.clock != nil || store.custody != nil || store.evidence != nil {
				t.Fatal("result-pack capacity accepted a deficient private input or created an execution role", mode, got)
			}
			// Positive capacity DATA never opens the actual retained-byte lane.
			if out, err := d.readOriginalPrefixCandidateLocked(1, admissionTestDigest("a")); err == nil || out != nil {
				t.Fatal("unowned result-pack capacity authorized an original read", mode)
			}
			if out, err := group.retainMemberCandidate(context.Background(), d, 0); err == nil || out != nil || store.activeMeter.Load() != nil {
				t.Fatal("unowned capacity DATA authorized a native member", mode)
			}
		})
	}
}
