# Authenticated completion boundary

The completion API composes fresh HPKE derivation with current registry key
selection, exact signed request retention, whole-envelope Ed25519 signatures,
the separate completion signature, constant-time ACK verification and pending
lifetime. No `authenticated` boolean or caller-supplied seed can construct its
result. Public tuple access returns a copy; retained seed bytes are private.

Start generates a fresh context, nonce, message ID and independent ephemeral
contributions. Its return is the local emission boundary; callers must start a
new handshake if they defer sending. Respond checks the request's closed payload,
participant/key/context/nonce bindings, signature and local private-key match,
then signs the completion and exact-request-bound response. Complete consumes
pending state on every success or failure and creates an initiator result only
after both response signatures, echoed fields, current pinned keys, ACK and the
final time checks succeed. The final gate also rechecks pinned key expiry,
including expiry reached during durable storage. Completion payload bytes must be JCS-canonical.

This API supports the metadata-free plain handshake subset with explicit context
and role fields, Ed25519 signing and X25519 KEM, and both peers in the configured
registry. Optional task/metadata fields, absent role/context, session envelopes,
failed application responses are not supported by these plain entry points.
HTTP signatures use the separate HTTP-bound methods described below. This is not a general WireTransport implementation. Payloads are
bounded to 16 KiB and these handshake envelopes to 32 KiB before resolution.

Pending state uses a 300-second monotonic cap and strict absolute envelope
expiries; equality expires without grace. Clock failure/rollback, delayed durable
reservation or expired input cannot create a result. A replay reservation that
has already durably committed remains a denial if its operation expires while
storage returns; no result or application effect is produced. Retransmission
does not extend a pending deadline. One-shot state rejects a second completion.

A trusted Registry Gate, Clock and ReplayStore are required. Source validation
of complete registry records/proofs remains a prerequisite. ReplayStore must
atomically reserve both ID and nonce per sender/recipient, retain them through
expires+30, reject reused nonempty initiation contexts per sender, and implement
durable recovery or restart quarantine. There is deliberately no production
in-memory fallback. The tests supply bounded synthetic dependencies; they do not
establish production replay durability, quarantine or deployed chain finality.

The initiator result is ESTABLISHED at this handshake boundary; the responder
result remains RESPONSE_SENT. Neither result exposes application send/dispatch
methods. First-record signature/AEAD/tuple verification and atomic replay/sequence
reservation with responder confirmation remain a separate integration. Current
key checks close results on unavailable, revoked, expired or changed pinned keys;
unrelated key additions keep the original selections. Checks alone are not
traffic and cannot reset idle lifetime; unused results close by ten minutes.

Close pending state and returned results on abandonment/restart, and retire the
endpoint's local credential copies separately. Go values containing ownership or
mutexes must not be copied; explicit Close is required. Rust private temporary
and result seeds use zeroizing ownership and no Clone/restore API. This is logical
buffer retirement, not a claim to prove physical memory erasure or to secure a
compromised trusted process.

A shared 36-scenario fixture covers signatures, ACK, echoed/request bindings,
closed schemas, monotonic/UTC boundaries, unavailable dependencies, delayed
storage and abandonment. Eight additional lifecycle cases cover request replay,
provisional expiry/closure and current-key checks. Inspector runs all four
Go/Rust combinations and independently verifies signatures with Node over fixed
public test seeds, preserving raw messages and failures outside historical
conformance evidence. Full protocol conformance remains unestablished.

## Subsequent record integration

The original completion-only boundary above is extended by [authenticated session
requests](RECORD010.md). Results now privately own record state; responder
confirmation and sending are available only through that verified boundary.
The earlier absence of record methods describes the completion-only revision.


HTTP carriage is now available through the separate endpoint-bound HTTP methods
and strict HTTP/1.1 codec documented in [HTTP010.md](HTTP010.md). The plain methods
above remain transport-independent; an HTTP-bound endpoint rejects them.
