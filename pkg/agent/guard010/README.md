# Execution Guard primitives

This module verifies the mandatory Ed25519 subset of the SAGE 0.10.0 Execution
Guard profile. It provides ordered original commitments, exact artifact manifest
verification, policy commitments, bounded strict JSON, and signed intent/result
verification. Other signature algorithms reject. Errors are uniform; integrations
must not disclose internal verification diagnostics to peers.

Configure the authority, policy and outstanding-invocation interfaces in trusted
host code. Never construct them from peer `active_key`, `approved`, `verified` or
`outstanding` fields. The authority must provide trustworthy time and a fresh
validated registry observation of the exact active key. The policy service is
scoped to the accepted request and provisioned issuer/policy/manifest mapping; it
must enforce the closed tool schema and all final arguments. A universal policy
language is intentionally not supplied. Errors, unavailable decisions and timeouts
must deny. Host callbacks must implement bounded deadlines.

Successful verification owns the canonical complete envelope including proof.
Intent digests therefore bind signatures as well as claims. Accessors expose copies
in Go and immutable borrows in Rust. They do not create a dispatch capability.
An outstanding-invocation service supplies protected, previously authorized intent
bytes; results can remain fresh after that accepted intent has expired. This is
not permission to begin a new invocation after expiry. The host must durably track
invocations and consume a terminal reply once.

Standalone manifests allow 4096 files. Policy descriptors and signed envelopes
apply the smaller 4096 aggregate object-member bound, depth 32 and 1 MiB limits.
UTF-8, duplicate decoded keys, lone surrogates, nonfinite numbers, negative zero and
negative underflow are rejected before canonicalization. Protocol timestamps are
checked as exact nonnegative safe integers before binary64 rounding.

Not implemented here: signed terminal publication/consumption storage, MCP result
mapping, a deployed validating registry Source, or host capability isolation. File byte verification does not
prove that a regular, non-symlink, immutable instance is the one actually loaded.
These are required integration boundaries, not properties established by a valid
signature or fixture flags.

The frozen `testdata/guard-records.json` comes from sage-inspector's independent
0.10.0 vector suite. Tests execute 78 applicable Guard cases through actual APIs;
the fixture service implementations are test-only. The remaining MCP mapping and
other primitive projections do not count as implemented Guard behavior here.

## Authenticated durable reservations

`GuardLedger` owns an execution ledger and a configured recipient. Each reservation
verifies the signed intent against current trusted authority and policy callbacks,
then privately derives every stored identity field and the complete canonical
envelope including proof. Callers cannot submit a raw entry or a cached verification
flag through this API. Call and nonce are reserved atomically. An identical retry
returns the existing state without appending or transitioning it. Changed claims,
proof, or a reused nonce under another call are denied.

The returned observation contains only the created flag, stored state and intent
digest. It exposes no stored terminal bytes and grants no execution permission.
Expiry is checked again after storage; a late denial retains any committed record.
Normal reopen never recreates missing storage and converts pending reservations
to UNKNOWN. Fresh key, policy and time checks also apply to retries after recovery.
Use only trusted local paths and explicit initialization of a new isolated scope.

Tests reuse the independent intent vectors, validate correctly signed conflicts,
concurrent identical reservations, current-authority denial, expiry during storage,
and recovery. Inspector additionally runs both languages in separate bounded local
processes and verifies exact journal identities across all four language pairs.
The separate DispatchGate connects serialized dispatch and retirement; signed result
release remains unimplemented.

## Serialized dispatch boundary

`DispatchGate` owns a separate ledger handle, current authority/policy providers and
one trusted pinned component. It verifies inside the same mutex used by administrative
Replace and permanent Retire. There is no public reusable execution capability or
pre-lock verification snapshot. Retire cancels all not-yet-committed local work;
Replace cannot reactivate a retired gate. Already committed work is not rolled back.

For a new identity it records RESERVED and durable EXECUTING, then repeats current
key/signature, policy, measured component and expiry checks before a bounded Commit.
The same component object receives the complete canonical intent and exact arguments.
An identical retry never commits again, even after replacement or UNKNOWN recovery.
A final check failure or uncertain Commit leaves UNKNOWN; storage failure denies
success and follows the poisoned-store lock rules. Callback panics also retire the
gate. The local receipt reports handoff, never tool completion or signed output.

Component is a trusted integration seam. Check must bind protected approved bytes
and covered dependencies to the same immutable loaded instance that Commit uses.
A matching caller-provided digest is insufficient. Commit must atomically accept
that exact pinned invocation into a protected execution boundary, with bounded
deadlines, no name/path reopening, no added arguments and no reentrant gate calls.
Do not run an unbounded tool while holding the gate. Background work must retain its
pinned instance and arguments after handoff. Go Commit must honor context cancellation;
local gate retirement provides the serialized cancellation boundary in both cores.

Providers must supply fresh registry observations and trusted time. Host policy and
component changes must use Replace/Retire; mutating them behind the gate defeats its
serialization. Administrative authentication, old/new baseline audit records, durable
retirement across restarts and all receivers, actual immutable loaders and host
capability isolation remain host responsibilities. This API does not claim to protect
a gate or its trusted providers already controlled by an attacker.

Native tests cover final denial, callback panic/uncertainty, exact arguments, concurrent
duplicates and deterministic retirement/replacement ordering. Inspector runs 30 local
processes against inert fixture sinks and checks complete handoff bytes and journals.
These sinks measure owned fixture bytes; they do not certify a production loader or
execute external tools. Full Guard lifecycle conformance remains unestablished.
