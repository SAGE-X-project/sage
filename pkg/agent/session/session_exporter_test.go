package session

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Sessions built from an HPKE exporter secret must carry usable legacy keys as
// well as the directional keys, so every Session method works on them.
var testCfg = Config{MaxAge: time.Minute, IdleTimeout: time.Minute, MaxMessages: 100}

func TestExporterSessions_LegacyMethodsUseDerivedKeys(t *testing.T) {
	exporter := make([]byte, 32)
	_, err := rand.Read(exporter)
	require.NoError(t, err)

	initiator, err := NewSecureSessionFromExporterWithRole("sid-1", exporter, true, testCfg)
	require.NoError(t, err)
	responder, err := NewSecureSessionFromExporterWithRole("sid-1", exporter, false, testCfg)
	require.NoError(t, err)

	require.NotEqual(t, make([]byte, 32), initiator.signingKey, "signing key must be derived, not zero")
	require.NotNil(t, initiator.aead, "legacy AEAD must be initialised")

	covered := []byte("covered-bytes")
	sig := initiator.SignCovered(covered)
	require.NoError(t, responder.VerifyCovered(covered, sig))
	require.Error(t, responder.VerifyCovered([]byte("tampered"), sig))

	ct, mac, err := initiator.EncryptAndSign([]byte("hello"), []byte("aad"))
	require.NoError(t, err)
	pt, err := responder.DecryptAndVerify(ct, []byte("aad"), mac)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt, []byte("hello")))

	// Directional path still works alongside the legacy path.
	out, err := initiator.EncryptOutbound([]byte("dir"))
	require.NoError(t, err)
	in, err := responder.DecryptInbound(out)
	require.NoError(t, err)
	require.Equal(t, []byte("dir"), in)
}
