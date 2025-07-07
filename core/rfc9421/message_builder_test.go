package rfc9421

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageBuilder(t *testing.T) {
	t.Run("Build complete message", func(t *testing.T) {
		now := time.Now()
		
		message := NewMessageBuilder().
			WithAgentDID("did:sage:ethereum:agent001").
			WithMessageID("msg-001").
			WithTimestamp(now).
			WithNonce("nonce123").
			WithBody([]byte("test body")).
			AddHeader("Content-Type", "application/json").
			AddHeader("X-Custom", "value").
			AddMetadata("version", "1.0").
			AddMetadata("feature", "test").
			WithAlgorithm(AlgorithmEdDSA).
			WithKeyID("key-001").
			WithSignedFields("agent_did", "message_id", "body").
			WithSignature([]byte("signature")).
			Build()
		
		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, "msg-001", message.MessageID)
		assert.Equal(t, now, message.Timestamp)
		assert.Equal(t, "nonce123", message.Nonce)
		assert.Equal(t, []byte("test body"), message.Body)
		assert.Equal(t, "application/json", message.Headers["Content-Type"])
		assert.Equal(t, "value", message.Headers["X-Custom"])
		assert.Equal(t, "1.0", message.Metadata["version"])
		assert.Equal(t, "test", message.Metadata["feature"])
		assert.Equal(t, string(AlgorithmEdDSA), message.Algorithm)
		assert.Equal(t, "key-001", message.KeyID)
		assert.Equal(t, []string{"agent_did", "message_id", "body"}, message.SignedFields)
		assert.Equal(t, []byte("signature"), message.Signature)
	})
	
	t.Run("Build with default signed fields", func(t *testing.T) {
		message := NewMessageBuilder().
			WithAgentDID("did:sage:ethereum:agent001").
			Build()
		
		assert.Equal(t, []string{"agent_did", "message_id", "timestamp", "nonce", "body"}, message.SignedFields)
	})
	
	t.Run("Build minimal message", func(t *testing.T) {
		message := NewMessageBuilder().Build()
		
		assert.NotNil(t, message)
		assert.NotNil(t, message.Headers)
		assert.NotNil(t, message.Metadata)
		assert.NotZero(t, message.Timestamp)
		assert.NotEmpty(t, message.SignedFields)
	})
}

func TestParseMessageFromHeaders(t *testing.T) {
	t.Run("Parse complete headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Agent-DID":           "did:sage:ethereum:agent001",
			"X-Message-ID":          "msg-001",
			"X-Timestamp":           "2024-01-01T12:00:00Z",
			"X-Nonce":               "nonce123",
			"X-Signature-Algorithm": "EdDSA",
			"X-Key-ID":              "key-001",
			"X-Signed-Fields":       "agent_did, message_id, body",
			"Content-Type":          "application/json",
		}
		
		body := []byte("test body")
		
		message, err := ParseMessageFromHeaders(headers, body)
		require.NoError(t, err)
		
		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, "msg-001", message.MessageID)
		assert.Equal(t, "2024-01-01T12:00:00Z", message.Timestamp.Format(time.RFC3339))
		assert.Equal(t, "nonce123", message.Nonce)
		assert.Equal(t, string(AlgorithmEdDSA), message.Algorithm)
		assert.Equal(t, "key-001", message.KeyID)
		assert.Equal(t, []string{"agent_did", "message_id", "body"}, message.SignedFields)
		assert.Equal(t, body, message.Body)
		assert.Equal(t, "application/json", message.Headers["Content-Type"])
	})
	
	t.Run("Parse minimal headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Agent-DID": "did:sage:ethereum:agent001",
		}
		
		body := []byte("test body")
		
		message, err := ParseMessageFromHeaders(headers, body)
		require.NoError(t, err)
		
		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, body, message.Body)
		assert.NotNil(t, message.Headers)
		assert.NotNil(t, message.Metadata)
	})
	
	t.Run("Parse with invalid timestamp", func(t *testing.T) {
		headers := map[string]string{
			"X-Timestamp": "invalid-timestamp",
		}
		
		message, err := ParseMessageFromHeaders(headers, nil)
		require.NoError(t, err)
		
		// Should use default timestamp when parsing fails
		assert.NotZero(t, message.Timestamp)
	})
}

func TestSignatureAlgorithmConstants(t *testing.T) {
	assert.Equal(t, SignatureAlgorithm("EdDSA"), AlgorithmEdDSA)
	assert.Equal(t, SignatureAlgorithm("ES256K"), AlgorithmES256K)
	assert.Equal(t, SignatureAlgorithm("ECDSA"), AlgorithmECDSA)
	assert.Equal(t, SignatureAlgorithm("ECDSA-secp256k1"), AlgorithmECDSASecp256k1)
}