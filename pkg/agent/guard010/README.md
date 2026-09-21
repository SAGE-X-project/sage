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


### MCP result carriage

The explicit supported negotiated version is `2025-06-18`. Hosts must authenticate
negotiation and reject unsupported versions before protected calls. The parser
checks the closed `structuredContent`, single canonical JSON text block and
`isError` mapping. It returns **unauthenticated bytes**, not a verification verdict.
Use fresh result verification or the client's MCP acceptance method before use.
Client acceptance consumes an invocation even for malformed carriage or unsupported
versions and retains durable first-terminal consumption. Verified result snapshots
can format MCP carriage and derive chapter 08 success/error without signing again.
Snapshot formatting does not extend freshness or replace the server response permit.
The outer wrapper permits 8 MiB for JSON escaping; the inner envelope retains
1 MiB, depth 32 and 4096 object-member limits. Unsigned annotations reject.
Host interception, RPC invocation correlation, transport protection, setup enforcement,
input tool schema and production MCP server integration remain host responsibilities.


### MCP RPC boundary

The tool descriptor exposes only `sage_secure_call` with required `envelope` and
`additionalProperties: false`. Structural schema validation supplements, never
replaces, signed intent and current policy validation. Request parsing returns
unauthenticated intent bytes; the session endpoint delegates to DispatchGate.

This binding supports UUID string JSON-RPC IDs, matching the durable client's
existing outer IDs. Numeric IDs and other JSON-RPC methods are unsupported here;
this is not a general MCP server implementation. The host supplies the expected
ID from the protected transport invocation and routes every protected tool call
through the endpoint. Notifications, batches, direct tools, extra arguments and
ID mismatches reject before tool commitment. One endpoint consumes up to 1024
attempt IDs, including failures, then requires session closure. Do not reconstruct
an endpoint to clear a live session's history. Cross-session/restart replay protection
belongs to the authenticated transport. Closing the endpoint does not close its gate.

The client sender serializes the exact authorized intent during bounded handoff.
The RPC response acceptance path checks the matching ID and consumes malformed
responses and JSON-RPC errors as unverified failures. Server receipts bind each
response to that endpoint and ID while preserving the gate's single-response permit.
Polling uses a fresh ID and unchanged intent, never another tool commitment.
The host still implements authenticated MCP initialization, protected transport,
trusted Source resolution and complete interception without alternate direct routes.
Request wrappers allow 2 MiB and response wrappers 9 MiB; nested Guard and MCP
representation limits remain enforced on original bytes before canonicalization.

### Registry-backed authority

`NewRegistryAuthority` binds one configured issuer/keyid to a trusted registry gate.
Both `ActiveKey` and `Now` perform a fresh `SelectWithTime`, including existing final
Guard time checks after policy, storage and component validation. The selected key
material and expiry are pinned for the authority lifetime; an administrative key
replacement requires a new binding. No positive observation is cached for later
operations. Use separate bindings for the intent issuer and result executor.

The Source must validate full records and proofs; the Store must provide durable
watermarks. The adapter is not a network resolver or host-isolation mechanism.
Bounded callbacks and the effect handoff remain trusted host responsibilities;
revocation observed after a committed effect cannot undo it. More authoritative
reads are deliberate and should be included in deployment latency measurements.

### Non-HTTP MCP session carriage

`SealMCPSessionRequest` and `OpenMCPSessionRequest` bind exact RPC bytes and the
inner issuer/recipient to an existing authenticated signed AEAD session. The returned
`MCPSessionCall` owns the distinct outer message ID and inner RPC ID. Its `SealReply`
and `OpenReply` enforce one response and the original intent digest, with protected
pending/completed/error mapping. Copies share the response permit. No wire fields or
cryptographic algorithms are added; the existing 16348-byte plaintext limit applies.

Use sealing inside the authorized client handoff. Dispatch the incoming call's
`ID()` and copied `Request()` through `MCPEndpoint`; pass opened replies to
`Client.AcceptMCPResponse`. This binding does not authenticate the inner Guard proof
or grant tool execution: those remain existing Guard and client responsibilities.
Authenticated MCP initialization, exclusive routing and bounded handoff are host
requirements. HTTP carriage has a different intent-payload mapping and is unsupported
by these helpers. Failure after cryptographic acceptance never restores outer replay
or a consumed response permit. Do not silently fall back to unprotected calls.

### Private non-HTTP setup implementation

The internal `mcpOwner` coordinator and setup codecs implement the first local
pieces of the adopted [non-HTTP profile](https://github.com/SAGE-X-project/sage-spec/blob/520e5ed9a896ff8ba8ade776484f41084957aaa2/profiles/non-http-mcp-security.md):
client/server phase order, retained setup/protected request-ID history, fixed
setup/protected deadline arithmetic, output publication identity, one copied
deferred frame, and closure independent of blocked transport work. The codecs
validate the selected initialize capabilities, exact initialized acknowledgement,
and complete pinned discovery descriptor. They discard descriptive metadata and
follow the [MCP 2025-06-18 schema](https://modelcontextprotocol.io/specification/2025-06-18/schema)
for initialize fields; the adopted SAGE profile further restricts capabilities
and discovery.

The private `mcpSetupSession` now connects these rules to actual HPKE-authenticated
sessions. It exchanges initialize, initialized acknowledgement and pinned discovery
through signed encrypted records, checks outer success and exact correlation, and
revalidates current registry authority before publication. The observation's start
time remains private and must be no more than 5000 ms old at the final locked clock
sample. Cancellation and clock rollback also deny publication.

`AuthenticatedCompletion010.TakeNonHTTP` transfers an unused non-HTTP session into
a retained handle, disables the old handle and its value copies, and preserves the
original creation clock. Shared use/HTTP flags prevent a stale copy from transferring
an already used or HTTP-bound session. Existing handles cannot erase the new owner's
keys. The endpoint clock used by this adapter must be bounded, thread-safe and purely
local; registry observation work is never performed under the lifecycle coordinator.

The adapter is still private and has no public protected dispatch API. Its server
preparation callback is trusted bounded host work, not evidence that the private
owner-aware Guard gate has been installed. These setup exchanges alone do not establish full binding conformance
or whole-host mediation. Production transport framing and host scheduling remain to be integrated. Transport providers must honor
cancellation, report complete local send results and bound pending input to one frame.
The adapter starts no worker per I/O; active work owns key cleanup on exit. Closing an
idle or completed adapter cleans up keys outside coordinator locks. Cleanup can wait
for a bounded provider or shared endpoint operation; it never delays recording closure.

Native tests use real Ed25519/HPKE handshakes, signed AEAD records, durable registry and
replay journals, and bounded `net.Pipe` framing. The test clock honors replay-store
startup quarantine. Scenario units cover stale aliases, false outer success, clock
panic/rollback, exact freshness boundaries and closure during a blocked send. They
create no attack program or host-bypass implementation. This is Go setup evidence,
not Go/Rust interoperability or Inspector catalog promotion. The isolated 1024-entry
history unit does not override the session's tighter record ceiling.

### Private protected admission and completion

The private admission gate authenticates encrypted requests from its exclusively
owned responder session, verifies the intent, and durably reserves/fences the call.
After storage it refreshes both intent authority and the complete session binding.
Only a final coordinator check of owner identity, generation, deadlines, key expiry,
clock monotonicity and observation freshness can insert the pinned invocation.
Persisting EXECUTING alone never authorizes an effect. Failed final validation
settles newly fenced work as UNKNOWN; uncertain persistence retires the gate.
Duplicates undergo current validation and never enqueue another execution.

All attached owners share a preallocated capacity limit, including authentication,
storage, queued and running work. Closing one owner before insertion prevents its
admission without retiring other owners. Closing after insertion preserves historical
admission. Replacement cancels queued work; a claimed task retains its original
executor and receives cancellation. Synchronous execution includes actual termination,
and capacity remains occupied until execution and durable settlement return. Exact
signed completion is stored before the slot is released. Recovery uses the existing
execution ledger and does not reconstruct an execution queue from unresolved fences.

Native runtime tests cover real protected records, signed durable completion,
duplicates, close/fence/claim ordering, independent owners sharing capacity,
responder signing/KEM revocation during storage, uncertain persistence, observation
ages of 4999/5000/5001 ms, clock rollback, and expiry with session cleanup. They use
only benign in-process effects and deterministic fault schedules.

This gate remains private. A trusted bounded host scheduler must call the worker
within its configured claim bound or arrange cancellation; a late claim settles
UNKNOWN, while an unexpected stall retains capacity. Providers must honor finite
completion bounds. Synchronous paths enforce observed expiry, but timely background
expiry enforcement, complete transport/result carriage and client publication are
still pending. The gate and session clocks must share one trusted monotonic origin.
These tests do not establish whole-host mediation, Go/Rust interoperability or
Inspector conformance. Those claims require the remaining integrations and catalog
execution.
