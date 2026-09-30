package registry010

import (
	"errors"
	"testing"
)

func TestWebRegistryMedia010(t *testing.T) {
	tests := []struct {
		name    string
		header  []HeaderField010
		trailer []HeaderField010
		valid   bool
	}{
		{"exact", []HeaderField010{{"Content-Type", "application/json"}}, nil, true},
		{"case and OWS", []HeaderField010{{"cOnTeNt-TyPe", "\tApplication/JSON "}}, nil, true},
		{"wrong type", []HeaderField010{{"Content-Type", "text/plain"}}, nil, false},
		{"DID document", []HeaderField010{{"Content-Type", "application/did+json"}}, nil, false},
		{"problem detail", []HeaderField010{{"Content-Type", "application/problem+json"}}, nil, false},
		{"missing", nil, nil, false},
		{"parameter", []HeaderField010{{"Content-Type", "application/json; charset=utf-8"}}, nil, false},
		{"duplicate", []HeaderField010{{"Content-Type", "application/json"}, {"content-type", "application/json"}}, nil, false},
		{"combined", []HeaderField010{{"Content-Type", "application/json, text/plain"}}, nil, false},
		{"gzip", []HeaderField010{{"Content-Type", "application/json"}, {"Content-Encoding", "gzip"}}, nil, false},
		{"identity coding", []HeaderField010{{"Content-Type", "application/json"}, {"Content-Encoding", "identity"}}, nil, false},
		{"trailer type", []HeaderField010{{"Content-Type", "application/json"}}, []HeaderField010{{"Content-Type", "text/plain"}}, false},
		{"trailer coding", []HeaderField010{{"Content-Type", "application/json"}}, []HeaderField010{{"Content-Encoding", "gzip"}}, false},
		{"non ASCII fold", []HeaderField010{{"Content-Type", "application/jſon"}}, nil, false},
		{"line break", []HeaderField010{{"Content-Type", "application/json\r\nContent-Encoding: gzip"}}, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckWebRegistryMedia010(tc.header, tc.trailer)
			if tc.valid && err != nil {
				t.Fatalf("valid media rejected: %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrInvalidRecord010) {
				t.Fatalf("invalid media must return record.invalid, got %v", err)
			}
		})
	}
}
