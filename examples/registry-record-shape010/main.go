// This local adapter exposes structural REG-08 checks, not record authority.
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
	Body        string `json:"body"`
	ExpectedDID string `json:"expected_did"`
	Now         int64  `json:"now"`
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
	verdict := "SHAPE_ACCEPT"
	if err := registry010.CheckWebRegistryRecordShape010([]byte(req.Body), req.ExpectedDID, req.Now); err != nil {
		verdict = "RECORD_INVALID"
		if errors.Is(err, registry010.ErrSizeExceeded010) {
			verdict = "SIZE_EXCEEDED"
		}
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict}) != nil {
		os.Exit(2)
	}
}
