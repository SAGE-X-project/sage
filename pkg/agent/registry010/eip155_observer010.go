package registry010

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// RecordSnapshot carries a complete REG-01 record and its eip155 observation.
// It is a local trusted reader boundary, never a wire format or peer assertion.
// The reader must authenticate its source, code/ABI and history, including
// immutable keys/controller and historical KEM endorsement authority. Flags
// alone prove none of these. Every record component comes from one finalized
// block; the hashes are lowercase hex without a 0x prefix, as in Snapshot.
type RecordSnapshot struct {
	Source, Registry, Network                                    string
	Record                                                       []byte
	BlockHash, RecordBlockHash, KeysBlockHash, ServicesBlockHash string
	AcquiredMS                                                   int64
	Ready, Validated, Finalized, Conflicting                     bool
}

// RecordReader reads a complete record afresh. No concrete RPC or ABI binding
// is supplied here. Its trusted observation is not a protected-operation grant.
type RecordReader interface {
	ReadRecord(context.Context, string) (RecordSnapshot, error)
}

// EIP155Observer checks a configured complete reader, all record proofs and
// durable finalized version continuity for public DID resolution. It does not
// select operation keys or bypass Gate for protected messages.
type EIP155Observer struct {
	gate   *Gate
	reader RecordReader
}

// NewEIP155Observer requires explicit local trust, clock and durable history.
// Blockchain must be true and Registry must be the canonical eip155 locator.
func NewEIP155Observer(c Config, r RecordReader, clock Clock, store Store) (*EIP155Observer, error) {
	if err := checkConfig(c); err != nil {
		return nil, err
	}
	if !c.Blockchain || !strings.HasPrefix(c.Registry, "eip155:") || r == nil || clock == nil || store == nil {
		return nil, ErrRejected
	}
	return &EIP155Observer{gate: &Gate{cfg: c, clock: clock, store: store}, reader: r}, nil
}

// Observe returns a copied complete record and the local completion time only
// after the final freshness gate. Missing fields never acquire default values.
func (o *EIP155Observer) Observe(ctx context.Context, did string) (RecordSnapshot, Stamp, error) {
	g := o.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	deny := func(err error) (RecordSnapshot, Stamp, error) { return RecordSnapshot{}, Stamp{}, err }
	if !g.validDID(did) {
		return deny(ErrInvalidRecord010)
	}
	if ctx.Err() != nil {
		return deny(ErrUnreachable)
	}
	start, err := g.sample()
	if err != nil {
		return deny(err)
	}
	s, err := o.reader.ReadRecord(ctx, did)
	if err != nil || ctx.Err() != nil {
		return deny(ErrUnreachable)
	}
	if len(s.Record) > maxWebRecord010 {
		return deny(ErrSizeExceeded010)
	}
	s.Record = bytes.Clone(s.Record)
	now, err := g.sample()
	if err != nil {
		return deny(err)
	}
	if !s.Ready || s.Source != g.cfg.Source || s.Registry != g.cfg.Registry || s.Network != g.cfg.Network {
		return deny(ErrUnreachable)
	}
	if !s.Validated || !s.Finalized || s.Conflicting || !fresh(start, now, s.AcquiredMS) || !hex32.MatchString(s.BlockHash) || s.RecordBlockHash != s.BlockHash || s.KeysBlockHash != s.BlockHash || s.ServicesBlockHash != s.BlockHash {
		return deny(ErrStale)
	}
	// An active record whose keys have expired still resolves for inspection.
	// Its projection carries no usable key and grants no message authority.
	if err := checkEIP155Record010(s.Record, did, now.Unix); err != nil {
		return deny(err)
	}
	var header struct {
		Version string
		State   string
	}
	if json.Unmarshal(s.Record, &header) != nil {
		return deny(ErrInvalidRecord010)
	}
	canonical, err := rfc8785.Canonicalize(s.Record)
	if err != nil {
		return deny(ErrInvalidRecord010)
	}
	digest := sha256.Sum256(canonical)
	version, err := strconv.ParseUint(header.Version, 10, 64)
	if err != nil {
		return deny(ErrInvalidRecord010)
	}
	if err := g.store.Advance(Scope{g.cfg.Registry, did}, version, hex.EncodeToString(digest[:]), header.State == "deactivated"); err != nil {
		return deny(err)
	}
	now, err = g.sample()
	if err != nil {
		return deny(err)
	}
	if ctx.Err() != nil {
		return deny(ErrUnreachable)
	}
	if !fresh(start, now, s.AcquiredMS) {
		return deny(ErrStale)
	}
	return s, now, nil
}

func checkEIP155Record010(raw []byte, did string, now int64) error {
	p, err := sagedid.ParseDID010(did)
	if err != nil || p.Kind != "eip155" {
		return ErrInvalidRecord010
	}
	if err := checkRegistryRecordShape010(raw, did, now, false); err != nil {
		return err
	}
	var r struct {
		Controller string
		Keys       []webProofKey010
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Controller) != 42 || !strings.HasPrefix(r.Controller, "0x") {
		return ErrInvalidRecord010
	}
	for _, c := range r.Controller[2:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ErrInvalidRecord010
		}
	}
	return checkRegistryKeyProofs010(r.Keys, did)
}
