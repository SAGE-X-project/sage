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

Not implemented here: serialized dispatch and retirement, terminal consumption storage, MCP result mapping, a deployed validating
registry Source, or host capability isolation. File byte verification does not
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
Serialized final dispatch, retirement and signed result release remain separate.
