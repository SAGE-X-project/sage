package hpke

import (
	"context"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"

	"github.com/google/uuid"
)

// X25519Custody010 is protected custody for one fixed registered X25519 KEM
// key. The endpoint never receives the private key: ECDH returns the raw
// X25519 shared value with the supplied 32-byte peer public key, and must
// refuse an all-zero result. Both calls must be bounded, cancellation-aware,
// concurrency-safe and immutable for the endpoint's lifetime. The returned
// shared value lets its holder derive this handshake's secrets, so custody
// separation protects the long-term key, not a compromised host's sessions.
type X25519Custody010 interface {
	PublicKey(context.Context) ([]byte, error)
	ECDH(context.Context, []byte) ([]byte, error)
}

// RFC 9180 identifiers for DHKEM(X25519, HKDF-SHA256), HKDF-SHA256 and
// ChaCha20Poly1305, the only suite of the 0.10.0 handshake.
const (
	hpkeKEMID010  = 0x0020
	hpkeKDFID010  = 0x0001
	hpkeAEADID010 = 0x0003
)

func hpkeSuiteKEM010() []byte {
	return binary.BigEndian.AppendUint16([]byte("KEM"), hpkeKEMID010)
}

func hpkeSuite010() []byte {
	s := binary.BigEndian.AppendUint16([]byte("HPKE"), hpkeKEMID010)
	s = binary.BigEndian.AppendUint16(s, hpkeKDFID010)
	return binary.BigEndian.AppendUint16(s, hpkeAEADID010)
}

func labeledExtract010(suite, salt []byte, label string, ikm []byte) ([]byte, error) {
	input := append(append(append([]byte("HPKE-v1"), suite...), label...), ikm...)
	return hkdf.Extract(sha256.New, input, salt)
}

func labeledExpand010(suite, prk []byte, label string, info []byte, n int) ([]byte, error) {
	if n < 0 || n > 0xffff {
		return nil, errDerivation010
	}
	labeled := binary.BigEndian.AppendUint16(nil, uint16(n))
	labeled = append(append(append(append(labeled, "HPKE-v1"...), suite...), label...), info...)
	return hkdf.Expand(sha256.New, prk, string(labeled), n)
}

// custodyECDH010 calls custody with an owned copy and turns a panic into a refusal.
func custodyECDH010(ctx context.Context, kem X25519Custody010, peer []byte) (dh []byte, err error) {
	defer func() {
		if recover() != nil {
			dh, err = nil, errDerivation010
		}
	}()
	if kem == nil || ctx == nil || ctx.Err() != nil {
		return nil, errDerivation010
	}
	return kem.ECDH(ctx, append([]byte(nil), peer...))
}

// hpkeOpenExport010 is the RFC 9180 base-mode receiver exporter with the KEM
// Diffie-Hellman delegated to custody: Decap's DH(skR, pkE), ExtractAndExpand,
// KeySchedule and Export(exporterContext, n).
func hpkeOpenExport010(ctx context.Context, kem X25519Custody010, pkR, enc, info, exporterContext []byte, n int) ([]byte, error) {
	if len(pkR) != 32 || len(enc) != 32 {
		return nil, errDerivation010
	}
	if _, e := ecdh.X25519().NewPublicKey(enc); e != nil {
		return nil, errDerivation010
	}
	dh, e := custodyECDH010(ctx, kem, enc)
	if e != nil || len(dh) != 32 || ctx.Err() != nil {
		return nil, errDerivation010
	}
	dh = append([]byte(nil), dh...)
	defer zeroBytes(dh)
	var zero [32]byte
	if subtle.ConstantTimeCompare(dh, zero[:]) == 1 {
		return nil, errDerivation010
	}
	kemSuite := hpkeSuiteKEM010()
	prk, e := labeledExtract010(kemSuite, nil, "eae_prk", dh)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(prk)
	shared, e := labeledExpand010(kemSuite, prk, "shared_secret", append(append([]byte(nil), enc...), pkR...), 32)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(shared)
	suite := hpkeSuite010()
	pskIDHash, e := labeledExtract010(suite, nil, "psk_id_hash", nil)
	if e != nil {
		return nil, errDerivation010
	}
	infoHash, e := labeledExtract010(suite, nil, "info_hash", info)
	if e != nil {
		return nil, errDerivation010
	}
	context010 := append(append([]byte{0x00}, pskIDHash...), infoHash...)
	secret, e := labeledExtract010(suite, shared, "secret", nil)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(secret)
	exporter, e := labeledExpand010(suite, secret, "exp", context010, 32)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(exporter)
	return labeledExpand010(suite, exporter, "sec", exporterContext, n)
}

// deriveResponderCustody010 is DeriveResponder010 with the KEM key in custody.
func deriveResponderCustody010(ctx context.Context, raw []byte, kem X25519Custody010, pkR, e2ePrivate []byte, kid string) (*Derivation010, error) {
	m, dom, e := initiation010(raw)
	if e != nil {
		return nil, e
	}
	if !uuid010.MatchString(kid) {
		return nil, errDerivation010
	}
	es, e := ecdh.X25519().NewPrivateKey(e2ePrivate)
	if e != nil {
		return nil, errDerivation010
	}
	enc, e := binary010(m["enc"], 32)
	if e != nil {
		return nil, e
	}
	exporter, e := hpkeOpenExport010(ctx, kem, pkR, enc, dom.Info, dom.ExportContext, 32)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(exporter)
	m["ephS"] = base64.RawURLEncoding.EncodeToString(es.PublicKey().Bytes())
	m["kid"] = kid
	return finish010(m, exporter, e2ePrivate, "ephC")
}

// respondFreshCustody010 is RespondFresh010 with the KEM key in custody.
func respondFreshCustody010(ctx context.Context, raw []byte, kem X25519Custody010, pkR []byte) (*Derivation010, error) {
	if _, _, e := initiation010(raw); e != nil {
		return nil, e
	}
	es, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		return nil, errDerivation010
	}
	private := es.Bytes()
	defer zeroBytes(private)
	kid, e := uuid.NewRandom()
	if e != nil {
		return nil, errDerivation010
	}
	return deriveResponderCustody010(ctx, raw, kem, pkR, private, kid.String())
}
