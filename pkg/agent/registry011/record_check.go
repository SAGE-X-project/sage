package registry011

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// ErrRecordContent means record bytes are not a valid current 0.11.0 record.
var ErrRecordContent = errors.New("registry011: invalid record content")

// CheckRecord validates one current record read from the chain Registry:
// exact canonical JCS bytes, the spec 09 §1 members, bounds and ordering, and
// every key proof with the 0.11.0 REG-04 challenge. An X25519 endorsement is
// verified with the signing key it names in the same record, whatever that
// key's current state (spec 09 §4: a retained historical signer).
//
// It does not replay history, check the authorization of the mutations that
// produced the record, or establish freshness or source authority.
func CheckRecord(registryID, agentID string, raw []byte) error {
	if len(raw) == 0 || len(raw) > 65536 || !rpcAgent(agentID) {
		return ErrRecordContent
	}
	did := "did:sage:" + registryID + ":" + agentID
	r := contentObject(contentJSON(raw), "id", "controller", "keys", "services", "state", "version")
	if r == nil || contentString(r["id"]) != did || !rpcAddress(contentString(r["controller"])) {
		return ErrRecordContent
	}
	switch contentString(r["state"]) {
	case "created", "active", "deactivated":
	default:
		return ErrRecordContent
	}
	version := contentString(r["version"])
	if n, err := strconv.ParseUint(version, 10, 64); err != nil || n == 0 || strconv.FormatUint(n, 10) != version {
		return ErrRecordContent
	}
	keys, ok := replayKeysWith(r["keys"], did, true)
	if !ok {
		return ErrRecordContent
	}
	if _, ok := replayServices(r["services"], keys); !ok {
		return ErrRecordContent
	}
	signers := map[string]replayKey{}
	for _, k := range keys {
		if k.Alg == "ed25519" {
			signers[did+"#"+k.Name] = k
		}
	}
	for _, k := range keys {
		signer, ok := signers[k.Proof.Signer]
		if !ok {
			return ErrRecordContent
		}
		key, _ := replayBinary(signer.Key, 32)
		proof, _ := replayBinary(k.Proof.Value, 64)
		if !replayStrictEd(key, replayChallenge(registryID, agentID, k), proof) {
			return ErrRecordContent
		}
	}
	return nil
}

// CheckCode compares, at this pinned block, the runtime code of every
// contract in the network profile's identity with its recorded SHA-256.
func (r *PinnedAgentReads) CheckCode(ctx context.Context) error {
	if r == nil || r.rpc == nil {
		return ErrConfiguration
	}
	var identity nodeIdentity
	if json.Unmarshal(r.rpc.network.identity, &identity) != nil || len(identity.Contracts) != 13 {
		return ErrConfiguration
	}
	for _, contract := range identity.Contracts {
		code, err := r.CodeBytes(ctx, strings.ToLower(contract.Address))
		if err != nil {
			return err
		}
		if len(code) == 0 || nodeHex(sha256Digest(code)) != "0x"+contract.SHA256 {
			return ErrRPCState
		}
	}
	return nil
}
