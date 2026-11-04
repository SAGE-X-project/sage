# Registry history batch candidate: schedule diagnostics

This implementation is an experimental, pre-acquisition diagnostic inside the
Agent's SAGE library. It does not enable a new Source, amend a protocol or
Inspector verdict, select an operating policy, create a key or send a request.
The existing `CollectReplayBatch`, per-block observations, strict gas and
authorization checks, and their result labels remain unchanged.

The user selected implementation and validation of the separate
`batch-observed-v1` proposal. This proposed temporal contract observes chain,
genesis, Registry and client identity at bounded batch boundaries. It preserves
complete block/receipt/gas/call/caller/signature/history reconstruction, but
does not separately observe transient identity/finality changes after every
interior block. Qualification and any eventual operating selection must retain
that difference. There is no automatic fallback from the existing collector.

## What the diagnostic establishes

An explicit proposal binds the complete deployment-to-maximum-selected span,
batch size, elapsed window, finite clock allowance and finite T/C/S record
allowance. The span includes deployment. All ranges are consecutive and bounded;
range lookup is DATA, not a validated cursor or evidence that an earlier range
was accepted. A changed bound produces a different diagnostic binding.

For B complete blocks and W batches, the candidate has at least three content
RPCs per block, four fresh start plus seven fresh end observations per batch,
and one separately funded comparison operation per batch. Thus RPC >=3B+11W,
the existing successful-RPC backend clock floor is >=8(3B+11W), and T/C/S records
are >=3(3B+12W). These are only lower bounds. The existing per-block schedule's
reference floor12B-1 includes its separate parent traversal; it is not changed.

For the previous observation's34206-block span, a16-block proposal has2138
batches,126136 RPCs and1009088 backend samples before other work. A32-block
proposal has1069 batches,114377 RPCs and915016 samples. Neither is a demonstrated
feasible profile. These figures describe that retained span, not today's height.

Every report fixes `complete_costs_qualified=false` and `operating_ready=false`,
including proposals whose lower bounds fit. A floor already above the unchanged
ceiling is sufficient to reject feasibility; a floor below it cannot approve
anything. No clock, record, resident-memory or operation ceiling is increased.

## Still required before a complete program

The separate actual program must close all initial/final identity and state
reads, dependent trace/code/history calls, separately funded local inspections,
all eleven resource coordinates, G custody costs, pages/indexes/catalogs,
physical storage, native allocations, four simultaneous replay models,
non-immediate-GC peaks, failed stages and final C0/Dp/C1 inspection. It must hold
those complete costs before H0, use the original epoch/start/deadline, and refuse
an unsupported complete span without dropping blocks or choosing a new bootstrap.

Each resource-stage ACK can fund only the next enumerated acquisition stage.
Only a separately retained complete comparison may yield a private validated
candidate cursor. Neither that cursor nor the last batch grants Agent authority;
complete ancestry, final originals, freshness and separately qualified selected
environment/policy are later mandatory gates. The diagnostic implements neither
ACK nor cursor and cannot be used as an execution or publication credential.

The selected demo preparation target is the current Mac/current user with a
repository-external dedicated data directory. Host, clock, filesystem and
custody qualification remain actual evidence obligations; selecting that target
does not itself activate a policy or authorize new registration transactions.

## Bounded retained-original reader candidate

The private ordinary-content reader checks an exact request, parameters,
response or TLS object against its declared length and ordinary SHA-256 digest.
It opens only the existing content-derived filename within the selected
retention root. It checks the opened file and path identity/size before and
after reading, probes EOF, checks the directory/lock identities, and refuses a
different digest. It returns a digest only. It does not parse the object's
schema or validate command, identity, caller, gas, history or publication.

Each reader owns a reusable 8,192-byte buffer, a one-byte EOF buffer and SHA-256
state. Calls are serialized. After valid preflight, an I/O or cancellation
failure poisons the reader; an invalid plan or missing context is refused
before I/O. Two independent
replay owners must have separate readers and independently read the originals.
Reusing a workspace resets only its hash state, never an authority budget,
clock epoch, consumed permission or checkpoint. Creating a reader is not
enrollment and cannot restore a failed execution permit.

For an admitted original of N bytes (1<=N<=32MiB), one scan has these source
call bounds:

| Quantity | Upper |
|---|---:|
| Explicit reusable byte buffers | 8,193 bytes |
| Bytes requested from the file, including EOF probe | N+1 |
| Bytes supplied to SHA-256 writes | N |
| Chunk ReadAt calls plus final EOF probe | ceil(N/8192)+1 |
| SHA-256 Write calls | ceil(N/8192) |
| Non-read filesystem calls | 14 |

The fourteen calls are two four-call directory/lock identity checks and the
object's Lstat/Open/Stat/Stat/Lstat/Close. Counts refer to Go source calls, not
kernel retry counts or qualified elapsed time. The byte-buffer term excludes
path/filename, file and metadata objects, hash-native overhead, returned digest,
owner/model/custody state and allocator behavior. Those need separate whole
program bounds; do not assume immediate garbage collection between scans.

The prospective upper remains unchanged after failure. This helper has no
hold, refund, dispatch, ACK, clock-sampling or Source-ready API. Its caller
must separately fund all work and bracket it with the original qualified
clock before use. Existing collectors and all of their checks and resident
quotes are unchanged. The grouped execution owner, full numerical certificate,
whole-history cursor and final C0/Dp/C1 inspection are still required.

## Pre-input byte accounting candidate

The new private input guard copies a fixed, disjoint array of event-class
ceilings. Each row fixes a maximum input width W, cumulative input bytes U and
invocations J. Under a separately proven per-invocation width bound, the
existing codec/encode metric is bounded by W*U+J; linear byte events use U+J.
Zero-byte invocations still consume their own call. Arithmetic, cross-class
sums, cumulative bytes and invocation counts are checked before input effects.
An incomplete, exhausted or failed instance cannot be resized, refunded or
restored from JSON. Multi-class claims are consumed atomically.

The original-reader adapter derives its claim from the retained reference,
consuming N+1 read bytes with ceil(N/8192)+1 calls and N hash bytes with
ceil(N/8192) calls before invoking the concrete reader. Every underlying input
is at most8192 bytes; the final read is the separate one-byte EOF probe. A
second read of the same original consumes a second claim. Failure retains the
whole attempted claim and closes the input-guard instance. The guard serializes
its debit with the actual scan; it does not accept an input-operation callback.

This is prospective input accounting, not measured actual work or a retained
whole-program hold. The private constructor can create an accounting instance;
creating another instance does not restore operational capacity. No current
Source, transport, custody owner or meter accepts it as authority. The future
grouped owner must derive its immutable rows, create exactly one original
instance per independent role under its previously retained full hold, bind
the original program/epoch/stage, and consume its own one-use member permission
before dispatch. Those connections are not implemented by this helper.

The adapter covers read/hash INPUT rows only. The fourteen filesystem metadata
calls, digest/path construction, allocator/native costs, owned clocks, storage,
semantic replay, failure custody and final scans remain separately funded
whole-program obligations. In particular the existing per-RPC full-width
reservations are unchanged: this guard cannot discount or replace them.
Neither this component nor a positive scalar envelope supplies a sufficient
whole-history certificate, a group ACK, a cursor or Source readiness. The
closed grouped permission/custody source and complete cost derivation still
have to close together before whole-history acquisition.

## Ordered group request table candidate

A new private preparation table fixes the program identifier, finite purpose,
stage ordinal and each distinct member ordinal, exact method, reconstructed
HTTP request/params identities, symbolic method class and all selected capture
limits. Its copied fixed array has at most32 members; canonical metadata is at
most64KiB. These are candidate preparation bounds, not a selected operating
profile or proof that the corresponding full group fits a custody commit.

The constructor rebuilds every request from bounded original parameter DATA,
checks its complete HTTP bytes and both digest contracts, and derives the class
again. Equal numeric caps never substitute for a method class. Caller pointers
and slices are not retained. Identical requests occupy separate ordered slots
and therefore remain separate future permissions and resource invocations.

An explicit parser reconstructs a table against the independently supplied
original requests and fixed expected program, purpose, stage and method bounds,
and compares the exact canonical original metadata. It does not accept a JSON
role or a digest-only claimed acknowledgement. The table domain is a private
candidate record format, not a change to the Agent wire protocol or RFC9421.

Both construction and parsing return DATA only. Neither validates that a
request is the next command allowed by the whole-history grammar. The future
program must validate that grammar and its response dependencies against prior
originals, fund all work beforeH0, retain independently acknowledged T/C/S,
consume all members before dispatch, and quarantine failure under its original
hold. Those duties, the complete numeric costs, qualified clocks, independent
readers and final C0/Dp/C1 are not provided by this preparation table. The private first-group owner uses the table in its retained T and C records,
but exposes no Source or network-permit API. The existing per-RPC path and its
full-width reservations remain unchanged.

## Ordered member bookkeeping candidate

A separate private counter models irreversible all-member consumption, a
separate consumption-confirmation step and one in-flight member in exact
ordinal order. A valid group has one to32members. Every member bit is consumed
before the first begin. Completed ordinals cannot be repeated, skipped or
reordered. Invalid transitions and explicit failures retain the consumed and
dispatched masks and close the counter; they cannot advance or reset it.
A normally closed counter preserves its complete history. Copies and JSON
records cannot recreate the original counter.

This remains serialized in-memory bookkeeping, not transport authority. The
private first-group writer now creates its original counter only after replaying
an actual retained T, consumes every member before writing C, and confirms it
only after independent C readback, physical HEAD replacement and the original
end-clock checks succeed. Counter confirmation precedes the single serialized
logical pin/model/ACK publication. A failed or uncertain write preserves every
consumed bit and the complete admitted upper while quarantining the same owner.
A copied ACK or counter never reinstalls a writer or renews a window.

The C path reconstructs the initial model from its selected immutable original
pin and replays the original request pack, T, inventory and framing under the
current original observer. This does not move today's HEAD back, restore a
closed inspection cache, or revive an earlier observer/clock role. Its permitted
catalog derives from the initial selected prefix plus the acknowledged T delta;
it does not accept directory contents or cached allowed-name maps as authority.
C uses the unspent suffix of T's exact group work/clock window and the same
pre-observation lifetime hold. The independently replayed T clock remains the
elapsed-time anchor through C; a new short C bracket cannot renew its five-minute
group interval. It cannot select fresh scalar limits.

Ordered transport execution, result-original semantics and successful/failed S
retention remain required. A concrete executor must bind its original C ACK and
counter before any member can be dispatched, preserve the exact ordinal order,
and verify independently retained S before closing. No current transport accepts
the private C ACK or its confirmed bookkeeping as a network permit. The complete
source-derived work, native memory/live-set, storage, clocks and failure/recovery
certificate and its production factory remain unimplemented. The provisioned
unit fixture is not operating qualification or a whole-program cost proof.

Creating a second DATA counter cannot restore operational capacity. Program,
table and original-instance identity must be bound and checked by the actual
parent. Source-derived transition, string-validation, native-memory, clock,
retention and failure costs still belong to the uncompleted whole-program
certificate. The counter itself implements none of those effects or checks,
and its finite state tests are not proof of durable group execution, whole
history, final publication or Source readiness.


## Committed group replay preparation

Within the unfinished dispatch and settlement bundle, an original C
acknowledgment can prepare two independent initial/T/C replay models without
rewinding the current phase to T or replacing the current C pin, last ACK or
models. The initial namespace and exact T/C file deltas are derived again;
discarded catalog caches do not select allowed names. Every retained request,
T/C object, inventory, frame and immutable highwater is read under the current
C HEAD. The original observer, group work window, native clock ceiling and T
time origin remain cumulative. No new hold, counter, clock backend or refund
is introduced. Failure after entry quarantines the original consumed state.

The private preparation object is not an execution permit, retained S or a
completed group. Actual ordered connection attempts, bound response/TLS
originals, S and terminal recovery, complete eleven-coordinate costs and peak
resident memory, and the production derivation factory are still required
before collecting the complete history or selecting an operating Source.
Source review, actual unit/runtime execution and CI evidence are separate
gates; preparing the code does not satisfy them.


## Ordered result original DATA

A separate bounded parser reconstructs each original HTTP/RPC envelope and
its canonical TLS-identity record from the selected binary result pack. The
retained T supplies the exact request table and capture class; the enclosing
settlement replay must select the completed prefix. Member digests are derived
from actual original response/TLS hashes plus program, stage, table, T, C,
request and ordinal bindings, including distinct identities for equal wire
bytes at different ordinals. A shorter prefix cannot become complete S.

TLS records retain the existing bounded verified-certificate digest chains and
selected roots artifact; parsing these records is DATA validation, not a new
certificate or live origin attestation. The concrete closed capture owner
still must bind the real verified connection and retain its actual originals.
HTTP/RPC framing and id validation do not prove method-specific chain/block/
execution semantics. No result DATA can issue a permit, publish S or return
a protected Agent result. Those connections and full cost/live-set derivation
remain unfinished. New scenario and normal owned-TLS tests are prepared; no
execution or CI acceptance is implied by this source proposal.


## Consensus content primitive observation

The private replay budget can carry the original reader's provider observer.
The existing canonical binary content decoder then records each actual header,
transaction and receipt decoding, canonical re-encoding, byte comparison and
entered block, transaction and receipt-log count. Complete-input evidence
admission remains distinct from actual entered processing. The nil-observer
DATA API keeps its existing commitment path and strict native gas checks.

The observed path derives the same Ethereum indexed Merkle Patricia roots
from the already canonical binary values. A closed private implementation
observes every concrete indexed key and compact-path encoding, node encoding,
byte copy and Keccak invocation, including inline nodes. No caller-provided
trie callback, generic live-RPC adapter or reported aggregate actual vector
selects these observations. Indexed keys are derived in their RLP lexical
order; root nodes are always hashed and small non-root nodes remain inline.
The pinned Ethereum library provides the independent expected commitments
in finite in-memory tests, including index127/128 and255/256 boundaries and
the existing recorded deployment content. These tests have not run yet.

This is logical content-processing observation only. JSON reconstruction,
native gas and execution-trace interpretation, ABI/command/signature semantics,
the independent original S content/effect/state/history transition and the
complete source-derived cost and resident-memory certificate still need
connection. The generic collection admission gate remains closed. This source
proposal does not collect history, authorize publication, activate a policy
or demonstrate new CI or operating qualification.


## JSON, native gas and call-tree observation preparation

The strict gas checker and top-only root normalizer now pass the same optional
original observer to their bounded JSON object/string/hex/quantity/array
primitives. The call-tree normalizer has an explicit private observed entry;
the existing exported DATA API delegates with a nil observer. No new live RPC
facade, tracer fallback, alternate profile or operating credential is added.
Required fields, native gas settlement and complete frame/log interpretation
remain the existing contracts.

Arrays allocate once within both the selected count and the possible count
of their raw JSON bytes. Their owned initialized slice storage and actual
element-header copies are recorded, including the copies after parsing an
earlier element. Frame, log and topic arrays also have fixed allocation and
copy paths. Entered frame/log counters are distinct from the existing complete
input count admission. A failed observer cannot select a nil-observer retry
or return a successful partial normalized tree or root witness.

Header and transaction hashes are reconstructed from canonical binary bytes
instead of treating library hash caches as fresh evidence. The binary header
preflight ceiling is64KiB; transaction hashing requires an already decoded
complete block within the existing64MiB evidence ceiling, including that full
encoding preflight headroom. Blob sidecars are excluded from transaction
hashing, without removing them from the original object. CREATE addresses use
the same sender/nonce RLP and Keccak contract. Lower-case address text avoids
an unnecessary intermediate EIP-55 checksum.

Finite unit scenarios compare retained120-transaction gas/root witnesses and
recorded Erigon/Nethermind call trees with the existing DATA path. No new test
installs a transport or responder or sends inconsistent data to a service.
The candidate CI selector is widened to retain every TestProvider case and
also run the existing node/replay/root-gas/diagnostic regression families for
the added legacy-file subjects. Exact inventory equality and each file's90%
requirement remain mandatory. No new test or coverage result is implied.

This is still a partial, unpublished implementation bundle. Full JSON block/
receipt typed reconstruction, signer recovery/cryptographic internals and gas
floor observation, ABI/command/actual-caller derivation, semantic state/index
copies and both original S content/effect/state/history transitions remain
unfinished. The sufficient whole-program11-coordinate bound must include the
new primitive preflight maxima and repeated prefix/native backing-memory,
clock/lifetime/failure/recovery/final publication costs. The generic collection
admission gate stays closed; no Source or policy is activated.


### Typed JSON block and receipt observation

The unpublished connected bundle now carries the original observer through the
existing post-Amsterdam JSON block and receipt reconstruction. UTF/unicode scans,
strict token parsing, required fields, bounded nested lists, canonical field
comparison, complete receipt locations and binary output encoding use that same
observer. Original nil-observer DATA entry points keep their existing checks.
Typed block and transaction entries and receipt log entries are counted where
they are actually entered; later binary validation is a separate entered pass,
not a reason to erase the earlier work. Budget refusal returns no block bytes
or partial receipt list and cannot reset the failed observer.

Canonical header/transaction comparisons build the pinned typed field values
with an explicitly observed fresh commitment hash. They do not wrap the library
JSON marshaler that hides a cached or fresh hash call. The field contract follows
go-ethereum v1.17.5 `gen_header_json.go`, `transaction_marshalling.go` and
`gen_withdrawal_json.go`, including null-only omission, the explicit-v yParity
alias and the existing dynamic-fee RPC gasPrice distinction. The closed reader
uses an output preflight of twice the original JSON length plus 65,536 bytes for
canonical JSON, and the full original JSON length for its consensus binary
encoding. Actual encoded lengths are recorded separately and checked against
those bounds. These maxima are not actual counters or a production certificate.

The observed path builds the transaction and withdrawal roots from those
canonical binary values with the private explicitly observed Ethereum indexed
trie; the nil path retains the library commitment calculation. No missing
receipt, changed location, transaction hash, required header field or gas witness
is repaired or inferred. Original responses remain the retained evidence.

Normal recorded block/receipt data, all supported typed transaction JSON forms,
decoded JSON structural comparisons, exact comparison quotas, missing fields,
short original budgets and no-partial-output scenarios have source tests.
Finite empty and one-blob sidecar hash subjects additionally cover cold and warm
size caches. These inert codec tests send no transaction or message and assert
no valid blob proof or signer authorization. Tests remain NOT_RUN and each-file
coverage remains UNMEASURED until the complete bundle reaches new native CI.

This is still partial implementation. Signature recovery, gas-floor internals,
ABI/command and actual-caller verification, accessor/header/body/map allocation
and native codec costs, complete semantic state/history copies, both original
S transitions, sufficient whole costs, RAM/lifetime/failure/final-page rules and
the production factory remain required. Neither this JSON connection nor the
old green CI opens the closed admission gate or qualifies an operating Source.


### Fresh transaction sender and fixed gas-floor observation

The same unpublished bundle now routes its original observer through sender
recovery in the typed JSON reader, root-gas context, strict gas floor and
execution-root caller check. The nil DATA path keeps the pinned library signer.
The observed path independently builds the exact Sepolia signing preimage for
legacy, access-list, dynamic-fee, blob and set-code transactions, including the
unprotected legacy six-field form. It preserves chain11155111, recovery parity
and low-S requirements; it does not trust a JSON sender or a warm sender cache.
This does not expand the existing execution replay transaction-type gate.

RLP output is preflighted at the existing complete64MiB transaction bound plus
256 bytes of fixed signing-field overhead and checked after encoding. Type
prefix copies, actual Keccak input, the65-byte signature encoding, the fixed
97-byte hash/signature recovery codec input and the64-byte public-key hash are
observed at their concrete calls. The Proofs coordinate is not repurposed as a
signature count. Recipient/value accessor-copy primitives and actual access-list
metadata traversal are separately observed in the unchanged gas-floor rule.
The fixed native recovery and accessor internals still require sufficient
whole-work/native CPU and backing-memory qualification; these primitive events
are not that certificate. The original mandatory gas witness remains required.

Recorded120-transaction senders and fixed gas-floor results are compared with
the pinned library even after its sender cache is warm. Typed signing preimages
are independently compared with the pinned signer for all supported forms.
Short original decoded budgets retain earlier work and return no account. The
full observed JSON reader also has positive effective-price/fee-cap tests with
yParity omitted and explicit v retained, followed by rejection of a changed
price and conflicting parity. All added tests remain NOT_RUN.

ABI/event/command verification, full execution-frame and receipt comparison
observation, semantic state/index/history copies, both actual original S
transitions, whole costs/RAM/lifetime/failure/final-page and factory integration
remain required. No Source, dispatcher, ACK or operating policy is qualified.


### Normalized execution, frame and receipt observation

The same original observer now covers the normalized execution-root binding,
recovered transaction sender, root input/value/recipient, recursive initiating
context and ordered emitted logs, and complete log/address/data/topic comparison
with the committed receipt. The existing execution type/protection/chain gate,
parent-context meaning, static propagation, failure propagation and eligibility
rules remain. A direct normalized DATA fixture is not a qualified EVM execution
or evidence of complete RPC call/log coverage. Both original readers must still
authenticate, retain and interpret that raw coverage themselves.

The observed frame walk uses one explicitly bounded log accumulator shared by
its recursion. It starts empty, grows to at most16 slots then doubles within
the remaining log ceiling, and separately observes the new storage, copied old
slots and each written slot. It never concatenates a child result into another
full parent prefix. Payload/topic clones and address decode/conversion use the
original observer. Frame pointers preserve the existing borrowed DATA meaning.
Failed quota/context/shape/receipt checks return no partial log slice; a failed
observer cannot resume with another frame. Empty successful output stays an
empty nonnil slice.

Finite unit subjects compare ordering/context/eligibility with the legacy DATA
walk, exercise exact and short original work vectors and explicit growth copy
counts, and compare execution roots and single changed fields with the existing
checks. Sender rejection cases now include invalid parity, negative/oversized
scalars, high-S, zero/range values and finite native recovery errors against the
pinned library. Gas-floor arithmetic has separate direct DATA cases for create,
self-recipient, nonzero value and access lists without creating a new key. The
actual gas-floor reader always recovers its sender before calling that private
arithmetic helper; its DATA test inputs are not sender credentials.

These tests are NOT_RUN and each-file90% is UNMEASURED. Native crypto/codec and
allocator/profile qualification, ABI/event/command verification, semantic
state/index/history copies, both original S transitions, sufficient whole
costs/RAM/windows/lifetime/failure/final-page and factory remain required. The
generic collection admission gate stays closed and no policy is activated.


### Original observer through bounded ABI framing and canonical binary decoding

The private normalized DATA reader now passes its existing optional original
observer through mutation selector lookup, the adopted ABI framing, canonical
ABI Unpack/Pack and exact byte comparison. The event reader uses that same
observer for the bounded non-indexed argument projection and RecordChanged
framing/round-trip. Nil legacy DATA callers retain the existing byte contract.
Only the eight adopted mutation methods can yield a decoded command. The
separate commit-claim branch keeps its existing semantics and still needs its
complete observation connection. No producer can supply an ABI type graph.

Observed framing preserves the existing method/field/array limits, exact tail
offsets, zero padding, ASCII strings and rejection of trailing call data. The
observed internal walker additionally bounds recursion to32 and argument heads
to128, checks internal tuple/element shape, and explicitly initializes/copies
argument storage. The actual logical observations count byte comparisons,
uint64 decoding, ABI decoding/encoding and concrete argument storage/copies.
The Pack preflight bound is the unchanged successfully framed original byte
length; it is not recorded as an actual encoder result. A refused original
observer returns no successful partial command/field result.

Finite offline tests include all eight existing mutation methods, independently
calculated exact word/string/codec quotas, one-short and late failures, isolated
otherwise-valid missing inputs, and the existing valid claim/event/state
fixture. They add no production keys, transactions, network calls or attack
producer. They are NOT_RUN; all future per-file90% checks are UNMEASURED.

This is a partial connection inside the single unpublished complete bundle.
Canonical command JSON, schema/key/service conversion, cryptographic claims,
commit-claim primitives, semantic indexes/state/expected-history copies and
both original S content/effect/state/grants/history transitions remain to be
connected to the original observer. In particular, the legacy command parser
called after ABI decoding is not thereby fully observed. This section does not
claim an entire observed replay, a native ABI/crypto/allocator CPU or backing
RAM profile, sufficient whole11-coordinate costs, a production factory or a
qualified live Source. Generic collection admission remains closed. Existing
strict gas rules, RPC selection, trust pins, trace requirements and operating
policy are unchanged. No new CI or publication has validated these changes.


### Original observer through the strict command consumer

The private nonnil command decoder now passes its original observer to the
final strict JSON consumer, key/service schema parsing, canonical base64 and
decimal decoding, owned raw command and output arrays, and ABI parameter
construction/packing. The same parsed command must still reproduce the actual
original calldata byte for byte. Nil legacy callers retain the original parser
and its failure result contract. The observed consumer is bound to the adopted
Sepolia Registry/agent grammar; it is not a caller-supplied ABI API.

The observed JSON path preserves UTF-8/unicode, duplicate/depth, ASCII and
canonical byte requirements. A bounded token walk precedes a UseNumber decode.
An independently owned native tree normalizes only the canonical writer's
Number values to finite float64; the original UseNumber tree remains unchanged
for schema validation. The native writer receives no custom marshaler. For raw
input of n<=262144 bytes its preflight6*n+64 accounts for compact-exponent
expansion; actual encoded bytes alone are recorded, and exact input equality is
still mandatory. The native tree copies/limits are logical observations, not a
complete native JSON/sort/allocator or backing-map memory qualification.

Keys retain required-suite algorithms, ordered names, material uniqueness,
exact proof shape/signer naming and expiry presence. Proof encoding validation
is not cryptographic proof verification. Services retain ordered names, key
namespace separation and the existing bounded HTTPS/IP/percent grammar. URI
validation is one observed bounded parser invocation; its CPU/allocator profile
remains unqualified. ABI arrays and fixed2..5 parameter slots have explicit
initialization/assignment observations and no append-growth copies. The fixed
adopted method caps bound Pack preflight; those caps are never actual output.
Failures return no successful partial observed command.

Offline tests compare all eight existing mutation results, raw ownership and
explicit zero expiry against the legacy consumer. Independent binary/decimal
quota arithmetic, JSON preflight-versus-actual accounting, existing malformed
JSON/schema scenarios and a final Pack preflight shortage are included. These
tests are NOT_RUN and all future per-file90% gates remain UNMEASURED. No
production key, network request, transaction or attack producer was added.

The consumer alone does not account for earlier tuple conversion or command
generation. Their subsequent observed connection is described below. ABI
initialization, claims/commit/PoP and semantic state/index/expected-history copies, both
original S transitions, complete11-coordinate sufficient costs/native CPU/peak
RAM/lifetimes/failure/final/factory and operating qualification remain incomplete.
Generic collection admission remains closed, and the entire candidate is still
unpublished with no new CI. No operating policy, budget, RPC, trust pin or
existing strict gas requirement changed.

### Original observer through decoded tuples and command generation

The nonnil decoder now converts the actual canonical ABI tuple values directly,
without a JSON marshal/unmarshal bridge. It checks fixed field tags/kinds and
bounded key/service arrays before making owned strings, byte payloads and
output slots. The raw service field `uri` maps explicitly to the owned `URI`
field. Nil and nonnil empty byte/tuple values retain their existing conversion
shape; generated command lists retain the existing empty-list normalization.
No caller marshaler is invoked. Logical reflection/slot/copy observations do
not qualify reflection, allocator or backing-array native costs.

Command construction uses closed builtin maps/arrays with explicit slot
observations. Actual base64 and decimal output lengths are recorded after
bounded preflights. Address formatting encodes the existing20bytes to lowercase
hex directly, avoiding an intermediate checksum hash. Expected versions remain
decimal strings, including full uint64 values. Key expiry retains its optional
presence and explicit zero; unsafe JSON integer expiry and contradictory ABI
presence flags are rejected.

The canonical writer first visits and owns the actual native tree under the
original observer. Independently counted source payload bytes and entered nodes
are each capped at262144 with depth32. A preflight of6times their sum plus64
bytes covers ASCII escapes, finite-number expansion, punctuation and literals;
only the actual encoded output is recorded. These counts are derived from the
tree itself, not calldata size or producer claims. Eligibility ensures the
writer uses its closed native path. Native sorting/JSON CPU/peak RAM remain
unqualified. For the literal `"a"`, the bounded encoder requires5782logical
headroom but records only15logical work; preflight is not actual output.

Generated bytes then enter the same strict consumer with the same original
observer and the same adopted ABI. Exact original calldata equality remains
mandatory, and every nonnil failure returns no successful partial command.
Nil legacy callers keep their previous parser/decoder and diagnostic failure
result. Tests compare all eight actual command results, owned conversions,
zero/absent expiry, independent primitive quota arithmetic and bounded failure
paths. All new tests remain NOT_RUN and future per-file90% coverage remains
UNMEASURED. ABI construction, claim/commit/proof verification, semantic state,
both original S transitions and full cost/factory/operating qualification are
still incomplete; collection admission stays closed and no policy is activated.

### Original observer through event digests and create claim bytes

The actual RecordChanged event caller now joins the DID, hashes its actual text,
constructs the indexed actor topic from the decoded20byte address and compares
the topic bytes under its original observer. Command, record and grants SHA256
inputs and their event digest comparisons use that same observer. The create
claim is derived from the actual decoded command fields, canonicalized as a
closed builtin tree, copied behind the exact `sage-claim-0.11.0` NUL-terminated
domain and freshly hashed. No caller-supplied digest substitutes for that work.

Raw hash payloads retain the existing262144byte command/grants bound with only
up to64fixed domain bytes of internal framing. Text conversion and explicit
claim buffer initialization/copies are observed before allocation or copying.
Nil callers retain existing bytes and results. Logical hash observations are
input-length work, not a native SHA256 CPU/allocator/RAM qualification. A
separate observed base64 digest primitive is prepared for state generation;
the existing state generator does not use it yet.

Private offline tests use the independent SHA256 `abc` vector, literal lower
address topic, exact hash4/text-copy-plus-hash8 and digest1854logical quotas,
one-short/sticky refusal, existing claim bytes/domain and direct event caller
comparison against eleven handwritten normal snapshots. Seven bounded unit
faults change DID/actor topics, command/record/grants digests or caller-scoped
prior claims; no successful history step may be returned. The tuple ownership
test also snapshots original Proof bytes before changing its owned output.
These are NOT_RUN tests, not new runtime/coverage or original-reader evidence.

This increment alone does not observe ClaimCommitted handling; its subsequent
connection is described below. State/proof/index/expected-history observations,
native ABI construction and both original S transitions remain incomplete.
All sufficient cost/native RAM/lifetime/final-page/factory gates remain closed.
This connection neither activates a policy nor publishes the candidate.

### Original observer through the fixed commit event and claim index

The actual nonnil commit handler now observes its fixed36byte calldata selector,
actual indexed actor/digest comparisons and canonical32byte event height. The
adopted commit method has a bytes32 input; its event has one nonindexed uint256
height. The mutation-only selector and uint64-only framing helpers are not used
for this distinct schema. After checking the embedded field type, the already
decoded uint64 header height is encoded as24zero bytes plus8big-endian bytes.
The entire32byte body must equal the original log data, including high padding.

Actor strings and the caller/digest claim key are copied explicitly. Duplicate
claims, including consumed tombstones, cannot refresh their original block or
consumed status. The original lifetime claim limit is preserved. The original
observer funds the logical map lookup and final key/value assignment before a
new claim is retained. A one-unit shortage at that final assignment leaves no
new successful claim and preserves the actual earlier work; retrying the failed
observer cannot reset its state. Nil legacy callers keep the existing handler.

Offline tests compare0,1and fulluint64 maximum heights against the existing ABI
encoder and nil handler, exercise the final store shortage, lifetime/tombstone
retention and nine bounded fixed-input/event faults. These tests are NOT_RUN;
the new file is included in the unchanged future per-file90% CI gate. Fixed
logical encoding/map observations do not qualify native ABI initialization,
big.Int getter cost, map buckets/backingRAM or whole-program sufficient cost.
State/PoP/authorization/expected-history and both original S transitions still
need their complete observer connection and failure lifecycle. No generic
collection, factory, operating policy or new runtime qualification is enabled.

### Original observer through owned state, ACL and required-suite proofs

The actual event reader now calls an observed state path with its unchanged
original observer. Nil callers delegate to the old state implementation. The
nonnil path owns a working state copy, including key expiry values and key,
service and grant arrays. Fixed-size insertion/removal and grant cleanup avoid
implicit append growth and preserve the existing ASCII key-name and
operator/scope ordering. Actual controller/operator authorization, lifecycle,
signer usability, namespace, monotonic time, full uint64 version and collection
limits remain in force. A completed result assigns the working state only after
its encoding/proofs and final observation succeed.

Record generation preserves revoked keys, optional expiry and explicit zero;
it does not use the command-only serializer that sets every key to accepted.
Version values stay strings. Admitted time uses a safe numeric value. History
has ten fixed fields with its own object builder; the command builder's existing
eight-field cap is unchanged. All generated trees enter the same closed native
canonical writer. The actual command/record/grants digests now use the observed
SHA/base64 primitive. The existing record65536 limit and exact future
deactivated/full-version size headroom are checked before new proof verification.

Proof challenges retain the exact adopted `sage-pop-0.11.0` domain and five
uint16-length-framed fields. Their actual lengths fund an exact output buffer
and each field write/copy. Newly added Ed25519 self proofs are verified before
X25519 endorsements, using the original strict canonical point/subgroup/scalar
checks. A KEM key never supplies a signing role; its named endorser must be an
accepted, unexpired Ed25519 key. Historical endorsements are retained without
being re-proven after later mutations.

One actual strict verification invocation atomically records its bounded input
invocation and one proof attempt in coordinate7. An unfunded input/attempt pair
records neither, while a cryptographically invalid attempted proof retains its
charge. This uses the original observer without resetting its limits. Only unit
subjects that enter these actual proofs receive an explicit existing4096-proof
allowance; execution-only unit meters retain their zero proof allowance. Native
curve/point/getter/JSON/sort/map/allocator/peakRAM and whole sufficient cost still
need separate qualification; a logical invocation is not that certificate.

Offline tests compare eleven handwritten normal states and all eight operations
against the nil implementation. They cover late failed proofs, original-state
isolation, ACL/lifecycle refusals, automatic last-signer deactivation, full
version/time boundaries, role-separated endorsements, exact native-input/proof
reservation and sticky refusal. A separate typed serialization-width fixture
checks65520present bytes versus65543terminal bytes; it explicitly does not claim
an admitted128-key state. New tests remain NOT_RUN and per-file90% coverage is
UNMEASURED. No new production key, transaction, peer attack producer or operating
policy is introduced.

The working-state guarantee belongs to this apply operation. Earlier event
claim/created-index consumption is not rolled back by it; the collector must
still quarantine a failed run/window. Original shared-index/expected-queue
observation, both complete original S transitions, all11 sufficient costs,
native backingRAM/lifetime/failure/final-page/C0-Dp-C1 and factory/operating
qualification remain incomplete. Generic collection admission remains closed
and this whole candidate has not been published or run in new CI.

## Original semantic observer, lifetime indexes and expected history

The explicit private observed fold now binds the same original meter to every
window. Its legacy constructor retains a nil observer. New windows clone the
confirmed state, expiry pointers and lifetime claim/created maps before replay;
they do not reset that original meter or erase consumed commitment tombstones.
Create events, including another selected subject's Registry effects, observe
their actual index lookups and fund both index writes before changing either.
A later proof or digest failure still requires discarding the complete window.

Expected commands and generated history entries use owned byte strings and
exact-sized queue arrays. Separate stop-state and ordered history comparisons
use that same observer before the confirmed prefix can advance. The strict
API's result-array storage is also explicitly copied when an observer is present.
These are logical copies and comparisons, not a native Go map bucket, allocator
or worst-case resident-memory certificate. Native configuration/ABI initialization,
remaining block trace indexes and implicit getter/header copies are not yet
fully qualified. Both original S reader transitions, all eleven sufficient
costs, failed/final-page/C0-Dp-C1 and the collector/factory gate remain unfinished.

Offline unit scenarios cover three multi-item windows against the independent
handwritten state/history fixture and legacy fold, global other-subject effects,
queue byte ownership, lifetime limits, original-observer binding, refusal and
confirmed-prefix isolation. Unit allowances are explicit DATA and do not change
production limits. These new scenarios are NOT_RUN and individual file coverage
remains UNMEASURED until the complete connected bundle's new native CI.

## Independent original-reader semantic graphs and commitments

Each existing original-reader cursor constructor now initializes its own private
semantic fold from that reader's reconstructed first finalized selection.
Copying a cursor deep-copies confirmed and working state, expiry pointers,
lifetime indexes, pending bytes, expected items and run/window ownership links.
The copy retains the same execution observer; a different observer may inspect
the read-only DATA commitment but cannot fork or execute that fold.
Parsed contract/outcomes ABI configuration remains shared and read-only. This
ownership boundary covers mutable semantic state and queues, not physical
independence of every reachable Go allocation or a native-memory certificate.

The cursor commitment now includes its semantic graph. Each invocation freshly
encodes actual state, visits all claim and created entries in an observed fixed
storage merge order, and hashes every expected command and entry plus window
counters and pending/comparison status. Full uint64 block/version values use
decimal strings. Claim and created indexes have distinct domains, including
empty indexes. Zero-version state rejects hidden record/grant content. Roots
are reconstructed DATA and never replace either reader's original T/C/S replay.

These initial-reader bindings and owned snapshots do not implement the remaining
content/effects/state/grants/history/close transitions. The generic collection
gate remains closed. Repeated full-prefix interpretation, queue and map copying,
native costs and the complete eleven-coordinate/RAM/final/factory envelope still
require qualification. No root is cached to grant authority or renew work.

Offline scenarios compare index ordering with an independent standard-library
sort reference, distinguish changed semantic fields, test source isolation and
pending ownership, and refuse both early and late snapshot quota failures.
The expiry-pointer ownership scenario explicitly installs typed unit DATA after
normal replay; the original handwritten commands contain no expiry. This setup
does not demonstrate an authenticated expiring-key command or its proof.
Existing collection DATA execution fixtures retain their original observers;
independent interpreters are initialized from separate cursors and observers.
The earlier strict-result append test now uses a deep snapshot and a normal
append's final-copy allowance-minus-one. This resolves that SOURCE test limit;
all new scenarios remain NOT_RUN and per-file coverage is still UNMEASURED.

## Caller cancellation through original replay

The private admission, consumption and dispatch candidates now require an
explicit caller context. The same context is passed through historical prefix
reconstruction, current settlement readback, recovery settlement and each
collection boundary. None of these paths creates a replacement background
context, deadline, work observer or clock epoch. The legacy context-free
settlement wrapper remains available only to nil-collection DATA models.

Stopped callers cannot begin a new transaction. Cancellation after entry
quarantines the existing owner without refunding a consumed allowance or
publishing a new logical pin, result or acknowledgment. A physical HEAD rename
that already happened remains an unacknowledged tail; cancellation does not
roll back its files. Cancellation alone does not skip the original post-rename
durability attempt. Existing original clock and final-bracket checks remain.

New unit scenarios cover nil/canceled collection calls and settlement with
retained consumed charges. Safe runtime unit scenarios use only the existing
owned TLS and temporary-directory fixtures. They cover stopped admission,
consumption and dispatch, plus ordinary cancellation after a successful local
sync before or after T/C/S rename. Earlier immutable originals and archives,
physical HEAD and the unpublished logical pin are compared independently.
These scenarios are NOT_RUN; individual-file coverage remains UNMEASURED.

Context propagation is a prerequisite for the remaining full content, effects,
state, grants, history and close transitions, not their implementation or a
sufficient native cost certificate. The generic collector/factory gate remains
closed pending the complete connected bundle and new native CI evidence.

## Original content through closed-window reconstruction

Each independent original S reader now passes its own exact result bytes
through content, effects, stop-state, grants, history and close transitions.
Complete raw or explicitly selected JSON blocks are checked against the known
spine, canonical consensus encoding, uncle and withdrawal commitments, complete
transactions and receipts, and the preselected strict native gas profile.
Registry receipt logs alone select full transaction traces. Root-only gas
observations never replace the full call/log evidence used for signatures,
proofs, account authority and the actual caller during semantic replay.

Pending batches retain immutable consensus byte strings with separately owned
mutable arrays. Every cursor commitment freshly hashes the actual pending
bytes, received-trace positions, applied prefix, stop record and cumulative
present-stop count. Copies preserve the original observer. Temporary decoder
limits never renew that observer or the cumulative semantic-window budget.
Derived pending bytes are released after each complete block is replayed; the
original immutable archive and its consumed allowance are not released or
refunded. A failed candidate cannot change its preceding reader cursor.

The stop record and grants must match independently derived state before the
window's exact version delta is requested. History begins at baseVersion + 1,
not version 1 in every window. An absent record produces DATA only. Content
close rechecks the window's high block (ancestry close retains its low block),
the original selected block, finality, chain, genesis, Registry and client
identity before confirming the semantic prefix. The final `done` phase cannot
plan another table and is not SourceReady or permission to publish.

Seven new test roots cover both independent whole synthetic ranges under three
window partitions, six phase refusals, pending-array ownership and four
context/observer/allowance refusals, plus recorded canonical content in both
explicit encodings. Full-range scenarios use explicitly synthetic private
cursor DATA: block zero cannot pass the production manifest's real Sepolia
deployment constructor. They make no actual-node or authenticated-source claim.
All new tests remain NOT_RUN and all per-file coverage remains UNMEASURED.

The full-content UNIT fixtures select count-capable original observers before
entry. The recorded-only decoder component separately selects explicit capture
limits large enough for its existing originals; production/default limits are
unchanged. The empty native case is handled before profile inference: only a
complete empty transaction/receipt set and an exact empty native array can use
the already selected schema. The unchanged strict gas checker still requires
zero header gas. Missing results, extra rows and a nonempty block with an empty
native result remain refusals. Existing classifier APIs/defaults are unchanged.

Offline scenarios cover all four preselected profiles and these empty-result
boundaries. An additional normal synthetic 40-claim/two-block fixture splits
effects across 32 and 8 original results. It verifies a fully applied first
block while the second remains pending, copied pending-byte ownership and fresh
roots, then either full completion or a late parse/cancellation refusal that
preserves the accepted predecessor. Public disposable fixture envelopes are
used only in memory; no network producer or transaction broadcast is added.

This is not the full sufficient-cost, native/backing-RAM, lifetime/failure,
final-page/C0-Dp-C1 or factory qualification. The generic collection admission
gate remains closed. One complete connected bundle still requires new native
remote CI and original artifacts before actual acceptance and publication.

## Native block representation and ABI initialization

The observed original-reader path now obtains an independent header directly
from the first canonical RLP element under its original observer. It checks
the entire block body independently and still matches the selected spine.
This avoids the pinned library's implicit deep-copying `Block.Header` getter.
Observed JSON construction encodes the same external block-body fields
directly, avoiding implicit header/body cloning in native block constructors.
Observed receipt reconstruction accepts that independently derived header;
other observed callers reconstruct it from their own canonical block bytes.
Unobserved legacy DATA callers keep their existing native path.

Each observed semantic window also accounts for parsing the same two fixed
embedded ABI definitions under the same original observer. No caller ABI,
replacement observer, reset allowance, new profile or execution authority is
introduced. ABI internals form part of this bounded native codec invocation;
logical units do not measure CPU instructions or allocator bytes. Parsed ABI
configuration remains fixed read-only data shared by semantic snapshots.

New offline units compare complete encoded blocks with the pinned native
encoder, verify independently decoded header fields and input isolation,
compare every fixed ABI selector/event, and retain original consumption on
early/late failures and across all semantic windows. All remain NOT_RUN and
individual-file coverage remains UNMEASURED. The dependency's primary versioned
source informed this change; module identity, native CPU/backing-live-RAM,
all eleven sufficient coordinates, elapsed/failure/final work and factory
qualification remain mandatory. Generic collection admission stays closed.

Whole-body codec consumption is recorded immediately at its own native decode
entry, after independent header validation. An earlier selected-header refusal
therefore consumes the actually entered header/shape codecs but not a body
decoder that was never entered. A complete recorded precursor and independent
input-length oracle cover that distinction; no consumed work is refunded.

## Ordered original-read recording inside both complete dispatch replays

Each actual dispatch-preparation replay now records its own ordered file-read
transcript through the fixed store reader, including repeated HEAD reads and
the first-P originals. Every row fixes the selected root, exact filename,
original read maximum, actual byte length and ordinary content digest. The
two full semantic passes keep independent arrays and role-separated rolling
commitments bound to their original C and model. A failed interpretation or
recording cannot complete a transcript, and the recorder is detached on every
return. All hashes, filename copies and explicit descriptor-copy observations
use the already-held original observer; no byte payload graph is retained.

These completed transcripts are DATA. They do not replace a semantic replay,
issue a permission or prove complete source costs. The member path still
performs its existing two full original semantic replays. A subsequent
equivalence connection must freshly reread every ordered original, authenticate
the current C's real control frame/model root, bind all fixed context, keep
independent model ownership and reconstruct fixed ABI maps that are outside
the semantic commitment. Current-member originals, clock/authority gates,
settlement, advance, recovery and final checks remain separately mandatory.
Matching cached roots alone can never substitute for these obligations.

The new arrays, rolling encodes and native/backing-memory peaks add costs
that the complete source compiler must fund. The explicit row limit is an
input-recording sublimit, not a sufficient-read-count proof or a resident-RAM
certificate. New normal and refusal units are NOT_RUN; per-file90 remains
UNMEASURED until the single complete connected bundle's new native CI. Generic
collection admission stays closed and whole5-1/5/7/8 remain incomplete.

The initial recording units use a legacy first-stage T/C fixture: one pass
has exactly24 successful content reads, HEAD ordinals0/4/5/15 and the C frontier
as its final input. They independently check all selectors/maxima and original
file SHA256. This is not a new manifest-backed firstP or earlier-stage runtime
test. The DATA-root quota units exercise logical encode/hash work, whose
decoded coordinate is zero. Their sufficient/one-below boundary includes the
encoder's conservative preflight and does not assume final actual work alone
funds each intermediate admission. All remain authored, unexecuted tests.
### Same-consumption member replay equivalence candidate

The first independent source review required two repairs. The binding now also
copies the complete original C-to-initial checkpoint chain, initial/admission
descriptors and archived order/completion/capture/intent/attempt/original member
values. Each role compares these with the original value baseline before and
after its fresh byte reads. Scalar selection contains only original pointers
and numeric facts; raw-bearing context and historical struct equality are
explicitly observed outside meter-lock selection. A finite observed typed
metadata-length walk derives conservative comparison weights without following
pointers or invoking value methods. Equal string bytes with new backing still
incur the comparison debit. These new arrays and logical events do not qualify
complete native/backing RAM or elapsed time.

The second source review found a separate current T/C input gap: an original
write-claim pointer does not bind its purpose, expected highwater, pointed
frame/inventory, or coordinator head and selected table digest. A closed typed
projection now captures those reader inputs and exact original pointers after
the successful full dispatch passes. Both roles freshly compare it before and
after original refresh under the same observer, outside scalar selection. Each
fixed typed copy and both-sided equality are charged before the work. Pending
write payloads and catalog maps are not consumed by these completed T/C readers;
their original catalog checks remain. Live order progress is not copied into
this fixed projection. Revocation also clears its two fixed input slots.

The enclosing original full wrapper also selects HEAD through the original
frontierFiles object/pinRaw and separately validates the original group pair,
which are different from the dispatch baseline models. The fixed context now
binds the original frontier selector pointer/full comparable value and both
original group ledger pointers. Each role retains BOTH original group validity
checks and observed head equality to actual C, alongside dispatch model checks.
Pointer selection is scalar; frontier raw/value comparison and model validation
are observed outside active selection. No changed HEAD baseline is adopted.
Fixed group-reference slots are also dropped on cache revocation.

Extra authored faults change only frontier pinRaw or unchanged-head charged
totals in either original group model. Each preserves scalar eligibility and
must fail the old full oracle and new observed boundary. Fresh prepare-entry
cases quarantine before request debit/transport. Nil/copied frontier and copied
group custodian pointers instead exercise zero-work original-selection refusal;
they are not advertised as reaching the value/validity boundary. These repairs
still require independent source review and actual native CI execution.

Authored units use a genuine positive full-reader and equivalent control, change
only one current purpose/highwater/C-frame clock/T-model expectation, require
old full refusal and new observed refusal for each role, then independently
restore UNIT values and require positive equivalence. Separate real prepare
entries must quarantine before request debit or transport; entered failures are
never restored. C-model and T-table scalar provenance refusals are tested and
distinguished from reaching the new comparison boundary. Nil selectors and
copy/metadata/equality shortages retain their original entered work. These are
fixed in-memory defensive UNIT cases, not a runtime attack producer.

Failure cleanup follows the trusted original recovery registration as well as
the current dispatch. A nil/copied current pointer cannot keep the original
transcript/ancestor arrays eligible or reachable through their old registrations.
Original models, ACKs and consumed/sent facts remain. Authored regression units
cover both roles' older checkpoint/descriptor/archive expectations, coherent
archive changes, positive restoration, orphan cleanup and fresh comparison byte
accounting. Binding-fault units now enter a genuine positive preparing-member
phase before mutation. Quota units use a monotone prefix inside the original T
window and check entered sticky exhaustion; no specific deep late-codec cutoff
is claimed. All these units remain NOT_RUN; per-file coverage is UNMEASURED.

The unpublished member preparation path now registers a private binding in both
original owners after two complete, separately interpreted dispatch replays. For
each role on every member, it freshly reads the complete ordered original input
transcript, including duplicates and original read maxima, and compares every
actual byte length and SHA-256 digest with that completed original baseline.
Existing SHA-256 collision resistance is an explicit equivalence assumption;
self-consistent cached hashes alone are not authority. Fixed configuration,
initial bridge, scope, roots and lock handles, windows, checkpoints and separate
role identities are checked independently. The actual physical C highwater and
intent/commit/frontier are parsed again and supply the model anchor.

The original full-pass semantic model and an independently owned copy must both
match that anchor. Non-root behavioral ABI maps are reconstructed from immutable
literals on each copied run under the original observer; empty step backing is
dropped from that copy. This never rewrites the cached source model. Live meter
actuals, clock reads, member retention and sent/completed bits continue growing;
they are not frozen to dispatch preparation. The original current-member prefix
is still read twice, and preflight, deadline, irreversible order and RPC debit
remain in their original order. Refusal has no fallback. Successful S and failure
revoke the binding and release its transcript arrays without resetting archived
originals, models or consumed/sent bits. Full semantic replay remains unchanged
for S, failed S, advance, recovery and final scans.

New tests are authored but NOT_RUN. They compare the member result with the
unchanged full replay oracle, exercise two prior genuine one-member T/C/S stages,
both roles' original refresh failures, actual C anchoring despite coherent cached
DATA, growing current members, quarantine and independent ABI ownership. The
semantic ABI test is typed DATA, not a live collection C. The two-prior-stage
fixture and several-current-members fixture now also have a combined authored
case: actual S1/advance/S2/advance, three original C3 members and full S3. Its
separate pre-H0 UNIT envelope funds at most nine RPCs and 2,359,305 wire units
under unchanged enclosing policy caps. Per-role old/new probes retain all work
and do not consume orders or sample clocks. The live exchanges use only owned
temporary TLS fixtures. This combined case remains NOT_RUN and does not qualify
production bounds, native timing or real Registry collection. Fresh runtime,
race, per-file coverage and the
complete source-derived eleven-coordinate cost, native window, resident/backing
memory, lifetime, failure/final/C0-Dp-C1 and production factory qualifications
remain required. Repeated raw I/O is preserved and may still be quadratic. No
whole-history feasibility, policy activation, Source qualification, protected
Agent exchange or completion of the existing implementation plan follows from
this source candidate. The generic collection admission gate remains closed.

The unpublished internal collection entry now accepts no caller purpose, request
list or numeric upper. A private admission row is part of the original pre-H0
derivation and is bound separately by both original program and recovery roles.
The cached cursors provide only a bounded scalar member-count proposal for
reserving a subwindow. Only the RPC count and worst-case member wire sum are
refined; every other work, storage and clock maximum remains unchanged. The
original whole-program hold and lifetime are never renewed.

Inside that original T window, after its initial native sample, each role
independently replays the complete actual initial and prior T/C/S originals and
derives its own next request table. Both complete tables, their exact proposed
counts, the original pin/inventory and both original model heads must agree
before any T write or ACK. A mismatch fails the entered owner without refund or
fallback. The unchanged T writer, independent custodian read and final sample
then publish the result; the existing C path consumes that exact original T.
The two extra full-prefix planning passes are additional source work and must
be funded by the eventual complete cost derivation, including all live backing
storage and failure paths. Prior member-only cost decompositions do not cover
this added work.

Authored tests exercise a separate one-block absent-agent UNIT subject selected
before H0, including real owned first P and original collection T/C. Other
scenarios check fixed pre/post-fork request counts, partial ancestry/effect/
history groups, unchanged non-cardinality budget fields, refusal of caller
tables or missing plans, post-H0 plan changes and changed owned original bytes
before a T write. These tests are NOT_RUN and their coverage is UNMEASURED.
The UNIT envelope is explicit synthetic input, not a source/native/RAM
certificate or a production factory. No production constructor mints the
admission row. Generic caller-supplied collection T remains refused and no
operating policy, real full-history collection or protected Agent demo has been
activated by this connection.

The private read-bound arithmetic now separates seven successful-path content
read rows for prospective group/request/first-P-object maxima G/R/O. It accepts
only nonempty bounded groups, at most 32 members per group, G <= R <= 32G and
the original stage/object domain. It allocates no input-sized arrays, reads no
files and grants no execution authority. It is a component of the unfinished
whole-cost compiler, not that compiler or its production derivation factory.

For an actual chronological group sequence m, a healthy-prefix replay reads
H(g) = 14 + O + sum(j<g)(23 + 2*m[j]) content files. The initial 14 include four
initial controls, four first-P inventories, **five first-P C/S controls** and
the final HEAD; O counts the first-P objects separately. Those five C/S controls
are actual reads in first-original control(), not optional metadata. Earlier
collection read-count preparation that omitted them is superseded for this
cost row. The legacy non-collection formula is unchanged. Current C adds ten
T reads and nine C reads, so its collection transcript has H(g) + 19 reads.

| Separate row | Source topology |
| --- | --- |
| initialModels | Two initial models, each 13+O reads, without the healthy replay's final HEAD |
| planning | Two complete healthy-prefix reads per T; already included in healthyPrefixes |
| healthyPrefixes | Twelve per group: new planning2 plus original T2/C3/dispatch2/S3 |
| memberReplay | Two complete current-C transcripts plus two eight-file C/HEAD anchors per member; excludes surrounding catalog content reads |
| memberPreparation | Spool response/TLS content only in two pre-send prefixes of earlier members; excludes nested catalog content reads |
| memberRetention | Spool response/TLS content only in two post-write prefixes including the new member; excludes nested catalog content reads |
| advance | Two complete healthy-prefix reads for every next stage, with no final-stage advance |

The closed arithmetic maximizes prior/late weighted members and bounds the
member cross term using the balanced minimum sum of squares. Those independent
upper rows need not be attained by the same sequence. Realized optional groups
can be fewer than G: member-local prefix squares therefore use at-most-G packed
32-member groups rather than assuming every declared group occurs. Splitting a
group only adds nonnegative healthy/model replay work, whereas member-local
prefix work can grow when members remain concentrated in fewer groups. The
separate bounds preserve that distinction. Authored units compare handwritten
rows and enumerate small chronological compositions, including fewer realized
groups/requests and a 33-member split. All remain NOT_RUN.

These are counts of content-read invocations, not a byte, filesystem metadata,
logical eleven-coordinate, native CPU or RAM upper. Remaining current T/C/S
suffix controls, S spool rereads, dispatch seal construction, codec/model/ABI/
crypto work, write/sync/identity/catalog operations, failure/recovery/final/page
paths and complete native/live-set qualification must still be composed. Never
sum planning a second time into healthyPrefixes or treat these rows as a
sufficient whole-program hold. No production cost factory is enabled here.

The pure arithmetic component keeps every sum/product and subtraction checked.
An invalid intermediate is sticky: later successful arithmetic cannot turn a
partial result into a quote, and the complete row is rejected. Authored numeric
units cover overflow, subtraction underflow and that no-revival property. This
consolidates rejection propagation without relaxing the finite domain or the
individual-file 90% requirement. Actual coverage is still UNMEASURED.

The next private component counts successful catalog content reads separately.
For original pre-H0 scan maxima A (attempts, 1 through 64) and C (chunks), one
initial catalog derivation reads K = 10 + 5*A + 4*C content files: scan control
reads 6+5*A, chunk and command receipts read in both original roots add 4*C,
and four legacy inventories supply the remaining four. Positive checked C is
numeric input here; qualifying it against the actual store, scan and profile
remains an obligation of the unfinished compiler. Directory enumeration and
map/name interpretation do not become content-file reads in this row.

| Successful catalog caller | Initial catalog derivations |
| --- | --- |
| T, C and dispatch | 1, 1 and 2 per group |
| First member | 14, including both original replay roles and post-write prefix wrappers |
| Each later member | 16, adding two earlier-member prefix checks |
| S | 12, including the initial prefix that precedes the selected dispatch catalog |
| Advance | Two for each nonfinal group; none after final S |

Thus at most G groups and R members require 16*R+16*G-2 successful catalog
derivations, each bounded by K. The component checks both the single-catalog
arithmetic and the full product; neither overflow may return a partial quote.
Authored handwritten rows and a chronological source-call walk check smaller
optional realizations as well as the original maximum domain. These are source
arithmetic units, not observations of production execution, and remain NOT_RUN.

A composed successful content-read count now adds those catalog reads to the
previous seven rows without adding planning twice. The remaining disjoint
current-control suffixes are T23, C53, dispatch45 and S76 per group, plus six
spool content reads per member during S. Each nonfinal advance adds one
separate final main HEAD read, beyond its two healthy-prefix replays, for a
remaining total of 197*G+6*R+(G-1). The shared physical writer
contributes three direct reads per T/C/S (pre-write HEAD and old highwater,
then pre-rename HEAD). The outer C and S readers' new highwater reads are
included. An independently written phase walk and literal one-member/one-block
UNIT totals check this composition. Two fully saturated groups provide a
separate numeric member-domain regression where every prefix row is tight, so
an omitted advance HEAD cannot hide behind unrelated upper-bound slack. This
DATA sequence is not a replacement for a genuine first-open collection cursor
or a runtime workload. A near-limit catalog that individually
fits must still be refused if adding other read rows overflows.

The same component counts the explicitly sampled owner clock calls from
initialization through final S: two initialization samples, eight per group,
two per member and two per nonfinal advance, or 10*G+2*R. It excludes startup
first P, pre-I scan and all other native transport, timer or deadline calls.
This is not a native elapsed-time qualification or a sufficient clock budget.

These additions close only the successful collection content-read topology
within the stated interval. Read bytes, directory and identity metadata,
write/sync operations, codec/model/ABI/crypto work, all eleven logical work
coordinates, retained/backing memory and GC, native windows and lifetime,
failure/recovery, final C0/Dp/C1 and page publication still require independent
composition. No complete source-derived production factory or execution
authority is created. All new tests remain NOT_RUN and every affected file's
90% coverage remains UNMEASURED until fresh remote CI of the connected bundle.


The successful allocation component separately bounds physical T/C/member/S
writes. Its inputs are original pre-H0 group/request, maximum-object,
maximum-wire and per-file-overhead maxima, never smaller observed results.
T and S each retain one pack bounded by maximumObjectBytes; ticket,
consumption and settlement are three small objects. Each T/C/S also allocates
seven outer metadata files. All those small metadata allocations have the
original 8KiB codec bound. Each member retains an additional response bounded
by maximumWireBytes and a TLS original bounded by 64KiB, even though S also
retains their result pack. These duplicate bytes are deliberately charged.

For G/R groups/members, payload is bounded by
2*G*maximumObjectBytes + R*maximumWireBytes, metadata by 24*G*8192 + R*65536,
objects by 5*G+2*R, and charged file allocations by 26*G+2*R. Raw is payload
plus metadata; physical allocation adds the original overhead for every
charged file. HEAD replacement remains charged without a refund. These
allocation counts are not unique retained directory-name counts, filesystem
block measurements or a backing-memory/RSS bound.

The scoped physical-writer work includes one write and two syncs per file,
three HEAD renames per group and one additional directory sync per rename.
Each write wrapper also charges ten metadata-read events: two four-event
identity checks, OpenFile and the close observation. Byte work includes all
raw bytes and the explicit invocation units. Only the scoped disk-write and
logical-work coordinates are returned; zeros in its other coordinates do not
claim those operations are absent from the complete program.

Authored numeric units expand T/C/S objects and seven outer files plus each
member's two spools independently of the closed formula, compare handwritten
rows, and check fewer optional realizations, invalid inputs and allocation
overflow. They remain NOT_RUN. This is a conservative successful-allocation
and fixed writer-primitive component, not the complete cost compiler. Owner
and catalog metadata checks outside the write wrapper, raw read bytes,
codecs/model/semantic/ABI/crypto, startup and failure/recovery/final/page work,
all eleven sufficient coordinates, native elapsed windows/lifetime and
backing/retained graph/GC/peak RAM still need composition and qualification.
No production derivation, budget renewal, gate or policy is enabled. Fresh
remote CI of the connected bundle and individual-file 90% remain pending.

The next successful-content component weights actual file classes by their
original byte maxima. It avoids treating every control, inventory and spool
as a maximum-size history pack. Let F be original firstRead.raw (the aggregate
first-P inventories, objects and C/S controls), M the original store metadata
read maximum, D=max(M,8192), P the original history-object maximum and
L=maximumWireBytes+65536 for a member's response/TLS pair. These are numeric
DATA inputs only; the missing compiler must rederive and qualify them from
the immutable store/manifest/profile before any execution authority exists.

One initial model reads at most 4*D+F bytes. A healthy-prefix pass reads
Hbyte(i)=U+sum(j<i)(V+L*m[j]), where U=5*D+F includes its final HEAD and
V=21*D+2*P covers an earlier T/C/S's small controls/objects and two packs.
D covers the larger read preflight maximum as well as successful small codec
caps: funding only the eventual 8KiB content could leave the last preflight
without room. F remains one aggregate bound instead of multiplying all first
objects by a large generic maximum. The original first-P sublimit is retained.

The weighted sum and member cross term use the same chronological weighted
maximum and balanced minimum square as the count component; local spool
prefixes still use the at-most-G packed maximum square for fewer optional
groups. Planning remains a subset of the twelve healthy-prefix passes. The
member's extra current T/C suffix and independent C/HEAD anchors contribute
2*P+52*D bytes per request. S's three whole spool prefixes add 3*R*L separately.
The remaining T/C/dispatch/S rows total G*(12*P+185*D), plus that S spool term
and (G-1)*D for the separate final main HEAD on each nonfinal advance.

One catalog derivation reads at most D*(6+5*A+4*C)+4*65536 bytes: scan
control/receipt reads have their metadata preflight maximum, and each of four
legacy catalog inventories is read with a 64KiB maximum. Its previously
verified caller multiplicity remains 16*R+16*G-2. No catalog bytes are hidden
inside the member spool-only row or added to the planning subset twice.

The scoped read-event work is total content bytes plus the independently
bounded content-read invocation count, because each explicit read(n) charges
n+1. This component returns only that logical-work coordinate. It excludes
codec decoded bytes and squared units, metadata identity/directory/native
work, clock samples, writes and all other program coordinates. Every sum,
product and ordered difference is checked; an unrepresentable component or
sum refuses the entire row. The largest numeric domain need not fit uint64.

Three authored units compare handwritten one-member/two-saturated-group rows,
an independent chronological file-class walk, fewer optional realizations,
a partial weighted tail, fixed metadata minima and invalid/overflow cases.
All are NOT_RUN, coverage remains UNMEASURED, and no whole program, native
CPU/RAM, failure/final/page, production factory or operating result follows.

The successful metadata component counts original owner identities and paired
custody/evidence directory sweeps from initialization through final S. It
excludes the write wrapper's ten already-counted metadata events per file.
Directory sweeps and initial-catalog derivations are distinct: T/C each sweep
four times, dispatch twice, the first member fourteen times and each later
member sixteen times, S eleven times and every nonfinal advance twice. The
whole sweep count is 16*R+21*G-2, five more per group than the catalog-content
derivation count. The S preparation's two catalog derivations perform no
sweep themselves; its separate capacity check performs one.

There are 47*G+9*R calls to the owner's two-root identity primitive. This
includes two first-P identities in each actual full initial/healthy replay:
two initialization passes, twelve H passes per group and two per nonfinal
advance, or 14*G full passes. Same-C member transcript refreshes reread raw
files and perform their own explicit identities; they do not reinterpret the
first-P closure. Their nine owner identities per member are counted once.
The fixed successful direct initialization/T/C/dispatch/S/advance identities
are included, including the collection T's final identity after its two full
original table derivations. Each owner identity emits eight zero-byte
read-event units.

For original total catalog limits E entries and B filename bytes across both
roots, each paired sweep emits at most 2*B+2*E+10 observed units: eight for
its original identities, one read and one comparison per name, and one EOF
observation per root. The two roots' combined counts and byte limits are
checked by both callers. Pending HEAD names are included in these totals;
this does not add an unqualified extra entry beyond E. The number of explicit
ReadDir(1) calls is at most E+2 per pair, rather than a filesystem syscall bound.

The directory primitive preflights read(255) before knowing whether the next
result is EOF. Actual EOF charges only read(0). A row funding only actual
observations can therefore fail its final preflight. The component retains
255 additional upper units per paired sweep so either root's final preflight
fits without borrowing from another phase. This is conservative padding in
the proposed upper, not extra measured work, a refund, or a renewed hold.
Authored numeric scan words verify the actual-only refusal and the padding's
sufficiency for normal bounded name lengths and different root distributions.

The input limits and returned rows are numeric DATA, not a qualified live
catalog. Four authored roots cover source-call literals, final preflight
slack, smaller optional sequences and invalid/overflow bounds. They remain
NOT_RUN and coverage UNMEASURED. This component returns only scoped work[10]
and does not include map/name interpretation outside the explicitly observed
entry events, file-content read internals, metadata from other native/TLS/
clock primitives, codecs/model/ABI/crypto, failure/recovery/final/page work,
backing graph/GC/peak RAM or native windows/lifetime. It neither qualifies
those effects nor creates the missing complete eleven-coordinate factory.

The collection capture component applies the existing BYTE-engine envelope
to each of the eight explicitly selected RPC classes. Its input and output
are unqualified numeric DATA: the separate whole-program compiler must still
validate and bind the immutable manifest, code/journal identities and method/
TLS selection before H0. No request, connection or execution is created by
calculating a positive row, and no current profile or operating limit changes.

Let B be the maximum inclusive block count, Wa=ceil((B-1)/window),
Wc=ceil(B/window), W=Wa+Wc, N native blocks, F effect transactions, V new
history versions and P present window stops. The actual open/close grammar
gives these prospective class counts:

| Class | Maximum requests |
| --- | ---: |
| scalar | 4W |
| header | B-1+6W |
| code | 13W |
| state | 2W+Wc+P+V |
| block | B |
| receipts | B |
| gas | N for stateGasTracer profiles, otherwise 0 |
| execution | F for stateGasTracer profiles, otherwise F+N |

Both root-call gas profiles select callTracer, which uses the execution class
even with onlyTopCall enabled. They cannot borrow a potentially smaller gas
wire/decoded bound. The two state-gas profiles use stateGasTracer and the gas
class. Both block encodings select the same block/receipts classes. The class
sum is checked against the fresh grammar request total rather than a cached
or reported count; optional smaller realizations remain below these maxima.

For one class with plaintext wire limit X and valid request length r, the
existing primitive's logical work is q(r)=(X-r)^2+X+3r+8+TLS, its wire upper
is X+1 and its decoded upper is X-r. TLS is the explicitly observed identity
hash/encoding/copy row, not native certificate verification. Since
q(r)-q(1)=(r-1)*(r+4-2X)<=0 for 1<=r<X and X>=3 (X=2 allows only r=1),
the numerical endpoint r=1 bounds all permitted request sizes, including the
16 KiB request cap. One byte here is an arithmetic endpoint, never fabricated
HTTP, a planned RPC or authority to substitute a different request. Each
class uses its own original wire limit instead of a global-size fallback.

The returned work sums the class-weighted primitive coordinates and adds one
irrevocable pre-dial attempt per prospective request. Overflow of an
individual product or the combined sum returns the whole zero row. This is
the successful member capture component only. First P, request construction
and class selection, original replay, persistence, native dial/TLS/OS work,
clocks/windows/lifetime, failure closure, final/page work and backing RAM/GC
remain separate. Six authored roots compare handwritten class/cost literals,
all four actual gas-tracer planner selections, permitted numeric request
sizes, invalid inputs, product/sum overflow and smaller optional realizations.
All remain NOT_RUN and coverage UNMEASURED; the whole eleven-coordinate cost,
production factory, Source qualification and original 5-1/5/7/8 are incomplete.

The semantic lifetime-index component derives the explicit work for one root
of the claims or created-identity index. It uses the actual root function's
two copy buffers, per-entry text/encoding/put observations, fixed-storage merge
rounds, final adjacent-key checks and canonical entry/root digests. A numeric
entry maximum up to 2^20 is DATA, not a supplied map, a reported actual, a
qualified Source or permission to reuse a cached semantic root.

For n entries, let L=ceil(log2(n)) when n>1 and zero otherwise, S be the
compiled target's semantic-index-entry type size and H its slice descriptor
size. These sizes describe the source's explicit copy events; they do not
qualify allocator, map buckets, backing storage, peak RAM or native CPU.
Claim keys contain at most 42 actor bytes, a separator and 64 digest hex
characters (K=107); created-identity keys contain at most 64 bytes (K=64).
The successful per-root logical work rows are:

| Row | Upper logical units |
| --- | ---: |
| Two entry buffers | 2nS+1 |
| Entry population | n(4292+S) for claims; n(131+S) for created identities |
| Merge rounds | L[n(2K+1+S+1)+2H+1] |
| Initial root digest | I^2+I+D+14+4 |
| Decimal block values | 401n |
| Final adjacent checks | max(n-1,0)(2K+1) |
| Entry digests | n(E^2+E+D+20+4) |
| Shared canonical preflight room | M^2-min(I,E)^2 |

M is the unchanged metadata limit 8192 and D is the 41-byte collection-state
domain. I and E are byte uppers for the fixed initial and entry schemas. The
four-field initial object has I=D+14+57=112. Its count uses at most seven
decimal digits. The eight-field entry object contains a 66-byte previous hash,
a seven-digit ordinal, a 20-byte decimal block string, a boolean and bounded
key text. Canonical JSON escapes each ASCII source byte to at most six bytes.
A claim key can expand to 6*42+1+64=317 bytes, because its separator/digest
suffix does not escape; a created key can expand to 6*64=384. Therefore
E=D+20+192+escapedKey gives 570 claims/637 created bytes. The claim label is
one byte shorter than the shared seven-byte index-label upper. Byte uppers
must fit M; this does not change any original encoding limit or output bytes.

Each digest row includes kind copy, its schema-sized encode and the domain-
separated hash. Canonicalization still preflights encode(M) every time. Keep
M^2-min(I,E)^2 additional units once across this entire root computation. This
padding is never charged as an event or refunded. Before each encode, its
own schema-sized actual upper and the retained padding cover M^2+1; all later
events are still funded. Decimal encode(20) preflights remain in their 401n
row. A quote of final actual usage alone can still fail a preflight. The empty
index performs copy(0) and its initial digest. The upper is conservative:
smaller allowances may succeed, and reserved padding is not measured actuals
or a renewed hold. Every quote operation is checked.

Five authored roots compare handwritten event totals with target copy-size
adjustments, check the actual canonical lengths of the fixed scalar schemas,
run ordinary empty/small index DATA through the original root and its
preflights, distinguish actual-only allowance from sufficient prospective
encoding room, and cover optional counts/domain limits. Map
iteration may change merge comparison counts between two executions; tests
require identical sorted roots and each execution within its upper, without
requiring identical observed totals. These are safe arithmetic/semantic units,
not a new live Source, registry call, native or full-program runtime result.

All five tests remain NOT_RUN and coverage UNMEASURED. This row returns only
scoped work[10]. Repeated root multiplicities across both streams/readers,
state serialization/cloning, queue/pending/whole-cursor roots, typed ABI and
cryptography, backing RAM/GC, native windows/lifetime, failure/recovery/final/
page and the complete production cost factory remain separate unfinished
obligations. No operating configuration or original plan order changes.

### Shared semantic state and lifetime-index clone event bound

The private prospective clone component quotes one successful original
`replaySharedIndexCloneObserved`, including `replayStateCloneObserved`. It
returns scoped work[10] only. It neither changes the clone nor admits a live
stream, Source, checkpoint or complete collector. Prospective claim/created
counts are 0..2^20, keys and grants 0..128, services 0..16; the actual original
maximumClaims and maximumIdentities parameters remain positive even for an
empty map. Consumed claims and false created entries remain lifetime data.

Let C/I/K/T/G denote those counts, and let Sx be the target type size in the
corresponding explicit copy event. A copy or compare of n bytes costs n+1.
The event rows are:

| Original operation | Logical work[10] upper |
| --- | --- |
| Stream descriptor | Sstream+1 |
| State descriptor and four text/assignment fields | Sstate+1+546+4*Sstring |
| Key storage, clone, six text/assignment fields, expiry and put | 1+K*(3*Skey+6*Sstring+809) |
| Service storage, three text fields and put | 1+T*(2*Sservice+4295) |
| Grant storage, two text fields and put | 1+G*(2*Sgrant+121) |
| Both map buffers, claim and created entries | 1+C*(2*SclaimSlot+162)+I*(2*ScreatedSlot+196) |

Here SclaimSlot=SclaimID+SclaimState and ScreatedSlot=Sstring+Sbool, matching
the original source's sum of two types rather than a padded combined struct.
The map buffers are one event. Empty/nil key and service arrays still have
copy(0); a nonnil empty grant array also does, while nil grants return before
that event and fit the same conservative upper. Only keys separately observe
string assignment. Each prospective key includes its optional expiry copy.
The final empty preflight adds no event. All quote arithmetic is checked.

Five authored tests use independent handwritten totals, ordinary full-width
typed DATA passed through the original clone and meter, an exact successful
precursor before a one-unit late shortage, optional nil/empty/short text and
expiry cases, and finite invalid cardinalities. The normal full-width fixture
includes consumed claims and false created entries. It is not a validated
credential or fabricated authority. A large scalar cardinality oracle never
allocates its prospective million-entry maps.

These tests are NOT_RUN and per-file coverage is UNMEASURED. Type sizes
describe logical observed copies only. Map hashing/buckets/growth, backing
allocations, peak RAM/GC, native CPU, repeated clone/root multiplicities,
state serialization, expected queues, the complete eleven-coordinate budget
and production factory remain separate unfinished obligations. This component
is not a new runtime result, sufficient full collector or operating activation.

### Complete semantic snapshot copy event composition

The prospective snapshot row now composes the separately derived shared clone
for confirmed state and, when a window exists, working state. The original
`snapshotObserved` still requires its original observer. A numeric upper never
transfers execution, constructs a selected window or authenticates a graph.
The original snapshot, validation, caps and nil-shape restoration are unchanged.

Let F/W/R/J denote the explicit copy sizes for the semantic fold, expected
window, replay run and expected item. Let C/A be the clone rows selected from
prospective confirmed/working cardinalities, Q<=4096 the planned queue bound,
and P indicate possible nonempty pendingCommand bytes. One no-window snapshot
needs F+1+C units. A window snapshot needs:

`F+1+C + W+R+1+A + 1+Q*(2*J+1048581) + P*524290`.

The window and run descriptor sizes share one original copy event. A nonnil
queue has one array event, including copy(0) when empty. Each retained entry
and command separately goes through byte conversion and owned-string copy;
successful strings are 1..262144 bytes. These four text events contribute at
most 1048580 units per item, and the item put adds J+1. Array bytes add the
other Q*J. The conservative extra array unit also covers a nil queue. Nonempty
pendingCommand separately needs two text copies, at most 524290 units. Its
presence is bounded directly, rather than trusting the window.pending flag.

The quote uses direct per-string caps. The original snapshot's predicates do
not independently reconstruct a queue byte sum, so a reported historyBytes
value cannot justify a smaller bound for every successful snapshot. No-window
plans reject queue, pending or working-state settings. Invalid clone counts,
queue bounds and arithmetic return the whole zero vector.

Five authored tests check independent queue literals with copy-size adjustment,
normal original no-window/pending/queued/confirmed-and-working paths, observer
binding, late shortage after a successful precursor, and invalid plans. The
safe semantic fixtures use the existing observed constructor and ordinary
replay transitions. For the scoped snapshot call they reduce the same inert
UNIT meter's allowance; neither cumulative actuals nor its observer is reset.
Read-only commitments compare source and copied meaning independently. The
late-shortage case uses actual measured UNIT work only to place that test's
boundary, never as the prospective production quote or a refreshed hold.

All five tests remain NOT_RUN and coverage UNMEASURED. This is a complete
one-snapshot logical work[10] component, not native CPU/backing RAM/GC or full
program sufficiency. Repeated snapshots and commitments, state serialization,
typed ABI/cryptography, native windows/lifetime, failure/recovery/final/page and
the eleven-coordinate production factory remain unfinished. Original 5-2
through 5-6 retain their order; whole 5-1, 5, 7 and 8 remain incomplete.

### Typed state serialization, including entered refusal paths

The next logical row covers one original record or grant serialization within
prospective K<=128 keys/T<=16 services or G<=128 grants. Record/grant plans are
separate. The quote includes object/array builders, the reconstructed native
tree walk, decimal preflights and the unchanged canonical preflight. It grants
no live state, Source, observer, operating policy or admission authority.

Unlike the clone helpers, these serializers do not check every typed string's
field ceiling. Their native walker independently limits total visited source
bytes to 262144. The record's 65536-byte output guard runs after canonicalization;
an attempt rejected there has already spent walk/encode work. The quote therefore
uses B=262144 for records as well as nonempty grants. Empty grants have no string
or Number bytes and B=0. A smaller successful-output-only bound is not used to
prefund entered attempts. All original byte/count/output limits remain unchanged.

Let M/S/A denote explicit copy sizes for a map descriptor/string/interface, and
X=S+A. A generated object with f fields emits M+1, field-name comparisons and
f*(X+1) puts. Its row is M+1+sum(name lengths)+f*(X+2). An array emits n*A+1
and n*(A+1) puts. For records these builders total:

`M+6X+51 + K*(2A+2M+8X+458) + T*(2A+M+3X+19)`.

For grants they total `1+G*(2A+M+2X+19)`. Every prospective key includes the
optional expiry field and an encode(20) allowance of401 before its actual
decimal encoding. Absent expiry only removes work.

The record's reconstructed tree has at most N=13+17K+7T nodes, 1+2K+T maps,
6+8K+3T fields, 10+14K+6T string nodes, K numeric nodes and two arrays holding
K+T items. Grants have N=1+5G nodes, G maps, 2G fields, 4G strings and one
array. Every map key is itself visited. Each node costs compare(0), each string
costs compare/copy of its bytes, each map/array emits its own buffer and puts,
and expiry Number text costs at most257 codec units (16 digits). Each such
codec event also records its actual input length in decoded coordinate[2].
The prospective record quote includes16*K decoded bytes; grants have no Number
nodes and contribute zero decoded bytes. Using 2B
conservatively covers string compare/copy bytes, including entered ASCII refusal
before a copy. Oversized strings refuse before their byte events.

The unchanged canonical preflight is encode(6*(actual visited nodes+source
bytes)+64). The row reserves `[6*(N+B)+64]^2+1` in addition to the builders and
walk, covering both that preflight and its bounded encode event. Prefix failures
perform a subset of these observed events; out-of-range expiry, invalid scalar, native-tree
or output-size refusal does not remove already spent work. This statement
covers the explicit logical events, not arbitrary native failure/CPU costs.

Six authored tests compare independent row literals with copy-size adjustments,
normal typed state bytes against the independent canonical writer, serializer
strings longer than clone field ceilings and a safe in-memory late output-size
refusal, original empty-grant preflight funding, and invalid plans. Empty grants
have eight actual units but need4904 prospective units to pass encode(70); tests
distinguish those roles. An original one-expiry precursor consumes16 decoded
bytes; a separate15-byte limit refuses despite ample work[10]. The128-key
case similarly uses2048 decoded bytes and refuses at2047, then checks that
the same failed observer cannot revive or add actuals on retry. No test submits
transactions, contacts a peer or
constructs a live credential/Source.

All six tests are NOT_RUN and coverage UNMEASURED. State history entries,
semantic/cursor root and clone multiplicities, typed ABI/crypto, native/backing
RAM/GC/window/lifetime/failure closure/final/page and the complete eleven-coordinate
factory remain separate unfinished obligations. Whole 5-1, 5, 7 and 8 remain
incomplete and original 5-2 through 5-6 ordering is unchanged.

### Complete state-apply event composition

The private apply row composes the original `applyObserved` call under its
actual selected-Registry/observed-command-parser route. It uses universal
schema ceilings of128 keys,16 services,128 grants and128 additions across
every intermediate array. These are prospective bounds, not observed success
counts or permission to accept direct handcrafted commands. Initial state
cloning, parsed arguments and actual insertion/removal guards must preserve
the field/cardinality constraints; the eventual factory must bind that route.

It combines the already bounded state/shared clone, two record serializations,
one grant serialization, new-key proof verification and one parsed history
entry. The second record serialization funds the original terminal-state and
version-exhaustion headroom check; its unused result does not make its work
optional. Reusing the shared clone row deliberately retains its unused
stream/empty-index baseline, a conservative excess over the actual state-only
clone. No extra full-stream clone is claimed to have executed.

All mutation branches are summed conservatively, rather than asserted to
execute together. With K=128, T=16, G=128, S the string descriptor, Q the key
descriptor, U the service descriptor, Z the grant descriptor, R the record
descriptor and H a slice descriptor, their explicit logical rows are:

| Mutation row | Upper |
| --- | --- |
| initial create comparison | 25 |
| create | 57K + keyCloneArray + serviceCloneArray + R + H + 222 |
| non-create state/authorization | 236 + 120G |
| activate | 24 + 57K + S + 1 |
| deactivate | S + 1 |
| insert key | 23 + 217K + 65T + (Q+6S+808) + (KQ+1) + K(Q+1) + H+1 |
| revoke key | 147K + 2(S+1) |
| update services | H + 25 + T(2U+4194483+65K) |
| change operator grant | H + 2235 + G(2Z+236) |

Here keyCloneArray=1+K(3Q+6S+809) and
serviceCloneArray=1+T(2U+4295). The authorization row includes actual controller
and operator/scope comparisons. Grant mutation includes its own strict40-byte
address codec and scope checks, sorted grant scans and owned storage/puts.
Service mutation re-enters the original URI parser despite earlier command
parsing, so its2048*T decoded bytes are included independently. The native
URI parser's CPU/allocator/RAM remain separate obligations.

Outside mutation, the row includes original agent/Registry guards182,
previous/current decimal preflights2*401, admitted/version copy17, version
setter S+1, lifecycle comparisons55, split-grant work G(2Z+29)+H+4, future
record copy/set R+2S+3, command clone262145, step copy and final state commit.
The universal vector is decoded[2]=99128, proofs[7]=128 and, for the ordinary
64-bit DATA layout, work[10]=10008433368160. Other target layouts adjust only
the explicitly copied type sizes; this is not physical RAM or a native CPU
measurement. The large logical value retains the original serializer
prospective preflights and is not a newly selected operating allowance.

Four authored tests compare an independent full vector with type-copy
coefficients; compare all11 handwritten normal history steps on one original
cumulative observer without resets; retain consumed work and unchanged
original state after independent decoded/proof/logical shortages; and verify
a normal2048-byte service URI consumes its own codec allowance plus the
one-digit admitted Number. The logical-shortage case intentionally uses a
clone-prefix allowance: actual output work alone cannot locate the original
canonical prospective preflight. The URI test separately rejects a2047-byte
allowance. These are owned offline DATA/unit cases using existing public
sample credentials; no attack reproduction, RPC or transaction is added.

All four tests remain NOT_RUN and each-file coverage UNMEASURED. This row
covers explicit state-apply events and entered refusal prefixes, not native
CPU/RAM, earlier caller claim/index rollback, whole-program root/copy/replay
multiplicity, windows/lifetime/recovery/final/page or production factory
sufficiency. A false proof still requires whole-run quarantine. The remaining
eleven-coordinate program contract must be completed before one connected
publication and remote native CI. Whole 5-1, 5, 7 and 8 remain incomplete;
original 5-2 through 5-6 ordering is unchanged.

### Parsed state history-entry event envelope

The private history-entry row covers one `replayStateEntryObserved` within the
actual observed-parser/apply caller's input envelope, including entered refusal
prefixes. It is not an upper for arbitrary handcrafted replayCommand values.
The command type alone does not prove the required input route or authorize
the eventual production factory.

The inspected route is original `replayRun.eventDeriving` -> observed ABI command
decode/generation -> `replayParseCommandObserved` -> its eight-operation argument
selector -> the same command passed by value to `applyObserved` and the history
entry helper. Successful parsing establishes operation length<=18. The apply
guard establishes raw command<=262144; its successful record and grant serializers
establish<=65536 and<=262144 before this entry call. No intervening step rewrites
the command operation/raw bytes. A production factory must bind this actual route;
a caller's declaration of those lengths is insufficient.

For prospective G<=128 removed grants, the entry first hashes command, record
and grants and encodes each fixed SHA-256 result to43 base64url bytes. Their
logical upper is595377. The now decimal preflight adds401; optional operation
comparisons add at most71, and target shape checks add15. With M/S/A the explicit
map/string/interface sizes and X=S+A, the removal builder contributes
`1+G*(2A+M+2X+19)` and the ten-field root builder `M+10X+125`. The original root
uses its history-specific ten-field helper, preserving the separate eight-field
limit for ordinary generated command objects.

The maximal target/removal tree has N=25+5G nodes,21+4G strings,2+G maps,12+2G
fields, one Number and one G-item array. The native walk is bounded with its
original global262144 source-byte cap, including all map-key nodes, buffers
and puts. The sole admitted Number costs up to16 decoded bytes and257 logical
codec units. Its encode(20) builder preflight is distinct. The final canonical
preflight remains `[6*(N+262144)+64]^2+1`; output refusal after encoding retains
all already observed work. The scalar quote returns decoded[2] and work[10]
only and changes no parser, serializer, meter, input/output limit or role.

Five authored tests compare independent event literals with copied-type size
adjustments; compare all11 normal parsed-command entries with the handwritten
history; distinguish a16-byte admitted-number allowance from15 with zero/128
primitive removal entries and a sticky failed-observer retry; cover a safe
in-memory escaped-string output refusal; and check invalid count/time guards.
Synthetic serializer/removal DATA is not a valid live version, credential,
authenticated history, peer message or Source. No test sends a transaction or
constructs executable attack contents.

All five tests remain NOT_RUN and coverage UNMEASURED. The eventual factory's
source binding, full state mutation/authorization/proofs, call multiplicities,
native CPU/backing RAM/GC, windows/lifetime/failure closure/final/page and full
eleven-coordinate sufficiency remain unfinished. Original 5-2 through 5-6
retain their order; whole 5-1, 5, 7 and 8 remain incomplete.

### New-key proof event envelope

The private proof row covers one original `replayVerifyNewKeysObserved` call
within the selected Registry and the actual observed parser/owned-state input
route. It does not cover arbitrary handcrafted replayKey fields. The apply
guard fixes Registry to the selected Sepolia identifier and agent length<=64;
the identifier is58 bytes, not57. Successful original command parsing and
owned-state cloning establish name<=32, algorithm<=24, state<=16 and signer
text<=189. The selected DID is at most132 bytes and its joined key name165.
Parsed added keys retain those fields through mutation; existing retained
keys have passed the owned clone. The quote itself authenticates none of
these facts: a production factory must bind the actual caller route.

Let A<=128 be the prospective added-key ceiling, N<=128 the retained-key
ceiling and S the explicit replayKey descriptor size. Counts need not assert
a particular successful transition. The scalar quote preserves three separate
coordinates:

| Coordinate | Prospective upper |
| --- | --- |
| decoded bytes [2] | 486*A |
| actual proof attempts [7] | A |
| logical work [10] | 133+A*(2*S+122027)+A*N*(S+522) |

One strict base64url binary field of n characters costs2*n*n+2*n+3 logical
units and n decoded bytes: codec, encode and comparison are distinct original
events. A32-byte key has43 characters and a64-byte signature86, giving3787
and14967 logical units. The adopted challenge contains its15-byte domain,
five two-byte prefixes, Registry58, agent64, name32, valid algorithm<=7 and
key material32: at most218 bytes. Its guards cost278 and its explicit
allocation/copies/prefix encodes458, plus the43-character binary operation,
for4523 logical units and43 decoded bytes. The original strict verification
admits at most32+218+64=314 codec bytes, costing98597 logical units together
with one proof attempt atomically. A complete proof bundle therefore costs
121874 logical units and486 decoded bytes.

The DID join costs133 even when no keys are added. Both outer loops together
cost A*(2*S+96). The Ed first loop and X second loop are disjoint on each
unchanged algorithm, so the total is at most A actual proof calls rather than
2*A. Every KEM key may still search every retained key. Each candidate costs
S+1 for its descriptor,166 for the joined name, and355 for signer comparison:
S+522. A matched signer's state/algorithm checks add at most57 per addition.
Those separate rows produce the logical upper above; counting proofs alone
would miss the quadratic A*N lookup cost.

Five authored tests compare handwritten three-coordinate event totals;
verify the original observed parser/clone followed by normal Ed and KEM
proofs, including optional no-addition and KEM-only cases; place a valid
32-character signing name last in a sorted normal Registry fixture; reject
independent decoded/proof/logical shortages after the first actual proof and
retain the same failed observer on retry; and check optional/invalid count
domains. Credentials use only the existing public RFC8032 unit samples.
There is no peer request, transaction, operating key or attack reproduction.

These tests remain NOT_RUN and coverage UNMEASURED. A failed signature can
consume an actual proof attempt without marking the meter failed; the whole
apply/window must still quarantine that result. This component does not
certify native curve CPU/allocator/backing RAM/GC, full state mutation and
authorization, repeated call multiplicities, window/lifetime/failure closure,
final/page work or production factory activation. Whole 5-1, 5, 7 and 8 remain
incomplete and original 5-2 through 5-6 ordering is unchanged.


### Complete semantic commitment event composition

The private commitment quote covers one original read-only semantic DATA root,
with separate prospective maxima for the confirmed and working claim/created
indexes. Each cardinality is at most2^20; a window may hold up to4096 queue
items and an optional pending command. Without a window, working counts,
queue and pending must all be zero. Actual original replay limits, owned
state/parser invariants and window guards must establish the corresponding
input route. A numeric quote does not authenticate a fold or any Registry
history and cannot transfer its execution observer.

Each stream root funds one record serializer at128 keys/16 services, one grant
serializer at128 grants, both lifetime index roots, two decimal preflights,
the successful record/grant raw domain hashes and its final state metadata.
Version0 can skip the actual serializers, while this universal quote retains
their allowance. Their entered refusal prefixes use the existing global tree
bounds; the later raw hashes use65536/262144 only after serializer success.
The closed lifetime-index quotes already contain their original8192-byte
canonical-preflight gap. No index padding or source work is silently removed.

For M=8192 metadata bytes and D=41 domain bytes, one inspected generic value
digest is bounded by33+(M*M+1)+(M+D+2)=67117133 logical units: kind copy<=32,
the original metadata encode preflight/actual encode, and its domain hash.
The fixed semantic root kinds are all within that bound. One262144-byte
semantic byte root costs2*262144+14+3=524305 logical units for the original
copy and domain hash. The two successful record/grant raw hashes cost327714.
These primitives add no decoded coordinate; each nonzero-version stream's
record expiry Numbers separately reserve2048 decoded bytes.

Writing Stream(C,I) for those complete stream components, V=67117133 and
B=524305, the original fold composition is:

- no window: confirmed Stream + V for the final fold;
- window: confirmed Stream + working Stream + 2*401 + V for the initial
  queue + Q*(401+2*B+V) for queued versions/entry/command/rolling metadata
  + optional B for pending bytes + V for the window + V for the final fold.

An initial queue digest is present even for Q0. The quote preserves confirmed
and working lifetime indexes independently and does not reset tombstones at
window boundaries. Pending has a bounded nonempty byte source when selected;
the actual commitment guard still requires its flag to match those bytes.
Copying the fold and taking a commitment are separate operations. In
particular, the original cursor copy performs a commitment before snapshot;
this quote alone does not fund that copy or later repeated prefix replay.

Four authored tests compare nine handwritten complete-root vectors with
portable copied-type coefficients; inspect normal no-window/pending/queued/
confirmed-and-working fixtures with a separately budgeted read-only observer
while keeping the original observer unchanged; derive a normal max-safe
expiry state through the actual observed parser/apply and distinguish its16
codec bytes from a15-byte quota, preserving state and failed-observer work;
and reject invalid count/window combinations. A separate small logical
allowance targets an observed traversal prefix, not a falsely claimed final
canonical boundary. Existing public sample keys remain offline unit DATA.

The tests remain NOT_RUN and coverage UNMEASURED. These are explicit decoded
and logical event uppers, not native CPU/backing RAM/GC measurements, a
whole-program root/copy/replay multiplicity proof, failure/recovery/final/page
closure or production factory qualification. No Source, ACK, policy activation
or helper-only publication follows. Whole5-1/5/7/8 remain incomplete and the
original5-2 through5-6 order is unchanged.


### Complete collection-cursor root and copy composition

The private cursor row returns separate uppers for one original cursor
commitment and one complete cursor copy. The latter includes that commitment
before its original array copying and semantic snapshot. Its inputs are
prospective immutable selected-input maxima: spine1..2^20, local effects up
to10*16384, pending blocks0..10, transactions per block1..16384, evidence
bytes1..64MiB and captured body bytes1..8MiB, plus the separate semantic
confirmed/working/window/queue/pending envelope. Local effect counts differ
from the lifetime effect count in the collection manifest.

Actual constructor/sealed input, decoded receipt effects, retained block
bytes, selected gas-profile/client bounds and original phase transitions
must establish those fields. The private cursor type or valid() alone cannot
prove arbitrary handcrafted body/gas/client values. Actual selected headers
and spine order are still rechecked; no supplied root replaces original data.

Let V=67117133 be the existing conservative generic metadata digest row.
Collection-pending-bytes has24 domain bytes, so a byte root of length n costs
2*n+27. Header/stop bytes65536 cost131099; one64MiB evidence string costs
134217755 and one8MiB captured trace costs16777243. The four closed gas
profiles have maximum length41, giving an83-unit profile comparison.

The journal/manifest rerender separately funds the original65536 encode
preflight/actual, owned journal string copy,8192 manifest encode/hash and
66-byte digest equality. Its maximal logical row is4362150070. This retains
the original serializer/canonical guards and is not a measured running cost.

For spine S, local effects E, pending blocks P and per-block transaction
ceiling T, the nonsemantic commitment rows are:

- spine roots: S*V+(S-1)*133, preserving every actual parent comparison;
- effects: E*(266+V), including spine-header and block-hash checks;
- pending: V + P*(133+headerBytesRoot+3*V+2*T*(evidenceBytesRoot+V)
  +83+evidenceBytesRoot) + E*(capturedBodyBytesRoot+V);
- optional stop record: headerBytesRoot, conservatively always reserved;
- final cursor metadata: V.

The pending V covers either the empty root or the final batch root. Each
unapplied block includes its original header, two independently rooted
transaction/receipt lists and optional post-fork gas bytes. Applied or
unreceived rows can consume less; they do not remove the prospective upper.
The full commitment also includes the previously bounded complete semantic
commitment with both independently retained lifetime indexes and queue.

The complete copy adds the semantic snapshot and original explicit copies:
S+E+4+P*(blockDescriptor+4*T*machineWordBytes+2)+E*stringDescriptor.
The source's first event deliberately counts numeric1+S+E bytes, not the
physical size of spine/effect arrays. The two pending list copies use the
original two-machine-word slice event. Immutable string backing remains part
of the separate physical resident-memory proof. These metrics are not a
native allocation/RAM certificate.

Five authored tests compare three independent nonsemantic root/copy literals
with bounded child components and target copy sizes; visit every phase of the
existing normal12-block/11-history synthetic DATA sequence; fund a pending
cursor copy with the SAME original cumulative observer and independent
mutable arrays; reject execution/snapshot transfer to a numeric inspection
observer after its allowed read-only work; and reject invalid input domains.
No real deployment constructor, Source, peer/RPC request or transaction is
created by those private synthetic fixtures.

Tests remain NOT_RUN and coverage UNMEASURED. One cursor quote does not fund
all repeated original-reader/cursor calls, native CPU/backing RAM/GC, whole
program/lifetime/failure/recovery/final/page work or factory qualification.
Full source call multiplicities and all eleven coordinates remain required
before one connected publication and remote native CI. Generic gates stay
closed; whole5-1/5/7/8 remain incomplete and original5-2 through5-6 ordering
is unchanged.


## Original successful cursor call multiplicities

The private root-call component now follows the actual original collection
callers, including both initial models, all prior T/C/S reconstructions,
the two same-C member equivalents and advancement before each later group.
It is cost DATA; it neither authenticates a cursor nor opens the generic
factory. Failed settlement, recovery and final scans remain separate paths.

For j complete earlier groups, one fresh healthy prefix has H(j)=3+9j cursor
commitments and K(j)=1+j cursor copies. Its ledger roots are 2+7j: the initial
constructor root, prior T/C pairs and original S with post-validation, then
the prefix's final validation. Each prior S also copies the cursor and
performs an additional final boundary commitment. Each full copy already
consumes its own initial cursor commitment.

The actual successful per-group schedule is:

| Caller | Cursor commitments | Cursor copies |
| --- | --- | --- |
| Two original planning roles | 2H+6 | 2K |
| Original T coordinator and two readers | 2H+9 | 2K+3 |
| Original C coordinator and two readers | 3H+24 | 3K+6 |
| Two recorded dispatch passes and same-C seal | 2H+22 | 2K+4 |
| Each member's two equivalents and dispatch validation | 12 | 2 |
| Original S coordinator and two readers | 3H+45 | 3K+12 |
| Advancement before a later group | 2H(next)+2 | 2K(next) |

The member equivalents freshly compare original inputs. They do not
reinterpret all prior semantic commands on every member. S and advancement
still perform full original semantic replay; their repeated work is retained.
Recording each dispatch pass adds a final model validation, and constructing
the same-C seal validates both original models. These are distinct calls.

For G prospective nonempty groups, G<=R<=32G requests, and J=G(G-1)/2, the
closed call rows are: initial models14G; full prefixes14G-2; successful
boundaries3G+14J; ledger roots108G-4+98J+10R; cursor commitments150G-4+126J+12R;
cursor copies39G+14J+2R. The last three obey commitments=ledgerRoots+copies
+boundaries. All rows monotonically cover smaller realized successful
prefixes, including initialization without a first T. No final advancement
is invented. G retains the original maximum floor(2^20/3).

The repeated cursor composition is
(commitments-copies)*oneCommitment+copies*oneFullCopy. The full-copy quote
already includes a commitment, so counting every commitment and full copy
independently would charge that root twice. Cursor state limits must cover
every possible original state, confirmed/working indexes and optional
queue/pending work; today's reported lengths are not a future envelope.
Any overflow or invalid child returns no partial quote. Even finite call
counts can produce a refused cost product. They do not prove sufficient
resources for the selected deployment-to-current history. Ledger metadata
encode/hash/comparison work remains outside this cursor-only composition.

Three authored test roots contain eight independent call-count literals,
a chronological expansion of the original caller grammar over all small
realized prefixes and a partially saturated two-group distribution, and
independent standalone/copy coefficients with invalid/overflow rejection.
The expansion checks call arithmetic rather than authenticating a real
reader or executing project/crypto/clock/RPC code. Tests remain NOT_RUN;
all connected individual 90% coverage targets remain UNMEASURED.

Full eleven-coordinate work, actual native/ABI/CPU and backing/peak/lifetime
RAM, window/tail qualification, failure/recovery/final/page work and the
production factory are still required. Only a complete connected bundle
will be applied and published for fresh native CI and ACTUAL review. Whole
5-1/5/7/8 remain unfinished; original5-2 through5-6 ordering is unchanged.


## Selected semantic call and mutation composition

The selected semantic component derives repeated window/block/subject-history
calls from the same original successful collection schedule. It does not
accept a caller-supplied replay multiplier or observed counts as authority.
A production factory must still bind G/R and every semantic maximum to the
same original immutable manifest and exact request grammar.

There are 14G-2 full original prefixes. Each interprets at most one complete
selected semantic journey. The three current S routes process only their next
boundary from the reconstructed prior state. Across all groups their new
deltas sum to at most three further whole journeys, not three whole journeys
per group. Thus F=14G+1 bounds the successful multiplicity. Cursor copies and
same-C member equivalents do not execute selected commands.

For prospective inclusive blocks B, content windows W, selected versions V
and present stop responses P, the rows are: models14G; window opens/closesFW;
block applicationsFB; selected mutations/history-item comparisonsFV; and
record-plus-grants stop serializations2FP. The B/W/V/P maxima are distinct.
V may be zero without removing required block/window/ABI work. All lifetime
selected versions and claim/created indexes retain their original counters.
Other-subject commands, claims and global logs are not bounded by V here.

The 2FP stop row is required by the actual code: a present state response
serializes BOTH record and grants before retaining its stop record, and the
later grants comparison independently serializes BOTH again. An absent
version-zero state enters neither serializer. Deactivation retains a present
versioned state. Window opens and the two same-C owned member copies also
call the observed ABI-pair initializer, bounded by FW+2R; native journal
configuration and first-input setup are separate obligations.

The state-only work composition is FV*stateApply + 2FP*(record+grants), using
the already bounded complete selected mutation (including both required
record encodings and new-key proof work) and independent stop serializers.
It preserves decoded2, signature verification7 and logical10. It includes
no command/event parser, block/native work, window clone, queue/history
comparison, root/copy, configuration or physical RAM cost. Zero state work
for V=P=0 is not a zero-cost collection certificate. Invalid or overflowing
products return the entirely zero quote and false without cap or renewal.

Three authored roots compare four independent original-call rows, preserve
proof and expiry coordinates with independent mutation/stop coefficients,
exercise the zero-version case while retaining other semantic calls, and
refuse invalid domains or finite-count cost overflow. They remain NOT_RUN;
individual coverage remains UNMEASURED. This is private cost DATA, not an
operating choice, permission or full eleven-coordinate sufficient budget.

Whole source/native/ABI/CPU/backing RAM/window/lifetime/failure/recovery/final/
page/factory qualification and connected native CI/ACTUAL are still required
before completion. One complete connected bundle remains the publication
boundary. Whole5-1/5/7/8 are unfinished; original5-2 through5-6 order remains.


## Original request and ledger control work

The private control component now prices explicit decoded2/logical10 events
for one original request copy, worst request classification, complete ticket
validation, nested consumption validation and ledger validation excluding its
separately composed cursor commitment. Native JSON/format/hash CPU, scratch
allocation/backing RAM and production authority are not established.

Request copies decode the original at-most8192-byte parameters, normalize
number values, reconstruct the original HTTP request and compare all seven
fields. Normalization precedes the builder's512-node/depth guard, so its
prospective node bound uses8192 raw JSON nodes. The closed builder can enter
the513th node comparison before refusal and counts at most8192 string bytes.
The existing body/head/wire input guards remain16384/512/16384. Reconstructed
body-plus-head scratch can reach16896 BEFORE equality to the input wire,
so using the smaller input-wire cap for that intermediate would omit work.
Existing encode preflights8192/16384/512 remain funded independently of the
smaller actual successful output. Every fixed admitted method fits32bytes;
the longest existing class has nine bytes. No new RPC method is permitted.

For m in1..32, one complete ticket revalidates every request, independently
classifies it, compares all seven fields again plus its class, canonicalizes
the65536-byte table envelope and8192-byte ticket envelope, then compares raw
record and digest. A consumption revalidates that ticket and its own8192-byte
record. A consumed ledger validates the ticket directly and AGAIN inside
consumption validation. Its control row therefore includes two ticket
validations, consumption record work, ledger encode/hash/root comparison and
entry comparison. Other ledger phases and root constructors use a subset.

The independent logical rows are: request copy403117431 with decoded8192;
request classification67117058 with decoded8192; ticket470318299*m+4362371417
with decoded16384*m; consumption=ticket+67141945; ledger control=2*ticket
+134259305 with decoded32768*m. Ledger encoding alone is67117227. These are
logical/preflight envelopes, not measured runtime usage or memory/CPU units.

The repeated successful ledger-control composition uses the already bounded
108G-4+98J+10R ledger-root calls (J=G(G-1)/2), conservatively at32members.
It does not include the cursor commitment already priced separately. Ticket,
consumption and settlement constructors/parsers OUTSIDE ledger validation
remain separate required source work. Invalid inputs or product overflow
return no partial quote and do not enlarge or renew an original allowance.

Five authored test roots check independent1/2/32-member literals, all eleven
closed request methods/classes, real typed ticket/C and prepared/admitted/
consumed/settled ledger validations, and composition with the original-shaped
synthetic cursor before/after T/C/S. These private DATA fixtures have no live
Source, RPC, deployment or execution permission. Finite unit refusal cases
retain body-preflight headroom and reject a short recursive decoded budget
without changing original DATA or reviving a failed meter. Repeated-count
and finite-domain overflow tests require an entirely zero refused quote.

Tests remain NOT_RUN and individual coverage UNMEASURED. The full all-eleven
source budget, block/command/event/global-claim/ABI/native work, physical RAM,
window/lifetime/failure/recovery/final/page and authenticated production
factory are still required before one complete connected publication and
fresh native CI/ACTUAL. Whole5-1/5/7/8 are unfinished; original5-2 through5-6
order and all existing gates/policies/protocol limits are unchanged.


### Record, diagnostic TLS and repeated result-context cost candidate

The original ledger-control row does not fund record operations outside a
ledger validation. `providerHistoryRecordControlUpper` adds separate private
2/10 envelopes for one T/C/S constructor, validator or parser. Each parser row
includes one bounded original caller text-to-byte copy. The S row alternatively
funds construction plus the actual fixed 32-result equality. It does not fund a
parser and a subsequent equality together. The extra parser allowance is
`4*8192+3 = 32771`; the separate S equality is at most 21040 for original closed
records. No malformed private string or matching former digest supplies input
provenance.

Both S construction and `providerHistoryResultContextObserved` validate T
directly and validate it again inside complete C validation. The context row
therefore sums the previously derived complete T and complete C rows. At one,
two and 32 members its logical values are 9732521377, 10673157975 and 38892255915;
the decoded values are 32768, 65536 and 1048576. Record-operation envelopes are
separate from the already counted ledger-root calls, so later composition must
use their original invocation sites rather than replacing the ledger row.

`providerHistoryTLSControlUpper` covers diagnostic TLS DATA construction or
parsing. Construction adds one actual request-head/body comparison, up to
`67*chains*certificates` digest-copy units, the existing 65536-byte canonical
preflight and output copy. Parsing adds one caller copy, the original codec,
canonical reconstruction and raw equality: `2*Q(65536)+3*65536+2`, where
`Q(n)=n*n+1`. It also reserves 65536 decoded bytes beyond its own complete T/C
context. Native TLS/certificate verification, HTTP/RPC response parsing and
physical allocation are not part of these diagnostic rows.

The successful original spool path requires two prior-prefix parses before
members after ordinal zero, two enlarged-prefix parses after each retention,
and three full-prefix parses for S. Thus a current m-member group invokes
`4*m+1` result-pack parsers and `2*m*m+3*m` TLS parsers, plus m TLS diagnostic
constructors. Each pack parser and TLS parser independently revalidates its
complete T/C context. All earlier successful groups are also parsed through
the previously derived fresh-prefix topology: 14 prior-group-weighted passes.
For prospective G/R, let J=G*(G-1)/2, E be the maximum early request weight
sum((G-1-j)*m_j), and Qs the packed-32 maximum sum(m_j*m_j) for at-most-G
realized groups. The call upper is:

- TLS constructions: R;
- result-pack parses: 4*R+G+14*J;
- TLS parses: 2*Qs+3*R+14*E;
- complete result-context validations: the sum of those three rows.

Independent maxima need not share a realization. Fewer realized groups can
increase Qs, so an exactly-G balanced square is not substituted for packed
members. For G/R=1/1, 1/32, 2/2, 2/33 and 3/96, the context uppers are 11, 2305,
54, 2778 and 8301. These are SOURCE-derived upper counts, not measured runtime
calls. `providerHistoryCollectionRepeatedResultControlUpper` composes TLS
construction and TLS parsing at the 32-member row, and adds only each pack's
own T/C context separately. The T/C context inside TLS is not added twice.
For G/R=1/1 with one chain/certificate its decoded/logical upper is
11862016/475060554678. Result-pack byte processing and member HTTP/RPC replay
remain separate. Overflow refuses the entire vector, including earlier
coordinates.

Six authored safe unit roots check independent numerical rows, original T/C/S
reconstruction with actual observers, diagnostic TLS DATA round trips,
source-topology enumeration including fewer groups, genuine nested/TLS decode
shortages with valid precursors, and invalid/overflow zero returns. No new
root has been run. No fabricated diagnostic metadata qualifies a live origin,
actual capture, deployment, production reservation or Source.

All new tests remain NOT_RUN and individual 90% coverage remains UNMEASURED.
The complete connected bundle, including original block/command/event/ABI and
native work, sufficient all-coordinate costs, physical peak/backing RAM/GC,
window/lifetime/tail, failure/recovery/final/page and production factory, is
still required before Git application/publication and native CI. This candidate
changes no protocol, allowance, observer, clock, permission or operating policy.
Whole 5-1/5/7/8 are unfinished; original 5-2 through 5-6 retain their order.


### Original pack and in-memory response byte-cost candidate

`providerHistoryOriginalPackControlUpper` funds one bounded opaque pack
construction or parsing, including one original caller raw copy for parsing.
It preserves verbatim payload bytes; only the descriptor is canonical JSON.
For descriptor cap D=65536, selected original object maximum P<=33554432 and
entry maximum E<=64, let L be the fixed magic length (39) and V the domain
length (38). The constructor logical bound is
`Q(D)+5*P+D+3*E+L+V+22`. The parser bound is
`2*Q(D)+4*P+2*D+134*E+3*L+V+157`, with `D+4` decoded bytes. Q(n)=n*n+1.
The descriptor retains its original 64 KiB encode preflight independently of
a smaller object maximum. Numeric shape acceptance does not prove a selected
record actually fits that object maximum or supplies its authentic table.

At P=33554432/E=64 the constructor/parser logical uppers are
4462805284/8724292282. Individual payloads are hashed/copied linearly under
the already bounded total; opaque payload bytes are not assigned a JSON codec
or re-encoded. Actual header/binary/magic/payload/raw copies, hashes, comparisons
and the four-byte codec/encode operations remain separate observed events.

`providerHistoryResponseControlUpper` describes the original in-memory
`readProviderHTTPObserved` route, using selected HeaderBytes H, BodyBytes B and
WireBytes W. Header reads include the H+1 sentinel, the body read preserves its
B+1 preflight and the same buffered reader's EOF probe is retained. Unicode and
envelope checks include two body compares and **three** body-sized codecs:
body token validation, envelope decoding and the raw jsonrpc value. The version
value is decoded before it is compared to 2.0, so assuming five raw bytes here
would not cover all entered input. The explicit decoded/logical row is
`H+W+3*B` / `13*H+4*B+20+Q(H)+Q(W)+3*Q(B)`.

The member-result row adds two response wire passes, two additional TLS raw
passes for its hash, both returned body/result string copies, the existing
8192-byte result-binding canonical preflight and its versioned digest. It
**excludes** the TLS parser and its own first caller copy, already funded by the
separate TLS/control component. Logical parser read events do not invent
another network request or physical retained-file read.

`providerHistoryCollectionRepeatedResultBytesUpper` composes normal current
result-pack constructions (4*R+G), all current/prior result-pack parses (the
previously derived packParses row), and one member-result byte parse per TLS
parse. Earlier healthy prefixes parse retained result packs without creating
a new one. Each row conservatively selects 64 entries and the bound selected
before execution. T/request packs and their reconstruction are separate.
At G/R=1/1, P=1048576 and H/B/W=512/4096/8192, this byte component is
432660 decoded /65397716670 logical; it composes with the earlier TLS/context
component without duplicating its TLS or nested T/C validation.

Six authored safe unit roots check independent constants, opaque request/result
pack round trips, normal content-length/chunked responses including a valid
escaped 2.0 value, member and full result-pack composition with the original
observer, genuine descriptor/RPC codec shortages after successful prefixes,
and invalid or finite-overflow zero returns. These are bounded in-memory DATA
checks, not live RPC/TLS, custody provenance or an operating Source. They have
not been executed. Individual 90% coverage is still unmeasured.

Native HTTP/JSON/token traversal/Unicode/hash CPU and physical allocation are
not qualified by weighted byte-event arithmetic. Other record/table/getter,
block/command/event/ABI/native routes, the complete eleven-coordinate bound,
physical/backing/peak/lifetime RAM/GC, original clock/window/tail,
failure/recovery/final/page and production factory remain required before the
one complete bundle is applied to Git, published or run in native CI. Protocol,
allowances, authority and policy gates remain unchanged. Whole5-1/5/7/8 are
unfinished and original5-2 through5-6 keep their order.


### Original request, table and outside-record control-cost candidate

`providerHistoryRequestControlUpper` composes explicit decoded/logical events
for the original HTTP request parser, table construction, complete request-pack
parsing and full table equality. It includes the original caller wire copy,
bounded header scan, body decode, parameter decode/normalization, reconstructed
wire copy and exact wire comparison. The original parser uses its own 512-node
and depth guards before its observed recursive descent. The existing request
copy bound is conservatively reused for constructor/normalizer work; the
original request row is 24576 decoded /671586687 logical.

The table constructor independently copies and classifies every request, then
selects its capture class. That selector may decode parameters again. The
member row is 24576 decoded /537351547 logical, including the V4 phase copy
even when an inert legacy observer does not charge that copy. The fixed table
canonical/copy/hash row is 4295098404 logical. The full equality row is 2812518
logical and includes all 32 slots, even unused ones. These rows do not fund
grammar-specific request generation before the table constructor.

Complete request-pack parsing adds the separately quoted raw pack parser,
every original request, one table construction and its final digest comparison.
At one, two and 32 requests with a 1 MiB pack maximum, decoded/logical rows are
114692/14098297187, 163844/15307235555 and 1638404/51575386595. This is exact
original reconstruction DATA, not an authenticated inventory, owner or permit.

For prospective G/R and previously defined J/E, successful raw request-pack
construction/parsing counts are G and 10*G+14*J. Original request parser calls
are 10*R+14*E. Table constructions, active members and full equalities are
26*G+42*J, 26*R+42*E and 15*G+28*J. The two planning readers are already inside
these counts; another writer-selection table is not added.

Outside ledger-root validation and TLS/result contexts, T operations comprise
one constructor, ten parsers and eleven extra admission validations per group.
C operations comprise one constructor, seven parsers and eight extra consume
validations. S operations comprise one constructor, two parsers and three
settlement reconstructions with fixed-array equality. Prior successful groups
are replayed fourteen times per later group. The resulting T/C/S operation
uppers are 22*G+28*J, 16*G+28*J and 6*G+28*J. SAME-C member equivalence uses
fresh original reads and validates/copies the existing model; it does not parse
or transition T/C again. Its separately counted ledger/cursor work is retained.

`providerHistoryCollectionRepeatedRequestControlUpper` expands raw packs,
originals, fixed/member table costs, equality and outside-record operations
without also multiplying the complete request-pack row. It additionally funds
the request-pack table binding, 22*G+42*J predecessor comparisons and
3*R+14*E settlement result bindings. At G/R=1/1 and a 1 MiB pack maximum,
this component is 27754536 decoded /1195153845351 logical. It returns zero on
any intermediate or later-coordinate overflow. It is not a complete budget.

Six authored safe unit roots compare independent literal rows, reconstruct all
eleven closed methods from exact HTTP originals, connect real table and full
request-pack routines with their separate equality observer, enumerate current
and archived call schedules without adding member T/C parsers, retain successful
prefixes on later decode shortages, and reject invalid/overflowing quotes.
These tests are NOT_RUN; per-file 90% coverage is UNMEASURED. The inert fixtures
provide no actual V4 Source, native execution, custody authentication or clock.

Source-specific nextTable generation, remaining raw/model/getter comparisons,
block/command/event/ABI/native work, complete eleven-coordinate bounds,
physical/backing/peak/lifetime RAM/GC, window/tail/failure/recovery/final/page and
production factory remain required before one complete connected bundle can
enter Git and native CI. No protocol, reservation, authority, clock or operating
policy is changed. Whole5-1/5/7/8 remain unfinished; original5-2 through5-6 retain
their order.


### Closed next-request generation-cost candidate

`providerHistoryRequestGenerationUpper` funds the explicit events before the
final table constructor in `nextTableObserved`. The original eight phases and
eleven read methods remain selected by the cursor's closed grammar. The larger
complete request-copy row conservatively funds the common request constructor;
its extra decoded params, normalization and field comparisons are retained as
an upper, although generation itself need not consume those events.

Each generated member additionally reserves its observed list append (2), a
maximum uint64 number tag (84), fixed trace-options encode (65), direct spine
lookup (133), and closed Registry calldata construction (27420). The last
bound uses rpcAgent's actual 64-byte ASCII cap: the largest historyItem call has
selector4 +head64 +length32 +paddedAgent64 =164 bytes. Its 26-byte selector
signature hash, data copy, data encode and hex string copy are all observed.
These alternatives are conservatively reserved for every member. The resulting
per-member decoded/logical row is 8192/403145135.

Effects selection scans the whole pending effect list before selecting a
bounded request suffix. Each pending effect performs one spine and one block
hash comparison, 266 logical units. Its prospective bound is local to one
pending content batch, at most 10*16384 effects, and must cover all pending
entries even when only the last request remains. A one-member quote with 33
pending effects is therefore 8192 decoded /403153913 logical. This is not a
claim that all generated request types execute every reserved alternative.

The successful aggregate subtracts request-pack reconstruction from the earlier
table counts: 16*G+28*J nextTable invocations and 16*R+28*E generated members.
It funds every generated member and the whole pending-effects scan per
invocation. It does not add table canonical/copy/classification costs again;
those are already included in the request/table component. G/R=2/3 with 33
pending effects gives 851968 decoded /41927620720 logical. Any invalid input
or later-coordinate overflow returns the complete zero vector.

Six additional authored safe roots check independent constants and aggregate
rows, closed Registry calldata against the pinned ABI, original grammar and
table composition across all eight phases, a last-member request that still
scans all 33 pending effects, genuine constructor/tag budget failures, and
invalid/overflow returns. Normal fixtures keep the selected timestamp in both
cursor and semantic fold from construction. An earlier content-plan fixture
changed that timestamp only on the cursor after sealing; it now supplies the
selected time before constructing either state. These are inert in-memory DATA
scenarios, not a qualifying live source, retained S transcript or chain proof.
All new tests remain NOT_RUN and individual 90% coverage is UNMEASURED.

Native Keccak/JSON/format/rpcWord/hex work and physical allocation are separate
from these event counts. Remaining raw/model/getter, original block/command/
event/ABI/native routes, all eleven sufficient costs, physical/backing/peak/
lifetime RAM/GC, clock/window/tail/failure/recovery/final/page and production
factory must still be completed before Git application/publication and native
CI of one complete connected bundle. No reservation, observer, protocol,
permission or policy is changed. Whole5-1/5/7/8 remain unfinished and original
5-2 through5-6 retain their order.


### Original outer object, inventory and frame-control cost candidate

`providerHistoryOuterControlUpper` prices the explicit events for an original
profile rebuild, highwater encoding/parsing/predecessor, stored object,
inventory and outer custody frame. It accepts numeric bounds only, not an
authenticated config, Source or resource certificate. Profile metadata M is
bounded by its actual 1 MiB cap; highwater records use H=8192 independently.
The profile domain has 30 bytes in every closed revision, so its logical row
is Q(M)+M+32. A preflight checks remaining work; it is not itself a charged
actual event. Every entered profile preflight and rebuild must still fit.

One highwater encoding includes one profile rebuild and Q(H). Parsing adds
another Q(H), H decoded bytes and its exact raw equality. A predecessor check
rebuilds profiles three times and encodes/hashes its before-pin. An outer frame
constructor rebuilds seven profiles, reserves six encode rows bounded by H,
hashes three highwaters and its three own versioned records, and
copies those three record strings. Frame parsing decodes only intent, rebuilds
all three records and includes their V4 copies and exact comparisons. It does
not decode commit/frontier as additional JSON bodies or include caller copies.

Stored objects retain opaque bytes and perform their independent copy/hash;
the maximum closed object-domain length is 42. Their logical row is 2*P+45.
Each inventory independently rebuilds and checks every supplied object. Its
aggregate raw input is at most min(objects*P,64 MiB), rather than eight times
an unconstrained object maximum. Its own canonical bound is min(M,8192).
Parsing rebuilds that inventory and compares both expected digest and exact
raw bytes; it does not JSON-decode the retained inventory as another body.

At M=65536/P=1048576/objects=2, the profile, highwater-encoding/parsing,
predecessor, frame-construction/parsing, object and inventory-construction/
parsing logical rows are respectively 4295032865,4362141730,4429266980,
12952215688,30467957209,30535139808,2097197,71319946 and71344657. Only highwater
and frame parsers have decoded rows, 8192 each. Native hashing/JSON/format and
physical allocation remain outside these event bounds.

The successful current/prior T/C/S wrapper subgraph has three frame and three
inventory constructions per group. Inventory parses are 19*G+42*J. Outer frame
parses are 20*G+42*J+2*R: SAME-C equivalence omits inner T/C parsing, but its
current-C anchor still parses an outer frame and two pins, once per seal and
once per member role. This distinction preserves the earlier inner-record
counts while funding the actual additional outer work.

Outside inventory's own object reconstruction, pack/record object operations
are 14*G+28*J and22*G+42*J. Their independent reads are 12*G+28*J and19*G+42*J,
each additionally copying the returned raw text and checking its expected
digest. Explicit highwater encodings outside frames are 10*G; parses are
42*J+2*G+4*R. Normal T/C/S inventories have at most two objects. The aggregate
uses their existing inner rebuild cost once, rather than adding its object
operations again. At G/R=1/1 and the bounds above it gives 229376 decoded /
834985939814 logical. A later-coordinate overflow refuses the complete vector.

Six additional authored safe unit roots check independent literal rows,
normal request/result/T/C/S objects and inventories, retained config/profile/
pins and all three frame phases, original call enumeration including member
anchors, genuine second-profile/canonical/codec budget refusal prefixes, and
invalid/late-overflow returns. Inspected fixture scopes and configs do not
replace a live observer, select another Source, acquire an ACK or enable a
request. Tests are NOT_RUN and each-file 90% remains UNMEASURED.

Initial first-P/catalog control interpretation, remaining raw/model/getter/
readback comparisons, names and other profile callers remain separate. Full
source block/command/event/ABI/native work, all eleven sufficient costs and
bindings, physical/backing/peak/lifetime RAM/GC, clock/window/tail/failure/
recovery/final/page and production factory remain required before Git/native
CI of one complete connected bundle. No protocol, quota, Source, permission,
clock or policy is changed. Whole5-1/5/7/8 remain unfinished, with original
5-2 through5-6 order preserved.


### Original legacy inventory, first-P control and scan-catalog cost component

The new legacy-control row prices only explicit decoded/logical coordinates2/10 in the successful original readers. Its numeric inputs are selected metadata M<=1MiB, object P<=32MiB, first-read object allowance O<=1Mi entries, first-read raw allowance B<=64MiB, per-inventory E<=256, scan attempts A<=64 and positive chunk maximum C. They are prospective DATA, not a profile binding, complete hold or execution capability. Four immutable inventory pages cap object references at N=min(O,4E); aggregate object raw is bounded by T=min(B,NP). No fifth page is invented when O is larger.

Let Q(n)=n*n+1, H=8192, L=65536, K=93 (maximum closed kind26 + colon1 + hash66). A reference pair costs4K+3: two copied keys and their comparison. Actual legacy kinds have maximum domain43. An individual domain-bearing object reserves decoded P/logical Q(P)+2P+46; plain request/params/response/TLS objects retain ordinary SHA and skip the actual codec. Across the complete first-P load, sum(n*n)<=P*sum(n)<=PT avoids claiming N objects all occupy P when the original raw allowance is smaller. Pairwise duplicate checks still cost185*N*(N-1)/2.

The legacy inventory parser decodes the raw page ONCE, checks adjacent sorted references, hashes it and retains its raw string. It does not sort, reconstruct object payloads or canonical-reencode the page. Its row is decoded L/logical Q(L)+2L+35+375(E-1). The empty-anchor constructor has no entries/sort/codec, but keeps the original L canonical preflight, hash and raw-string copy: Q(L)+2L+35. The first-P load and catalog each independently decode every raw inventory once more to obtain its previous pointer; those are real additional invocations. First-P load separately rereads/rebuilds every selected original; the catalog uses original references and does not reread object payloads. Filename copies, expected digest checks and append events at these actual call sites remain explicit.

A first-P selected-object lookup reserves N reference comparisons, the original raw copy, independent domain hash and expected digest comparison. The successful selected/exchange path performs twelve object lookups and three unique-kind scans. This lookup cost does not supply the selected policy, admission, ticket, consumption, settlement, HTTP/base64/header or semantic parsers that consume the returned bytes. Their remaining rows must still be connected.

The initial-model control row keeps both original pin-byte copies and HEAD/immutable comparisons, original frontier/commit hashes, frontier decode and its separate L-preflight canonical rebuild/comparison. The first-P C/S control row checks the previous frontier and twice decodes each original intent: once for its direct check, once inside commit reconstruction. The commit encoder always preflights1MiB, even with a smaller selected M. No actual work is fabricated for a preflight; the complete quote leaves enough remaining for that original check.

One initial catalog reconstructs three scan codecs: readPrefix, its final readHighwater, and the catalog binding. Each keeps profile reconstruction and optional guardian's two encodes, one decode and canonical comparison. The fixed scan portion has eight encodes, five hashes and six comparisons. Each prior attempt adds six encodes, two decodes, four hashes, five comparisons and one original policy-byte copy. Each chunk adds six encodes, five decodes, four hashes and seven comparisons: both chunk/command originals in both roots and the separately generated expected command. Scan encoding keeps selected-M preflight while successful raw records remain <=H. These rows price each encoder with Q(M); each decoder with Q(H). Reader and catalog receipt-name copies are separate calls.

The successful repeated composition uses the independently derived initial-model14G and catalog16R+16G-2 counts. At M65536/P1MiB/O64/B64MiB/E256/A4/C32:

| Groups / requests | Decoded upper | Logical upper |
|---|---:|---:|
| 1 /1 | 1,010,122,752 | 1,048,439,013,903,752 |
| 1 /32 | 1,964,982,272 | 1,569,667,739,554,328 |
| 2 /3 | 2,054,897,664 | 2,115,793,586,399,662 |
| 2 /33 | 2,978,955,264 | 2,620,208,482,190,542 |
| 3 /96 | 5,902,647,296 | 4,713,206,676,127,908 |

Valid G349525/R11184800 yields finite decoded688156179906560 but logical549372721301211542288 exceeds uint64: the complete result must be zero, never a partial decoded quote. These are source-derived sufficient upper components, not measured actuals or proof that an existing selected policy accepts the composed total.

Eight new authored UNIT roots cover independent numeric oracles, all32 closed kind/domain maxima and ordinary SHA bytes, retained inventory parsing/empty anchor, immutable retained scan DATA, frontier/commit primitives, genuine atomic codec/canonical-preflight shortages, repeated readers/page ceiling and sticky late overflow. They are NOT_RUN. The new product file is added to the unchanged each-file90% connected inventory; all269 individual targets remain UNMEASURED.

This component excludes selected first-P parsers/request construction/initial-root encoding, general namespace-name/map work, remaining caller/raw/model/getter/ABI and other profile operations, native CPU, physical/backing/peak/lifetime RAM/GC, clock/window, failure/recovery/final/page/factory and complete all-eleven costs. No project runtime, Git application, commit, push, CI, Source or operating-policy activation is claimed. Only one complete connected bundle may enter Git/native CI. Whole5-1/5/7/8 stay unfinished; the original5-2 through5-6 order is unchanged.


### Selected first-P interpretation cost component

This separate component funds the successful selected first-P policy/enrollment/admission/T/C/S interpretation, original request construction, in-memory HTTP/RPC/header parsing and initial commitment. It composes with the legacy-control row: original filesystem reads, four-page load, twelve object lookups and three unique-kind scans are already separate and are not added again. It grants no Source, custody, request authorization, consensus finality or complete program cost certificate.

Let Q(n)=n*n+1, P<=32MiB be the selected original object maximum, L=65536, A=min(P,L), and capture H/B/W retain their actual header/body/wire limits. Fixed config policy bytes have their original L ceiling independently of P. The selected record chain requires decoded6L+6A+2P and logical9Q(L)+6Q(A)+2Q(P)+Q(4096)+11L+10A+4096+31+D, where D=261 is the sum of the actual three policy domains, five record domains and segment domain lengths. This includes policy subtree Marshal calls, intent's independent policy/limits handling, ticket's4096 segment preflight, extra admission-field decoding, original exchange/wire-binding decoding and the separately reparsed configuration policy. Scalars/native JSON/hash/field validation do not acquire invented observed events.

Base64's decoder event charges ENCODED input length. For maximum n, E(n)=4*floor(n/3)+{0,2,3}[n%3]. One exact unpadded URL-base64 parse needs decoded E(n), logical Q(E(n))+Q(n)+2E(n)+1, including raw re-encoding and canonical spelling comparison. Both the16KiB request body and B-sized response are independently parsed; treating decoded output n as the input event size would underfund this route.

The existing HTTP row already parses the original RPC envelope once. The first-P caller deliberately ignores that returned result and decodes the same original body again. Between those parser calls the actual caller separately compares the parsed HTTP body to the independently decoded exchange response, adding2B+1 logical units exactly once. The second RPC parse itself needs a separate decoded3B/logical3Q(B)+3B+3 row. Method-specific finalized header parsing then needs its own raw-token/map/four-field codecs and two uint64 quantity codecs/16-digit encoder preflights. Its conservative decoded6B+32/logical6Q(B)+2B+4Q(16)+275 row preserves ordinary provider field order and does not rewrite original JSON.

Transport metadata and both independent response-header arrays retain the original1MiB canonical preflight Z=1048576. Their combined encoder/hash/comparison/header-count-copy row is3Q(Z)+5Z+min(H/4,128)+4. Expected intent/ticket/retained-root digests, expected request wire, params and request-body comparisons are separate from the original-object lookup hashes. The fixed request constructor reuses the larger existing request-copy row once; this is not another nextTable generation or table parse. Initial-root construction retains its8192 preflight, versioned hash and two selected-header checks. The third selected-header check and native semantic initialization belong to the still-separate new-cursor constructor.

For P1MiB/H8192/B131072/W262144, the total interpretation component is decoded4931617/logical5885891016550. Repetition uses the original14G initial-model invocations, so G1 is69042638/82402474231700 and G2 is138085276/164804948463400. It is independent of R once G is selected, unlike the separate catalog row. At valid G349525/R11184800 decoded24132128046950 is finite but logical28801724805834942500 exceeds uint64; the whole vector must refuse and return zero.

Eight new UNIT roots cover independent small/ordinary/maximum literals, actual typed record parsers, all Base64 remainder cases and encoded-byte counters, unchanged finalized-header DATA, both actual HTTP/RPC envelope invocations and their intervening body equality on the same observer, fixed request/root work, genuine codec shortages preserving earlier observed work and invalid/late-overflow refusal. They are NOT_RUN. Total authored roots292 remain NOT_RUN and all270 individual coverage targets remain UNMEASURED.

Remaining caller/raw/name/model/getter/ABI work, new-cursor/semantic initialization, native CPU, all-eleven sufficient composition, physical/backing/peak/lifetime RAM/GC, clock/window/failure/recovery/final/page and production factory are still required before one complete connected bundle enters Git/native CI. No new runtime, Git, push, CI or operating activation is claimed. Whole5-1/5/7/8 and original5-2..5-6 stay unfinished and ordered.

The first source review identified the missing explicit caller-body comparison; its original REVISE report and frozen source are preserved. The successor adds only that2B+1 allocation and corresponding independent literal/document/regression changes. This is a source accounting correction, not evidence of an executed runtime failure or measured total-budget shortage.

### First-cursor and journal initialization component

This component funds the separate first-cursor constructor called by every original initialization model. It does not add another semantic window, cursor root/copy or ABI parse. The previous interpretation row funds two selected-header checks; this constructor has its own third check265, a four-hash-text copy265, the cursor plus one FinalizedHeader backing value, semantic fold and temporary ObservationChecks copies. Fixed structure sizes are logical copy arguments only. They are not heap allocations, backing-string ownership, physical/peak RAM, native CPU qualification or a complete program cost certificate.

The observed semantic path now calls the same original finite journal validator/encoder exactly once with its original observer. Legacy journal and pure semantic callers keep a nil observer and exactly the existing JSON schema, validation and returned bytes. Native JSON encoding keeps a separate64KiB preflight after finite validation and before json.Marshal. Only the actual successful encoded length is charged, including when MaxStoredBytes subsequently rejects the encoded result. A preflight failure keeps all earlier actual copies and never records the preflight as actual work or retries with another observer.

The closed default JSON schema has six objects,36 keys with387 raw key bytes,12 string values of at most1498 raw UTF8 bytes and19 numeric values of at most20 decimal bytes each. Object/key/string punctuation uses174 bytes. Standard JSON escaping is at most six output bytes per raw UTF8 input byte, so174+387+6*1498+19*20=9929, below65536. This is a source-derived encoder envelope, not a new journal wire limit: the original unobserved journal API has its existing MaxStoredBytes check. No request header, RFC9421 coverage or protocol algorithm is changed.

Let Q=65536*65536+1, C=sizeof(ObservationChecks), F=sizeof(providerHistorySemanticFold) and U=sizeof(providerHistoryCollectionCursor)+sizeof(FinalizedHeader). The configuration logical upper is C+1+Q. Semantic initialization includes that configuration row and adds F+1. Cursor initialization includes that semantic row and adds265+265+U+1. These are nested rows and must not be summed a second time. Their other ten work coordinates are zero: an encoder does not invent decoded bytes, a block, a proof or a network call. Repetition is14G*cursor, independently of R within the original valid G<=R<=32G domain. For the fixed closed sizes the full valid domain remains finite; invalid topology returns the entire zero vector.

Eight authored safe UNIT roots check actual legacy JSON bytes against a separately declared original schema, ordinary/escaped maximum configurations, reflection of the closed finite schema, actual encode versus preflight charges, unchanged invalid/storage refusal, exact original selected cursor and observer, genuine shortages at successive initialization calls and14G composition. They are NOT_RUN. Total authored roots300 remain NOT_RUN and all272 individual coverage targets remain UNMEASURED. Standalone formatting is not compilation or a passing test.

Both actual independent first-P model readers already enter this observed constructor. No factory, Source-ready credential, admission/dispatch authority, operating profile or network policy is introduced. Remaining caller/raw/name/model/getter/profile/ABI/native work, all-eleven sufficient composition, physical/backing/peak/lifetime RAM/GC, clock/window/failure/recovery/final/page and production factory must still be connected before the single complete bundle enters Git/native CI. Whole5-1/5/7/8 and the ordered5-2 through5-6 remain unfinished. The earlier published green CI never validates this unpublished source.

The first independent initialization review identified a prospective coverage conflict: native Marshal-error and oversize-only return blocks cannot be reached under the same closed scalar schema. The successor keeps every error/size/storage check but uses the reachable original storage refusal's common return. It keeps actual encoding before storage refusal, and meter event/preflight shortages remain sticky. The unreachable current-schema oversize guard now returns ErrConfiguration without separately poisoning the meter; this defensive-state difference is explicit and does not create an admitted path under the9929-byte proof. The genuine small-storage test also confirms a subsequent valid DATA call on the same healthy observer retains both calls' actuals. No fault-injected encoder, weakened90% gate, measured coverage or CI failure is claimed. The frozen R1 and REVISE report remain preserved.

### Retained ordered read and rolling-root component

This component funds the additional observation of each ordered original read, the rolling transcript commitment and SAME-C member refresh. Actual file-read events already belong to the read-byte row. Recording adds a separate raw SHA, two owned filename copies and a descriptor copy; refresh adds a separate raw SHA and digest comparison. Neither is another HTTP request or a decoded semantic object. The recorder constructor now also charges its fixed logical structure copy before allocating it on the original observer. This is not physical allocation/backing/peak RAM or native CPU qualification.

The exact recording interval contains the full healthy prefix H(g), then current T10 and C9 reads. Catalog derivations occur outside that interval and are not added to these transcripts. Thus each recorded original has H(g)+19 rows, including repeated HEAD/highwater inputs. Let SH=sum H(g) and MH=sum m(g)*H(g), using the already derived conservative original-prefix upper. Two roles record2*(SH+19G) rows. Their member refreshes process2*(MH+19R) rows. Each role computes one root while recording and again while sealing; each role then computes its root before and after each member's refresh. There are4*(G+R) base root calls and twice the sum of recorded/refreshed rows in the rolling-row calls.

The recorder's original capacity remains catalogEntries+firstObjects+128. A prospective maximum-row upper is33+firstObjects+23*(G-1)+2*min(32*(G-1),R-1). The numeric component refuses if that maximum cannot fit its original capacity. It does not adopt the actual transcript length or increase its cap at runtime. The finite domains keep catalogEntries/firstObjects<=1Mi and the original metadata/object/wire/first-read maxima; production source binding remains mandatory.

For byte envelopes retain D=max(metadata,8192), U=5D+firstRaw, V=21D+2P, L=wire+65536, J=G*(G-1)/2 and E=max sum prior-member counts from the earlier prefix derivation. Then SB=G*U+V*J+L*E and MB=U*R+V*E+L*(R*R-minSumSquares)/2. A current T/C suffix adds P+18D: the request pack uses P, while successful ticket/consumption records retain their existing8KiB ceiling. Recorded bytes are2*(SB+G*(P+18D)); refreshed bytes are2*(MB+R*(P+18D)). These bounds include original selected read maxima, not today's claimed row lengths.

One recorded row adds rawBytes+2*nameBytes+132 logical units, conservatively rawBytes+642 for nameBytes<=255. One refreshed row adds rawBytes+134 for raw SHA and66-byte digest equality. Each base rolling root retains Q8192+8192+37+2=67117096; each row retains Q8192+8192+36+2=67117095, preserving its separate canonical encoder preflight and domain/NUL hash. Seal root/model comparisons add532G, and member root comparisons add798R. The fixed recorder-copy row is2G*(sizeof(providerHistoryReplayReads)+1). Only work10 is nonzero. Model/cursor roots, reflection of fixed context/claims/ancestors and other caller work are separate.

At O64/F64MiB/D65536/P1MiB/wire262144, independent source literals excluding the target-dependent fixed recorder size give:

| G /R | Recorded /refreshed rows | Base /row root calls | Logical upper excluding recorder copies |
|---|---:|---:|---:|
| 1 /32 | 194 /6208 | 132 /12804 | 872825598228 |
| 2 /3 | 442 /682 | 20 /2248 | 152942059658 |
| 3 /96 | 1104 /35328 | 396 /72864 | 4933561440348 |

Valid scalar G10000/R320000/O1048576/catalog900000/F64MiB/metadata1MiB/P32MiB/wire16MiB has maximum1918522 rows below capacity1948704. Its raw-byte totals remain finite, but rolling commitments alone require131435929309500420000 logical units, exceeding uint64. The whole component must return zero, including earlier finite fields. No partial quote is usable.

Seven authored UNIT roots check independent count/byte literals, real duplicate file reads and exact additional hash/name/descriptor charges, independent ordinary JSON/SHA for both genuine role transcripts, a real late rolling-row preflight shortage retaining its base actuals, constructor observation with closed/cancelled observer refusal, invalid original capacity/domains and late overflow. They are NOT_RUN. All307 authored roots remain NOT_RUN and273 individual coverage targets remain UNMEASURED; inventory adds only the new product file and keeps every existing gate.

General names/maps, fixed context/claim/ancestor reflection/comparison, other caller/raw/model/getter/profile/ABI work, native CPU/all-eleven sufficient composition, physical/backing/peak/lifetime RAM/GC, original clock/window/lifetime/failure/recovery/final/page and production factory remain unfinished before the single complete bundle enters Git/native CI. No Source, ACK, hold, profile, endpoint, operating activation or helper-only publication is introduced. Whole5-1/5/7/8 and the original5-2 through5-6 order remain unchanged.


### SAME-C fixed context, claims and ancestor comparison component

This separate logical-work component retains every original fixed-context, current-T/C claim and complete ancestor/fact comparison. Pointer equality is not pointed-child data: the original explicit snapshots and fresh retained-byte reads remain required. Its successful multiplicities come from both independent roles and both before/after boundaries; it does not grant a Source, dispatch permission or a complete program certificate.

The six closed reflected shapes are fixed context, C frame, read context, claim, ancestor value and ancestor fact. Their visited-node counts are1007/73/110/196/396/122, their string-leaf counts are227/44/44/83/140/28 and their maximum depths are6/4/4/5/6/4. The actual observer charges one compare(0) per visited node and never traverses pointer children. Type-shape regression tests inspect the native Go types, including the three scalar/pointer fields of time.Time, rather than trusting the passive source declaration walker. Changing a nested value type requires rederiving these ceilings.

Let M=1048576 and D be the prospectively selected SUM of both original directory-string lengths. The quoted domain is2<=D<=2097152. This is a separate candidate-factory obligation, not an existing legacy directory cap or an operating-policy selection. The eventual complete pre-H0 factory must derive and bind D from its original immutable configuration; an arbitrary reported D is not production evidence. Every other original string already has a smaller source ceiling: policy64KiB, the closed journal9929, manifest/inventory8KiB, request params8192/body16384/head512/wire16896, metadata1MiB, and closed identifiers/hash/names. In particular, archive original.response/tls and capture.response/tls contain66-character hashes, not the potentially16MiB response. Raw response-read cost remains in the read-byte row. The six string-byte upper sums are225M+D,44M,42M+D,83M,140M,28M. Add the actual target type size to obtain each comparison weight. A comparison spends2*weight+1; these sizes do not prove native CPU, physical allocation or backing RAM.

For G groups/R requests retain J=G*(G-1)/2, E=max sum of prior member counts and Smin=min sum of squared nonempty group sizes under1<=m<=32. The original domains remain1<=G<=349525 and G<=R<=32G. Weight-walk invocation counts are[G,G,2G,2G,3(G+J),E]. Equality counts are[G+4R,G+2R,2G+4R,2G+8R,3(G+J)+12(R+E),E+2*(R*R-Smin)]. Independently maximizing E and minimizing squared sizes can be conservative; for G2/R4 the fact count upper19 covers actual maximum18. This is deliberate upper slack, not an observed actual.

Each original chain count adds3(G+J) compare(0) events. There are G fixed seal copies,4G+8R fixed claim copies,3(G+J) ancestor rows in G exact arrays and E fact rows with one uint64 weight each in J member-array pairs. Preserve each actual copy event's extra1. The additional current-C model-head equalities add532R and were explicitly excluded from the previous read-retention row; its532G+798R must not be added again here. Other models/cursor commitments, raw-body and namespace/name work remain separate.

Independent numeric literals excluding target-dependent copy/compare sizes give total logical9309261240 for G1/R1/D2;236851331730 for G1/R32/D2048;34473116451 for G2/R3/D2048;45864879604 for G2/R4/D2048;1778616761175 for G3/R96/D2048. Target-size coefficients are the explicitly listed original equality/copy multiplicities. Valid scalar G20000/R640000/D2097152 overflows repeated typed comparison work and must return the ENTIRE zero result, including finite earlier fields. All ten other work coordinates are zero.

Seven authored safe UNIT roots cover those independent literals, actual reflected type shapes and per-node debits, genuine third-stage claims and both earlier archived child facts, independently counted role/member/ancestor topologies, an actual DATA walk followed by a one-short equality event with sticky prefix retention, invalid prospective domains and valid late overflow. They are NOT_RUN. All314 authored roots remain NOT_RUN and274 required individual coverage targets remain UNMEASURED. The genuine precursor uses the existing legacy multistage fixture; this proves no actual collection/runtime/factory qualification until executed and separately connected.

This pure component changes no legacy path, clock, quota, wire coverage, algorithm, selected profile or operating policy. It requires source-bound original inputs; it is not a sufficient cost derivation by itself. Remaining general namespace/name/map, caller/raw/model/getter/profile/ABI/native CPU, complete all-eleven composition, physical/backing/peak/lifetime RAM/GC, original clock/window/lifetime, failure/recovery/final/page and production factory remain required before ONE complete connected bundle enters Git/native CI. Whole5-1/5/7/8 and the ordered5-2 through5-6 remain unfinished. Published green CI does not verify this unpublished source.


### Allowed catalog namespace construction component

This logical-work row funds the remaining successful catalog filename construction, prior checkpoint-count comparisons and both allowed-map copies. It preserves original authentication: observed directory entries never become the allowed set, cached historical catalogs never replace original-prefix reconstruction, and filename generation grants no write, ACK or dispatch authority. Original scan/receipt/inventory/object-reference names already priced by the legacy catalog component are excluded. Directory enumeration/identity work, retained-read recorder name copies, raw reads and native map allocation/hashing/CPU/RAM remain distinct.

The original bridge has sequence4. Its authenticated highwater satisfies ordinal-sequence<=setupRecoveries, so the original highwater-name loop has at most5+setupRecoveries rows. Original setup records number at most1+setupRecoveries. The namespace's remaining initial indexed-name event sum is31H+101S+170A+532, hence at most788+132*setupRecoveries+170*attempts. Setup recoveries remain in0..64, scan attempts in1..64. These are original pre-H0 selected maxima, not later observed smaller counts. This is tighter and better sourced than treating the initial ordinal as the maximum whole-program commit count.

Let K=16R+16G-2 be the already derived initial-catalog invocation count, J=G*(G-1)/2, E the maximum weighted prior-member count and X=R*R-minSumSquares. Each successful group's16m+14 catalog derivations retain every earlier T/C/S; each of the two nonfinal advance derivations additionally includes the just-settled stage. Thus repeated prior-stage rows are at most16E+16J, and repeated prior-member rows at most8X+16E. A prior T/C/S adds five stored-object names447, three inventory names249, fifteen indexed names462 and three compare(1) events6:1164. Each archived member adds two independently regenerated and compared names of at most344 logical units each. Those comparisons must not be mistaken for raw-spool digest comparison or read-recorder copies.

Current T/C extension pairs occur at both dispatch endpoints, every member catalog and eleven S catalogs, giving16R+11G pairs. The twelfth S prefix derivation is an initial allowed set that preparation discards before deriving the selected dispatch set. C preparation separately extends T once. The two extensions cost413+329=742; hence the extension row is742*(16R+11G)+413G. S's prepare/reader/generic/outer boundaries clone both maps eight times; with T1 and C1 there are10G clone calls. Prepared new names cost T413+C329+8*S416=4070G. Each original map admits at most entries names of at most255 bytes, so copying BOTH maps costs at most512*entries per call, including every copy event's extra1. This bound does not trust a claimed observed name-byte sum or establish backing RAM ownership.

For member ordinal k starting at0, the catalog checks regenerate32k+15 current spool filenames; completed S's eleven selected catalogs add22m per group. Total current filename calls are bounded by16*maxSumSquares+21R. The maximum squares use packed32-member groups, since fewer optional groups can increase the current prefix. Each generated current name is at most114 bytes and its copy event115. Pending HEAD names are generated only before rename: T1/C1/S3, giving5G calls of34 units. Physical writer/rename names and direct original-spool readers outside allowed-set construction remain separate caller work.

The component populates coordinate10 only. Independent total literals are385362 for G1/R1/recoveries0/attempts1/entries64;3185747 for G1/R32 with the same limits;10805811 for G2/R3/recoveries2/attempts4/entries1024;12640434 for G2/R4/recoveries64/attempts64/entries1024;16181730469 for G3/R96/recoveries64/attempts64/entries1048576. The largest valid original scalar domain G349525/R11184800/recoveries64/attempts64/entries1048576 remains finite at749486734675731443 logical units. Every intermediate is finite in this bounded component; invalid parameters still return the entire zero result. No fabricated valid late-overflow scenario or actual native feasibility is claimed.

Seven authored UNIT roots cover independent whole-vector literals, genuine original T/C allowed-map copies and exact names without cached-catalog substitution, names of both genuine earlier archived stages with separate copy/equality charges, skewed and fewer optional-group call sequences, a real DATA name-prefix shortage with sticky no-refund retry, invalid source limits and the source-valid maximum. Tests use existing safe owned fixtures; no new attack reproduction or live quota/window reset is introduced. All321 authored roots are NOT_RUN and275 individual coverage targets are UNMEASURED. Formatting is not compilation, coverage or runtime success.

No existing product path, parser, clock, budget, endpoint, Source or operating policy changes. Complete factory binding, direct caller filename work, remaining raw/model/getter/profile/ABI/native CPU work, all-eleven sufficient composition, physical/backing/peak/lifetime RAM/GC, original clock/window/lifetime, failure/recovery/final/page and production factory remain required before one complete connected bundle enters Git/native CI. Whole5-1/5/7/8 and original5-2..5-6 remain unfinished and ordered. Earlier published CI success does not verify this source.


### Direct reader, writer and getter filename component

This separate logical-work row covers successful filename construction at actual read/write/getter sites outside catalog allowed-set construction. The same helper can run separately for a catalog and a physical read; those are distinct events, not a duplicate charge for one invocation. The original pre-H0 G/R domains remain1<=G<=349525 and G<=R<=32G. No observed smaller directory, request or stage count selects the quote. All ten nonlogical work coordinates are zero.

An indexed name spends kind length+22, including its copy event's extra1. Thus highwater31/frontier30/commit28 add89 per original initial-model call. Two initialization readers, twelve healthy-prefix calls per group and two nonfinal advance readers give14G such calls. The last advance additionally reads literal HEAD and does not generate another highwater name. The legacy first-P object/inventory/custody names already priced by the legacy-control component are excluded here. Each complete prior T/C/S adds fifteen indexed names462, five stored names447 and three inventory names249, totaling1158 per earlier stage. Each prior archived member adds two regenerated-and-compared names of at most344 each. The original healthy multiplicities are14J and14E, where J=G*(G-1)/2 and E is the maximum weighted prior-member count under nonempty groups of at most32. Healthy chain-count and pointer-array events are separate from these filename events.

Current T data names cost444: two highwater names62, request/ticket names176, inventory83 and four controls123. C data costs360:62+92+83+123. Original current-C authentication costs185: two highwaters62 plus controls123. A physical writer independently generates before-highwater31, stored namesT176/C92/S179, inventory83, five indexed names154 and HEAD.pending34 twice, once for writing and once for rename; totals areT512/C428/S515. The current-phase rows are T1400=2*444+512; C2542=3*444+2*360+2*outer31+428; dispatch1793=2*(444+360)+seal185; S3759=3*(444+360)+2*(outer31+179+83+123)+515. Their sum9494G plus original initial-model names1246G is10740G. Member preparation's two original-C authentication calls add370R.

For current original spool names, two preparation-prefix readers and two post-write readers add4*maxSumSquares calls. Three S readers add6R names. Original member-intent preparation adds2R, both memberAttemptDataMatches invocations add4R, and the actual response/TLS physical writes add2R. Thus the direct name-copy count is4*maxSumSquares+14R, each at most115 logical units. maxSumSquares packs32-member groups because fewer optional groups can increase current-prefix work. A current name's maximum114 bytes follows its fixed17-byte prefix,64 digest bytes, three separators,20 stage digits, two ordinal digits and eight kind bytes. Generic uint64 stage width is conservative for the smaller original configured stage domain. Both memberAttemptDataMatches calls also compare two expected names each, at most229 units per comparison, adding916R. Their other five request-field comparisons remain separate caller work; the existing TLS request-byte match row does not pay for these names.

The total is10740G+1158*(14J)+688*(14E)+370R+115*(4*maxSumSquares+14R)+916R. Independent complete-vector logical literals are14096 for G1/R1;574452 for G1/R32;69784 for G2/R3;85532 for G2/R4;912984 for G2/R33;2696664 for G3/R96. The original maximum G349525/R11184800 remains finite at19817948201525100. Every valid intermediate is finite in this component; invalid inputs return the entire zero result. This is a numeric bound, not native feasibility or a full program certificate.

Seven authored safe UNIT roots cover independent complete-vector literals, genuine original indexed/current names and exact +1 debits, the genuine prepared member's additional name comparisons separated from request comparisons without dialing, chronological skewed/optional-group call sequences, a copied maximum DATA name followed by a one-short sticky comparison refusal, invalid original domains and the largest finite row. No live allowance, meter, clock or window is reset to make a test pass. All328 authored roots are NOT_RUN and276 individual coverage targets remain UNMEASURED; formatting and passive integer checks are not Go compilation or test execution.

This pure component changes no existing product behavior, parser, clock, endpoint, selected algorithm or operating policy. Catalog construction, recorder name copies, raw/frame/body/parser/model/getter/profile/ABI work, native string formatting/hash/allocation CPU and physical/backing/peak/lifetime RAM/GC stay separately scoped. Complete source binding and all-eleven composition, original clock/window/lifetime, failure/recovery/final/page and production factory remain unfinished before ONE complete connected bundle enters Git/native CI. Whole5-1/5/7/8 and original5-2..5-6 remain ordered and incomplete; earlier published green CI cannot verify this source.


### Direct caller request-field comparison component

This successful-path logical row funds direct caller request comparisons that are outside request reconstruction, ticket validation, whole-table equality and typed metadata. Its five-field part is the original member intent's method/params/head/body/wire comparison before send and before the retained member end. Its seven-field part is preparation's independently reconstructed selected request plus each full healthy S replay's sealed archived request, including both request/params digests. The original filename copies and two expected-name comparisons at each intent match remain solely in the preceding filename component. Those actual invocations are distinct from the TLS sample head/body match or request copies/table equalities inside the earlier record/ledger/request rows.

Each field retains its own compare event's extra1. Existing original source bounds are method32, params8192, head512, body16384 and reconstructed wire16896. The input copy validator guards input.wire at16384, but its body+head reconstruction can reach16896 before equality; using16896 on BOTH sides here is deliberately conservative. No existing wire/profile limit is enlarged by this quote. The five-field per-call bound is84037 and the seven-field bound adds two66-byte digest equalities266, totaling84303. Empty unused table slots are already part of the earlier whole-table row, not new caller events here.

The original prospective domains remain1<=G<=349525 and G<=R<=32G. Let E be the previously derived maximum weighted count of prior members under nonempty1..32-member groups. Five-field intent matches occur2R times. Seven-field direct callers occurR+14E times: once per current member preparation and fourteen full original archived views per later stage. The two nonfinal advance views include the just-settled S. Neither member SAME-C replay nor a catalog namespace walk reparses an additional inner T/C request merely because it compares original metadata. Recovery-only request comparisons remain outside this successful-path component.

The total logical bound is168074R+84303*(R+14E); all other work coordinates are zero. Independent full-vector totals are252377 at G1/R1;8076064 at G1/R32;3117615 at G2/R3;4550234 at G2/R4;46096185 at G2/R33;137531424 at G3/R96. The largest valid original domain G349525/R11184800 stays finite at2306995915634528800. Checked failures and invalid inputs return the entire zero result; no fabricated valid late-overflow case is claimed. The original factory must bind G/R rather than adopting reported smaller actuals.

Six authored safe UNIT roots cover independent whole-vector literals, actual exact per-field observer debits on all eleven ordinary closed methods, skewed/optional chronological caller sequences, an ordinary six-field DATA prefix followed by a one-short final digest comparison with sticky retry refusal, invalid original domains and the largest finite row. The preceding original-prepared-member test also checks the actual complete intent matching routine and separates its five request fields from its filename events; that existing precursor is carried rather than replaced with a synthetic live intent or repeated here. New tests construct ordinary DATA and do not dial, claim a Source, patch a live observer or reset its allowance/window. All334 authored roots are NOT_RUN and277 individual coverage targets are UNMEASURED. Formatting/passive integer checks do not prove compiler, lint, test, coverage or native success.

Other direct raw/head/readback/frame/policy/getter/model/ABI work, native CPU and physical/backing/peak/lifetime RAM/GC, complete all-eleven composition, original clock/window/lifetime, failure/recovery/final/page and production factory remain mandatory unfinished work. No parser, clock, cap, endpoint, algorithm, operating policy or existing product behavior changes. ONE complete connected bundle remains required before Git/native CI. Whole5-1/5/7/8 and original5-2..5-6 remain ordered and incomplete.


### Direct raw readback, copy, hash and policy-getter component

This grouped component covers successful explicit byte operations at original reader/writer/getter call sites. Actual filesystem I/O and names are already separate, as are object/inventory/frame/record/request/TLS/result parsers, ledger/cursor roots and typed metadata. A hash/copy/compare done by an outer reader is a different invocation from a parser operating on the same bytes. Native formatting, allocation, hashing CPU and backing-memory costs are not invented as observed events here. Positive numeric DATA does not create a Source, original owner, ACK, complete hold or factory certificate.

Inputs are original prospective G/R, selected metadata M in1..1MiB, object/pack P in1..32MiB, selected capture wire W in1..16MiB and body B in1..8MiB. They must later be bound to a valid immutable original profile; numeric validity alone does not qualify that profile. Let h=8192, i=min(M,h), L=W+65536, J=G*(G-1)/2, K=14J and A=14E for the existing maximum weighted prior-member count E. Let S be packed32-member maximum squares, F=14G-2, N=R-ceil(R/32), V=2S+3R, Q=2S+R, Z=R+N+3G+K and H=24G-1+3R+N. Optional groups may be absent: R-G would underfund nonfirst-member callers, whereas N uses the smallest possible realized group count. The original source domains remain1<=G<=349525 and G<=R<=32G.

Each copy/hash/compare event includes its own extra1. The logical rows, in their explicit implementation order, are:

| Successful direct source class | Logical bound |
|---|---|
| Full healthy boundary and chain count/array | (5h+271)F+15K |
| Three prior checkpoints and outer S operation binding | (14h+3i+277)*3K+133K |
| Current T original data readers | (11h+273)*10G |
| Current C original data readers | (10h+139)*7G |
| Current S reader suffix outside T/C | (5h+2P+270)*2G |
| Two outer C highwater checks | (3h+2)*2G |
| Original current-C authentication anchors | (7h+4)*(G+2R) |
| All three physical writers, raw file input/claim copies | (2P+45h+3i+41)*G |
| Final HEAD check in nonfinal advance | (3h+2)*(G-1) |
| Final S pack/result copies | (2P+2)G+(4B+138)R |
| Normal retained capture copies/hashes | (3L+6)R |
| Direct current spool reads outside their parsers | (2L+270)V |
| Direct archived spool reads outside their parsers | (4L+272)A |
| Outer current spool result bindings | 133Q |
| Outer archived sealed-result bindings | 133A |
| Actual policy roots getter byte copy and JSON decode | (65536^2+65536+2)Z |
| Checkpoint clones/final matches, T descriptor and initial descriptor | 3G*(12h+3i+10)+77G+h+2M+70 |
| Completed-member clone and S/advance fixed archive slots | 67R+4097*(2G-1) |
| Remaining observed wrapper hash comparisons | 133H |
| Final HEAD check in each member-equivalence role | (3h+2)*2R |

The policy row additionally charges decoded coordinate2 by65536Z. All other rows affect coordinate10 only. Native physical size is not4097: that number is the existing4096-unit explicit archive copy event plus its extra1. The same distinction applies to logical chain-array copies. Policy parsing/native cost and lifetime storage are not qualified by this row's explicit codec event alone.

The full healthy boundary keeps its selected raw-HEAD comparison, initial/cutoff model hash comparisons and final HEAD copy/readback; its chain count/reversal events occur at every actual earlier checkpoint. Earlier initial-model raw checks are already in the legacy component and are excluded. Prior checkpoint readback adds independent retained frame/inventory comparisons after the frame/inventory parsers' own reconstructed bytes. Current C's beforeRaw is already a byte slice from the earlier quoted highwater encoding, so this caller adds no extra text-to-byte copy for that field. The common writer copies its before pin once and reuses it for three readback comparisons. Final member-equivalence HEAD readback is additional to current-C authentication and fresh read transcript verification.

Current direct spool members are visited twice before send, twice after writing and by all three S views. Pending new members are parsed but are not yet inside d.retained for the outer existing-result loop; this yields V=2S+3R but Q=2S+R. The direct current result loop and direct archived sealed-result comparison are distinct from the already quoted result-to-settlement comparison inside ledger.settleOriginalsObserved. Their separate133Q/133A charges must not be removed because the same digest is compared in a nested helper. Request-field comparisons and original filename copies/equalities remain solely in their preceding components. In addition to these per-member bindings, each earlier S compares its parsed settlement digest with the original frame operation. Its separate133K is assigned to the prior-checkpoint row. The ledger predecessor comparison and ledger per-result comparison remain separate and must not be subtracted. Published S results contain one66-character digest, the complete HTTP body and the JSON result bytes; both variable strings retain body ceiling B. Their six copy events cost2*(66+len(body)+len(result))+6, bounded by4B+138 per member. A captured response hash is not the published response field.

The remaining wrapper hash count includes planning4G, committed-T6G, committed-C10G, dispatchG, current member preparationR, C/S publication2G, nonfinal advanceG-1, earlier pack pairsN and retained pack/result pairs2R. Initial-model pair and T publication use direct scalar string inequality and emit no new compare event. Initial descriptor variable strings are bounded by selected M and copied only once by the original healthy initializer. Pending write checkpoint strings are cloned and compared before the final native bracket. Those observations never renew a live allowance/window or supply an executable recovery role.

Independent decoded/logical totals are262144/17184695146 for G1/R1/M4096/P65536/W8192/B4096;4325376/283805460062 for G1/R32 with the same limits;1638400/107401574493 for G2/R3;5505024/361258358414 for G2/R33;15728640/1346944542476 for G3/R96 at M1MiB/P32MiB/W16MiB/B8MiB. The same maximum sizes at G10000/R320000 remain finite at45913866240000 decoded/4519070802464024144 logical. At valid G22000/R704000 every individual row remains finite, but total logical21864026210626460144 overflows uint64; the ENTIRE result, including finite decoded/count/row fields, must be zero with refusal. The original scalar maximum also overflows a later repeated row. Neither is an invalid-parameter test or a native-feasibility claim. All products, subtractions and accumulated coordinates retain sticky checked arithmetic.

Nine authored safe UNIT roots cover six independent complete20-row/count/vector literals; normal genuine owned retained response/TLS/HEAD/checkpoint/policy bytes with the original observer and no renewal; chronological skewed/optional callers including G3/R4 realized[4]; actual hash/equality prefixes followed by one-short final copies and copied-policy prefixes followed by one-short atomic codec refusal; invalid original numeric domains; valid late aggregate overflow; and valid later repeated-row overflow. The genuine fixture first completes one normal owned member of two. Subsequent calibration enters the existing read-only DATA phase, does not dial/sample/write or claim another original archived invocation, and preserves the original pin/hold/meter/clock/window/order. Oracle file reads are UNIT setup and do not stand for observed product I/O. All343 authored roots are NOT_RUN and278 individual coverage targets are UNMEASURED. Formatting and passive integer/literal checks are not compilation, lint, runtime or coverage results.

The successor source repairs two findings from the preserved first independent review. A genuine completed earlier S is retained and advanced before its ordinary typed original is parsed on the next original observer; the extra outer operation comparison charges133 independently of the parser and ledger, with G2/K14 requiring1862 additional units. A second genuine fixture retains an ordinary near-ceiling client-version response. The actual HTTP/RPC parser returns both body and result strings longer than66 bytes; six publication copies are calibrated independently from their separate parser prefix, and the old2B+270 per-member expression is explicitly insufficient for that ordinary input. A fresh one-short DATA observer refuses the last result-string copy while preserving all five earlier copy events. Neither calibration invokes a second publication or resets the live original meter. These tests are authored only and remain NOT_RUN.

The second review found a field-selection error in that authored prior-S test: a stored reference has only kind, digest and length. The corrected precursor selects the actual retained settlement filename from the original immutable reference, reads it as UNIT setup before the next DATA phase, and independently checks exact byte length and SHA256(domain+NUL+raw) against the original reference. Only these original file bytes enter the observed typed parser. This is a source-test correction, not an executed compiler, test or CI result.

The component changes no existing runtime path, protocol/parser, cap, endpoint, algorithm or policy. Remaining unlisted source-profile/model/ABI/native work, all-eleven sufficient composition, physical/backing/peak/lifetime RAM/GC, original clock/window/lifetime, failure/recovery/final/page and production factory still require completion before ONE connected bundle enters Git/native CI. No operating policy is activated. Whole5-1/5/7/8 and original5-2..5-6 remain ordered and incomplete.

### Grouped ABI initialization, binary framing and event cost component

This component funds successful observed fixed ABI initialization, the eight mutation selectors and binary framing, RecordChanged argument projection/framing and canonical round-trips, and the complete observed commit-claim branch. Command tuple conversion, generated/consumed JSON, mutation claim hashing/consumption and other outer RecordChanged operations are separate. In particular a binary ABI result is not a complete command, verified proof, original Source or production reservation. Native ABI/crypto CPU, allocator/backing RAM and the full source-bound factory remain unqualified.

Let A/M/I/S denote the actual target sizes of abi.Argument, abi.Method, replayClaimID and replayClaimState. These sizes are arguments to existing logical copy events, not physical RAM certificates. The two unchanged embedded ABI inputs are6790/3225 bytes; their observed copies and codecs cost10015 decoded and56514744 logical units per pair. The original semantic topology bounds window opens by O=(14G+1)*W and observed ABI pairs by O+2R. The extra member pairs occur only when a copied semantic window exists; its absence lowers actual work.

Each selector candidate copies a Method and compares four selector bytes on each side, costing M+10; eight candidates cost8M+80. A framing offset/length word costs114 logical/8 decoded. A bounded dynamic string adds at most63 padding units and two comparisons per ASCII byte. Tuple/slice argument storage and assignment cost2*n*A+n+1. The pinned revealClaim graph with up to128 seven-field keys and16 three-field services bounds the other seven graphs componentwise: framing decoded12208/logical2176A+364678. The source DID ceiling is132, derived as9+58+1+64, and operation strings have ceiling18. RecordChanged's eight nonindexed fields need16A+9, and their separate framing needs decoded32/logical1029. Its binary round-trip belongs only to the shared pool below.

The mutation branch excluding its two canonical ABI calls therefore has decoded12240/logical2192A+8M+365796. The complete commitObserved branch has decoded40/logical M+2A+2I+S+3189, including actor/topic/uint256-height validation and final claim insertion. Its fixed32-byte encoder belongs solely to this branch. Later outer-event components must exclude the entire commitObserved invocation. Take the coordinatewise maximum of these two alternative branch rows for each possible accepted Registry event; do not add both branches as if one event executed both.

Canonical calls share the SAME original per-window replayBudget byte allowance E, with1<=E<=64MiB, and log allowance L, with1<=L<=65536. The original frame walker charges every frame.Input and log.Data before event replay. The event loop rejects a second Registry event for the same frame. Thus the combined successful mutation-call payloads data[4:] and RecordChanged event payloads have sum(n)<=E per window. Each n<=148800: adopted calldata has at most148804 bytes including its selector, and a valid RecordChanged has at most512 bytes. Each mutation makes two canonical calls, so their count is at most2L, including events for other subjects.

Each canonical call costs decoded n/logical2*n*n+2*n+3. Since n*n<=148800*n, the whole per-window canonical pool is decoded E/logical297602E+6L. This uses the existing shared byte contract rather than L times the largest command for every event. It is a prospective bound, never a codec output recorded as actual or a renewed allowance. The commit branch's separate32-byte encoder is not placed in that pool. The window row is L times the alternative branch maximum plus that canonical pool; total ABI work is (O+2R) times the fixed pair plus O times the window row. All products, sums and all eleven coordinates use sticky checked arithmetic and return the ENTIRE zero struct on any overflow.

Independent target-size literals give decoded369215 and logical32880A+120M+5537404398 for G1/R1/W1/L1/E1024; decoded2298368 and logical254272A+928M+74360088584 for G2/R3/W2/L2/E4096. At G1/R32/W3/L65536/E64MiB the total is decoded39118219315/logical6464471040A+23592960M+899812900587096. These are numeric source-component envelopes, not feasible live selections. At valid scalar G100000/R100000/B100000/W100000/L1/E64MiB, the decoded total remains finite but repeated logical window work exceeds uint64; earlier finite counts/rows must not leak from that failure.

Eight authored safe UNIT roots check complete independent component/repetition literals; exact embedded SHA256 and eight closed schemas/field ceilings; every ordinary mutation's actual selector/framing/codec stages on one DATA observer; an ordinary RecordChanged and the complete real commit branch including one-short final-store refusal; exact fixed ABI pair initialization and one-short second codec preserving its first-input prefix; both actual canonical calls coupled to the original charged frame/log bytes plus independent shared-sum boundary lengths; invalid original domains; and valid late logical overflow. Tests use the existing normal offline fixtures. They add no external peer, operating key, transaction, production profile or attack producer. Actual framing precedes canonical decoding under the same observer; prior work is never reset or subtracted.

All351 authored roots are NOT_RUN; all279 individual coverage targets remain UNMEASURED. The workflow only adds the new product file to its existing exact inventory and retains nonzero/per-file90 thresholds and every runtime gate. Standalone formatting and passive source/integer checks are not compilation, lint, runtime or coverage. Remaining tuple/command/claim/event/native cost, source-profile binding, complete all-eleven composition, physical/backing/peak/lifetime RAM/GC, original clock/window/lifetime, failure/recovery/final/page and factory still require completion before ONE connected bundle enters Git/native CI. Whole5-1/5/7/8 and the original5-2 through5-6 remain incomplete and ordered; no policy is activated.

### Successful decoded-command and claim cost

The next numeric component covers tuple conversion, generated command objects/scalars, strict command JSON/schema/parameter conversion and the create claim. It excludes the preceding binary ABI component, the entire commitObserved branch, outer RecordChanged checks and state/proofs. Only successful paths are quoted; rejected-prefix/failure/recovery bounds, complete native CPU/allocator/RAM and production binding remain mandatory separate work. This constructor has no observer, retained acknowledgment, reservation or execution authority.

A closed generated create graph bounds all eight command graphs and the separate claim graph:2311 visited nodes including map keys,274 maps,1083 map entries,2 arrays/144 elements,1907 strings totaling94431 bytes and128 number strings of at most20 bytes. The tuple's65-byte input cap has base64url output87, not86. The graph's nodes+bytes total99302 bounds the first generated encoder preflight at595876. Safe expiry actually needs at most16 decimal digits, but20 remains a conservative bound. Operator scope comes from the pinned ABI/consumer's15-byte grammar.

Let P=sizeof(map[string]any), U=sizeof(map[[32]byte]bool), T=sizeof(string), V=sizeof(any), Q=sizeof(reflect.Value), F=sizeof(reflect.StructField), K/S=sizeof(replayKey/replayService), and A/B=sizeof(replayABIKey/replayABIService). These are existing logical copy-event arguments; they do not measure physical allocations. Independent source rows are:

| Successful stage | Decoded | Logical upper |
|---|---:|---|
| One native tree |2560|274P+1083T+1371V+247818|
| Tuple conversion |0|1888Q+944F+384A+48B+162402|
| Generated objects/scalars/caller |0|274P+1083T+1371V+2001596|
| Strict consumer except raw JSON and final Pack |68455|U+10V+256K+256A+32S+32B+72310923|
| Claim preconditions/six-field object |0|P+6T+6V+353|

The per-command linear envelope sums three tree rows (generation, strict consumer and possible claim) plus the other four rows. Its decoded upper is76135 and its logical polynomial is1097P+U+4338T+5500V+1888Q+944F+640A+80B+256K+32S+75218728. Alternative branches may use less; no alternative's maximum is asserted as actual. The consumer includes both material/proof binary conversions, key-material uniqueness, optional expiry, URI codecs and at most16x128 service/key comparisons. In particular its26-byte binding comparison is53 units.

For actual original calldata length D, each generated graph satisfies nodes+bytes<=D+512. A key subtree has17 nodes and38 field-name bytes, plus accepted-state8, expiry<=20 and original text/encoded binary. For a raw binary input n<=65, base64url length<=2n; the possible expansion of two such inputs fits the key tuple's384-byte head/length overhead (416 with an array offset). A service subtree has7 nodes,11 field-name bytes and its three original strings, bounded by its192-byte head/length overhead (224 with an array offset). The fixed outer remainder of every adopted method, including full Registry/DID strings, fits at most512 bytes beyond its fixed calldata. The claim uses the same original create fragments and a smaller fixed remainder. Thus both command and claim JSON lengths satisfy R<=6D+3136 and, on successful strict paths, R<=262144.

Unique accepted Registry frames share the SAME original per-window replayBudget. Their actual full calldata lengths sum to at most original evidence E, with at most original L logs. Therefore the sum of command JSON lengths and, separately, the sum of optional claim JSON lengths are each at most H=6E+3136L. Command generation and strict consumption use four actual quadratic encode/codec events at R; the optional claim adds one. Since R^2<=262144R, these contribute at most5*262144H, without multiplying every log by the largest possible JSON twice. Actual raw copy/comparisons and claim buffer/hash contribute8H. Final consumer Pack has length D only after exact equality with the original and contributes at most148804E; its final comparison contributes2E. Per-event constants add69L, including the18-byte adopted claim domain. The pooled row is decoded2H/logical1310728H+148806E+69L.

The successful actual window upper is L times the linear row plus that shared pool. Encoding preflight is distinct: strict consumer canonicalization preflights at most6*262144+64=1572928 bytes, or2474102493185 logical units. One additional, unconsumed headroom of that size is reserved per original window, on top of its ENTIRE actual upper. At any prefix this funds the largest remaining preflight without recording it as output, refunding prior work or resetting the meter. Repeated total uses O=(14G+1)W times this reserved window; original factory/window binding remains unimplemented.

For G1/R1/W1/L1/E1024, the reserved logical constant is2486493645166 per window/37297404677490 total, plus respectively one/fifteen copies of the linear type-size polynomial; total decoded is1420425. G2/R3/W2/L2/E4096 gives total decoded12410028 and logical constant145887155094942 plus116 type-polynomial copies. G1/R32/W3/L65536/E64MiB gives decoded279266918400 and logical constant36654328742400045 plus2949120 type-polynomial copies. At scalar-valid G=R=B=W=10000/L1/E64MiB, decoded1127552338887750000 remains finite while even the repeated logical constant overflows; the complete result must be zero with refusal.

Eight authored safe UNIT roots cover independent whole-window polynomials; every ordinary decoded method and its real create claim on one observer; tuple/argument generation without borrowing the nonlinear pool; a hand-counted ordinary tree including map keys and one-short last-store failure; actual JSON event deltas and its original-calldata relation; late second-JSON-decoder and later actor-codec shortages retaining the original generator prefix; invalid domains; and late logical overflow. They use existing disposable offline fixtures, add no attack producer and do not qualify an actual original replay profile. All359 authored roots remain NOT_RUN and all280 per-file coverage targets UNMEASURED. Standalone formatting and passive checks do not supply compiler/lint/test/runtime/coverage evidence. Remaining outer event/profile/model/getter/native/full-eleven/RAM/window/failure/final/factory obligations and the original5-2 through5-6 order are unchanged; no policy is activated.


### Successful outer events, expected queues and semantic wrappers

The next numeric component covers the successful collection's nonnil expected-window route. It excludes the legacy nil-history append path, the preceding complete ABI/commit and decoded-command/claim components, state/proofs and serializers. The original positive MaxClaims/MaxIdentities caps must also be bound before the future factory can use this DATA. There is still no production reservation/authority, native CPU/backing RAM, rejected-prefix/failure/final or whole-program acceptance.

Let F=14G+1, O=F*W, J=F*V, and u=min(V,4096,L). Here L/E are the same original per-window replay log/evidence limits; V is the original whole selected-version bound. The actual source calls windows O times, blocks F*Blocks times and selected commands at most J times. Initial fold/configuration work stays in the preceding cursor-initialization component. Repeated cursor snapshots already quote their own distinct copies; the actual begin-window clone below is another original invocation.

An outer changed event, excluding prior components and state work, costs decoded40/logical2745 plus its actual command raw length. This includes DID join/copy/hash, actor topic, both indexed comparisons, event DID/operation checks, command-hash equality and selected-agent comparison. A possible create consumes its original claim/global identity index at additional logical2I+2C+T+B+359, with I/C=sizeof(replayClaimID/replayClaimState), T=sizeof(string), B=sizeof(bool). Both writes are funded together before changing either index; consumed tombstones and other subjects are retained. The preceding command component proves aggregate command raw lengths at most H=6E+3136L per original window. Thus the outer-event row is O*L times the combined fixed rows plus OH logical. All possible log slots may conservatively use the changed/create envelope; this does not assert that commit and mutation execute together. Complete commitObserved stays solely in the ABI component.

For one selected command, let R/X/Y/Z be actual command/entry/record/grants bytes and q the existing queue length. Preparation costs R+T+B+10. Retention costs4R+X+(2q+2)Q+q+T+B+A+32, where Q=sizeof(providerHistoryExpectedItem), A=sizeof([]providerHistoryExpectedItem). Actual history equality plus its original charge/counter wrapper costs Q+3X+3R+N+23, with N=sizeof(int). Both post-state record/grants hashes and comparisons add Y+Z+132. The combined source row is8R+4X+Y+Z+(2q+3)Q+q+2T+2B+A+N+197. These are distinct invocations, even when they revisit the same bytes.

Retention charges R+X+Y+Z to the SAME original replayBudget; subsequent history comparison charges R+X again. Therefore aggregate weighted byte work is at most8OE. The independently enforced successful caps R/X262144, Y65536, Z262144 also bound it by3473408J. The component uses the smaller of these two proven bounds. Across original queues with at most u elements, sum(2q+3)<=J*(u+2) and sum(q)<=floor(J*(u-1)/2). Zero versions have zero subjects, moves, old slots and queued byte work; no unsigned decrement is performed for u=0. Native slice backing/peak/lifetime memory remains separate from logical copy events.

Stop comparison occurs once per window, including absence. Its two comparisons, original history-byte charge and mode copy cost21 plus2*(record+grants). Present stops number F*PresentStops, and their variable bytes are at most min(655360,2E) each. The preceding state component funds both present serializer invocations independently; this row adds no extra serializer. Absence still costs21. Finish copies the confirmed stream/int/window pointer and flag at sizeof(replayStream)+N+sizeof(*providerHistoryExpectedWindow)+2. Each completed block's separate counter/header copy costs sizeof(FinalizedHeader)+N+9; native block/trace/header-hash processing is not priced by that scalar row.

Every begin-window additionally calls the existing full shared-index/state clone with original maximum Claims/Created and schema ceilings128 keys,16 services,128 grants. It then copies sizeof(providerHistoryExpectedWindow)+sizeof(replayRun)+1. This is distinct from cursor-copy clones and the initial constructor. The clone's constant is188312+162*Claims+196*Created before this final allocation's extra1, plus its source-derived type-size terms. Native map buckets, allocator/GC and resident memory still require qualification. All component products/sums retain checked sticky arithmetic; a late logical overflow discards the complete struct, including finite decoded/count fields.

Independent literals for G1/R1/B1/W1/V0/P0/L1/E1024/Claims1/Created1 give total decoded600 and logical constant3016305 plus the recorded type polynomial. G2/R3/B5/W2/V3/P1/L2/E4096/Claims2/Created3 gives decoded4640/logical constant15282231; queue moves348/old slots43. G1/R32/B3/W3/V4096/P3/L65536/E64MiB/Claims7/Created11 gives decoded117964800/logical constant60857109975; queue moves251781120/old slots125798400. These are DATA envelopes, not feasible original profiles. A valid wide scalar case retains decoded367001862144000000 while even its repeated event logical constant28479344502374400000 exceeds uint64, requiring whole-zero refusal.

Nine authored safe UNIT roots check complete independent polynomials/counts, actual atomic two-index consumption and final-write shortage, preparation and repeated queue growth on one observer, late final queue-store refusal preserving already charged work/bytes, the complete history comparison wrapper, real begin/stop/finish including absence, ordinary commit/create/mutation expected-mode events with disjoint prior component rows, invalid domains and late logical overflow. The consumption test separately asserts the actor42/agent64 maximum and derives the unchanged ordinary actor42/agent11 fixture's exact cost, 2I+2C+T+B+200. Its last-write shortage uses that actual cost minus one, preserving both indexes and all charged prefix work; a maximum allowance minus one would still fund the shorter fixture. They use existing disposable offline fixtures, no external peer/key/transaction and no attack producer. Isolated event DATA does not claim native block/trace execution or whole collection qualification. All368 authored roots are NOT_RUN and all281 per-file coverage targets UNMEASURED. Source/profile/model/getter/native CPU, physical/backing/peak/lifetime RAM, full all-eleven composition, original window/lifetime, failure/recovery/final/page and factory remain mandatory before ONE complete connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete and ordered; no policy is activated.

### Indexed content-root costs and original reader multiplicities

The next private numeric component prices the explicit indexed-trie observations. It accepts prospective item, aggregate-byte and individual-value bounds; these DATA do not authenticate inputs, select a reservation or qualify native CPU/RAM. The root code's original16384-item/64MiB guards remain unchanged. Already canonical transactions, receipts and withdrawals still come from each original reader; no cached root replaces them.

An indexed RLP key has at most three bytes/six nibbles. Recursive depth strictly increases, so at most seven nodes occur per value. Every branch has at least two nonempty children, giving at most N-1 branches. Key encodes number at most36N: first/last keys at most14N, successful partition visits6N and failed nibble tests at most16N. Their logical cost is at most360N. Shared-path comparisons cost126N, partition comparisons66N and compact-path encodes119N.

An internal branch encodes at most16*33+1=529 body bytes plus three prefix bytes, hence532 bytes; extensions are smaller. A leaf of value length v encodes at most v+16 bytes. Leaf encoding work is at most sum(v^2)+32S+257N; internal encoding work at most6N*(532^2+1). Leaf copies add S+6N and internal copies at most6N*17*33. At most7N node hashes cost S+(16+6*532+7)N, and child-hash copies cost at most7N*33. Together these give VS+34S+1705896N, using sum(v^2)<=V*S. A separate four covers an empty root's encode/hash pair. For prospective aggregate ceiling B and value ceiling V, the actual pool is at most min(B,N*min(V,B)). This bounds pooled bytes without multiplying each node by the largest whole input. Only logical coordinate10 is populated.

Let F=14G+1. Every successful whole journey retains independent acquisition and semantic roots. JSON acquisition computes a transaction and withdrawal root; complete binary interpretation computes withdrawal/transaction/receipt roots; semantic replay independently computes transaction/receipt roots. Thus JSON uses at most5FB content roots and2FB withdrawal roots. Debug-RLP skips the two JSON roots, using4FB andFB respectively. Its successful native withdrawal path is bounded by the trie's16384-item guard, whereas the JSON parser independently restricts its withdrawal array to16. The quote explicitly selects these two existing encodings; it does not silently apply JSON's tighter guard to debug-RLP or change either reader.

For original per-window transaction ceiling T and evidence E, transaction and receipt counts each share T and their combined canonical bytes share the SAME original semantic replayBudget. An individually bound value ceiling V gives P=min(E,2TV). JSON's content row is F*W*(3VP+102P+5*1705896*T)+20FB. Debug-RLP uses F*W*(2VP+68P+4*1705896*T)+16FB. The weights preserve the three/two versus two/two independent transaction/receipt root invocations; every empty root is still funded. These bounds require future source binding of E/T/V, not a reported smaller array or an inferred operating profile.

The pinned go-ethereum v1.17.5 Withdrawal has three uint64 values and one20-byte address. Their canonical RLP is at most3*9+21+1=49 bytes. The JSON withdrawal root upper for16 values/784 bytes is27359412; the distinct debug-RLP upper for16384 values/802816 bytes is28016033796. These rows cover explicit trie observations only, excluding typed RLP encoding and native allocation. For G1/R1/B1/W1/E1024/T1/V1024, the JSON total is997477500 and debug-RLP total420375362700 logical units. Independent five whole-polynomial cases include both encodings and pooled byte limits; a valid wide numeric case refuses with the entire result zero when its content row overflows.

Seven authored safe UNIT roots cover those independent literals, inert indexed-list values and the127/128 and255/256 key-width boundaries against the pinned library, an exact44-unit ordinary single leaf and a final-hash shortage retaining its38-unit prefix, maximum-width canonical withdrawal fields in memory, encoding-specific whole-journey counts, invalid domains and late overflow. No native debug block with an excessive withdrawal list, peer, network request, transaction or attack producer is constructed. All375 authored roots remain NOT_RUN and all282 per-file coverage targets UNMEASURED. Standalone formatting/passive source arithmetic are not compiler/lint/runtime/coverage evidence. Remaining JSON/RLP/gas/frame/profile/model/getter work, native CPU/backing/peak/lifetime RAM, full-eleven composition, original window/failure/recovery/final/page and factory must still close before ONE complete connected bundle enters Git/native CI. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### Canonical binary replay costs in both original passes

The next numerical component prices only replayDecodeContent's explicit observations. Both acquisition and later semantic replay independently decode the original canonical header, transactions and receipts; a cursor copy or cached decoder replaces neither. The earlier indexed-trie component funds the two root computations. Their two32-byte root-to-header comparisons belong here, at65 units each. A provisional preparation omitted that130-unit caller row; it was not accepted as product cost and is superseded before this implementation. Original guards, strict gas requirements, observed ownership and protocol behavior are unchanged.

For one canonical raw value of length n, the original codec, encoder and equality contribute decoded n/logical2n^2+2n+3. The successful encoder preflight uses that same original n, which equals the later actual canonical encoding; it does not consume a second actual event or rely on a larger unrelated native/body preflight. Header plus transaction and receipt bytes share the SAME original per-window replayBudget E; transaction and receipt cardinalities each share T, and complete receipt logs share L. Headers independently have a64KiB cap. With a separately bound transaction/receipt value ceiling V, define M=min(E,max(65536,V)), preserving headers when V is smaller. Then sum(n^2)<=M*E. One original pass costs decoded E, transactions T and logs L per window, block count1 per block, and logical2ME+2E+6T+133B_window. The133 combines three header event constants and two65-unit root comparisons.

Using F=14G+1 original journeys, O=F*W windows and both original passes gives work2=2OE, work3=2FB, work4=2OT, work6=2OL and work10=4OME+4OE+12OT+266FB. All other coordinates are zero in this component, not in the complete program. Native RLP internals/CPU/allocator/RAM, JSON/binary acquisition construction, gas/frames and other components remain distinct. Prospective E/T/L/V and the full manifest must still be derived and bound by the complete factory; a positive quote neither installs limits nor supplies admission/operating authority.

Independent full-vector literals include G1/R1/B1/W1/E1024/T1/L1/V1024: decoded30720, blocks30, transactions30, logs30, logical62980170. G2/R3/B5/W2/E4096/T2/L10/V1024 gives475136/290/232/1160/3893304346. G1/R32/B3/W3/E64MiB/T16384/L65536/V8MiB gives6039797760/90/1474560/5898240/101331003704291010. E131072/V1 versus V65537 distinguishes the independent header ceiling from the transaction/receipt ceiling, with logical515403944010 versus515411808330. At G1/R1/B100/W100/E=V64MiB/T16384/L65536, logical27021598167171471000 overflows while every other count remains finite, requiring whole-result zero/refusal.

Seven authored safe UNIT roots cover those five independent whole-vector literals; ordinary canonical rows against original byte lengths and separately observed root computations on the same inert meter; both original core passes with distinct temporary acquisition budgets and one retained semantic window budget; a one-short final header comparison preserving entered codecs and original byte/transaction charges; an exhausted whole-window byte budget refusing the next unchanged ordinary block without renewing allowance; invalid domains; and late logical overflow. They use the existing handwritten offline blocks, no peer, RPC, transaction or attack producer. The core-pass unit does not claim the complete acquisition/semantic owner pipeline or native qualification. All382 authored roots remain NOT_RUN and all283 per-file coverage targets UNMEASURED. Standalone formatting and passive arithmetic are not compiler/lint/runtime/coverage evidence. JSON/RLP construction, gas/frame/profile/model/getter work, native CPU/backing/peak/lifetime RAM, full-eleven/window/failure/recovery/final/page and factory remain required before ONE complete connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### Common binary acquisition, immutable retention and preflight room

The next private numerical component covers common binary acquisition outside the preceding core/trie computations, the JSON-only native body builder, and the actual immutable retain/replay copy pair. JSON/RPC parsing, receipt conversion branches, gas/frame/effect callers, native CPU/allocator/backing RAM and complete window/factory acceptance remain separate. Original parsing, caps and operating policy are unchanged. Positive DATA do not authorize a Source, ACK or reservation. The original final header encoder preflights65536 against E; this constructor therefore refuses E<65536 rather than suggesting an impossible full-reader configuration is qualified.

Let F=14G+1, O=FW, B be whole blocks, E/T the same original semantic-window limits, C the original capture BodyBytes, V a separately derived canonical transaction/receipt value bound, and N the original maximum native blocks. Define b=min(E,C), h=min(b,65536), machine word w, logical body type size A and gas-evidence size Z. These sizes are existing copy-event arguments, not measured RAM. Each common reader performs three whole-body codecs and one body-payload codec, one header codec, canonical body/uncle/header encodes, the selected header hash and text comparisons, typed withdrawal encodes and output transaction encodes. Its root computations and both core passes remain in the preceding disjoint components.

For JSON only, pinned go-ethereum v1.17.5 encodes legacy transactions directly and typed transactions as their canonical bytes with at most five extra RLP-prefix bytes in a block. Its original closed JSON body has no uncles, at most16 withdrawals of at most49 bytes, and at most three enclosing prefixes of at most five bytes. Thus one body length is at most canonicalHeader+canonicalTransactions+5*transactions+800. Successful original semantic bytes/counts then bound the full-journey body sum P by min(bB,W(E+5T)+800B). Debug-RLP has no equivalent empty-uncle/16-withdrawal guarantee, so its P remains bB. Both modes have full-journey header sum J<=min(hB,EW). Squared observations use sum(bodyLength^2)<=bP and sum(headerLength^2)<=hJ, preserving these shared byte bounds instead of multiplying each block by the largest whole input.

The common row is decoded F(4P+J), logical F[6bP+4hJ+3P+3J+B(2C+9128+D(2402+3w))], where D is16 for JSON or16384 for the original successful debug withdrawal trie. Here2402=49^2+1, and9128 is the independently added fixed row: raw-copy constants, body/header codec/encoder constants, selected hash texts/comparisons8980, uncle/withdrawal root comparisons and output-slice constants. Actual transaction materialization adds O[VE+(3w+1)T]. Its one-per-block slice constant is already in9128. JSON's native body builder separately adds F[bP+B(A+2)]; debug adds no builder. JSON receipt parsing and debug receipt hex/array parsing remain separate, while the common initial copies of both original response strings are included.

The retain/replay pair costs2(S+g)+(10w+4)n+6+optional(Z+3) for one original block, with S canonical header+transaction+receipt bytes, n transactions and g gas-trace bytes. Strict semantic gas checking charges profile/hash/trace to the SAME window budget after canonical content, so sum(S+g)<=E and sum(n)<=T. Present gas blocks are bounded by original N. Thus the repeated pair is2OE+(10w+4)OT+6FB+(Z+3)FN. Pending/effect-array construction, effect scans/hashes and cursor snapshots are distinct original invocations and are not claimed by this pair. Immutable owned strings and fresh replay byte buffers remain required.

Included encoders retain their original preflight ceilings even for smaller successful outputs. An additional unconsumed logical O[(max(C,65536))^2+1] headroom funds those included preflights on top of the entire actual row; it is not recorded as actual output or a refunded allowance. Wider64MiB transaction-hash preflights inside JSON/gas/effect callers remain separate. Actual and reserved vectors are returned separately, and the full program/window/lifetime factory is still absent. Checked arithmetic refuses with the whole result zero if any intermediate overflows. At valid wide debug DATA G1/R1/B10000/W10000/E64MiB/C=V8MiB/T16384/N0, just6bPF=63331869759897600000 overflows while decoded2=5042995200000 remains finite; no partial field may survive.

Eight authored safe UNIT roots cover six independent portable whole-polynomial cases including pooled JSON bytes and separate debug limits; exact native-header codecs with a one-short last equality; unchanged canonical JSON native body bytes and original encode-preflight refusal; the actual ordinary debug decoder with RPC/receipt/core/trie contributions separately observed on the SAME inert meter; exact owned retain/replay copies for ordinary legacy and successful empty native-gas DATA; a final gas-trace copy shortage preserving the entered prefix; fifteen invalid domains; and late overflow. Empty arrays are compared by their lengths and canonical bytes, without inventing a nil-versus-empty representation requirement. No oversized native debug block, external peer, RPC, paid transaction or attack producer is created. All390 authored roots are NOT_RUN and all284 per-file coverage targets UNMEASURED. Standalone formatting and passive source arithmetic are not compiler/lint/runtime/coverage evidence. Remaining JSON/RPC/receipt, gas/frame/effect/profile/model/getter work, native CPU/backing/peak/lifetime RAM, complete all-eleven/window/failure/recovery/final/page and factory must close before ONE complete connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### JSON shape, scalar and array observation costs

The next numerical component covers successful nonnil calls of the original JSON preflight, string, hex DATA, canonical uint64/value, null/false literal, closed-field and raw-array primitives. These are alternative per-invocation rows, not the complete typed JSON reader or one combined actual. Canonical typed JSON binding, sender/signing/hash work, receipt callers, native CPU/allocator/physical RAM and full collection multiplicities remain separate. No original parser, limit, protocol, observer owner or operating policy changes. Positive numerical DATA cannot authorize parsing, collection or a Source.

Let n be the selected original raw-input upper,4<=n<=64MiB, w the machine word bytes and k=min(selected array maximum,(n-1)/2), with array maximum0..65536. Successful raw values may be shorter than n; the numeric upper is never recorded as actual. For codec q(x)=x^2+1, explicit decoded2/logical10 rows are: JSON preflight n/(n^2+2n+3), string n/(n^2+n+2), hex DATA (2n-4)/(n^2+(n-4)^2+n+1), raw array n/(n^2+(6w+1)k+2), and null/false literal0/(2n+7). Other coordinates are zero in these selected observations only. Hex JSON strings decode to at most n-2 text bytes, including the0x prefix; the hex codec has at most n-4 bytes. JSON preflight's two raw scans and one codec are distinct original observations. Native JSON parser internals remain part of their separate qualification.

Raw arrays initialize k three-word RawMessage headers before reading elements, even when whitespace leaves unused slots. Each entered element adds one more three-word header copy. The bound includes both copies and their invocation constants. Ordinary `[0,1]` has decoded5/logical12w+29; `[]` has2/6; `[ ]` has3/(3w+11) because its one spare initialized slot is real observed work. A one-short final element copy retains decoded5/logical9w+28, returns no partial array and leaves the same original observer failed.

For a fixed original field schema, A is its distinct allowed-name count, S their summed bytes and R its distinct required-name count. Allowed keys are a subset of those names. Membership comparisons are bounded by2AS+A^2. Required RawMessage values are disjoint in the original object and sum to at most n; each null check contributes at most twice its value length plus6. Thus fields cost decoded n/logical n^2+3n+2+2AS+A^2+6R. Original block/transaction/access/authorization/withdrawal/receipt/log tuples are respectively(29,298,27),(24,201,12),(2,18,2),(6,28,6),(4,32,4),(17,163,10),(10,97,9). Their constants are18287,10296,88,408,296,5891,2094. They are bound to the closed source allowlists, not an arbitrary caller-reported schema. The ordinary73-byte access object independently enters5565 logical units; its last mandatory-value comparison costs7, so a5564 allowance preserves exactly5558 units before sticky refusal.

Successful canonical quantities have at most16 hexadecimal digits. Their explicit row is decoded16/logical566, from2d^2+3d+6 at d16. The original16-digit encode preflight still needs257 unconsumed units. For ordinary0x1, actual logical work is only11, but the preflight occurs at actual prefix6 and requires263 available total units.262 must fail at prefix6;263 succeeds without recording the257 preflight as actual. The bounded256-bit value primitive similarly has decoded64/logical4164 and separate4097-unit preflight room. Each room is an additional reservation alternative for its selected primitive, not an actual event, refund or renewed allowance. Combined caller/window/lifetime room remains to be derived.

The numerical domain makes every returned component less than2^54, including the64MiB boundary; invalid domains return an entirely zero structure before arithmetic. Five independent portable whole-row cases distinguish the array word coefficient, raw-input clipping and all seven schema constants. Nine authored safe UNIT roots also cover the exact mandatory-field and array final-copy prefixes; JSON scans/strings; both empty and nonempty hex DATA; original quantity preflight boundaries; scalar/literal rows; and cumulative calls on the same original inert observer without resetting actuals. No external responder, RPC, attack producer or transaction is constructed. These private primitive fixtures do not claim complete typed-block or owner-window validation.

All399 authored roots remain NOT_RUN and all285 per-file coverage targets UNMEASURED. Standalone formatting, source arithmetic and independent source review are not compiler/lint/runtime/coverage evidence. Complete typed JSON/RPC/receipt/gas/frame/effect/profile/model/getter work, native CPU/backing/peak/lifetime RAM, all-eleven/window/lifetime/failure/recovery/final/page and factory still must close before ONE complete connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### Ethereum transaction signatures and closed JSON binding costs

The next private numerical rows cover the original Registry chain transaction signing-hash/sender observations and successful closed JSON canonical binding. These are Ethereum transaction checks, distinct from SAGE message signing and HPKE key roles. No protocol algorithm, parser, nil API, caller, operating setting or admission path changes. Native curve/CPU/allocator/physical RAM and complete JSON/RPC/receipt/gas/frame/effect/caller costs remain separate. Positive numerical DATA does not authorize a source, a transaction or collection.

For one original transaction pass, E is the original canonical transaction byte-pool upper, T its count upper and V a separately derived per-value ceiling. Original semantic E may also fund other evidence, so ignoring those positive bytes is conservative. With M=V+256 and P=E+256T, pinned consensus tuples preserve the common transaction fields and remove V/R/S. Protected legacy inserts only the fixed Sepolia chain and two zero integers. Six inserted RLP payload bytes, list framing and the type byte remain below256 extra bytes. A blob sidecar only enlarges the enclosing canonical input; it is not inserted into the signing tuple. Thus raw signing-list length y<=V+256 and sum(y)<=E+256T. The exact legacy observed row is y^2+y+2; typed is y^2+2y+5. A whole-pass signing upper is(M+2)P+5T in logical10 and zero elsewhere. Generic signature-tail RLP projection in the authored unit independently checks the supported existing offline types, rather than repeating the original per-type tuple constructor.

The original sender additionally observes validation of65 bytes, signature encoding65, recovery codec97, public-key hash64 and address copy20. These add decoded97 and logical66+4226+9410+65+21=13788 per transaction. Its whole-pass row is decoded97T/logical(M+2)P+13793T. Native elliptic-curve internals are not priced by those logical primitive units. Both original signing preflights retain64MiB+256 and therefore require4503633987174401 unconsumed logical units in addition to the included actual upper; two checks do not become two actual charges or renew an allowance. The fixed domain E1..64MiB,T1..16384,V1..E keeps these numerical rows below2^53. Complete caller repetitions still need checked composition.

The JSON binder supports only its original Header/Transaction/Withdrawal projections. Direct source maps have respectively24/at-most18/4 outer keys; a conservative K32/key-length24 envelope also covers every optional key. Original incoming block/transaction/withdrawal schemas have at most29 names/298 name bytes, with at most5 extras of length24. Pinned hexutil/common and generated access/authorization codecs yield compact ASCII hex strings, nulls and their closed lists/maps. No expected number-valued JSON leaf permits a shorter noncanonical float spelling. On a successful comparison, each present value's canonical compact spelling is no longer than its original raw spelling; whitespace, escapes and ignored extras cannot shorten the original below it. Only absent null fields and the explicit-v yParity alias may be inserted. A null key/value member has at most24+two quotes+colon+comma+four null bytes=32 bytes. yParity's7-byte key and at most20-byte quoted uint64 hex value also fit32. At most32 such insertions give canonical j<=s+1024 for original complete source upper s. This tightens only the derived actual-output bound; it does not change the original2s+65536 encode preflight.

Excluding the typed projection's separately observed header/transaction hashes, binder output encoding/map decoding contributes2j^2+2 and decoded j. Present distinct source/canonical field values have disjoint raw byte pools<=s/<=j, so their codecs cost at most s^2+j^2+2K and decoded s+j. Structural equality operates only on standard decoded JSON trees. Its node/string/map-key work is bounded by5s+4j. Absent-null/alias checks add at most2j+6K. Outer key and extras comparisons are bounded by S+A+25K+49AX, with A<=29,S<=298,K<=32,X<=5. Summing gives decoded s+2j/logical s^2+3j^2+5s+6j+8490. With j=s+1024 this is decoded3s+2048/logical4s^2+6155s+3160362. The separate unconsumed room remains(2s+65536)^2+1. All intermediates are below2^55 for original s4..64MiB. Invalid domains return the entire zero result before arithmetic. Neither quote covers arbitrary marshalers, arbitrary schemas, cyclic graphs, failed prefixes or reported actual lengths.

Targeted pinned v1.17.5 source files were copied as inert text from the previously verified public module ZIP. Its23460456 bytes/SHA25679c3193e151ed261293f32b6910ed2e3869beb328ab98cefd4c559550be617a1 and original content-identity record were rechecked; no module/project code was executed. The relevant signing encoders, transaction/header field projections, hex serializers and required nested access/authorization codecs support these arguments. They are source identity evidence, not compiler/native/runtime qualification.

Nine authored safe UNIT roots cover four independent pooled signature vectors and five JSON binder vectors; generic RLP signature-tail lengths against original supported signing hashes; repeated recorded sender recovery on the SAME observer after warming the library cache; genuine wide-preflight and97-byte recovery shortage prefixes with no account returned; exact ordinary108-byte withdrawal binding decoded226/logical27542 and unchanged4323325505-unit preflight room; all supported closed types, omitted nulls, explicit-v aliases, an ordinary JSON escape and the recorded header's declared extras; decoded-budget shortage with no partial acceptance; and whole-zero invalid domains. A map-order-dependent final field is not assigned an invented deterministic prefix. The wide original signing preflight also prevents a smaller fixed logical allowance from reaching an artificial last-copy boundary; the authored refusal uses the real recovery decoded boundary instead. No quota is changed midway, no failed observer is reset, and no key, peer, RPC or transaction is generated.

All408 authored roots remain NOT_RUN and all286 per-file coverage targets UNMEASURED. Formatting and passive source arithmetic are not compiler/lint/runtime/coverage evidence. Complete typed JSON/RPC/receipt/gas/frame/effect/profile/model/getter costs, native CPU/backing/peak/lifetime RAM, all-eleven/window/lifetime/failure/recovery/final/page and factory remain required before ONE complete connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### Original JSON and debug content branches

This component composes the original successful JSON block, transaction, nested-list, receipt and log readers under their shared semantic-window limits, and separately prices the selected debug RPC encoding. It does not replace the parsers or select an encoding. Common binary acquisition/retention, core replay, indexed trie computations and gas/frame/effect work retain their preceding disjoint rows. Native CPU, allocator, backing/peak/lifetime RAM, failed prefixes, final pages and the complete admission factory remain required. A numerical positive result supplies no execution authority.

Let F=14G+1, O=FW, B be whole blocks, E/T/L the same original window's canonical evidence/transaction/log ceilings, C the selected capture BodyBytes and V an independently derived canonical transaction/receipt value ceiling. The domain is E65536..64MiB, T1..16384, L0..65536, C4..8MiB and V1..min(E,C), with unchanged semantic collection limits. A zero actual transaction/log count remains covered by a positive selected transaction ceiling. H=min(E,C,65536). All arithmetic, including late reservation addition, is checked and any failure returns the entire zero structure.

For actual original JSON byte pools Rb/Rr over one complete journey, each per-block result is bounded by min(E,C), and each temporary acquisition charges both results to its same E. Therefore Rb,Rr<=min(E,C)B and Rb+Rr<=EB. Every coefficient on Rb is larger than its Rr coefficient. An extremal linear allocation first fills rb=min(E,C)B, then rr=min(min(E,C),E-min(E,C))B. These named extremal fields maximize the weighted sum; they are NOT separate bounds on an actual receipt pool. In particular rr=0 when E<=C does not assert that the real receipts are empty.

The original transaction location always attempts transactionHash before taking its present hash alias. That absent attempt costs one unit and is included. Its string/quantity/hash-text comparisons are bounded by decoded r+32 and logical (C+1)r+10122. A complete transaction's fields, typed decoding, from-address comparison, location, optional timestamp/price, and canonical binding contribute decoded9r+2160/logical(10C+6164)r+3187375 before nested lists, sender and separately counted native hashes. The three optional list literals, their top-level arrays and disjoint access/storage/blob/authorization rows add decoded5r/logical(5C+12w+515)r+24. Thus complete raw transaction work is bounded by decoded14r+2160/logical(15C+12w+6679)r+3187399, with sender decoded97 counted separately. No absent alias, spare array initialization or nested field is treated as free.

Header/root comparisons, selected-header hash text, required/optional header scalars, three original arrays and typed header binding contribute decoded10Rb+2368B/logical(10C+6w+7196)Rb+3204823B before transaction/withdrawal rows and native hashes. A withdrawal row costs decoded5r+2048/logical(6C+6158)r+3160661; its at-most49-byte RLP and outer array copies add2402+8w+2 per withdrawal. There are at most16 per JSON block. Raw transaction and withdrawal row pools are disjoint inside Rb, so the larger transaction byte coefficient covers both; their per-object constants remain separate. The combined block result contributes decoded24Rb+35136B+2160WT and logical(25C+18w+13875)Rb+(53813863+128w)B+(3187399+8w+2)WT before its native encoders/hashes.

Receipts include one preflight and outer array, fields/log arrays, type/location, typed receipt codecs and the output slice. Each log's fields are checked twice; topics initialize at most four slots; optional timestamp, location and removed=false checks remain distinct. Including receipt-native fixed constants but leaving their byte polynomials in the shared canonical pool, receipt work is decoded12Rr+48WT+48WL/logical(12C+12w+19)Rr+6B+(16590+6w)WT+(14895+24w)WL. The original receipt RLP output and one transaction hash are included here; common replay receipt processing is not repeated.

For canonical transaction/receipt/header pools X/Y/Z sharing E per window, block JSON performs two transaction hashes and one transaction RLP, and the receipt loop performs one additional transaction hash and one receipt RLP. Variable terms are4 sum(tx^2)+3X+sum(receipt^2). Their fixed native constants are5 per block-JSON transaction and3 per receipt; the latter are already in16590. Three block-level header hashes contribute3 sum(header^2)+3Z+6B. Each transaction also recomputes its header hash, bounded by H^2+H+2. The sender row adds (V+258)X+(256V+79841)T. The common original canonical pool is consequently bounded by max(5V+261,3H+3)E; summing unrelated independent E pools would overcount. All fixed terms together give the transaction coefficient H^2+H+256V+14w+3283839.

The returned successful JSON vector is decoded2=F[24rb+12rr+35136B+2305WT+48WL], block3=FB, transaction4=OT, log6=OL and logical10=F[(25C+18w+13875)rb+(12C+12w+19)rr+(53813875+128w)B+W(max(5V+261,3H+3)E+(H^2+H+256V+14w+3283839)T+(14895+24w)L)]. Other coordinates are zero in this selected component only. Separate unconsumed logical room O*4503633987174401 funds the unchanged signing preflight and dominates all included transaction/header, scalar and2C+65536 JSON preflights. It is neither actual output nor a renewed allowance, and is not the complete program's reserve.

The alternative debug branch includes only original block hex decoding, receipt raw-array initialization/copies, receipt hex decoding and output slice initialization. For k=min(T,(C-1)/2), its decoded2 upper is F(5C-4)B and logical10 is F[B(5C^2-6C+20+(6w+1)k)+(3w+1)WT]. No block/transaction/log counter, native common row or JSON signing room is invented for this alternative. Original result copies already belong to common binary acquisition.

Nine authored safe UNIT roots cover independent portable complete JSON/debug polynomials and raw slot clipping; unchanged recorded canonical block/receipt outputs over two cumulative calls on the SAME observer, with only disjoint body-builder/trie observations subtracted arithmetically; exact debug primitive rows; the original absent transactionHash attempt; all existing finite typed nested-list shapes; genuine early codec shortage preserving both entered scans and sticky refusal; invalid domains; and late actual/reservation overflow returning the whole zero result. No quota or actual is reset, no native value is sent to a peer, and no key, RPC or transaction is generated. The ordinary exclusion prefix is independently bounded before arithmetic subtraction. These are authored units, not runtime results.

All417 authored roots remain NOT_RUN and all287 per-file coverage targets UNMEASURED. Only approved standalone formatting and passive source/hash/literal arithmetic have run. The remaining gas/frame/effect/profile/model/getter work, native CPU/backing/peak/lifetime RAM, all-eleven/window/lifetime/failure/recovery/final/page and complete factory must close before ONE connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.


### Original gas acquisition and both strict gas checks

The next numerical component covers the original post-fork gas-profile classification or top-only normalization, the initial original-result/descriptor copies, and BOTH strict gas checks. Collection checks gas after acquiring/normalizing it; the later original persistent semantic replay calls replayDecodeBlock, which checks gas again. Neither a warm sender cache nor the preceding binary-core quote replaces that second check. The four profiles remain explicit previously selected alternatives; the bound cannot select a backend, infer consensus gas, supply missing evidence, retry or activate policy. Native CPU/allocator/backing/physical RAM, frames/effects, deferred/failed prefixes and complete factory/final admission remain separate.

Let F=14G+1, N the original maximum post-fork native blocks, n=min(W,N) the maximum original windows containing such a block, T/E the same original window limits, C selected BodyBytes, V the independent canonical transaction ceiling, H=min(E,C,65536), w machine word bytes, I the big.Int logical copy-size argument, Z the ReplayGasEvidence logical copy-size argument and S=sizeof(types.Transaction)+sizeof(types.BlobTx)+2 the two native sidecar copy events. Domain E65536..64MiB,T1..16384,C4..8MiB,V1..min(E,C),N0..B preserves the previous semantic domains. These reflected type sizes are existing copy-event arguments, not measured RAM. No-native N0 produces zero actual/reserved gas vectors, including no phantom signing reserve.

The final strict trace pool R is bounded by min(CN,En). The E coupling comes from each successful ORIGINAL persistent semantic window charging its final gas trace, not the temporary acquisition budget. For direct native profiles, the original result is that final trace, so R0=R. For root profiles, the original full top-only result is charged to each temporary per-block budget and normalized output is no larger than it; only R0<=min(C,E)N is justified. It must not borrow a single En pool. Array initialized-slot sums are bounded by P=min(NT,R) and P0=min(NT,R0); actual entered rows are separately bounded by nT. Spare slots remain priced even for sparse arrays or empty results.

One strict gas pass has two trace raw codecs and two scans, a fresh native Header hash/text binding, closed txHash/result and four-dimension field checks, scalar codecs and a fresh transaction hash/sender/gas-floor calculation. Its decoded upper is6R+161nT. Outer pair A2/S12/R2 gives fields constant64, and the longer execution-gas four-dimension schema A4/S44/R4 gives392. Their field event constants are66/394. The raw trace/row/string coefficients sum to(6C+10)R. Native header/text and raw parser fixed constants sum4497N. Array spare slots cost3wP. Transaction fixed constants before sender/floor are7227+3w, including raw element copies; sender fixed79841 and derived-floor62 give87130. This includes both ordinary codec/hash constants and the original scalar preflights, without charging preflights as actual.

Each canonical access-list tuple has at least23 RLP bytes:21 for its fixed20-byte address,1 for an empty key list and1 enclosing list byte. Its gas-floor comparison therefore costs at most floor(X/23) for canonical transaction byte pool X. Successful native value copies sum to at most X+(I+w)T, since canonical integer payload bytes are contained in X and each normalized nat can round by at most w-1 bytes before its copy-event constant. Recipient copying/comparison adds at most62T. Together with sender and transaction hash, the canonical X coefficient is2V+260. Native Header hashing has coefficient H+1 on Header pool Zheader. Since X+Zheader<=E in the successful persistent window, use max(2V+260,H+1)E+floor(E/23), rather than an unrelated full E for each accessor. Thus the strict logical upper for ONE pass is (6C+10)R+4497N+3wP+n[max(2V+260,H+1)E+floor(E/23)+(256V+I+4w+87130+S)T]. Multiply the strict decoded/logical rows by2F to include both original passes; no other coordinate is invented.

The native/debug block path can preserve a network BlobTx sidecar through its custom decode and re-encode. The JSON path's separate sidecar rejection does not constrain that native path. Every native transaction hash therefore reserves S per transaction: WithoutBlobTxSidecar copies the Transaction and BlobTx structs, and each existing copy event adds one logical unit. It does not copy blob payloads or add decoded bytes. Strict gas performs that hash twice and root normalization once more, giving additional2FnTS and, for root profiles,FnTS. No predicate or input contract is narrowed to discard this branch.

Native profile classification performs the complete raw-array parse, then only the first row's closed pair/result shapes. Those original object membership constants including codec/scan are54 and607; array constants give663N. Its decoded/logical upper is3R0 and(3C+2)R0+663N+3wP0+(3w+1)nT, multiplied by F. Empty original results use one array check without a field to infer; this larger row still covers that branch. The selected input, not an empty result, determines the profile.

Root normalization instead observes the original top-only context and derives a four-field witness. The15 allowed result names have121 total bytes, giving object constant2*15*121+15^2+2=3857. Context string/hex inputs are distinct raw fields. Their codec coefficients are bounded by(2C+1)r, including optional output/error/revert/reservoir fields. Gas/reservoir quantities, value decoding, from/to text comparisons, the optional canonical64-byte create-address derivation and transaction-input comparison add13198 per transaction plus a shared2E data comparison term. They do not authenticate a synthetic root or discard unknown fields. A second dimension extraction copies only the validated original values. Original JSON output encoding is bounded by its original source length, not a producer claim.

After combining outer pair/hash, context, scalar/dimension copies, witness slice and fresh sender, root normalization has decoded8R0+257nT and logical(9C+8)R0+5N+3wP0+n[(2V+261)E+(256V+9w+103734+S)T], multiplied by F. Here103734=99877+3857; ordinary output bytes remain separate from the full original result. Native To/Value accessors inside the existing root checker that have no logical meter event remain in the mandatory native CPU/backing/RAM qualification; no fake actual copy event is invented for them.

Initial original-trace copying, exact selected-profile text equality and gas descriptor construction add logical F[R0+(2*len(profile)+Z+3)N]. Later immutable retain/replay trace copies are already in the binary-wrapper row and are not repeated here. Additional unconsumed logical room F*n*4503633987174401 funds the original signing preflights and dominates the included header/scalar/create/value-copy/output preflights. It is neither actual nor a renewed allowance. Sticky checked arithmetic returns the entire zero structure, including shapes, if actual aggregation or this late reserve overflows. A positive result remains successful-path numerical DATA only; entered/deferred/failing-window admission needs its own proof.

Ten authored safe UNIT roots cover six independent portable whole-vector rows with separate strict/root S coefficients, all four explicit profile alternatives, native-window clipping and N0; two full original strict gas passes on the SAME meter after a legacy cache-warming precursor; unchanged recorded top-only normalization followed by both original strict passes while preserving the acquisition byte budget; direct native profile classification; the independent23-byte RLP access tuple and existing finite derived-floor inputs; real codec shortages preserving strict Header/scan work or refusing the first root codec atomically, with sticky retry; invalid domains; and late actual/reserve overflow. The empty-sidecar and otherwise identical canonical twin compare every observed coordinate, their exact S difference, equal hash and unchanged original bytes/sidecar ownership. They preserve the original64MiB Encode preflight with sufficient inert headroom; final-actual-minus-one would fail earlier and is not claimed as a late-copy oracle. They read existing finite public fixture bytes and invoke no new transport, responder, key, RPC or transaction. Every eventual runtime comparison uses the original observer pointer without resetting actuals or its quota.

All427 authored roots remain NOT_RUN and all288 per-file coverage targets UNMEASURED. Only approved standalone formatting and passive source/hash/literal arithmetic ran. Selected frame/log normalization and semantic execution, effect transitions, model/getter/native CPU/backing/peak/lifetime RAM, full eleven coordinates/window/lifetime/failure/recovery/final/page and complete factory remain required before ONE connected Git/native-CI bundle. Original5-2 through5-6 and whole5/7/8 remain incomplete; no policy is activated.

### Complete selected trace normalization, execution and effect transitions

This component combines the original complete callTracer normalization, transaction/root/caller/receipt binding and recursive frame/log walk with the content/effect transition control that selects, retains, waits for and releases those traces. It does not replace the original block/receipt decoding, both strict gas checks, immutable binary retention, cursor commitment/copy, command/state/event semantics or native qualification. Successful persistent-window DATA is its scope. Entered, deferred or failing prefixes need their own admission proof. No runtime factory, Source, policy, profile, key or transaction is created.

Inputs are the same original G/R/B/W and prospective E/T/A/L/C/V/Q/K. E65536..64MiB is shared persistent semantic evidence, T1..16384 transactions, A/L1..65536 frames/receipt logs/trace logs in their respective original counters, C4..8MiB selected BodyBytes, V1..min(E,C) the independent canonical transaction ceiling, Q0..WT the original whole effect maximum and K1..10 original content blocks per batch. Each selected receipt needs a transaction, one Registry log and at least one frame, so refine Q to min(Q,W*min(T,A,L)). Let n=min(W,Q), F=14G+1, frame maximum a=nA, trace-log maximum z=nL, child maximum d=n(A-1), and raw complete-trace pool U=min(C,E)Q. Receipt logs and trace logs have distinct counters; using the same L ceiling does not merge them.

Prospective Q is not an exact entered root count. Subtracting Q from the maximum frame count would underfund a smaller number of trees that still fill the frame capacity. Every active window has at least one root, giving d=n(A-1) independently of whether the admitted root maximum is reached. Normalization parses every original child/log, including reverted and static branches, and later persistent replay bounds the same complete tree. Ignored diagnostics and whitespace are not semantic byte charges; U cannot be replaced by a single nE pool.

The closed frame schema has17 allowed names/130 name bytes and6 required fields. Membership plus required-null constant is4745; codec/scan adds2. At depth0..32 each raw byte can occur in at most33 frame objects, while nonrecursive log raw objects share one raw pool. The frame-object row is decoded4r+176 and logical(4C+6w+5)r+12906. Its fields contribute4747, at most13 string attempts, two hex values, seven bounded quantities, one signed-state extra scan and one256-bit value contribute8155, and the optional calls/log arrays contribute4. Their raw value pools are disjoint at their respective schema levels. Empty, absent and mutually exclusive diagnostic forms consume less than the conservative row.

An encoded log has five allowed names/30 name bytes, object constant327, optional decimal/hex position and index attempts, address/data and up to four topics. Its decoded row is4r+296 and logical(4C+3)r+18763+24w. The two index attempts are bounded by the decimal20-digit constant885 each, the four topic codecs/copies add16657 including topic strings/output initialization, and the scalar constants sum327+2+1770+1+6+16657=18763. Fixed normalized storage initializes and assigns19w-byte child frame values and9w-byte log values. The complete normalizer therefore has coordinates2=138U+176a+296z,5=a,6=z and10=(138C+198w+170)U+12906a+(38w+2)d+(18765+42w)z+4Q, multiplied by F.

The enclosing execution hashes its header separately for every selected transaction; several selected transactions can share that header. H=min(E,C,65536), so that variable cost is Q(H^2+H), not a once-per-block pool. Its strict existing transaction-type/protection/chain gate is unchanged. Successful execution excludes sidecars only after that earlier hash; failed prefixes remain separate. Fresh sender recovery, root value/amount/input/from equality, original recipient or CREATE address, type/target equality and final receipt-log comparison add decoded161Q. Their per-transaction base constant is100951=8984+79841+1850+4164+65+1+4183+1863. The two native getter alternatives cost at most4183; copied native integer payloads belong to canonical transaction bytes, with an additional I+w per transaction. Returned log equality contributes at most2nE+46z.

The recursive semantic walk independently costs decoded64a+40z, frames a, logs z and logical4533a+2nE+(Slog+1755+7Srow)z+(16Srow+25)Q. Slog and Srow are the original logical types.Log/replayLog copy arguments. The accumulator's initialized slots are bounded by4m+16, old copied slots by2m, writes bym and allocation/old-copy constants by25 for m<=65536; it owns one output for the entire tree. Native storage internals are not inferred from these logical arguments. Combining the enclosing and recursive rows gives execution coordinates2=161Q+64a+40z,5=a,6=z and10=(2V+266)nE+4533a+[H^2+H+256V+I+w+100976+16Srow]Q+(Slog+1801+7Srow)z, multiplied by F. No proof or native cost coordinate is fabricated.

Original effect selection scans every canonical receipt log twice: logical40WL+2WT. It hashes each selected transaction and retains its canonical order. There is no type guard at that selection point, so include the checked sidecar-copy S from the gas component per selected hash. Effects reset at every original content batch; with at most K blocks, the cumulative descriptor initialization is at most KQ, rather than a global BQ. Initial/final pending arrays and their original spine lookup cost(2Sblock+137)B, trace-result header initialization/assignment/clear cost3Sstring per selected result, execution slice initialization costs Sexecution, raw trace conversion contributes U+Q, and the remaining fixed per-selected constants sum4363 including hash text.

Each fully processed block scans its pending effect descriptors once, totaling at most KQ visits. Every additional blocked visit belongs to a nonempty original32-member trace group. With b=ceil(min(T,Q)/32), at most bQ further descriptor visits are needed; each costs9 logical units. These successful transition calls do not add an unrestricted polling loop. The resulting collection row is logical F[40WL+2WT+(2Sblock+137)B+U+(V+1)nE+(KSeffect+3Sstring+Sexecution+S+4363+9(K+b))Q]. Its other coordinates are zero. Zero selected effects still require receipt scans and pending content storage, but produce no normalization/execution or signing reserve.

Components retain separate vectors. Normalization and semantic execution each count their own frame/log entries; the aggregate has twice their successful entry maxima. Additional unconsumed signing room is F*n*4503633987174401. It is not consumed actual, a refund or a new allowance. The complete-program composition must still prove sufficient and feasible headroom rather than assuming these conservative component rows fit a selected real policy. Every sum/product and late reserve preserves sticky whole-zero refusal, including all metadata. A finite arithmetic tuple is not a native block/trace certificate.

Twelve authored safe UNIT roots cover six independent portable whole-vector rows/type-size coefficients, original capacity clipping and the prospective child-bound case, repeated complete normalization on the SAME observer, supported optional diagnostics/opcodes/log scalar forms, exact first-codec refusal/sticky retry, original transaction/root/receipt and CREATE joins, exact canonical2+38 effect descriptor/hash/copy work, the original32+8 result boundary and blocked scan, no-selection control,21 invalid domains and distinct actual/reserve overflows. One-journey comparisons explicitly divide out F so its repetition slack cannot hide a primitive undercount. Tests use existing finite owned fixtures/direct functions, with no network, producer callback, operational key, VM or transaction broadcast.

All439 authored roots remain NOT_RUN and all289 individual coverage targets UNMEASURED. Only approved standalone formatting and own passive FS/hash/text/AST/literal/integer analysis ran. Native JSON/RLP/curve/allocator/backing/peak/lifetime RAM and elapsed-window qualification, full eleven-coordinate composition, entered/deferred/failure/recovery/final/page and complete factory remain required before ONE connected Git/native-CI bundle. Existing5-2 through5-6 and whole5/7/8 remain incomplete.

## Original transaction input ceilings

The original group meter now retains a separate private route seal. Its owner predicate checks that seal before every role branch; removing the ordinary group tag cannot select a wide legacy route. A collection size lookup requires the same active original group, store, program, manifest, config and current work/clock window. Invalid or copied state refuses. Genuinely eligible noncollection, first-operation and read-only recovery roles retain their previous64MiB transaction and64MiB+256 signing ceilings. A failed original producer does not make its independent read-only recovery role a healthy execution owner.

A limit alone does not prove a particular native object's size. The original JSON transaction reader checks the actual row length against the sealed capture bound C before typed decode, sender recovery or hash. The binary content reader checks each actual transaction byte string against C in its existing initial budget loop, before native header/transaction decode. Immutable retained strings cross reader boundaries; every reconstruction re-enters that binary check. Complete canonical byte equality, transaction/receipt roots, both strict gas checks, original caller/permission/type/schema checks and final output ceilings remain mandatory.

Before final canonical binding, the pinned five typed JSON transaction shapes have at most14 consensus fields. Fixed fields take at most33 RLP bytes each. At most three dynamic raw spans are disjoint within the original JSON row of length s; their enclosing prefixes add at most27 bytes. AccessTuple's mandatory key syntax pays both nested list/tuple prefixes, and authorization key syntax pays its tuple prefix. Outer list/type framing adds at most10. Thus canonical consensus bytes are at most s+499, independently of later signature, location or canonical refusal. The collection uses P=C+1024 with explicit spare, signing P+256, and checked ceil(P/word)*word+sizeof(big.Int) for the copied native value. No tx.Size, warm hash/sender cache, producer-reported length or caller-supplied narrower bound is accepted.

The added pointer is included automatically in existing reflect-sized page, first-operation and continuing memory-plan inputs. This does not qualify allocator/backing RAM or native CPU. Existing standalone conservative gas/effect component totals remain unchanged. Whole-program headroom must still cover every reachable primitive and composite preflight in its original group, advance, first-read, recovery and final scope. No global-only maximum, ledger refund, counter reset, window renewal or partial headroom compiler is supplied here.

Nine additional authored UNIT roots cover unchanged wide routes, genuine original capture selection and nonaligned word rounding, five-type before-binding JSON sizes, repeated joined raw/JSON output and all-coordinate comparisons, early raw/JSON/retained byte refusal, exact and one-short original primitive envelopes, original lineage/route/window faults and the independent terminal recovery role. Finite UNIT windows and scalar fault injection are not a complete original S, actual Registry history or operating certificate. All448 authored roots remain NOT_RUN and all290 per-file coverage targets UNMEASURED; only the allowed formatter and passive source/arithmetic checks ran. Full sufficient and feasible cost, native/RAM/time, failure/final/factory and the existing complete-bundle publication gate remain open.

The first independent SOURCE review required two test corrections. Pure capture/word/struct arithmetic is now separate from original-owner selection, retaining all validation and zero-on-error guards. Meaningful invalid capture, unsupported word and overflowing struct DATA exercise its refusal paths without manufacturing a live allowance. Native consumers still use only the authenticated original selector. This addresses an authored coverage blind spot; it is not a measured coverage result.

Only the new decoder UNIT opts into finite block/transaction/log count rows before profile, anchor and H0. The same checked derivation includes these rows in both the selected stage and original whole future; other shared fixtures retain their previous zero rows. The normal canonical precursor checks its exact one-block/one-transaction/one-log actual deltas before testing JSON equality, oversized input refusal or primitive shortages. No live meter, global budget, clock or failed state is reset. These corrections remain authored and NOT_RUN until native CI verifies the complete connected bundle.

## Entered component rows and standalone reservations

The common binary, JSON, gas and selected-effect component calculations now expose separate private actual-only DATA rows. Their original standalone Upper functions retain the same conservative preflight formulas, checked total sums, complete outputs and whole-zero refusals. The existing independent portable polynomial tests continue to apply to those standalone results. No native consumer, observer, group ledger, scope check, quota, clock or runtime factory selects an actual-only row.

Each actual-only result deliberately leaves both preflight and total zero. It preserves its successful entered-work vector, component breakdown and shape metadata, while refusing the whole result on invalid input or entered-work overflow. A later enclosing derivation must fund every primitive and composite preflight in its original stage, first-read, advance, recovery and final scope. A finite actual-only row is not a sufficient admission bound and cannot replace that work.

Four additional authored safe UNIT roots check all encoding/profile alternatives, unchanged standalone vectors and row shapes, finite entered-work DATA even when the old repeated standalone reserve overflows, whole-zero invalid/actual-overflow refusal and dormant native/effect paths. The overflow tuples are numeric composition subjects, not claims that a selected real history fits the final policy. All452 authored roots remain NOT_RUN and all290 individual coverage targets UNMEASURED. D2, source-derived sufficient and feasible full-eleven-coordinate admission, native CPU/backing/peak/lifetime RAM, entered/deferred/failing/final/page and complete factory remain open before the same ONE connected publication.

## Closed spine-value encoding

Each original spine item now uses a private typed digest operation with a512-byte canonical encoding ceiling. The complete guarded ASCII shape is at most402 bytes: outer keys/punctuation52, previous68, ordinal9, header223, format43 and kind7. Header number remains a full uint64 decimal string; timestamp keeps its original53-bit bound. Before encoding, the typed operation checks the ordinal range, empty previous at ordinal zero, a lowerHash previous at later ordinals and the existing validHeader predicate.

The original cursor still interprets every spine item and retains its selected-height, parent, timestamp and exact deployment checks. It builds the same header/map, records CopyEvent(5) with logical cost6, then encodes and hashes the same bytes and domain. No root cache, caller-reported size, skipped item or transferable size credential is added. Other generic values retain their8192-byte ceiling. Only the proved spine term in the cursor quote changes, reducing its previous conservative logical row by66854400 per item; original parent comparisons, semantic/pending/manifest rows and reflected copy costs remain separate and unchanged.

Four additional authored safe UNIT roots compare the old generic oracle and typed path on the same cumulative observer/all11 coordinates, cover the independent402-byte extreme and invalid typed DATA, check exact262151 versus one-short262150 preflight with original copy-prefix6 and sticky failure, keep generic non-spine limits, and exercise genuine collection-reader and terminal inspection roles. The collection UNIT is not a complete original S or actual Registry run. All456 authored roots remain NOT_RUN and all290 per-file targets UNMEASURED. The local quote improvement does not establish whole-history feasibility, complete D2/native/RAM/failure/final/factory acceptance or policy activation; the original complete-bundle publication gate remains open.


### Original stage windows and shared initialization composition

The original advancing role remains inside the preceding T window, after its S. It reconstructs two prefixes at depth j+1 for a stage that began with j settled groups. A new direct stage call row therefore separates its twelve healthy prefixes at depth j from those two post-S prefixes. Before advancement its ledger/cursor/copy counts are 102+10m+84j, 142+12m+108j and 37+2m+12j. Advancement adds 20+14j, 26+18j and 4+2j respectively, including both retained-model validations. Current boundary counts are 3+12j and the optional advancement adds 2(j+1). The two program-wide initial models remain disjoint from every stage.

The existing admission contract has one uniform pre-H0 maximum. Its cursor subrow now compares the direct final-stage row at depth G-1 with the direct nonfinal-stage row at depth G-2, both at the original maximum32 members, and takes each component maximum. No difference or average of global maxima grants a stage allowance. Standalone commitments remain roots-copies because the full-copy row already commits once. This is a cursor 2/10 component, with the existing preflight-inclusive per-call ceiling; it is not entered actual or a complete all-coordinate certificate.

The constructor already reserved the legacy initialization plus two original first-reader sublimits. A shared profile-aware checked calculation now supplies that same envelope to both the constructor and irrevocable admission accounting. The constructor does not charge either reader again; the later accounting no longer reserves only the narrower legacy envelope. Legacy profiles retain their exact old result, and either an invalid sealed reader or any intermediate overflow returns the entire zero row. Collection UNIT future inputs include the same two readers before profile/anchor/H0, while exact request/wire accounting is unchanged.

Six authored safe UNIT roots cover independent stage literals, original chronological window cuts and the unchanged complete grammar, every smaller optional stage, copy-root accounting/overflow, both original reader sums and genuine-owner exact/one-short admission accounting with unchanged cumulative meter/clock/hold. The old chronological oracle keeps every original event in its original order; the added cuts expose ownership rather than replace the grammar with quote equations. All462 authored roots remain NOT_RUN and all290 individual coverage targets UNMEASURED. Only the approved standalone formatter and passive source/integer checks ran. Complete D2/all-eleven sufficient and feasible composition, all actual/nativeCPU/backing/peak/lifetime RAM, first-read subrow derivation, failure/recovery/final/page and factory remain OPEN before the same ONE complete connected publication. Whole5-1/5/7/8 and the original5-2..5-6 order remain incomplete; no policy is activated.


### Original-window fixture correction

Independent R1 SOURCE review found the new genuine accounting test was blocked before H0: its shared collection UNIT selected pre-I32MiB plus four8MiB stages and positive startup under an unchanged64MiB store. R1 remains REVISE history. The collection-only UNIT now fixes physical storage to6MiB per stage BEFORE profile/anchor/H0, retaining4MiB raw and256 files at4096-byte allocation overhead (5MiB combined). The selected-header startup storage is948444 bytes, so the complete prospective physical sum is33554432+948444+25165824=59668700 bytes, below67108864. The helper explicitly checks both per-stage raw/file allocation and the whole pre-I/startup/stage sum. Legacy stage inputs, source/host/store/operating caps, exact requests/wire, work/clock rows, actuals and original live holds are unchanged.

A seventh safe UNIT root requires the original owner to construct successfully and checks that full physical sum and unchanged prepared owner/meter/hold. The existing genuine18-member exact/one-short complete-initialization regression is retained byte-for-byte. All463 authored roots remain NOT_RUN and all290 per-file targets UNMEASURED; static arithmetic and SOURCE review are not executed tests or runtime qualification. FullD2/all11 sufficientANDfeasible/nativeCPU/backingRAM/window/failure/recovery/final/page/factory and ONEcompleteconnectedpublication remain OPEN.


### Original-window first-P fixture correction

Independent R2 SOURCE review confirmed the physical repair and found the next direct UNIT precursor still refused: selecting a1MiB object maximum made the original first-P plan reserve at least1123MiB decoded against unchanged256MiB operation/segment limits. R2 remains REVISE history. A separate explicit collection-first-P UNIT selection now fixes maximumObjectBytes128KiB BEFORE startup plan/profile/anchor/H0. Only the collection-admission and native-input collection helpers opt into it, including the missing-admission negative subject's genuine first-P setup. Other fixture selections and every product/policy/operation/segment guard remain unchanged.

The actual original request wire is262 bytes (177 head +85 body). Selected header wire32768 and normalization n131072 give complete decoded work1123*131072+(32768-262)=147226362, below268435456. Header transport61716 and exchange92104 also fit the selected128KiB object limit. Startup physical remains948444 and the complete59668700-byte hold remains inside64MiB. The shared pre-H0 helper checks the complete decoded scalar; the genuine regression additionally requires the actual completed original first-P owner and checks ALL eleven reservation coordinates against BOTH unchanged operation/segment limits and the pre-H0 startup row. The exact/one-short original accounting regression remains unchanged. Earlier267-byte wording was an intermediate evidence error; the original product bytes and storage literals were262-based throughout.

All463 authored roots remain NOT_RUN /290 individual90% targets UNMEASURED. This is authored UNIT preparation and static source correction, not executed first-P, all-eleven-program/native/RAM or operating qualification. FullD2/first-reader derivation/sufficientANDfeasible cost/nativeCPU/backingRAM/windows/failure/recovery/final/page/factory and ONEcompleteconnectedpublication remain OPEN.

### Original first-reader scoped source composition

The private first-reader calculation now composes the exact firstPReading interval: legacy load, twelve object lookups, three unique-kind scans, C/S custody control, selected interpretation and cursor initialization. Cursor initialization already includes semantic/configuration construction. Surrounding main-control reads, ledger/model roots and transcript sealing are outside this row. The row remains numeric decoded/logical DATA; it neither constructs a production factory nor supplies native CPU, allocation or whole-program authority.

The original raw guard checks cumulative firstRaw plus each prospective read maximum before that read. Four inventories, at most N=min(firstObjects,4*entries) objects and five C/S files therefore have read-event logical bound firstRaw+N+9. Two identity calls add sixteen read(0) events. Recorded dispatch replay also hashes each original and owns its name and descriptor INSIDE this interval. That distinct recording costs at most firstRaw+642*(N+9), from rawSHA(r+1), two name copies2*(n+1), descriptor129 and n<=255. The earlier unimplemented preparation omitted this optional recording path; that preparation is preserved as history and is not an accepted sufficient bound. Full-program composition must partition these constituents rather than add them again as disjoint costs.

Four authored safe UNIT roots compare independent literal rows, select the derived sublimit before manifest/profile/anchor/H0 and follow genuine original P, independent initial models, T/C and both recorded dispatch roles; exercise known-insufficient decoded/logical inputs without replacing the failed observer; and reject invalid or overflowing children with a complete zero result. The selected UNIT row has decoded4751393 and logical6101990678681 plus the existing reflected logical-copy sizes. Reflection supplies an existing event argument, not measured RAM. Legacy fixture inputs retain their defaults.

All467 authored roots remain NOT_RUN /290 individual90% targets UNMEASURED. These source changes and static arithmetic are not compiled or executed owner/coverage/native evidence. The separate inherited first-finalized fixture selection issue remains tracked before whole-suite CI. Complete D2/all-eleven sufficient AND feasible cost, native CPU/backing/peak/lifetime RAM, original windows/failure/recovery/final/page/factory and the ONE complete connected publication remain OPEN; the original ordered5-2 through5-6 and whole5/7/8 remain incomplete.

### First-operation class bound with larger future objects

The inherited first-P precursor cannot be repaired by globally selecting smaller objects: the original32-member result-pack guard already exceeds128KiB from its descriptor and TLS rows before any response. Its1MiB global object envelope and every member/capture/TLS/pack guard remain unchanged. The four original first-finalized, member-dispatch, multistage and recovery-before-admission helpers retain those original inputs.

The original first-operation E/R graph is closed to the fixed finalized-header request, its selected header capture, original typed C/X/S delta and independently read references. History adds scalar owner/handoff checks to that graph, not a later arbitrary group object. The normalization branch already used for guardian now also applies to history: n=max(metadata,64KiB,transport,exchange,selectedWire). The original larger serializer/global-object headroom, global header/body replay allowance,552/571 codec and3106/3284 linear coefficients, capture row, clock counts, storage and memory calculations stay intact. Remaining-phase guardian-only reconstruction and memory scratch stay guardian-only. Standalone normalization stays unchanged. This component does not qualify native CPU/RAM or the whole history program.

For the selected fixed header, transport61716/exchange92104/wire32768 give n92104 and actual decoded103465298. At global object1MiB the complete original work vector is [1,32768,103465298,0,0,0,0,0,1,817372,11727834507083]; at global object128KiB only logical differs,11727832672075. The synthetic pre-H0 startup is still computed before attaching history, so its larger original decoded ceiling remains1177583354 or147226362 respectively. The original startup claim accepts actual<=ceiling and never refunds storage/work or resets time. Both operation and segment guards still compare the complete actual plan before that claim. Collection startup physical948444 and whole retained59668700 remain unchanged; its regression now distinguishes the exact actual vector from the original startup ceiling. Earlier147226362 actual/equality reports remain immutable historical evidence.

Three authored safe UNIT roots require genuine1MiB/32-member T/C/dispatch with the unchanged complete pack guard and each of the four helper routes, an exact versus one-byte-short decoded policy selected before profile/anchor/H0, unchanged generic quote and whole-zero invalid/overflow results. The exact/short subjects use equal operation/segment values to preserve the policy hierarchy; no live limit or observer is patched. Existing physical-startup shortage, original capture/failure closure and unrelated1MiB continuing-pin regressions remain. New tests do not perform hostile network activity or extra external RPC/transactions.

The source argument has independent bounded acceptance; the implementation still requires its separate SOURCE review and eventual connected CI. All470 authored roots remain NOT_RUN /290 individual90% targets UNMEASURED. FullD2/all-eleven sufficient AND feasible cost, original windows/failure/recovery/final/page/factory, native CPU/backing/peak/lifetime RAM and the ONE complete connected publication remain OPEN. This closes no operating selection, full fixture suite, whole5/7/8 or later ordered5-2..5-6 requirement.

### Original first-reader initialization, window and future composition

The decoded/logical firstPReading row now has a private enclosing composition using the already derived original window topology. Two independent readers belong to program initialization; each later original T window reserves its uniform reader count, including advancement after its own S. One group uses twelve readers per window. Two or more prospective groups use fourteen. The complete component reservation is two plus G times that same uniform count; the original ledger does not refund the final window's unused capacity. This differs intentionally from the exact successful chronological total of fourteen G.

The returned first-read tuple must be selected in the original sealed input before profile, anchor and H0. A numeric composition cannot replace a different live first-reader sublimit. The genuine-owner UNIT precursor now selects that exact derived tuple and requires the existing complete initialization row to equal its legacy part plus this component's initialization part. The existing real first P, independent initialization, T/C and both recorded dispatch readers retain their original observer, hold, clock and immutable input. No runtime constructor or factory selects the new composition.

The first-reader row already includes original loading, lookup, custody, interpretation and semantic/cursor initialization. None of those constituents may be added again as separate costs. Its initialization part is ALREADY inside the existing profile-aware initialization envelope; adding the full component whole row on top of that envelope would count those two readers twice. Surrounding main/control/ledger/cursor commitments, catalogs, native CPU/backing/peak/lifetime RAM, failing/recovery/final/page and every other work coordinate remain separate obligations.

Three additional safe UNIT roots compare independent finite window counts and all eleven coordinates, use the preserved chronological caller grammar with window cuts after advancement, retain complete irrevocable rows for optional shorter histories, and refuse the complete result on invalid inputs or a late whole-future overflow. The hypothetical full-size 9,226-group/162,391-request input remains in the numeric check; it is not replaced by a small demo. Its reader component fits uint64, which neither proves that all remaining components fit nor certifies actual execution, elapsed time, memory or operating limits. All473 authored roots remain NOT_RUN /290 individual90% targets UNMEASURED. FullD2 and the ONE complete connected publication gate remain OPEN; the original5-2 through5-6 and whole5/7/8 order is unchanged.

### Applied-state entered work and complete stream refusal closure

The generic state serializer, stream-root and semantic-fold Upper functions retain their original standalone results and runtime guards. New private numeric parts distinguish successful entered work from a complete one-call envelope on the original parsed, cloned, mutated and successfully applied stream path. They do not authenticate that path, replace an original observer, or select a production factory. The eventual complete compiler must establish the actual route and cardinalities before using these parts; the private record/grant types alone do not establish it.

The applied record uses the clone field ceilings and its original successful65,536-byte output limit. The generated maximum still uses six times visited nodes plus source bytes, plus64. ASCII includes control bytes, so six-byte escaping stays necessary. At128 keys and16 services the typed source bound is91,615 bytes/2,301 nodes and the unchanged encoder can preflight563,560 bytes. Successful entered encoding is bounded by65,536 bytes; a late output refusal can consume more and belongs to the separate complete envelope. Expiry formatting's encode20 preflight is not entered actual: actual format text is at most16 decimal digits, and the existing final larger envelope funds every numeric preflight.

Applied grants come only from the validated address parser and the five fixed scopes, with at most15 unescaped ASCII bytes in a scope. A nonempty array of g items has at most84g+1 canonical bytes, or10,753 at128 grants; the empty array has2. This is a property of that complete original mutation path, not arbitrary clone-bounded ASCII DATA. The wider clone16-byte scope and six-byte source expansion remain in the envelope. Generic serializer-only records with larger text keep using their original universal quote.

The semantic index now exposes its already derived schema-bounded entered-work part separately from its unchanged standalone encoder headroom. The original standalone wrapper still adds that headroom and every existing result is preserved. The exact semantic-state metadata schema has a697-byte ASCII upper, while its original encode8192 guard stays intact. These numeric refinements change no serialized message, signature coverage, state transition, policy, quota, window, clock or execution role.

A stream's complete envelope includes BOTH serializers, both version/admitted decimal calls and BOTH index roots. A serializer's output-cap refusal occurs after encoding and does not itself poison its meter: the caller still executes these deferred siblings before testing their combined success. A first-failed-primitive-only allowance would therefore be insufficient. The envelope also conservatively includes the later successful hash/metadata tail. Outer semantic-window/fold/cursor callers can have further deferred work of their own; this one-call row does not certify those outer closures or a whole stage.

Six additional authored safe UNIT roots check independent complete value/vector literals and copy-argument portability, the actual observed mutation path and original bytes, unchanged standalone preflights versus entered-only DATA, the697-byte metadata extreme, unchanged independent confirmed/working stream roots and original fold observer, and bounded in-memory deferred output refusal with decoded-quota failure and sticky prefix. No attack/reproduction, external endpoint or network operation is added. All479 authored roots remain NOT_RUN /290 individual90% targets UNMEASURED. Complete D2/all-eleven sufficient AND feasible composition, pointwise primitive/composite and outer deferred-failure envelopes, native CPU/backing/peak/lifetime RAM, original windows/recovery/final/page/factory and the ONE complete connected publication gate remain OPEN. Whole5-1/5/7/8 and the original5-2 through5-6 order remain incomplete.


### Private original fold cost composition (source proposal)

The applied serializer/index/one-stream increment completed independent SOURCE-only review with no required corrections. This does not certify its479 authored test roots,290 individual coverage targets, runtime selection or the whole connected plan. The next numeric increment preserves every prior generic upper and every runtime guard byte for byte.

One private original fold commitment now has separate successful entered-work and complete caller-envelope parts. The selected window's original History.MaxBytes is an explicit prospective input. On the actual owned route, retain charges record+grants+entry+command before publishing immutable queue fields; failed publication can increase the charge without adding an item, and snapshots preserve both bytes and counters. Therefore all queued entry+command bytes together are bounded by the same selected history budget. The bound also retains the unchanged262144-byte per-field guard and4096-item guard. Pending command bytes remain separate. At4096 queued items and8MiB selected history, the queue byte-root row is16916480 logical units instead of repeating both maximum fields per item. No runtime quota, input interval or source validation is relaxed.

The fixed queue/item/window/fold metadata byte uppers are91/370/664/879. Canonical scalar widths are conservative; the19/20-digit synthetic metadata extremes do not authenticate exact large integer identity, selected counters or headers. Successful parts price these generated shapes; caller envelopes retain the original encode8192 guards. A failed working stream can still be followed by base/pending decimals, the initial queue, every valid queue item's decimal/entry/command and chained root, and pending bytes before its combined condition returns. The complete one-fold envelope includes that deferred work. It is not reusable headroom for a containing cursor, stage or program.

Four authored safe unit roots compare independent all-coordinate literals and invalid whole-zero inputs, actual original fixed-schema encoding/event totals, real applied/observed fold and snapshot roots with their selected immutable queue aggregate, and a finite unqualified in-memory late-size-refusal closure plus sticky quota refusal. The latter is DATA only, not a parsed command, registration, credential or attack path. All483 authored roots are NOT_RUN and290 individual90% targets are UNMEASURED. Only standalone formatting and passive text/hash/integer inspection have occurred.

Full original stage/future multiplicities, pointwise primitive/composite headroom, all eleven sufficient AND feasible coordinates, native CPU/backing/peak/lifetime RAM, windows, failing/recovery/final/page and factory binding remain open. A single complete connected bundle is still required before Git/nativeCI publication; old published green CI does not validate this source proposal. Whole5-1/5/7/8 remain incomplete and operating policy remains inactive.


### Private exact planned-request and ledger control parts (source proposal)

The previous ONE-fold composition completed independent SOURCE-only review and root integrity closure with no required corrections;483 test roots remain NOT_RUN. The new increment appends only private numeric request/record/ledger control parts and leaves every old standalone result and runtime source byte unchanged. It is not a cursor cost, admitted program derivation, native CPU/RAM qualification or source/role certificate.

The original collection planner has eleven closed methods and fixed parameter templates. Generated hash/address/calldata text does not escape; the longest history calldata is164 bytes encoded as330 hex characters, with agent bytes at most64. This gives parameters<=500, body<=578 with a deliberately32-byte method, head<=178 and wire<=756. Fixed trace options with at most three config booleans fit32 visited nodes including decoded map keys, and duration output is over-reserved at32 bytes, including the unescaped UTF-8 U+00B5 microsecond suffix. The generic8KiB params/16KiB body and wire/512-byte head guards remain unchanged. Arbitrary raw JSON normalization occurs before the constructor's guard and keeps the original wider control envelope. Both independent original readers must prove exact next-table/request provenance before a future complete compiler can use the smaller successful entered-work parts. A private type or a reported input size cannot establish this.

Table objects contain request and params digests, not raw request bytes. Conservative generated upper lengths are185+420*members for the table,1647 for the ticket,505 for consumption and2270 for ledger metadata with collection digest. Normal ledger control includes ticket validation directly and again within consumption; full cursor work is separately counted. Each original request, table and record encoding preflight remains funded by the unchanged old one-call control envelope. It is not multiplied as entered actual, refunded or promoted to an enclosing scope's headroom. At32 members the new normal control row is454576758 logical units; the original standalone38959373275 remains exact. Neither is native work or whole-program feasibility.

Four authored safe UNIT roots compare independent complete vectors/invalid zero results and exact original envelopes; all fixed template/calldata/tracer forms through original constructors, copies and classification; real original recursive T/C/ledger validation in prepared/admitted/consumed/settled DATA phases; and original wide preflight refusal/sticky retry plus finite generic DATA outside the planned grammar. The latter performs no endpoint/transport/network activity and demonstrates that general request handling is not narrowed. All487 authored roots remain NOT_RUN and290 individual90% targets remain UNMEASURED; only standalone formatting and own passive text/hash/integer checks occurred.

Complete enclosing pointwise primitive/composite failure scopes, repeated stage/future work, all eleven sufficient AND feasible coordinates, native CPU/backing/peak/lifetime RAM/windows/recovery/final/page/factory remain open before ONE connected Git/nativeCI publication. No policy, selected quota, record/header contract, keys, deployment, paid transaction, monitor or operating activation changes. Whole5-1/5/7/8 remain incomplete.


The planned-ledger SOURCE R1 numerical review and root integrity closure found no required correction. Its nonblocking T1 is addressed by the precise no-JSON-escape UTF-8/byte explanation and1µs/1.5µs variants in the same original template UNIT alongside the long duration. Both call-tracer config forms and state-gas template pass through the original constructors/copy/classification in those authored scenarios. This changes no coefficient, standalone envelope, runtime guard, normal selection or test-root count. All487 roots remain NOT_RUN and290 individual90% targets UNMEASURED; the clarification does not claim CI1.26.10 execution or operating qualification.

### Conditional original cursor and owned-copy parts (source proposal)

The original ancestry route constructs an empty semantic state and does not begin semantic replay until the complete parent spine reaches the content boundary. An empty-stream component can omit the record/grant serializers that this original branch never enters. It still includes both independent lifetime indexes, both decimals and the original metadata preflights. Other-subject commands can populate those indexes even when the selected subject remains empty. The condition therefore means an actually established version-zero empty record/grant graph, not an arbitrary private object, a reported version, or permission to replace the required registered-Agent case. Full nonempty/general support and every old standalone result remain unchanged.

A private composition now separates successful entered work from the complete one-call envelope for the original cursor root and owned copy. Successful manifest encoding bounds its embedded journal string by the later8192-byte manifest cap; the original65536-byte journal guard is preserved. The generated spine/effect/cursor metadata ceilings are402/393/3062 bytes. The cursor ceiling retains the256-byte client field's possible six-byte JSON control escaping. Header timestamps keep the original safe-integer ceiling; wider block-number fields remain conservative. None of these size facts authenticates a source or selected counter.

An unapplied block is retained only after the original content decoder charges header plus all transactions and receipts to its selected evidence budget. The original gas checker uses a separate budget with the same limit. Their safe combined byte bound is twice that limit, not one shared limit and not a separate limit for every transaction. Other effect trace results keep their own per-field body bound. Snapshot queue text uses the same immutable aggregate already established for the fold; pending-command text stays separate. Logical descriptor/copy events do not establish shared backing size, native CPU or simultaneous RAM.

The complete cursor envelope retains all remaining old caller work, including pending and optional stop-record roots after a healthy late semantic refusal. It substitutes only a separately complete eligible semantic child. The owned-copy part also includes its initial root, arrays, pending descriptors and semantic snapshot. Snapshot ownership still requires the original observer; a numeric quote and a read-only inspection meter cannot acquire that role. Every invalid child or checked arithmetic failure returns a whole zero result.

Seven authored safe UNIT roots add independent integer/schema rows, empty/index and snapshot cases, original ancestry boundaries, all original nonempty content phases and owned copies, and a finite in-memory deferred-refusal comparison. The latter changes inert DATA only and adds no attack, endpoint, network or transaction operation. All494 authored roots remain NOT_RUN and290 individual90% targets UNMEASURED. Source review, repeated pointwise stage/future composition, all eleven sufficient AND feasible coordinates, native CPU/backing/peak/lifetime RAM, windows/failure/recovery/final/page/factory and the ONE complete connected publication gate remain open. Whole5-1/5/7/8 and the original5-2 through5-6 order remain incomplete; policy activation is separate.

### Successful root placements by original cursor state

The preceding one-cursor parts completed bounded independent SOURCE review. A separate successful-path argument checked each original caller's before/after-state placement. Its first review required an initialization attribution clarification, which the bounded successor resolved. FirstPReading already includes its own input interpretation and cursor CONSTRUCTION. The initial two models' cursor commitment/copy term, four roots and two copy extras, executes OUTSIDE that reader. A future complete source-priced initializer must assign both components once each. The current legacy-control-plus-two-reader numeric envelope alone is not proof that those external roots are priced; unused slack is not such proof.

State k means the original cursor after k successful S boundaries. A complete depth-j prefix has three roots and one copy when j is zero. Otherwise it has eight roots/two copies at state zero, nine roots/one copy at every interior state and four roots/no copies at state j. A current T window contains twelve such prefixes,97+12m direct roots and25+2m copies before S, and nine roots after S. Its own advancement adds two depth-j+1 prefixes and two after-state validations before that SAME window closes. The new private interval calculations preserve every existing coarse total; copies already include their initial root.

The whole successful topology includes the external initial models. For G groups, fixed state-zero weights are112G+25 roots and28G+11 copies. Interior state k has164+126(G-1-k) roots and25+14(G-1-k) copies; the terminal state has nine roots. Member requests add twelve roots and two copies at their original before-state. A prospective interval uses at most min(R,32*memberStates) requests. It does not subtract assumed requests in nonexistent groups: a shorter admitted program may place more members inside the interval. Separate interval maxima need not be attainable together. Their sum is conservative, not an exact transcript or proof of feasibility.

A conditional interval composition compares all eleven successful normal-work coordinates and counts root-only plus full copies. Its selected source bounds and empty/full condition must cover EVERY actual state in that interval. It supplies no original preflight or healthy-late-failure allowance, and cannot authenticate a range or mint a factory. Four authored safe UNIT roots add independent interval literals, source-call chronology and shorter-program comparisons, unchanged original empty/nonempty observer/copy behavior, and invalid/late-overflow whole-zero rejection. All498 authored roots are NOT_RUN and290 individual90% targets UNMEASURED. Complete pointwise stage/future headroom, failure/recovery/final/page, all-eleven sufficient AND feasible cost, native CPU/backing/peak/lifetime RAM and factory remain OPEN before the ONE connected publication. Whole5-1/5/7/8 and original5-2 through5-6 are incomplete.


### Completed known transport failure keeps the original before cursor

The completed retained known-failure path has three independently reconstructed original consumed-C readers: coordinator, before rename and after rename. Each branch computes thirteen before-state cursor roots, including three full owned copies, and no after-state cursor root. Failed S increments the ledger stage but leaves the collection cursor at the successful prefix; terminal DATA validation checks that exact relationship. This is a transport failure whose custody and roots succeed under the original healthy meter. A refused cursor primitive is a separate abort/quarantine path.

`providerHistoryCollectionFailedStageStateCallsBound` counts twelve depth-j prefixes, 100+12m direct before roots and 22+2m before copies, with no advancement. Actual attempts are retained+1 <= m; an earlier abort does not thereby receive a successful-normal price. The common interval includes j+1 with zero failed-path roots. External initial models remain outside this stage and firstPReading.

For a whole successful grammar or one whose last S records a known failure, `providerHistoryCollectionTerminalStateCallsBound` retains the successful interval upper and adds three roots when the interval can include the failed before state. The exact replacement removes nine after roots and three before copies; keeping those removals reserved gives a conservative bound. The shorter-program and independent request-band bounds remain in force. Full copies already include their root; normal coordinate pricing requires the same independently source-bound child for every state in an interval.

The existing failed-settlement regression now passes the required original context and observer to `settleOriginalsObserved`; its old context-free wrapper necessarily refuses a real collection. The legacy nil-profile behavior and the separate context-free rejection test remain unchanged. Three new authored UNIT roots cover independent state-interval literals and weighted nondominance, shorter/differently packed terminal grammars, and three separately owned terminal ledger models with the original observer, cursor and irreversible charged upper preserved. The latter exercises primitives and DATA, not authenticated original I/O readers.

All 501 authored tests remain NOT_RUN and all 290 individual 90% coverage targets UNMEASURED. Normal root work does not supply original incoming preflight headroom or healthy deferred-refusal allowance. Noncursor failure/result/capture/frame/inventory/write costs, all-eleven sufficient AND feasible whole/window/future reservations, native CPU/backing/peak-lifetime RAM, recovery/final/page and factory remain open before ONE connected publication and native CI. This change supplies no operational authority or activation.


### Original cursor component pointwise allowance and irrevocable reservations

The original live group caller graph stops later independent outer cursor roots/copies after the first such false return. This includes failed preparation before the member caller sets its entered flag: the original active-entry cleanup still quarantines the same lifetime. Semantic/pending/optional-stop deferred siblings are inside the complete child envelope. A completed known transport-failure custody path instead computes successful roots at its retained before state; it is not a refused cursor primitive or a retry. Recovery and final roles retain their own separate obligations.

`providerHistoryCursorPointwiseParts` keeps conditional successful-normal root work separate from complete incoming/refusal headroom for ONE fixed original accounting scope. Full-copy prices include their initial root. `pointwisePartsUpper` computes all eleven coordinates with checked arithmetic; the child root and full-copy envelope differences are equal because their snapshot/copy additions are the same. Taking their maximum preserves either primitive type. Invalid/zero-call bands and any late arithmetic failure return the whole zero result; valid zero-after failed-stage bands must be omitted.

For source-qualified disjoint bands in that SAME actual scope, `joinWithinScope` adds normal bounds and takes the largest headroom difference; `actualAndIncomingUpper` sums those two parts. Before any invocation, successful component prefix plus that invocation's complete incoming envelope fits this component upper. Neither numeric band counts nor a returned positive vector establishes the original graph/phase/limits/observer eligibility. Nonroot K and other composite incoming preflights still require their own proofs at initialization, whole lifetime, T window and own-advance origins.

The actual-component theorem DOES NOT reduce the irrevocable future hold. Every complete T/window upper must include its own headroom and nonroot/member/closure/advance/custody/storage/clock/native costs. The original future derivation must retain the complete initializer plus the SUM of those complete admitted uppers, including repeated per-stage headroom, and any separately priced outside-window tail. Uniform selected admission requires the original complete uniform maximum where appropriate. Actual-plus-incoming and initialization-plus-charged checks remain distinct. External initial4N0+2X0 stays once outside firstPReading; constructor/interpretation stays once inside the first-reader row. Recovery remains a disjoint original hold row. No slack-based attribution, refund, reset, role borrowing or old-hold reduction is accepted.

Four new authored UNIT roots compare the same source child and all-coordinate parts, exercise finite counter prefixes with the original inert meter's preflight/add primitives, show the original charged-upper addition rejects an actual-only bound, and require whole-zero on late overflow. Their190/189 and280/380 quantities are symbolic DATA, not operating quotes, actual cursor/I/O replay, native measurements or an authenticated source. All505 authored tests are NOT_RUN and all290 per-file90% targets UNMEASURED. Original guards/workflow/upper charging remain unchanged; complete all-eleven sufficient AND feasible costs, native CPU/backing/peak-lifetime RAM, recovery/final/page/factory and ONE connected publication/native CI are still open.


### Planned internal ledger control in one original scope

The private `providerHistoryCollectionPointwiseLedgerControlUpper` numerical
component uses the existing uniform32 planned ledger normal N and unchanged
complete generic control envelope E. For a positive, source-derived ledger-root
cap q in ONE original actual-accounting scope it returns checked `(q-1)*N+E`
across all eleven coordinates. A later overflow clears the whole result. This
is not the complete actual vector, an irrevocable reservation, a source witness
or an operating permission; no production caller uses the new function.

The normal row applies only to owned internal ledger state whose ticket was
installed AFTER independently deriving and comparing the exact original next
request table. Consumption is installed only after complete validation against
that ticket. Original value copies preserve fixed member/request/record values.
External request-pack/raw parsing and incoming ticket/C/S construction or
validation BEFORE that comparison retain their complete wider envelopes. A
private generic ledger, valid digest or positive numeric count cannot establish
the selected source grammar. Cursor commitment/copy work is priced separately.

For an eligible internal call, the complete E includes all internal guarded and
unconditional sibling work, including both direct and recursive ticket
validation in a consumed ledger. Complete control failure, including a refusal
AFTER successful cursor commitment, propagates to the same original caller and
quarantine before another independent ledger invocation. The arithmetic covers
prior successful calls plus the complete current incoming/refusal envelope.
It does not price unrelated nonroot/deferred cleanup, native CPU or backing RAM.

Completed known transport-failed S and successful S each have ten ledger-control
calls per original S replay branch. Their cursor state placements differ and
remain separately funded. Original counts are prefix `2+7j`, stage
`102+84j+10m`, same-window own advancement `20+14j`, and two external initial
model roots once. Complete I/T reservations each keep their OWN component and
all other nonroot/member/closure/advance/custody/storage/clock/native resources.
The irreversible future retains complete I plus SUM of FULL admitted uppers;
its per-stage headroom is never replaced by one whole-actual headroom, refunded
or borrowed across scopes. Original firstPReading excludes the external initial
models; their ledger roots and cursor4N0+2X0 must be assigned exactly once.

Four additional UNIT roots cover independent literal rows, repeated original
recursive ledger DATA under separate preselected inert meters, the real
counter's exact incoming/one-unit-short boundary and sticky refusal, and late
logical overflow after a positive earlier coordinate. These are authored,
NOT_RUN tests; symbolic envelope additions are cost DATA, not a reproduced
ledger failure or I/O/native experiment. All509 authored roots remain NOT_RUN
and all290 per-file90% targets UNMEASURED. All other original preflight scopes,
external parsers/builders, complete all-eleven sufficient AND feasible budgets,
native CPU/backing/peak-lifetime RAM, windows/failure/recovery/final/page/factory
and ONE connected publication/native CI remain open. Whole5-1/5/7/8 is incomplete.


### Planned record-control parts outside ledger validation

The private `providerHistoryPlannedRecordControlUpper` constructor returns
conditional normal control parts and the unchanged generic envelopes for one
T, C, S or result-context operation. It has no production caller or authority.
The existing planned-ledger request grammar must be known BEFORE using these
normal parts. Owned T/C values preserve that grammar only after the exact
original next-table admission and independent original reconstruction checks.
A typed private object or matching digest alone does not establish eligibility.
Incoming T/raw parsers and validators before exact source-table comparison keep
the original wider envelopes. No aggregate ticket-operation count is newly
eligible for the small normal by virtue of this numerical helper.

The normal T/C rows reuse the planned-ledger child and retain the FULL generic
8192-byte parser caller-copy/reconstructed-copy/equality suffix,32771 logical
units. Context retains direct T validation plus complete C validation, including
its second T validation. S keeps that context AND its entire old8192-byte
metadata canonical/copy/hash body67125293 plus the32771 suffix. Keeping the old
body/suffix deliberately over-reserves these normal parts; no new small S raw
size or untrusted input cap is inferred. One S row covers a parser with caller
copy OR construction plus full fixed-array equality; it does not fund both
operations together. The old generic record/TLS/result-context quotes are exact
unchanged values. Generic source domains and raw JSON guards are not narrowed.

C construction/parsing on a freshly reconstructed original admission and S or
result contexts on original consumed/settled models may use qualified planned
values. Arbitrary generic pure DATA and terminal recovery do not inherit this
qualification. TLS/HTTP/pack remainders, old generic raw comparisons, original
entered/deferred/native cleanup and all other callers remain separate. A normal
row alone need not fund the original wide incoming preflights; the separate
complete envelope is still required. No new repeated/pointwise/failure or full
composite cost derivation is supplied by this append.

Four additional UNIT roots contain independent1/2/3/32-member literal rows and
all valid member-count envelope comparisons; actual original typed T/C/S
constructors/parsers and first/last ordinal contexts; separate complete and
known-failed S DATA with full equality; standalone incoming shortage/sticky
refusal and generic1024-byte parameter behavior; and invalid-cardinality
whole-zero results. Their inert meters have full original envelopes selected
before observation. These are bounded offline DATA tests, with no source,
original-reader, live-role, TLS/native or I/O qualification. They are authored
NOT_RUN. All513 authored roots remain NOT_RUN; all290 per-file90% targets remain
UNMEASURED. Complete original scope-qualified counts, all17 composite/53scalar/
31zero-owner applications, full memberremaining SUM after entered preparation,
I plus SUM of FULL admitted uppers with each scope's own headroom/resources,
all-eleven sufficient AND feasible budgets, native CPU/backing/peak-lifetime RAM,
windows/failure/recovery/final/page/factory and ONE connected native CI remain
open. Whole5-1/5/7/8 is incomplete; no hold, guard, role or policy is changed.


### Flat admission CONTROL composition outside ledger roots

The two completed planning replays finish BEFORE a coordinator admission
leaf begins. A replay leaf instead contains one request-pack parse, every
original HTTP reconstruction, the rebuilt table, T parser, DISTINCT T
validation by admission, full table equality and admitted-root completion.
The independent flat-leaf argument accepts only the completed-postcondition /
one-pending-generic structure. It does not grant planned shape before the fence.

`providerHistoryAdmissionClusterControlUpper` prices the assigned CONTROL
subgraph of both leaf kinds: coordinator pack build + two T operations + full
table equality, or replay pack parse + m original HTTP reconstructions + rebuilt
table + two T operations + full table equality. Each coordinate takes the larger
kind. Ledger copy/control/root/cursor, source nextTable generation and all
surrounding object/inventory/frame/file/native, clock, RAM and cleanup work are
separate K at their ACTUAL positions. This component alone is not a sufficient
whole-atom incoming allowance or a complete admission compiler.

Completed full wire and source next-table equality constrain earlier original
requests to the existing500/578/178/756-byte template rows. The successful
request descriptor has seven closed fields and m fixed request rows. Its ASCII
outer extent is below512 and each row below192 even over-reserving every decimal
at20 bytes; therefore header<=512+192*m and successful raw bytes are at most
header + magic +4 +756*m, also capped by the unchanged selected raw maximum.
The generic old64KiB descriptor/raw/record envelopes and ALL original guards
remain unchanged. An unfinished pre-fence leaf always retains the whole generic
component; a private table or matching digest cannot select its normal.

The separate pointwise function is conditional DATA for one original actual
scope: `(atoms-1)*normal32 + envelope32`. It takes a flat LEAF count, never the
old count of individual T operations. Source-derived counts, original root abort
propagation and external K/deferred incoming bounds are still required. EACH
I/T stage retains its OWN full component; future remains I + SUM of full admitted
stage uppers, with the original post-preparation closure + remaining FULLmember
SUM. No global single headroom, refund, borrowed window or production caller is
introduced. Existing repeated request/control functions are byte-for-byte intact.

Four new authored UNIT roots compare independent complete1/2/3/32-member rows,
all32 member/cardinality domains and a genuine late logical overflow; observe
both original CONTROL call sequences on separate preselected generic inert
meters; retain an actual positive decode-refusal prefix and reject retry; keep
the larger generic DATA domain and differing full-table rejection; and check an
independent over-reserved closed descriptor extent. These are typed DATA/control
fixtures, not real original readers, a complete admit call, source qualification
or whole-atom reservation tests. No project/compiler/test/runtime was executed.

The current517 authored roots are ALL_NOT_RUN and290 mandatory individual90%
targets UNMEASURED. The restored complete1478-file checkout is only a separate
673-source integration PREPARATION; it has no new commit/push/CI and does not yet
contain this later674-source append. No result is carried across those versions.
All17 full composite/53 scalar/31 owner-check scopes, all-eleven sufficient AND
feasible I/stage/future/whole totals, native CPU/backing/peak-lifetime RAM,
windows/failure/recovery/final/page/factory and whole5/7/8 remain open.


The flat-leaf CONTROL normal now prices the HTTP scan explicitly at
min(wire756,512)+5, and its local seven-field equality explicitly includes
2*wire+1. This adds334*m+5696 logical units to the prior whole normal (16384
at32 members); its generic envelope and every original API/guard are unchanged.
The scan/equality terms no longer need the other pack/request-copy spare used
by the earlier coupled whole-normal proof. One component cannot spend another
component's remaining headroom. Per-kind maxima and every full stage SUM remain.
The source table domain `sage-provider-history-members-v1` is32 bytes; earlier
private evidence prose saying31 is an explicit prose-only erratum, while its
numeric field/product len and historical literal calculations already used32.


### Completed-stage scratch retirement and source atom counts

Successful nonfinal advancement retains all original checkpoint/claim/ACK,
frame/inventory/admission/member archives and consumed/sent facts. After both
fresh retained-original replays and every final check, it drops only the old
T/C/S claims' raw-pack/catalog/model pointers and old dispatch model pointers.
The exact distinct original claims are captured and checked before the native
end; only that fixed local selection is used in the scalar publication tail.
No pointed ledger/cursor, original file, reservation or counter is reset. A
final, canceled, failed or pending current stage is not retired by this path.
The old S model can alias the live owner: field detachment must never overwrite
that model. Newly published coordinator/custodian models remain independent.

The unchanged pre-end CopyEvent(4096) funds the combined fixed structural tail
once. Conservative64-bit target values are442 old bytes,1288 retirement bytes,
32 captured-pointer bytes; double-sided traffic plus32 words is3780<=4096.
This is logical structural accounting, not native CPU/allocator/write-barrier,
GC, backing-memory or a complete resident/lifetime certificate. Remaining
checkpoint/request facts and caller-held outputs still need full accounting.
Genuine owned multi-stage and collection fixtures, exact original files and
separate terminal readers are authored test subjects; all remain NOT_RUN here.

Flat admission leaf counts now have separate same-window, whole-successful
and uniform derivations. With j completed prior groups, T planning contributes
2j, new T/readbacks2j+3, C3j+3, dispatch2j+2 and S3j+3. Own nonfinal advancement
adds2j+2 inside the SAME old T window. This gives11+12j without advance,
13+14j with advance and11G+14G(G-1)/2 over G successful groups. The uniform
maximum compares the last12G-1 with the nonfinal14G-15 when G>=2; neither
same-C member replay nor initialization adds a new T admission leaf.

These counts supply no role/provenance or factory. Every original full stage
retains its OWN (q-1)N32+E32 component and all other original K/native/memory/
abort/failure/final work. A single whole-program E cannot replace that SUM.
Using current R2 CONTROL N32 logical806614817 and E32 logical90403378492,
the illustrative G9226 uniform q129149 component is decoded19782478068,
logical104263093764408; the SUM of9226 such full stage components is
decoded182513142655368/logical961931303070428208. This is finite arithmetic
for this subgraph ONLY, not a selected real-history/profile/operating bound.
All11 sufficient AND feasible totals, native CPU/backing/peak-lifetime RAM,
remaining scopes/failure/recovery/final/page and production factory remain open.


The first retirement bundle's SOURCE review found two authored test defects,
not executed CI failures. Its18-member collection positive used the old128KiB
pack selection, which fails the original prospective guard before any member;
the13 normal recorded code response bodies alone total304000 bytes. A separate
UNIT selection now chooses4MiB aggregate objects and8MiB first-read raw headroom
BEFORE manifest/profile/anchor/H0. The original128KiB fixture remains default.
The existing closed-history first-P plan is recomputed against this selection
and must fit the unchanged conservative startup147226362 decoded/948444 retained
and256MiB operation/segment limits. Whole held storage remains59668700 bytes
inside64MiB; stage raw4MiB/physical6MiB and every product guard remain intact.
The loose selected18-member pack bound2040867 fits4MiB; individual response size
is never substituted for that guard. The genuine positive explicitly checks it
before member entry. This does not qualify every possible response/profile.

The portable structural UNIT's32-bit expectations are corrected to old250,
retired644 and combined1948;64-bit442/1288/3780 is unchanged. The earlier350/2148
literals were an arithmetic error and remain in immutable failed-review history.
Neither correction changes the full4096 event, source retirement/count behavior,
operating caps, protocol or source/native/host qualification. Tests are NOT_RUN.


### Original-window composition for entered member preparation

The same original T window includes T/C/dispatch, every member, successful or
recognized failed S, and its own successful nonfinal advance. The literal
remaining-member preflight runs after the current member's begin sample, two
original role-equivalence passes and retained-prefix preparation. A complete
normal sum alone therefore omits preparation already entered when the guard
asks for every remaining **full** member and closure.

The private numeric `providerHistoryCollectionStageWorkUpper` assembles
`U = B + P + n*M + C + optional A` in all eleven work coordinates and the
separate clock-count domain. B is a complete T/C/dispatch prefix envelope, P
bounds the current preparation already entered before that guard, M is the
same complete member row used by `remaining`, C must cover both successful and
recognized failed closure, and A is the complete advance inside the old T.
Every child still needs its original source-completeness and internal incoming
proof, including recurring first-reader work, deferred cleanup and failure.

At ordinal k the actual prefix is at most `B+k*M+P`; the unchanged incoming row
is `(n-k)*M+C`. Their sum fits U, including on a final stage with no A. The
extra P is reserved headroom, not a second actual event or an additional
member. B/P/C/A request and wire coordinates must be zero; the resulting
request count is exactly n and wire is n times the selected member row.
P must fit inside M, including clock counts. Invalid shapes and any checked
integer overflow return an entirely zero result with false.

This numeric result does not replace the original collection selector's exact
`32*(capture.WireBytes+1)` maximum or authorize refining non-0/1 coordinates
of an already selected full maximum. Every original admitted T still carries
its own complete U; the irrevocable future sums initialization and every full
stage, with separate full physical/resource rows. First-reader, advance,
member-capture and lifetime sublimits remain enforced independently.

The conditional argument was independently accepted before this source append.
Four new safe UNIT roots cover every ordinal at counts 1 through 32, all eleven
coordinates, exact request/wire, the omitted-P counterexample without advance,
invalid source-row shapes, and early/late work or clock overflow with whole-zero
failure. The tests use scalar DATA only and do not patch a live owner or quota.
They remain NOT_RUN; all 528 newly authored roots are NOT_RUN and the existing
290 individual coverage targets remain UNMEASURED. A positive scalar row is not
a measured test result, source witness, sufficient/feasible selected profile,
production factory, native CPU/backing/peak/lifetime RAM or elapsed-time proof.
No original guard, cap, clock origin, admission, storage domain, policy or
operating authority changes. The complete source-cost compiler and connected
qualification remain required before production opens or one complete bundle
is published.


### Role CONTROL components and owned-clock counts

The selected original call graph partitions the two previously derived CONTROL
components into initializer I, prefix B, entered preparation P, full member M,
closure C and own nonfinal advance A. At j prior successful groups, ledger-root
counts are I2, B(66+63j), P10, M10, C(36+21j), A(20+14j). Flat admission leaf
counts are I0, B(8+9j), P0, M0, C(3+3j), A(2+2j). P/M's zero admission counts
mean the fixed source subgraph contains no such leaf; they do not grant an
untrusted caller permission to omit a required fence. The collection wrapper
still validates the original maximum through its nonempty B component.

Each role's private numeric CONTROL DATA adds its own complete pointwise ledger
and nonempty admission component, then its original owned-clock event count in
logical coordinate 10 and in the separate clock-count domain. The same call's
normal/envelope fences, current owner and first-refusal obligations still apply.
No independent global headroom can replace every full admitted role/stage.
All counts, products and eleven-coordinate additions are checked; any invalid
input or late overflow discards the entire result. Final-stage A stays empty.

Explicit owned-clock counts are I2, B6, P1, M2, C2 and A2. The successful source
stage executes 8+2n+2adv samples, but the unchanged literal remaining guard still
asks for the full current M after P. Its full upper therefore holds
9+2n+2adv samples. Summing I once and every full stage gives 11G+2R, compared
with actual source count 10G+2R. Reserving only one global P underfunds G-1.
Clock event weight one is a logical metric, not native CPU, elapsed time or all
implicit clock instructions in TLS/context/backend sampling.

The 73 final /75 nonfinal numbers at n=32 require exact incoming M2/C2/A2 rows
selected and bound before H0. Existing original rows may declare wider member
or closure clock allowances. The unchanged stage assembler retains those
values: M4/C3/A2 at n=3 requires 24 clocks, not the source-count 17. This helper
neither lowers existing holds nor raises the original clock cap or renews time.

Successful and recognized failed closure share these CONTROL counts but retain
distinct state-weighted cursor costs. Cursor commitments/copies, other K work,
RPC/wire/capture/retention, file/deferred/failure/final/recovery, native CPU and
backing/peak/lifetime RAM remain separate. CONTROL rows alone have no complete
RPC/wire/write dispatch. This append supplies no production caller, source
witness, factory, profile, policy activation or authority.

Five new safe UNIT roots use independent numeric literals, absent-component and
final-stage cases, invalid scope and late overflow boundaries, every full-stage
clock sum, and preservation of wider original clock rows. They use scalar DATA
and do not change a live owner, hold or quota. All 533 newly authored roots remain
NOT_RUN and 290 individual coverage targets remain UNMEASURED. Source-complete
all-eleven-dimensional sufficiency AND feasibility, native/resource/window and
connected qualification gates remain required before the complete publication.


### Per-role cursor state bands and terminal alternatives

The same original B/P/M/C/A cuts now have a private cursor-root/copy component.
A fresh prefix at j previous successes is distributed over its actual earlier
states. B uses nine such prefixes plus61 direct roots/13 copies at state j.
P and M each use12 roots including2 copies, entirely at j and before literal
remaining. Successful closure uses three prefixes plus36 roots/12 copies at j
and9 roots at j+1. Known-failed closure instead uses39 roots/9 copies at j and
no j+1 state. Own nonfinal advance has two prefixes through j+1 and two more
roots at j+1, inside the old successful T. Copies include their initial root.
The external initializer's4 roots/2 copies at state0 remain a separate row
that the complete compiler must add only once.

The prospective state bands must cover0 through j+1 in order, without gaps,
overlaps or reordered intervals. Every band's selected limits/history maximum
and source kinds must bound EVERY original state in that band. This numeric
input cannot establish those facts: the original source factory still must bind
them before H0. Earlier states are never priced using a later cheaper state
or unqualified current cardinality. A source-proven empty role/band contributes
no component; it does not weaken positive-count child validation. The successful
closure's after-state always validates the final band, even if failed closure
has no calls there.

Each role joins its state-weighted normal parts and the largest complete
first-refusal/incoming delta from the existing original cursor children. That
headroom belongs to the same role/accounting scope. It does not replace each
admitted role/stage with one whole-program headroom or erase the incoming full
member requested after P. P and M's cursor components are equal; the other
CONTROL/clock/capture/retention components remain distinct.

Successful and recognized failed closure are priced separately, then compared
coordinate by coordinate. A failed terminal can retain expensive pending work
that successful S clears. Using only the cheaper after-success result would
underfund failure; inventing a simultaneous union of both paths is not their
source count either. Invalid bands/kinds, child refusal, repeated-role overflow,
late band-join overflow and final normal-plus-headroom overflow all return the
entire empty result. Earlier successful initializer/band parts cannot escape.

Six safe UNIT roots cover independent state-cut literals, conservation against
the existing successful/failed grammar, state-domain boundaries, independent
uniform root/envelope constants with portable explicit copy-event sizes,
partition invariance, combined disjoint CONTROL/cursor stage DATA, failed-before
cost dominance, invalid later bands, and checked late overflow. Their scalar
OTHER-domain examples create no real owner/profile or execution authority.
All539 authored roots remain NOT_RUN and290 individual targets UNMEASURED.
No production caller/witness/factory, original guard/cap/window/clock/policy or
operating permission changes. Boundary application and the other K, raw/capture,
file/storage/deferred/failure/final/recovery, all-eleven-coordinate sufficiency
AND feasibility, native CPU/elapsed and backing/peak/lifetime RAM remain separate
obligations before one complete connected qualification/publication.


### Per-role generated tables and remaining standalone record control

The next disjoint component prices each original nextTable's request generation
AND final table construction, residual whole-table equality, and standalone C/S
record work outside ledger validation and TLS/result-context validation. Pack/T
table reconstruction, admission equality and both ticket operations already
belong to the earlier admission atom; they are not priced again here.

For j previous successful groups the source counts are:

| Role | nextTable | residual table equality | standalone C | standalone S |
|---|---:|---:|---:|---:|
| B | 10+18j | 1+9j | 10+18j | 18j |
| Successful C closure | 6+6j | 3+3j | 6+6j | 6+6j |
| Known-failed C closure | 3+6j | 3j | 6+6j | 6+6j |
| Own nonfinal A | 4+4j | 2+2j | 4+4j | 4+4j |

I/P/M have no calls in this component. Their complete work remains nonzero and
separate. B includes two direct planning selections, eight current admission
atoms and nine prior-prefix reconstructions. Each prior successful group inside
a fresh prefix adds one T selection and one successful S boundary selection.
Each of the three current successful S views adds both its T selection and its
boundary selection. Recognized failed S adds only the former. Own A replays two
prefixes through j+1; selection occurs before each boundary transition, so it
uses states0..j and never constructs the after-state's next table.

The residual equality count removes the equality already included in every
admission atom. With G successful groups and J=G(G-1)/2, summing the new nextTable
counts gives16G+28J, exactly old tableBuilds minus packParses. Residual equality
is4G+14J; adding11G+14J admission equalities recovers15G+28J. The two ticket
operations per admission atom already recover22G+28J ticket calls. New C/S sums
are16G+28J and6G+28J. A known-failed final group removes three current nextTable
calls and three residual equalities; it retains the same C/S record calls and
has no own advancement. These are conditional source-count identities, not
measured live execution or permission to skip an original reader.

The generated-table normal row reuses only the completed original planned
request shape. Tags/options/spine/ABI and the entire selected pending-effect
scan keep their prior overhead. Final table construction retains both capture
class decodes, the full fixed-table canonicalization and digest. Every historic
state must be bounded by the selected effects/maximum before H0; reported actual
values and private typed tables do not prove this. Generic raw/unfinished work
keeps the unchanged complete envelope and all original guards/preflights.

Each positive component uses(q-1)*N+E inside its own role/accounting scope. A
source-proven absent component alone skips its envelope. All eleven coordinates
are checked, including N<=E, invalid original bounds and late record or sum
overflow. Any refusal discards the whole result. Success and known-failed S
are priced separately before coordinatewise MAX. Full admitted roles/stages
still sum their own complete allowances; no global headroom or refund replaces
that sum. Final A is empty, and no new production caller/factory is supplied.

Six safe UNIT roots cover independent numeric pairs, every original request
phase under the unchanged generic incoming envelope, independent role counts,
conservation against the existing whole grammar, independent success/failed
prices, legal last-stage and invalid/late-overflow boundaries. They do not run
a network, transaction or adversarial host. All545 newly authored roots remain
NOT_RUN and290 individual targets remain UNMEASURED.

Hash-binding comparisons, TLS/result-context controls, actual boundary application,
raw/capture/file/storage/deferred work and full failure/final/recovery remain
separate K obligations. The complete source cost across all eleven coordinates,
sufficiency AND feasibility, native CPU/elapsed work, backing/peak/lifetime RAM,
resources/windows/child sublimits and original complete factory must still close
before one connected publication and new CI/actual-artifact qualification.
Existing published CI does not validate this unpublished source increment.


### Per-role TLS diagnostics and standalone result contexts

The next disjoint CONTROL component assigns diagnostic TLS construction/parsing
and standalone result-context validation. A TLS constructor/parser already
contains ONE complete context validation, which validates T directly and again
inside C. Only result-pack and failure-original context call sites are added
separately. Earlier ledger/admission and standalone C/S record components remain
distinct invocations. HTTP/RPC/result-pack/failure-format bodies and their native
work are not silently included in a diagnostic TLS row.

The successful original canonical TLS shape is bounded by2048 fixed bytes plus
70 bytes per selected certificate digest and4 per selected chain. Fixed fields
use the official endpoint/server, bounded stage/ordinal/version/cipher strings
and six hashes. Canonical equality and original owned T/C validation reject
extra, changed, escaped or unbounded fields. With at most8 chains/16 certificates
the prospective maximum is11040 bytes. This is a completed-normal postcondition,
not a smaller raw decoder, incoming allowance or native certificate bound.
The original generic64KiB envelope, certificate-byte selection and guards stay
unchanged. Diagnostic JSON does not establish a live connection or its origin.

For j prior successful groups, s prior members in total, n current members and
k current ordinal/retained completed prefix, the actual source counts are:

| Role | TLS constructors | TLS parsers | Standalone contexts |
|---|---:|---:|---:|
| B | 0 | 9s | 9j |
| P | 0 | 2k | 2 when k>0, otherwise0 |
| M, including P | 1 | 4k+2 | 4 when k>0, otherwise2 |
| Successful closure | 0 | 3(s+n) | 3j+3 |
| Known-failed closure | 0 | 3(s+k) | 3j+3 when k>0, otherwise3j, then+5 |
| Own nonfinal A | 0 | 2(s+n) | 2(j+1) |

The five failed-closure contexts are real distinct source invocations: the
failure writer calls its constructor once, and each of the two readback parsers
validates context in its guard and again in its reconstructed constructor.
The failed result pack is absent when k=0; these five failure contexts remain.
They were outside the older successful-only aggregate. A future complete
failure-format component must exclude these already assigned contexts when
composed here, rather than adding its whole recursive context a second time.

For a successful actual program, adding M once per member, B/C/A once per group
and no separate actual P gives R TLS originals,4R+G+14J result-pack contexts and
2*sum(n_j^2)+3R+14*sum((G-1-j)*n_j) TLS parsers. Every TLS row includes its own
context, so all actual contexts are the sum of these three counts. This agrees
with the original prospective aggregate, including variable group cardinality.
The full stage HOLD still retains P once per stage before the unchanged literal
remaining guard; actual-count conservation does not remove that headroom.

The uniform wrapper bounds historical T/C at32 members and earlier total s by
32j, with k=n-1 bounding every possible current ordinal. All original immutable
profile/ownership/normal fences must be bound before H0 by the future complete
factory. Private numeric counts, diagnostic data and zero component rows do
not supply those facts or authority. Each positive disjoint component retains
its own(q-1)*N+E in every role. Success and failed closure are independently
priced before coordinatewise MAX: at n2/k1 with TLS2x4, five extra contexts
make failed closure exceed successful closure by120152 decoded bytes and
857758276 logical units despite three fewer TLS records. Final A is absent.

Six safe UNIT roots cover independent diagnostic/context numeric pairs, original
typed constructors/parsers under unchanged incoming E with completed actual<=N,
maximum certificate-digest shapes, normal-row incoming refusal, independent
source cuts, variable-member aggregate conservation, failed-closure dominance,
P<=M and final A, invalid original domains, and late child/join overflow with
whole-zero refusal. No live connection, certificate verification, network or
transaction is created by these numeric/diagnostic tests. All551 newly authored
roots remain NOT_RUN and290 individual targets remain UNMEASURED.

No production caller/factory/source witness, profile/cap/window/clock/hold or
operating policy changes. Hash and boundary application, raw/HTTP/RPC/result and
failure bodies, file/storage/internal incoming/deferred/final/recovery work,
all-eleven-coordinate sufficiency AND feasibility, native CPU/elapsed time,
backing/peak/lifetime RAM and complete resource/window qualification remain
required before one connected publication and new CI/actual-artifact validation.


### Per-role request, predecessor and settled-result hash bindings

This disjoint component covers only the final rebuilt request-table digest
comparison, the T/C/S transition predecessor comparisons and successful S's
original result-digest comparisons. Generic pack-internal hashes, ledger/record
validation and the many other member/boundary comparisons retain their earlier
rows or separate K. This is not a bound on every hash in the program.

For j prior successful groups, s prior members and n current members:

| Role | Rebuilt request-table binding | Transition predecessors | Settled original result bindings |
|---|---:|---:|---:|
| B | 7+9j | 13+27j | 9s |
| Successful closure | 3+3j | 9+9j | 3(s+n) |
| Known-failed closure | 3+3j | 9+9j | 3s |
| Own nonfinal A | 2+2j | 6+6j | 2(s+n) |

I/P/M have no calls at THESE three sites. B's coordinator builds its own request
pack, while seven current readers parse it. Its eight current T transitions
and five current C transitions give thirteen predecessor comparisons. Each old
group in a prefix has one request-pack binding, three transitions and its own
member result bindings. Closure has three current T/C/S views; failed S skips
its three current successful result loops. A has two whole successful prefixes
through j+1, entirely inside the old nonfinal T. Summing successful actual cuts
gives10G+14J pack bindings,22G+42J predecessors and3R+14*weightedPriorMembers
results, matching the older three assigned aggregate counts.

Each comparison observes two original66-byte hashes plus one logical event,
so N=E=133 at these source-qualified sites. Request-pack shape validates its
expected table before parsing, and the rebuilt table produces its own digest.
T/C validation precedes their predecessor comparisons. A successful ledger
validation means its head equals a fresh66-byte digest. The assigned external
request-table binding was deliberately excluded when the earlier admission
atom unrolled pack/request/table work; pack-internal digest events remain in
the raw-pack component and are not added again here.

The S predecessor requires special care: settle compares S.previous BEFORE its
own reconstructed S checks. The width must already come from the original
caller's new/parseSettlement constructor, which checks the expected previous
hash before returning. Similarly, original member parsing produces result.digest
before settle compares it; the later boundary guard is not an earlier fence.
Arbitrary damaged private S/result fields are not covered by this numeric row.
The original source factory must bind these actual reader/producer routes and
uniform historical cardinalities before H0; private hashes and counts are not
proof of those conditions or permission.

All counts, their sum and multiplication by133 are checked. Any invalid domain
or overflow returns the entire zero result. Every other work coordinate stays
zero for this comparison-only component. Success and failed closure are priced
separately before coordinatewise MAX. Final A is absent, and no global headroom
or refund replaces each original admitted stage's full allowance.

Five safe UNIT roots cover independent source-cut/count/price literals, skewed
and uniform whole-grammar conservation, original equal/unequal fixed-hash event
behavior, one-short refusal retaining an earlier successful prefix, original
T/C/S/member producer width guards, final-stage/domain checks and finite/product
overflow endpoints. They use finite inert DATA and create no live owner, TLS
connection, transaction, attack reproduction or source authority. All556 newly
authored roots remain NOT_RUN and290 individual targets remain UNMEASURED.

No production caller/factory, original guard/cap/profile/clock/window/hold or
operating policy changes. Boundary application, other K/raw/HTTP/RPC/failure
bodies, file/storage/internal incoming/deferred/final/recovery work, all-eleven
coordinate sufficiency AND feasibility, native CPU/elapsed work and backing/
peak/lifetime RAM and complete resource/window qualification remain open before
one complete connected publication and new CI/actual-artifact validation.


### Original boundary RPC interpretation CONTROL

The boundary interpreter now has separate normal/envelope DATA prices for original headers, RPC hex data and the four pinned Registry ABI result forms. The prices include one caller copy of the result string. They do not include the earlier HTTP/RPC-envelope work, semantic state/history work, table generation/equality, cursor copies/roots, native parser/hash CPU, backing/peak/lifetime RAM, storage, window qualification or a source derivation witness. They have no production caller and do not open a profile.

Raw result bytes must come from the original successful member reader, whose strict RPC envelope already checked the selected method body limit, UTF8, surrogate pairing and duplicate names. The original header reader independently repeats those checks. A numeric quote is not that producer proof. In particular arbitrary private strings that did not pass the earlier envelope are outside the data/ABI rows. The factory must bind each selected class and that exact original route before H0. The header caller still supplies its original global guard; the smaller class bound comes from the earlier member producer, not a changed parser limit.

A decoded short quantity is not a bound on raw JSON. Whitespace, ordinary field order, additional header fields and valid escapes remain supported. No whole-header canonical equality currently limits their original spelling. The full selected class raw maximum b therefore remains in both N and E. UTF8 plus valid surrogate pairing ensure a returned JSON string is no longer in bytes than its original raw value. In the header object the four named RawMessage values occupy disjoint ranges: their total length is <=b and their sum of squared lengths is <=b squared. This is a raw-range proof, not an assumption that every field has a typical short representation.

For header CONTROL, including the caller copy:

- N decoded = 3b+32; N logical = 3b squared+3b+1310.
- E decoded = max(N decoded,4b); E logical = max(N logical,4b squared+5b+730).

Both quantity calls are evaluated even if the first refuses. The failed quantity codec keeps its raw-derived width; it is not reduced to16 digits. Each quantity encoder keeps its original16 preflight. Successful quantity equality establishes the16-digit bound only for completed normal calls. E does not infer a small width from a later comparison. The maximum with N preserves a valid pair for tiny allowed raw quotas, which need not admit any successful header.

For original RPC data CONTROL, N=E: decoded =2b and logical =2b squared+2b+4. This includes JSON decode, lowercase check, hex decode and one caller copy. The earlier strict-envelope UTF8 fence is necessary; arbitrary malformed UTF8 passed directly to encoding/json can be repaired and is not represented by this narrower original-route row.

For ABI, f is the number of byte fields (one or two), z is one for the bool/bytes record form, and D=32(2f+z)+sum(roundUp32(field maximum)). Only the pinned bool/one-field and non-bool/one-or-two-field shapes are quoted; per-field maxima are <=262144. Define h=min(b,2D), t=min(b,2D+2), copied=min(sum(field maxima),floor(b/2)), and tail=2114f+copied+63z. Then:

- N decoded = b+h+64f; N logical = b squared+h squared+b+t+4+tail.
- E decoded =2b+64f; E logical =2b squared+2b+4+tail.

Successful exact ABI cursor exhaustion bounds inner hex to D, while the outer JSON retains b. A failed tuple may have already decoded all allowed original bytes before the offset/length/cursor check refuses; E keeps that work. Each offset and length is rpcUintObserved on a full32-byte word (decoded32/logical1025), not the distinct replay ABI word primitive. Per field, padding equality costs at most63 logical units, and copying contributes its actual length plus one. Exact offset==cursor precedes every copy, so already copied value slices do not overlap even on a later refusal. Optional bool equality contributes63. No runtime ABI limit or parser behavior changes.

The phase DATA quote uses the selected scalar, header, code and state classes. Opening combines two original header calls, chain/client/Registry checks, genesis equality and all13 code decodes/hashes/equalities. Closing combines four header calls (both extra headers are evaluated before their errors are tested), those identity checks and one independent spineHeader equality. The latter is a direct indexed lookup, not a scan. Ancestry covers up to32 headers and parent equalities plus a charged copy of at most the selected inclusive block count in spine slots. The slot event is the original logical argument, not an allocator-size claim. State, grants and history quote their original ABI forms, with history up to32 result parses. Begin/finish-window and post-ABI semantic work remain separate. Content/effects have no result parser from this assigned subgraph and are not certified as zero-cost boundaries.

The fixed expected Registry URN comes from the original journal constructor. ABI value limit128 and current client limit256 precede their respective later comparisons; a prior client came from the same successful original version guard. Chain ID is compared before acceptance: N uses the successful8-byte spelling, while E keeps its raw-derived width. A bad code decode can still be followed by Digest(nil) in the original loop; both that row and the final code hash equality are retained.

Seven UNIT roots contain independent literal parser and phase sums, ordinary whitespace/order/unknown-header-field/valid-escape forms, all four ABI shapes, original opening/ancestry/closing functions using existing inert recorded responses, ordinary incomplete input refusal, sticky first-codec quota refusal and invalid/overflowing quote checks. No responder, bypass, attack reproduction, RPC call, policy, new authority or paid transaction is created. The tests are NOT_RUN; coverage remains UNMEASURED until new native CI. Standalone formatting is not a compile/test result.

This is a disjoint interpretation component, not the complete source cost. Older first-P interpretation or broad ABI rows must not be added again for the same assigned call. Phase occurrence caps and weighted prior-prefix placement, post-ABI/window/content/effects semantics, other K/failure/raw/file/internal incoming/deferred/recovery/final costs, all eleven coordinates and feasibility, native/RAM/resources, complete factory and one complete connected publication/native-CI/actual-artifact validation remain open. The original order 5-1 then 5-2 through5-6, fixed1-4 complete/5 in progress/6-10 pending, and whole5/7/8 incomplete are unchanged.


### Shared-item boundary interpretation roles

The boundary interpretation cost now has a finite, source-derived marginal model for original T roles. It uses the same immutable collection manifest and selected method limits; it does not create a production caller, runtime upper, source witness, owner, permit, profile or factory. Runtime/parsing limits, holds, remaining guards, clocks, protocol and public APIs are unchanged.

The eight prospective phase group caps match the original MaximumCounts grammar: opening and closing each have ancestryWindows+contentWindows; ancestry has ceil((blocks-1)/32)+ancestryWindows-1 when blocks>1; content and effect groups use the original partition bounds; state has contentWindows; grants has presentStops; history has min(versions,ceil(versions/32)+contentWindows-1) when versions>0. Their sum is the same original maximum group count. The source stage domain remains1..349525. No new smaller operating cap is chosen.

Two shared item constraints matter independently of group caps. One original prefix visits at most blocks-1 ancestry parents, even when windows split them into more groups. History likewise visits at most versions items across all windows, not32 new items in every split group. The history proof uses beginWindow's confirmed baseVersion, selectHistory's base<=stop<=selected versions and exact queue difference, sequential historyNext and final confirmation of that window's same state. The differences telescope across successful windows. A failed current group stops before confirming another window; it does not renew this maximum.

For normal interpretation, there are11 fixed marginal tiers: opening; closing; ancestry full32-item groups; one partial ancestry group; remaining ancestry copy-only group positions; state; grants; history full32-item groups; one partial history group; remaining empty history positions; and content/effect positions with zero ONLY for this assigned interpretation component. The ancestry per-group copy uses the original <=blocks+1 logical event, and each parent uses its own original header/copy and133 comparison. History uses its per-item original two-field ABI quote. Split groups can add copy positions, but never extra raw items. Empty marginal tiers are a conservative allocation relaxation, not an assertion that a real group with zero members can be admitted.

The per-coordinate marginal costs are nonincreasing within ancestry/history item allocation. Taking the k highest prices among at most11 tiers gives U(k), a conservative maximum for any k earlier original groups subject to all these shared caps. Each coordinate can select a different ordering; the result is an upper, not an original trace. The algorithm loops over fixed11 tiers and11 coordinates, not over G or a supplied byte length. It leaves caller-owned model arrays unchanged.

Let delta be the coordinatewise maximum Ephase-Nphase over phases actually present in the selected grammar. Phase E retains all sibling header calls even after a prior header error. For ancestry/history a variable group has at most32 items, so the existing maximum-size phase delta bounds every smaller group. No individual-header first-false shortcut replaces the complete phase E.

The original prefix loop returns immediately after a bad healthy checkpoint. Planning's two prefix reconstructions are sequential loop iterations with an error return before the second; the two dispatch passes and own advance also check the first error before starting the second. Nested readers propagate a bad prefix. A bad successful-S closure defers failLocked and cannot fall through to retainFailedCaptureLocked, which requires the earlier capturing-member phase and the same healthy meter. Thus one complete attempted phase E, including its siblings, is needed in each assigned whole-role row; previously completed phases use N.

For j prior groups, the assigned role prices are:

| Role | Normal prefix and phase envelope |
|---|---|
| B, j>0 | 9U(j)+delta |
| B, j=0 |0 assigned boundary interpretation |
| successful closure Cs |3U(j+1)+delta |
| known-failed closure Cf, j>0 |3U(j)+delta |
| Cf, j=0 |0 assigned boundary interpretation |
| own nonfinal advance A |2U(j+1)+delta |
| final advance |0 |
| I/P/M |0 assigned advanceBoundary invocation, not zero whole cost |

Cs and Cf are priced independently and only then combined with coordinatewise MAX. Their raw rows are retained separately; a future compiler uses closure MAX, not Cs+Cf+MAX. Own A remains in the old T window. None of these prices borrows another T's headroom or subtracts an actual report.

The irrevocable sum of this component across G original T windows is:

14*sum(U(j), j=0..G-1) +3U(G) +(3G-2)*delta.

The normal term follows the9/3/2 prefix topology and no final A. For a sorted tier of c positions starting at rank r, its contribution to sum(U(j)) is price*[c*(G-1-r)-c*(c-1)/2]. The one-call model computes this with checked integer arithmetic and finite tier loops. Prefix-only queries do not compute an unused weighted result; a weighted future overflow cannot retroactively reject a finite numeric prefix. Every invalid source selection/count, late multiplication, sum or incoming delta overflow returns complete zero and false. Positive DATA remains insufficient for execution.

Independent scalar examples using the existing inert method classes (scalar1024/header16384/code65536/state65536) give the following component-only future pairs (decoded,logical):

- deployment-only four-group grammar: (95907840,5738866279720);
-13-block/4-wide grammar: (8190773568,477881741930739);
-71-block/64-wide grammar: (9026750336,502149187897860);
-38420-block/4096-wide hypothetical grammar with4096 effects/versions: (298838437306720,8520045711850852781).

These finite CONTROL sums do not establish the sufficiency or feasibility of the complete eleven-coordinate program or actual operating configuration. The complete source hold still needs all other disjoint rows, native/elapsed work, memory/backing/lifetime and window qualification, source factory, failed/final/recovery/page behavior and actual native CI.

Seven added UNIT roots check independent phase caps, shared-item partial/empty tiers, exhaustive small independent allocations, literal B/Cs/Cf/A rows, explicit small original-stage future sums and a large literal, actual existing split-ancestry grammar, changed selections/ordinal/endpoints and late overflow/refusal. The new manifest test inputs are explicit inert DATA selected before any H0; no running budget/clock/window is enlarged. No malicious responder, bypass or attack reproduction is made. These tests remain NOT_RUN and individual coverage UNMEASURED. Source review and standalone formatting are not compile/test results.

This component has no production caller. Post-ABI/window/content/effects semantics and other K/raw/file/storage/capture/failure/internal-incoming/deferred/recovery/final work remain separate. The complete all-eleven sufficiency AND feasibility/nativeRAM/resources/window/factory gate remains open. The original one-complete-connected-publication/new-native-CI/actual-artifact gate and5-2 through5-6 order remain; fixed1-4 complete/5 in progress/6-10 pending and whole5/7/8 incomplete are unchanged.


### Original post-ABI and content-window CONTROL assignment

The shared-item RPC placement above has an actual independent SOURCE FINAL and root closure. Its 675-file snapshot and full isolated checkout preparation remain execution-unverified. The next component assigns only original semantic work after RPC ABI decoding and at CONTENT-window begin/end; it does not reprice ancestry opening as a semantic window.

`provider_history_semantic_boundary_bound.go` keeps normal entered work N separate from the complete caller/preflight envelope E. Opening includes the two original embedded ABI copies/codecs, the shared confirmed state/index clone and ExpectedWindow/replayRun copy. Both original parse/clone paths have only zero outstanding preflights, so this component's opening E=N. Closing is the original stream/int/window-pointer copy, including its event overhead. Neither measures Go heap/native CPU or source trust.

For one parsed/applied typed state let R and G be the already bounded successful record/grant output lengths, and let SR/SG denote the corresponding existing serializer N/E parts. Present state calls BOTH serializers, compares the record, then copies it: N=SR.N+SG.N+logical(3R+2). E retains BOTH complete serializer envelopes and max(3R+2,65536+R+1). The second term preserves a full original selected record before a failed equality even for a small typed state; record copy occurs only after equality. A record size refusal occurs after encoding and does not skip the grant serializer.

The grants phase first copies the already accepted stopped record, independently serializes BOTH typed outputs again, compares BOTH, charges the same history/replay budget, marks comparing, and selects history. Its N tail is 3R+2G+39; E uses 3R+262144+G+39 because a failed grant result can have the original maximum width before equality. The successful fixed-scope grant output bound does not replace that failure width. Absent state performs two empty comparisons and the original charge/copy/select events:38 logical units, no serializer or invented decoded bytes. `replayBudget.charge` itself only updates a scalar byte budget; it is not a codec/event.

One compared history item includes the original ExpectedItem copy, owned entry/command copies and equalities, shared history charge and compared-int copy. Its conservative N=E logical bound is sizeof(ExpectedItem)+sizeof(int)+23+6*262144. Both source queue text and decoded ABI field caps precede these uses. This is separate from the already priced raw JSON/ABI word decoding. The selected whole-fold V counter and confirmed-version telescoping continue to limit all history items together; window splits add positions rather than fresh32-item pools.

The semantic role model reuses only the accepted finite rank/role arithmetic. With CW content windows, P present-stop cap and H history-group cap, its original assignments are CW content openings, CW content closings, CW state stops, at most P later grants, and at most V shared history items. The state allocation reserves P richer present-state positions and CW-P absent positions. A missing optional present stop can use a richer position because present N dominates absent N in every coordinate. This dominance is checked explicitly. The original present-stop capacity check follows BOTH serializations and record equality: one extra present state may enter before that cap refuses, including P=0. When an absent slot remains, delta therefore retains complete present-state E minus absent proxy N, including decoded expiry bytes. A successful-count cap cannot erase this first failed phase. When every state slot is already richer, its present proxy is retained. History uses full32/partial/empty marginal positions; all other G positions are zero for THIS component only. Delta comes from the complete enclosing original phase E-N. The original per-role first-refusal propagation, separately priced successful/failed closure MAX and nonfinal own advance remain mandatory.

There is intentional overlap with older broad DATA quotes: `providerHistoryCollectionRepeatedStateUpper` contains stop serializers, `providerHistoryEventQueueUpper` contains begin/finish/stop/history-comparison events, and `providerHistoryABIUpper` contains fixed window ABI initialization. The eventual complete compiler must select the disjoint new assignments IN PLACE OF those overlapping old charges, while retaining their other events/commands/queue retention. Adding every old total and every new component wholesale is not a sufficient feasible derivation. No factory/admission/profile is connected or altered here.

Eight safe UNIT roots inspect source-call deltas and independent phase allocations: fixed present/failure widths; actual begin/absent/finish; ordinary parsed states; continued grant serialization after record size refusal; actual history shared-budget charge; content-only phase counts; independent ranked/future role allocation; and invalid whole-zero numeric DATA. No host, network, paid operation, exploit or runtime-policy change is involved. They are authored, not executed:578 total new roots remain NOT_RUN and291 individual90% targets UNMEASURED until new native CI. All-eleven sufficiency AND feasibility, native CPU/peak lifetime backing RAM, resource/window/final/recovery/factory qualification and ONE complete connected publication remain open. Original5-2..5-6 order and whole5/7/8 incompleteness are unchanged.


### JSON refusal envelopes for block, gas and execution callers

The source-closed semantic-window assignment remains separate from the next block/gas/execution composition. `providerHistoryJSONControlUpper` now supplies N/E parts for the original JSON atoms and fourteen fixed field schemas: the original seven block/transaction/access/authorization/withdrawal/receipt/log schemas, gas row, execution and regular gas dimensions, gas classification, root context, call frame and trace log. The first seven successful N rows are byte-for-byte compatible with their previous formulas. No runtime parser, accepted field, limit or profile changes.

For raw ceiling C, a field schema has A distinct allowed names, S total name bytes and R required names. Successful membership uses the existing 2AS+A*A constant; mandatory values add6R and at most2C. A first unknown decoded key may have C bytes and may be visited after all preceding known keys. Its extra membership cost is at most A*C+S+A. The new E retains that complete prefix rather than assuming every failed key already has an allowed fixed name. Source UTF8 and surrogate validation precede these original routes, so a decoded key/string does not expand beyond its raw spelling. Standalone arbitrary invalid-UTF8 DATA does not inherit that source guarantee.

An RPC quantity's ParseUint64 range rejection follows its lowercase scan and input-sized codec. Successful N remains decoded16/logical566; E retains decoded max(16,C-2) and logical max(566,(C-2)^2+C+2). The existing unconsumed encoder16 preflight is preserved. An index first attempts string JSON and may continue to decimal on a non-budget parse failure for raw<=20. Its E retains both entered JSON codecs, including40 decoded bytes for a20-byte failed-string fallback; a failed observer is never renewed. Signed diagnostics retain their late int64 range decision and optional minus prefix after original string/quantity work.

A complete seven-call prefix of `nodeRootContextObserved` is priced jointly. Four string values plus hex input come from FIVE DISTINCT original fields: from/type/gas/input/value. Their raw lengths sum<=C; the input and gas inner-codec lengths share the same pool. All seven calls occur before the combined boolean guard, so input/value still execute after a quantity parse refusal. Normal prefix costs decoded2C+80/logical2C*C+C+4739. E keeps the same decoded upper and logical2C*C+2C+4739. The proof uses sum(raw_i^2)<=C^2 and input_inner^2+gas_inner^2<=C^2, with original quantity/value scalar bounds. It does not apply a first-false shortcut to an individual atom or duplicate a full C pool for every sibling. Later sender/signature, transaction context, destination, optional diagnostics and the rest of whole block/gas/effects phases are NOT included in this prefix.

This new prefix overlaps the initial normalization work already inside older broad successful gas/effect DATA quotes. Event-site selection must replace matching sites rather than add every old aggregate and every new part wholesale. Whole caller multiplicities, failed-phase closures and stage/future placement remain required, followed by all-eleven sufficiency AND feasibility, native CPU/backing peak RAM, resource/window/final/factory qualification and ONE complete connected publication/native CI.

Eight further safe UNIT roots cover fixed-schema normal/E identities; the first long unknown name; input codec before uint64 rejection; the two-codec index fallback; signed diagnostic range; actual continued root-prefix calls after early false results; bounded common-atom refusals; and whole-zero invalid domains. They use fixed in-memory JSON and preselected inert observers only, without network/host/attack/authority operations.586 cumulative authored roots remain NOT_RUN,291 per-file90% targets remain UNMEASURED, and existing published green CI is not evidence for this unpublished source. The original5-2..5-6 order and whole5/7/8 incompleteness remain unchanged.


### Keep trace normalization inside the original semantic window

A ready block previously normalized all selected transaction traces with a fresh per-trace frame/log budget before the semantic replay checked their aggregate window capacity. The existing effect-cost helper explicitly describes successful work only. Its successful aggregate frame/log maxima therefore cannot silently serve as the refusing normalization envelope.

The private decoder now accepts an explicit normalization budget. The public `DecodeCallTracer` API and its ordinary observed counterpart still create a fresh budget for each standalone trace. The closed collection reader instead creates one temporary normalization budget for each complete ready block, seeded from the frame and trace-log counts already consumed by its own selected semantic window, with the same original replay limits and observer. Every selected trace for that block uses this one temporary budget. A block waiting for additional original response groups is not normalized, so a response-group boundary cannot renew capacity. After the complete block is semantically replayed, its semantic frame/log counters must equal the temporary normalization counters before transient data is released.

This does not spend the execution budget during parsing, persist a second quota, alter a manifest limit, reset an original meter, or grant authority to a private budget pointer. The seed is derived anew from the original semantic window after each successfully replayed block. A refused candidate is discarded by the existing snapshot transition, retaining its predecessor cursor and the original observer's entered work. Only the existing close/open window transition creates a new semantic window budget. The public standalone decoder remains independently bounded per trace.

For every previously successful whole window, semantic replay visits every normalized frame and every trace log, including reverted, static and uncommitted subtrees. Their aggregate counts already fit the selected original limits. At any normalization prefix, the seed plus newly entered counts cannot exceed that successful window's total. The earlier aggregate guard therefore preserves previously successful whole-window input and output semantics; it can refuse an already-failing window before constructing more trees. The smaller remaining array capacity may reduce successful allocation-copy events, while the pre-existing successful upper remains conservative. No new observation events, struct fields, persisted cursor representation, source window, limit, or authority are introduced here. Complete native CPU, backing and lifetime-memory accounting remains unqualified.

This change does not erase the entered raw validation prefix of a failed root. Unicode/JSON/duplicate preflight still runs before its root frame-capacity check. Frame or log array-capacity refusal can retain entered parent work and raw array decoding. An eventual complete phase E must fund these prefixes, continued siblings and original caller multiplicities; the successful N alone remains insufficient. Raw diagnostic/whitespace bytes retain their independent per-result pool and are not charged against an invented shared semantic payload cap.

The ordinary 40-root collection fixture has semantic counters of 80 frames and 40 trace logs. Its cumulative original observer instead records 160 frame events and 160 log events: acquisition receipt decoding, normalization, semantic receipt decoding and semantic frame traversal each observe the 40 logs at their own real source site. Window counters and whole-observer actuals are distinct; a closed window does not reset the latter.

Ten offline unit scenarios cover standalone behavior, shared frame/log exhaustion, the complete legacy walk including reverted/static nodes, invalid initial budgets, the same failed observer, both readers' complete two-block/40-root path across two response groups, exact failed prefixes with predecessor preservation, and renewal only after a closed semantic window. Fixture limits are selected before constructing the manifest, sealed state, fold or cursor. No live quota, host, clock, RPC, account, transaction, operating policy or Source credential is patched. All ten scenarios are authored but NOT_RUN; cumulative authored tests are 596 ALL_NOT_RUN and the 291 per-file coverage targets remain UNMEASURED. Full enclosing cost composition, the production factory, ONE connected publication/native CI and original steps 5-2 through 5-6 remain open.


### Fund the complete refusing trace-normalization prefix

A separate private numeric envelope now covers one original semantic-window trace-normalization phase, including a terminal refusal. It is not a Source derivation, an enrolled RPC, a production caller or operating permission. The compiler must derive its parameters from complete original result spans and the same selected-window normalization route; arbitrary private maps and fresh per-trace budgets do not supply those premises. This row replaces an assigned whole normalization scope rather than being added to an overlapping successful aggregate.

Let C be the selected per-result maximum, R the sum of complete original raw result lengths, q the normalization attempts, F/L the original window frame/trace-log maxima, and w the Go word size in bytes. C is between4 and8MiB; F/L are each1..65536. At most F roots can enter their frames and one terminal root may still enter raw validation before the exhausted-frame check, so q<=F+1. Each counted attempt has a nonempty original raw span, giving q<=R<=qC. No attempts requires zero raw bytes and produces zero work. Invalid input or arithmetic overflow returns a whole zero result with false; this is refusal of numeric DATA, not a weaker selected limit.

Every visited frame and reserved log has a distinct nonempty original JSON value in the strict original calls/log tree. Thus f=min(F,R) and l=min(L,R) safely cap their counts without borrowing a semantic decoded-payload allowance. Frame depth0..32 gives total frame raw lengths<=33R and their squared lengths<=33CR. Nonrecursive log spans are disjoint across the original tree, giving total log lengths<=R and squares<=CR. Diagnostic and whitespace bytes remain included in R.

The source-derived per-frame prefix is decoded<=4n+64 and logical<=4n^2+(3w+24)n+13061 before separately assigned child/log storage. It includes the fixed allowed-field failure row, all fifteen possible scalar JSON spans, gas/hex inner codecs, signed state scan, bounded value decoding and raw call/log array header allocation/copies. A per-log prefix has decoded<=4n+256 and logical<=4n^2+9n+18166+24w, including both index forms and their failed-string decimal fallback, the four-topic cap and fixed topic conversions. Original quantity encode16, decimal encode20 and value64 incoming preflights are retained in these bounds; they are not charged a second time as actual events.

A refusing tree may allocate siblings it never visits. Declared child counts therefore cannot be replaced by f minus the root count. The raw array spans instead bound their sum by33R/2. Child output initialization costs at most19w*(33R/2)<=314wR; completed child copies cost at most(19w+1)f. Log initialization/copies cost at most(18w+1)l because every declared own log is reserved in the shared trace-log counter before descent. Initialization event constants are already in the frame row. This is a logical copy bound, not a heap/backing or peak-live-memory qualification.

Combining the original raw validation, frame/log pools and separately assigned storage gives the numeric refusing envelope:

- decoded:138R+64f+256l;
- frame observations:f;
- log observations:l;
- logical:(138C+413w+803)R+(13062+19w)f+(18167+42w)l+4q.

This row owns only normalization observations. Both canonical receipt decoder sites, semantic frame traversal, outer retained-string copies, transaction/native work and original reader/role repetitions belong to separate disjoint rows. In particular it does not replace the four-site cumulative log accounting of the complete collection fixture. All-eleven sufficiency AND feasibility, native CPU/backing/peak/lifetime RAM, source resources/windows, failure/recovery/final/page and the production factory remain open. Extreme permitted scalar combinations can overflow uint64 and correctly refuse; the selected whole program still needs sufficient feasible bounds.

Six additional safe offline unit roots cover portable literal rows, invalid/overflow/zero domains, original shared frame/log refusals with the last raw preflight retained, a fixed oversized quantity and unknown-field prefix, and a small tree whose declared sibling storage exceeds its visited child count. They construct no network process, host bypass, transaction or Source authority. The six roots are authored but NOT_RUN: cumulative602 ALL_NOT_RUN,1384 unique TestProvider roots and292 individual coverage targets UNMEASURED. The workflow adds only the new product filename to the unchanged individual90% coverage contract. ONE connected publication/native CI and original5-2..5-6 remain pending.


### Complete original root-context refusal envelope

`providerHistoryRootContextRefusalUpper(C,H)` is conditional numeric DATA for
ONE original `nodeRootContextObserved` invocation. C is the complete strict
original root-object byte ceiling in4..8MiB. Its ten scalar RawMessage spans
are disjoint. H is the original native-input selector's transaction ceiling,
not a caller credential: the admitted collection route selects capture+1024,
while inert DATA and other original roles retain64MiB. M=H+256 is derived.

The scalar JSON/inner pools contribute decoded2C+64 and logical2C²+2C+5320.
Signing with original preimage length r<=M-1 costs at most M²+4, including the
unconsumed M²+1 encoding preflight. No actual event occurs between its repeated
preflights. Sender fixed work is13788; its continued address conversion1765
runs even after an ordinary sender error. CREATE retains destination string,
creation and address conversion before the combined guard; the creation
maximum4183 and destination address1765 remain in the refusal prefix. Together
with the three binding comparisons the complete row is decoded2C+161 and
logical2C²+5C+M²+H+26912. The accepted domain's maximum4644371584608544 is below
2^53; invalid inputs return the entire zero vector with false.

This is one assigned root-context envelope, not an add-on to an overlapping
older seven-call prefix or successful aggregate. The outer object/dimensions,
execution header/receipt binding, role counts, native CPU, tx accessor/Cmp
allocations, backing/peak/lifetime RAM, factory and operating authority remain
separate. Six finite UNIT roots use existing public V0/R1/S1 DATA and the real
wide inert native preflight; no owner is fabricated or reduced after admission.
The destination test compares bad JSON with a short wrong string: their exact
53 logical/3 decoded difference requires the later native siblings on BOTH
routes. Whole source tests are608 NOT_RUN and293 individual90% targets remain
UNMEASURED until ONE complete connected publication and new native CI.


### Complete original root-gas normalization refusal envelope

`providerHistoryRootGasNormalizationRefusalUpper(C,H,T)` is conditional numeric
DATA for ONE whole original `nodeRootGas` call. C is the complete original raw
trace ceiling4..8MiB, H is the original native-input selector's transaction
ceiling and M=H+256. T is the positive original temporary per-block transaction
cap1..16384, not actual completed rows or a later semantic-window allowance.
Strict Unicode/duplicate/raw-size and complete original decoded transaction/
receipt shape/non-nil premises remain with the original caller. No production
consumer, owner, Source or factory is supplied by these three numbers.

For original length n, declared raw slots s<=min(T,(n-1)/2) can exceed actual
rows t<=s. Two raw codecs plus header/result initialization cost decoded2n and
logical2n²+(9w+1)s+4. Per witness row u, the txHash/result original spans are
disjoint; fixed witness/root schemas, both continued native hash/text calls,
FULL root context, all four dimensions and result append cost decoded6u+161
and logical6u²+29u+H²+M²+2H+S+37690+3w. Here S is the original Transaction and
BlobTx struct-copy logical metric with both event constants; it is not the
referenced payload graph's memory cost. Final encoding/preflight contributes
at most n²+1, without charging an unconsumed preflight again as actual.
The whole envelope is decoded8C+161s and logical9C²+29C+(9w+1)s+
s*(H²+M²+2H+S+37690+3w)+5. Every growing sum/product is checked; late overflow
returns the whole zero vector, not a partial quote or a relaxed limit.

A two/three-byte parsed root object can only be empty. Its seven scalar calls
record four compare1 events and one codec0 event: decoded0/logical5, then
return before native access. This fits the retained root constant. Successful
final output uses the same validated dimension RawMessages/fixed names and a
hash equal to the original string; canonicalizing escapes can shorten it.
The original final encoded-size guard remains mandatory.

Empty DATA normalization does not mean zero cost. With original cap1, raw[]
records decoded4/logical17; raw ` [ ] ` records decoded10/logical59+3w because
one unused RawMessage slot was allocated. The original cap must not be inferred
from zero completed transactions. The collection's empty-block branch bypasses
nodeRootGas; these empty tests cover the existing private DATA API only.
Seven finite UNIT roots cover portable literal/overflow rows, empty allocation,
selected regular/execution dimensions and escaped hashes, fourteen original
refusal shapes, work retained before a later row refusal, and no renewal of a
failed observer. Repeated public V0/R1/S1 DATA does not authenticate a canonical
block, gas settlement, RPC, deployment or caller. Inert tests retain64MiB native
ceilings. Tests615 NOT_RUN and294 individual90% targets remain UNMEASURED.

This E replaces its assigned normalization scope; it is not added to older
root-context/normalization aggregates owning the same work. BOTH strict gas
checks, acquisition/retention/semantic execution, original failed whole-phase
counts, matchedN/E/role composition, native CPU/accessor/backing/peak/lifetime
RAM, resources/windows/final/factory, ONE publication/native CI and5-2..5-6
remain separate and OPEN. In particular K per-block temporary budgets may be
entered before a later semantic-window refusal; a successful windowT is not
a bound for all such prior entered work. A conservative overflowing quote does
not by itself prove actual execution or every implementation infeasible.


The root-gas normalization R1 source review accepted the whole conditional
logical envelope without mandatory findings. Its preventive S1 advice is
applied by expressing the same fourteen test modes with a tagged switch;
scenario data, limits, expected values, calls and product code are unchanged.
This is source-style cleanup, not a claim that QF1003 was enabled, lint ran or
CI failed. Tests615 NOT_RUN and294 targets UNMEASURED remain unchanged.


### Matched whole root-gas work and original incoming room

An alternative numeric pair covers the same complete original root-gas normalizer.
Actual native encodings at distinct original block indexes share a source-derived
canonical byte pool Q and per-transaction ceiling V. Canonical block encoding,
standalone legacy/typed transaction projection, the temporary per-block replay
evidence charge and the original native owner must establish these premises.
The arithmetic helper itself authenticates none of them. The raw trace ceiling C
and original positive transaction cap remain separate; empty arrays do not reset
the original cap to zero.

The normal actual-prefix bound is decoded8C+161s and logical
9C²+29C+(9w+1)s+5+s(S+37691+3w)+VQ+2Q+(V+258)(Q+256s),
where s=min(T,(C-1)/2), w is the Go word size and S retains both logical
sidecar structure-copy events. The corresponding refusal envelope adds exactly
one widest unconsumed original native encoding preflight (H+256)²+1.
Both unchanged native preflights still run. This pair replaces the overlapping
whole-normalizer assignment; it does not add to the earlier loose envelope or
record unconsumed preflights as actual work. Native CPU, accessor allocations and
backing/peak/lifetime RAM are outside this logical metric.

Eight finite offline unit roots were authored for portable independent rows,
invalid data, complete canonical block/index projection, real observed native
encodings, unchanged successful normalizers, empty raw-array slots, later-row
ordinary refusal and the unchanged native-preflight refusal with failed-observer
non-renewal. Their public fixture values are not an authenticated Source, valid
settlement, accepted block or operating permission. These tests are NOT_RUN;
individual-file coverage remains UNMEASURED. Full source-derived phase/role
composition, all-eleven sufficiency AND feasibility, native resource qualification,
production factory and one complete connected publication with new native CI
remain open.


### One complete strict gas check with original native room

The next candidate pair describes one original replayCheckGas invocation. It
retains canonical header bytes<=65536 and the original selected66-byte block hash,
the explicit selected profile, bounded raw trace C, positive temporary transaction
cap T and the original canonical transaction Q/V pool. The helper authenticates
none of these premises and has no production consumer. Both original strict
passes and their caller/role/future reservation composition remain separate.

With s=min(T,(C-1)/2), original value-copy ceiling A=roundUp(H,w)+sizeof(big.Int),
and the same logical sidecar S, normal actual-prefix work is decoded6C+97s and
logicalB²+B+4497+6C²+16C+(6w+1)s+s(S+A+21078)+VQ+2Q+(V+258)(Q+256s),
where B=65536. The corresponding envelope adds one original (H+256)²+1
unconsumed native encoding preflight. The original value-copy ceiling is kept
without a new magnitude-pool assumption; its preflight and all gas settlement
checks are unchanged. Native CPU/accessor allocation and backing/peak/lifetime
RAM are not certified by these logical event weights.

Eight safe offline unit roots were authored for portable rows, invalid bounds,
all four selected profiles with one/two rows, empty raw-array slots, equal-sized
early/later quantity refusal, two original passes on one cumulative observer,
unchanged native-preflight refusal with no renewal and original early guards.
Private public transaction/header/receipt values are not authenticated source
evidence, valid settlement, consensus-valid blocks or dispatch authority.
Tests remain NOT_RUN and per-file coverage UNMEASURED; source review, complete
phase/role/all-eleven feasibility, native resources, factory and one complete
connected publication with new native CI remain mandatory.


### One complete transaction-bound execution replay

The next private numeric pair covers the whole original replayExecutionObserved:
its four eager hash/text calls, fixed hash bindings, original protected-type gate,
fresh sender recovery, root/value/input/recipient checks, every entered frame,
log construction and output growth, and the final ordered receipt-log join.
Original successfully normalized tree shapes and source byte pools are mandatory
premises. A private typed object or a positive numeric result authenticates none
of them, and this helper has no production consumer.

With original header B=65536, trace C, canonical transaction ceiling V, native
owner H/signing M=H+256, retained value-copy ceiling A, original frame/log caps
F/L and temporary canonical receipt-data bound Qr, actual-prefix N is decoded
161+64F+40L, frames F, logs L and logical
B²+B+V²+2V+S+(V+258)(V+256)+34899+2A+3C+Qr+16R+4433F+(Jlog+2315+7R)L.
S retains the two logical sidecar structure copies, R is sizeof(replayLog),
and Jlog is sizeof(types.Log). Complete output growth is bounded by16+7L
logical copied row slots and3L event constants. A first receipt mismatch retains
its actual comparison bytes as well as the entire earlier frame walk.
The corresponding envelope adds exactly one unconsumed original M²+1 native
encoding preflight. All existing checks, temporary/shared quotas and preflights
remain unchanged; overlapping normalization/selected-effect rows must not be
added again. Native CPU/accessor/backing/peak/lifetime RAM is a separate domain.

Ten safe offline unit roots were authored for independent portable rows,
invalid bound data, CALL/CREATE and committed/reverted/static child joins, full
log-buffer growth, early/later receipt mismatches, root/child semantic refusals,
unchanged shared quotas and the original native-preflight refusal without renewal.
The finite public signature-decoder values and private header/receipt/JSON graphs
are not consensus-valid blocks, valid gas settlements, authenticated providers
or operating authority. All tests remain NOT_RUN, individual-file coverage
UNMEASURED. Whole phase/role composition, all-eleven sufficiency AND feasibility,
native resources, factory and one complete connected publication/new native CI
remain open.

The final receipt join also compares every returned topic after its address and
data checks. Each 32-byte pair costs65 logical units; at most four topics add260
per log to the earlier fixed assignment, giving2315. This is separate from the
two topic copies during construction. Corrected portable rows include that room
in both N and E without changing decoded/frame/log counts or native preflights.
The first/last data-mismatch oracle with one topic and two data bytes retains111
units for each preceding successful receipt row:41 address+5 data+65 topic.
Added finite topic-width cases0..4 verify129 per added topic per log, including
64 construction-copy bytes and65 final comparison units. First/last last-topic
mismatches retain46+65n per preceding complete row for n topics. These are authored
UNIT expectations, not executed tests or a reported current CI failure. The
independent earlier missing assignment does not by itself prove the padded total
N was numerically insufficient for a reachable execution. Original runtime,
source provenance, quotas, incoming native room and publication gates are kept.


### Complete original content decoding, including both indexed commitments

A separate private numeric pair now covers ONE whole replayDecodeContent call,
including all original canonical header/transaction/receipt codec and equality
rows, both indexed transaction/receipt commitments and their two header joins.
Q is the original per-block replay evidence cap, V bounds each original raw
transaction/receipt before native parsing, and T/L are the original whole-window
transaction/log caps. The original header guard supplies65536 independently.
Without a tighter original source proof V must be Q; private numeric DATA cannot
authenticate the source or renew any remaining quota.

Let U=min(Q,max(65536,V)). The full original canonical primitive prefixes cost
at most2UQ+2Q+6T+3. Both ordered tries share the disjoint tx/receipt Q pool and
add UQ+34Q+2*1705896*T+8; their two32-byte comparisons add130. Thus the proposed
N and E are both decoded Q, blocks1, transactions T, logs L and logical
3UQ+36Q+(2*1705896+6)T+141. Each incoming encode/hash/copy here is already a
pending event in those rows; there is no additional deferred native signing
request in this function. This conditional one-function equality must be
independently checked and does not collapse enclosing stage/role reserves.

Seven safe UNIT roots cover independent literal/domain rows, three complete
original private fixtures, final receipt-root mismatch, malformed receipt after
header/transaction work, canonical receipt-type mismatch, and exhausted shared
original evidence/count budgets. Malformed and mismatching inputs are finite
in-memory DATA, without a node, RPC, VM or broadcast. All tests remain NOT_RUN
and each product-file90% coverage remains UNMEASURED. Native parsing/CPU/backing,
peak/lifetime RAM, gas, acquisition, execution, whole caller/role multiplicity,
all-eleven sufficiency AND feasibility, factory and the complete connected
publication/new native CI remain separate requirements. Whole5/7/8 is incomplete.


### Preserve the original unfinished content frontier in decoder pools

The original acquisition and semantic decoder costs now have a checked private
one-journey pool. W is the original content-window bound, B the selected block
bound, K the original contentBlocks limit1..10, and E/T/L the original replay
caps. V bounds raw transaction/receipt bytes before decoding, otherwise V=E;
U=min(E,max(65536,V)). A numeric tuple supplies no source or live authority.

Every previously cleared acquisition batch has passed semantic admission of
those same raw bytes, declared transactions and receipt logs. Their cumulative
pools fit WE/WT/WL. One current retained or locally entered batch may precede
that admission and needs KE/KT/KL separately, including normal pending S cuts
and the first local failure. Partly applied blocks may be counted in both pools
conservatively. A new batch is prohibited until the previous batch clears;
windows cannot close while that frontier remains. There is one +K exception,
not WK and not a renewed allowance per successful step. Failed original ledger
branches terminate rather than retrying the same failed transition for free.

Combined acquisition and semantic pools use factor2W+K and at most2B decoder
calls. The pair N=E is decoded(2W+K)E, blocks2B, transactions(2W+K)T,
receipt logs(2W+K)L and logical
3U(2W+K)E+36(2W+K)E+(2*1705896+6)(2W+K)T+282B.
This recombines the independently closed whole-decoder event families, not a
call of the one-block helper with an aggregate cap beyond its64MiB domain.
All growing arithmetic is checked; overflow erases the entire returned pair.
It replaces overlapping decoder/core/two-pass content-trie rows. Extra JSON
transaction roots, withdrawals, outer acquisition, gas, execution, copies and
other original roles remain separately assigned.

Seven safe UNIT roots cover independent complete literals/domains, a valid
late logical overflow, both original decoder passes with independent temporary
and shared semantic budgets, pending acquisition before failed semantic byte
admission, the actual private cursor's complete/partly applied pending guards,
and separate transaction/log exhaustion with sufficient byte room. A reordered
canonical pair isolates decoder DATA only and is not a valid chain journey.
The actual cursor fixture checks original phase guards separately; its outer
work is not assigned to this decoder-only polynomial. All tests are NOT_RUN,
individual90% coverage UNMEASURED. Independent reader reconstructions need their
own source-weighted journeys. Complete all-eleven/resource/native/RAM/factory
sufficiency AND feasibility, full admitted-stage sums, one connected publication
and new native CI remain open. Whole5/7/8 is incomplete.


### Original gas preparation and both strict passes in one journey

A checked private component now recombines complete original gas normalization,
first-row classification, initial trace/profile/descriptor assembly and BOTH
strict gas passes. Its new source proof is pending independent review; no
production collector consumes this numerical DATA. W/B/K/E/T/C/H/V/N remain
original preselected content windows, blocks, frontier cap, evidence and
transaction caps, capture-body ceiling, native owner bound, canonical value
ceiling and native-block cap. The immutable original source must establish
each relationship before use. A finite quote supplies no owner or authority.

Previously cleared batches share their original semantic byte/transaction
charges; one pending or first locally failing batch contributes K temporary
caps. Strict trace and canonical transaction pools use (2W+K)E; root acquisition
canonical bytes use (W+K)E. Whole original block encodings also cap canonical
transaction bytes at N*min(C,E), with twice that for both strict passes.
Individual raw canonical values are bound before native interpretation.

Root normalization may discard large original output or diagnostic fields.
Keep its complete NC raw pool, separately from the normalized strict trace
pool min(2NC,(2W+K)E). Direct classification precedes strict byte admission,
so it keeps min(NC,WE+KC), including a C-sized first failing pending input.
Never use the later E debit to erase earlier observed parsing or copying.

Declared RawMessage array slots differ from matched native transaction rows.
Even empty padded JSON arrays allocate headers. Their pools are bounded by
raw lengths and original temporary T, independently of WT matched native rows.
Only loops AFTER the exact array/native count check use pooled canonical
transaction counts. The complete direct classifier includes unknown-key
refusal: 3n^2+9n+740+(6w+1)s, instead of the successful-only663 constant.
The longest returned profile in the selected family funds ordinary profile
mismatch too. No alternate profile is tried after refusal.

The combined actual-prefix row replaces these overlapping gas components.
One current native (H+256)^2+1 encoding preflight is retained above that row;
original guards are unchanged. This component theorem neither refunds held
stage uppers nor replaces initializer plus SUM FULL admitted stages with a
global pad. With N=0 the original pre-fork scalar path observes no gas events,
so the whole numerical pair is zero without granting permission to any path.

Ten safe UNIT roots are authored for portable independent complete literals,
domain and late overflow refusal, child-row recomposition, actual private root
and direct preparation plus both strict invocations, original wide incoming
refusal, padded empty arrays with exact spare-header counts, unknown keys with
exact prefix observations and the zero-native pre-fork path. Invocation tests
are private offline DATA, not authentic admitted cursors or acquired RPC
responses. All are NOT_RUN; individual90% coverage remains UNMEASURED.
Retention/reconstruction trace copies, block projection/decoding, effects and
execution, other readers/roles, full all-eleven/native CPU/backing/peak/lifetime
RAM/resources/windows/failure/final/factory sufficiency AND feasibility remain
separate. One complete connected publication and new native CI are still
required; original5-2..5-6 order and whole5/7/8 incomplete remain unchanged.


### Original complete trace normalization and semantic execution journey

One checked private component now composes whole raw call-trace normalization,
transaction/receipt-bound semantic execution, ready-block ReplayExecution array
initialization and preceding raw-result copies. This FIRST complete journey
source join is pending independent review. No production consumer or source
credential is supplied. Original W/B/K/E/C/H/V/T/F/L/X must be bound before use;
X is the original global Registry effect-trace cap and may be zero.

Each ready block seeds normalization frames/logs from its original semantic
window and carries those counts across all roots. Only matching semantic
success permits later progress. Thus cumulative normalization frame/log use
fits WF/WL; at most one final raw attempt can precede a frame-cap refusal.
Attempts fit min(X,WF+1). Internal observed raw inputs fit min(C,E) per attempt,
while their preceding copies still need C before the E length guard. Diagnostic
and whitespace bytes can disappear and do not borrow later semantic WE.

Declared execution-array slots are initialized before the first normalization
call. Their pool min(X,(W+K)T,BT) can exceed visited roots/frames after an early
failure. All such slots and each nonempty array initialization event remain
funded. A first failed raw parse does not erase their original copy work.

Only semantic execution uses original upfront canonical WE/WT debits. Its
unique needed receipt indexes enter at most min(X,WT,WF) roots. Complete
canonical transactions share min(WE,roots*V,B*min(C,E)); canonical receipt bytes
share min(W,roots)*E. Normalization and semantic traversal retain separate
frame/log observations. The complete execution row retains all four final
topic comparisons, all output growth, header work and original value-copy H.
Native polynomial pooling retains (2V+516)Q+66048*roots rather than multiplying
a maximum-sized canonical value for each root. Original native guards remain.

The normal upper preserves the complete normalization refusal envelope and
semantic actual-prefix row plus disjoint outer copies. The envelope retains
one current native signing encoding preflight when execution may enter. This
component does not refund initializer plus SUM FULL original admitted stage
uppers or grant one global allowance. Zero X produces the entire zero pair.

Ten safe UNIT roots are authored. They include independent portable complete
vectors and invalid/overflow/zero/child identities; actual private complete
normalization and CALL/CREATE/child receipt execution; unchanged wide incoming
refusal; exact raw-copy-before-E and final shared-window refusal prefixes;
original private cursor allocation of two declared roots before its first
ordinary parser error; and a final fourth-topic mismatch after all17 logs.
The cursor test uses its broad original inert allowance and prices its outer
reconstruction independently; a trace-only quote is not applied to that whole
transition. Other invocation tests are private offline DATA, not an admitted
cursor, acquired RPC response, consensus block or authoritative Source.
All are NOT_RUN and individual90% coverage remains UNMEASURED. All-eleven
source-weighted role/stage sufficiency AND feasibility, nativeCPU/backing/peak/
lifetimeRAM/resources/windows/failure/recovery/final/factory, ONE connected
publication and new native CI remain open. Original5-2..5-6 order and whole
5/7/8 incomplete remain unchanged.


### Complete original block and receipt projection

The private whole-journey numeric row combines both JSON and debug conversion, common binary processing, immutable retention and reconstruction, and the existing content decoder once. Successful acquisition uses the original W+K pending bound; one complete final-call allowance covers refusing prefixes without authorizing another call or retry. Early native transaction work retains the one unclosed raw row, a failing typed JSON binding keeps its original output ceiling, and eager quantities and unknown field names remain funded. Debug withdrawal encodings retain their earlier 65536-item slice boundary before the later 16384-item trie guard.

Original observers, native selectors, guards and irreversible full stage reservations are unchanged. Nine safe UNIT roots are authored but have not run. Independent design acceptance does not approve this implementation. Whole all-coordinate feasibility, native CPU and backing/peak/lifetime memory, remaining effect/control work, resources and factory qualification stay open. Publication remains one complete connected change followed by fresh native CI; these numeric parts do not grant source or operating authority.


### Effect selection and ready-block control

The private numeric row assigns the original two receipt scans, repeated selected-effect array copies, canonical transaction hashes, pending block allocation and release, trace string headers, and complete effect-list scans while waiting for ready blocks. It keeps the original fixed 32-result group width. Pending allocation counts every declared slot before the first decoder error; zero selected effects still retain empty-array and block-control events. Canonical selected bytes use bounds established before hashing, while the original wide native preflight remains unchanged.

Seven safe UNIT roots are authored for independent portable components, whole-zero invalid and overflow refusal, empty effects, exact canonical selection and capacity refusal, a blocked ready entry, and declared pending allocation before an ordinary decoding error. Private fixture invocations create no source, chain, dispatch or operating authority. These tests have not run and individual 90% coverage remains unmeasured. Canonical decoding, gas, trace execution, retained raw bytes, spine lookup and cursor copying remain separately assigned work. Whole composition, native CPU and physical memory, resources, feasibility and factory qualification remain open before one complete connected publication and fresh native CI.


### Disjoint mechanical content composition

The private content row joins the separately qualified original block projection, gas processing, trace normalization and execution, and effect selection and ready control for the same monotone journey. Canonical content decoding occurs once inside the projection assignment; both strict gas passes stay inside the gas assignment. Indexed spine lookup and the successful semantic block's final fixed copy are priced separately. Overlapping older aggregate rows are not added again.

The full entered-work bounds are summed coordinate by coordinate. Their largest outstanding incoming preflight is retained for the same serial original meter. This component calculation changes no full admitted stage reservation, initializer, remaining guard or role allowance. Seven safe UNIT roots include independent portable complete vectors, original wide incoming ceilings, invalid and whole-zero overflow refusal, and the initial private content prefix before any semantic event callback. A scalar case checks that individually finite children cannot leak partial data when their parent sum overflows. No large workload is created.

Command, claim, state and queue callbacks, window and history boundaries, cursor copies and commitments, outer readers and physical resources are separate. A later successful effect transition cannot use this mechanical row alone. Native CPU and backing, peak and lifetime memory, complete source composition, feasibility and factory qualification remain open. These tests have not run and per-file coverage remains unmeasured; publication still requires one complete connected change and new native CI.


### Semantic window atoms and a refusing command prefix

The private ABI and command cost constructors expose their original one-window parts separately from whole-program repetition. The original whole-program wrappers retain the same finite domains, counts, complete results and checked refusal behavior. Existing workflow coverage targets already include both product files; no coverage requirement is removed.

The new command refusal row retains the producer graph before the consumer's later 262144-byte acceptance check. Its closed graph includes 2311 nodes, 94431 string bytes and 2560 Number bytes, giving an original generated output ceiling of 595876 bytes. A complete conservative refusing generator, consumer and claim prefix uses that wider source ceiling, the original full calldata ceiling and the unchanged current encode preflight. It creates no large payload, callback permission or new allowance.

Seven safe UNIT roots cover independent window and fixed-shape polynomials, unchanged whole-program wrappers and refusal, original ordinary decoder paths and one short invalid service URI. No oversized graph or attack reproducer is constructed. Source-bound callback counts and the refusing prefix require their own independent review; earlier successful-path acceptance does not establish them. Tests have not run and coverage remains unmeasured. Outer event, state and queue work, full source and resource composition, native CPU and physical memory, feasibility and factory qualification remain open before one complete connected publication and fresh native CI.


### First refused ABI callback prefix

The private ABI refusal row prices the entered prefix of one mutation or claim-commit callback under the original embedded schema. Mutation framing runs before native decoding and permits at most 148800 argument bytes; the nonindexed RecordChanged fields occupy at most 512 bytes after their own framing. The row includes the original selector, framing and complete fixed commit alternatives, the canonical codecs/encodes/comparisons and one outstanding encode preflight. It does not count setup, decoded command generation, outer event hashes, selected state/proofs, queues or original role multiplicity. Native CPU and physical memory remain separate qualification gates.

Five additional safe UNIT tests are authored but not executed: independent full scalar rows, an unsupported short selector, a short ordinary method, one rejected duplicate after a valid in-memory claim commit, and short ordinary changed-event framing. They use the unchanged original inert observer and keep entered work; they do not reset a failed meter, send peer requests or create an attack reproducer. The current green published CI does not validate these unpublished bytes. The existing complete connected publication gate and original order remain unchanged.


### Disjoint callbacks through one semantic journey

The private callback journey row combines the qualified ABI, command, outer event and selected state components with callback-only expected-queue retention. Window ABI setup, open/close/stop, later history comparison and the mechanical content journey remain separately assigned. The global selected-state guard bounds all attempts by min(maximumItems, windows times log limit), including the last attempt that changes private working state but aborts before queue retention. A zero selected capacity does not discard other-agent claim/creation indexes.

Queue retention is priced from its original copies and hashes. Later history comparison is excluded. Completed byte charges share the original evidence limits; one first-aborted pre-charge tail receives a separate conservative byte allowance before the independent count cap. A failed append can retain its entered copies, and the hard4096 queue check precedes any further array allocation. One shared pending primitive is retained in addition to the full assigned normal work; the state record encoder uses its producer graph and global walker limit before its later output cap.

Six safe UNIT roots are authored, not executed, including a genuine ordinary callback sequence, another selected subject with zero capacity, and one in-memory digest mismatch after state work but before successful queue retention. These private callback units do not claim acquisition/trace provenance, production Source authority or fold quarantine that they do not execute. Native CPU/RAM, original role/whole-program feasibility and the complete connected publication gate remain open.


### Assigned components at original role boundaries

A private calculation now joins the separately reviewed control and clock, request generation, digest binding, TLS and result context, cursor, RPC boundary, semantic boundary, and mechanical and callback components at their original processing cuts. The same sealed prospective manifest supplies the global limits; method and certificate limits remain explicit. Cursor states use conservative original cardinality bounds. Canonical transaction and value limits are distinct from the padded native integer copy size.

Each independent original reader receives its own complete component envelope. One mechanical and callback journey bounds the content that reader can revisit; it is not an already repeated whole-program aggregate. Before-processing, successful and failed settlement, member copying and subsequent advancement retain their original separate assignments. Full admitted group reservations, initialization, remaining-work checks and the future sum keep their existing rules. The calculation grants no requests, wire, segments or physical write permission.

Six safe UNIT roots are authored for processing-cut separation, zero selected capacity with shared work retained, invalid or unpriceable contracts, independent content-role literals, child and parent overflow, and late coordinate or clock overflow. The inputs are offline prospective data and never install a live owner or executable source witness. These tests have not run; individual 90% coverage remains unmeasured.

The result covers assigned components only. First-original read work, catalog and raw interpretation, file reads and writes, retained context and capture still require their complete source rows. Physical resources, native CPU and backing, peak and lifetime memory, recovery, final processing, feasibility and the production derivation factory remain open. One complete connected publication and fresh native CI are still required.


### Original first-reading and successful file role components

The conditional original-file component binds its first-reading object/raw selections and checks the independently derived first-reading work against the same sealed manifest sublimit. It leaves the constructor's existing two-reader initialization bridge separate from its joined initial row. Later B/S/own-advance readers retain their original nine/three/two complete first-reading and surrounding model-control calls; member preparation freshly reads the recorded raw originals without repeating their parsers.

The same I/B/P/M/C/A cuts now have explicit payload-read bytes/calls, catalog payload and legacy scan-codec work, directory/owner events and writer allocations. Full M includes its entered P. An additional P is still required by the unchanged remaining-work composite. The B and S suffix each conservatively use one complete current-stage noncatalog suffix; this is an explicit constant relaxation, not a whole-history bound repeated per stage. Metadata identity counts exclude only the exact two calls already inside each first-reading row. Write resources split T/C, one member's response/TLS and S; there is no allocation refund.

Four authored UNIT roots check independent ordinary cut literals, the separate existing initializer bridge, split physical/raw writer literals, unchanged local first-reading refusal and whole-zero late overflows. They are NOT_RUN. These are successful structural-path DATA components, not a complete failure/recovery/final bound, native CPU/allocator/backing/RAM certificate, selected-plan feasibility or production factory. Raw copies/hashes/filenames, catalog namespace construction, recorder work outside first-reading and the other remaining original scopes still require separate source composition. No source guard, role, counter, clock, remaining predicate or runtime entry is changed. ONE complete connected publication and fresh native CI remain required after all original scopes and feasibility are complete.


### Retained original replay and typed context role components

The successful retained-context component prices both original dispatch recorders, complete rolling transcript roots and the fresh same-C member hashes/comparisons at their original B/P/M cuts. Recording inside firstReading is already funded by the firstRead component and is excluded from the additional B recording row. Rolling commitments still include every original row, including firstReading. Each fresh P uses four complete transcript-root checks, and full M contains that P. The unchanged remaining-work predicate retains its additional already-entered P.

The original descriptor capacity is unchanged. The prospective row maximum is checked across all original cuts; at least one member belongs to the current C, so preceding S archives contain at most the original request maximum minus one. The six original comparable type shapes keep their existing string/size weights. The new local source counts explicitly cover claims, the entire ancestor chain and archived member facts; pointer identity is never substituted for the pointed-to behavioral data. Directory lengths remain explicit original selections for the future factory to bind before H0.

Four authored UNIT roots check independent ordinary/empty/last prefix literals, complete roots, firstReading recording exclusion, unchanged type-shape binding, insufficient future recorder capacity and the original local firstRead guard. They are NOT_RUN. These are conditional successful structural DATA rows; changed-shape failures, native reflection/allocator/backing/RAM, complete failure/recovery/final/all11 sufficiency AND feasibility and factory/operating qualification remain separate. No recorder, observer, live seal, source, counter, clock, allowance or runtime path is installed by a numeric quote. Full connected publication and fresh native CI remain gated by the original complete obligations.


### Direct filenames and catalog namespace at original execution roles

The successful name component assigns direct read/write/getter names separately from allowed-map construction at the unchanged I/B/P/M/C/A cuts. One surrounding initial model contributes its original three indexed names; each preceding T/C/S and archived response/TLS pair keeps its source-derived name work. Legacy inventory/ref names and firstReading names remain in their existing component, and transcript recording copies remain in the recorder component. No whole-history polynomial is divided into local quotes.

The current C retains at least one of the original global requests. Only its own completed-S advance may include the entire original request maximum. Per-member direct spool names and catalog insertions use the original k<=31 and m<=32 bounds. Full M includes its entered P; the unchanged remaining-work guard still requires its additional P. Catalog setup-recovery, attempt and entry caps are explicit original selections for the future pre-H0 factory to bind. Both root maps are copied once per actual after-catalog, not once per map element.

Five safe UNIT roots check independent empty/ordinary/last prefix literals, selected setup and map-copy deltas, current-versus-own-advance request caps, whole-zero invalid selections and the unchanged firstRead sublimit. They are authored but NOT_RUN; per-file coverage is UNMEASURED. These rows only price observed logical name events, not native formatting/map hashing/CPU, allocation/backing/RAM, changed-shape failure, complete raw/codec/failure/recovery/final work, all-eleven sufficiency, feasibility or factory qualification. Original runtime guards, allowances, clocks and counters are unchanged. ONE complete connected publication with fresh native CI remains gated by those full obligations.


### Direct raw copies, hashes and comparisons at original roles

The successful direct-raw component now has local I/B/P/fullM/S/ownA rows for healthy/prior checkpoints, current T/C/S suffixes, raw writer inputs, spool hashes and bindings, policy-root getters, archive copies and final member HEAD checks. Initial-model controls already in the legacy component are excluded. Raw firstReading refresh and transcript recording belong to their existing components. These are additional direct callsites, not repeated parsers or a whole-history polynomial divided by a stage count.

The fixed wrapper distribution follows actual calls: B has sixteen comparisons, successful S seven and own advance one. P has at most its member-head pair and a nonfirst spool-pack pair; full M additionally compares the new pack and result. That final result comparison is not counted again in the retained-prefix result loop, which still compares only prior retained members. T/C and S writer input copies are split from their actual reused before-pin and object/metadata copies. Current prior membership uses R-1 while own completed-S advance may use the full R.

Four safe UNIT roots are authored for independent original-cut decoded/logical literals, wrapper/result/writer separation, optional own advance and whole-zero refusal with the unchanged local firstRead guard. They remain NOT_RUN and each-file coverage is UNMEASURED. The component creates no Source, meter, hold, request allowance or clock. Full M retains its P and the original remaining predicate still requires its additional P. Full parser/control composition, partial failures, native CPU/accessors/allocator/backing/peak/lifetime RAM, resource/window/elapsed feasibility, recovery/final/page paths and factory qualification remain open before ONE complete connected publication and fresh native CI.


### Outer frame, inventory and stored-object codec role components

The successful outer-codec component assigns existing profile/highwater/frame, inventory and stored-object primitives to their actual I/B/P/fullM/S/ownA locations. It does not add a frame's internal profiles or predecessor work again. Initial legacy/firstReading and catalog codecs remain in their original components. Inventory reconstruction already rebuilds its supplied objects; only separate outer object constructors and reader text/digest comparisons are added outside that child.

Current B has seven T readers, four C readers and the seal's C anchor. Successful S has three T/C reconstructions and two S suffix readers. P and full M have only the two same-C frame authentications and four parsed pins in this subgraph. Each prior complete T/C/S contributes three checkpoint parsers per actual healthy prefix, with B9/S3/ownA2 multipliers. Own advance reads through its newly completed S. No whole-history polynomial is divided into local allowances. Full M includes P and the original remaining-work predicate still requires its separate entered P.

Four safe UNIT roots cover independent role and codec-count literals, object-rebuild versus caller-copy separation, a smaller scalar metadata/object selection, optional own advance and whole-zero original-limit/local-firstRead refusal. They are authored but NOT_RUN; per-file coverage is UNMEASURED. These are conditional successful numeric DATA rows, not an original Source/owner, complete incoming/refusal envelope, native CPU/backing/RAM, full failure/recovery/final/resource/window feasibility or a qualified factory. Other source scopes and feasibility still gate ONE complete connected publication and fresh native CI. No runtime guard, counter, allowance or clock is changed.


### One selected graph for the six original local source components

The new numeric source graph forwards the same sealed manifest, method/TLS selection, role cut and legacy limits to the assigned control, original-file, retained-context, direct-name/namespace, direct-raw and outer-codec components. The assigned object maximum must equal the legacy object maximum. Catalog filename bytes must cover the original catalog-entry maximum times255; this field is distinct from the combined byte length of the two original directory PATH strings. Every child keeps its original domain and local firstRead guard. A child refusal or checked sum failure returns the wholly empty graph.

The six same-role sums retain all child values for provenance. The file child deliberately excludes the constructor's existing two complete firstRead initialization rows from its joined initial value; its separate bridge is preserved and is not added again here. Successful writer storage remains in that one file child, and only the assigned control child supplies clock counts. Full M retains its entered P, while the unchanged original remaining-work check still requires the separate P. This graph does not replace FULL admitted stage sums, the complete initializer, original hold or any runtime allowance.

Four safe UNIT roots check shared original scalar selections and their immutability, independent additional initial-work literals, existing firstRead bridge and writer allocation ownership, inconsistent object/name/path/capacity inputs, late child arithmetic refusal, and unchanged local or mechanical child refusals. They are authored but NOT_RUN, with each-file coverage UNMEASURED. This is a conditional partial numeric graph; it has no production consumer, source derivation token or execution authority. Remaining incoming/refusal/failure/capture/native CPU/backing/peak/lifetime RAM, resources/windows, recovery/final/page and full feasible-profile/factory qualification still gate ONE complete connected publication and fresh native CI.


### Failed record byte costs

The failed-record body now has explicit prospective constructor and parser byte-event rows. The original fixed 16 KiB header, complete wire allowance and rejected extra byte must fit the selected object maximum before a dial. A parser row includes one bounded original caller copy, its framing and JSON decode, prefix comparison, rebuilt body and exact-byte comparison. It retains the original generic header bound for rejected input; a small normal header does not reduce that allowance.

One failed settlement constructs the failure body once and reads it back twice. Its five result-context validations remain in the existing record-control component and are excluded from this body row. The failed settlement uses the same original closure allowance and clock/window while the member invocation is active. This quote does not multiply a terminal failure by the member count, reset a meter, refund an admitted upper or create a source witness.

The authored tests compare three independent integer byte-cost vectors, inert finite failure DATA with the original observer and two readbacks, the exact pre-dial header/wire/sentinel boundary, and the existing context ownership. They make no network attempt and obtain no settlement authority. They remain unexecuted until fresh native CI. Outer failed-settlement object/inventory/file/namespace/context differences, incoming and uncertain failures, native CPU and memory, recovery, final publication, complete source-derived sufficiency and feasible factory qualification remain separate open requirements.


### Failed settlement additions and read-sentinel checkpoint

A failed settlement can retain three objects: the failure original, optional successful-prefix results, and the settlement record. The prospective additions price that failure object's three external constructions and two readbacks, and change only this settlement's one inventory construction and two inventory readbacks from two to three objects. Inventory-internal object reconstruction is already inside the child cost. The original 64 MiB inventory aggregate, metadata caps, same closure and original clock/window remain unchanged.

The additional failure file keeps its fixed header metadata and wire-prefix payload separate, including the funded rejected byte. Its write/sync/identity costs and one physical file overhead are separate from the existing settlement rename and clock. An absent results prefix does not refund the original successful-path upper. These additions are conditional arithmetic DATA; catalog/namespace/recording/typed-context, incoming and uncertain/native/memory, recovery, final publication and feasible source-factory qualification are still required for a complete program.

The original recovery checkpoint previously rejected the one extra byte that the byte capture, failure-original layout, object-fit check and work allowance already fund to detect an oversized read. The checkpoint now retains this same single sentinel as failed DATA. It does not make that response successful or permit any further byte. Finite unit scenarios construct diagnostic member-failure DATA after a normal owned fixture setup; no oversized response is sent on a connection. They require terminal failed custody for the funded boundary and refusal beyond it, with the same original hold, window and order. All new tests remain unexecuted until fresh native CI.


The failed-settlement additions explicitly assign two further direct-raw copies to a checked component: the writer caller copies the new failure object before invoking its file primitive, and the checkpoint clones the whitelisted failure-stage string. Their combined upper is F+1+8=F+9, with only logical byte work. This is added to the extra work, separately from body construction, parser input, stored-object/inventory rebuilding and file I/O. The finite-layout tests also observe these two real copy helpers on one fresh inert meter and require the exact byte count plus two event sentinels to fit this component. Native CPU/memory and the other remaining complete-program requirements are unchanged.


### Same-original local transport and failed-settlement composition

The prospective transport component retains the sealed grammar's eight class counts and the actual selected method/TLS limits. It derives one member BYTE row by taking coordinatewise maxima only over classes that can occur in that original grammar. It retains each class's individual quote. The complete collection capture total is provenance DATA, never divided by requests or copied into a local stage. The one irrevocable request is charged by the original owner before dial, not again by the BYTE engine.

Capture uses the analytic request endpoint r=1: q(r)-q(1)=(r-1)(r+4-2W)<=0 for valid 1<=r<W. Failed-body request terms instead increase with request length, so each active class uses min(16384,W-1). These are independent analytic endpoints; no actual request is replaced. Known write/read/framing prefixes consume a subset of the original BYTE event envelope, including the already funded rejected read byte. Dial, native TLS/CPU, observer uncertainty and complete failed-S behavior remain separate qualifications.

One terminal failed-S extra is selected coordinatewise across those same active classes. Its body/outer/directRaw/read/write/storage provenance comes from the accepted failed-extra child. The existing successful S upper is kept in full and the failed extra is added once; missing successful spools or an absent prefix do not create refunds. Storage dimensions are bounded separately, not summed over hypothetical failures. Zero-count gas or execution classes supply no failure allowance and cannot force an unrelated object-cap rejection. They must still satisfy the original method/global domain.

The parent combines the six earlier source components with these full-M and closure additions at the same original cuts. I, B, entered P and own advance are unchanged. The existing firstRead initializer bridge, full admitted-stage SUM, extra preparation in dispatch.remaining, clocks, request order, original quotas/windows and all runtime guards remain unchanged. The source/storage additions are checked and any child failure returns the whole zero result. This is not a factory, authority, admission or proof of complete all11 sufficient-and-feasible native resources. Incoming/refusing/recording/typed-context, native backing/peak/lifetime RAM, recovery/final/page and full source binding still need completion before ONE connected publication and fresh native CI.

### Disjoint final reservation and original failed inspection entry

The selected successful-final read envelope and its two additional original
clock events now belong to the same pre-H0 whole hold. They do not consume the
failed-inspection row, renew the execution allowance or change the old initial
seed. A zero final row preserves the existing incomplete candidate profile
object. A selected final row is committed into that profile and participates
in the existing checked lifetime clock total and snapshot comparison. This
reservation alone does not install a final observer or permit its execution.

The original parent also has a private failed-collection inspection entry.
Only its genuinely failed, closed original group can consume the existing
one-use read-only lane. One product function runs both complete inventory,
scan, legacy, initial, group, member and tail reconstructions between the
original catalog observations. Each pass derives its collection header from
its own first-response originals. It accepts no producer object resolver,
replacement clock, execution retry or caller-selected filesystem path.
Failure or cancellation closes only that inspection lane; returned catalog
DATA cannot acknowledge a group or authorize a Source or Agent message.

The two added production files are connected to both existing strict 90%
coverage inventories. Their safe scalar, owned-file and cancellation tests
are authored but NOT_RUN. The successful final reader/clock role, complete
source cost compiler, production collection caller, physical resources and
same-profile sufficiency and feasibility remain unfinished. This unpublished
part stays in the original connected implementation bundle; it does not
qualify a smaller operating profile or complete the whole program.

The ordinary offline 256KiB object selection is deliberately rejected for an active 256KiB wire class: the unchanged pre-dial failure guard requires 40+4+16384+W+1=278573 bytes. New UNIT DATA may select 512KiB prospectively before any source/H0; it does not patch an owned source or qualify a deployable profile. Exact 278573/278572, independent per-class literals, inactive/active tracer transitions, original-cut/storage ownership and complete-zero refusals are authored safe UNIT scenarios, not execution results. Four new roots remain NOT_RUN; the new product coverage target remains EACH90 UNMEASURED. Existing published green CI does not validate these unpublished additions.


### Successful original readback connection (unqualified candidate)

The private verified-collection caller now connects the existing collector to a
separately selected successful-final observer. The observer is allocated before
the original H0 and uses the existing final read row plus two clock claims from
the same original backend. It shares only sealed checkpoint/member provenance
with the failed-inspection recorder; both independent decoders, inventories,
semantic models and catalogs belong to the fresh reader. The failed-inspection
entry still requires a genuinely failed original execution.

A complete healthy last settlement closes its original execution meter without
refunding its actual prefix. Final verification reconstructs both models from
the original opened roots, checks the complete catalog and selected end state,
copies its private result before the final native sample, and closes the final
observer. Failure quarantines the original collection. Reads cannot spend the
two reserved clock units or use a writer, network role or replacement clock.

The result remains private collection DATA. This connection does not provide
production derivation, complete source-cost compilation, native CPU/backing or
peak/lifetime RAM qualification, profile feasibility, a protected page, Source
readiness or an operating policy. New owned-TLS tests use the existing benign
recorded deployment and an explicit UNIT-only profile selected before H0; they
are not deployment-to-current or production acceptance. Native execution and
file-level coverage must still be established by CI for this exact source.
