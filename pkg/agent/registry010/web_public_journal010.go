package registry010

import "strings"

// PublicEnvelope010 creates one fresh public response from this journal's
// current committed state. The caller must supply a trusted clock and serve
// these exact bytes only through the configured authenticated HTTPS origin.
// This local reference does not itself bind a deployed public server.
func (j *WebRegistryWriteJournal010) PublicEnvelope010(did, source string, now int64) ([]byte, error) {
	if j == nil {
		return nil, ErrUnreachable
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.failed || j.file == nil || len(j.state.History) == 0 ||
		did != j.did || source != j.source {
		return nil, ErrUnreachable
	}
	url, err := WebRegistryRequestURL010(did, []string{source})
	if err != nil || !strings.HasPrefix(url, source+"/.well-known/sage/agents/") {
		return nil, ErrUnreachable
	}
	fresh, err := webWriteJournalRefresh010(j.state.Envelope, now)
	if err != nil {
		return nil, err
	}
	if _, err = webTransitionRecordFromEnvelopeWithPolicy010(fresh, did, now, false); err != nil {
		return nil, err
	}
	return fresh, nil
}
