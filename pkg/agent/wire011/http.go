package wire011

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Chapter 03 MSG-02 and MSG-03 fixed coverage.
const (
	requestComponents  = `"@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "x-sage-version"`
	responseComponents = `"@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "x-sage-version";req "content-type" "content-digest" "x-sage-did" "x-sage-version"`

	maxBody        = 16 << 20
	maxFieldBytes  = 32 << 10
	maxFields      = 256
	maxSignatureFV = 8 << 10
)

// Field is one HTTP field line exactly as received; repeated fields stay separate.
type Field struct{ Name, Value string }

// HTTPRequest carries the values the trusted HTTP stack observed.
type HTTPRequest struct {
	Method    string
	TargetURI string // absolute target URI as received, not derived from Host or Forwarded
	Authority string
	Fields    []Field
	Body      []byte
}

// HTTPResponse carries a response as received or as produced.
type HTTPResponse struct {
	Status int
	Fields []Field
	Body   []byte
}

// httpContext is the request state a response signature covers with ;req.
type httpContext struct {
	method, target, authority, digest, signature, version string
}

type httpParams struct {
	keyid, alg, nonce, tag string
	created, expires       int64
	input                  string // serialized inner list with parameters, as covered by @signature-params
	signature              []byte
}

func contentDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha-256=:" + base64.StdEncoding.EncodeToString(sum[:]) + ":"
}

// checkFields applies MSG-04 bounds and duplicate rules and returns the
// critical fields by lowercase name.
func checkFields(fields []Field, body []byte) (map[string]string, error) {
	if len(body) > maxBody || len(fields) > maxFields {
		return nil, fmt.Errorf("size")
	}
	h := map[string]string{}
	total := 0
	for _, f := range fields {
		name := strings.ToLower(f.Name)
		total += len(f.Name) + len(f.Value) + 4
		if name == "" || !token(name) || !ascii(f.Value) || strings.TrimSpace(f.Value) != f.Value || total > maxFieldBytes {
			return nil, fmt.Errorf("field syntax or size")
		}
		switch {
		case name == "content-encoding", name == "transfer-encoding", name == "trailer", strings.HasPrefix(name, "x-sage-meta-"):
			return nil, fmt.Errorf("field %s not admitted", name)
		case (name == "signature" || name == "signature-input") && len(f.Value) > maxSignatureFV:
			return nil, fmt.Errorf("signature field size")
		}
		critical := name == "signature" || name == "signature-input" || name == "content-digest" ||
			name == "content-type" || name == "content-length" || name == "host" || strings.HasPrefix(name, "x-sage-")
		if _, dup := h[name]; dup && critical {
			return nil, fmt.Errorf("duplicate %s", name)
		}
		h[name] = f.Value
	}
	switch {
	case h["content-type"] != "application/json":
		return nil, fmt.Errorf("content-type")
	case h["x-sage-version"] != Version:
		return nil, fmt.Errorf("x-sage-version")
	case h["x-sage-did"] == "":
		return nil, fmt.Errorf("x-sage-did")
	case h["content-digest"] != contentDigest(body):
		return nil, fmt.Errorf("content-digest")
	}
	if v, ok := h["content-length"]; ok && v != strconv.Itoa(len(body)) {
		return nil, fmt.Errorf("content-length")
	}
	return h, nil
}

// parseSignature parses exactly one sig1 Signature-Input and Signature with
// the fixed coverage. It accepts the RFC 8941 spacing variants the 0.10.0
// parser accepts and serializes them canonically for the signature base;
// escaped strings and alternate number spellings are refused.
func parseSignature(h map[string]string, response bool) (*httpParams, error) {
	components := requestComponents
	if response {
		components = responseComponents
	}
	input, err := canonicalInput(h["signature-input"], components)
	if err != nil {
		return nil, err
	}
	p, err := inputParams(input, components)
	if err != nil {
		return nil, err
	}
	v := h["signature"]
	if !strings.HasPrefix(v, "sig1=:") || !strings.HasSuffix(v, ":") {
		return nil, fmt.Errorf("signature member")
	}
	sig, err := base64.StdEncoding.Strict().DecodeString(v[6 : len(v)-1])
	if err != nil || len(sig) != 64 || "sig1=:"+base64.StdEncoding.EncodeToString(sig)+":" != v {
		return nil, fmt.Errorf("signature encoding")
	}
	p.input = input[len("sig1="):]
	p.signature = sig
	return p, nil
}

func canonicalInput(v, components string) (string, error) {
	if len(v) > maxSignatureFV || !ascii(v) {
		return "", fmt.Errorf("signature-input")
	}
	rest, ok := strings.CutPrefix(v, "sig1=(")
	if !ok {
		return "", fmt.Errorf("signature label")
	}
	rest = strings.TrimLeft(rest, " ")
	for i, item := range strings.Split(components, " ") {
		if i > 0 {
			if !strings.HasPrefix(rest, " ") {
				return "", fmt.Errorf("coverage")
			}
			rest = strings.TrimLeft(rest, " ")
		}
		name := strings.TrimSuffix(item, ";req")
		if rest, ok = strings.CutPrefix(rest, name); !ok {
			return "", fmt.Errorf("coverage")
		}
		if strings.HasSuffix(item, ";req") {
			if rest, ok = strings.CutPrefix(rest, ";"); !ok {
				return "", fmt.Errorf("coverage req")
			}
			rest = strings.TrimLeft(rest, " ")
			if rest, ok = strings.CutPrefix(rest, "req"); !ok {
				return "", fmt.Errorf("coverage req")
			}
			rest = strings.TrimPrefix(rest, "=?1")
		}
	}
	rest = strings.TrimLeft(rest, " ")
	if rest, ok = strings.CutPrefix(rest, ")"); !ok {
		return "", fmt.Errorf("coverage end")
	}
	var out strings.Builder
	out.WriteString("sig1=(" + components + ")")
	quoted := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if c == ',' && !quoted {
			return "", fmt.Errorf("second signature")
		}
		if c == '"' {
			quoted = !quoted
		}
		out.WriteByte(c)
		if c == ';' && !quoted {
			for i+1 < len(rest) && rest[i+1] == ' ' {
				i++
			}
		}
	}
	return out.String(), nil
}

func inputParams(input, components string) (*httpParams, error) {
	rest := input[len("sig1=("+components+")"):]
	seen := map[string]string{}
	for rest != "" {
		if rest[0] != ';' {
			return nil, fmt.Errorf("parameter syntax")
		}
		rest = rest[1:]
		n := strings.IndexByte(rest, '=')
		if n < 1 {
			return nil, fmt.Errorf("parameter syntax")
		}
		key := rest[:n]
		rest = rest[n+1:]
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate parameter")
		}
		var value string
		switch key {
		case "keyid", "alg", "nonce", "tag":
			if len(rest) < 2 || rest[0] != '"' {
				return nil, fmt.Errorf("parameter %s", key)
			}
			end := strings.IndexByte(rest[1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("parameter %s", key)
			}
			value = rest[1 : end+1]
			if strings.ContainsRune(value, '\\') {
				return nil, fmt.Errorf("escaped parameter")
			}
			rest = rest[end+2:]
		case "created", "expires":
			end := strings.IndexByte(rest, ';')
			if end < 0 {
				end = len(rest)
			}
			value, rest = rest[:end], rest[end:]
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil || i < 0 || len(value) > 15 || strconv.FormatInt(i, 10) != value {
				return nil, fmt.Errorf("parameter %s", key)
			}
		default:
			return nil, fmt.Errorf("unknown parameter %s", key)
		}
		seen[key] = value
	}
	if len(seen) != 6 {
		return nil, fmt.Errorf("missing parameter")
	}
	if seen["alg"] != "ed25519" {
		return nil, fmt.Errorf("alg %s", seen["alg"])
	}
	if seen["tag"] != Tag {
		return nil, fmt.Errorf("tag")
	}
	created, _ := strconv.ParseInt(seen["created"], 10, 64)
	expires, _ := strconv.ParseInt(seen["expires"], 10, 64)
	return &httpParams{keyid: seen["keyid"], alg: seen["alg"], nonce: seen["nonce"], tag: seen["tag"],
		created: created, expires: expires}, nil
}

func signatureInput(components, kid string, created, expires int64, nonce string) string {
	return "(" + components + ");keyid=" + strconv.Quote(kid) + `;alg="ed25519";created=` + strconv.FormatInt(created, 10) +
		";expires=" + strconv.FormatInt(expires, 10) + ";nonce=" + strconv.Quote(nonce) + `;tag="` + Tag + `"`
}

func requestBase(method, target, authority string, h map[string]string, input string) []byte {
	lines := []string{
		`"@method": ` + method,
		`"@target-uri": ` + target,
		`"@authority": ` + authority,
	}
	return finishBase(lines, h, input)
}

func responseBase(status int, q httpContext, h map[string]string, input string) []byte {
	lines := []string{
		`"@status": ` + strconv.Itoa(status),
		`"@method";req: ` + q.method,
		`"@target-uri";req: ` + q.target,
		`"@authority";req: ` + q.authority,
		`"content-digest";req: ` + q.digest,
		`"signature";req: ` + q.signature,
		`"x-sage-version";req: ` + q.version,
	}
	return finishBase(lines, h, input)
}

func finishBase(lines []string, h map[string]string, input string) []byte {
	for _, k := range []string{"content-type", "content-digest", "x-sage-did", "x-sage-version"} {
		lines = append(lines, `"`+k+`": `+h[k])
	}
	lines = append(lines, `"@signature-params": `+input)
	return []byte(strings.Join(lines, "\n"))
}

// endpoint returns the canonical https endpoint and its authority, refusing
// anything that would need URI normalization.
func endpoint(target string) (string, string, error) {
	u, err := url.Parse(target)
	bad := err != nil || !ascii(target) || strings.ContainsAny(target, "\"\\ %") || u.Scheme != "https" || u.Opaque != "" ||
		u.User != nil || u.Host == "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path == "" ||
		u.RawPath != "" || u.String() != target || strings.ToLower(u.Host) != u.Host ||
		strings.HasSuffix(u.Hostname(), ".") || strings.HasSuffix(u.Host, ":") ||
		strings.Contains(u.Path, "/./") || strings.Contains(u.Path, "/../") || strings.HasSuffix(u.Path, "/.") || strings.HasSuffix(u.Path, "/..")
	if !bad && u.Port() != "" {
		port, perr := strconv.Atoi(u.Port())
		bad = perr != nil || port < 1 || port > 65535 || port == 443 || strconv.Itoa(port) != u.Port()
	}
	if bad {
		return "", "", fmt.Errorf("endpoint %q is not canonical", target)
	}
	return target, u.Host, nil
}

func token(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0) {
			return false
		}
	}
	return true
}

func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
