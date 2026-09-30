// Local readback adapter for one journal and one authenticated HTTPS origin.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/netip"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type configuration struct {
	JournalPath         string   `json:"journal_path"`
	DID                 string   `json:"did"`
	Source              string   `json:"source"`
	AllowedOrigins      []string `json:"allowed_origins"`
	Destination         string   `json:"destination"`
	AllowedDestinations []string `json:"allowed_destinations"`
	Root                string   `json:"root"`
	Now                 int64    `json:"now"`
}

type readOnlyAuthority struct{}

func (readOnlyAuthority) AuthenticatedActor(context.Context) (string, error) {
	return "", registry010.ErrRejected
}
func (readOnlyAuthority) Delegated(context.Context, string, string, string, string, string) (bool, error) {
	return false, registry010.ErrRejected
}

func main() {
	var cfg configuration
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32_769))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cfg) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	destination, err := netip.ParseAddrPort(cfg.Destination)
	if err != nil {
		os.Exit(2)
	}
	approved := make([]netip.AddrPort, 0, len(cfg.AllowedDestinations))
	for _, value := range cfg.AllowedDestinations {
		parsed, parseErr := netip.ParseAddrPort(value)
		if parseErr != nil {
			os.Exit(2)
		}
		approved = append(approved, parsed)
	}
	root, err := base64.RawURLEncoding.Strict().DecodeString(cfg.Root)
	if err != nil {
		os.Exit(2)
	}
	journal, err := registry010.OpenWebRegistryWriteJournal010(cfg.JournalPath, cfg.DID, cfg.Source, readOnlyAuthority{}, false)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": "RECORD_UNREACHABLE"})
		return
	}
	_, err = registry010.ObserveWebRegistryJournal010(context.Background(), journal,
		cfg.AllowedOrigins, destination, approved, root, cfg.Now)
	if closeErr := journal.Close(); closeErr != nil {
		err = closeErr
	}
	verdict := "MATCH"
	if err == registry010.ErrStale {
		verdict = "RECORD_STALE"
	} else if err != nil {
		verdict = "RECORD_UNREACHABLE"
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict})
}
