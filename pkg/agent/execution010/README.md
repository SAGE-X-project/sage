# Durable execution storage for 0.10.0

This is a storage building block for EXEC-04/05/07, not a complete Execution Guard.
The caller must authenticate the exact canonical intent, verify its signatures,
identity, expiry, current policy and pinned component before each operation. Entry
identity fields are trusted projections, not decoded or verified by this module.
Intent/result hex contains opaque bytes here; storage tests use synthetic envelopes.

## Atomic state and recovery

The store serializes `(issuer, call_id)` and `(issuer, recipient, nonce)` in one
fsynced journal row. Changed intent bytes (including proof), recipient, nonce or
expiry cannot replace a reserved identity. A new entry is RESERVED or a signed
REJECTED outcome. Rejected entries reserve the nonce and can never execute later.

| Existing state | Permitted change |
|---|---|
| absent | RESERVED, or REJECTED with terminal bytes |
| RESERVED | EXECUTING, REJECTED with terminal bytes, UNKNOWN |
| EXECUTING | COMPLETED with terminal bytes, UNKNOWN |
| UNKNOWN without result | UNKNOWN with its one signed terminal envelope |
| terminal with result | no change; exact identical retry returns changed=false |

Commit reports changed=true only after append and file sync. No return value is an
authorization grant. The caller must persist EXECUTING before effects, serialize
live authorization/retirement with dispatch, and never dispatch for changed=false.
A separate current-policy gate is required; this API alone cannot enforce it.

Ordinary open requires intact existing storage. Before returning it durably changes
every RESERVED/EXECUTING entry to UNKNOWN, even following clean close. This assumes
all prior execution owners have stopped; never reopen a live execution scope while
an old tool is still running. UNKNOWN cannot become COMPLETED, REJECTED or EXECUTING.
Recovery invokes no tool. An unavailable signer may attach the one UNKNOWN envelope
later without changing the outcome. Already stored terminal bytes are returned
unchanged, with no signature/time refresh. Lookup is storage inspection only; live
retrieval authentication, policy, intent/result expiry and Client consumption are
outside this module. A crash may leave an external effect whose outcome is unknown;
this does not provide exactly-once tool effects or transaction rollback.

## Storage boundary

Linux/macOS only. The byte-compatible Go/Rust format starts with
`sage-execution-ledger|0.10.0` and LF, followed by canonical JSON rows containing
issuer, recipient, call_id, nonce, expires, intent_hex, state, result_hex in that
order. Canonical here means this storage serializer, not a substitute for JCS
validation of envelope contents. Exact serialized rows are checked on recovery;
duplicate/unknown fields, partial rows and invalid state histories are rejected.

Identifiers are 1–256 ASCII bytes in 33–126 excluding quote, backslash, `<`, `>`,
and `&`. Expiry is 0–9007199254740991. Hex is lowercase/even-length, at most 1 MiB
decoded per envelope. RESERVED/EXECUTING have no result; COMPLETED/REJECTED must
have one; UNKNOWN may await its first result. Journal size is at most 64 MiB and
4096 rows, including recovery rows. Capacity exhaustion denies writes or recovery;
there is no pruning/compaction, so identities and unresolved state are retained
beyond the minimum expires+30 window. Never delete state to free capacity while
old authorizations remain valid.

A trusted local path, protected filesystem integrity and exclusive writer are
required. Creation is explicit, exclusive and syncs the file plus parent directory.
It is for a new isolated execution scope with no outstanding grants, **not** a
lost-ledger reset. Missing normal reopen fails closed. Policy/key epoch migration
across all affected receivers is a separate administrative protocol; a new filename
or epoch string cannot establish safe recovery. No migration API is provided here.

The exclusive `.lock` is released on healthy explicit Close only. Unclean process
exit/drop retains it. Administration must prove exclusive ownership and stopped
prior workers before removing it; no stale-lock heuristic is used. Write/sync failure
poisons the handle, blocks reads/writes and retains its lock. No terminal success
may be published on failure; a subsequent controlled recovery may find a complete
row or reject a torn one. This is not power-cut, filesystem rollback, replication,
network storage or shared-host isolation certification. Go serializes methods with
a mutex; Rust uses exclusive mutable access and can be shared under a caller mutex.

## Verification

A common fixture has 14 scenarios and 72 steps per core. Native tests cover real
file recovery, missing/torn state, exclusive ownership, bounded capacity, injected
owned-file IO failure and 20 concurrent attempts with one new reservation. Inspector
uses real core adapters for 28 scenarios, 20 cross-process/cross-language recovery
combinations and six lost/torn/locked controls. Only owned local test files/processes
are touched. Cryptographic Guard dispatch and malicious host bypass are not tested.
