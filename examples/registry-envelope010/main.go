// This local adapter exposes only the untrusted REG-08 JSON envelope check.
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
	Body string `json:"body"`
	Now  int64  `json:"now"`
}

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 280001))
	if err != nil || len(raw) > 280000 {
		os.Exit(2)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var req request
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	verdict := "ENVELOPE_ACCEPT"
	if err := registry010.CheckWebRegistryEnvelope010([]byte(req.Body), req.Now); err != nil {
		verdict = "RECORD_INVALID"
		if errors.Is(err, registry010.ErrSizeExceeded010) {
			verdict = "SIZE_EXCEEDED"
		}
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict}) != nil {
		os.Exit(2)
	}
}
