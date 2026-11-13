package registry011

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// This context only changes one response file owned by the enclosing UNIT,
// once the ordinary member capture has produced a known failure. It supplies
// no alternate context, transport, request, limit, clock or recovery role.
type historyFailedPrefixDataContext struct {
	context.Context
	t       *testing.T
	group   *providerHistoryGroupOwner
	path    string
	changed bool
}

func (c *historyFailedPrefixDataContext) Err() error {
	if !c.changed && c.group.phase == "capturing-member" && c.group.dispatch != nil &&
		c.group.dispatch.attempt != nil && c.group.dispatch.attempt.failure != nil {
		historyChangeOwnedPrefixResponse(c.t, c.path)
		c.changed = true
	}
	return c.Context.Err()
}

func historyChangeOwnedPrefixResponse(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || raw[0] != 'H' {
		t.Fatal("ordinary nonempty owned HTTP prefix response required", err)
	}
	raw[0] = '!'
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal("finite same-width owned response DATA fault failed", err)
	}
}

func TestProviderHistoryFailedSettlementRechecksItsActualCompletedPrefix(t *testing.T) {
	for _, mode := range []string{"before-preparation", "before-readback"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 2)
			historyRetainNormalPrefix(t, o, d, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			setupWork, setupClocks, setupReads := m.actual, m.clocks, s.clock.read
			sealed := r.members[0]
			if d.retained != 1 || d.originals[0] == nil || sealed == nil || sealed.capture == nil || sealed.capture.intent == nil ||
				sealed.capture.original != d.originals[0] || sealed.capture.intent.dispatch != d || sealed.capture.intent.ordinal != 0 ||
				sealed.capture.intent.responseName == "" || m.failed || m.closed || s.activeMeter.Load() != nil {
				t.Fatal("actual completed first member and original sealed response selection required")
			}
			// Filename selection here is UNIT setup while the original meter is
			// idle. Reuse the completed original's immutable receipt; invoking
			// originalNameLocked would spend observed work outside its role.
			path := filepath.Join(s.config.evidenceDirectory, sealed.capture.intent.responseName)
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			if m.actual != setupWork || m.clocks != setupClocks || s.clock.read != setupReads ||
				m.failed || m.closed || s.activeMeter.Load() != nil {
				t.Fatal("owned filename setup changed the original idle observer")
			}
			ack, pin, confirmed, held, work, upper := o.last, o.pin, r.confirmed, s.stored, m.actual, m.upper
			syncFile := s.custody.syncFile
			defer func() { s.custody.syncFile = syncFile }()
			changed := false
			input := &historyFailedPrefixDataContext{Context: context.Background(), t: t, group: o, path: path}
			var ctx context.Context = input
			if mode == "before-readback" {
				ctx = context.Background()
				s.custody.syncFile = func(file *os.File) error {
					if err := syncFile(file); err != nil {
						return err
					}
					c := s.frontierFiles.historyPending
					if changed || c == nil || c.frame == nil || c.failure.kind != "failure" || filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
						return nil
					}
					if c.used || c.renamed || c.count != 3 || !o.claimActiveLocked(c) {
						t.Fatal("actual unrenamed failed-S originals required")
					}
					historyChangeOwnedPrefixResponse(t, path)
					changed = true
					return nil
				}
			}
			member := d.coordinator.ticket.table.members[1]
			connection := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(ctx, d, 1, connection)
			if mode == "before-preparation" {
				changed = input.changed
			}
			if err == nil || out != nil || !changed || o.last != ack || o.pin != pin || s.frontierFiles.pin != pin ||
				r.confirmed != confirmed || s.stored != held || m.upper != upper || m.actual[0] != work[0]+1 ||
				!o.failed || !o.parent.failed || !s.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				d.retained != 1 || o.order.next != 1 || o.order.state != providerHistoryOrderFailed {
				t.Fatal("changed prefix became a failed-S acknowledgment or renewed execution", mode, err)
			}
			if mode == "before-preparation" {
				if r.pending != nil || s.frontierFiles.historyPending != nil || r.pendingAck != nil {
					t.Fatal("invalid prefix was admitted into a new S claim")
				}
			} else {
				c := s.frontierFiles.historyPending
				if c == nil || c.count != 3 || c.used || c.renamed || c.checkpoint == nil || r.pending != c.checkpoint || r.pendingAck != c.checkpoint.ack {
					t.Fatal("unacknowledged S lost its original pending evidence")
				}
			}
			after := providerHistoryProvisionedOriginalBytes(t, s.config)
			for original, raw := range before {
				if original == path {
					if after[original] == raw || len(after[original]) != len(raw) {
						t.Fatal("intended same-width owned response fault was not retained")
					}
					continue
				}
				if got, exists := after[original]; !exists || got != raw {
					t.Fatal("prefix integrity refusal changed another original", original)
				}
			}
			for coordinate, value := range work {
				if m.actual[coordinate] < value {
					t.Fatal("failed prefix inspection refunded original work", coordinate)
				}
			}
		})
	}
}
