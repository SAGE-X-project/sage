package registry011

import (
	"crypto/sha256"
	"encoding/base64"
	"sort"
	"strconv"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// State is private to one replay invocation; failures publish no snapshots.
type replayState struct {
	record            replayRecord
	grants            []replayGrant
	version, admitted uint64
}

func (s *replayState) apply(registry string, c replayCommand, now uint64) (HistoryContentStep, bool) {
	var none HistoryContentStep
	if now > 1<<53-1 || now < s.admitted || s.version == ^uint64(0) || c.expected != s.version {
		return none, false
	}
	added := []replayKey{}
	if c.op == "create" {
		if s.version != 0 || !replayHasSigner(c.keys, now) {
			return none, false
		}
		s.record = replayRecord{ID: "did:sage:" + registry + ":" + c.agent, Controller: c.actor, Keys: append([]replayKey{}, c.keys...), Services: append([]replayService{}, c.services...), State: "created"}
		s.grants = []replayGrant{}
		added = c.keys
	} else {
		if s.version == 0 || s.record.State == "deactivated" || !s.authorized(c) {
			return none, false
		}
		switch c.op {
		case "activate":
			if s.record.State != "created" || !replayHasSigner(s.record.Keys, now) {
				return none, false
			}
			s.record.State = "active"
		case "deactivate":
			s.record.State = "deactivated"
		case "add-key":
			if s.record.State != "active" || len(s.record.Keys) >= 128 {
				return none, false
			}
			k := c.keys[0]
			for _, old := range s.record.Keys {
				if old.Name == k.Name || old.Key == k.Key {
					return none, false
				}
			}
			for _, svc := range s.record.Services {
				if svc.Name == k.Name {
					return none, false
				}
			}
			s.record.Keys = append(s.record.Keys, k)
			sort.Slice(s.record.Keys, func(i, j int) bool { return s.record.Keys[i].Name < s.record.Keys[j].Name })
			added = c.keys
		case "revoke-key":
			found := false
			for i, k := range s.record.Keys {
				if k.Name == c.args["name"] && k.State == "accepted" {
					s.record.Keys[i].State = "revoked"
					found = true
					break
				}
			}
			if !found {
				return none, false
			}
			if !replayHasSigner(s.record.Keys, now) {
				s.record.State = "deactivated"
			}
		case "update-services":
			if s.record.State != "active" {
				return none, false
			}
			services, ok := replayServices(c.args["services"], s.record.Keys)
			if !ok {
				return none, false
			}
			s.record.Services = services
		case "authorize-operator", "revoke-operator":
			g := replayGrant{contentString(c.args["operator"]), contentString(c.args["scope"])}
			if g.Operator == s.record.Controller {
				return none, false
			}
			index := -1
			for i, old := range s.grants {
				if old == g {
					index = i
					break
				}
			}
			if c.op == "authorize-operator" {
				allowed := g.Scope == "deactivate" || (s.record.State == "created" && g.Scope == "activate") || (s.record.State == "active" && (g.Scope == "add-key" || g.Scope == "revoke-key" || g.Scope == "update-services"))
				if !allowed || index >= 0 || len(s.grants) >= 128 {
					return none, false
				}
				s.grants = append(s.grants, g)
				replaySortGrants(s.grants)
			} else {
				if index < 0 {
					return none, false
				}
				s.grants = append(s.grants[:index], s.grants[index+1:]...)
			}
		default:
			return none, false
		}
	}
	previous := ""
	if s.version > 0 {
		previous = strconv.FormatUint(s.version, 10)
	}
	s.version++
	s.admitted = now
	s.record.Version = strconv.FormatUint(s.version, 10)
	removed := []replayGrant{}
	retained := []replayGrant{}
	for _, g := range s.grants {
		if s.record.State == "deactivated" || (c.op == "activate" && g.Scope == "activate") {
			removed = append(removed, g)
		} else {
			retained = append(retained, g)
		}
	}
	s.grants = retained
	raw, e := rfc8785.Marshal(s.record)
	if e != nil || len(raw) > 65536 {
		return none, false
	}
	// Preserve exact terminal/version-exhaustion headroom before crypto work.
	if s.record.State != "deactivated" {
		future := s.record
		future.State = "deactivated"
		future.Version = "18446744073709551615"
		b, e := rfc8785.Marshal(future)
		if e != nil || len(b) > 65536 {
			return none, false
		}
	}
	if !replayVerifyNewKeys(registry, c.agent, s.record.Keys, added, now) {
		return none, false
	}
	grants, _ := rfc8785.Marshal(s.grants)
	var target any
	if c.op == "authorize-operator" || c.op == "revoke-operator" {
		target = c.args
	}
	entry, _ := rfc8785.Marshal(map[string]any{"version": s.record.Version, "previousVersion": previous, "actor": c.actor, "operation": c.op, "commandDigest": replayDigest(c.raw), "target": target, "automaticRemovals": removed, "admitted": now, "recordDigest": replayDigest(raw), "grantsDigest": replayDigest(grants)})
	return HistoryContentStep{History: HistoryBytes{Version: s.version, Entry: entry, Command: append([]byte{}, c.raw...)}, Record: raw, Grants: grants}, true
}
func (s *replayState) authorized(c replayCommand) bool {
	if c.actor == s.record.Controller {
		return true
	}
	if c.op == "authorize-operator" || c.op == "revoke-operator" || (c.op == "revoke-key" && s.record.State != "active") {
		return false
	}
	for _, g := range s.grants {
		if g.Operator == c.actor && g.Scope == c.op {
			return true
		}
	}
	return false
}
func replayDigest(raw []byte) string {
	h := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(h[:])
}
