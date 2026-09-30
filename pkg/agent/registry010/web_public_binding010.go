package registry010

import (
	"context"
	"net/netip"
	"reflect"
	"strings"
)

// ObserveWebRegistryJournal010 reads the configured HTTPS origin over a fresh,
// authenticated connection and compares its validated record with one local
// journal snapshot. It is a bounded publication consistency check, not an
// authorization for a later operation or a full REG-08 deployment binding.
func ObserveWebRegistryJournal010(ctx context.Context, journal *WebRegistryWriteJournal010, allowedOrigins []string, destination netip.AddrPort, allowedDestinations []netip.AddrPort, rootDER []byte, now int64) ([]byte, error) {
	if ctx == nil || ctx.Err() != nil || journal == nil {
		return nil, ErrUnreachable
	}
	journal.mu.Lock()
	if journal.failed || journal.file == nil || len(journal.state.History) == 0 {
		journal.mu.Unlock()
		return nil, ErrUnreachable
	}
	did, source := journal.did, journal.source
	snapshot := webCopyWriteState010(journal.state)
	journal.mu.Unlock()
	url, err := WebRegistryRequestURL010(did, allowedOrigins)
	if err != nil || !strings.HasPrefix(url, source+"/.well-known/sage/agents/") ||
		source != "https://"+strings.SplitN(strings.TrimPrefix(did, "did:sage:web:"), ":", 2)[0] {
		return nil, ErrUnreachable
	}
	observed, err := FetchWebRegistryRecord010(ctx, did, allowedOrigins, destination, allowedDestinations, rootDER, now)
	if err != nil {
		return nil, err
	}
	fresh, err := webWriteJournalRefresh010(snapshot.Envelope, now)
	if err != nil {
		return nil, err
	}
	written, err := webTransitionRecordFromEnvelopeWithPolicy010(fresh, did, now, false)
	if err != nil {
		return nil, err
	}
	public, err := webTransitionRecordFromEnvelopeWithPolicy010(observed, did, now, false)
	if err != nil {
		return nil, err
	}
	journal.mu.Lock()
	unchanged := !journal.failed && journal.file != nil &&
		reflect.DeepEqual(journal.state, snapshot)
	journal.mu.Unlock()
	if ctx.Err() != nil || !unchanged || !webSameRecord010(written, public) {
		return nil, ErrStale
	}
	return observed, nil
}
