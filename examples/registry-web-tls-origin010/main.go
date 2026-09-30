// Local Inspector adapter for a bounded, authenticated TLS handshake only.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type request struct {
	ExpectedDID         string   `json:"expected_did"`
	AllowedOrigins      []string `json:"allowed_origins"`
	Destination         string   `json:"destination"`
	AllowedDestinations []string `json:"allowed_destinations"`
	RootDER             string   `json:"root_der"`
}

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16385))
	if err != nil || len(raw) > 16384 {
		os.Exit(2)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var req request
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	destination, err := netip.ParseAddrPort(req.Destination)
	if err != nil {
		os.Exit(2)
	}
	approved := make([]netip.AddrPort, 0, len(req.AllowedDestinations))
	for _, entry := range req.AllowedDestinations {
		address, parseErr := netip.ParseAddrPort(entry)
		if parseErr != nil {
			os.Exit(2)
		}
		approved = append(approved, address)
	}
	root, err := base64.RawURLEncoding.DecodeString(req.RootDER)
	if err != nil {
		os.Exit(2)
	}
	err = registry010.CheckWebRegistryTLSOrigin010(context.Background(), req.ExpectedDID, req.AllowedOrigins, destination, approved, root)
	verdict := "TLS_ACCEPT"
	if errors.Is(err, registry010.ErrInvalidRecord010) {
		verdict = "RECORD_INVALID"
	} else if err != nil {
		verdict = "RECORD_UNREACHABLE"
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict}) != nil {
		os.Exit(2)
	}
}
