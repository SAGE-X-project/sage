# Legacy APIs and the SAGE 0.10.0 deprecation plan

Status: class A symbols are marked `Deprecated:` (phase P2). This document
records the classification and the order of the work. It classifies exported APIs that predate the SAGE 0.10.0 protocol,
names their 0.10.0 replacements, records known callers at `c2b8d21`, and orders
the work needed before marking them. The Rust core keeps a matching plan in
`rs-sage-core/docs/legacy-010-deprecation.md`.

Legacy APIs remain available and keep their historical behavior. They are not
0.10.0 conformance subjects: Inspector observations of 0.10.0 cases must call
the 0.10.0 entry points. Removal follows [VERSION_POLICY](refactoring/v2/VERSION_POLICY.md)
rule 3 (two minor releases after `Deprecated:`) and the decision that
deprecated code leaves no earlier than `v1.8.0`.

## Why preparation comes before marking

- **0.10.0 code used the legacy JCS package internally.** `guard010/json.go`
  (`Canonicalize`, after its own strict validation), `hpke/record010.go`,
  `hpke/derivation010.go` and `hpke/completion010.go` called `crypto/jcs`.
  Marking `jcs.Canonicalize` or `jcs.Marshal` would have made these
  cross-package calls fail staticcheck SA1019, which `.golangci.yml` enables
  for all files. P1 moved the implementation to `pkg/agent/internal/rfc8785`;
  these packages now call it directly and `crypto/jcs` is a thin wrapper.
- **Exported helpers call legacy APIs on purpose.** `pkg/vectors` (imported
  by sage-inspector) generates and checks the published sage-spec
  `1.0.0-draft.1` vector suites, whose outputs are defined by the legacy DID,
  HPKE and session APIs. `cmd/sage-did` operates the legacy on-chain registry
  through `did.Manager` (class C) and reads the chain from legacy DIDs with
  `did.ParseDID`. Moving either to 0.10.0 APIs would change published vector
  outputs or break the CLI, so both stay legacy callers.
- **External consumers exist** (pins vary): sage-adk
  `core/guardbinding/load.go` calls `jcs.Canonicalize`; sage-gateway and
  sage-adk use `core/rfc9421`, `did.Manager` and the multi-chain resolver;
  sage-inspector harness adapters call several legacy APIs on purpose.

## Classification

### A. Deprecate after preparation (a 0.10.0 replacement exists)

| Legacy | 0.10.0 replacement | Behavioral difference |
|---|---|---|
| `did.ParseDID`, `did.ValidateDID` | `did.ParseDID010`, `did.ParseDIDURL010` | Legacy accepts only `ethereum`/`eth`/`solana`/`sol` case-insensitively and rejects canonical `web` and `eip155` DIDs |
| `jcs.Canonicalize` (lenient entry point) | `guard010.Canonicalize` | Legacy keeps the last duplicate key, normalizes `-0`, and has no size, member or depth limits |
| `hpke.CombineSecrets` | `hpke.CombineSecrets010` | Different salt and label; 0.10.0 binds the transcript hash and rejects a zero shared secret |
| `hpke.MakeAckTag` | `hpke.MakeAckTag010`, `hpke.VerifyAckTag010` | v1 labels versus transcript-bound 0.10.0 labels |
| `hpke.DefaultInfo`, `DefaultExportContext`, `DefaultInfoBuilder`, `InfoBuilder` | `hpke.BuildDomains010`, `Domains010` | v1 info and export labels |
| `hpke.Client`, `NewClient`, `Server`, `NewServer` and their payload types | `hpke.CompletionEndpoint010` (`New…`, `NewCustody…`, `NewProtected…`) | Legacy handshake without the 0.10.0 authenticated completion, replay and current-key gates |
| package `handshake` (already deprecated) | `hpke.CompletionEndpoint010` | Its doc comment still points to legacy `hpke`; correct it in P1 |
| `session.SecureSession`, `NewSecureSession*`, `DeriveSessionSeed`, `ComputeSessionIDFromSeed`, `Params`, `Manager` | `session.RecordSession010` (via `CompletionEndpoint010`) | Historical wire format and derivation (session README) |

### B. Keep (shared building blocks, not deprecated)

- `jcs.Marshal` remains as the canonical serializer used by 0.10.0 code until
  P1 provides an internal equivalent; it is not a 0.10.0 validation entry point.
- HPKE KEM, HKDF and hash primitives used by both generations.
- Key, format and wrapper packages (separately governed by their own
  `Deprecated:` markers).

### C. Legacy with no 0.10.0 replacement (decision needed before deprecating)

| API | Consumers | Note |
|---|---|---|
| `core/rfc9421` general signer/verifier, `HTTPVerifier`, `Canonicalizer`, parsers | sage-gateway (prod), sage-inspector (prod), sage-adk `adapters/sage` (prod) | 0.10.0 only provides session-bound HTTP (`hpke/http010`); no general RFC 9421 replacement |
| `did.Manager`, `did.Resolver`, multi-chain resolver, `ethereum.AgentCardClient` | sage-gateway, sage-adk (prod) | `registry010.Gate` needs a trusted Source; it is not a network resolver |
| `did` key proof-of-possession (`SAGE-PoP`) | `pkg/vectors`, `cmd/sage-did`, sage-inspector harness | `registry010.PoPChallenge010` builds challenge bytes only |
| A2A card proof (`did/a2a_proof.go`) | sage-inspector (prod) | No 0.10.0 card verifier; depends on `ParseDID` and `jcs.Marshal` |

Class C items must not be marked until a replacement exists or the owners decide
to retire the feature. Until then their documentation states that they are not
0.10.0 conformance paths.

## Phases

1. **P0 (this document).** Classify, list callers and replacements.
2. **P1. Decouple and migrate inside this repository, without behavior change.**
   - Done: 0.10.0 packages call `pkg/agent/internal/rfc8785` instead of
     `crypto/jcs`; output is unchanged because both run the same code.
   - Reviewed: no remaining internal caller should move. `pkg/vectors`
     (1.0.0-draft.1 suites), `cmd/sage-did` (legacy registry CLI), the legacy
     `hpke` client and server, the A2A card proof and the tests of the legacy
     APIs themselves are intended legacy callers. P2 gives each a scoped
     `//nolint:staticcheck` with that reason.
   - Done: the `handshake` package comment names `CompletionEndpoint010`.
3. **P2. Mark class A.** Done: each class A symbol carries `// Deprecated:`
   naming its replacement, `CHANGELOG.md` lists them under Deprecated, and the
   intended legacy callers carry a scoped `//nolint:staticcheck` with the
   reason. sage-adk no longer calls `jcs.Canonicalize` (it uses
   `guard010.CanonicalManifest`).
4. **P3. Consumers.** Ask sage-adk to switch `guardbinding` to
   `guard010.Canonicalize`; update sage-inspector harness adapters to route
   0.10.0 cases to 0.10.0 entry points (legacy routes stay labeled legacy).
5. **Removal.** Two minor releases after P2 and not before `v1.8.0`.

## Verification for each phase

- `go build ./...`, `go vet ./...`, `go test -race ./...` and golangci-lint.
- 0.10.0 vectors and Inspector observations unchanged by P1.
- A search for each class A symbol shows only intended legacy callers before P2.
