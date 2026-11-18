// Package apileak is a fixture for TestCheckerFindsKnownExposures. Every
// declaration except Clean, holder.String and holder.Public exposes a
// private key in a way the API checker must report.
package apileak

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/mlkem"

	"github.com/sage-x-project/sage/core/secret"
)

func Direct() *ecdh.PrivateKey { return nil }

type Holder struct{ *ecdh.PrivateKey }

type SignKey = ed25519.PrivateKey

func Alias() SignKey { return nil }

type Scalar ecdh.PrivateKey

func Defined() *Scalar { return nil }

type KeyFunc func() *ecdh.PrivateKey

func Getter() KeyFunc { return nil }

func Visit(f func(*ecdh.PrivateKey)) {}

func Load(dst *ed25519.PrivateKey) error { return nil }

type Seeder interface{ Seed() ed25519.PrivateKey }

type Exposer interface{ Seeder }

func DecapKey() *mlkem.DecapsulationKey768 { return nil }

func Generic[K interface{ *ecdh.PrivateKey }]() {}

type Field struct{ Key ed25519.PrivateKey }

func Slice() []ed25519.PrivateKey { return nil }

type holder struct{ p *ecdh.PrivateKey }

func (h *holder) Bytes() []byte                        { return h.p.Bytes() }
func (h *holder) Scalar() [32]byte                     { return [32]byte{} }
func (h *holder) Into(dst *[]byte)                     {}
func (h *holder) String() string                       { return "[redacted]" }
func (h *holder) Public() *ecdh.PublicKey              { return h.p.PublicKey() }
func NewHolder() interface{ Public() *ecdh.PublicKey } { return &holder{} }

type secretHolder struct{ s secret.Bytes }

func (h secretHolder) Raw() string { return "" }

func Export() secret.Bytes { return secret.Bytes{} }

func Clean() *ecdh.PublicKey { return nil }
