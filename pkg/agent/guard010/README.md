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

Not implemented here: MCP result
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
publication is described below.

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


## Signed outcome publication

A committed Invocation supplies a private, gate-bound Completion token to its trusted
worker. Finish records the first completed output and full signed envelope atomically,
but never sends a response. Identical canonical output retries reuse that envelope;
conflicting output and recovered UNKNOWN cannot become completed. Keep completion
tokens and signer capabilities outside plugin/model write access.

Dispatch and explicit Reject return a receipt containing one response permit. Reply
consumes it even on failure; copying a Go receipt does not duplicate the permit.
The host must bind each receipt to one fresh outer invocation and must not dispatch
again for the same outer request. After pending, completion cannot be pushed through
that invocation. A later, newly authenticated invocation can retrieve the outcome.
Client polling and consumption are described below. MCP mapping remains separate.

ResultSigner provides a trusted executor key, current key authority and time. The core
constructs the closed domain-separated result, with a 300-second validity interval,
and verifies the signature before persistence and again before release. The first
terminal bytes survive restart without refreshing timestamps or signatures. Expired,
revoked or unavailable result authority denies publication. Accepted work may finish
and reply after intent expiry or local gate retirement; new retrieval must pass all
current intent authentication, policy and expiry checks. Result expiry still applies.

An unsigned recovered UNKNOWN can acquire one signed unknown outcome once the signer
returns. Explicit Reject verifies the incoming intent and current policy, then reserves
an absent call and nonce atomically with a signed rejection. It cannot overwrite an
existing execution. Authentication/policy failures remain unverified local failures;
Reject is not a conversion of failed verification into an authenticated verdict.

Callbacks are trusted and bounded, and must not reenter the gate. A storage failure
cannot publish an unpersisted terminal. A post-persistence validity failure preserves
the terminal but returns no signed response. Filesystem durability assumptions and
rollback limitations are those of execution010; these tests do not establish hardware
power-loss behavior or protect compromised gate/signing infrastructure.

Native tests cover single permits, concurrent completion, exact byte reuse, late
accepted replies, malformed outputs, signing failures, revoked/expired stored results,
capacity denial and restart ownership. Inspector runs 44 bounded local processes,
including all four language pairs and UNKNOWN reopens, and independently verifies
102 published/stored fixture signatures with Node/OpenSSL. The sink never executes
external tools. Full Guard lifecycle and host conformance remain unestablished.


## Client consumption and polling

Client owns one operation's separate protected journal. The host must maintain a
stable issuer/call-to-journal mapping and never initialize another journal for the
same operation. Explicit creation verifies the signed original and local policy;
normal reopen requires that exact original and existing state. Missing or corrupt
state denies. The original envelope, used outer UUIDs and first consumed terminal
survive restart; pre-restart invocation handles are abandoned, not reconstructed
from untrusted response fields. Filesystem integrity and rollback protection remain
trusted deployment responsibilities.

Begin verifies current intent authority, policy and expiry on every attempt, records
a fresh outer UUID, and invokes ClientSender under the same lock as result acceptance.
Sender is a bounded protected transport handoff, not a reusable permission to send
later. It must bind this exact intent/UUID to one actual request, supply fresh outer
nonce/session sequence as applicable, honor expiry and never reenter the client or
queue delayed duplicates. The fixture sender records bytes without network effects.
Production transport integration is still required and is not certified here.

A new handoff requires at least 1000 milliseconds on both UTC and monotonic clocks
since the previous handoff finished. Reopen additionally requires one second of
local monotonic elapsed time before any retry, as well as the persisted UTC interval.
A changed call, inner nonce or proof cannot be substituted through Begin. A transport
failure is unverified and only permits identity-preserving reconciliation under the
same checks; it never manufactures a remote rejection or a new call authorization.
The original could execute once if it never reached the receiver's intact ledger.
This is not a strictly read-only query API or a remote cancellation guarantee.

Accept consumes a private outstanding invocation even on malformed or unverifiable
responses. It validates the current executor key, result time and exact intent binding.
Pending provides no output. The first terminal is durably recorded before output is
released, and only completed provides output. Already-outstanding identical terminal
or delayed pending replies are ignored without output; a conflicting terminal denies
without replacing the first. Terminal acceptance and intent expiry stop new polling;
an accepted invocation may still receive a fresh result after intent expiry.

A crash or post-persistence validity failure can lose delivery after the consumed
marker is durable. Reopen never redelivers it. This is at-most-once output release,
not exactly-once downstream effects; unresolved delivery requires protected operator
reconciliation. Storage failure or unavailable/backwards clocks deny and retain the
exclusive lock for administration. Journals are bounded to 1024 events and 8 MiB,
with no automatic pruning. Host services must be bounded, trustworthy and consistent;
no peer field can install authority, policy, time or transport providers.

Tests cover 19 independent fixture scenarios per core, concurrent terminal delivery,
late pending/conflicts, handoff duration/uncertainty, storage failure and expiry after
persistence. Inspector executes 64 bounded processes, including real signed server
results across all four language pairs and cross-language client journal reopen.
MCP result mapping, full lifecycle certification and deployed host enforcement remain
separate work; no network attacks or host-bypass tools are used.
