// Package wire011 implements SAGE 0.11.0 signed plain messages: the chapter 08
// transport envelope with its inner signature and the chapter 03 HTTP message
// signature that binds it to HTTP routing. Session (HPKE) encoding is not part
// of this package; a plain envelope is authenticated but not confidential.
//
// Every check fails closed. Callers outside the trusted boundary see only
// ErrAuthentication; the internal reason is reported through the Tracer.
package wire011

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

const (
	// Version is the SAGE wire and HTTP version this package speaks.
	Version = "0.11.0"
	// Tag is the RFC 9421 signature tag of the 0.11.0 HTTP profile.
	Tag = "sage-0.11.0"

	requestDomain  = "sage-wire-request|" + Version + "\n"
	responseDomain = "sage-wire-response|" + Version + "\n"

	maxPayload   = 8 << 20
	maxLifetime  = 300
	clockSkew    = 30
	maxDIDLength = 255
)

// ErrAuthentication is the only error untrusted callers may learn.
var ErrAuthentication = errors.New("wire011: authentication failed")

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	metaKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	responseErrors = map[string]bool{"authentication_failed": true, "policy_denied": true, "operation_failed": true, "unavailable": true}
	// Optional members must be omitted, not present with an empty value.
	optionalMembers = memberSet("context_id task_id error metadata")
	requestMembers  = memberSet("version id did recipient kid created expires nonce encoding signature payload context_id task_id metadata")
	responseMembers = memberSet("version id did recipient kid created expires nonce encoding signature message_id request_hash success data error context_id task_id metadata")
)

// Request is a chapter 08 request envelope with plain encoding.
type Request struct {
	Version   string            `json:"version"`
	ID        string            `json:"id"`
	DID       string            `json:"did"`
	Recipient string            `json:"recipient"`
	KID       string            `json:"kid"`
	Created   int64             `json:"created"`
	Expires   int64             `json:"expires"`
	Nonce     string            `json:"nonce"`
	Encoding  string            `json:"encoding"`
	Signature string            `json:"signature,omitempty"`
	Payload   string            `json:"payload"`
	ContextID string            `json:"context_id,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Response is a chapter 08 response envelope with plain encoding.
type Response struct {
	Version     string            `json:"version"`
	ID          string            `json:"id"`
	DID         string            `json:"did"`
	Recipient   string            `json:"recipient"`
	KID         string            `json:"kid"`
	Created     int64             `json:"created"`
	Expires     int64             `json:"expires"`
	Nonce       string            `json:"nonce"`
	Encoding    string            `json:"encoding"`
	Signature   string            `json:"signature,omitempty"`
	MessageID   string            `json:"message_id"`
	RequestHash string            `json:"request_hash"`
	Success     bool              `json:"success"`
	Data        string            `json:"data"`
	Error       string            `json:"error,omitempty"`
	ContextID   string            `json:"context_id,omitempty"`
	TaskID      string            `json:"task_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func memberSet(names string) map[string]bool {
	out := map[string]bool{}
	for _, name := range strings.Fields(names) {
		out[name] = true
	}
	return out
}

// decodeClosed parses a closed JSON object: unknown or duplicate members and
// trailing data are refused before the typed decode.
func decodeClosed(raw []byte, allowed map[string]bool, out any) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8")
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return fmt.Errorf("not an object")
	}
	seen := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, ok := tok.(string)
		if !ok || !allowed[name] || seen[name] {
			return fmt.Errorf("member %q not allowed", tok)
		}
		seen[name] = true
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil || string(value) == "null" {
			return fmt.Errorf("member %q invalid", name)
		}
		if optionalMembers[name] && (string(value) == `""` || string(value) == "{}") {
			return fmt.Errorf("member %q present but empty", name)
		}
		if name == "metadata" {
			if err := closedStringMap(value); err != nil {
				return fmt.Errorf("metadata: %w", err)
			}
		}
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
}

// closedStringMap requires a JSON object of string values without duplicates.
func closedStringMap(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("not an object")
	}
	seen := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, _ := tok.(string)
		if seen[name] {
			return fmt.Errorf("duplicate key %q", name)
		}
		seen[name] = true
		tok, err = dec.Token()
		if _, isString := tok.(string); err != nil || !isString {
			return fmt.Errorf("value of %q is not a string", name)
		}
	}
	return nil
}

func validCommon(version, id, did, recipient, kid string, created, expires int64, nonce, encoding, signature string,
	contextID, taskID string, metadata map[string]string) error {
	switch {
	case version != Version:
		return fmt.Errorf("version")
	case !uuidPattern.MatchString(id):
		return fmt.Errorf("id")
	case !validDID(did) || !validDID(recipient) || did == recipient:
		return fmt.Errorf("did")
	case kidDID(kid) != did:
		return fmt.Errorf("kid")
	case created < 0 || expires <= created || expires-created > maxLifetime:
		return fmt.Errorf("lifetime")
	case !validNonce(nonce):
		return fmt.Errorf("nonce")
	case encoding != "plain":
		return fmt.Errorf("encoding")
	case contextID != "" && !uuidPattern.MatchString(contextID), taskID != "" && !uuidPattern.MatchString(taskID):
		return fmt.Errorf("context")
	}
	if err := validMetadata(metadata); err != nil {
		return err
	}
	if signature != "" {
		if _, err := decodeB64(signature, 64); err != nil {
			return fmt.Errorf("signature encoding")
		}
	}
	return nil
}

func validDID(did string) bool {
	return len(did) <= maxDIDLength && strings.HasPrefix(did, "did:") && !strings.ContainsAny(did, "#?/ ") && ascii(did)
}

// kidDID returns the DID of a full key URL "did#fragment", or "" if malformed.
func kidDID(kid string) string {
	did, fragment, ok := strings.Cut(kid, "#")
	if !ok || fragment == "" || strings.Contains(fragment, "#") || !validDID(did) || len(kid) > maxDIDLength+65 {
		return ""
	}
	return did
}

func validNonce(nonce string) bool {
	_, err := decodeB64(nonce, 16)
	return err == nil && len(nonce) == 22
}

func validMetadata(m map[string]string) error {
	if len(m) > 32 {
		return fmt.Errorf("metadata entries")
	}
	for k, v := range m {
		if !metaKeyPattern.MatchString(k) || len(v) > 1024 {
			return fmt.Errorf("metadata entry")
		}
	}
	if len(m) > 0 {
		raw, err := rfc8785.Marshal(m)
		if err != nil || len(raw) > 4096 {
			return fmt.Errorf("metadata size")
		}
	}
	return nil
}

// decodeB64 accepts only canonical unpadded base64url of exactly size bytes
// (size < 0 means any size up to maxPayload).
func decodeB64(s string, size int) ([]byte, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s)
	if err != nil || base64.RawURLEncoding.EncodeToString(raw) != s {
		return nil, fmt.Errorf("base64url")
	}
	if size >= 0 && len(raw) != size || size < 0 && len(raw) > maxPayload {
		return nil, fmt.Errorf("size")
	}
	return raw, nil
}

func b64(raw []byte) string { return base64.RawURLEncoding.EncodeToString(raw) }

func signingBytes(domain string, envelope any) ([]byte, error) {
	raw, err := rfc8785.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	return append([]byte(domain), raw...), nil
}

func (r Request) unsigned() Request   { r.Signature = ""; return r }
func (r Response) unsigned() Response { r.Signature = ""; return r }

func signEnvelope(domain string, unsigned any, key ed25519.PrivateKey) (string, error) {
	msg, err := signingBytes(domain, unsigned)
	if err != nil {
		return "", err
	}
	return b64(ed25519.Sign(key, msg)), nil
}

func verifyEnvelope(domain string, unsigned any, signature string, key ed25519.PublicKey) bool {
	sig, err := decodeB64(signature, 64)
	if err != nil {
		return false
	}
	msg, err := signingBytes(domain, unsigned)
	return err == nil && ed25519.Verify(key, msg, sig)
}

// requestHash is base64url(SHA-256(JCS(complete signed request))).
func requestHash(r Request) (string, error) {
	raw, err := rfc8785.Marshal(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return b64(sum[:]), nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := fmt.Sprintf("%x", b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}

func newNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return b64(b[:]), nil
}

func ascii(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
