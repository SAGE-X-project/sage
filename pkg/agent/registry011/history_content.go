package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

var ErrHistoryContent = errors.New("registry011: inconsistent history content")

// HistoryContentStep includes the complete intermediate state independently
// reconstructed by the caller, not returned by historyItem. These bytes are
// untrusted input here. This check does not reconstruct or authenticate them.
type HistoryContentStep struct {
	History        HistoryBytes
	Record, Grants []byte
}

// CheckHistoryContent checks bounded canonical bytes, closed ROOT member sets,
// the complete 1..current version sequence and each state's SHA-256 bindings.
// Command argument member sets, target, removal ordering, admission order and
// immutable controller are checked. Nested key/service/proof schemas, crypto,
// lifecycle/ACL effects, claim provenance and actual execution MUST be checked
// by the separate full replay implementation; they are not established here.
//
// Every intermediate Record/Grants must come from that independent replay.
// Internally consistent invented states can pass this content-only check.
// nil is never a Source-ready credential, authenticated absence or authority.
// The final bytes must be the separately block-pinned record and grants. Input
// buffers must not be concurrently mutated. No input bytes are retained.
func CheckHistoryContent(registry, agent string, record, grants []byte, steps []HistoryContentStep, limits HistoryReadLimits) error {
	if registry != "eip155:11155111:"+SepoliaRegistryAddress || !rpcAgent(agent) ||
		limits.MaxItems == 0 || limits.MaxItems > 4096 || limits.MaxBytes <= 0 || limits.MaxBytes > 8<<20 {
		return ErrConfiguration
	}
	if len(steps) == 0 || uint64(len(steps)) > limits.MaxItems {
		return ErrHistoryContent
	}
	// Preflight the complete allocation budget before parsing any JSON. Unlike
	// HistoryRange, this also counts all intermediate states and final bytes.
	remaining := limits.MaxBytes
	charge := func(raw []byte, maximum int) bool {
		if len(raw) == 0 || len(raw) > maximum || len(raw) > remaining {
			return false
		}
		remaining -= len(raw)
		return true
	}
	if !charge(record, 65536) || !charge(grants, 262144) {
		return ErrHistoryContent
	}
	for _, s := range steps {
		if !charge(s.History.Entry, 262144) || !charge(s.History.Command, 262144) ||
			!charge(s.Record, 65536) || !charge(s.Grants, 262144) {
			return ErrHistoryContent
		}
	}
	did := "did:sage:" + registry + ":" + agent
	var controller string
	var admitted uint64
	var terminal bool
	for i, step := range steps {
		version := uint64(i) + 1
		predecessor := ""
		if i > 0 {
			predecessor = strconv.FormatUint(version-1, 10)
		}
		h := contentObject(contentJSON(step.History.Entry), "version", "previousVersion", "actor", "operation", "commandDigest", "target", "automaticRemovals", "admitted", "recordDigest", "grantsDigest")
		c := contentObject(contentJSON(step.History.Command), "binding", "registry", "did", "operation", "expectedVersion", "arguments", "actor")
		r := contentObject(contentJSON(step.Record), "id", "controller", "keys", "services", "state", "version")
		previous, previousOK := h["previousVersion"].(string)
		expected, expectedOK := c["expectedVersion"].(string)
		if h == nil || c == nil || r == nil || !previousOK || !expectedOK || terminal || step.History.Version != version ||
			contentString(h["version"]) != strconv.FormatUint(version, 10) ||
			contentString(r["version"]) != strconv.FormatUint(version, 10) ||
			previous != predecessor || expected != predecessor ||
			contentString(c["binding"]) != "sage-registry-chain-0.11.0" ||
			contentString(c["registry"]) != registry || contentString(c["did"]) != did || contentString(r["id"]) != did {
			return ErrHistoryContent
		}
		actor := contentString(c["actor"])
		owner := contentString(r["controller"])
		op := contentString(c["operation"])
		stamp, ok := contentSeconds(h["admitted"])
		if !contentAddress(actor) || !contentAddress(owner) || contentString(h["actor"]) != actor ||
			contentString(h["operation"]) != op || !ok || stamp < admitted ||
			(i > 0 && owner != controller) || (op == "create") != (i == 0) {
			return ErrHistoryContent
		}
		args, valid := contentArguments(op, c["arguments"])
		if !valid || !contentTarget(h["target"], op, args) || !contentGrants(h["automaticRemovals"]) ||
			!contentGrants(contentJSON(step.Grants)) || !contentDigest(h["commandDigest"], step.History.Command) ||
			!contentDigest(h["recordDigest"], step.Record) || !contentDigest(h["grantsDigest"], step.Grants) {
			return ErrHistoryContent
		}
		state := contentString(r["state"])
		keys, keysOK := r["keys"].([]any)
		services, servicesOK := r["services"].([]any)
		if !keysOK || len(keys) < 1 || len(keys) > 128 || !servicesOK || len(services) > 16 ||
			(state != "created" && state != "active" && state != "deactivated") ||
			(i == 0 && (state != "created" || contentString(args["controller"]) != actor || owner != actor)) {
			return ErrHistoryContent
		}
		controller, admitted, terminal = owner, stamp, state == "deactivated"
	}
	last := steps[len(steps)-1]
	if !bytes.Equal(record, last.Record) || !bytes.Equal(grants, last.Grants) {
		return ErrHistoryContent
	}
	return nil
}

// contentJSON rejects duplicate names and ambiguous Unicode before map parsing.
// All admitted chain-schema text is ASCII, including escaped ASCII controls.
// Requiring exact canonical bytes avoids repairing a signed/digested object.
func contentJSON(raw []byte) any {
	if !utf8.Valid(raw) || !validRPCUnicode(raw) {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !readRPCValue(d, 0, 32) {
		return nil
	}
	if _, err := d.Token(); err != io.EOF {
		return nil
	}
	canonical, err := rfc8785.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return nil
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil || !contentASCII(v) {
		return nil
	}
	return v
}

func contentASCII(v any) bool {
	switch x := v.(type) {
	case string:
		for i := range x {
			if x[i] > 127 {
				return false
			}
		}
	case []any:
		for _, child := range x {
			if !contentASCII(child) {
				return false
			}
		}
	case map[string]any:
		for name, child := range x {
			if !contentASCII(name) || !contentASCII(child) {
				return false
			}
		}
	}
	return true
}

func contentObject(v any, names ...string) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || len(m) != len(names) {
		return nil
	}
	for _, name := range names {
		if _, ok := m[name]; !ok {
			return nil
		}
	}
	return m
}

func contentString(v any) string {
	s, _ := v.(string)
	return s
}

func contentSeconds(v any) (uint64, bool) {
	n, ok := v.(json.Number)
	u, err := strconv.ParseUint(string(n), 10, 64)
	return u, ok && err == nil && u <= 1<<53-1 && strconv.FormatUint(u, 10) == string(n)
}

func contentDigest(v any, raw []byte) bool {
	sum := sha256.Sum256(raw)
	return contentString(v) == base64.RawURLEncoding.EncodeToString(sum[:])
}

func contentArguments(op string, v any) (map[string]any, bool) {
	var names []string
	switch op {
	case "create":
		names = []string{"controller", "keys", "services", "salt"}
	case "activate", "deactivate":
	case "add-key":
		names = []string{"key"}
	case "revoke-key":
		names = []string{"name"}
	case "update-services":
		names = []string{"services"}
	case "authorize-operator", "revoke-operator":
		names = []string{"operator", "scope"}
	default:
		return nil, false
	}
	m := contentObject(v, names...)
	return m, m != nil
}

func contentTarget(v any, op string, args map[string]any) bool {
	if op != "authorize-operator" && op != "revoke-operator" {
		return v == nil
	}
	pair := contentObject(v, "operator", "scope")
	return pair != nil && contentGrant(pair) && pair["operator"] == args["operator"] && pair["scope"] == args["scope"]
}

func contentGrant(pair map[string]any) bool {
	if !contentAddress(contentString(pair["operator"])) {
		return false
	}
	switch contentString(pair["scope"]) {
	case "activate", "add-key", "revoke-key", "update-services", "deactivate":
		return true
	}
	return false
}

// Principal grammar has no additional zero-address reservation. The separate
// configured RPC/Registry address checks deliberately retain their own policy.
func contentAddress(s string) bool {
	return len(s) == 42 && s[:2] == "0x" && lowerHash("0x000000000000000000000000"+s[2:])
}

func contentGrants(v any) bool {
	pairs, ok := v.([]any)
	if !ok || len(pairs) > 128 {
		return false
	}
	previous := ""
	for _, v := range pairs {
		pair := contentObject(v, "operator", "scope")
		if pair == nil || !contentGrant(pair) {
			return false
		}
		order := contentString(pair["operator"]) + ":" + contentString(pair["scope"])
		if order <= previous {
			return false
		}
		previous = order
	}
	return true
}
