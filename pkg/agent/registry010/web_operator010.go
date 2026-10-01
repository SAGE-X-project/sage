package registry010

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
)

// WebRegistryOperatorGrant010 is one active, exact-byte management grant.
// It is never a public record member or a transport credential.
type WebRegistryOperatorGrant010 struct {
	Operator string `json:"operator"`
	Scope    string `json:"scope"`
}

func webOperatorScope010(scope, state string) bool {
	switch state {
	case "created":
		return scope == "activate" || scope == "deactivate"
	case "active":
		return scope == "add-key" || scope == "revoke-key" || scope == "update-services" || scope == "deactivate"
	default:
		return false
	}
}

func webValidOperator010(operator, controller string) bool {
	return operator != controller && webASCII010(operator, 1, 256)
}

func webGrantLess010(a, b WebRegistryOperatorGrant010) bool {
	if a.Operator != b.Operator {
		return a.Operator < b.Operator
	}
	return a.Scope < b.Scope
}

func webGrantPresent010(grants []WebRegistryOperatorGrant010, actor, scope string) bool {
	for _, grant := range grants {
		if grant.Operator == actor && grant.Scope == scope {
			return true
		}
	}
	return false
}

func webValidGrants010(grants []WebRegistryOperatorGrant010, controller, state string) bool {
	if len(grants) > 128 {
		return false
	}
	for i, grant := range grants {
		if !webValidOperator010(grant.Operator, controller) || !webOperatorScope010(grant.Scope, state) ||
			(i > 0 && !webGrantLess010(grants[i-1], grant)) {
			return false
		}
	}
	return true
}

func webNextGrants010(previous []WebRegistryOperatorGrant010, before, after webTransitionRecord010, entry WebRegistryHistoryEntry010) ([]WebRegistryOperatorGrant010, error) {
	if !webValidGrants010(previous, before.Controller, before.State) ||
		(entry.Actor != before.Controller && !webValidOperator010(entry.Actor, before.Controller)) {
		return nil, ErrInvalidRecord010
	}
	next := append([]WebRegistryOperatorGrant010(nil), previous...)
	management := entry.Operation == "authorize-operator" || entry.Operation == "revoke-operator"
	if management {
		if entry.Actor != before.Controller || !webValidOperator010(entry.Target, before.Controller) ||
			!webOperatorScope010(entry.Scope, before.State) {
			return nil, ErrRejected
		}
		present := webGrantPresent010(next, entry.Target, entry.Scope)
		if entry.Operation == "authorize-operator" {
			if present || len(next) == 128 {
				return nil, ErrRejected
			}
			next = append(next, WebRegistryOperatorGrant010{entry.Target, entry.Scope})
			sort.Slice(next, func(i, j int) bool { return webGrantLess010(next[i], next[j]) })
		} else {
			if !present {
				return nil, ErrRejected
			}
			for i, grant := range next {
				if grant.Operator == entry.Target && grant.Scope == entry.Scope {
					next = append(next[:i], next[i+1:]...)
					break
				}
			}
		}
	} else {
		if entry.Target != "" || entry.Scope != "" ||
			(entry.Actor != before.Controller && !webGrantPresent010(previous, entry.Actor, entry.Operation)) {
			return nil, ErrRejected
		}
		kept := next[:0]
		for _, grant := range next {
			if webOperatorScope010(grant.Scope, after.State) {
				kept = append(kept, grant)
			}
		}
		next = kept
	}
	if !webValidGrants010(next, after.Controller, after.State) {
		return nil, ErrInvalidRecord010
	}
	return next, nil
}

func webSameGrants010(a, b []WebRegistryOperatorGrant010) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

// Reconstruct authority from committed versions rather than trusting a
// separately asserted positive grant set in a store snapshot.
func webCheckGrantHistory010(state WebRegistryWriteState010, did string) error {
	var grants []WebRegistryOperatorGrant010
	var before webTransitionRecord010
	for index, entry := range state.History {
		after, err := webTransitionRecordFromEnvelopeWithPolicy010(entry.Envelope, did, entry.At, false)
		if err != nil {
			return err
		}
		if index == 0 {
			if entry.Operation != "create" || entry.Actor != after.Controller || entry.Target != "" || entry.Scope != "" {
				return ErrInvalidRecord010
			}
		} else {
			grants, err = webNextGrants010(grants, before, after, entry)
			if err != nil {
				return ErrInvalidRecord010
			}
		}
		before = after
	}
	if !webSameGrants010(grants, state.Grants) {
		return ErrInvalidRecord010
	}
	return nil
}

// ApplyWebRegistryOperatorCommand010 performs one controller-only grant or
// revoke inside the same serialized store transaction as the public record.
// The store must durably commit the returned complete state before success.
func ApplyWebRegistryOperatorCommand010(ctx context.Context, store WebRegistryWriteStore010, trustedSource, did string, now int64, expectedVersion, operation, target, scope string) error {
	if ctx == nil || ctx.Err() != nil || store == nil || trustedSource == "" || did == "" ||
		now < 0 || now > 9007199254740986 ||
		(operation != "authorize-operator" && operation != "revoke-operator") {
		return ErrRejected
	}
	return store.Update(ctx, did, now, func(snapshot WebRegistryWriteSnapshot010) (WebRegistryWriteState010, error) {
		state := snapshot.State
		if state.Source != trustedSource || snapshot.Authority == nil || len(state.History) == 0 || state.Tombstoned {
			return WebRegistryWriteState010{}, ErrUnreachable
		}
		last := state.History[len(state.History)-1]
		if last.At > now || CheckWebRegistryHistoryContinuity010(state.History, last.Envelope, did, last.At) != nil ||
			webCheckGrantHistory010(state, did) != nil {
			return WebRegistryWriteState010{}, ErrInvalidRecord010
		}
		historical, err := webTransitionRecordFromEnvelopeWithPolicy010(last.Envelope, did, last.At, false)
		if err != nil {
			return WebRegistryWriteState010{}, ErrInvalidRecord010
		}
		before, err := webTransitionRecordFromEnvelopeWithPolicy010(state.Envelope, did, now, false)
		if err != nil || !webSameRecord010(historical, before) ||
			state.Tombstoned != (before.State == "deactivated") {
			return WebRegistryWriteState010{}, ErrInvalidRecord010
		}
		if before.Version != expectedVersion || before.State == "deactivated" {
			return WebRegistryWriteState010{}, ErrStale
		}
		actor, err := webAdminActor010(ctx, snapshot.Authority)
		if err != nil || actor != before.Controller {
			return WebRegistryWriteState010{}, ErrRejected
		}
		version, err := strconv.ParseUint(before.Version, 10, 64)
		if err != nil || version == ^uint64(0) {
			return WebRegistryWriteState010{}, ErrRejected
		}
		entry := WebRegistryHistoryEntry010{At: now, Operation: operation, Actor: actor, Target: target, Scope: scope}
		after := before
		after.Version = strconv.FormatUint(version+1, 10)
		grants, err := webNextGrants010(state.Grants, before, after, entry)
		if err != nil {
			return WebRegistryWriteState010{}, err
		}
		var wrapper struct {
			Record  map[string]json.RawMessage `json:"record"`
			Issued  int64                      `json:"issued"`
			Expires int64                      `json:"expires"`
		}
		if err := webWriteJournalDecode010(state.Envelope, &wrapper); err != nil || wrapper.Record == nil {
			return WebRegistryWriteState010{}, ErrInvalidRecord010
		}
		wrapper.Record["version"] = json.RawMessage(strconv.Quote(after.Version))
		wrapper.Issued, wrapper.Expires = now, now+5
		candidate, err := json.Marshal(wrapper)
		if err != nil || checkWebRegistryTransitionShapeWithPolicy010(state.Envelope, candidate, did, now, now, operation, false) != nil {
			return WebRegistryWriteState010{}, ErrInvalidRecord010
		}
		if ctx.Err() != nil {
			return WebRegistryWriteState010{}, ErrUnreachable
		}
		next := webCopyWriteState010(state)
		next.Envelope = candidate
		entry.Envelope = candidate
		next.History = append(next.History, entry)
		next.Grants = grants
		return next, nil
	})
}

func webVerifyGrantStep010(previous, next WebRegistryWriteState010, did string) error {
	entry := next.History[len(next.History)-1]
	after, err := webTransitionRecordFromEnvelopeWithPolicy010(entry.Envelope, did, entry.At, false)
	if err != nil {
		return err
	}
	if len(previous.History) == 0 {
		if entry.Actor != after.Controller || entry.Target != "" || entry.Scope != "" || len(next.Grants) != 0 {
			return ErrInvalidRecord010
		}
		return nil
	}
	last := previous.History[len(previous.History)-1]
	before, err := webTransitionRecordFromEnvelopeWithPolicy010(last.Envelope, did, last.At, false)
	if err != nil {
		return err
	}
	want, err := webNextGrants010(previous.Grants, before, after, entry)
	if err != nil || !webSameGrants010(want, next.Grants) {
		return ErrInvalidRecord010
	}
	return nil
}
