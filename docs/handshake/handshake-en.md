# SAGE Handshake Package

This Go package from the SAGE (Secure Agent Guarantee Engine) project provides the pre-negotiation required for secure session communication.

## Key Features

It extends the existing [A2A protocol](https://a2a-protocol.org/latest/topics/what-is-a2a/#a2a-request-lifecycle) and performs the handshake over gRPC.

<img src="../assets/SAGE-handshake.png" width="450" height="550"/>

- **DID signature validation & bootstrap encryption**: Verifies the DID and Ed25519 signature included in metadata, and encrypts request/response payloads with the peer's DID public key to block man-in-the-middle attacks.
- **Ephemeral key agreement and session establishment**: Uses an X25519 ephemeral exchange to derive a shared secret, then derives signing and encryption keys for the session. Messages inside the session are encrypted and signed with these keys.
- **Session and nonce management**: Automatically handles creation, lookup, and expiration according to policy, checking per-request identifiers such as key IDs and nonces to prevent replay attacks. All related material is securely disposed of once the session ends.
- **Event-driven extensibility**: The `Events` interface exposes hooks such as OnInvitation/OnRequest/OnComplete, key ID issuance, and automated outbound responses.

**Four handshake phases**

The requesting agent already knows the peer's DID via [A2A Agent Discovery](https://a2a-protocol.org/latest/topics/agent-discovery/), and both DIDs are assumed to be registered on-chain. Each side resolves the DID Document to obtain the peer's public key for identity verification and bootstrap encryption.

1. **Invitation (agent A -> agent B)**:
   - Agent A sends an intent to establish a session along with its DID.
   - Agent B resolves A's DID to obtain the public key and verifies the signature to confirm the request is authentic.
2. **Request (agent A -> agent B)**:
   - Agent A generates an X25519 ephemeral public key and sends it to agent B. The payload is encrypted with B's DID public key and signed with A's Ed25519 identity key.
   - Agent B verifies the signature, decrypts the payload, and stores A's ephemeral public key.
   - Because the payload is encrypted, only the intended peer with the correct keys can read it.
3. **Response (agent B -> agent A)**:
   - Agent B generates an X25519 ephemeral public key and sends it back to agent A. The payload is encrypted with A's DID public key and signed with B's Ed25519 identity key.
   - Agent A verifies the signature, decrypts the payload, and stores B's ephemeral public key.
   - As with the request, only the peer holding the correct keys can read the encrypted data.
4. **Complete (agent A -> agent B)**:
   - After both sides hold the shared secret, agent A sends the complete message.
   - Each side derives a pseudorandom seed from the shared secret and uses it to compute a session ID, ensuring both agents instantiate the same session.
   - The session is bound to a random string key identifier (`kid`). Agent B returns the `kid` in the complete response, and agent A binds the received `kid` to its session. The `kid` later becomes the `keyId` field in HTTP Message Signatures (RFC 9421), allowing either agent to look up the session during signature verification.

## Installation

```bash
go get github.com/sage-x-project/sage/handshake
```

## Architecture

```bash
├── client.go           # requesting agent implementation
├── server.go           # responding agent implementation
├── session             # session and nonce management
│   ├── manager.go      # creates and tears down sessions
│   ├── metadata.go     # tracks session state and expiration
│   ├── nonce.go        # nonce cache per session
│   ├── session.go      # session key derivation and crypto
│   └── types.go        # session interfaces
├── types.go            # handshake interfaces
└── utils.go
```

### Session management components

- `handshake/session/manager.go`: Manages session creation, lookup, and expiration, and runs a periodic cleanup loop in the background.
- `handshake/session/nonce.go`: Stores nonces per session with TTL semantics to detect replayed requests.
- `handshake/session/session.go`: Uses HKDF-derived keys to drive ChaCha20-Poly1305 and HMAC-SHA256 operations and includes logic to securely wipe key material.
- `handshake/session/metadata.go`: Produces session metadata IDs, creation/expiration timestamps, and states that can feed external audit or observability systems.

## Build

**Build the CLI tool**

```bash
# Run from the project root
go build -o sage-crypto ./cmd/sage-crypto

# Or install directly
go install ./cmd/sage-crypto
```

## Usage

**Requesting agent**

```go
package main

import (
   "encoding/json"
   "fmt"

   "github.com/sage-x-project/sage/core/message"
   "github.com/sage-x-project/sage/crypto"
   "github.com/sage-x-project/sage/crypto/formats"
   "github.com/sage-x-project/sage/handshake"
)

// Create the requesting agent
agentA := handshake.NewClient(conn, clientKeypair)

// Invitation
inv := handshake.InvitationMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
}
if _, err := agentA.Invitation(ctx, inv, string(myDID)); err != nil {
   panic(err)
}

// Request
eph := mustX25519()
jwk := must(formats.NewJWKExporter().ExportPublic(eph, crypto.KeyFormatJWK))

reqMsg := handshake.RequestMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
   EphemeralPubKey: json.RawMessage(jwk),
}
if _, err := agentA.Request(ctx, reqMsg, serverPub, string(myDID)); err != nil {
   panic(err)
}

// Complete
comMsg := handshake.CompleteMessage{
   BaseMessage: message.BaseMessage{
      ContextID: ctxID,
   },
}
if _, err := agentA.Complete(ctx, comMsg, string(myDID)); err != nil {
   panic(err)
}
```

## Security considerations

1. **Ephemeral key management**: `Events.AskEphemeral` returns a 32-byte X25519 public key (raw) and its JWK representation, but the private key remains owned by the application. Store private keys securely in the event implementation and rotate them per session without reuse.
2. **Bootstrap encryption**: During the request/response phases, `keys.EncryptWithEd25519Peer` encrypts payloads with the peer's DID public key. Regularly ensure the peer's DID Document is up-to-date and reflects rotated identity keys.
3. **Session key disposal**: `session.SecureSession.Close()` zeroes the AEAD key, HMAC key, and HKDF seed. Always call `Manager.RemoveSession` or `Close` when a session expires so key material does not linger in memory.
4. **Nonce reuse prevention**: `session.NonceCache` tracks `kid`-`nonce` combinations with a TTL. Populate the nonce field in HTTP Message Signatures and check the `Seen` result for each message to block replay attempts.
5. **Cleanup of incomplete contexts**: When the server receives a Request it stores the peer's ephemeral key in the `pending` map. If Complete never arrives, `cleanupLoop` removes expired contexts—tune the TTL and cleanup cadence to match your service policy and monitor the metrics.

## Error handling

### Common errors

**Missing DID**

```bash
missing did
```

- The DID field is absent from metadata. Always provide the DID when calling `signStruct`, and confirm that proxies do not strip gRPC metadata.

**Signature verification failed**

```bash
signature verification failed
```

- The public key in the DID Document does not match the signing key, or the message was tampered with. Verify your DID resolution pipeline and time synchronization, and use the same TaskID and ContextID for invitation and request messages.

**Decryption failed**

```bash
request decrypt: ... / response decrypt: ...
```

- Bootstrap decryption failed. Check that the peer's DID public key is current, the Base64URL encoding is intact, and both sides use the Ed25519 key format.

**Session key creation failed**

```bash
ask ephemeral: ...
```

- The event layer could not generate a new ephemeral key. Ensure the key management service is reachable and design fallback or retry paths for failures.

**Session expired**

```bash
session expired
```

- `SecureSession` violated the `MaxAge`, `IdleTimeout`, or `MaxMessages` policies. Adjust the session configuration to match your traffic patterns or trigger a fresh handshake.

## Policies and observability

- **Session policies**: Configure `session.Config` with MaxAge (absolute expiry), IdleTimeout (idle expiry), and MaxMessages (allowed message count) to control long-lived or bursty connections.
- **Logging and audit**: Use the `handshake.Events` OnInvitation/OnRequest/OnComplete callbacks to log DID verification results, ephemeral key metadata, and session parameters for audit trails.
- **Monitoring metrics**: Track session creation/expiration rates, nonce reuse detections, and signature verification failures to aid security incident detection and performance tuning.

## Advanced features

- **Automatic KeyID issuance**: If your events implementation also satisfies `KeyIDBinder`, the server calls `IssueKeyID` after Complete and immediately includes the `kid` in its response. You can then wire the `kid` into the `keyId` field of HTTP Message Signatures to simplify verification.
- **Outbound response flow**: Inject an outbound gRPC client into `NewServer` to push Responses via `sendResponseToPeer` immediately after receiving a Request. This is helpful when the peer sits behind NAT or requires asynchronous negotiation.
- **Simplified session derivation**: `session.Manager.EnsureSessionWithParams` derives identical session IDs and keys on both sides using only the shared secret and context information, preventing duplicate sessions and reducing race conditions.
- **Replay window control**: Tune `IdleTimeout`/`MaxMessages` in `session.Config` to match business traffic, and keep the `NonceCache` TTL shorter than message lifetimes to precisely control retries or flood attempts.
- **Metadata and audit integration**: Record DID verification results and session parameters in `Events.OnRequest` and `OnComplete` callbacks to pipe them into audit logs, SIEM systems, or policy engines.
