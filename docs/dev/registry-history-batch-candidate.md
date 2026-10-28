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
