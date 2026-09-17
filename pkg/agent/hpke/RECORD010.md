# Authenticated session request boundary

The completion result now privately owns a RecordSession010 from key-state
creation. No seed export, second owner or restore constructor is introduced.
SealRequest/seal_request emits a signed, encrypted request with the pinned
participant, key, role, context and session identifiers. OpenRequest/open_request
checks the closed envelope, current pinned keys, signature and AEAD with JCS of
the envelope excluding payload/signature as caller AAD. Invalid tuples,
signatures and tags do not consume sequence or transport replay entries.

The supported carriage is a metadata-free session **request** in either
participant direction, with explicit context/role and Ed25519 signatures. It is
bounded to 32 KiB JSON, 16384 record bytes and 16348 plaintext bytes. Unsupported
optional members and response shapes are rejected. This is not a general HTTP,
WebSocket or response-correlation implementation. Return from seal is the local
emission boundary; a transport retry reuses the exact returned bytes.

The responder cannot send while RESPONSE_SENT. Its first valid unseen sequence
may be nonzero. One endpoint lock (Go) or exclusive borrow (Rust) spans record
validation, the trusted replay transaction, sequence publication and confirmation.
Concurrent callers therefore observe at most one acceptance and one confirmation.
Close is serialized by the same ownership boundary. Invalid authentication leaves
provisional state unchanged; failed current-key validation, expiry, clock failure
or explicit closure retires it. Application authorization is deliberately later:
a rejected application intent does not undo accepted cryptographic state. No
protected application operation is executed by this API.

The trusted replay dependency must implement ReserveRecord/reserve_record. It
atomically checks scoped ID and nonce, stages durable state and calls the final
validation callback exactly once at publication. All fallible/blocking storage
work precedes that callback. Failure publishes neither entry; success durably
publishes both without a later fallible step. The callback rechecks current
operation age, envelope expiry, pending/session lifetime and pinned-key expiry.
The record owner publishes its sequence and confirmation only on successful
return, without another fallible operation. Sessions are never restored: a crash
before in-memory publication discards the session while committed replay denial
remains. This explicit dependency contract does not prove an arbitrary database
implementation has these properties. Unsupported stores reject record receive;
there is no silent nontransactional fallback or production in-memory store.

Both keys and KEM are revalidated per operation. The five-second observation
bound applies through final publication. Session lifetime is measured from
original key-state creation, including responder provisional time; confirmation
does not restart it. Successful emission/acceptance updates idle time, while
Check, invalid traffic and failed reservations do not. Retention, restart
quarantine, deployed registry validation and unavoidability of a host gate remain
requirements of trusted deployment dependencies.

The shared 32-scenario fixture covers tuple/signature/tag rejection, nonzero and
out-of-order acceptance, exact duplicates, transport ID/nonce denial, storage
failure and delay, key changes, expiry and application rejection after acceptance.
Additional unit tests cover absolute lifetime, idle expiry and size boundaries;
Go also checks 16 concurrent copies with race detection. Inspector independently
verifies signed handshakes and records across all four Go/Rust process combinations,
retaining raw messages and revision hashes. Synthetic replay controls test the
integration contract; they are not deployment durability evidence. Full protocol
conformance remains NOT_ESTABLISHED.
