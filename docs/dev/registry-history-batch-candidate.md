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
readers and final C0/Dp/C1 are not provided by this preparation table. No current
Source, owner, meter or transport uses it, and the existing per-RPC path and its
full-width reservations remain unchanged.
