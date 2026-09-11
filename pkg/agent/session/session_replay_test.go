package session

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

// pair returns an initiator/responder session pair derived from one exporter secret.
func pair(t *testing.T, cfg Config) (*SecureSession, *SecureSession) {
	t.Helper()
	exporter := make([]byte, 32)
	_, err := rand.Read(exporter)
	require.NoError(t, err)
	a, err := NewSecureSessionFromExporterWithRole("sid-replay", exporter, true, cfg)
	require.NoError(t, err)
	b, err := NewSecureSessionFromExporterWithRole("sid-replay", exporter, false, cfg)
	require.NoError(t, err)
	return a, b
}

func TestSession_WireFormatCarriesSequence(t *testing.T) {
	a, b := pair(t, Config{})
	for i := 0; i < 3; i++ {
		ct, err := a.Encrypt([]byte("m"))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(ct), HeaderSize+chacha20poly1305.Overhead)
		assert.Equal(t, uint64(i), binary.BigEndian.Uint64(ct[:SeqSize]))
		pt, err := b.Decrypt(ct)
		require.NoError(t, err)
		assert.Equal(t, []byte("m"), pt)
	}
	assert.Equal(t, uint64(3), a.SendSequence())
}

func TestSession_RejectsReplayedCiphertext(t *testing.T) {
	a, b := pair(t, Config{})
	ct, err := a.Encrypt([]byte("once"))
	require.NoError(t, err)
	_, err = b.Decrypt(ct)
	require.NoError(t, err)
	_, err = b.Decrypt(ct)
	require.ErrorIs(t, err, ErrReplayedMessage)

	// Legacy single-key sessions enforce the window too.
	s, err := NewSecureSession("legacy", []byte("seed-seed-seed-seed-seed-seed-32"), Config{})
	require.NoError(t, err)
	ct, mac, err := s.EncryptAndSign([]byte("once"), []byte("covered"))
	require.NoError(t, err)
	_, err = s.DecryptAndVerify(ct, []byte("covered"), mac)
	require.NoError(t, err)
	_, err = s.DecryptAndVerify(ct, []byte("covered"), mac)
	require.ErrorIs(t, err, ErrReplayedMessage)
}

func TestSession_OutOfOrderWithinWindowAccepted(t *testing.T) {
	a, b := pair(t, Config{})
	var cts [][]byte
	for i := 0; i < 10; i++ {
		ct, err := a.Encrypt([]byte{byte(i)})
		require.NoError(t, err)
		cts = append(cts, ct)
	}
	for _, i := range []int{9, 3, 0, 7, 1} {
		pt, err := b.Decrypt(cts[i])
		require.NoError(t, err, "seq %d", i)
		assert.Equal(t, []byte{byte(i)}, pt)
	}
	_, err := b.Decrypt(cts[3])
	require.ErrorIs(t, err, ErrReplayedMessage)
}

func TestSession_StaleBeyondWindowRejected(t *testing.T) {
	a, b := pair(t, Config{MaxMessages: 10000})
	first, err := a.Encrypt([]byte("first"))
	require.NoError(t, err)
	var last []byte
	for i := 0; i < ReplayWindowSize; i++ {
		last, err = a.Encrypt([]byte("x"))
		require.NoError(t, err)
	}
	_, err = b.Decrypt(last)
	require.NoError(t, err)
	_, err = b.Decrypt(first)
	require.ErrorIs(t, err, ErrStaleMessage)
}

func TestSession_TamperedSequenceFailsAuthentication(t *testing.T) {
	a, b := pair(t, Config{})
	ct, err := a.Encrypt([]byte("m"))
	require.NoError(t, err)
	forged := append([]byte(nil), ct...)
	binary.BigEndian.PutUint64(forged[:SeqSize], 5)
	_, err = b.Decrypt(forged)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrReplayedMessage))

	// The forgery must not have advanced the window: the genuine message still decrypts.
	_, err = b.Decrypt(ct)
	require.NoError(t, err)
}

func TestSession_ForgedHighSequenceDoesNotPoisonWindow(t *testing.T) {
	a, b := pair(t, Config{})
	garbage := make([]byte, HeaderSize+32)
	binary.BigEndian.PutUint64(garbage[:SeqSize], 1<<40)
	_, err := b.Decrypt(garbage)
	require.Error(t, err)
	for i := 0; i < 3; i++ {
		ct, err := a.Encrypt([]byte("ok"))
		require.NoError(t, err)
		_, err = b.Decrypt(ct)
		require.NoError(t, err)
	}
}

func TestSession_ShortDataRejected(t *testing.T) {
	_, b := pair(t, Config{})
	_, err := b.Decrypt(make([]byte, HeaderSize-1))
	require.ErrorIs(t, err, ErrDataTooShort)
}

func TestSession_RekeyEveryInterval(t *testing.T) {
	const interval = 4
	a, b := pair(t, Config{RekeyInterval: interval, MaxMessages: 1000})
	var cts [][]byte
	for i := 0; i < 3*interval; i++ {
		ct, err := a.Encrypt([]byte{byte(i)})
		require.NoError(t, err)
		cts = append(cts, ct)
	}
	// Every generation decrypts on the peer.
	for i, ct := range cts {
		pt, err := b.Decrypt(ct)
		require.NoError(t, err, "seq %d", i)
		assert.Equal(t, []byte{byte(i)}, pt)
	}
	// A generation-1 ciphertext does not open under the generation-0 key.
	ct := cts[interval]
	_, err := b.aeadIn.Open(nil, ct[SeqSize:HeaderSize], ct[HeaderSize:], ct[:SeqSize])
	require.Error(t, err)
	// ...but a generation-0 ciphertext does.
	ct = cts[0]
	_, err = b.aeadIn.Open(nil, ct[SeqSize:HeaderSize], ct[HeaderSize:], ct[:SeqSize])
	require.NoError(t, err)
	// Only the current and previous generation are cached per direction.
	b.genMu.Lock()
	assert.LessOrEqual(t, len(b.genAEAD), 2)
	b.genMu.Unlock()

	// The reverse direction rotates independently with its own key schedule.
	for i := 0; i < 2*interval; i++ {
		ct, err := b.Encrypt([]byte("r"))
		require.NoError(t, err)
		_, err = a.Decrypt(ct)
		require.NoError(t, err)
	}
}

func TestSession_ConcurrentReplayAcceptedOnce(t *testing.T) {
	a, b := pair(t, Config{})
	ct, err := a.Encrypt([]byte("once"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := b.Decrypt(ct); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, ok)
}

func TestSession_AADIsBoundWithSequence(t *testing.T) {
	a, b := pair(t, Config{})
	ct, err := a.EncryptWithAAD([]byte("m"), []byte("request-1"))
	require.NoError(t, err)
	_, err = b.DecryptWithAAD(ct, []byte("request-2"))
	require.Error(t, err)
	_, err = b.DecryptWithAAD(ct, []byte("request-1"))
	require.NoError(t, err)
}

func TestReplayWindow(t *testing.T) {
	var w replayWindow
	require.NoError(t, w.check(10))
	w.mark(10)
	require.ErrorIs(t, w.check(10), ErrReplayedMessage)
	require.NoError(t, w.check(9))
	w.mark(9)
	require.ErrorIs(t, w.check(9), ErrReplayedMessage)
	// jump far ahead: everything old is stale, bitmap is cleared
	require.NoError(t, w.check(10+ReplayWindowSize*3))
	w.mark(10 + ReplayWindowSize*3)
	require.ErrorIs(t, w.check(10), ErrStaleMessage)
	require.NoError(t, w.check(10+ReplayWindowSize*3-1))
	// small advance clears the slots it passes over
	w.mark(10 + ReplayWindowSize*3 + 5)
	require.NoError(t, w.check(10+ReplayWindowSize*3+3))
	require.ErrorIs(t, w.check(10+ReplayWindowSize*3), ErrReplayedMessage)
}
