# Authenticated HTTP application metadata

`AuthenticatedCompletion010.TakeHTTPApplication(target)` transfers an unused
session to `HTTPApplicationOwner010`. It implements the existing SAGE 0.10.0
signed envelope `metadata` field for a trusted application adapter. It does not
change the wire version, create a network client or authorize application effects.
The target is exact trusted configuration; callers must not derive it from a
received document, alias or metadata value.

Acquire the owner after a successful ordinary or HTTP-bound handshake and before
any record. A provisional responder can acquire it and receive its first request.
An existing HTTP binding must match the target. Failure leaves ownership with the
original handle. A successful transfer invalidates every old session value copy,
including raw/HTTP/non-HTTP methods and rebinding. Closing an old handle does not
erase the new owner's keys. Copies of the owner share sequence, replay, terminal
and lifetime state; endpoint retirement or owner closure stops all those copies.

Use `SealRequest(ctx, payload, metadata, ttl)` and `OpenRequest(ctx, message)` for
requests; use `SealResponse` and `OpenResponse` for the exactly correlated terminal
response. The returned request and response views contain owned payload, metadata,
authenticated identities, key URL, context/session and message identifiers.
`Payload`, `Data` and `Metadata` return copies. A response's `Success` reports its
transport result, never local execution permission. Response metadata may differ
from request metadata. No view is released before tuple, current configured
Registry, envelope/HTTP signatures, AEAD and transactional replay acceptance.

Metadata is cloned before provider calls. An absent map stays absent; an empty
map is a present empty object. Keys match `[a-z][a-z0-9_]{0,31}`, at most 32 unique
entries. Values are valid UTF-8 strings of at most 1024 bytes each. Complete
metadata JCS is at most 4096 bytes. Explicit null, duplicate decoded keys,
nonstring values, invalid UTF-8 and unpaired UTF-16 escapes are rejected. Unknown
envelope fields remain closed. The complete envelope signature, record AAD and
retained complete-request hash bind metadata; it is never copied into HTTP headers
or used to select Registry authority or signing keys.

The existing bounded subset also requires **complete record AAD <=4033 bytes**.
Participant/correlation fields consume that budget, so individually valid metadata
can still be too large for a request or response. The envelope cap is 32768 bytes,
record cap 16384 bytes and plaintext cap 16348 bytes. This API does not implement
the full chapter 08 large-message limits or optional task fields. Existing raw,
non-HTTP and ordinary HTTP APIs retain their metadata-free contract: they reject a
metadata member, including `{}`, rather than decrypting and dropping it.

`Check`, `Observe`, `LocalNow`, `LocalStamp` and `Participants` retain the existing lifetime and
configured Source boundaries. Clock samples and observations are diagnostics,
not portable authorization permits. Source/history/finality correctness and
signing custody remain properties of the configured providers. The final replay
publication gate checks call cancellation after the local clock provider returns,
local time and pinned-key liveness. Cancellation there returns no view and closes
the session. This shared request/response gate also strengthens the older APIs
without changing uncanceled wire bytes. It cannot predict cancellation after its
publication point, and it does not claim a
second Registry read or reverse durable denial staging on rejection.

`LocalStamp()` returns Unix seconds and monotonic milliseconds from **one**
sample of the owner's original clock. It shares the exact local checks used by
`LocalNow`: ordered clock history, session and idle lifetime, provisional
responder deadline, pinned-key expiry, rollback and Duration representability.
It neither reads the Registry nor extends activity or a prior authority check.
The returned struct is a value; changing it cannot change the session clock.
Clock providers must remain bounded, thread-safe and non-reentrant. A clock
panic or error returns an error and a zero stamp. Ordinary concurrent validation
progress is not mistaken for clock rollback: the helper captures published
bounds before sampling, as the existing native timer does.

A trusted adapter can use this single sample to compare its own retained signed
message/card deadlines with elapsed time in the core clock's original domain.
It must still perform current authority checks and enforce its own no-bypass
final write/dispatch boundary. Never combine a monotonic value from another
clock origin with this sample or treat a previously returned stamp as permission.
This API does not expose a signing key, mutable session or current Registry
snapshot, and does not predict later cancellation or key revocation.

A real adapter must obtain Method, Target, Authority, Status, field occurrences
and body bytes from its authenticated transport. TLS, framing, trusted routing,
no redirects/retries/downgrades and bounded reads remain adapter responsibilities.
A final send check must be unavoidable after blocking connection setup and before
first application bytes. These core methods alone do not implement that network
boundary or the A2A production dispatch barrier. Core-valid but A2A-policy-invalid
metadata must be rejected by the actual A2A ingress after cryptographic acceptance;
that policy rejection must not undo replay denial.

`TestHTTPApplication*` uses owned fixture keys and a prevalidated Source. Mutation
controls are confined to in-process unit tests. The owned TLS test exchanges inert
content and checks unavailable authority without creating a network attack tool.
These tests do not close arbitrary peer-document/alias protected-verifier cases,
attest a live Registry deployment or establish complete protocol conformance.
