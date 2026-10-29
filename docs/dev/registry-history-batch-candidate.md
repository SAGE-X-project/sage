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
