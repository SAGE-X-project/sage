# Complete eip155 records for DID resolution

The normative design is SAGE 0.10.0 chapters 09 and 10 at sage-spec
`06b50e177ccd22bce7d70fd80c2d7f822ae3fd3b`, including the adopted DIDP rules.
Historical frozen evidence at `1820ab5` keeps its original scope.

## Design and boundaries

Keep the existing key-oriented `Snapshot` and protected-operation `Gate`
unchanged in behavior. Add `RecordSnapshot` and `RecordReader` for the complete
REG-01 bytes and source observations needed by DID resolution. No missing
controller, service, lifecycle or version is inferred from a legacy endpoint,
nonce or boolean. Empty services must be an explicit array, not an absent field.

`EIP155Observer` serializes a new read, verifies configured source/network,
readiness/finality and the record/key/service block identities, checks the full
bounded JSON and all REG-04 proofs, and persists the canonical-record digest
and finalized version through the supplied Store. It rechecks the operation's
five-second freshness window after durable storage. Its final local clock
sample supplies observedTime. Record and returned key bytes have independent
ownership. All supported chapter 11 key encodings are checked; no optional
algorithm is silently coerced into Ed25519. Expired keys may resolve for
inspection and are omitted from the DID verification relationships.

`didsource010.EIP155Source` maps this observation to did-go advanced/sage.
DID 1.0 and 1.1 projections come from the same complete record. Public DID
resolution never grants permission for a protected message; those operations
still use their own fresh Gate observation.

## Caller obligations and unfinished deployment work

The configured reader is an explicitly trusted local boundary, not remote JSON
with ready/validated/finalized flags. It must verify source identity, the pinned
chain/address/code/upgrade policy and authenticated ABI, finality/readiness,
complete history, controller immutability and historical KEM signer authority.
The protected clock and durable Store must retain their existing ownership and
recovery guarantees. Digest/version persistence detects observed rollback and
equal-version conflict; it does not establish an unspecified storage's rollback
resistance or detect every invalid transition in an unobserved history.

This package provides no RPC implementation or selected operational trust root.
The old sage-contracts ABI cannot supply the missing complete record semantics.
Contract changes, authenticated read/write mapping, deployment and actual Source
qualification stay in program stage 8. IdentityAndReadiness remains NOT_BOUND,
deployed host controls remain NOT_RUN, and full conformance is not established.
Tests with an injected local reader are adapter evidence only. The future binding
also needs an authenticated absence result mapped to `record.not-found`; current
reader failures uniformly mean `record.unreachable`, never an invented absence.

## Validation

Run the changed packages with the race detector and statement coverage:

```sh
GOTOOLCHAIN=local go test -race -coverprofile=/tmp/eip155-coverage.out \
  ./pkg/agent/registry010 ./pkg/agent/didsource010
```

Scenario tests cover record/proof integrity, mixed blocks, missing fields,
clock/freshness, source failures, durable rollback/conflicts and terminal state.
The safe runtime tests resolve through did-go and its authenticated local HTTPS
handler without accessing a blockchain or sending a transaction.

## Independent review

A separate read-only LLM reviewer checked the adopted registry/resolution
requirements and found a valid-integer spelling mismatch in the record mapper.
The correction uses exact rational arithmetic for issued/expiry and adds web
and eip155 regression scenarios for exponent/decimal integer spellings and
non-integers, negative zero and limits. A second review found no remaining
specific defect. This code review is not an organizationally independent audit
or runtime observation of an actual blockchain Source.

## Public DID dependency adoption

The core selects public did-go root
`v0.0.0-20261018080828-3b7a79c1c326` and advanced/sage
`v0.0.0-20261018120945-5fac371e54db`, with checksums and no local
replacement. This includes the earlier date-consumer, recursive-fragment,
relative-ID, URL-fragment and metadata-presence repairs. The SAGE envelope
now retains explicit active/created `deactivated:false`; unsupported Accept
negotiation retains the standard typed problem and SAGE diagnostic code.

`TestDIDConsumerMetadata`, `TestDIDConsumerAcceptDiagnostics` and
`TestDIDConsumerTLS` exercise the core's complete-record Source through native
public APIs and owned TLS GET/POST requests. Malformed Accept, unsupported
options and Source failures keep their separate diagnostics. The reader is
an injected fixture; no operational chain, history authority or custody is
selected. CI retains the exact selected module versions/sums, fresh test events
and coverage profile. Historical Inspector subjects are not updated in place.

URI-list intermediate resolution still requires a separately configured
`resolver.ForDereferencing()` instance where the application explicitly chooses
that policy. This dependency update does not enable it globally or claim
ADK/Rust adoption, a production transport binding or whole DID conformance.
