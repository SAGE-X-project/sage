package hpke

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

var errDerivation010 = errors.New("invalid 0.10.0 HPKE derivation")
var bindingFields010 = []string{"v", "ctx", "initDid", "respDid", "initKid", "respKid", "kemKid", "suite", "combiner", "nonce"}
var uuid010 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var agent010 = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
var name010 = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var chain010 = regexp.MustCompile(`^[1-9][0-9]{0,31}$`)
var address010 = regexp.MustCompile(`^0x[0-9a-f]{40}$`)
var label010 = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// Domains010 contains public, locally recomputed HPKE context bytes.
type Domains010 struct{ Binding, Info, ExportContext []byte }

// Derivation010 is only a cryptographic result, not an authenticated session.
// The caller must validate signatures, selected current keys and pending lifetime.
// Caller owns Seed and must erase it when no longer needed.
type Derivation010 struct {
	Transcript, TH, Seed, AckTag []byte
	SID                          string
}

func fields010(raw []byte, names []string) (map[string]string, error) {
	if len(raw) > 16384 || !utf8.Valid(raw) {
		return nil, errDerivation010
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, errDerivation010
	}
	m := make(map[string]string)
	for d.More() {
		tok, err = d.Token()
		if err != nil {
			return nil, errDerivation010
		}
		k, ok := tok.(string)
		if !ok {
			return nil, errDerivation010
		}
		if _, ok = m[k]; ok {
			return nil, errDerivation010
		}
		var v *string
		if d.Decode(&v) != nil || v == nil {
			return nil, errDerivation010
		}
		for _, c := range *v {
			if c > 127 {
				return nil, errDerivation010
			}
		}
		m[k] = *v
	}
	if tok, err = d.Token(); err != nil || tok != json.Delim('}') {
		return nil, errDerivation010
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errDerivation010
	}
	if len(m) != len(names) {
		return nil, errDerivation010
	}
	for _, n := range names {
		if _, ok := m[n]; !ok {
			return nil, errDerivation010
		}
	}
	return m, nil
}

func did010(s string) bool {
	if len(s) > 256 {
		return false
	}
	p := strings.Split(s, ":")
	if len(p) < 5 || p[0] != "did" || p[1] != "sage" {
		return false
	}
	agent := p[len(p)-1]
	if !agent010.MatchString(agent) || agent == "." || agent == ".." {
		return false
	}
	switch p[2] {
	case "eip155":
		return len(p) == 6 && chain010.MatchString(p[3]) && address010.MatchString(p[4])
	case "web":
		if len(p) != 5 || len(p[3]) > 64 || net.ParseIP(p[3]) != nil {
			return false
		}
		for _, label := range strings.Split(p[3], ".") {
			if !label010.MatchString(label) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func key010(s, did string) bool {
	prefix := did + "#"
	return len(s) <= 289 && strings.HasPrefix(s, prefix) && name010.MatchString(strings.TrimPrefix(s, prefix))
}
func binary010(s string, n int) ([]byte, error) {
	b, e := base64.RawURLEncoding.Strict().DecodeString(s)
	if e != nil || len(b) != n || base64.RawURLEncoding.EncodeToString(b) != s {
		return nil, errDerivation010
	}
	return b, nil
}
func binding010(m map[string]string) (map[string]string, error) {
	b := make(map[string]string)
	for _, n := range bindingFields010 {
		b[n] = m[n]
	}
	if b["v"] != "0.10.0" || b["suite"] != "hpke-base+x25519+hkdf-sha256" || b["combiner"] != "e2e-x25519-hkdf-v1" || !uuid010.MatchString(b["ctx"]) {
		return nil, errDerivation010
	}
	if !did010(b["initDid"]) || !did010(b["respDid"]) || !key010(b["initKid"], b["initDid"]) || !key010(b["respKid"], b["respDid"]) || !key010(b["kemKid"], b["respDid"]) {
		return nil, errDerivation010
	}
	if _, e := binary010(b["nonce"], 16); e != nil {
		return nil, e
	}
	return b, nil
}
func domains010(b map[string]string) (Domains010, error) {
	raw, e := jcs.Marshal(b)
	if e != nil {
		return Domains010{}, errDerivation010
	}
	info := append([]byte("sage-hpke-info|0.10.0\n"), raw...)
	h := sha256.Sum256(info)
	return Domains010{raw, info, append([]byte("sage-hpke-export|0.10.0\n"), h[:]...)}, nil
}

// BuildDomains010 validates the closed B shape and derives info/exportCtx.
// Syntax does not prove registration or authority; web origin policy is external.
func BuildDomains010(raw []byte) (Domains010, error) {
	m, e := fields010(raw, bindingFields010)
	if e != nil {
		return Domains010{}, e
	}
	b, e := binding010(m)
	if e != nil {
		return Domains010{}, e
	}
	return domains010(b)
}
func initiation010(raw []byte) (map[string]string, Domains010, error) {
	m, e := fields010(raw, append(append([]string{}, bindingFields010...), "task", "enc", "ephC"))
	if e != nil {
		return nil, Domains010{}, e
	}
	b, e := binding010(m)
	if e != nil {
		return nil, Domains010{}, e
	}
	if m["task"] != "hpke/init@0.10.0" {
		return nil, Domains010{}, errDerivation010
	}
	for _, n := range []string{"enc", "ephC"} {
		if _, e = binary010(m[n], 32); e != nil {
			return nil, Domains010{}, e
		}
	}
	dom, e := domains010(b)
	return m, dom, e
}
func finish010(m map[string]string, exporter, private []byte, peerField string) (*Derivation010, error) {
	peer, e := binary010(m[peerField], 32)
	if e != nil {
		return nil, e
	}
	sk, e := ecdh.X25519().NewPrivateKey(private)
	if e != nil {
		return nil, errDerivation010
	}
	pk, e := ecdh.X25519().NewPublicKey(peer)
	if e != nil {
		return nil, errDerivation010
	}
	shared, e := sk.ECDH(pk)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(shared)
	t, e := jcs.Marshal(m)
	if e != nil {
		return nil, errDerivation010
	}
	th := sha256.Sum256(t)
	seed, e := CombineSecrets010(exporter, shared, th[:])
	if e != nil {
		return nil, errDerivation010
	}
	ack, e := MakeAckTag010(seed, th[:])
	if e != nil {
		zeroBytes(seed)
		return nil, errDerivation010
	}
	sid := sha256.Sum256(append([]byte("sage-session|0.10.0"), th[:]...))
	return &Derivation010{t, th[:], seed, ack, base64.RawURLEncoding.EncodeToString(sid[:16])}, nil
}

// DeriveResponder010 accepts a validated initiation and explicit private keys.
// The caller must supply a fresh independent E2E private key and UUIDv4 kid;
// RespondFresh010 provides those values automatically. Neither API authenticates.
func DeriveResponder010(raw, kemPrivate, e2ePrivate []byte, kid string) (*Derivation010, error) {
	m, dom, e := initiation010(raw)
	if e != nil {
		return nil, e
	}
	if !uuid010.MatchString(kid) {
		return nil, errDerivation010
	}
	sk, e := ecdh.X25519().NewPrivateKey(kemPrivate)
	if e != nil {
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
	exporter, e := keys.HPKEOpenSharedSecretWithX25519Priv(sk, enc, dom.Info, dom.ExportContext, 32)
	if e != nil {
		return nil, errDerivation010
	}
	defer zeroBytes(exporter)
	m["ephS"] = base64.RawURLEncoding.EncodeToString(es.PublicKey().Bytes())
	m["kid"] = kid
	return finish010(m, exporter, e2ePrivate, "ephC")
}

// RespondFresh010 draws a fresh E2E key and handle using the system CSPRNG.
func RespondFresh010(raw, kemPrivate []byte) (*Derivation010, error) {
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
	return DeriveResponder010(raw, kemPrivate, private, kid.String())
}

// Initiator010 retains cryptographic material for one derivation. It is not the
// authenticated HPKE pending-state machine. Do not copy; call Close on abandon.
type Initiator010 struct {
	mu                sync.Mutex
	init              map[string]string
	exporter, private []byte
	closed            bool
}

// StartInitiator010 generates independent HPKE and E2E ephemeral keys. The caller
// supplies fresh authenticated context/nonce and the exact current kemKid bytes.
func StartInitiator010(binding, kemPublic []byte) (*Initiator010, []byte, error) {
	m, e := fields010(binding, bindingFields010)
	if e != nil {
		return nil, nil, e
	}
	b, e := binding010(m)
	if e != nil {
		return nil, nil, e
	}
	dom, e := domains010(b)
	if e != nil {
		return nil, nil, e
	}
	pk, e := ecdh.X25519().NewPublicKey(kemPublic)
	if e != nil {
		return nil, nil, errDerivation010
	}
	enc, exporter, e := keys.HPKEDeriveSharedSecretToX25519Peer(pk, dom.Info, dom.ExportContext, 32)
	if e != nil {
		return nil, nil, errDerivation010
	}
	c, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		zeroBytes(exporter)
		return nil, nil, errDerivation010
	}
	m["task"] = "hpke/init@0.10.0"
	m["enc"] = base64.RawURLEncoding.EncodeToString(enc)
	m["ephC"] = base64.RawURLEncoding.EncodeToString(c.PublicKey().Bytes())
	raw, e := jcs.Marshal(m)
	if e != nil {
		zeroBytes(exporter)
		return nil, nil, errDerivation010
	}
	return &Initiator010{init: m, exporter: exporter, private: c.Bytes()}, raw, nil
}
func (s *Initiator010) destroy() {
	zeroBytes(s.exporter)
	zeroBytes(s.private)
	s.exporter = nil
	s.private = nil
	s.closed = true
}

// Close erases retained byte buffers and prevents another derivation.
func (s *Initiator010) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.destroy() }

// Derive validates every echoed initiation field and consumes the retained
// material on success or failure. Verify completion signatures and authority
// separately; this function does not accept a completion envelope or verify ACK.
func (s *Initiator010) Derive(transcript []byte) (*Derivation010, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errDerivation010
	}
	defer s.destroy()
	names := append(append([]string{}, bindingFields010...), "task", "enc", "ephC", "ephS", "kid")
	m, e := fields010(transcript, names)
	if e != nil {
		return nil, e
	}
	for k, v := range s.init {
		if m[k] != v {
			return nil, errDerivation010
		}
	}
	if !uuid010.MatchString(m["kid"]) {
		return nil, errDerivation010
	}
	return finish010(m, s.exporter, s.private, "ephS")
}
