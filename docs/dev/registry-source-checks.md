# Registry Source clock and checkpoint checks

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
