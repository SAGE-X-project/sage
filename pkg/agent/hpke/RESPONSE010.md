# Correlated session responses

The authenticated session retains an independent copy of every emitted signed
request and the canonical complete envelope of each accepted request. Callers
cannot supply a request hash or an authentication flag. SealResponse/seal_response
requires the identifier of a request already accepted by this exact session;
OpenResponse/open_response looks up the internally retained sent envelope and
recomputes SHA256(JCS(complete signed request)), including its signature.

The message_id, request_hash, opposite roles, both participants, pinned signing
key, session and context must match. Response ID and nonce are fresh and cannot
equal the original request values. The complete response uses the response
signature domain and the existing session record encryption. Its caller AAD is
JCS of all response members except data and signature; status, error and request
correlation fields are therefore authenticated by both signature and AEAD.

There is one terminal emission and one terminal acceptance per request. A
successful store transaction and sequence acceptance precede marking the request
terminal while the endpoint ownership boundary remains held. Unsolicited or
second terminal responses are rejected. Invalid responses and failed storage do
not consume pending correlation or sequence state. Failed current-key validation,
expiry of the session, and final gate failures close the owner without plaintext
fallback. Application failure after cryptographic acceptance does not reopen the
request. Callers can retransmit the exact previously returned response bytes;
they cannot encrypt a new response to the same request through this API.

A successful result omits error. A failed result requires exactly one of
`authentication_failed`, `policy_denied`, `operation_failed`, `unavailable`.
The verified result exposes message ID, success, validated error and decrypted
data together; an authenticated error is still a terminal response. Empty
application data is encrypted into a 36-byte record, not sent as empty ciphertext.
Malformed status/error combinations, null, duplicate/unknown fields, plaintext
encoding and mismatched requests fail closed.

This remains the metadata-free profile with explicit context/role, 32 KiB JSON,
16384 record bytes and 16348 plaintext bytes. Both participants may issue requests
and correlated responses after confirmation. Optional task/metadata fields,
HTTP/TLS and WebSocket binding remain separate integrations. Request records are
retained until session retirement, bounded by the existing 1000-record cap per
direction, and are cleared on close; neither timeout nor restart restores old
requests or counters. Request envelope expiry controls request freshness, not a
promise of a response deadline. Applications may close earlier for local policy.

The existing trusted transactional replay-store and registry/clock contracts
continue to apply; test stores do not establish deployment durability or host
unavoidability. A shared 38-case fixture covers success and four error codes,
request correlation, identity/context, duplicate terminal results, malformed
schemas, storage and key failures, expiry, reverse-direction requests and
out-of-order responses. Additional units check independent retained copies,
unsolicited emission, admission bounds and concurrent terminal acceptance in Go.
Inspector runs 152 exchanges across four Go/Rust process combinations and verifies
response signatures and exact request hashes independently. Historical findings
and full NOT_ESTABLISHED conformance status are preserved.
