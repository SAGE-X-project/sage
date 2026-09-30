package registry010

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
)

type webTransitionKey010 struct {
	Name  string `json:"name"`
	Alg   string `json:"alg"`
	Key   string `json:"key"`
	Proof struct {
		Signer string `json:"signer"`
		Value  string `json:"value"`
	} `json:"proof"`
	State   string      `json:"state"`
	Expires json.Number `json:"expires"`
}

type webTransitionRecord010 struct {
	ID         string                `json:"id"`
	Controller string                `json:"controller"`
	Keys       []webTransitionKey010 `json:"keys"`
	Services   []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		URI  string `json:"uri"`
	} `json:"services"`
	State   string `json:"state"`
	Version string `json:"version"`
}

func webTransitionRecordFromEnvelope010(raw []byte, did string, now int64) (webTransitionRecord010, error) {
	var wrapper struct {
		Record webTransitionRecord010 `json:"record"`
	}
	if err := CheckWebRegistryProofs010(raw, did, now); err != nil {
		return wrapper.Record, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&wrapper); err != nil {
		return wrapper.Record, ErrInvalidRecord010
	}
	return wrapper.Record, nil
}

func webUsableSigning010(record webTransitionRecord010, signer string, now int64) bool {
	for _, key := range record.Keys {
		if signer != record.ID+"#"+key.Name || key.Alg == "x25519" || key.State != "accepted" {
			continue
		}
		if key.Expires == "" {
			return true
		}
		expiry, valid := webInteger010(key.Expires)
		return valid && now < expiry
	}
	return false
}

func webHasUsableSigning010(record webTransitionRecord010, now int64) bool {
	for _, key := range record.Keys {
		if webUsableSigning010(record, record.ID+"#"+key.Name, now) {
			return true
		}
	}
	return false
}

func webSameKeyMaterial010(a, b webTransitionKey010) bool {
	if a.Name != b.Name || a.Alg != b.Alg || a.Key != b.Key || a.Proof != b.Proof ||
		(a.Expires == "") != (b.Expires == "") {
		return false
	}
	if a.Expires == "" {
		return true
	}
	aExpiry, aValid := webInteger010(a.Expires)
	bExpiry, bValid := webInteger010(b.Expires)
	return aValid && bValid && aExpiry == bExpiry
}

// CheckWebRegistryCreationShape010 checks the structural and proof conditions
// for an initial version-1 created record. The caller must independently
// authenticate the controller, bind now to the creation time, reserve the
// name, and commit it atomically.
func CheckWebRegistryCreationShape010(raw []byte, did string, now int64) error {
	record, err := webTransitionRecordFromEnvelope010(raw, did, now)
	if err != nil {
		return err
	}
	if record.State != "created" || record.Version != "1" || !webHasUsableSigning010(record, now) {
		return ErrInvalidRecord010
	}
	for _, key := range record.Keys {
		if key.Alg == "x25519" && !webUsableSigning010(record, key.Proof.Signer, now) {
			return ErrInvalidRecord010
		}
	}
	return nil
}

// CheckWebRegistryTransitionShape010 checks one proposed mutation between
// already authenticated historical record envelopes. It does not authenticate
// actors, prove an unbroken source history, or atomically write either record.
// candidateNow must be bound by that source to the historical mutation time,
// especially when a KEM endorser later expires or is revoked. Those duties
// belong to the trusted Registry deployment, not this predicate.
func CheckWebRegistryTransitionShape010(previous, candidate []byte, did string, previousNow, candidateNow int64, operation string) error {
	before, err := webTransitionRecordFromEnvelope010(previous, did, previousNow)
	if err != nil {
		return err
	}
	after, err := webTransitionRecordFromEnvelope010(candidate, did, candidateNow)
	if err != nil {
		return err
	}
	version, err := strconv.ParseUint(before.Version, 10, 64)
	if err != nil || version == ^uint64(0) || after.Version != strconv.FormatUint(version+1, 10) ||
		before.Controller != after.Controller || before.State == "deactivated" {
		return ErrInvalidRecord010
	}
	byName := make(map[string]webTransitionKey010, len(after.Keys))
	for _, key := range after.Keys {
		byName[key.Name] = key
	}
	revoked := 0
	for _, old := range before.Keys {
		current, found := byName[old.Name]
		if !found || !webSameKeyMaterial010(old, current) {
			return ErrInvalidRecord010
		}
		if old.State != current.State {
			if old.State != "accepted" || current.State != "revoked" {
				return ErrInvalidRecord010
			}
			revoked++
		}
	}
	added := len(after.Keys) - len(before.Keys)
	if added < 0 {
		return ErrInvalidRecord010
	}
	servicesEqual := reflect.DeepEqual(before.Services, after.Services)
	switch operation {
	case "activate":
		if before.State != "created" || after.State != "active" || added != 0 || revoked != 0 || !servicesEqual {
			return ErrInvalidRecord010
		}
	case "add-key":
		if before.State != "active" || after.State != "active" || added != 1 || revoked != 0 || !servicesEqual {
			return ErrInvalidRecord010
		}
		for _, key := range after.Keys {
			found := false
			for _, old := range before.Keys {
				if old.Name == key.Name {
					found = true
					break
				}
			}
			if !found && (key.State != "accepted" || (key.Alg == "x25519" && !webUsableSigning010(after, key.Proof.Signer, candidateNow))) {
				return ErrInvalidRecord010
			}
		}
	case "revoke-key":
		if before.State != "active" || added != 0 || revoked != 1 || !servicesEqual {
			return ErrInvalidRecord010
		}
		remaining := webHasUsableSigning010(after, candidateNow)
		if (remaining && after.State != "active") || (!remaining && after.State != "deactivated") {
			return ErrInvalidRecord010
		}
	case "update-services":
		if before.State != "active" || after.State != "active" || added != 0 || revoked != 0 {
			return ErrInvalidRecord010
		}
	case "deactivate":
		if (before.State != "created" && before.State != "active") || after.State != "deactivated" || added != 0 || revoked != 0 || !servicesEqual {
			return ErrInvalidRecord010
		}
	default:
		return ErrInvalidRecord010
	}
	return nil
}
