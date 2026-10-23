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
boundary. Complete historical command replay and actual Registry caller/outcome
evidence remain separate implementation work. Existing
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
