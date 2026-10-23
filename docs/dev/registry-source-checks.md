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
