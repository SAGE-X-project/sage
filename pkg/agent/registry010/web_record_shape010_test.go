package registry010

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func recordShapeFixture010() map[string]any {
	const did = "did:sage:web:agents.example.com:billing-bot"
	return map[string]any{
		"id": did, "controller": "operator", "state": "active", "version": "1",
		"keys": []any{map[string]any{
			"name": "sign-1", "alg": "ed25519",
			"key":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
			"proof": map[string]any{"signer": did + "#sign-1", "value": base64.RawURLEncoding.EncodeToString(make([]byte, 64))},
			"state": "accepted",
		}},
		"services": []any{map[string]any{"name": "api", "type": "A2A", "uri": "https://api.example.com/v1"}},
	}
}

func shapeBody010(t *testing.T, record map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"record": record, "issued": 100, "expires": 105})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestWebRegistryRecordShape010(t *testing.T) {
	const did = "did:sage:web:agents.example.com:billing-bot"
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
		want bool
	}{
		{"valid", func(map[string]any) {}, true},
		{"different id", func(r map[string]any) { r["id"] = "did:sage:web:other.example.com:billing-bot" }, false},
		{"extra field", func(r map[string]any) { r["extra"] = true }, false},
		{"zero version", func(r map[string]any) { r["version"] = "0" }, false},
		{"leading zero", func(r map[string]any) { r["version"] = "01" }, false},
		{"max version", func(r map[string]any) { r["version"] = "18446744073709551615" }, true},
		{"overflow version", func(r map[string]any) { r["version"] = "18446744073709551616" }, false},
		{"missing keys", func(r map[string]any) { r["keys"] = []any{} }, false},
		{"bad algorithm", func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["alg"] = "X25519" }, false},
		{"padded key", func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["key"] = strings.Repeat("A", 43) + "=" }, false},
		{"key extra", func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["private"] = "secret" }, false},
		{"proof signer", func(r map[string]any) {
			r["keys"].([]any)[0].(map[string]any)["proof"].(map[string]any)["signer"] = did + "#other"
		}, false},
		{"expired active", func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["expires"] = 100 }, false},
		{"fractional expiry", func(r map[string]any) { r["keys"].([]any)[0].(map[string]any)["expires"] = 100.5 }, false},
		{"service collision", func(r map[string]any) { r["services"].([]any)[0].(map[string]any)["name"] = "sign-1" }, false},
		{"service http", func(r map[string]any) { r["services"].([]any)[0].(map[string]any)["uri"] = "http://api.example.com" }, false},
		{"service userinfo", func(r map[string]any) {
			r["services"].([]any)[0].(map[string]any)["uri"] = "https://user@api.example.com"
		}, false},
		{"service fragment", func(r map[string]any) { r["services"].([]any)[0].(map[string]any)["uri"] = "https://api.example.com/#" }, false},
		{"service IP", func(r map[string]any) { r["services"].([]any)[0].(map[string]any)["uri"] = "https://127.0.0.1:8443/v1" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := recordShapeFixture010()
			tc.edit(record)
			err := CheckWebRegistryRecordShape010(shapeBody010(t, record), did, 100)
			if (err == nil) != tc.want {
				t.Fatalf("got %v, want success %v", err, tc.want)
			}
		})
	}
	record := recordShapeFixture010()
	encoded, _ := json.Marshal(record)
	spaces := strings.Repeat(" ", maxWebRecord010-len(encoded))
	boundary := []byte(`{"record":` + string(encoded[:1]) + spaces + string(encoded[1:]) + `,"issued":100,"expires":105}`)
	if err := CheckWebRegistryRecordShape010(boundary, did, 100); err != nil {
		t.Fatalf("exact record size: %v", err)
	}
	tooLarge := []byte(`{"record":` + string(encoded[:1]) + spaces + " " + string(encoded[1:]) + `,"issued":100,"expires":105}`)
	if err := CheckWebRegistryRecordShape010(tooLarge, did, 100); !errors.Is(err, ErrSizeExceeded010) {
		t.Fatalf("oversize record: %v", err)
	}
}
