// This local adapter exposes REG-08 request and response policy decisions only.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type request struct {
	ExpectedDID    string                       `json:"expected_did"`
	AllowedOrigins []string                     `json:"allowed_origins"`
	Status         int                          `json:"status"`
	Headers        []registry010.HeaderField010 `json:"headers"`
	Trailers       []registry010.HeaderField010 `json:"trailers"`
}

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 8193))
	if err != nil || len(raw) > 8192 {
		os.Exit(2)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var req request
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	url, err := registry010.WebRegistryRequestURL010(req.ExpectedDID, req.AllowedOrigins)
	if err == nil {
		err = registry010.CheckWebRegistryResponsePolicy010(req.Status, req.Headers, req.Trailers)
	}
	verdict := "POLICY_ACCEPT"
	if errors.Is(err, registry010.ErrUnreachable) {
		verdict = "RECORD_UNREACHABLE"
	} else if err != nil {
		verdict = "RECORD_INVALID"
	}
	if verdict != "POLICY_ACCEPT" {
		url = ""
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict, "url": url}) != nil {
		os.Exit(2)
	}
}
