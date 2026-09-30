package registry010

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const maxWebRecord010 = 65536

// CheckWebRegistryRecordShape010 checks the exact encoded record inside a
// bounded, fresh REG-08 envelope against the web record's structural rules.
// It does not verify key points, proofs, historical immutability, HTTPS origin,
// or controller authority. Success is never an authorization decision.
func CheckWebRegistryRecordShape010(raw []byte, expectedDID string, now int64) error {
	if err := CheckWebRegistryEnvelope010(raw, now); err != nil {
		return err
	}
	if !validWebDID010(expectedDID) {
		return ErrInvalidRecord010
	}
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(raw, &wrapper) != nil {
		return ErrInvalidRecord010
	}
	recordRaw := wrapper["record"]
	if len(recordRaw) > maxWebRecord010 {
		return ErrSizeExceeded010
	}
	decoder := json.NewDecoder(bytes.NewReader(recordRaw))
	decoder.UseNumber()
	value, err := readWebValue010(decoder, 1)
	if err != nil {
		return ErrInvalidRecord010
	}
	record, ok := value.(map[string]any)
	if !ok || !webFields010(record, "id", "controller", "keys", "services", "state", "version") {
		return ErrInvalidRecord010
	}
	id, ok := record["id"].(string)
	if !ok || id != expectedDID {
		return ErrInvalidRecord010
	}
	controller, ok := record["controller"].(string)
	if !ok || !webASCII010(controller, 1, 256) {
		return ErrInvalidRecord010
	}
	version, ok := record["version"].(string)
	if !ok || len(version) == 0 || version[0] == '0' || !webDigits010(version) {
		return ErrInvalidRecord010
	}
	if n, err := strconv.ParseUint(version, 10, 64); err != nil || n == 0 {
		return ErrInvalidRecord010
	}
	state, ok := record["state"].(string)
	if !ok || (state != "created" && state != "active" && state != "deactivated") {
		return ErrInvalidRecord010
	}
	keys, ok := record["keys"].([]any)
	if !ok || len(keys) < 1 || len(keys) > 128 {
		return ErrInvalidRecord010
	}
	keyNames := make(map[string]bool, len(keys))
	keyBytes := make(map[string]bool, len(keys))
	previous := ""
	activeSigning := false
	for _, entry := range keys {
		key, ok := entry.(map[string]any)
		if !ok || !webFieldsOptional010(key, []string{"name", "alg", "key", "proof", "state"}, "expires") {
			return ErrInvalidRecord010
		}
		name, ok := key["name"].(string)
		if !ok || !webKeyID010(name) || (previous != "" && name <= previous) {
			return ErrInvalidRecord010
		}
		previous = name
		keyNames[name] = true
		alg, ok := key["alg"].(string)
		if !ok {
			return ErrInvalidRecord010
		}
		want := 32
		switch alg {
		case "ed25519", "x25519":
		case "sage-secp256k1-keccak256", "ecdsa-p256-sha256":
			want = 65
		default:
			return ErrInvalidRecord010
		}
		encoded, ok := key["key"].(string)
		if !ok {
			return ErrInvalidRecord010
		}
		material, ok := webBase64URL010(encoded, want)
		if !ok || (want == 65 && material[0] != 4) || keyBytes[encoded] {
			return ErrInvalidRecord010
		}
		keyBytes[encoded] = true
		keyState, ok := key["state"].(string)
		if !ok || (keyState != "accepted" && keyState != "revoked") {
			return ErrInvalidRecord010
		}
		expires := int64(0)
		hasExpiry := false
		if rawExpiry, present := key["expires"]; present {
			hasExpiry = true
			var valid bool
			expires, valid = webInteger010(rawExpiry)
			if !valid || expires < 0 {
				return ErrInvalidRecord010
			}
		}
		proof, ok := key["proof"].(map[string]any)
		if !ok || !webFields010(proof, "signer", "value") {
			return ErrInvalidRecord010
		}
		signer, ok := proof["signer"].(string)
		if !ok || !strings.HasPrefix(signer, id+"#") || !webKeyID010(strings.TrimPrefix(signer, id+"#")) ||
			(alg != "x25519" && signer != id+"#"+name) || (alg == "x25519" && signer == id+"#"+name) {
			return ErrInvalidRecord010
		}
		signature, ok := proof["value"].(string)
		if !ok || len(signature) == 0 || len(signature) > 87 || !webBase64URLAny010(signature) {
			return ErrInvalidRecord010
		}
		if alg != "x25519" && keyState == "accepted" && (!hasExpiry || now < expires) {
			activeSigning = true
		}
	}
	if state == "active" && !activeSigning {
		return ErrInvalidRecord010
	}
	services, ok := record["services"].([]any)
	if !ok || len(services) > 16 {
		return ErrInvalidRecord010
	}
	previous = ""
	for _, entry := range services {
		service, ok := entry.(map[string]any)
		if !ok || !webFields010(service, "name", "type", "uri") {
			return ErrInvalidRecord010
		}
		name, ok := service["name"].(string)
		if !ok || !webKeyID010(name) || keyNames[name] || (previous != "" && name <= previous) {
			return ErrInvalidRecord010
		}
		previous = name
		typeName, ok := service["type"].(string)
		if !ok || !webASCII010(typeName, 1, 64) {
			return ErrInvalidRecord010
		}
		uri, ok := service["uri"].(string)
		if !ok || !webServiceURI010(uri) {
			return ErrInvalidRecord010
		}
	}
	return nil
}

func webFields010(object map[string]any, fields ...string) bool {
	if len(object) != len(fields) {
		return false
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return false
		}
	}
	return true
}

func webFieldsOptional010(object map[string]any, required []string, optional string) bool {
	if len(object) != len(required) && len(object) != len(required)+1 {
		return false
	}
	for _, field := range required {
		if _, ok := object[field]; !ok {
			return false
		}
	}
	if len(object) == len(required)+1 {
		_, ok := object[optional]
		return ok
	}
	return true
}

func webASCII010(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] > 127 {
			return false
		}
	}
	return true
}

func webDigits010(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func webKeyID010(value string) bool {
	if len(value) < 1 || len(value) > 32 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func webBase64URL010(encoded string, want int) ([]byte, bool) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	return decoded, err == nil && len(decoded) == want && base64.RawURLEncoding.EncodeToString(decoded) == encoded
}

func webBase64URLAny010(encoded string) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	return err == nil && len(decoded) > 0 && base64.RawURLEncoding.EncodeToString(decoded) == encoded
}

func validWebDID010(did string) bool {
	if len(did) > 256 || !strings.HasPrefix(did, "did:sage:web:") {
		return false
	}
	rest := strings.TrimPrefix(did, "did:sage:web:")
	domain, agent, ok := strings.Cut(rest, ":")
	if !ok || strings.Contains(agent, ":") || len(domain) < 1 || len(domain) > 64 || len(agent) < 1 || len(agent) > 64 || agent == "." || agent == ".." {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	if net.ParseIP(domain) != nil {
		return false
	}
	for i := 0; i < len(agent); i++ {
		c := agent[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func webServiceURI010(value string) bool {
	if !webASCII010(value, 1, 2048) {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] <= 0x20 || value[i] == 0x7f || value[i] == '#' || value[i] == '\\' {
			return false
		}
		if value[i] == '%' {
			if i+2 >= len(value) || !webHex010(value[i+1]) || !webHex010(value[i+2]) {
				return false
			}
			i += 2
		}
	}
	parsed, err := url.Parse(value)
	return err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Host != "" && parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == ""
}

func webHex010(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
