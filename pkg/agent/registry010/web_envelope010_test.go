package registry010

import (
	"errors"
	"strings"
	"testing"
)

func TestWebRegistryEnvelope010(t *testing.T) {
	const good = `{"record":{},"issued":100,"expires":105}`
	for _, tc := range []struct {
		name string
		body string
		now  int64
		want error
	}{
		{"issued boundary", good, 100, nil},
		{"before expiry", good, 104, nil},
		{"at expiry", good, 105, ErrInvalidRecord010},
		{"before issue", good, 99, ErrInvalidRecord010},
		{"long lifetime", `{"record":{},"issued":100,"expires":106}`, 100, ErrInvalidRecord010},
		{"zero lifetime", `{"record":{},"issued":100,"expires":100}`, 100, ErrInvalidRecord010},
		{"missing record", `{"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"nonobject record", `{"record":[],"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"unknown member", `{"record":{},"issued":100,"expires":105,"extra":0}`, 100, ErrInvalidRecord010},
		{"duplicate root", `{"record":{},"issued":100,"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"duplicate nested", `{"record":{"a":1,"a":2},"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"escaped duplicate", `{"record":{"a":1,"\u0061":2},"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"negative zero", `{"record":{},"issued":-0,"expires":105}`, 100, ErrInvalidRecord010},
		{"lone surrogate", `{"record":{"x":"\ud800"},"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
		{"fractional time", `{"record":{},"issued":100.5,"expires":105}`, 100, ErrInvalidRecord010},
		{"exact exponent", `{"record":{},"issued":1e2,"expires":105}`, 100, nil},
		{"string time", `{"record":{},"issued":"100","expires":105}`, 100, ErrInvalidRecord010},
		{"trailing object", good + `{}`, 100, ErrInvalidRecord010},
		{"invalid clock", good, 9007199254740992, ErrInvalidRecord010},
		{"depth overflow", `{"record":` + strings.Repeat("[", 64) + `0` + strings.Repeat("]", 64) + `,"issued":100,"expires":105}`, 100, ErrInvalidRecord010},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckWebRegistryEnvelope010([]byte(tc.body), tc.now)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if err := CheckWebRegistryEnvelope010(make([]byte, maxWebBody010+1), 100); !errors.Is(err, ErrSizeExceeded010) {
		t.Fatalf("oversize result: %v", err)
	}
	const start = `{"record":{"pad":"`
	const end = `"},"issued":100,"expires":105}`
	boundary := start + strings.Repeat("a", maxWebBody010-len(start)-len(end)) + end
	if err := CheckWebRegistryEnvelope010([]byte(boundary), 100); err != nil {
		t.Fatalf("exact body limit: %v", err)
	}
}
