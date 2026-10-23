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
Transport binding, complete historical command replay and actual Registry
caller/outcome evidence remain separate implementation work. Existing
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
