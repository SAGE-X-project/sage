package guard010_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sage-x-project/sage/pkg/agent/guard010"
	"strings"
)

// Fixture controls emulate trusted host services only. They are not a wire
// authorization format, registry implementation, or production policy engine.
type guardFixture map[string]json.RawMessage

func (f guardFixture) s(k string) string { var s string; _ = json.Unmarshal(f[k], &s); return s }
func (f guardFixture) yes(k string) bool { var b bool; _ = json.Unmarshal(f[k], &b); return b }
func (f guardFixture) Now(context.Context) (int64, error) {
	var n int64
	if !f.yes("clock_trusted") || json.Unmarshal(f["now"], &n) != nil {
		return 0, guard010.ErrInvalid
	}
	return n, nil
}
func (f guardFixture) ActiveKey(_ context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if !f.yes("active_key") {
		return nil, guard010.ErrInvalid
	}
	if expected := f.s("expected_issuer"); expected != "" && issuer != expected {
		return nil, guard010.ErrInvalid
	}
	b, e := hex.DecodeString(f.s("public_key_hex"))
	return ed25519.PublicKey(b), e
}
func (f guardFixture) Bindings(_ context.Context, issuer, requestID string) (string, []byte, []byte, error) {
	if issuer != f.s("expected_issuer") {
		return "", nil, nil, guard010.ErrInvalid
	}
	return f.s("original_digest"), f["approved_policy"], f["approved_manifest"], nil
}
func (f guardFixture) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	var schema struct {
		Tool       string
		Required   []string
		Properties map[string]string
	}
	var a map[string]any
	if !f.yes("policy_allow") || json.Unmarshal(f["tool_schema"], &schema) != nil || schema.Tool != tool || json.Unmarshal(args, &a) != nil {
		return guard010.ErrInvalid
	}
	for _, k := range schema.Required {
		if _, ok := a[k]; !ok {
			return guard010.ErrInvalid
		}
	}
	for k, v := range a {
		if schema.Properties[k] != "string" {
			return guard010.ErrInvalid
		}
		if _, ok := v.(string); !ok {
			return guard010.ErrInvalid
		}
	}
	return nil
}
func (f guardFixture) Intent(_ context.Context, request, call string) ([]byte, error) {
	if !f.yes("outstanding") {
		return nil, guard010.ErrInvalid
	}
	return f["intent_envelope"], nil
}
func guardObserve(op string, raw json.RawMessage) (string, map[string]any, error) {
	f := guardFixture{}
	if e := json.Unmarshal(raw, &f); e != nil {
		return "", nil, e
	}
	output := map[string]any{}
	var e error
	switch op {
	case "sage.guard.original.commit":
		var items []struct {
			Hex    *string
			Byte   int
			Length int
		}
		if e = json.Unmarshal(f["items"], &items); e != nil {
			return "", nil, e
		}
		data := [][]byte{}
		for _, item := range items {
			var b []byte
			if item.Hex != nil {
				b, e = hex.DecodeString(*item.Hex)
			} else {
				if item.Length < 0 || item.Length > (1<<20)+1 || item.Byte < 0 || item.Byte > 255 {
					return "", nil, fmt.Errorf("invalid fixture recipe")
				}
				b = bytes.Repeat([]byte{byte(item.Byte)}, item.Length)
			}
			if e != nil {
				return "", nil, e
			}
			data = append(data, b)
		}
		var d string
		d, e = guard010.OriginalCommitment(data)
		output["original_digest"] = d
	case "sage.guard.manifest.verify":
		var items []struct {
			Path  string
			Bytes string `json:"bytes_hex"`
		}
		if e = json.Unmarshal(f["artifacts"], &items); e != nil {
			return "", nil, e
		}
		art := []guard010.Artifact{}
		for _, item := range items {
			b, err := hex.DecodeString(item.Bytes)
			if err != nil {
				return "", nil, err
			}
			art = append(art, guard010.Artifact{Path: item.Path, Bytes: b})
		}
		var d string
		d, e = guard010.VerifyManifest(f["manifest"], art)
		output["manifest_digest"] = d
	case "sage.guard.policy.commit":
		var d string
		d, e = guard010.PolicyCommitment(f["descriptor"])
		output["policy_digest"] = d
	case "sage.guard.json.bounds":
		var n, b int
		if json.Unmarshal(f["repeat_count"], &n) != nil || json.Unmarshal(f["repeat_byte"], &b) != nil || n < 0 || n > (1<<20)+1 || b < 0 || b > 255 {
			return "", nil, fmt.Errorf("invalid fixture recipe")
		}
		prefix, err := hex.DecodeString(f.s("prefix_hex"))
		if err != nil {
			return "", nil, err
		}
		suffix, err := hex.DecodeString(f.s("suffix_hex"))
		if err != nil {
			return "", nil, err
		}
		_, e = guard010.Canonicalize(append(append(prefix, bytes.Repeat([]byte{byte(b)}, n)...), suffix...))
		output["valid"] = true
	case "sage.guard.intent.verify", "sage.guard.result.verify":
		b, err := hex.DecodeString(f.s("envelope_hex"))
		if err != nil {
			return "", nil, err
		}
		if strings.Contains(op, "intent") {
			_, e = guard010.VerifyIntent(context.Background(), b, f.s("expected_recipient"), f, f)
		} else {
			_, e = guard010.VerifyResult(context.Background(), b, f, f)
		}
		output["valid"] = true
	default:
		return "UNSUPPORTED", map[string]any{}, nil
	}
	if e != nil {
		return "REJECT", map[string]any{}, nil
	}
	return "ACCEPT", output, nil
}
