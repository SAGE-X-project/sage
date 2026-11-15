package registry011

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistorySettlementReadbackRefusesMissingOrCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, input := range []context.Context{nil, ctx} {
		for _, group := range []*providerHistoryGroupOwner{nil, {}} {
			if model, prefix, err := group.readSettlementLocked(input, nil, false); err == nil || model != nil || prefix != nil {
				t.Fatal("absent context supplied a successful settlement model", err)
			}
			if model, err := group.readFailedSettlementLocked(input, nil, false); err == nil || model != nil {
				t.Fatal("absent context supplied a failed settlement model", err)
			}
			if err := group.retainFailedCaptureLocked(input, nil, nil); err == nil {
				t.Fatal("absent context retained a failure acknowledgment")
			}
		}
	}
}

// Faults affect only newly written files in this test's ordinary owned
// directories, after the physical S rename. Original C and its successful
// prefix stay unchanged. No failed lifetime is reset or retried.
func TestProviderHistoryFailedSettlementRequiresPostRenameOriginals(t *testing.T) {
	for _, mode := range []string{"intact", "failure", "results", "settlement", "inventory", "highwater", "intent", "pending", "commit", "frontier", "failure-stage"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 2)
			historyRetainNormalPrefix(t, o, d, 1)
			s, m, recovery := o.store, o.meter, o.parent.recovery
			pin, ack, held, reads, work := o.pin, o.last, s.stored, s.clock.read, m.actual
			confirmed := recovery.confirmed
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			var captured *providerHistoryWriteClaim
			reached := false
			syncFile := s.custody.syncFile
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				claim := s.frontierFiles.historyPending
				info, err := file.Stat()
				if reached || err != nil || !info.IsDir() || claim == nil || claim.failure.kind != "failure" {
					return err
				}
				head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
				if err != nil {
					return err
				}
				if string(head) != claim.highwater {
					return nil
				}
				if claim.count != 3 || claim.results.kind != "results" || claim.dispatch != d || claim.checkpoint == nil ||
					claim.consumed != ack || d.retained != 1 || d.attempt == nil || d.attempt.failure == nil {
					t.Fatal("ordinary known failure and completed original prefix required")
				}
				captured = claim
				if mode == "intact" {
					reached = true
					return nil
				}
				if mode == "failure-stage" {
					// Only the expected private DATA differs from the original
					// recorded failure. Do not change its provenance or flags.
					if stage := d.attempt.failure.failureStage; stage == "" || stage == "different-private-stage" {
						t.Fatal("ordinary recorded failure stage required before the DATA fault")
					}
					d.attempt.failure.failureStage = "different-private-stage"
					reached = true
					return nil
				}
				directory, name := s.config.evidenceDirectory, ""
				switch mode {
				case "failure":
					name = providerHistoryStoredName(claim.failure, nil)
				case "results":
					name = providerHistoryStoredName(claim.results, nil)
				case "settlement":
					name = providerHistoryStoredName(claim.settlement, nil)
				case "inventory":
					name = providerHistoryInventoryName(claim.inventory.digest, nil)
				case "highwater":
					directory, name = s.config.custodyDirectory, s.name("highwater", claim.frame.after.ordinal)
				case "pending":
					name = s.name("history-pending", claim.frame.after.sequence)
				default:
					directory, name = s.config.custodyDirectory, s.name(mode, claim.frame.after.sequence)
				}
				path := filepath.Join(directory, name)
				entry, err := os.Lstat(path)
				if err != nil || name == "" || !entry.Mode().IsRegular() {
					t.Fatal("owned newly retained regular original required", mode, err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal("owned original removal failed", mode, err)
				}
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("selected original must actually be absent after removal", mode, err)
				}
				reached = true
				return nil
			}
			member := d.coordinator.ticket.table.members[1]
			connection := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, connection)
			if err == nil || out != nil || !reached || captured == nil || !captured.renamed ||
				!o.failed || !o.parent.failed || !s.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				s.stored != held || s.clock.read <= reads || d.retained != 1 || o.order.next != 1 {
				t.Fatal("known failed member lost its terminal original outcome", mode, err)
			}
			for coordinate, before := range work {
				if m.actual[coordinate] < before {
					t.Fatal("failed settlement refunded original work", coordinate)
				}
			}
			if mode == "intact" {
				if o.last == nil || o.last == ack || o.last.claim != captured || !captured.used || o.pin == pin || recovery.pending != nil ||
					recovery.confirmed != captured.checkpoint || s.frontierFiles.historyPending != nil {
					t.Fatal("intact ordinary failed S must retain its terminal acknowledgment")
				}
			} else if o.pin != pin || o.last != ack || s.frontierFiles.pin != pin || captured.used ||
				recovery.confirmed != confirmed || recovery.pending != captured.checkpoint || s.frontierFiles.historyPending != captured {
				t.Fatal("missing or inconsistent post-rename original published an S acknowledgment", mode)
			}
			after := providerHistoryProvisionedOriginalBytes(t, s.config)
			for name, before := range originals {
				if name == filepath.Join(s.config.custodyDirectory, "HEAD") {
					continue // The physical alias advanced; selected C did not.
				}
				if got, ok := after[name]; !ok || got != before {
					t.Fatal("failed S changed the original C prefix", name)
				}
			}
			head, err := s.custody.root.ReadFile("HEAD")
			if err != nil || string(head) != captured.highwater {
				t.Fatal("physical S must remain visible after a readback refusal", err)
			}
		})
	}
}
