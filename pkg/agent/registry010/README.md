# Operation-scoped registry observations

`registry010` implements the observation and key-selection boundary for the
0.10.0 protocol. Supply an explicitly trusted `Source`, local `Clock`, and durable
`Store`. The source must validate the entire closed record, cryptography, proofs,
source identity, network binding, readiness and finality before returning a
projection. Boolean fields in remote JSON are not evidence of these checks.
This module is not a network resolver or a full record validator.

Every observe, select and pinned-key check performs a new source read. Acquisition
must occur after the operation starts and at most five seconds before the gate,
including time spent committing durable state. A clock failure, rollback,
unfinalized or inconsistent snapshot, unavailable source or store failure denies
the operation. A stale result requires another read before a deferred decision.
The final clock sample is this module's observation linearization point; callers
must arrange any later authorization gate and side effect accordingly.

Selection requires the exact usable Ed25519 signing URL. A handshake additionally
selects the first usable X25519 key in ASCII name order. Pinned keys retain their
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
