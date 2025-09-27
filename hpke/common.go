package hpke

import (
	"crypto"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	a2a "github.com/a2aproject/a2a/grpc"
	sagecrypto "github.com/sage-x-project/sage/crypto"
)

const TaskHPKEComplete = "hpke/complete@v1"

type InfoBuilder interface {
	BuildInfo(ctxID, initDID, respDID string) []byte
	BuildExportContext(ctxID string) []byte
}

type DefaultInfoBuilder struct{}

func (DefaultInfoBuilder) BuildInfo(ctxID, initDID, respDID string) []byte {
	return []byte("sage/hpke v1|ctx=" + ctxID + "|init=" + initDID + "|resp=" + respDID)
}
func (DefaultInfoBuilder) BuildExportContext(ctxID string) []byte {
	return []byte("exporter:" + ctxID)
}

// firstDataPart extracts the first DataPart struct payload.
func firstDataPart(m *a2a.Message) (*structpb.Struct, error) {
	if m == nil || len(m.Content) == 0 {
		return nil, errors.New("empty content")
	}
	dpart, ok := m.Content[0].GetPart().(*a2a.Part_Data)
	if !ok || dpart.Data == nil || dpart.Data.Data == nil {
		return nil, errors.New("missing data part")
	}
	return dpart.Data.Data, nil
}

func signStruct(k sagecrypto.KeyPair, msg []byte, did string) (*structpb.Struct, error) {
	sig, err := k.Sign(msg)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(map[string]any{
		"signature": base64.RawURLEncoding.EncodeToString(sig),
		"did":       did,
	})
}

// verifySenderSignature checks metadata.signature against deterministic-marshaled message bytes.
func verifySenderSignature(m *a2a.Message, meta *structpb.Struct, senderPub crypto.PublicKey) error {
	field := meta.GetFields()["signature"]
	if field == nil {
		return errors.New("missing signature")
	}
	sig, err := base64.RawURLEncoding.DecodeString(field.GetStringValue())
	if err != nil {
		return fmt.Errorf("bad signature b64: %w", err)
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		return err
	}
	// Support either a custom Verify interface or raw ed25519.PublicKey
	type verifyKey interface {
		Verify(msg, sig []byte) error
	}

	switch pk := senderPub.(type) {
	case verifyKey:
		// Your key type implements Verify([]byte, []byte) error
		if err := pk.Verify(bytes, sig); err != nil {
			return fmt.Errorf("signature verify failed: %w", err)
		}
		return nil
	case ed25519.PublicKey:
		// Standard ed25519
		if !ed25519.Verify(pk, bytes, sig) {
			return errors.New("signature verify failed: invalid ed25519 signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type: %T", senderPub)
	}
}

// ===== Nonce cache (replay protection) =====
type nonceStore struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]time.Time
}

func newNonceStore(ttl time.Duration) *nonceStore {
	return &nonceStore{ttl: ttl, entries: make(map[string]time.Time)}
}
func (s *nonceStore) checkAndMark(key string) bool {
	now := time.Now()
	exp := now.Add(s.ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.entries {
		if now.After(v) {
			delete(s.entries, k)
		}
	}
	if _, ok := s.entries[key]; ok {
		return false
	}
	s.entries[key] = exp
	return true
}

// ===== structpb helpers =====
func toStruct(m map[string]any) *structpb.Struct { st, _ := structpb.NewStruct(m); return st }
func getString(st *structpb.Struct, key string) (string, error) {
	v := st.GetFields()[key]
	if v == nil {
		return "", fmt.Errorf("missing %s", key)
	}
	return v.GetStringValue(), nil
}
func getBase64(st *structpb.Struct, key string) ([]byte, error) {
	s, err := getString(st, key)
	if err != nil {
		return nil, err
	}
	return base64.RawURLEncoding.DecodeString(s)
}
func putBase64(m map[string]any, key string, b []byte) {
	m[key] = base64.RawURLEncoding.EncodeToString(b)
}

// ===== ACK (HMAC) -- key confirmation without ciphertext =====
func hkdfExpand(key []byte, info string, outLen int) []byte {
	h := hmac.New(sha256.New, key)
	var out []byte
	var counter uint32 = 1
	for len(out) < outLen {
		h.Reset()
		h.Write([]byte(info))
		var c [4]byte
		binary.BigEndian.PutUint32(c[:], counter)
		h.Write(c[:])
		out = append(out, h.Sum(nil)...)
		counter++
	}
	return out[:outLen]
}

// ackKey = HKDF(exporter, "ack-key"), ackTag = HMAC(ackKey, "hpke-ack|"+ctxID+"|"+nonce+"|"+kid)
func makeAckTag(exporter []byte, ctxID, nonce, kid string) []byte {
	ackKey := hkdfExpand(exporter, "ack-key", 32)
	mac := hmac.New(sha256.New, ackKey)
	mac.Write([]byte("hpke-ack|"))
	mac.Write([]byte(ctxID))
	mac.Write([]byte("|"))
	mac.Write([]byte(nonce))
	mac.Write([]byte("|"))
	mac.Write([]byte(kid))
	return mac.Sum(nil)
}

// HPKEInitPayload
type HPKEInitPayload struct {
	InitDID   string
	RespDID   string
	Info      []byte
	ExportCtx []byte
	Enc       []byte // HPKE enc (sender ephemeral KEM pub) - raw bytes
	Nonce     string
	Timestamp time.Time
}

// ParseHPKEInitPayload parses the required fields from a2a DataPart Struct.
// Expects base64url-encoded "enc". All other fields are strings.
// Returns strongly-typed payload with timestamp parsed.
func ParseHPKEInitPayload(st *structpb.Struct) (HPKEInitPayload, error) {
	var out HPKEInitPayload

	var err error
	if out.InitDID, err = getString(st, "initDid"); err != nil {
		return HPKEInitPayload{}, err
	}
	if out.RespDID, err = getString(st, "respDid"); err != nil {
		return HPKEInitPayload{}, err
	}
	var infoStr string
	if infoStr, err = getString(st, "info"); err != nil {
		return HPKEInitPayload{}, err
	}
	out.Info = []byte(infoStr)

	var exportCtxStr string
	if exportCtxStr, err = getString(st, "exportCtx"); err != nil {
		return HPKEInitPayload{}, err
	}
	out.ExportCtx = []byte(exportCtxStr)

	if out.Enc, err = getBase64(st, "enc"); err != nil {
		return HPKEInitPayload{}, err
	}
	if out.Nonce, err = getString(st, "nonce"); err != nil {
		return HPKEInitPayload{}, err
	}
	tsStr, err := getString(st, "ts")
	if err != nil {
		return HPKEInitPayload{}, err
	}
	out.Timestamp, err = time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return HPKEInitPayload{}, fmt.Errorf("bad ts: %w", err)
	}
	return out, nil
}
