package registry010

import "context"

// WebRegistryWriteState010 is the complete state supplied and replaced by one
// trusted Registry transaction. Envelope is the committed response for the
// current record; the store refreshes its response timestamps for the
// transaction snapshot. History contains every committed version at its mutation time.
// A deactivated identifier remains reserved by Tombstoned forever.
type WebRegistryWriteState010 struct {
	Source     string                        `json:"source"`
	Envelope   []byte                        `json:"envelope"`
	History    []WebRegistryHistoryEntry010  `json:"history"`
	Grants     []WebRegistryOperatorGrant010 `json:"grants,omitempty"`
	Tombstoned bool                          `json:"tombstoned"`
}

// WebRegistryWriteSnapshot010 binds the current state to credentials and
// controller-authorized delegation read inside the same transaction.
type WebRegistryWriteSnapshot010 struct {
	State     WebRegistryWriteState010
	Authority WebRegistryAdminAuthority010
}

// WebRegistryWriteStore010 must serialize writes for each identifier. It must
// obtain Snapshot from its authenticated source with an envelope fresh at now,
// call decide exactly once while
// holding the transaction, and durably replace the complete state only if
// decide returns nil and ctx is still live. A decision error leaves state
// unchanged. An I/O failure must quarantine the store because commit status
// may be uncertain.
// The binding must never build Authority from caller-supplied JSON fields.
type WebRegistryWriteStore010 interface {
	Update(context.Context, string, int64, func(WebRegistryWriteSnapshot010) (WebRegistryWriteState010, error)) error
}

func webCopyWriteState010(state WebRegistryWriteState010) WebRegistryWriteState010 {
	copyState := state
	copyState.Envelope = append([]byte(nil), state.Envelope...)
	copyState.Grants = append([]WebRegistryOperatorGrant010(nil), state.Grants...)
	copyState.History = make([]WebRegistryHistoryEntry010, len(state.History))
	for index, entry := range state.History {
		copyState.History[index] = WebRegistryHistoryEntry010{
			Envelope: append([]byte(nil), entry.Envelope...), At: entry.At, Operation: entry.Operation,
			Actor: entry.Actor, Target: entry.Target, Scope: entry.Scope,
		}
	}
	return copyState
}

// ApplyWebRegistryWrite010 builds one complete next state within a deployment
// transaction. trustedSource is local configuration, never request data. This
// function does not itself implement a durable store or credential verifier.
func ApplyWebRegistryWrite010(ctx context.Context, store WebRegistryWriteStore010, trustedSource, did string, candidate []byte, now int64, expectedVersion, operation string) error {
	if ctx == nil || ctx.Err() != nil || store == nil || trustedSource == "" || did == "" {
		return ErrRejected
	}
	return store.Update(ctx, did, now, func(snapshot WebRegistryWriteSnapshot010) (WebRegistryWriteState010, error) {
		state := snapshot.State
		if ctx.Err() != nil || state.Source != trustedSource || snapshot.Authority == nil {
			return WebRegistryWriteState010{}, ErrUnreachable
		}
		if len(state.Envelope) == 0 {
			if len(state.History) != 0 || len(state.Grants) != 0 || state.Tombstoned || operation != "create" || expectedVersion != "" {
				return WebRegistryWriteState010{}, ErrStale
			}
			if err := CheckWebRegistryCreationAdmission010(ctx, snapshot.Authority, candidate, did, now); err != nil {
				return WebRegistryWriteState010{}, err
			}
		} else {
			if len(state.History) == 0 || operation == "create" {
				return WebRegistryWriteState010{}, ErrInvalidRecord010
			}
			last := state.History[len(state.History)-1]
			if last.At > now || CheckWebRegistryHistoryContinuity010(state.History, last.Envelope, did, last.At) != nil ||
				webCheckGrantHistory010(state, did) != nil {
				return WebRegistryWriteState010{}, ErrInvalidRecord010
			}
			historical, err := webTransitionRecordFromEnvelope010(last.Envelope, did, last.At)
			if err != nil {
				return WebRegistryWriteState010{}, err
			}
			current, err := webTransitionRecordFromEnvelopeWithPolicy010(state.Envelope, did, now, false)
			if err != nil || !webSameRecord010(historical, current) || state.Tombstoned != (current.State == "deactivated") {
				return WebRegistryWriteState010{}, ErrInvalidRecord010
			}
			if expectedVersion != current.Version {
				return WebRegistryWriteState010{}, ErrStale
			}
			if err := checkWebRegistryTransitionShapeWithPolicy010(state.Envelope, candidate, did, now, now, operation, false); err != nil {
				return WebRegistryWriteState010{}, err
			}
		}
		actor, err := webAdminActor010(ctx, snapshot.Authority)
		if err != nil {
			return WebRegistryWriteState010{}, err
		}
		var before webTransitionRecord010
		if len(state.Envelope) != 0 {
			before, err = webTransitionRecordFromEnvelopeWithPolicy010(state.Envelope, did, now, false)
			if err != nil {
				return WebRegistryWriteState010{}, err
			}
		}
		if ctx.Err() != nil {
			return WebRegistryWriteState010{}, ErrUnreachable
		}
		next := webCopyWriteState010(state)
		next.Envelope = append([]byte(nil), candidate...)
		record, err := webTransitionRecordFromEnvelope010(candidate, did, now)
		if err != nil {
			return WebRegistryWriteState010{}, err
		}
		entry := WebRegistryHistoryEntry010{Envelope: append([]byte(nil), candidate...), At: now, Operation: operation, Actor: actor}
		if len(state.Envelope) == 0 {
			if actor != record.Controller {
				return WebRegistryWriteState010{}, ErrRejected
			}
		} else {
			next.Grants, err = webNextGrants010(state.Grants, before, record, entry)
			if err != nil {
				return WebRegistryWriteState010{}, err
			}
		}
		next.History = append(next.History, entry)
		next.Tombstoned = record.State == "deactivated"
		return next, nil
	})
}
