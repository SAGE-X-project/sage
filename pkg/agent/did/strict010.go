package did

import (
	"errors"
	"net"
	"strings"
)

var (
	ErrMalformedDID010   = errors.New("id.malformed")
	ErrUnknownDIDKind010 = errors.New("id.unknown-kind")
)

// DID010 is a parsed, already canonical 0.10.0 registry identity.
// It does not resolve the record or establish registry authority.
type DID010 struct {
	Kind    string
	Locator string
	AgentID string
}

// DIDURL010 names one key in a canonical 0.10.0 DID.
type DIDURL010 struct {
	DID   DID010
	KeyID string
}

// ParseDID010 applies the 0.10.0 grammar without changing legacy ParseDID.
func ParseDID010(raw string) (DID010, error) {
	if len(raw) > 256 || !strings.HasPrefix(raw, "did:sage:") {
		return DID010{}, ErrMalformedDID010
	}
	parts := strings.Split(strings.TrimPrefix(raw, "did:sage:"), ":")
	if len(parts) < 3 || !validKind010(parts[0]) {
		return DID010{}, ErrMalformedDID010
	}
	var locator, agent string
	switch parts[0] {
	case "eip155":
		if len(parts) != 4 || !validChain010(parts[1]) || !validAddress010(parts[2]) {
			return DID010{}, ErrMalformedDID010
		}
		locator, agent = parts[1]+":"+parts[2], parts[3]
	case "web":
		if len(parts) != 3 || !validDomain010(parts[1]) {
			return DID010{}, ErrMalformedDID010
		}
		locator, agent = parts[1], parts[2]
	default:
		return DID010{}, ErrUnknownDIDKind010
	}
	if !validAgent010(agent) {
		return DID010{}, ErrMalformedDID010
	}
	return DID010{Kind: parts[0], Locator: locator, AgentID: agent}, nil
}

// ParseDIDURL010 requires exactly one key fragment and a canonical DID.
func ParseDIDURL010(raw string) (DIDURL010, error) {
	if len(raw) > 289 {
		return DIDURL010{}, ErrMalformedDID010
	}
	did, key, found := strings.Cut(raw, "#")
	if !found || !validKeyID010(key) {
		return DIDURL010{}, ErrMalformedDID010
	}
	parsed, err := ParseDID010(did)
	if err != nil {
		return DIDURL010{}, err
	}
	return DIDURL010{DID: parsed, KeyID: key}, nil
}

func validKind010(value string) bool {
	if len(value) == 0 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func validAgent010(value string) bool {
	if len(value) == 0 || len(value) > 64 || value == "." || value == ".." {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func validChain010(value string) bool {
	if len(value) == 0 || len(value) > 32 || value[0] < '1' || value[0] > '9' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func validAddress010(value string) bool {
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return false
	}
	for i := 2; i < len(value); i++ {
		c := value[i]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

func validDomain010(value string) bool {
	if len(value) == 0 || len(value) > 64 || net.ParseIP(value) != nil {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || !asciiAlnum010(label[0]) ||
			!asciiAlnum010(label[len(label)-1]) {
			return false
		}
		for i := 1; i+1 < len(label); i++ {
			if !asciiAlnum010(label[i]) && label[i] != '-' {
				return false
			}
		}
	}
	return true
}

func asciiAlnum010(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func validKeyID010(value string) bool {
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}
