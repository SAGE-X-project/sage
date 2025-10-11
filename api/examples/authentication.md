# SAGE Authentication Example

This example demonstrates the HPKE-based authentication and session establishment flow in SAGE.

## Overview

SAGE uses Hybrid Public Key Encryption (HPKE) for secure agent-to-agent authentication. The flow involves:

1. Client obtains server's KEM public key
2. Client initiates HPKE handshake with encrypted credentials
3. Server responds with session ID and encrypted response
4. Subsequent requests use the established session

---

## Prerequisites

```bash
# Generate client credentials
./sage-crypto generate -o alice

# Start SAGE server
go run tests/session/handshake/server/main.go
```

---

## Step 1: Get Server's KEM Public Key

**Request:**
```bash
curl -X GET http://localhost:8080/debug/kem-pub
```

**Response:**
```json
{
  "kem_public_key": "j8tTZ3xQ9K2nL4mP5rS6vW7xY8zA1bC2dE3fG4hI5jK="
}
```

**Note:** The KEM public key is a base64-encoded X25519 public key (32 bytes).

---

## Step 2: Get Server DID

**Request:**
```bash
curl -X GET http://localhost:8080/debug/server-did
```

**Response:**
```json
{
  "did": "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
}
```

---

## Step 3: Register Client Agent (Development Only)

In production, agents are resolved from blockchain. For development, manually register:

**Request:**
```bash
curl -X POST http://localhost:8080/debug/register-agent \
  -H "Content-Type: application/json" \
  -d '{
    "did": "did:sage:ethereum:0x123...",
    "name": "Alice Agent",
    "is_active": true,
    "public_key": "Y29ycmVjdF9lZDI1NTE5X3B1YmxpY19rZXk=",
    "public_kem_key": "Y29ycmVjdF94MjU1MTlfcHVibGljX2tleQ=="
  }'
```

**Response:**
```json
{
  "message": "agent registered"
}
```

**Key Fields:**
- `public_key`: Ed25519 public key for signature verification (32 bytes, base64)
- `public_kem_key`: X25519 public key for HPKE encryption (32 bytes, base64)

---

## Step 4: Initiate HPKE Handshake

**Client Code (Conceptual):**
```go
package main

import (
    "crypto/ed25519"
    "encoding/base64"
    "time"

    "github.com/sage-x-project/sage/pkg/crypto/hpke"
)

func initiateHandshake(serverKemPubKey []byte, clientPrivKey ed25519.PrivateKey) {
    // Create HPKE context
    hpkeCtx, encapsulatedKey, err := hpke.SetupSender(serverKemPubKey)
    if err != nil {
        panic(err)
    }

    // Create handshake message
    handshake := map[string]interface{}{
        "type": "handshake",
        "client_did": "did:sage:ethereum:0x123...",
        "timestamp": time.Now().Unix(),
    }

    plaintext := marshalJSON(handshake)

    // Encrypt with HPKE
    ciphertext, err := hpkeCtx.Seal(plaintext, nil)
    if err != nil {
        panic(err)
    }

    // Combine encapsulated key + ciphertext
    message := append(encapsulatedKey, ciphertext...)
    messageB64 := base64.StdEncoding.EncodeToString(message)

    // Sign the message
    timestamp := time.Now().Unix()
    toSign := fmt.Sprintf("%s|%s|%s|%d",
        "did:sage:ethereum:0x123...",
        "did:sage:ethereum:0x456...",
        messageB64,
        timestamp,
    )
    signature := ed25519.Sign(clientPrivKey, []byte(toSign))
    signatureB64 := base64.StdEncoding.EncodeToString(signature)

    // Send request
    sendA2AMessage(messageB64, signatureB64, timestamp)
}
```

**HTTP Request:**
```bash
curl -X POST http://localhost:8080/v1/a2a:sendMessage \
  -H "Content-Type: application/json" \
  -d '{
    "sender_did": "did:sage:ethereum:0x123...",
    "receiver_did": "did:sage:ethereum:0x456...",
    "message": "AgECA...encrypted_handshake_data...",
    "timestamp": 1234567890,
    "signature": "bXlfc2lnbmF0dXJlX2RhdGE="
  }'
```

**Response:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "response": "AgECA...encrypted_response..."
}
```

**Response Fields:**
- `session_id`: UUID for the established session (use in subsequent requests)
- `response`: Base64-encoded encrypted response containing session confirmation

---

## Step 5: Decrypt Response

**Client Code:**
```go
func decryptResponse(response string, hpkeCtx *hpke.Context) {
    responseBytes, _ := base64.StdEncoding.DecodeString(response)

    plaintext, err := hpkeCtx.Open(responseBytes, nil)
    if err != nil {
        panic(err)
    }

    var sessionInfo map[string]interface{}
    json.Unmarshal(plaintext, &sessionInfo)

    fmt.Printf("Session established: %v\n", sessionInfo)
}
```

**Decrypted Response:**
```json
{
  "status": "session_established",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": 1234571490
}
```

---

## Step 6: Send Authenticated Message

Once session is established, subsequent messages use the session context:

**Request:**
```bash
curl -X POST http://localhost:8080/v1/a2a:sendMessage \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "sender_did": "did:sage:ethereum:0x123...",
    "receiver_did": "did:sage:ethereum:0x456...",
    "message": "AgECA...encrypted_message...",
    "timestamp": 1234567900,
    "signature": "c2lnbmF0dXJlX2RhdGE="
  }'
```

---

## Security Notes

### Signature Verification

The signature is computed over:
```
sender_did|receiver_did|message|timestamp
```

Using Ed25519 signature scheme with the sender's private key.

### Replay Attack Prevention

- Each message must have a unique timestamp
- Server tracks used nonces (derived from message signatures)
- Nonces expire after configured TTL (default: 5 minutes)
- Server rejects messages with timestamps outside clock skew window (default: ±5 minutes)

### Session Management

- Sessions expire after max age (default: 1 hour)
- Sessions have idle timeout (default: 10 minutes)
- Server periodically cleans up expired sessions (default: 30 seconds interval)

---

## Error Responses

### Invalid Signature
```json
{
  "error": "signature verification failed",
  "code": "INVALID_SIGNATURE"
}
```

### Replay Attack Detected
```json
{
  "error": "nonce already used",
  "code": "REPLAY_ATTACK"
}
```

### Session Expired
```json
{
  "error": "session not found or expired",
  "code": "SESSION_EXPIRED"
}
```

### Clock Skew
```json
{
  "error": "timestamp outside acceptable range",
  "code": "CLOCK_SKEW",
  "details": {
    "server_time": 1234567890,
    "client_time": 1234567000,
    "max_skew_seconds": 300
  }
}
```

---

## Complete Example (Go)

See `tests/session/handshake/client/main.go` for a complete working example of the HPKE authentication flow.

**Run the example:**
```bash
# Terminal 1: Start server
go run tests/session/handshake/server/main.go

# Terminal 2: Run client
go run tests/session/handshake/client/main.go
```

---

## References

- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 8032: Ed25519](https://www.rfc-editor.org/rfc/rfc8032.html)
- [SAGE Cryptography](../../pkg/crypto/)
- [SAGE Session Management](../../pkg/session/)
