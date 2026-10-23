package registry011

import (
	"encoding/base64"
	"net/netip"
	"sort"
	"strings"
)

type replayProof struct {
	Signer string `json:"signer"`
	Value  string `json:"value"`
}
type replayKey struct {
	Name    string      `json:"name"`
	Alg     string      `json:"alg"`
	Key     string      `json:"key"`
	Proof   replayProof `json:"proof"`
	State   string      `json:"state"`
	Expires *uint64     `json:"expires,omitempty"`
}
type replayService struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URI  string `json:"uri"`
}
type replayGrant struct {
	Operator string `json:"operator"`
	Scope    string `json:"scope"`
}
type replayRecord struct {
	ID         string          `json:"id"`
	Controller string          `json:"controller"`
	Keys       []replayKey     `json:"keys"`
	Services   []replayService `json:"services"`
	State      string          `json:"state"`
	Version    string          `json:"version"`
}

func replayName(s string) bool {
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	for i := range s {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func replayBinary(v any, size int) ([]byte, bool) {
	s, ok := v.(string)
	if !ok || len(s) != (size*8+5)/6 {
		return nil, false
	}
	b, e := base64.RawURLEncoding.Strict().DecodeString(s)
	return b, e == nil && len(b) == size && base64.RawURLEncoding.EncodeToString(b) == s
}
func replayKeys(v any, did string) ([]replayKey, bool) {
	a, ok := v.([]any)
	if !ok || len(a) == 0 || len(a) > 128 {
		return nil, false
	}
	out := make([]replayKey, 0, len(a))
	materials := map[string]bool{}
	previous := ""
	for _, value := range a {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		names := []string{"name", "alg", "key", "proof", "state"}
		if _, exists := m["expires"]; exists {
			names = append(names, "expires")
		}
		if contentObject(m, names...) == nil {
			return nil, false
		}
		k := replayKey{Name: contentString(m["name"]), Alg: contentString(m["alg"]), Key: contentString(m["key"]), State: contentString(m["state"])}
		// Exactly the deployed required-suite capability, before binary decoding.
		if (k.Alg != "ed25519" && k.Alg != "x25519") || !replayName(k.Name) || k.Name <= previous || k.State != "accepted" {
			return nil, false
		}
		material, ok := replayBinary(m["key"], 32)
		if !ok || materials[string(material)] {
			return nil, false
		}
		materials[string(material)] = true
		p := contentObject(m["proof"], "signer", "value")
		if p == nil {
			return nil, false
		}
		k.Proof = replayProof{contentString(p["signer"]), contentString(p["value"])}
		if _, ok := replayBinary(p["value"], 64); !ok {
			return nil, false
		}
		if !strings.HasPrefix(k.Proof.Signer, did+"#") || !replayName(strings.TrimPrefix(k.Proof.Signer, did+"#")) {
			return nil, false
		}
		if k.Alg == "ed25519" && k.Proof.Signer != did+"#"+k.Name {
			return nil, false
		}
		if x, present := m["expires"]; present {
			n, ok := contentSeconds(x)
			if !ok {
				return nil, false
			}
			k.Expires = &n
		}
		out = append(out, k)
		previous = k.Name
	}
	return out, true
}
func replayServices(v any, keys []replayKey) ([]replayService, bool) {
	a, ok := v.([]any)
	if !ok || len(a) > 16 {
		return nil, false
	}
	out := make([]replayService, 0, len(a))
	prev := ""
	for _, v := range a {
		m := contentObject(v, "name", "type", "uri")
		if m == nil {
			return nil, false
		}
		s := replayService{contentString(m["name"]), contentString(m["type"]), contentString(m["uri"])}
		if !replayName(s.Name) || s.Name <= prev || len(s.Type) < 1 || len(s.Type) > 64 || !contentASCII(s.Type) || !replayURI(s.URI) {
			return nil, false
		}
		for _, k := range keys {
			if k.Name == s.Name {
				return nil, false
			}
		}
		out = append(out, s)
		prev = s.Name
	}
	return out, true
}
func replayURI(s string) bool {
	if len(s) < 9 || len(s) > 2048 || !strings.EqualFold(s[:8], "https://") {
		return false
	}
	end := strings.IndexAny(s[8:], "/?")
	if end < 0 {
		end = len(s) - 8
	}
	end += 8
	a := s[8:end]
	if a == "" {
		return false
	}
	port := ""
	if a[0] == '[' {
		x := strings.IndexByte(a, ']')
		if x < 0 || !replayIP(a[1:x]) {
			return false
		}
		if x+1 < len(a) {
			if a[x+1] != ':' {
				return false
			}
			port = a[x+2:]
		}
	} else {
		host := a
		if i := strings.IndexByte(a, ':'); i >= 0 {
			host, port = a[:i], a[i+1:]
		}
		if host == "" || !replayURIChars(host, false) {
			return false
		}
	}
	for i := range port {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	return replayURIChars(s[end:], true)
}
func replayIP(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] == 'v' || s[0] == 'V' {
		i := strings.IndexByte(s, '.')
		if i < 2 || i == len(s)-1 {
			return false
		}
		for _, c := range []byte(s[1:i]) {
			if !replayHex(c) {
				return false
			}
		}
		for _, c := range []byte(s[i+1:]) {
			if !replayUnreserved(c) && !strings.ContainsRune("!$&'()*+,;=:", rune(c)) {
				return false
			}
		}
		return true
	}
	a, e := netip.ParseAddr(s)
	return e == nil && a.Is6() && a.Zone() == ""
}
func replayURIChars(s string, path bool) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if replayUnreserved(c) || strings.ContainsRune("!$&'()*+,;=", rune(c)) || (path && strings.ContainsRune("?:/@", rune(c))) {
			continue
		}
		if c == '%' && len(s)-i >= 3 && replayHex(s[i+1]) && replayHex(s[i+2]) {
			i += 2
			continue
		}
		return false
	}
	return true
}
func replayHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
func replayUnreserved(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~", rune(c))
}
func replayUsable(k replayKey, now uint64) bool {
	return k.State == "accepted" && k.Alg == "ed25519" && (k.Expires == nil || now < *k.Expires)
}
func replayHasSigner(keys []replayKey, now uint64) bool {
	for _, k := range keys {
		if replayUsable(k, now) {
			return true
		}
	}
	return false
}
func replaySortGrants(g []replayGrant) {
	sort.Slice(g, func(i, j int) bool {
		return g[i].Operator < g[j].Operator || (g[i].Operator == g[j].Operator && g[i].Scope < g[j].Scope)
	})
}
