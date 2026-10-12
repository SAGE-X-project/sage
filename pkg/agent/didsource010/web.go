// Package didsource010 supplies verified SAGE 0.10.0 registry records to the
// did-go advanced/sage module, which presents them as DID documents and DID
// Resolution results (spec chapter 10 RESOLVE-01 to RESOLVE-05 and DIDP-01
// to DIDP-05).
//
// A Source here performs a new authoritative observation on every call and
// returns a record only after the registry checks of this core have passed.
// Which registry origins, destinations and roots are trusted is deployment
// configuration, never peer input. The documents built from these records
// are views for external DID consumers; they grant no authority for a
// protected operation (DIDP-04).
package didsource010

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/did"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// WebConfig is the trusted deployment configuration of a web registry
// observation (REG-08). It is never received from a peer.
type WebConfig struct {
	// AllowedOrigins are the HTTPS origins of the registry.
	AllowedOrigins []string
	// Destination is the address to connect to; it must be one of
	// AllowedDestinations.
	Destination         netip.AddrPort
	AllowedDestinations []netip.AddrPort
	// RootDER is the DER certificate of the trusted root.
	RootDER []byte
	// SourceID is the configured source identity that RESOLVE-03 reports,
	// printable ASCII of at most 2048 bytes.
	SourceID string
	// Now returns trusted Unix seconds. An error means the clock cannot be
	// trusted now, and the observation fails closed.
	Now func() (int64, error)
}

// fetchFunc reads and verifies one web registry response body.
type fetchFunc func(ctx context.Context, did string, allowedOrigins []string, destination netip.AddrPort, allowedDestinations []netip.AddrPort, rootDER []byte, now int64) ([]byte, error)

// WebSource observes did:sage:web records through the bounded REG-08
// adapter of registry010, which authenticates the TLS origin and checks the
// envelope, the record shape and every key proof.
type WebSource struct {
	cfg   WebConfig
	fetch fetchFunc
}

var _ sage.Source = (*WebSource)(nil)

// NewWebSource checks c and returns a source.
func NewWebSource(c WebConfig) (*WebSource, error) {
	if len(c.AllowedOrigins) == 0 || len(c.AllowedDestinations) == 0 || len(c.RootDER) == 0 || c.Now == nil {
		return nil, errors.New("didsource010: origins, destinations, a root and a clock are required")
	}
	if !printable(c.SourceID, 1, 2048) {
		return nil, errors.New("didsource010: the source identity is not 1 to 2048 printable ASCII bytes")
	}
	c.AllowedOrigins = append([]string(nil), c.AllowedOrigins...)
	c.AllowedDestinations = append([]netip.AddrPort(nil), c.AllowedDestinations...)
	c.RootDER = bytes.Clone(c.RootDER)
	return &WebSource{cfg: c, fetch: registry010.FetchWebRegistryRecord010}, nil
}

// Observe reads the current record of d. A did:sage identifier that is not
// canonical is id.malformed, and one of another registry kind is
// id.unknown-kind, before any connection. A failed read is
// record.unreachable, a response over the size bound size.exceeded, and a
// response that fails the envelope, shape or proof checks record.invalid.
func (s *WebSource) Observe(ctx context.Context, d did.DID) (sage.Record, error) {
	id := d.String()
	parsed, err := sagedid.ParseDID010(id)
	switch {
	case errors.Is(err, sagedid.ErrUnknownDIDKind010):
		return sage.Record{}, wrap(sage.CodeIDUnknownKind, err)
	case err != nil:
		return sage.Record{}, wrap(sage.CodeIDMalformed, err)
	case parsed.Kind != "web":
		return sage.Record{}, wrap(sage.CodeIDUnknownKind, fmt.Errorf("this source reads web registries, not %q", parsed.Kind))
	}
	now, err := s.cfg.Now()
	if err != nil {
		return sage.Record{}, wrap(sage.CodeRecordUnreachable, fmt.Errorf("trusted clock: %w", err))
	}
	body, err := s.fetch(ctx, id, s.cfg.AllowedOrigins, s.cfg.Destination, s.cfg.AllowedDestinations, s.cfg.RootDER, now)
	switch {
	case errors.Is(err, registry010.ErrSizeExceeded010):
		return sage.Record{}, wrap(sage.CodeSizeExceeded, err)
	case errors.Is(err, registry010.ErrInvalidRecord010):
		return sage.Record{}, wrap(sage.CodeRecordInvalid, err)
	case err != nil:
		return sage.Record{}, wrap(sage.CodeRecordUnreachable, err)
	}
	rec, err := parseWebBody(body, d)
	if err != nil {
		return sage.Record{}, wrap(sage.CodeRecordInvalid, err)
	}
	done, err := s.cfg.Now()
	if err != nil {
		return sage.Record{}, wrap(sage.CodeRecordUnreachable, fmt.Errorf("trusted clock: %w", err))
	}
	rec.Observation.Source = s.cfg.SourceID
	rec.Observation.Time = done
	return rec, nil
}

// wrap returns a SAGE error with no public detail and err as its cause.
func wrap(c sage.Code, err error) *sage.Error { return &sage.Error{Code: c, Cause: err} }

type webEnvelope struct {
	Record webRecord   `json:"record"`
	Issued json.Number `json:"issued"`
}

type webRecord struct {
	ID         string       `json:"id"`
	Controller string       `json:"controller"`
	State      string       `json:"state"`
	Version    string       `json:"version"`
	Keys       []webKey     `json:"keys"`
	Services   []webService `json:"services"`
}

type webKey struct {
	Name    string       `json:"name"`
	Alg     string       `json:"alg"`
	Key     string       `json:"key"`
	State   string       `json:"state"`
	Expires *json.Number `json:"expires"`
}

type webService struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// parseWebBody converts a verified REG-08 response body into a record. The
// body has already passed the envelope, shape and proof checks; this only
// maps its members. The issued integer is the RESOLVE-03 observedAt of a web
// registry.
func parseWebBody(body []byte, d did.DID) (sage.Record, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var env webEnvelope
	if err := dec.Decode(&env); err != nil {
		return sage.Record{}, fmt.Errorf("record body: %w", err)
	}
	if env.Record.ID != d.String() {
		return sage.Record{}, errors.New("the record id is not the requested identifier")
	}
	issued, err := strconv.ParseInt(env.Issued.String(), 10, 64)
	if err != nil {
		return sage.Record{}, fmt.Errorf("issued: %w", err)
	}
	rec := sage.Record{
		DID:         d,
		Controller:  env.Record.Controller,
		State:       sage.State(env.Record.State),
		Version:     env.Record.Version,
		Observation: sage.Observation{Issued: &issued},
	}
	for _, k := range env.Record.Keys {
		material, err := base64.RawURLEncoding.Strict().DecodeString(k.Key)
		if err != nil {
			return sage.Record{}, fmt.Errorf("key %q: %w", k.Name, err)
		}
		key := sage.Key{Name: k.Name, Alg: k.Alg, Material: material, State: sage.KeyState(k.State)}
		if k.Expires != nil {
			n, err := strconv.ParseInt(k.Expires.String(), 10, 64)
			if err != nil {
				return sage.Record{}, fmt.Errorf("key %q expiry: %w", k.Name, err)
			}
			key.Expires = &n
		}
		rec.Keys = append(rec.Keys, key)
	}
	for _, sv := range env.Record.Services {
		rec.Services = append(rec.Services, sage.Service{Name: sv.Name, Type: sv.Type, URI: sv.URI})
	}
	return rec, nil
}

func printable(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
