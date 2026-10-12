package didsource010

import (
	"context"
	"errors"

	"github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/did"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// EIP155Source supplies complete, freshly validated eip155 observations to
// did-go. It includes no RPC, inferred ABI, deployed registry or trust default.
// Successful public resolution is not protected-operation authorization.
type EIP155Source struct{ observer *registry010.EIP155Observer }

var _ sage.Source = (*EIP155Source)(nil)

// NewEIP155Source requires explicit reader, clock, history and trust configuration.
func NewEIP155Source(c registry010.Config, reader registry010.RecordReader, clock registry010.Clock, store registry010.Store) (*EIP155Source, error) {
	observer, err := registry010.NewEIP155Observer(c, reader, clock, store)
	if err != nil {
		return nil, err
	}
	return &EIP155Source{observer: observer}, nil
}

// Observe rejects malformed and unsupported identifiers before any read, then
// maps the complete observation. Source identity and completion time belong to
// the configured local observer, not a peer timestamp.
func (s *EIP155Source) Observe(ctx context.Context, d did.DID) (sage.Record, error) {
	p, err := sagedid.ParseDID010(d.String())
	switch {
	case errors.Is(err, sagedid.ErrUnknownDIDKind010):
		return sage.Record{}, wrap(sage.CodeIDUnknownKind, err)
	case err != nil:
		return sage.Record{}, wrap(sage.CodeIDMalformed, err)
	case p.Kind != "eip155":
		return sage.Record{}, wrap(sage.CodeIDUnknownKind, errors.New("this source reads eip155 registries"))
	}
	observed, now, err := s.observer.Observe(ctx, d.String())
	if err != nil {
		code := sage.CodeRecordUnreachable
		switch {
		case errors.Is(err, registry010.ErrStale):
			code = sage.CodeRecordStale
		case errors.Is(err, registry010.ErrSizeExceeded010):
			code = sage.CodeSizeExceeded
		case errors.Is(err, registry010.ErrInvalidRecord010), errors.Is(err, registry010.ErrRejected):
			code = sage.CodeRecordInvalid
		}
		return sage.Record{}, wrap(code, err)
	}
	r, err := parseRecordBody(observed.Record, d)
	if err != nil {
		return sage.Record{}, wrap(sage.CodeRecordInvalid, err)
	}
	r.Observation = sage.Observation{BlockHash: observed.BlockHash, Source: observed.Source, Time: now.Unix}
	return r, nil
}
