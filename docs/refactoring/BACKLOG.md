# SAGE improvement backlog

Single tracking list for every problem and improvement identified in the 2026-09 analysis. Each row points to the document that holds the evidence and the proposed fix; this file only tracks status. Update the Status column in the PR that closes an item.

Severity: [치명] security, legal, stability or data-loss; [중요] correctness or maintainability with user-visible impact; [권장] hygiene. Status: Done (merged), PR #n (open PR), Open, Decision (needs a maintainer decision first).

Sources: `SUPPLY_CHAIN_AUDIT.md` (F-ids), `SECURITY_WIRING_AUDIT.md` (§11 a/b/c), `REFACTORING_DESIGN.md` (defects D1-D8, phases), `DECISIONS.md`, `DOCS_GRAPH.md` (§5 stale list), `STRATEGY.md` (§5 migration steps).

## A. Governance and CI (supply chain)

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| A-01 | Bind ruleset to `main` (PR required, 1 review with admin bypass, required checks, linear history, no force-push/delete) | [치명] | F01, F26 | Done (ruleset 8640770, 2026-09-11) |
| A-02 | Restrict `v*` tag creation/update/deletion to administrators | [치명] | F02 | Done (ruleset 22864955) |
| A-03 | Re-enable `security.yml`; add `workflow_dispatch` | [치명] | F03 | Done (enabled) / PR #220 (dispatch) |
| A-04 | Enable secret scanning, push protection, private vulnerability reporting | [중요] | F04 | Done |
| A-05 | Workflow-level `permissions: contents: read`; job-level write only where needed | [중요] | F21, F22 | PR #220 |
| A-06 | Fix `loadtest.yml` invalid YAML; manual-only until harness exists | [권장] | F19 | PR #220 |
| A-07 | `SECURITY.md`; align `CONTRIBUTING.md` with the ruleset | [권장] | F25, F26 | PR #220 |
| A-08 | SHA-pin all 44 remaining action references; require SHA pinning in repo settings; Dependabot `actions` group | [중요] | F05 | Done (#224; `sha_pinning_required` enabled 2026-09-11) |
| A-09 | Pin run-time tool installs (gosec, slither, gitleaks, go-licenses, golangci-lint) and make scanners blocking (remove `\|\| true`, `-no-fail`, `continue-on-error`); triage existing findings first | [중요] | F09, F10 | PR (gosec v2.29.0 blocking, 0 findings; gitleaks v8.30.1 by digest over full history with `.gitleaks.toml` allowlist, 0 findings; Slither 0.11.6 + crytic-compile 0.4.2 `--fail-high`; go-licenses v2.0.1 and license-checker 25.0.1 blocking; golangci-lint v2.13.2) |
| A-10 | `npm ci --ignore-scripts`; `go mod verify`; `GOFLAGS=-mod=readonly`; `govulncheck` (source + binary mode) | [중요] | F11, F15 | Open |
| A-11 | Pin `alpine`/`golang` images by digest; align Docker Go version with `go.mod`; remove `make build-lib \|\| true` | [중요] | F12, F13, F14 | Open |
| A-12 | Dependabot coverage for `tools/codegraph`, `sdk/typescript`, maven, cargo, pip; commit `Cargo.lock`; add TS lockfile; fix nonexistent reviewers team | [권장] | F16, F17 | Open |
| A-13 | Signed, attested, reproducible releases (GoReleaser + cosign keyless + SLSA provenance + SBOM; `-trimpath`, `CGO_ENABLED=0`) | [중요] | F06, F07, F08 | Open |
| A-14 | Gitleaks over history; remove stale `contracts/ethereum/bindings` exclusions; fix `./test/e2e` path | [권장] | F20, F23, F24 | Partly in PR (history scan, gosec exclusion removed); gofmt/golangci exclusions and `./test/e2e` still open |
| A-15 | `CODEOWNERS`, `CODE_OF_CONDUCT.md`, `.editorconfig`, pre-commit | [권장] | F25 | Open |
| A-16 | License consistency (LGPL vs MIT across SDKs/contracts); decide Apache-2.0 relicense | [권장] | F27, `STRATEGY.md` §7 | Decision |
| A-17 | Single version source (`pkg/version` read by every binary); CHANGELOG gaps; release-please or equivalent tagging | [권장] | F28, F29, F30 | Open |
| A-18 | Pin images in `deployments/docker/*.yml` | [권장] | F31 | Open |

## B. Security controls not wired (Go)

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| B-01 | RFC 9421 HTTP path: replay check on `nonce`; enforce `RequiredComponents` (make `content-digest`, `@method`, `@target-uri`, `@authority` mandatory); forward skew bound on `created` | [치명] | §11 a1, a2, b1 | PR (replay guard, skew bound, `RequiredComponents`, `StrictHTTPVerificationOptions`; examples use strict mode) |
| B-02 | HPKE-derived sessions: derive `encryptKey`/`signingKey`, initialise `aead`; regression test | [치명] | §11 a3, D2 | PR |
| B-03 | secp256k1 signing convention: one hash per key type (Keccak-256 vs SHA-256) across `KeyPair.Sign`, envelope verifier, HTTP verifier, HPKE verifier | [치명] | §11 c1 | PR (decided 2026-09-11: Keccak-256 / r\|\|s\|\|v everywhere for Ethereum compatibility; `keys.SignSecp256k1Keccak`/`VerifySecp256k1Keccak` shared by KeyPair, envelope and HTTP paths) |
| B-04 | `did.Manager.Configure` installs a real chain client; `sage-did resolve/list/verify/update/deactivate/card/key` work | [치명] | D1, `DECISIONS.md` 1 | PR (`did/ethereum` registers the creator; `Manager.HasClient`; actionable error when unwired) |
| B-05 | `sage-did update/deactivate` load the given key file instead of generating a random key | [치명] | D3 | PR (`internal/cli.LoadKeyPair` shared by sage-did and sage-crypto) |
| B-06 | Resolver honours on-chain `verified` flag and revocation; returns Ed25519 keys; replaces the `"mock-public-key"` resolver used by `sage-did debug` | [중요] | §11 a5, c5 | Open |
| B-07 | A2A card validation against the chain (`ValidateA2ACardWithDID`) in `sage-did card verify` | [중요] | §11 a4 | Open |
| B-08 | Session-layer replay/ordering: per-direction counter in AEAD nonce/AAD, sliding window, rekey at `MaxMessages` | [중요] | §11 c2 | Open |
| B-09 | Response / tool-result authentication (RFC 9421 response signing or AEAD wrap with request id as AAD) | [중요] | §11 c3, `STRATEGY.md` §2.1 MCP binding | Open |
| B-10 | Audience binding (`@authority` / `RespDID` / `did` component compared with resolved DID) | [중요] | §11 b2, c4 | Open |
| B-11 | Canonical JSON (RFC 8785) for A2A card proofs and the HPKE signed response; deterministic signature selection; P-256 detection; low-S | [중요] | §11 b9, b10, b12, c6; `STRATEGY.md` §3 | Open |
| B-12 | Transport hardening: body size limits, WebSocket origin default-deny, drop `X-SAGE-*` header override, generic auth error messages | [중요] | §11 b4, b5, b6, b13 | Open |
| B-13 | Locks: `SecureSession.Close`, `Manager.SetDefaultConfig`; atomic `CheckAndMark` + `Close()` in `nonce.Manager`; `VerifyAgentMessage(nil opts)` panic | [권장] | §11 b7, b8, b11 | Partly in PR (`nonce.Manager.CheckAndMark`/`Close`); locks and nil-opts still open |
| B-14 | HPKE: cookie check before DID resolution; replace O(n) nonce store; `RespDID` check | [권장] | §11 a7, b2, b3 | Open |
| B-15 | Verifier DoS budget: DID resolution cache with TTL, rate limit before signature verification | [권장] | §11 c7 | Open |
| B-16 | Wire or delete `core/message/validator`, `storage.NonceStore`, `session.ReplayGuardSeenOnce` (currently dead) | [권장] | §11 a6, `DECISIONS.md` 1 | Decision (delete with handshake) |

## C. Contracts

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| C-01 | `ERC8004ValidationRegistry`: access control on `addTrustedTeeKey`, `removeTrustedTeeKey`, `setMinStake`, `setMinValidators`, `setConsensusThreshold`, `setMaxValidatorsPerRequest`; `ERC8004ReputationRegistry.setValidationRegistry` initial-set gating; re-enable Slither `missing-events-access-control`; redeploy on Sepolia | [치명] | `DOCS_GRAPH.md` headline, verified 2026-09-11 | PR (both contracts inherit `Ownable2Step`, 17 tests); **Sepolia redeploy still open** (needs the deployer key); Slither exclusion unchanged |
| C-05 | `scripts/deploy-all-contracts.js` deploys `ERC8004ReputationRegistry` with no constructor argument although the constructor takes the validation registry address | [권장] | found while fixing C-01 | Open |
| C-06 | Slither Medium `uninitialized-local`: `AgentCardRegistry.registerAgentWithParams(...).kemKey` (`contracts/AgentCardRegistry.sol:168`) is never initialised; only High findings block CI | [권장] | Slither 0.11.6 run 2026-09-11 | Open |
| C-02 | Go bindings generated in CI from Hardhat artifacts (`make bindings`) with drift check; `KEMKeyUpdated` event missing today | [중요] | `analysis/05` §5 | Open |
| C-03 | Drop `SageRegistryV2` support: delete `SageRegistryABI`, rename config key, fix Kaia preset (needs AgentCard address on Kaia) | [권장] | `DECISIONS.md` 5 | Decision (address) |
| C-04 | Solana program: placeholder program IDs, no `Anchor.toml`, never built in CI | [권장] | `analysis/05` §5 | Open |

## D. Refactoring (Go library structure)

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| D-01 | Phase 0 safety net: `depguard`, `make codegraph`, CI gate on layer violations; storage conformance test | [권장] | `REFACTORING_DESIGN.md` Phase 0 | Open |
| D-02 | Phase 1: remove `crypto.Manager`/`internal/cryptoinit` (keep `Set*Constructors` as deprecated no-ops), metrics interfaces + `pkg/telemetry`, `pkg/health` decoupling, `EnhancedProvider` move, no `init()` registration | [중요] | Phase 1, `DECISIONS.md` 2 | Open |
| D-03 | Phase 2: single sources of truth (chain enum/presets, DID parser, key ID helper, algorithm table, wire codec, replay guard, `AgentMetadata`, version, key-file loader) | [중요] | Phase 2 | Open |
| D-04 | Phase 3: `did` interfaces (`Registry`/`Resolver`/`Lister`), `EthereumClient` as deprecated wrapper over `AgentCardClient`, `KeyIDBinder` single definition, `SignatureVerifier` in `crypto` | [중요] | Phase 3, `DECISIONS.md` 2, 5 | Open |
| D-05 | Phase 4: `did/a2a`, `internal/{app,cli,config,testutil}`, `pkg/blockchain/ethereum`; no path flatten | [권장] | Phase 4, `DECISIONS.md` 2 | Open |
| D-06 | Phase 5: delete `handshake` (deprecate v1.6, remove v1.7), `core/message/*`, `crypto/vault`, `lib/`, `tests/random`+`cmd/random-test`, dead exports | [권장] | Phase 5, `DECISIONS.md` 1, 3 | Open |
| D-07 | Phase 6: `ARCHITECTURE.md` rewrite, `INDEX.md`, CI dead paths, Dockerfile Go version | [권장] | Phase 6 | Open |
| D-08 | Mark `sdk/*` experimental; remove interoperability claims; add build CI; TS to 0.x | [중요] | `DECISIONS.md` 4 | Open |
| D-09 | `pkg/health` Windows build; P-256 key-type detection; JWK coordinate padding; RSA PSS vs PKCS#1 naming | [권장] | D6, D8 | Open |

## E. Documentation

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| E-01 | Remove or rewrite security-overclaiming examples and SDK READMEs (`a2a-integration/04`, `mcp-integration/*`, `sdk/*/README.md`) | [중요] | `DOCS_GRAPH.md` §5 priority 1 | Open |
| E-02 | `contracts/ethereum/docs/GOVERNANCE-SETUP.md` assumes Ownable ERC-8004 registries (see C-01) | [중요] | §5 priority 1 | Open |
| E-03 | `docs/INDEX.md`: 17 dead targets, missing newer docs, wrong `deployment-verify` description | [권장] | §5 priority 1 | Open |
| E-04 | Retire V2/V4-era documents (`pkg/agent/did/README.md`, `docs/did/*`, `docs/SAGE_A2A_INTEGRATION_GUIDE.md`, `docs/audit/*`, verification guides, `DETAILED_GUIDE_PART*`, test sections) | [권장] | §5 priority 2 | Open |
| E-05 | Archive planning-era design docs (`docs/dev/*`, `docs/planning/*`, optimisation plans) under `archive/` with dates | [권장] | §5 priority 3 | Open |
| E-06 | Write missing docs: commit-reveal client, `sage-did commit/activate/key/card`, `hpke` package API, session exporter path | [권장] | `DOCS_GRAPH.md` §6 | Open |
| E-07 | Fix 33 contradictions listed in `DOCS_GRAPH.md` §7 (CLAUDE.md handshake description, README code samples, test counts) | [권장] | §7 | Open |

## F. Strategy (multi-repository, protocol, SDKs)

| ID | Item | Severity | Source | Status |
|---|---|---|---|---|
| F-01 | `sage-spec`: RFC 9421 agent profile, canonicalisation rules, HPKE profile with mandatory AEAD, `did:sage` method, MCP binding, test vectors; `cmd/sage-vectors` generator | [중요] | `STRATEGY.md` §2.1, §5 step 2 | Decision (AEAD choice) |
| F-02 | `sage-contracts` extraction with ABI publishing and binding generation on tag | [권장] | §5 step 3 | Open |
| F-03 | `sage-core` (Rust): align session layer with the spec, vectors in CI, stable C header, `uniffi`/`wasm` packaging, provenance | [중요] | §2.3, §5 step 5 | Decision (owner) |
| F-04 | `sage-sdk-python` / `-typescript` / `-java` on the Rust core; archive current `sdk/*` | [권장] | §2.4, §5 step 6 | Open |
| F-05 | `sage-gateway`: MCP wrapper + HTTP proxy + A2A endpoint; client recipes for Codex / Claude Code / Hermes | [중요] | §2.5, §5 step 7 | Open |
| F-06 | Cross-repository version policy and compatibility matrix | [권장] | §6 | Open |

## Decisions still open

| Decision | Options | Recommendation |
|---|---|---|
| secp256k1 hash convention (B-03) | decided: Keccak-256, RFC 6979, r\|\|s\|\|v (Ethereum convention) for every secp256k1 path | record in `sage-spec` as the `es256k` profile; SHA-256 remains for P-256 only |
| Mandatory session AEAD (F-01) | AES-256-GCM vs ChaCha20-Poly1305 | AES-256-GCM mandatory (broadest library reach), ChaCha20 optional |
| Rust core owner (F-03) | maintainer / new contributor | required before step 5 |
| Kaia AgentCard address (C-03) | known / unknown | leave preset empty until confirmed |
| Go core licence (A-16) | keep LGPL-3.0 / Apache-2.0 | Apache-2.0 after contributor consent |
