// This bounded example exposes only the REG-08 media subcondition to a local
// Inspector process. It does not fetch or authorize a Registry record.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type request struct {
	HeaderLines  [][]string `json:"header_lines"`
	TrailerLines [][]string `json:"trailer_lines"`
}

func fields(lines [][]string) ([]registry010.HeaderField010, bool) {
	result := make([]registry010.HeaderField010, 0, len(lines))
	for _, line := range lines {
		if len(line) != 2 {
			return nil, false
		}
		result = append(result, registry010.HeaderField010{Name: line[0], Value: line[1]})
	}
	return result, true
}

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16*1024+1))
	if err != nil || len(raw) > 16*1024 {
		os.Exit(2)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var req request
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	header, ok := fields(req.HeaderLines)
	if !ok {
		os.Exit(2)
	}
	trailer, ok := fields(req.TrailerLines)
	if !ok {
		os.Exit(2)
	}
	verdict := "MEDIA_ACCEPT"
	if registry010.CheckWebRegistryMedia010(header, trailer) != nil {
		verdict = "RECORD_INVALID"
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict}) != nil {
		os.Exit(2)
	}
}
