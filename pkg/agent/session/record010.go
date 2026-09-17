package session

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const record010Limit = 1000
const record010MaxAAD = 4033
const record010MaxWire = 8 * 1024 * 1024

// RecordSession010 owns the record keys and counters for SAGE 0.10.0.
// Construct only after an authenticated handshake. This low-level API does not
// validate envelope tuples, registry status, HPKE confirmation or authorization.
// Call Close when any of those external checks fail. Do not copy a session.
type RecordSession010 struct {
	mu        sync.Mutex
	seed      [32]byte
	th        [32]byte
	initiator bool
	sid       string
	next      uint64
	seen      [16]uint64 // All 1000 permitted sequence values fit inside the 1024-slot window.
	created   time.Time
	active    time.Time
	closed    bool
}

// NewRecordSession010 consumes the authenticated handshake seed and transcript
// hash. Reconstructing with the same inputs resets counters and is forbidden;
// restart/recovery must perform a fresh authenticated handshake.
func NewRecordSession010(seed, th []byte, initiator bool) (*RecordSession010, error) {
	if len(seed) != 32 || len(th) != 32 {
		return nil, errors.New("seed and transcript must be 32 bytes")
	}
	now := time.Now()
	s := &RecordSession010{initiator: initiator, created: now, active: now}
	copy(s.seed[:], seed)
	copy(s.th[:], th)
	h := sha256.New()
	_, _ = h.Write([]byte("sage-session|0.10.0"))
	_, _ = h.Write(th)
	s.sid = base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])
	return s, nil
}

// ID returns the public transcript-derived session identifier.
func (s *RecordSession010) ID() string   { return s.sid }
func (s *RecordSession010) closeLocked() { clear(s.seed[:]); s.closed = true }

// Close retires this session. No further record can be sent or accepted.
func (s *RecordSession010) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.closeLocked() }
func (s *RecordSession010) live() error {
	if s.closed || time.Since(s.created) >= time.Hour || time.Since(s.active) >= 10*time.Minute {
		s.closeLocked()
		return errors.New("record session is closed or expired")
	}
	return nil
}
func (s *RecordSession010) direction(sending bool) byte {
	if s.initiator == sending {
		return 0
	}
	return 1
}
func (s *RecordSession010) key(direction byte, seq uint64) ([32]byte, error) {
	var key [32]byte
	label := "c2s"
	if direction == 1 {
		label = "s2c"
	}
	info := append([]byte("sage-"+label+"-key|0.10.0"), s.th[:]...)
	info = binary.BigEndian.AppendUint64(info, seq/256)
	_, err := io.ReadFull(hkdf.Expand(sha256.New, s.seed[:], info), key[:])
	return key, err
}
func (s *RecordSession010) aad(direction byte, seq uint64, caller []byte) []byte {
	a := append([]byte("sage-record|0.10.0"), s.th[:]...)
	a = append(a, direction)
	a = binary.BigEndian.AppendUint64(a, seq)
	a = binary.BigEndian.AppendUint32(a, uint32(len(caller)))
	return append(a, caller...)
}
func record010Nonce(seq uint64) [12]byte {
	var n [12]byte
	binary.BigEndian.PutUint64(n[4:], seq)
	return n
}

// Seal allocates a sequence and returns the complete wire record. A transport
// retry must reuse the returned bytes, never call Seal to recreate that record.
func (s *RecordSession010) Seal(plaintext, callerAAD []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.live(); err != nil {
		return nil, err
	}
	if len(callerAAD) > record010MaxAAD || len(plaintext) > record010MaxWire-36 {
		return nil, errors.New("record size limit")
	}
	if s.next >= record010Limit {
		s.closeLocked()
		return nil, errors.New("record sequence limit")
	}
	seq := s.next
	s.next++
	key, err := s.key(s.direction(true), seq)
	if err != nil {
		return nil, err
	}
	defer clear(key[:])
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	nonce := record010Nonce(seq)
	wire := binary.BigEndian.AppendUint64(nil, seq)
	wire = append(wire, nonce[:]...)
	wire = aead.Seal(wire, nonce[:], plaintext, s.aad(s.direction(true), seq, callerAAD))
	if err := s.live(); err != nil {
		return nil, err
	}
	s.active = time.Now()
	return wire, nil
}

// Open authenticates and accepts a record atomically. Unseen out-of-order
// records are permitted; callers must separately enforce business ordering.
func (s *RecordSession010) Open(wire, callerAAD []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.live(); err != nil {
		return nil, err
	}
	if len(wire) < 36 || len(wire) > record010MaxWire || len(callerAAD) > record010MaxAAD {
		return nil, errors.New("record size limit")
	}
	seq := binary.BigEndian.Uint64(wire[:8])
	if seq >= record010Limit {
		return nil, errors.New("record sequence limit")
	}
	nonce := record010Nonce(seq)
	if string(wire[8:20]) != string(nonce[:]) {
		return nil, errors.New("invalid record nonce")
	}
	bit := uint64(1) << (seq % 64)
	if s.seen[seq/64]&bit != 0 {
		return nil, errors.New("duplicate record")
	}
	key, err := s.key(s.direction(false), seq)
	if err != nil {
		return nil, err
	}
	defer clear(key[:])
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce[:], wire[20:], s.aad(s.direction(false), seq, callerAAD))
	if err != nil {
		return nil, err
	}
	if err := s.live(); err != nil {
		clear(plaintext)
		return nil, err
	}
	s.seen[seq/64] |= bit
	s.active = time.Now()
	return plaintext, nil
}
