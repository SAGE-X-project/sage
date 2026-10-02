# Operation-scoped registry observations

`registry010` implements the observation and key-selection boundary for the
0.10.0 protocol. Supply an explicitly trusted `Source`, local `Clock`, and durable
`Store`. The source must validate the entire closed record, cryptography, proofs,
source identity, network binding, readiness and finality before returning a
projection. Boolean fields in remote JSON are not evidence of these checks.
This module is not a network resolver or a full record validator.

`CheckWebRegistryMedia010` checks only the optional web Registry response's
`Content-Type`, `Content-Encoding`, and forbidden trailer fields under REG-08.
The trusted HTTP adapter must pass individual field lines before coalescing or
decompression. Success does not authorize a record: HTTPS origin, cache,
response body, proof, freshness and policy checks remain with the trusted Source.

`CheckWebRegistryEnvelope010` checks the separate 69632-byte JSON wrapper,
duplicate-member and Unicode boundaries, and the five-second `issued`/`expires`
lifetime. Its `record` member is only checked to be an object. A positive result
does not validate that record's separate 65536-byte bound, schema, proofs, DID
or authorization. The trusted
adapter must enforce the read limit before parsing and supply a trusted clock.

`CheckWebRegistryRecordShape010` checks the nested record's exact encoded
65536-byte bound, closed fields, web DID binding, version, key and service
structure after the envelope check. It does not check cryptographic key points,
proof signatures, historical immutability, HTTPS origin or write authority.
Only a trusted Source that has completed all of those checks may authorize a
protected operation. The local `registry-record-shape010` adapter is for
bounded Inspector observations, not a network resolver.

`CheckWebRegistryProofs010` additionally validates public-key points,
REG-04 signing proofs for the three registered signing suites, and X25519
endorsements. It rejects non-prime-order Ed25519 points, high-S ECDSA values,
wrong secp256k1 recovery keys, and unusable X25519 points. Retained historical
KEM signers need authenticated mutation history before their earlier authority
can be trusted. This check does not authenticate the HTTPS origin, controller,
or mutation history and cannot authorize an operation alone.

`WebRegistryRequestURL010` constructs the exact web DID read target only when
its HTTPS origin appears in a locally configured allowlist.
`CheckWebRegistryResponsePolicy010` rejects non-200 status, wrong media, and
responses lacking an origin `no-store` directive. The local policy adapter
exercises these decisions without networking. A production HTTP adapter must
still authenticate TLS, enforce a network destination allowlist after DNS
resolution, avoid intermediated caches and redirects, reject ambiguous framing,
and bound the body before these helpers can contribute to an observation.

`CheckWebRegistryTLSOrigin010` makes a fresh TLS connection to one exact IP
endpoint that appears in a local destination allowlist. It validates the web
DID domain through certificate name and chain verification against an explicit
root. The connection is closed without an HTTP request, so this is a TLS
subcondition only; a future record fetch must bind its response to its own
authenticated connection and still enforce cache, framing, body and authority.

`FetchWebRegistryRecord010` now performs a fresh GET on its own authenticated
TLS connection and verifies a bounded JSON record and its key proofs. It accepts
only one unambiguous HTTP/1.1 `Content-Length` response, with a 16 KiB header
section and at most 64 field lines. Chunked or other framing is rejected in
this bounded adapter. It does not authenticate controller writes, tombstones or
mutation history, and its result must not be treated as complete REG-08 authority.

`CheckWebRegistryCreationShape010` and `CheckWebRegistryTransitionShape010`
validate a version-1 creation and one claimed historical change. They check
proofs at each supplied observation time, retained immutable keys, exact
version increments, terminal deactivation, and whether a newly endorsed KEM
key had a usable signing endorser at addition. The caller must supply a
complete authenticated history and trusted observation times. These pure
checks neither authenticate controller/operator actions nor commit an atomic
write, so they are not a Registry administration implementation.

`CheckWebRegistryHistoryContinuity010` applies those checks to a caller-supplied
sequence beginning with creation and compares its last record with a current,
proof-valid envelope. It rejects missing versions, time regression, an invalid
mutation, or a current record that differs from the last supplied version.
Sequence continuity does not prove that the source supplied every real
mutation; the Registry deployment must authenticate and retain the log and
bind each historical time to the committed write.

`CheckWebRegistryCreationAdmission010` and
`CheckWebRegistryMutationAdmission010` enforce controller identity, exact
expected version and operation-scoped delegation through a deployment-supplied
administration authority. That authority must derive the actor from verified
credentials and supply controller-authorized management state. These functions
only check a proposed write; the deployment must repeat the decision while
atomically reserving or comparing and writing the record and durable history.
They do not define an administrative API, authenticate transport credentials,
or grant a reusable authorization token.
An expired but still accepted signing key may leave an `active` prior record
unusable for ordinary reads. Write admission can use that prior record only to
add a newly proven, usable signer or move to a valid terminal state. It still
verifies every prior key proof and rejects an `active` record with no accepted
signing key at all. Ordinary read and historical transition checks stay strict.

`ApplyWebRegistryWrite010` moves admission into a store-provided transaction.
The store supplies a current, freshly enveloped record, complete asserted
history, source identity, and credential-backed authority in one serialized
snapshot. The core checks history against the current record and returns a
complete replacement containing the candidate, new history entry and terminal
tombstone. A decision error leaves state unchanged. An uncertain I/O outcome
must quarantine the writer until the journal is inspected.

`OpenWebRegistryWriteJournal010` is a local single-DID reference store. It
holds an exclusive writer lock, appends one complete state row and syncs it
before exposing the change, and rejects incomplete state on restart. A prior
response is re-enveloped at the trusted mutation time so later legitimate
writes are not blocked by its five-second HTTP response lifetime. The path,
source identity, credentials and disk integrity are deployment
inputs; this journal does not authenticate them or establish REG-08 deployment
conformance.

`ApplyWebRegistryOperatorCommand010` implements the controller-only grant and
revoke transaction from the amended REG-03/REG-08 design. An exact operator
and scope pair is stored with the next public record version, full history and
tombstone under the journal's one-writer lock. Lifecycle writes now use that
committed grant set for delegated authority and retire scopes that are invalid
in the next state. On each write, the core reconstructs active grants from
the complete committed history rather than trusting a standalone positive
grant assertion. The local journal rechecks each grant transition on restart.
The `operator-1` journal header deliberately rejects older local journal
files; migration requires an explicit, trusted reconstruction of actor and
grant history. The standalone `CheckWebRegistryMutationAdmission010` still
accepts a deployment-supplied delegation predicate for bounded compatibility
checks, but its result alone is not evidence of the atomic grant contract.
The Registry service must still bind exact administrative request fields,
authenticated credentials, trusted storage ownership and a read-only
Inspector view before claiming deployed conformance.

`AcceptWebRegistryAdminMTLS010` is an optional deployment reference for
controller-only administrative writes. It performs a fresh server-side TLS
handshake with required client certificate verification, then maps the verified
leaf certificate's SHA-256 to a configured controller identifier. The returned
session reads the administrative request from that same connection and supplies
the journal's `WebRegistryAdminAuthority010`. The deployment remains responsible
for bounded request framing, trusted clock, exact candidate bytes, certificate
pin provisioning and rotation, and the public Registry source. The reference
adapter grants no delegated operator authority or complete REG-08 conformance.

`ObserveWebRegistryJournal010` compares a fresh, TLS-authenticated public
record with the same DID and exact HTTPS origin configured for a local
administrator journal. It rejects a different record or a journal change
during the fetch. This checks one publication snapshot; it does not bind a
deployed public server's storage to the administrator journal, authorize a
later operation, or establish complete REG-08 conformance.

`WebRegistryWriteJournal010.PublicEnvelope010` creates a new five-second
response from the same committed journal that accepted an administrator write.
The deployment must bind its HTTPS handler to these exact returned bytes and
the configured origin. The local reference API does not prove deployed
storage ownership, clock integrity, or production server behavior.

`PoPChallenge010` constructs the exact five-field REG-04 challenge bytes from
already validated record components. It keeps the new `sage-pop-0.10.0`
domain separate from the legacy DID `SAGE-PoP` path. The bundled 0.10.0
fixture checks signing-key and KEM-endorsement challenge bytes. This byte
constructor does not verify a signature or establish the signer's authority;
the trusted validating Source still owns those checks.

Every observe, select and pinned-key check performs a new source read. Acquisition
must occur after the operation starts and at most five seconds before the gate,
including time spent committing durable state. A clock failure, rollback,
unfinalized or inconsistent snapshot, unavailable source or store failure denies
the operation. A stale result requires another read before a deferred decision.
The final clock sample is this module's observation linearization point; callers
must arrange any later authorization gate and side effect accordingly.

The Gate parses each DID and signing key URL with the strict 0.10.0 grammar
before consulting the Source. The DID must name the configured registry, and
the key URL must name that exact DID; legacy aliases and malformed fragments
are rejected. Selection then requires the exact usable Ed25519 signing URL.
A handshake additionally selects the first usable X25519 key in ASCII name
order. Pinned keys retain their
original identity, bytes and expiry; unrelated record changes do not replace them.
Pins are metadata, not reusable grants. A session owner must call the check for
each operation and close the session on failure. Session ownership, authenticated
transcripts, signatures and dispatch integration are outside this module.

The journal durably records the highest finalized version and full-record digest
per registry/DID, rejecting rollback and conflicting equal versions. Confirmed
record deactivation is permanent; unfinalized observations never update this
state. Supported key algorithms in this module are Ed25519 and X25519.

Create a journal only during explicit initialization. Ordinary restart must open
an existing complete journal. One writer holds an exclusive `.lock` file; a crash
leaves that lock in place. An operator must establish exclusive ownership before
removing it. Missing/truncated state fails closed. Paths and storage are trusted;
this file format does not detect malicious rollback of the disk itself. The
journal has a 64 MiB size and 4096-record capacity; reaching capacity denies new
writes. It is not an unbounded production database or automated crash recovery.

The shared 39-scenario fixture injects a synthetic source and clock. Its digest
commits a test projection, not a complete production record. Unit tests and the
Inspector's separate CLI executions validate gate behavior and actual storage,
including process restart. They do not establish deployed chain finality,
publication latency, full protocol conformance or protection of compromised
trusted dependencies.
