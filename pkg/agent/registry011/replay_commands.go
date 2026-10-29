package registry011

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// Exact adopted ABI bytes; tests pin both files to the specification digests.
//
//go:embed replay_contract_abi.json
var replayContractABI string

//go:embed replay_outcomes_abi.json
var replayOutcomesABI string

type replayABIKey struct {
	Name, Alg string
	Key       []byte
	Signer    string
	Proof     []byte
	HasExpiry bool
	Expires   uint64
}
type replayABIService struct {
	Name, ServiceType string
	URI               string `abi:"uri"`
}

type replayCommand struct {
	op, actor, agent string
	expected         uint64
	args             map[string]any
	keys             []replayKey
	services         []replayService
	raw, calldata    []byte
}

var replayMethods = map[string]string{
	"create": "revealClaim", "activate": "activate", "deactivate": "deactivate",
	"add-key": "addKey", "revoke-key": "revokeKey", "update-services": "updateServices",
	"authorize-operator": "authorizeOperator", "revoke-operator": "revokeOperator",
}

func replayABIs() (abi.ABI, abi.ABI, error) {
	a, e := abi.JSON(strings.NewReader(replayContractABI))
	if e != nil {
		return a, abi.ABI{}, e
	}
	b, e := abi.JSON(strings.NewReader(replayOutcomesABI))
	return a, b, e
}

func replayParseCommand(registry, agent string, raw []byte, contract abi.ABI) (replayCommand, bool) {
	c := replayCommand{agent: agent, raw: bytes.Clone(raw)}
	m := contentObject(contentJSON(raw), "binding", "registry", "did", "operation", "expectedVersion", "arguments", "actor")
	did := "did:sage:" + registry + ":" + agent
	if m == nil || m["binding"] != "sage-registry-chain-0.11.0" || m["registry"] != registry || m["did"] != did {
		return c, false
	}
	c.op, c.actor = contentString(m["operation"]), contentString(m["actor"])
	args, ok := contentArguments(c.op, m["arguments"])
	if !ok || !contentAddress(c.actor) {
		return c, false
	}
	c.args = args
	expected, ok := m["expectedVersion"].(string)
	if !ok {
		return c, false
	}
	var params []any
	if c.op == "create" {
		if expected != "" || args["controller"] != c.actor {
			return c, false
		}
		c.keys, ok = replayKeys(args["keys"], did)
		if !ok {
			return c, false
		}
		c.services, ok = replayServices(args["services"], c.keys)
		if !ok {
			return c, false
		}
		salt, ok := replayBinary(args["salt"], 32)
		if !ok {
			return c, false
		}
		params = []any{agent, common.HexToAddress(c.actor), replayABIKeys(c.keys), replayABIServices(c.services), [32]byte(salt)}
	} else {
		var e error
		c.expected, e = strconv.ParseUint(expected, 10, 64)
		if e != nil || c.expected == 0 || strconv.FormatUint(c.expected, 10) != expected {
			return c, false
		}
		params = []any{agent, c.expected}
		switch c.op {
		case "add-key":
			c.keys, ok = replayKeys([]any{args["key"]}, did)
			if !ok {
				return c, false
			}
			params = append(params, replayABIKeys(c.keys)[0])
		case "revoke-key":
			if !replayName(contentString(args["name"])) {
				return c, false
			}
			params = append(params, args["name"])
		case "update-services":
			c.services, ok = replayServices(args["services"], nil)
			if !ok {
				return c, false
			}
			params = append(params, replayABIServices(c.services))
		case "authorize-operator", "revoke-operator":
			if !contentGrant(args) {
				return c, false
			}
			params = append(params, common.HexToAddress(contentString(args["operator"])), args["scope"])
		}
	}
	var e error
	c.calldata, e = contract.Pack(replayMethods[c.op], params...)
	return c, e == nil
}

func replayABIKeys(keys []replayKey) []replayABIKey {
	out := make([]replayABIKey, len(keys))
	for i, k := range keys {
		raw, _ := replayBinary(k.Key, 32)
		proof, _ := replayBinary(k.Proof.Value, 64)
		out[i] = replayABIKey{Name: k.Name, Alg: k.Alg, Key: raw, Signer: k.Proof.Signer, Proof: proof, HasExpiry: k.Expires != nil}
		if k.Expires != nil {
			out[i].Expires = *k.Expires
		}
	}
	return out
}
func replayABIServices(services []replayService) []replayABIService {
	out := make([]replayABIService, len(services))
	for i, s := range services {
		out[i] = replayABIService{s.Name, s.Type, s.URI}
	}
	return out
}

// Decode then re-pack without repairing offsets, padding or trailing bytes.
// Tuple conversion is data-only JSON between two pinned ABI field sets.
func replayDecodeCommand(registry, actor string, data []byte, contract abi.ABI) (replayCommand, bool) {
	var empty replayCommand
	if len(data) < 4 {
		return empty, false
	}
	method, e := contract.MethodById(data[:4])
	if e != nil {
		return empty, false
	}
	if !replayCallFraming(*method, data) {
		return empty, false
	}
	values, e := method.Inputs.Unpack(data[4:])
	if e != nil {
		return empty, false
	}
	packed, e := method.Inputs.Pack(values...)
	if e != nil || !bytes.Equal(data[4:], packed) {
		return empty, false
	}
	op := ""
	for operation, name := range replayMethods {
		if name == method.Name {
			op = operation
		}
	}
	if op == "" {
		return empty, false
	}
	agent := values[0].(string)
	if !rpcAgent(agent) {
		return empty, false
	}
	expected := ""
	args := map[string]any{}
	if op == "create" {
		var keys []replayABIKey
		var services []replayABIService
		if !replayConvert(values[2], &keys) || !replayConvert(values[3], &services) {
			return empty, false
		}
		ks := make([]replayKey, len(keys))
		for i, k := range keys {
			if !k.HasExpiry && k.Expires != 0 {
				return empty, false
			}
			ks[i] = replayKey{Name: k.Name, Alg: k.Alg, Key: base64.RawURLEncoding.EncodeToString(k.Key), Proof: replayProof{k.Signer, base64.RawURLEncoding.EncodeToString(k.Proof)}, State: "accepted"}
			if k.HasExpiry {
				n := k.Expires
				ks[i].Expires = &n
			}
		}
		ss := make([]replayService, len(services))
		for i, s := range services {
			ss[i] = replayService{s.Name, s.ServiceType, s.URI}
		}
		salt := values[4].([32]byte)
		args = map[string]any{"controller": strings.ToLower(values[1].(common.Address).Hex()), "keys": ks, "services": ss, "salt": base64.RawURLEncoding.EncodeToString(salt[:])}
	} else {
		expected = strconv.FormatUint(values[1].(uint64), 10)
		switch op {
		case "add-key":
			var k replayABIKey
			if !replayConvert(values[2], &k) || (!k.HasExpiry && k.Expires != 0) {
				return empty, false
			}
			entry := replayKey{Name: k.Name, Alg: k.Alg, Key: base64.RawURLEncoding.EncodeToString(k.Key), Proof: replayProof{k.Signer, base64.RawURLEncoding.EncodeToString(k.Proof)}, State: "accepted"}
			if k.HasExpiry {
				n := k.Expires
				entry.Expires = &n
			}
			args["key"] = entry
		case "revoke-key":
			args["name"] = values[2].(string)
		case "update-services":
			var services []replayABIService
			if !replayConvert(values[2], &services) {
				return empty, false
			}
			ss := make([]replayService, len(services))
			for i, s := range services {
				ss[i] = replayService{s.Name, s.ServiceType, s.URI}
			}
			args["services"] = ss
		case "authorize-operator", "revoke-operator":
			args["operator"] = strings.ToLower(values[2].(common.Address).Hex())
			args["scope"] = values[3].(string)
		}
	}
	raw, e := rfc8785.Marshal(map[string]any{"binding": "sage-registry-chain-0.11.0", "registry": registry, "did": "did:sage:" + registry + ":" + agent, "actor": actor, "operation": op, "expectedVersion": expected, "arguments": args})
	if e != nil {
		return empty, false
	}
	c, ok := replayParseCommand(registry, agent, raw, contract)
	return c, ok && bytes.Equal(c.calldata, data)
}
func replayConvert(v, into any) bool {
	raw, e := json.Marshal(v)
	return e == nil && json.Unmarshal(raw, into) == nil
}

func replayClaim(registry string, c replayCommand) [32]byte {
	raw, _ := rfc8785.Marshal(map[string]any{"registryId": registry, "agentId": c.agent, "controller": c.args["controller"], "keys": c.args["keys"], "services": c.args["services"], "salt": c.args["salt"]})
	return sha256.Sum256(append([]byte("sage-claim-0.11.0\x00"), raw...))
}
