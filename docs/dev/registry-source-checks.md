# Registry Source clock and checkpoint checks

Source is a Registry observation boundary that can run inside an importing Agent;
it does not mandate a separate service. See [Agent embedding](agent-embedding.md)
for the application responsibilities, current HTTP coverage limits and ordered
demo/example integration. Authenticated recovery is separate from ordinary startup.

`pkg/agent/registry011` provides the clock-window and finalized-header continuity
checks needed by the SAGE 0.11.0 Source. The governing requirements are CM-09 in
`profiles/sage-0.11.0/registry-management.md`, revision 2, at sage-spec
`a22a275b4d88f1d415b53cdffc5bed41231b77a0`. This package is separate from the
existing 0.10.0 observer and does not change its five-second freshness rule.

Construct `ObservationChecks` with explicit clock identity, policy generation
and time/work limits. There are no operating defaults. A nil or zero-value
checker refuses. Configuration is copied; changing the input struct later does
not change an existing checker.

The clock provider is a trusted local boundary. Its wall and monotonic samples
must identify the same clock and epoch. Negative samples, clock rollback,
empty or inconsistent epochs, excessive uncertainty and wall/monotonic disagreement refuse.
`CheckReadWindow` bounds the entire acquisition. After durable commit, call
`CheckPublication` using the **original acquisition-start** sample again: a new
completion timestamp must not make old reads fresh. Both methods check finalized
header age conservatively using the completion clock's upper uncertainty bound.
Equality with a configured time maximum is permitted; exceeding it refuses.

`CheckLineage` starts with an already enrolled, durably retained anchor. Supply
every parent from the new header's parent down to that exact anchor, inclusive.
Hashes are canonical lowercase 32-byte hex strings; heights must decrement by
one and parent timestamps must precede their children. Repeated hashes,
rollback, omitted or extra headers, a different same-height anchor and a work
budget overflow refuse. An unchanged complete anchor can pass this check but
still needs the independent freshness and complete-state checks.

These pure checks do not fetch or authenticate headers, enroll a Source,
establish consensus, select a live policy, implement durable storage, validate
commands/history or grant protected-operation authority. A nonempty policy
generation is configuration, not authenticated reconfiguration evidence.
The caller must separately qualify all of those requirements. In particular,
`nil` from a check is not a `ready`, `validated` or `finalized` credential.

The selected Sepolia observation plan permits only
`https://rpc.sepolia.ethpandaops.io`, with no redirect or provider fallback.
This package makes no RPC calls and does not create or sign transactions.
The bounded transport and block-pinned byte reads below implement part of that
boundary. The bounded stateless replay below checks supplied complete historical commands
and normalized actual-call evidence; live trace qualification and durable
incremental history remain separate implementation work. Existing
0.10.0 subjects and their findings remain unchanged.

Finite fixed-data tests and the import example exercise clock boundaries,
restart epochs, publication after the freshness limit, integer overflow,
bounded complete ancestry and fork conflicts. They do not produce network
attacks or claim a live Source/host qualification. CI runs these tests with the
race detector, requires at least 90% statement coverage for the new source file,
and retains source, compiler, revision, test events and coverage evidence.
The four-header historical fixture was captured from the selected official-only
RPC. Its linkage test runs offline and does not claim fresh live qualification.

`DecodeRPCResult` separately checks one JSON-RPC 2.0 response envelope. Supply
explicit byte and container-depth limits, and the exact numeric request ID in
1..2^53-1. This client profile emits canonical decimal IDs and requires that
same spelling in the response; it does not coerce string, fractional or exponent
IDs. The accepted envelope contains exactly `jsonrpc`, `id` and `result`.
Remote errors, batches, extra envelope members, duplicate names at any depth,
invalid UTF-8, unpaired surrogate escapes, trailing values and exceeded limits
refuse. Correctly paired escapes and equivalent escaped field names are decoded
before duplicate detection. A returned result is an independent byte copy.

The implementation supports at most 128 container levels; the actual operating
limit must be qualified explicitly. A future transport must also bound bytes
while reading and refuse redirects, provider fallback and non-allowlisted RPCs.
This decoder cannot prevent an earlier unbounded allocation by its caller and
does not perform HTTP requests, method-specific result or ABI validation, state
authentication, history reconstruction, durable storage or publication. JSON
`null` and `false` remain values requiring those checks, not authenticated absence.

Finite fixed-data tests cover the envelope rules and copied/concurrent use. An
additional offline test decodes the original finalized-header response captured
from the selected RPC, preserving its exact SHA-256 and header identity. This is
historical parser evidence, not a fresh RPC or operating Source qualification.
The envelope follows the [JSON-RPC 2.0 response rules](https://www.jsonrpc.org/specification#response_object)
with the client-profile restrictions above. Duplicate and Unicode validation
are explicit because the [Go JSON decoder](https://pkg.go.dev/encoding/json@go1.26.10)
otherwise permits duplicate names and replacement of invalid Unicode.

CI and container builders select Go 1.26.10, including the crypto/x509 fix for
[GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809). The language version and
module dependency checksums are unchanged. This security patch is separate from
the new Source checks; it does not change a SAGE version or acceptance policy.

## Bounded RPC transport and block-pinned reads

`NewSepoliaReadRPC` owns a certificate-verifying HTTPS transport for exactly
`https://rpc.sepolia.ethpandaops.io`. There is no URL, proxy, arbitrary-method,
redirect, cookie, transaction, signing or provider-fallback API. Supply explicit
request timeout, response-byte and JSON-depth limits; the implementation supports
at most 8 MiB per response, 128 containers and five minutes per request. These are implementation ceilings,
not selected operating defaults. Status other than 200, non-JSON media type,
compressed bodies, read failures, oversized bodies and refused JSON-RPC envelopes
return no successful result. Bytes are bounded while reading, before parsing.
Use a context deadline for the whole acquisition in addition to each request's
own timeout; call `Close` to release idle connections.

`PinAgentReads` checks Sepolia chain ID and genesis, selects the provider-reported
`finalized` header and checks the Registry's initialized identity. The Registry
argument must equal the explicitly selected Sepolia instance
`0x9361223865e641f9941f4640d8fe34faad54bf23` before any network call; a
different valid address cannot enroll itself by returning a matching string. It returns a
private immutable binding to that header, Registry address and agent ID. Every
`CodeBytes`, `RecordBytes`, `GrantsBytes` and `HistoryRange` call from the binding
uses the same [EIP-1898](https://eips.ethereum.org/EIPS/eip-1898) object with that
exact `blockHash` and `requireCanonical: true`. Object-form refusal is an error;
there is no retry by height, latest, safe, another Registry or another provider.
`Header` returns a value copy. The RPC's claims are trusted observations under
the selected trust boundary, not locally verified consensus or proof of freshness.
Clock-window, complete ancestry and post-commit publication checks still apply.

The four read signatures and ABI shapes follow the thirteen-method ABI at
sage-spec `a22a275b4d88f1d415b53cdffc5bed41231b77a0`,
`proposals/management-security-binding/contract-abi.json`, SHA-256
`0f31f995b0169b27c2ce4797b967a754b4c0b2b0a733e4c21da1b54089380f5f`.
This is the normative interface file, distinct from the compiled deployment ABI
artifact. Its exact bytes are retained as a public test fixture. Tests compare
production calldata and decoded return bytes to an independent go-ethereum ABI
implementation. Production checks complete offsets, lengths, zero padding,
canonical bools and exact end of input before returning copied bytes. Records
are bounded to 65536 bytes; retained entry and command each to 262144 bytes.
Empty grants bytes refuse; exact `[]` is still a legitimate raw result.

`HistoryRange` collects every requested version in `(after, current]` using the
same block binding. Explicit item and cumulative entry/command-byte budgets are
required, with implementation ceilings of 4096 items and 8 MiB of cumulative
entry/command content per range. Larger settings refuse before any range call.
These maxima are not live operating defaults. A missing item, RPC refusal, invalid ABI, empty entry/command, backwards
range or exceeded budget returns an error and no partial list. An unchanged
range can return an empty list. Even the uint64 final version cannot wrap.
The range endpoints must come from separately validated record and retained
state. Supplying a smaller current version is not proof of complete history.

These APIs deliberately return **unvalidated content bytes**. They do not yet
verify JCS schemas/digests, replay historical commands or establish actual EOA/
wallet execution, immutable controller, key tombstones, current-state agreement,
complete required code/ABI qualification, enrollment or durable storage. A false
record bit is not authenticated absence, and a successful collection cannot
mark a Source ready or grant message authority. Comparing all thirteen exact
runtime codes, validating every historical transition and persisting continuity
remain mandatory later gates. No live operating numeric policy is selected.
Existing 0.10.0 behavior and all earlier findings remain separate.

Refusal tests use finite inert data and in-memory transport responses. A normal
owned TLS server exercises the positive HTTP exchange at runtime; it is not a
live Sepolia qualification or an attack reproduction. CI applies the race
detector, the separate 90% gate to each new production file, and the existing
source/compiler/test-event/coverage evidence retention. No paid transaction,
private-key access, local heavy build or external RPC test is added.

## Canonical history content and digest agreement

`CheckHistoryContent` checks a complete version-1-through-current list against
separately block-pinned final record and grant bytes. Each `HistoryContentStep`
contains the retained entry/command plus the **complete intermediate record and
grants independently reconstructed by a replay implementation**. `historyItem`
does not supply those historical states. Callers must not substitute copies of
the current state or assume this function reconstructs an omitted version.

The check preflights explicit item and cumulative byte budgets before JSON
parsing. It reuses `HistoryReadLimits` with ceilings of 4096 items and 8 MiB;
**here the bytes include all four fields per step and both final byte strings**,
not only the two fields counted by `HistoryRange`. Limit exhaustion is unavailable
content, not permission to truncate history or lower a protocol bound. Individual
record and entry/command/grant byte ceilings remain in force. Operating numeric
values are not selected by these implementation ceilings.

Inputs must already be exact RFC 8785 bytes: whitespace, duplicate members,
invalid Unicode, noncanonical escapes/numbers and trailing values refuse rather
than being repaired. Chain schema text is ASCII; canonical escapes for ASCII
control characters are retained. Parsing is bounded to 32 containers. Record,
command, history and operation-argument **root member sets** are closed. Every
version/previousVersion, binding, Registry, DID, actor and operation must agree.
Admission seconds are exact safe integers in nondecreasing order. Controller
changes, a successor after terminal state and a create at another version refuse.
Explicit grant targets must match their command; grant/removal arrays are closed,
bounded and sorted by exact operator then scope. SHA-256 over each complete
command, intermediate record and grants must equal that version's canonical
unpadded base64url digest. Final record/grants must match the final step byte for
byte. There is no retained mutable state or successful partial list.

This is a **content integrity check**, not full replay or source authentication.
Nested key/service/proof/argument schemas and their cryptographic eligibility,
lifecycle effects, ACL/grant cleanup correctness, claim admission provenance,
actual EOA/wallet execution and block binding still require the separate replay
and Source gates. An internally consistent invented history can pass this check.
Its successful result must never set readiness or grant protected-message
permission. It does not establish authenticated absence, independently qualify
an ABI or replace the current-state/durable anchor and freshness checks.

The fixed `history-content.json` expectations use independent Python standard
library ASCII JSON construction and SHA-256. Nine version snapshots cover all
eight mutation operations, empty and nonempty grants, original/revoked key
entries, service punctuation/control escapes and terminal state. Their proof
bytes are inert placeholders, not valid PoPs or execution evidence. These data
checks do not relabel the fixture as a successful on-chain lifecycle. CI applies
the race detector and a separate 90% statement gate to `history_content.go`, and
retains the exact fixture, source hashes, compiler, test events and coverage.
No external RPC, transaction, private key or local heavy build is used here.


## Stateless command replay and actual execution content

`ReplayCommands` rebuilds the selected identity from creation through every
observed version. It returns complete intermediate record/grants/history bytes,
checks each against the retained history, and requires the final bytes to match
separate reads at the same anchor. It implements all eight mutations, immutable
controller/key tombstones, exact scoped grants and implicit cleanup, admission
time and expiry, terminal state and exact successor serialization headroom.
Only new admission proofs are verified: strict canonical prime-subgroup Ed25519
PoP and X25519 endorsements by a usable accepted Ed25519 signer. Historical KEM
endorsements are retained without conferring current authority. Optional signing
algorithms are unsupported, matching the deployed required-suite contract.
Registry admission does not invent an X25519 low-order blacklist; DH result
validation belongs to an actual KEM operation. Signature validity never replaces
the separate account authorization check. Principal grammar does not reserve
zero as an additional operator value; configured Registry address policy remains
separate.

Input must include every consecutive canonical header, complete ordered signed
transaction body and consensus receipts from a **separately authenticated**
deployment height through the selected anchor. Header hashes, transaction and
receipt trie roots, receipt types/status, height/parent/time continuity and the
final anchor are checked. All Registry claim and mutation events are derived
from these receipts; there is no supplied filtered success inventory. All
caller/commitment pairs are reconstructed, including presence at block zero,
non-refreshable duplicates, consumed claims, other identities' accepted creates
and the strict later-block / inclusive 256-block reveal window.

Every Registry event transaction requires a complete `ReplayExecution` joined
by block hash, signed transaction hash and index. Protected legacy, EIP-2930 and
EIP-1559 transaction signatures on chain 11155111 are supported; other event
transaction formats are unavailable. There is no EOA heuristic or direct-call
shortcut. Root sender/to/input/value must match the signed transaction. The
whole committed call-tree log sequence must equal the complete receipt sequence.
Each Registry event belongs to one unique successful non-static Registry CALL
frame with zero value, matching actual principal, exact canonical ABI calldata,
identity/version/admission and command/state/grant digests. ABI framing, scalar
and method limits are checked before generic unpacking. Parent failures and
static/delegate execution cannot be relabeled as a successful Registry CALL.

The normalized tree is an **unqualified adapter boundary**, not raw Geth JSON,
a live debug-method implementation, or evidence of node honesty. `ReplayFrame`
`From` means the execution-context address initiating a call (the tracer's
OnEnter `from`), not the child's inherited EVM `msg.sender`. CALL/STATICCALL and
CREATE/CREATE2 enter their target context; DELEGATECALL/CALLCODE retain the
parent storage context. A subsequent Registry CALL must originate from that
context, including a wallet executing delegated implementation code. Own-frame
logs specify the number of preceding immediate child calls; reverting subtrees
contribute no committed logs. Enrollment must separately qualify the exact
node/client/schema, full call/log coverage, raw evidence retention, code/ABI,
deployment provenance, finalized header authenticity and pre/post acquisition
identity. No boolean label establishes that qualification.

All limits are explicit acquisition/work budgets, without operating defaults.
Implementation ceilings are 4096 consecutive blocks, 16384 transactions, 65536
frames/logs, depth 32, 4096 new proofs and 64 MiB evidence. History additionally
retains its 4096-item / 8 MiB bound including all reconstructed states. Exceeding
any budget returns no partial output and is unavailability, not a new protocol
limit or permission to truncate. A deployment older than the block budget cannot
be replayed by this API: separately qualified durable incremental replay and
anchor retention remain necessary for long-running operation. Input is copied
where returned and must not be concurrently mutated during checking.

Owned offline unit fixtures cover all eight operations with handwritten expected
snapshots and normal public RFC 8032 sample-key signatures, plus independent
Python JSON/SHA-256/len16 and RFC 3986 grammar expectations. Headers, signed test
transactions, receipts and normalized traces in these tests are synthetic
in-memory content, not live execution or Source qualification. Finite refusal
cases produce no network traffic, attack generator, operator key access or
transactions. Existing safe transport runtime tests remain separate. CI retains
source/ABI/fixtures/compiler/test/coverage evidence and enforces 90% statement
coverage per new production file. No replay success is SourceReady, authenticated
absence, enrollment, durable commit or permission to process a protected message.


## Connecting original node evidence

`PinnedAgentReads.ReadNodeBlock` acquires a complete selected canonical block
from the sole approved RPC, using `debug_getRawBlock` and
`debug_getRawReceipts` with its exact hash. Transaction count is checked before
allocating typed transactions. Canonical RLP, header identity, ordered signed
transactions, receipt and transaction trie roots, gas, uncle and withdrawal
commitments are checked; gas accounting is separate below. Every transaction emitting a Registry log requires a
full native `callTracer` tree with `onlyTopCall=false`, `withLog=true`, a bounded
timeout and `reexec=0`. No Javascript tracer or transaction submission exists.

`DecodeCallTracer` preserves the initiating context address and own-log position
relative to completed immediate children. Its closed schema supports the known
Geth-style fields and Reth 2.8.1 gas accounting extensions. A missing value is
zero only for STATICCALL; a failed CREATE with no destination or an unsupported
opcode is unavailable. Duplicate names, invalid Unicode, unknown fields,
inconsistent positions and exhausted budgets refuse. The resulting tree is
joined to the selected transaction signature, root call and all consensus
receipt logs. The provider-reported client version is retained as a claim,
not proof of the node software or its trustworthiness.

`ObserveDeployment` additionally compares all thirteen runtime byte sequences at
the same EIP-1898 block hash to the pinned public deployment manifest. The full
local deployed ABI is pinned by SHA-256. It binds the one existing Registry
creation transaction to the deployment block, expected sender, nonce, address,
receipt and all eight constructor dependencies, and checks its complete call
tree. This establishes thirteen **anchor-time code identities and one Registry
creation**, not fresh provenance for all nine creation transactions or all
thirteen instances. The recorded creation has sixteen frames and no logs; it
cannot qualify live Registry mutation logs or Agent lifecycle behavior.

Both APIs recheck chain ID, genesis and Registry identity, retain exact response
envelopes and fixed request parameters, and recheck the canonical selected block,
the retained anchor and the provider-finalized height after acquisition.
`NodeLimits` requires explicit total wire bytes, request count, acquisition time
and native tracer time in addition to transport and replay limits. The smaller
remaining response budget is enforced while streaming, before JSON decoding.
Frame, log and transaction limits apply across the complete acquisition. These
bounds are ceilings, not approved operating defaults. They cover each node API
call; a future Source must separately enforce the entire initial pin, acquisition,
durable commit and publication window with its qualified clock.

The opt-in `registry_live` test and Registry Node Evidence CI workflow perform
only bounded reads of the already deployed Sepolia contract. They retain source,
compiler, execution context and the original responses. Finite refusal tests
use inert in-memory data; normal wallet/log fixtures are offline test evidence.
No operating key, paid transaction, alternate provider or local heavy build is
needed. The live job uses explicit trial limits, not an operating policy.

A selected block is not the full interval since deployment. If that interval
exceeds the stateless replay limit of 4096 blocks, it remains unavailable until
durable incremental replay is implemented. No history is skipped, truncated or
rebased. These APIs do not grant Source readiness, authenticated absence,
consensus/finality proof, custody or host assurance. Normal paid Agent lifecycle
verification remains a separately proposed and approved action.

Wire references: [Geth native tracers](https://geth.ethereum.org/docs/developers/evm-tracing/built-in-tracers),
[Geth debug API](https://geth.ethereum.org/docs/interacting-with-geth/rpc/ns-debug),
and [Reth debug API](https://reth.rs/jsonrpc/debug/). The deployed ABI and thirteen
code identities come from the existing public Sepolia deployment record in
`sage-contracts`, not from an ABI advertised by the RPC.

The content-only collector checks commitments, not current fork gas accounting.
Every block and deployment observation includes the header gas, final receipt
cumulative gas, and `LegacyGasCompatible`, computed only from those retained
bytes. The recorded deployment has 142,149,240 header gas and 206,878,059 receipt
cumulative gas: the legacy flag is false. This content-only bundle has no gas
witness and remains rejected by command replay.
The shared content decoder retains all canonical encoding, type, status, log,
monotonic cumulative gas, transaction-root and receipt-root checks. Command replay
always calls its strict wrapper; there is no bypass option and no fork inferred
from a header shape, equal totals, or client string.
A successful collection or matching legacy gas equality is not consensus/fork
validation, valid gas accounting, complete history or Source qualification.

## Pinned Sepolia gas observations for strict replay

`ReadReplayBlock` collects the same complete block contents and required Registry
call trees, plus a native `debug_traceBlockByHash` `stateGasTracer` result for
**every** transaction. The request uses the exact selected hash, explicit timeout,
and `reexec=0`. Original response envelopes and method/parameter bindings are
retained. The live artifact stores each response in a separate file with its
exact byte count and SHA-256; the prettified bundle alone is not raw wire
evidence. HTTP 200 is the transport's mandatory admission condition, not a raw
HTTP-header capture. Methods and serialized parameters are retained, rather than
claiming a capture of full HTTP or JSON-RPC request bytes. The response's gas
dimensions are **trusted-provider assertions**, not values committed individually
by the transaction or receipt root. No custom tracer, code execution request,
alternate provider, transaction submission or local operating key is used.

The narrow `sepolia-amsterdam-reth-state-gas-v1` format uses the Sepolia
Amsterdam activation timestamp **1791294816 (2026-10-06 13:53:36 UTC)** from
[the pinned official network configuration](https://github.com/eth-clients/sepolia/blob/7b1b9d9549787b6bbf538f943304c4cf84f7e39e/metadata/genesis.json).
This is not a rule for mainnet or an automatic adoption of future EIP changes.
The pinned specifications are [EIP-8037](https://github.com/ethereum/EIPs/blob/5ef380124bd68d09c02fa90d0de0f2455f37d5ee/EIPS/eip-8037.md)
and [EIP-7778](https://github.com/ethereum/EIPs/blob/f1ade090fd315895461fc3e249cb1a675aeb55a9/EIPS/eip-7778.md).
Implementation references are [Reth 2.8.1](https://github.com/paradigmxyz/reth/blob/d763342fa185e4e90ee79dc65f6a5706168b0bcb/Cargo.lock),
[its native tracer](https://github.com/paradigmxyz/revm-inspectors/blob/a983a3b69641c56ba070f1318a5e1834dafa525d/src/tracing/debug.rs),
[revm's gas result](https://github.com/bluealloy/revm/blob/6b2e655b91f21c3a21770bed78def5b379ed1d17/crates/context/interface/src/result.rs),
and [the independent Geth accounting implementation](https://github.com/ethereum/go-ethereum/blob/9dffe6e976cbbb87a84c5fd395448f0950290def/core/state_transition.go).
The reported client version remains an unauthenticated claim, not profile
selection or evidence that the endpoint runs those exact bytes.

Before activation, the strict wrapper retains exact final-receipt/header gas
equality and rejects an unexpected split-gas witness. At and after activation,
even matching totals cannot substitute for the required pinned-profile witness.
It verifies closed JSON fields, duplicate names, Unicode, exact quantities,
one-to-one transaction hash/order, paid gas against each committed receipt's
delta, transaction limits and both pre-inclusion reservation dimensions. The
maximum of the cumulative execution and state dimensions must equal header gas
and must not exceed its limit. Existing canonical RLP, types, status/log,
cumulative monotonicity, roots, signature/caller, request/byte/time budgets and
post-acquisition canonical/finalized rechecks all remain enforced.

For total gas before refunds `T`, state gas `S`, transaction floor `F`, and raw
refund `R`, the pinned rules use `E=max(T-S,F)` and `P=max(T-R,F)`; the header uses
`max(sum(E),sum(S))`, while receipts accumulate `P`. Refunds remain capped at
`T/5`. The tracer reports an **effective** refund, which is zero when the payment
floor clips. The implementation recovers `T=P+R` only when the payment floor is
inactive. It checks the decomposed EIP-2780 base, uniform 64-gas calldata byte
cost, and EIP-7981 access-list byte floor using signed transaction fields and a
recovered sender for self-transfers. It does not assume `P=E+S-refund` when the
execution floor binds. At a clipped payment floor it accepts only the narrow
zero-state case; positive-state ambiguous settlements refuse rather than invent
missing pre-refund counters. The observed 120-transaction block has no such
ambiguous positive-state case. A future provider format or fork needs a separately
reviewed adapter; no unknown fields, missing data, fallback or truncation passes.

The retained deployment block's sums are **66,635,112 execution gas**,
**142,149,240 state gas**, and **206,878,059 paid gas**. The larger dimension equals
its header; paid gas equals the final receipt. The original content fixture is
unchanged, and a separate unaltered native gas-response fixture supplies the
additional evidence. The opt-in live CI retains both the old content-only bundle
and a separately labelled `strict-replay-block.json`. It checks **provider-observed
gas consistency**, not independent EVM execution, header consensus or finality.
This completes a bounded strict block admission path; complete durable history,
Source qualification, live Agent lifecycle and operating approval remain separate.

## Local incremental replay journal

`CreateReplayJournal` and `OpenReplayJournal` add an explicit local persistence
boundary. They do not enroll or ready a Source, authenticate an RPC, grant
protected use, or establish authenticated absence. The operator must separately
qualify the directory, filesystem durability, clock, node/ancestry/finality,
complete code and ABI identities, and independent checkpoint retention. There
are no operating defaults. The immutable local configuration binds these named
identities, policy generation, exact deployment header and work limits; supplying
an identity hash is not proof of its qualification.

Each `ReplayBatch` contains every consecutive complete block after the retained
cursor (including deployment for the first batch), only newly observed selected
history items, and separately pinned final state. A private incremental replay
path derives each global successor through the existing strict body/gas, actual
caller, command, proof, ACL and lifecycle checks. It carries all consumed and
expired caller/hash claims, created identities, controller, key tombstones,
grants, terminal state and version. It never treats a history suffix as a new
version-one create and never prunes lifetime duplicate indexes. A failed batch
discards its private state copies. The older complete-range public APIs keep
their contracts and ceilings.

Batch work counters reset; lifetime state does not. Explicit total journal,
claim and identity budgets refuse without truncating history or weakening
protocol limits. Reopening replays all retained batches from deployment; it
never trusts a serialized state snapshot. This supports more than 4096 total
blocks through bounded batches, at the cost of bounded full recovery work.

Backfill progress is distinct from fresh finalized selection. Each batch carries
`SelectedFinalized` and the entire descending ancestor path through its previous
cursor (initially deployment). Every decoded visited block must match that path;
the previous finalized selection must also remain on it. Acquisition freshness
is checked against the fresh selected header, not the historical cursor. The
supplied ancestor metadata still needs external authentication; hash-shaped
fields alone cannot establish a real chain. A cursor behind the selected header
cannot publish. No height-only branch selection or skipped deployment interval
is permitted.

The existing directory must be empty for explicit creation. `Open` never creates
missing history. An exclusive local lock is held until `Close`; a crashed
process's stale lock is not automatically stolen. A separately authenticated
operator must resolve ownership. This is conservative refusal, not automatic
crash recovery or a distributed lock. Complete immutable sequence-numbered batch
files are SHA-256 chained to the configuration and prior batch. Files and the
directory are synced. A durable PENDING marker precedes append work and is
removed only after the new batch is synced. A torn or uncertain append,
quarantine marker, unexpected file, sequence gap or mismatch refuses reopening;
there is no prefix fallback, truncation, cleanup, reset or repair path. IO failure
cannot guarantee successful quarantine persistence and is returned as failure.
The actual filesystem's power-loss behavior remains an external qualification.

`Open` requires the exact independently retained latest `JournalCheckpoint`,
including sequence, digest, anchor and version; both a rolled-back prefix and an
unacknowledged newer tail refuse. A writer of the trusted directory can fabricate
a new self-consistent journal; its hash chain is not authenticity or rollback
protection without that external trusted checkpoint. Before dependent publication
the caller must durably retain the returned checkpoint independently and supply
its exact readback to `Publish`. The API checks equality, not the honesty or
durability of a caller-provided acknowledgement.

Neither append nor restart enables publication. `ConfirmObservation` requires a
complete fresh, separately qualified read after the disk commit, with exact
current anchor and final bytes; it records that new read's original start.
Publication uses that original start and a current trusted sample after disk
commit and checkpoint retention. It checks the confirmed epoch,
completion ordering, observation age and finalized lag again. An old completion
timestamp cannot be substituted as a new start. Expired, conflicting or unknown
inputs quarantine dependent use; this API has no quarantine-clearing or
authenticated reconfiguration operation.

Local deterministic closed JSON is a storage format, not an additional network
or RFC 8785 wire contract. Reads preflight bounded file sizes, directory counts,
96-container depth, 32 members per object, 2^22 streamed tokens and configured
array counts before typed decoding, then compare exact typed re-encoding; unknown/duplicate
members or repaired spelling do not pass. Fixed names and `os.Root` confine path
resolution, but do not protect against a writer of the trusted directory, hard
links, mounts or a compromised host. Embedded gas JSON may be compacted by the
local encoder; this journal is not original RPC response-byte retention. A Source
must separately retain those original exchanges and their acquisition bindings.
The owned temporary-directory runtime tests
exercise real writes, syncs, restart, exclusivity and finite failures. Their
synthetic offline blockchain inputs and handwritten expectations do not qualify
live mutation history, a production filesystem/host, external retention or a
Source. CI retains their source/events/coverage and applies a separate 90% gate
to every added production file. No RPC, operating key, paid transaction,
deployment, hostile runtime producer or local heavy build is added.

## Same-endpoint Nethermind trace representation

The selected official endpoint also returned a client-version claim of
Nethermind 2.2.1+c4f1015b. The claim does not attest its binary or choose a
trusted profile. An exact public Core call response retained by CI has sixteen
frames: the root supplies `regularGasUsed`, `stateGasUsed`, `gasRefund`, while
all fifteen children omit dimensional metadata. The pinned reference is
[Nethermind c4f1015be8b989dd609803cc50d3928d6e0dfd4f](https://github.com/NethermindEth/nethermind/tree/c4f1015be8b989dd609803cc50d3928d6e0dfd4f).
Its native CallFrame converter and stateGasTracer converter define the known
regular/state/refund representation. Per-frame dimensional diagnostics are
optional, but if present must be one complete known trio; partial trios and
simultaneous execution/regular aliases refuse. These diagnostics never replace
the separate complete block gas witness.

`sepolia-amsterdam-nethermind-state-gas-v1` is a distinct closed witness schema.
Each result has exactly `gasUsed`, `regularGasUsed`, `stateGasUsed`, `gasRefund`.
Acquisition classifies a known bounded data shape, then checks every row against
exactly that profile; a mixed or hybrid array refuses. Empty results retain the
existing empty representation and still require an empty committed block. No
client string, endpoint fallback, retry, unknown field or validation-failure
fallback selects an alternative profile. The existing Reth v1 schema remains
separate and unchanged.

The pinned [block inclusion helper](https://github.com/NethermindEth/nethermind/blob/c4f1015be8b989dd609803cc50d3928d6e0dfd4f/src/Nethermind/Nethermind.Evm/GasPolicy/Eip8037BlockGasInclusionCheck.cs),
GasConsumed, TransactionProcessor and NativeStateGasTracer implementations bind
the same independent execution/state reservations and execution floor. However,
this tracer reports **raw capped refunds**, including when payment floor clips;
the earlier Reth format reports effective refunds. Above the payment floor,
`T=P+rawRefund` is available and the existing settlement equations apply. At
`P=F` the Nethermind adapter supports only `E=F, S=0, rawRefund=0`; all other
clipped settlements refuse as unavailable. It does not invent an unreported
pre-refund total or inherit the Reth clipped-refund inference. This is an
explicit unsupported evidence shape, not a change to the protocol or a lower
required transaction bound.

The untouched captured call response is an offline compatibility fixture.
Controlled field-name changes to the older Reth gas fixture test the known
mathematical overlap and finite refusal cases only; they are not a new live
Nethermind stateGasTracer observation. Fresh CI must independently retain and
check its actual complete gas witness before reporting that observation. Node,
finality, whole-history Source and operating qualification remain separate.

## Explicit native root-call gas adapter

The approved official endpoint also returned the Erigon call-trace format
defined by the immutable upstream implementation revision
[`cf6f14132ff042df33b8ce64b0114daeb0d15145`](https://github.com/erigontech/erigon/blob/cf6f14132ff042df33b8ce64b0114daeb0d15145/execution/tracing/tracers/native/call.go).
Its root reports the regular/state/refund trio plus a nonzero
`stateGasReservoir` when present. That exact root shape selects child frames
with signed int64 net `stateGasUsed` and an optional unsigned reservoir. Mixed
child trios, malformed quantities, negative zero, overflow and unknown fields
refuse. The reservoir is not assumed to equal `gasLimit - 2^24`. These are call
diagnostics only; accepting the complete tree does not qualify block gas.

`ReadReplayBlockWithRootGas` explicitly selects native `callTracer` for a
complete block, with `onlyTopCall=true`, `withLog=false`,
`includePrecompiles=true`, the existing bounded timeout and `reexec=0`.
`ReadReplayBlock` still selects native `stateGasTracer`. Neither API retries
the other, falls back to another provider or selects by a client-version claim.
Full call/log traces remain separately required for every Registry-event
transaction; top-only gas data cannot stand in for its actual execution tree.

The root adapter checks every row's order/hash and context against the signed
transaction and receipt: sender, CALL/CREATE kind, destination (including the
derived successful creation address), input, value and gas limit. A failed
creation may omit its destination; this gives no successful-execution claim.
Nested calls/logs and unknown root fields refuse. Quantities are bounded and
the original responses remain untouched in `NodeBlockEvidence.Exchanges`.
`Block.Gas.Trace` is the separate **derived** four-dimension witness; it must
not be described as an original native stateGasTracer response.

The profiles `sepolia-amsterdam-root-regular-gas-v1` and
`sepolia-amsterdam-root-execution-gas-v1` preserve their respective exact
dimension names. Every row uses the same selected contract; mixed aliases
refuse. Both use the conservative payment-floor subset: at `P=F`, require
`P=E=F` and `S=refund=0`. Otherwise, the existing receipt, reservation, refund,
execution-floor and block gas equations remain mandatory. This supports only
unambiguous observations; no hidden raw refund or unsupported clipped case is
inferred. The Erigon
[settled gas structure](https://github.com/erigontech/erigon/blob/cf6f14132ff042df33b8ce64b0114daeb0d15145/execution/protocol/mdgas/md_gas.go)
defines its refund as capped before the calldata floor. Changing diagnostic
field names in tests proves synthetic schema compatibility, not another node
implementation's observed behavior.

The public `erigon-core-trace.json` fixture is the untouched existing creation
response retained by the failed CI. `erigon-root-gas.json` is one separately
reviewed, fixed read of that already deployed block from the same sole approved
endpoint. It contains 120 top-level results, including reverted transactions;
there are no new transactions or deployment keys. Tests bind both originals by
SHA-256 and use finite in-memory refusals. Production acquisition still needs
its own fresh evidence. No result here is independent EVM/consensus validation,
complete Registry history, host qualification or permission for Agent dispatch.

## Explicit standard JSON block content

`NodeLimits.BlockEncoding = EthereumJSONBlocks` selects `ethereum-json-v1`
before acquisition. This uses `eth_getBlockByHash(hash, true)` and
`eth_getBlockReceipts(hash)` at the sole approved origin, with unchanged byte,
request and time limits. An empty encoding retains the original raw debug RLP
path; unknown encodings refuse before network work. There is no retry or
validation-failure fallback between them. The live check explicitly selects the
standard encoding to remove its dependence on stalled raw debug methods.

The standard encoding supports the selected post-Amsterdam PoS shape, requiring
all 23 known header fields, including the block access-list hash and slot number.
Missing/null header fields, unknown fields, duplicate JSON names, unsupported
shapes, nonempty uncle lists, oversized arrays and mismatched hash/ordinal
metadata refuse. The pinned go-ethereum 1.17.5 codec reconstructs consensus RLP
from complete signed transactions, withdrawals and receipts. Transaction hashes
and recovered senders, withdrawal and transaction roots, receipt/log locations,
receipt types/status and cumulative paid gas remain bound to that exact block.
Typed array counts and original JSON bytes are bounded before allocation.
This encoding caps each access, authorization and blob-hash lists and storage-key list at
16,384 entries, withdrawals at 16, topics per log at four, and nesting at 32.
These are explicit implementation ceilings, not new protocol or operating defaults.

Reconstructed RLP is **derived consensus data**, not an original RLP response.
`Exchanges` retains the unchanged original JSON envelopes and exact standard
method parameters. The existing strict decoder still verifies canonical RLP,
the header identity, full body/receipt commitments, gas witnesses, Registry
callers/logs and canonical/finalized rechecks. JSON acquisition cannot supply
missing execution/state/refund dimensions, and never changes a failed strict
result to PASS. Actual method availability and the complete gas witness still
require fresh CI; code compatibility tests do not qualify an operating Source.

Finite units compare every reconstructed consensus byte with the existing
120-transaction RLP and receipt originals, test omissions and inconsistent
metadata, and preserve refusal of the recorded dimensionless gas response.
Only normal positive responses use an owned TLS server at runtime. Refusal
scenarios remain inert data units. The new production file retains the 90%
per-file gate. No endpoint, signing, transaction, policy activation or protected
Agent dispatch API is added.

## Bounded collection inside an Agent

`ReadRPC.CollectReplayBatch` joins the existing readers into one bounded
acquisition for an importing Agent. Supply the complete `JournalConfig`, the
exact externally retained checkpoint, `SourceReadLimits` and a trusted local
`SourceClock`. `SepoliaEvidenceDigests` identifies the exact bundled code
manifest and deployment ABI bytes; a digest is an identity reference, not trust
enrollment. The configuration must select this precise deployment and bundle.

Every initialization, ancestry, code, block, receipt, trace, state, history and
final recheck passes through one private RPC scope. Its request, response JSON
byte and absolute time counters never reset between constituent APIs. Failed
attempts and consumed failed-response bytes also count. The scope uses the
existing owned HTTPS transport and sole approved endpoint. No alternative
provider, tracer retry, caller transport, signature, key or transaction API is
introduced. The original reader remains usable independently after refusal.

The first batch includes deployment; later batches start immediately after the
retained cursor. Each batch includes every body/receipt in its interval and the
entire selected-finalized-to-cursor parent path. A short backfill interval does
not skip the remaining ancestry or pretend its stop block is current finality.
All thirteen code identities are checked at the selected block. Record, grants
and every newly claimed history version are collected at the one exact stop
hash with EIP-1898 canonical pinning. The current version comes from bounded
canonical record bytes, rather than an independent counter. Version zero maps
only an absent, empty record to the internal empty state. CM-08 specifies that
`activeGrants` refuses an absent identity, so that read is not issued and no
successful empty-grants response is fabricated. This internal state is never
an authenticated absence credential. Original response
envelopes remain in the returned `Exchanges`.

The returned `SourceBatchEvidence` is a candidate for full journal replay.
`ReplayJournal.Append` must still validate all command effects, signatures,
permissions, lifetime indexes and final bytes. Successful transport collection
does not make an invented record authoritative. Historical code at each call,
provider/clock/storage qualification, exact external checkpoint acknowledgement,
post-commit confirmation and publication freshness remain separate requirements.
No method here activates a generation or permits protected Agent dispatch.

The upper bounds are 65,536 attempted requests, 256 MiB of response JSON and
five minutes per batch; these are implementation ceilings, not operating
defaults. Explicit per-node and batch replay budgets apply too. A large initial
ancestry gap can remain unavailable within these bounds. Refusal returns no
partial batch and must not be handled by inventing an initial checkpoint,
skipping headers, resetting counters or silently increasing limits.

Tests use finite recorded responses and explicit synthetic header/state
scenarios. Owned temporary journals check that collection cannot substitute for
replay. The repeated-transaction block is an offline cumulative-budget refusal
scenario, not a consensus-valid chain extension or an executable attack. These
tests do not claim a new live collection or actual Registry-authorized Agent
message exchange. That integration follows the existing work order in
[the Agent embedding notes](agent-embedding.md).

The same approved RPC can return different native client schemas. The original
`geth-root-gas.json` response from the read-only CI contains 120 ordinary
top-level callTracer frames without execution, state or refund dimensions.
`TestRootGasRecordedMissingDimensionsRemainUnavailable` preserves that refusal
and checks that it performs no tracer retry. An HTTP success, client-version
claim or a previous richer response cannot fill these missing values. Such a
response does not establish strict gas support or Source readiness. The live
check continues to fail when this required evidence is unavailable; the unit
test's successful refusal must not be reported as a successful live replay.

## Candidate provider transport capture

`CaptureSepoliaProviderFinality` is a separate, data-only feasibility probe for
one `eth_getBlockByNumber("finalized", false)` call at the selected official RPC.
It owns certificate-verifying TLS and negotiates HTTP/1.1 so bounded original
response headers can be retained before normalization. Whole response-wire bytes
(including transfer framing) are bounded and retained, and a framed entity must
be followed by actual EOF on the same reader. Original name/order,
duplicates of non-framing fields, entity bodies and verified certificate-chain
hashes are retained. Duplicate framing fields, content compression, trailers,
redirects, malformed or excessive bytes and invalid JSON-RPC envelopes refuse.
There is no alternate origin, proxy, caller transport, retry, transaction or
signing API. A wire budget includes plaintext HTTP framing and both request and
response bytes; it does not bound TLS packets or the platform network stack.

The sample is diagnostic data, not `ExchangeV1`, a checked batch, a policy
credential or Source readiness. Its system-root selection, wall timestamps and
single request are not qualified manifest trust roots, clock samples, durable
budget tickets, complete ancestry or history. They cannot authorize protected
Agent dispatch. On read refusal a nonnil sample may retain bounded incomplete
wire diagnostics with Complete=false, a capture error and no Result. These are
not a successful exchange or a retry instruction. Existing strict replay, gas
checks and required node CI remain
unchanged. Actual policy activation requires a separate later decision.

Finite inert fixture units cover malformed data. An ordinary test owns both TLS
endpoints and a test-only certificate; it provides no production endpoint or
certificate override. A separate opt-in CI test performs one official read and
retains raw source, headers/bodies, TLS metadata and coverage. Its result is only
transport feasibility, never full provider execution/Registry qualification.

## Candidate policy byte identity

`ParseProviderPolicyCandidate` accepts only bounded, already canonical local
candidate JSON. All root and nested member sets are closed; duplicate names,
unknown fields, invalid Unicode, trailing values and alternate encodings refuse.
Unsigned counts are decimal strings without leading zeroes or overflow. Every
budget is explicit and positive. Operation ceilings are 65,536 requests,
256 MiB wire/decoded bytes and five minutes; operation/segment/bootstrap
budgets must not decrease between scopes. These relations do not prove that a
workload fits or qualify an estimator, clock or retaining store.

This implementation candidate fixes `format` to
`sage-provider-registry-observation-v1`, `retention.format` to
`sage-provider-retention-v1`, and `spec_revision` to the existing normative
commit `a22a275b4d88f1d415b53cdffc5bed41231b77a0`. These are local candidate
identifiers, not registered standards or peer message formats. The supported
instance is the already selected Sepolia genesis, Registry ID and deployment
block hash. The provider origin remains the sole official RPC, with redirects
denied and no proxy. Carrier ordering and the explicit trust assumptions and
unverified guarantees are fixed; artifact digests are checked for spelling only.
No referenced artifact or operating policy is accepted on the strength of a hash.

The parser computes separate domain-separated identities for policy, instance
and provider. `BindCandidate` records proposed namespace, Source ID and u64
generation as candidate metadata. The instance identity excludes policy,
namespace, Source ID and generation; changing those cannot describe a fresh
instance or erase a retained root. Neither parsing nor binding selects policy,
authenticates an Agent setting, enrolls a Source, establishes authoritative
absence, creates `ExchangeV1` or returns a checked replay/dispatch capability.
Zero values refuse access and returned bytes/metadata are copies. There is no
activation method, automatic fallback from strict replay or operating default.

Finite inert fixture tests check schema refusal, inclusive ceilings, identity
separation, immutable concurrent reads and rejection of fabricated JSON state.
Fixture digests and numeric limits name no qualified artifacts or operating
values. Full manifest/artifact qualification, authoritative enrollment,
durable accounting, owned store/clock checks, replay/history and complete
bootstrap/catch-up publication remain separate gates. Strict gas checks and
their required live CI result are unchanged; actual policy activation remains
a separate decision after qualification.

## Candidate first-enrollment persistence

`OpenCandidateProviderNamespace` opens only an existing Agent-owned retaining
directory with a preprovisioned canonical `ANCHOR`. The namespace, instance,
anchor digest and independent expected-tip pin are explicit trusted integration
inputs. Matching hashes do not authenticate their custody. No production API
creates the anchor, selects an authority or infers first use from missing data.
The retaining directory must be independent of candidate staging and trusted
against external writers, hard links and mount changes; path confinement does
not establish those assumptions.

Only a null anchor root with no enrollment entry can accept one initial append.
An exclusive lock, expected old tip, closed bounded inventory, exclusive write,
file and directory sync, and exact readback precede the private receipt. Existing
roots, including quarantined, fenced and retired states, cannot be overwritten.
Changed policy, Source, generation or directory is not proof of a fresh instance.
Evidence files have an explicit byte ceiling; the lock has fixed seven-byte
contents. Failed writes, syncs or readbacks produce no receipt and fence the
handle. A failed close remains an error on later close calls.

`RecoverCandidateProviderEnrollment` recovers only the identical retained first
operation after a lost ACK and released lock. It verifies exact canonical bytes
and the expected old pin, resyncs and rereads without rewriting evidence. A lock
left by a crash is never stolen; authenticated crash recovery is unsupported.
An independently retained newer tip rejects an empty replacement directory.
This depends on independent pin custody: losing the directory and its trusted
high-water state is not detected merely by recomputing a digest.

Snapshots are bound to their owner and current tip; stale, closed and failed
handles refuse. Receipts expose copied historical diagnostics only. JSON cannot
restore private state, and no Source API accepts these candidate types. Finite
unit fixtures and owned temporary-file tests cover first-registration races,
state combinations, reopen, exact ACK recovery, invalid evidence and I/O failure.
This component does not qualify an authority, operating filesystem or policy,
implement later enrollment transitions, account for all resource vectors, or
authorize Agent messages. Full retention/clock qualification, replay/history,
bootstrap/catch-up and actual policy activation remain separate gates. Existing
strict verification and its required live CI result are unchanged.

## Candidate budget accounting model

`CandidateBudgetModel` checks proposed work across all eleven resource counters
and the operation, segment and bootstrap scopes. It is an ephemeral accounting
model: it has no persistence, trusted clock, actual-work meter, store authority
or work-execution capability. Recreating the model cannot be used to reset an
operational ledger. No Source, capture or replay API accepts its tickets.

The model copies explicit limits from a parsed candidate policy and requires
an explicit bounded ticket count. All additions are checked by subtraction
from the ceiling, including u64 maxima. Refused reservations change no state.
Each new segment consumes one segment count; existing segment histories remain
after failure. Creating a new segment cannot reset the bootstrap total.

Failed, cancelled and uncertain outcomes charge the full original work vector
and keep the full capacity upper bounds occupied. Repeating the identical
outcome is idempotent; conflicting outcomes refuse. There is no success/refund
or garbage-collection method and no caller-reported actual work is accepted.
Live/retained capacity counters are conservative projections, not measured file
sizes. Diagnostic values are copies and JSON cannot restore private state.
Owner checks reject copied models and foreign tickets. Finite scenario tests
cover all dimensions, exact limits, overflow, scope isolation, capacity retention
and concurrent reservations/charges under the race detector.

This arithmetic component does not complete durable Ticket/Settlement/BudgetRoot
storage, crash recovery, authoritative absence or operating qualification. Full
metadata accounting, owned work measurements, complete settlement, retained
clock/deadline checks and persistent CAS must be integrated and qualified before
protected work. The existing strict verifier and separate policy-activation
decision remain unchanged.

The retained standard block response in `testdata/standard-block-rpc.json`
was collected from the sole official RPC in run 40181518482, artifact
12593854612, `diagnostic-023.bin`. Its 99 dynamic-fee transactions report the
signed fee cap in the unsigned `gasPrice` field; 80 differ from the computed
effective price. This provider representation is retained as such, without
claiming Execution API conformance for that metadata. The adapter accepts
only the signed cap or the computed effective price, validates the signed
fee fields and transaction roots independently, and never treats this field
as a gas witness. A fixture test compares the full response with the existing
independent block RLP bytes; strict replay still requires its original witness.


### Complete gas results for failures before call hooks

The live check explicitly selects the existing `ReadReplayBlock` transaction-result
contract (`stateGasTracer`) before acquiring strict gas evidence. The request keeps
the same official RPC, block hash, two-second tracer bound and zero reexecution.
The selected acquisition issues one gas request and fails if that result is absent,
incomplete or unsupported. The root-only API remains available with its original
strict refusal behavior.

The retained `testdata/runtime-oog-root-rpc.json` is the exact `diagnostic-036.bin`
from run40181838571/artifact12594457975, SHA-256
`82f084b9df92012822b8a159b403bda6ddccb566bb1bec3f463550fa0ae51e6e`.
It has120 ordered roots, but index65's runtime out-of-gas result omits all three
dimensional fields. Its omission is preserved in a finite refusal unit. The
transaction-result regression uses the separate, unchanged complete native gas
fixture and retains that same failed transaction.

The pinned [inspector result conversion](https://github.com/paradigmxyz/revm-inspectors/blob/a983a3b69641c56ba070f1318a5e1834dafa525d/src/tracing/debug.rs#L212-L260)
recovers a call root when runtime gas charging fails before the call/create hooks.
The [recovery code](https://github.com/paradigmxyz/revm-inspectors/blob/a983a3b69641c56ba070f1318a5e1834dafa525d/src/tracing/mod.rs#L737-L769)
leaves the unknown execution spec unset; its
[call-frame builder](https://github.com/paradigmxyz/revm-inspectors/blob/a983a3b69641c56ba070f1318a5e1834dafa525d/src/tracing/builder/geth.rs#L217-L239)
therefore omits fork-specific dimensions. The transaction-result tracer instead
reads all four gas amounts directly from the execution result. This is a source
explanation matching the observed omission, not an attestation of the endpoint's
binary or routing; the advertised client version remains an unauthenticated claim.

The existing closed gas profiles, receipt deltas, reservation limits, floor/refund
rules, header totals, signatures, roots and caller checks are unchanged. Every
transaction still requires a complete witness; values are not inferred from an
out-of-gas label. Fresh live CI must establish whether the selected transaction-result
contract is supported at the official RPC. Source review and fixture success do
not establish actual provider qualification or Source readiness.

### Separate module acquisition from live checks

Run40205227654's provider-capture job failed during package setup, before any
test ran: the Go module proxy terminated HTTP/2 streams while downloading
`go-eth-kzg` and `fastssz`. The original failed artifact12603267628 is retained;
this setup failure is not a provider-capture result.

Both node-evidence jobs acquire and verify the locked modules before testing.
Only that single acquisition command selects the documented
[`GODEBUG=http2client=0`](https://pkg.go.dev/net/http#hdr-HTTP_2) setting.
Its JSON result, errors and verification output are retained with the artifact.
The setting is not exported to the tests. Existing module versions, checksum
verification, TLS verification, RPC selection, request limits and refusal checks
remain unchanged. There is no retry loop or automatic RPC fallback.


### Pending local retention across process exit

`CandidateRetentionLog` adds a separate bounded local record store for proposed
initial enrollments. It does not change the existing namespace or replay journal
storage, their lock lifecycle, the strict verifier or the policy-activation gate.
The Agent supplies the existing directory, namespace, anchor digest, expected tip
and explicit inventory/byte ceilings. There is no provision or default directory.
These values are comparison inputs; their equality authenticates no custodian,
absence witness, operating policy or selected namespace.

An explicitly pre-existing permanent `AUTHORITY.LOCK` inode uses nonblocking
exclusive advisory locking on Linux and Darwin. Open never creates or replaces
it, and Close never removes it. It is not an authority credential.
Other platforms return unsupported rather than providing unlocked access.
The opened directory and lock descriptor must match their selected paths before
and after prefix validation and append. Every cooperating writer must share the
qualified lock domain. Noncooperating writers, cross-host filesystems, mount or
backup rollback and replacement of the retaining authority require a separately
selected and qualified host/store boundary.

The closed initial record binds the namespace, sequence, previous tip, operation
ID, policy-independent Registry instance and canonical enrolling root. Each read
validates the whole bounded immutable prefix. Existing instances and operation
IDs cannot be replaced by choosing another policy, Source ID or candidate path.
A handle appends at most one record, synchronizes the file and directory, and
reads back the exact prefix before returning **pending external ACK** evidence.
This local empty-instance comparison does not establish authoritative absence.
The byte ceiling covers anchor and entry payloads; it is not a disk-allocation or
work-budget meter. No record is a Ticket, qualified root, Source or work token.

Normal reopen requires the supplied tip to match the complete local prefix.
If the external comparison pin is still old, only the identical final enrollment
with that predecessor, operation and identity can be re-synchronized. Its result
stays pending and permits no further append. A new expected tip is an external
selection input; this API does not authenticate or advance it. Production final
ACK requires a separately qualified custodian protocol with matching durable
local and external states. In particular:

- Before a write, no entry exists and no pending result is returned.
- A partial entry or synchronization error returns no result and fences the
  handle. No truncation, automatic reset or guessed acceptance is available.
- A complete local tail with the old external pin refuses normal reopen; exact
  recovery returns only pending local evidence.
- A new external pin with missing local bytes, a different operation or an extra
  tail refuses. Recovery never updates or rolls back the external pin.

Finite test-only unit fixtures cover malformed prefixes, stale/foreign reads,
limits, identity drift, uncertain synchronization and duplicate enrollment.
The owned-process runtime test performs a normal candidate append and process
exit with the lock open, then verifies OS release and exact pending recovery.
It operates only in a temporary fixture directory. These tests do not qualify
an actual operating filesystem, external custody or authenticated absence.
Durable eleven-dimensional work accounting, retained clock/deadline integration,
complete history and publication qualification remain required before operating
use; this component alone does not complete that integration.


### Proposed pre-request admission data

`CandidateEnrollmentRoot` and `CandidateAdmissionIntent` parse separately
versioned local candidate data. They do not change the existing V1 parsers,
SAGE0.10.0 messages, strict verification, operating format selection or policy
activation. No Source, RPC, retained-store or work-admission API accepts them.

The proposed V2 enrollment root represents an enrolling state with a budget
reference before a real replay checkpoint exists. A pure transition comparison
checks same identity, exact predecessor/sequence and supported status/fence
changes. It neither authenticates these roots nor verifies their referenced
evidence. Active shape requires checkpoint, budget and transition references;
parsing or comparing that shape grants no active Source authority. Generation
migration, owner transfer and authenticated reconfiguration remain unsupported.

The proposed admission intent references the original initial enrolling anchor
and a bounded original clock/deadline. It contains no future finality exchange,
ticket or later root, avoiding the first-request content-hash cycle. For this
candidate implementation the limits artifact is the exact canonical
`policy.limits` subtree and its ordinary SHA-256 digest. The deadline equals
the original earliest wall bound plus the explicitly supplied bootstrap
budget. Arithmetic overflow and unsupported signed-clock ranges refuse.
These are local candidate relations, not operating values or a trusted clock.
The copied clock diagnostic has a separate type from the trusted-input sample.

Both parsers require explicit bounded canonical input and the previously parsed
candidate identity. Missing/extra/duplicate fields, foreign context, ambiguous
JSON, invalid integer representations and unsupported transitions refuse.
JSON cannot reconstruct their private state, and returned bytes are copies.
Finite unit fixtures cover initial/intermediate/active shapes, transition
refusals, original deadline arithmetic, context binding and V1 isolation.

The remaining integration still needs versioned ticket/consumption/settlement
and budget records, qualified composite custody and durable one-use admission,
owned work/storage measurements, a qualified restart clock and complete history
and publication. The candidate data types alone do not permit a request or
establish any of those properties. Actual format adoption and the exact
manifest/generation activation decision remain separate from their tests.

`CandidateAdmissionTicket` checks proposed V2 reservation data against the
original intent, all eleven per-operation dimensions, both storage upper
bounds and a segment identity derived from the intent and ordinal. RPC records
always reserve positive retained space for their own records and consumption/ACK
metadata; local work follows the same rule. These are declared upper bounds,
not a measurement or qualification of the storage overhead. RPC records
name an allowlisted read method and input/parameter digests; local work has no
RPC method or request count. These checks do not inspect actual request bytes
or prove that a reservation covers their worst-case processing cost. Adjacent
ticket comparison rejects predecessor, sequence and segment resets. It does
not derive cumulative segment/bootstrap totals or establish a durable debit.

`CandidateConsumption` checks proposed pre-dispatch metadata against that exact
ticket and original clock/deadline, including bounded wall/monotonic differences.
Adjacent comparison also refuses clock rollback and the same adjacent ticket.
It is data validation only: no record consumes a ticket, returns a work permit
or invokes a transport. Whole-ledger uniqueness, custody commit/ACK, crash
recovery without redispatch and the owned executor remain required. The parsed
clock remains a candidate diagnostic type rather than a trusted clock input.
Finite inert unit scenarios exercise these boundaries; they are not actual
provider calls, host compromise simulations or operating qualification.

### Proposed bounded admission ledger

`CandidateAdmissionSettlement` validates proposed V2 terminal data against one
ticket. A complete outcome requires a bounded end clock, retained-result hash
and actual-work declaration within every reserved dimension. Failed, cancelled
and uncertain outcomes contain neither actual work nor a result hash; they may
close a reservation after expiry without extending its original deadline.
These declarations are candidate data, not owned measurements or retained ACKs.

`CandidateAdmissionLedger` is an ephemeral, owner-bound model of one complete
candidate-data prefix. Its first ticket may be local work or any parsed
allowlisted read. It does not qualify the actual bootstrap's first finalized
request, its exact method/parameters or owned request bytes. Those independent
checks remain mandatory in the later owned executor and bootstrap integration;
this generic model cannot substitute for them. It retains every ticket,
consumption and terminal state for
its bounded lifetime. A reserved ticket can be consumed once or terminally
charged without consumption. Complete settlement requires consumption first.
Any terminal state refuses further consumption or settlement, including reuse
separated by other tickets. Mutex serialization covers this model only.

Every outcome transfers the full original reservation from outstanding to
charged, even if a complete record declares less actual work. All eleven
operation, segment and bootstrap dimensions remain bounded. Charged segments
and both live and retained storage upper bounds are never refunded. Candidate
clocks must advance across record kinds; a retained expired sample prevents
new tickets or consumption while allowing later full-charge terminal recovery.
No sample becomes a trusted clock input.

Each append generates one fixed-shape ledger node and one budget root linked
to exact predecessors. The initial change also includes three explicit empty
anchors. Appends never clone, serialize or rehash the historical prefix. All
quota checks and new metadata preparation precede model mutation. Immutable
limits bound the lifetime to 4,096 records, 16 MiB of record payload and 64 MiB
of generated metadata; callers must supply smaller or equal positive bounds.
Individual generated nodes and roots are limited to 8,192 bytes. These model
quotas do not measure physical allocation or qualify an operating work budget.

`ReplayCandidateAdmissionLedger` reconstructs typed candidate records once in
order and compares every exact budget root. Payload and expected-root bytes
have separate preflight quotas; expected roots are a subset of generated
metadata and are not counted twice. A successfully created prefix remains
replayable with identical bounds. Any refusal returns no reconstructed model.
Finite inert unit scenarios cover terminal ordering, nonadjacent reuse, all
resource dimensions, storage peaks, clock expiry, quota boundaries, unchanged
state after refusal and generation-to-replay equality.

These candidate types do not persist records, authenticate absence, implement
custody CAS or ACK, issue work permits, dispatch requests or resume work after
a restart. Recreating a model loses state and cannot reset an operating ledger.
Qualified durable storage, independent custody, an owned clock and executor,
complete history, bootstrap and publication remain required. Existing V1 and
SAGE 0.10.0 formats, strict checks and operating-policy activation are unchanged.

### Private connected provider candidate

The private coordinator joins retained request bytes, admission, consumption,
the owned HTTP/TLS capture and independently reconstructed settlement. Its first
request is exactly the finalized-header read retained before admission. A later
closed Registry pin reuses that start and admits the chain ID, genesis header,
client version and Registry identifier reads individually. Each result is
checked before issuing the next read. Local parsing also has a separately
consumed ticket; successful HTTP alone cannot complete that local command.

The Agent owns this coordinator. A Registry Source denotes the Agent's state
observation boundary. HTTP message signing and selection of covered components
remain governed by the adopted messaging profile and the Agent's integration
logic. The evidence collector does not introduce a blanket message-signing rule.

All eleven work coordinates remain owned measurements. Execution and remaining
retention/readback envelopes are separate, finite reservations fixed before
work. Terminal outcomes retain the full original charge. Complete measurements
end before their own future settlement storage; that later work is covered by
the already reserved remaining envelope. Wire counts include both plaintext
HTTP directions, headers and transfer framing. TLS records and kernel traffic
have a different accounting boundary.

The separate private continuation draft preserves global ticket tombstones,
consumption and settlement sequences, original clocks and deadlines across
pages. A nonfinal page is full; only a quiescent final admission may seal a short
page. Page, seal and index metadata share one cumulative quota. Page maps and
codec scratch are preflighted before allocation. Index application reconstructs
the exact typed transition again; well-formed JSON and a matching hash cannot
clear terminal history. A pending seal permits only its single exact operation
and pre-seal index heads. These arithmetic/cache objects remain separate from
durable custody authentication.

V2 inventory, intent, commit and frontier data have distinct domains and exact
predecessors. The initial frontier has no future admission or budget reference.
A later enrollment root must immediately follow the existing one. Parsing these
records does not implement external current-tip CAS, qualify a filesystem or
restore an old permit. The V1 opener explicitly refuses a continuation profile;
there is no automatic transition when its existing complete-lifetime cap is
reached. Concrete V2 custody, full scans and full-history publication are still
required before this candidate can become an operating Source.

The V2 candidate also has an initial-only metadata store and a typed retained
delta writer. It opens only two preprovisioned distinct roots with exact empty
anchor/lock views. Finite setup reservation covers the initial high-water's two
copies before setup writes. Each later transaction checks complete physical,
resident, global and active-page bounds before retaining its immutable intent,
objects, inventory, pending record and commit. Independent replay reads those
originals, derives the frontier and combines custody, index and high-water bytes
in one common debit. The high-water's generated pending file becomes HEAD only
after exact predecessor checks; sync, readback and owned-clock checks surround
that replacement. All store/file/replay owners remain quarantined after an
uncertain transaction, and no reservation or partial file is refunded.

Closing a page checks both the nonclosing debit with its still-reserved future
page upper and the later exact page/seal debit. The private one-use file claim
prevents the already reserved high-water allocation from being charged twice.
Global original admission, clock, deadlines and ticket tombstones remain fixed.
The legacy whole-prefix store cannot accept this continuation profile.

These file and typed-writer results are retained metadata only. A candidate
empty view is not independently authenticated absence; a HEAD file or a private
pointer does not qualify a retaining principal. The opener cannot recreate a
live owner from a completed prefix. Whole selected-prefix initial, inspection
and final scans, qualified custody/clock/filesystem selection, the coordinator
acknowledgment boundary and complete history/publication remain required for
operating use. No candidate metadata result grants protected-message authority.

The tagged `TestProviderCaptureConnectedAdmissionAndRegistryPin` is a bounded
normal read-only probe against the sole configured Sepolia TLS origin. It uses
isolated, explicitly unqualified test stores, retains the original custody and
request/response bytes under `connected/`, and reports `source_ready=false`,
`policy_activated=false` and zero paid transactions. Its request ceiling is the
existing twenty seconds and its overall context is two minutes; failures are
retained without retries or endpoint changes. A successful probe establishes
only that joined admission/bootstrap/pin subject, not complete Registry history
or protected Agent message authority.

The provider CI job measures the new files directly in one tagged test run,
including finite units, normal owned TLS/filesystem checks and that official
read-only probe. It requires ninety percent statement coverage for each listed
source file and checks the list against the complete connected source inventory.
It does not union previous jobs, older sources or distinct operating subjects
to qualify a new implementation. Actual custody, clock, resource feasibility,
full-history readiness and policy activation retain their separate gates.

The fresh private V3 candidate profile adds explicit scan and cumulative
pre-admission limits. It requires separately provisioned initial scan high-water
and HEAD originals in both roots and their independently configured digest.
Adding scan limits never converts existing V1/V2 files or recreates missing
initial records. The initial setup upper is held once and includes the bounded
new original-file, directory and codec observations.

A candidate scan attempt holds its complete preparation and whole-profile body
uppers before parsing policy data, encoding control records or reading the old
control prefix. Each of the eleven work coordinates, retained bytes and clock
reads accumulates without refund. The two roots retain matching intent/claim,
commit and high-water originals before sequential HEAD replacement and exact
readback. Any uncertain result quarantines the owner and keeps the full hold.
The concrete codec counts repeated encoding, decoding, hashing and comparison
calls; caller-reported counters or a smaller body upper cannot qualify a claim.

This control writer implements initial-inspection metadata only. A one-use
bounded empty-prefix inspection now independently checks the original anchor,
setup and main high-water, reconstructs its empty state and exhausts the exact
two-root catalogs under the same held claim and original clock. Every target
reread contributes its actual bytes and validation count. An unknown file,
different target or insufficient final-record/catalog upper refuses without
refund. The result is private diagnostic data. Before E0 the empty index
anchors have no retained inventory originals, so the result deliberately cannot
claim to be a retained ScanChunk, settlement or execution acknowledgment.

Full main/history/recovery scanning, retained command/chunk completion, the
independently qualified root principal and post-admission coordinator
acknowledgment are still required.
Neither a candidate prepared attempt nor two matching directories is an actual
irreversible retaining service, complete inspection or permission to execute
an Agent operation. Recovery cannot reconstruct such permission from JSON.

The independent pre-admission replay instance keeps its already claimed body
meter separately from the producer's creation-time setup observation. Typed
enrollment, admission, ticket, consumption and settlement parsing observes each
actual input and repeated policy decode, standard closed-schema encoding,
ordinary or domain-separated hash, and retained string copy. The scan object
reader also counts the domain NUL and its one actual format parse; the digest
helper hashes without parsing. Repeating an object read counts those invocations
again rather than reusing a previous diagnostic.
Existing public DATA parsers retain their exact bytes and nil-observer behavior;
they cannot select a scan meter or grant an acknowledgment. An inactive or
different body observer refuses before typed replay. This does not yet close
all index, page, setup, high-water and retention codec paths, nor implement the complete
main-prefix scanner or its command/chunk/settlement bridge. Those paths must be
observed before any complete scan-work or operating qualification claim.

Frontier record, intent, commit, object and inventory codecs also have concrete
observed variants. Independent replay uses them to reproduce existing bytes,
including repeated root/inventory decoding, canonical encoding, sorting,
comparison and domain-separated hashing. A consumed allowance refuses before
the next primitive; it never returns a partial record as success. Nil-observer
DATA callers preserve the existing V1/V2 schema and digest contracts. These
additional observations do not confer custody authority or close the remaining
ledger, high-water, full-prefix and operating qualification requirements.

The private independent replay also attaches its original live observer to its
own ledger and index models. Every repeated node/budget encoding, policy decode,
index value re-encoding/comparison, page/seal rebuild and retained-object digest
uses that same observer. A refusal cannot install the partly built ledger or
index change. Public candidate construction keeps a nil observer and its original
4096-record contract; private metadata and page encoders retain their existing
8192-byte and 4 MiB ceilings. The closed page encoder ceiling is not an increase
of a store metadata limit or the public lifetime cap.

These hooks remain incomplete integration work: setup/high-water/retention and
request/result parsers must be observed too, and a complete original catalog
reader must drive this independent model. An arithmetic unit fixture or a
metered page seal does not qualify an actual principal, acknowledge custody,
restore execution, activate policy or make an Agent Source ready.

The same independent observer now covers the repeated setup/profile/high-water,
retention-rederivation and old-material checks as well. Finite units compare
ordinary retained E0/first-ticket files under independently rebuilt ledger and
index state. These tests still do not supply a full original catalog reader,
retained scan-command/chunk/settlement chain, full request/result observation,
selected host authority or any operating acknowledgment.

Request rebuilding and retained response reconstruction have concrete observer
paths as well. They preserve the original request parts, observe the RPC token,
envelope and version JSON passes separately, and observe base64 decode/re-encode,
header parsing, standard HTTP framing and nested RPC validation. Observed HTTP
framing receives the actual original wire length from the retained-byte reader;
it cannot accept an unknown caller-reported input size. Failed header reads keep
the successful byte-read/copy work already observed. Finite tests use ordinary
memory bytes and the existing normal owned TLS fixture; no hostile peer is added.
The remaining pin/ABI-specific parser paths still need the same integration,
and none of these parser results is a scan completion or custody credential.

The independent pin path uses the same observer for rebuilding all four exact
requests, reading the child settlement and its policy, base64/RPC validation,
and the pinned ABI JSON-string, hex and fixed-word decoders. Revalidation repeats
these charges. The observed lower-case hex check scans bytes without claiming a
copy when a canonical lower-case value requires no allocation. Pure nil callers
keep the original public/legacy accounting contract. The ordinary pin parser
unit uses references selected from its preexisting fixture only as DATA inputs;
it is explicitly not the full independent-map catalog inspector still required.

The private original-file replay reader creates fresh object, operation,
binding, setup, position, ledger and index state under the already claimed scan
body observer. It reads structural setup closure from both roots, then reads
each main intent/pending/commit/frontier, V2 inventory, new object original and
all typed transitive/high-water originals. Setup `expected_tip` obligations are
closed against reconstructed main positions, rather than accepted during the
first structural pass. Policy parsing observes its two subtree encoders, copied
raw bytes and three domain hashes. Neither the reader nor its lineage predicate
is a complete scan: whole-record chunk reservations, raw/catalog accounting,
both catalog EOFs, retained command/chunk/settlement receipts and the private
scan-to-first-admission binding remain integration requirements. The finite
owned-file tests do not qualify an operating host, clock or retaining authority.

The separate pre-admission body joins that reader to one claimed cumulative
meter, bounded nonempty setup/main windows and both exact directory catalogs.
It keeps successful raw target reads, including repeats, separate from catalog
name work and separately checked live scan-control files. The final main
window reserves the entire finite catalog before its first target record read;
both EOFs and every required name must agree. Chunk encoding occurs after the
work snapshot and spends the already held output upper. The returned chunk
bytes remain unretained candidate DATA. This path requires existing E0/index
anchor originals and rejects a post-I target: it cannot supply an admitted
post-I command/S chain, the first-I scan binding, an ACK or operating readiness.

The optional V4 candidate profile fixes a separate finite first-admission
envelope before H0. Its canonical live-accounting rule derives live occupancy
from the same committed upper that includes all retained and temporary copies.
Opening observes and caches the policy capacities under O; a later scan claim
checks all bootstrap work coordinates and both disk capacities before holding
P+B. B includes the complete first-E1 envelope, so a future ticket cannot fund
its own uncertain creation. Old profiles retain their original bytes and cannot
be adopted as V4. Private ledger arithmetic can seed the original pre-I charged
work and both disk-used totals before the first ticket; exported constructors
still start at zero and keep the original lifetime cap. These primitives are
not yet the retained chunk/binding/typed first-E1 writer, an authenticated
restart, a qualified physical bound or permission to send Agent messages.

V4 additionally commits its receipt-accounting revision in the profile before
H0. Its command/chunk V2 records name the exact selected scan HEAD, high-water
and attempt, and retain two immutable original copies with observed readbacks.
Earlier originals are re-read before each next nonempty record. The final
record reserves both full catalogs before starting: one precedes its chunk
snapshot, and the other checks the exact final filenames after retention.
Later encoding, writes and the second catalog still spend the original held
meter even though that work is outside the chunk's earlier diagnostic snapshot.
Inspection cannot borrow the separately held first-E1 work or clock allowance.
Output events stay in the same cumulative actual vector, while their observed
delta is excluded from the target-only subwindow. Starting the next nonempty
record clears only that subwindow delta, never the cumulative meter or its
upper. Command retention therefore cannot make a later target read spend its
write allowance twice, and unused output funds cannot extend target work.
The four readbacks and two receipt decoders have separate output accounting;
old profile bytes cannot silently acquire that revision. Receipt ownership is
live and nonserializable. Retained DATA still needs the mandatory first-I
binding, independently seeded typed E1 replay and qualified custody boundary;
the owned E0 candidate does not provide those operating guarantees.

The fresh V4 bootstrap-accounting revision keeps the original open-O meter
through E0. The first-admission bounds format is separately pinned as V2
before H0, so an earlier candidate identity cannot acquire this path on reopen.
The live owner can enter enrollment once, before any scan claim. It compares
the complete E0 work, resident-memory, clock and physical-file plan with the
remaining original O before allocation or writes. Its seven new objects are
the enrollment, three independent index anchors, original setup/commit, and
one operations-index mutation; inventory, intent/pending/commit/frontier and
the two new head copies bring the physical plan to fourteen new files.

The physical allowance includes the uncertain pending head. The writer
consumes that one private phase claim without increasing the already held
store total. Exact originals are read back and independently replayed with
different coordinator/custodian index maps under the same O. Every newly
observed byte/filename copy and serializer spends the current phase, while
earlier profile filename bytes and copy-accounting behavior remain unchanged.
The bootstrap meter closes permanently on completion or failure; losing the
owner or encountering an uncertain original cannot renew O or refund its tail.
Only a completed E0 can begin its separately funded first inspection. These
are candidate filesystem and accounting checks, not authenticated absence,
qualified durability/clock, an ACK, first-I/E1 binding, or Agent-message use.

A terminal V4 inspection now prepares one private finishing owner while the
original store mutex and B meter are still held. Its bounded immutable target
copy is charged before final IO, and the handoff follows both final catalogs,
lineage closure and the final owned sample. No IO follows that handoff. The
normal return still closes the inspection body and detaches the active meter.
Starting finishing requires the exact unused store-owned object, original
meter/counters/uppers, predecessor pins, setup origin and unchanged profile.
It marks the owner started before any primitive and reattaches that same B
without reactivating inspection or receipt APIs. Finishing work and clock
deltas must fit the already held first-E1 envelope as well as cumulative B.
Owned samples retain the original deadline and setup duration; the finishing
duration also includes any wait after inspection. Any failed or uncertain
finishing primitive permanently closes the owner and retains the full hold.
There is no success/ACK path yet: mandatory first-I binding, independent
seeded typed E1 replay and qualified custody remain separate work.

The closed scan-admission binding format names only existing E0, setup, scan
head/high-water/attempt, target, terminal chunk and admission digests. It binds
the original clock/deadline and exact open, cumulative scan and total uppers;
live occupancy is derived from the same committed retained upper. A metered
reader reconstructs H0 through the bound historical H_k from both original
copies and every original intent/commit. It recalculates the held sum and
policy capacities, paying again for each repeat read/codec/hash/copy. A later
provisional current scan head cannot substitute for that historical evidence.
The returned seed is only arithmetic DATA: target/chunk/I/E1 validation,
mandatory typed object integration and current-pin qualification are still
required before any concrete finishing or custody result can be accepted.

The separate receipt reader follows the immutable terminal chunk backward to
index zero, reading both original command/chunk copies at each record. It
checks the exact historical attempt/target, command bytes, contiguous cursors,
counter differences, cumulative work bounds and clock order, and requires
setup zero with zero prior totals. Memory does not grow with chain length;
each repeated read and codec spends the current owned meter. Finishing also
takes an owned sample after every bounded record. The returned diagnostic
closure and once-counted schema byte total are DATA: they neither authenticate
past physical work nor replace frozen-target reconstruction or typed E1.

First-admission finishing replays E0 from fresh original setup/main maps under the same held inspection meter. It drops its closed inspection caches while retaining the full returned-output RAM upper, checks the exact original main/scan heads and samples the original clock throughout. This supplies candidate typed inputs only: the first admission, mandatory binding, seeded E1 write, acknowledgment and operating Agent integration remain incomplete. Finite missing-original, budget, clock, owner and RAM units do not qualify an operating filesystem or principal.

For the V4 first typed admission, one mandatory scan binding names the independently replayed E0 boundary and original historical Hk/command/chunk chain. The original O+Q seed enters bootstrap charged and disk-used totals before the first ticket append. Both original copies of scan metadata are charged once to the separate common metadata quota; old physical files are not rewritten or charged as fresh E1 files. Pure candidate DATA units compare seeded B0 and reject a zero-seed expected record. They do not implement or qualify the still-pending prepaid first-E1 writer, custody acknowledgment or operating Agent. Legacy/public zero-seed behavior is unchanged.

Common store samples during first-admission finishing now route through that
same live owner, applying both its original deadline and its finishing limits.
Foreign views cannot sample its clock. Original-byte/text comparisons pay for
their copies and comparisons under the current meter. First-finishing inventory
sorting has a concrete bound of n*(n-1)/2 comparisons, with two observed key
copies per comparison; earlier sorting and event contracts remain unchanged.

The first-E1 storage quote separately accounts for14 fresh objects and all21
possible file allocations, including the uncertain pending head. Its binding
allowance conservatively uses the configured metadata cap even though that
codec actually enforces both this cap and8192 bytes. This is an intentional
overestimate, not a correction of an earlier64KiB underestimate.
This quote is only the physical-file component. Complete work, resident-memory
and clock preflight, the prepaid writer, original readbacks and operating
qualification still remain; a quote or successful unit does not authorize them.
