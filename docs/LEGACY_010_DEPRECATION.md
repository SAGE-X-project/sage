# Legacy APIs and the SAGE 0.10.0 deprecation plan

Status: preparation (phase P0). No symbol is newly marked `Deprecated:` by this
document. It classifies exported APIs that predate the SAGE 0.10.0 protocol,
names their 0.10.0 replacements, records known callers at `e6c40f4`, and orders
the work needed before marking them. The Rust core keeps a matching plan in
`rs-sage-core/docs/legacy-010-deprecation.md`.

Legacy APIs remain available and keep their historical behavior. They are not
0.10.0 conformance subjects: Inspector observations of 0.10.0 cases must call
the 0.10.0 entry points. Removal follows [VERSION_POLICY](refactoring/v2/VERSION_POLICY.md)
rule 3 (two minor releases after `Deprecated:`) and the decision that
deprecated code leaves no earlier than `v1.8.0`.

## Why preparation comes before marking

- **0.10.0 code uses the legacy JCS package internally.** `guard010/json.go`
  (`Canonicalize`, after its own strict validation), `hpke/record010.go`,
  `hpke/derivation010.go` and `hpke/completion010.go` call `crypto/jcs`.
  Marking `jcs.Canonicalize` or `jcs.Marshal` would make these cross-package
  calls fail staticcheck SA1019, which `.golangci.yml` enables for all files.
- **Exported helpers call legacy APIs.** `pkg/vectors` (imported by
  sage-inspector) and `cmd/sage-did` call `did.ParseDID`, legacy HPKE helpers
  and `session.SecureSession`.
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
   - Give 0.10.0 packages an internal canonical-JSON entry point so they no
     longer call the lenient `jcs.Canonicalize`.
   - Move `pkg/vectors` 0.10.0 vectors and `cmd/sage-did` to the 0.10.0 APIs
     where a replacement exists; keep legacy vectors behind an explicit
     legacy section.
   - Fix the `handshake` package comment to name `CompletionEndpoint010`.
3. **P2. Mark class A.** Add `// Deprecated: use …` with the replacement,
   list them in `CHANGELOG.md` under Deprecated, and use `//nolint:staticcheck`
   with a reason only where a legacy test or vector must keep calling them.
4. **P3. Consumers.** Ask sage-adk to switch `guardbinding` to
   `guard010.Canonicalize`; update sage-inspector harness adapters to route
   0.10.0 cases to 0.10.0 entry points (legacy routes stay labeled legacy).
5. **Removal.** Two minor releases after P2 and not before `v1.8.0`.

## Verification for each phase

- `go build ./...`, `go vet ./...`, `go test -race ./...` and golangci-lint.
- 0.10.0 vectors and Inspector observations unchanged by P1.
- A search for each class A symbol shows only intended legacy callers before P2.
