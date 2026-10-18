# Handshake key authority observations

SAGE 0.10.0 completion endpoints take exact DID/key references, closed
handshake bytes and a locally configured Registry Gate. They do not take a
peer DID document. A DID document is not an alternative signing or KEM
key source. This test does not add a DID-document parser or change that API.

`TestHandshakeKeyAuthorityBoundary010` uses fixed, in-memory fixtures:

- Normal and matching-current-KEM controls complete the authenticated
  handshake and exchange an inert HTTP request/response. Exact tuple/key
  references, plaintext, correlation, request replay and response reuse
  are checked. No HTTP listener or network destination is opened.
- The static version3 Bob fixture retains the original `kem-1` as revoked
  and adds distinct accepted `kem-2`. REG-02 selects the ASCII-first usable
  KEM; the new key is the only usable one. Matching local key bytes work;
  old local key bytes refuse completion without replay publication. This
  is not an executed Registry rotation or a proof of management authority.
- Current signing-key mismatch, either participant's signing-key revocation
  and Source failure refuse a usable completion and replay publication.
  Another accepted, unselected signing key is present from the start.
  Revoking signing-1 therefore preserves an active record and demonstrates
  refusal of fallback to signing-2, not last-key revocation semantics.
- Four document/alias names in the outer request and nested initiation
  are correctly re-signed with owned fixture credentials, then refused
  without a server Source read or replay publication. Nested extras have
  string values to separate unknown member names from value-type errors.
  These are closed-wire-field refusal observations, not interpretation of
  an accepted document, alias normalization or generic DID interoperability.

Each leaf emits one bounded JSON witness with actual Source attempts,
returned public snapshot fields, exact requested DID and bounded raw public
request with its hash. Positive controls also retain signed completion/application
frames and the authenticated tuple. Handshake Source traces are captured
before later record-message checks. The test checks the result directly;
a witness's boolean by itself is not an independent authentication proof.
Source `Validated`, `Ready` and `Finalized` are trusted fixture inputs,
not remotely established Registry proof/readiness/finality evidence.
The Snapshot API has no controller field, so no observed controller is
invented. Source fixture construction preserves existing key history;
its key-list digest is a test commitment, not an on-chain full-record proof.

All contradictory inputs stay fixed unit data. No attack/reproduction CLI,
arbitrary destination or private-key export is added. Secret keys, shared
values, derivation seeds and traffic keys are never emitted. Existing
opaque informational-payload tests, custody stand-ins and native TLS tests
remain separate evidence. This group exercises local key provisioning;
it does not establish a protected external custody deployment, actual
Registry/contract Source, full DIDP-04, or whole-protocol conformance.

The Test workflow retains exact revision/tree, selected HPKE/Registry/DID
source hashes, compiler identity and raw race-test output in the separate
`handshake-key-authority` artifact. Selected-source provenance is not a
complete transitive compiler-input attestation. Full CI and the independent
artifact review remain required in addition to the selected unit run.
