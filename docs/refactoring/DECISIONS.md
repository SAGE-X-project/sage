# Refactoring decisions (proposal, 2026-09-11)

Answers to the five open questions in `REFACTORING_DESIGN.md` section 7, grounded in `FEATURE_MAP.md` (entry-point reachability) and an inspection of the Go repositories in the SAGE-X-project organisation that import this module. Each decision lists the evidence, the proposal, what it costs, and the alternative considered.

## Evidence that applies to every decision

- Two external Go consumers exist and build against current `main`: `sage-a2a-go` (requires v1.5.2) and `sage-multi-agent` (v1.3.1 with a local replace). They import `pkg/agent/{did,did/ethereum,crypto,crypto/keys,crypto/formats,crypto/storage,hpke,session,transport,transport/http,core/rfc9421}`. Symbols they depend on that the design planned to remove or change: `crypto.SetKeyGenerators/SetStorageConstructors/SetFormatConstructors`, `ethereum.EthereumClient`/`NewEthereumClient`, `did.Resolver` (satisfied by `EthereumClient`), `did.RegistryConfig`, `did.AgentMetadataV4`, `hpke.KeyIDBinder`, `hpke.CookieSource/CookieVerifier`.
- A third consumer, `sage-adk`, still imports the pre-v1.0 flat layout (`sage/crypto`, `sage/did`, `sage/config`) and has not compiled since the v1.0 move to `pkg/agent/` (2025-10-11). That move shipped without alias packages; the consumer was abandoned rather than migrated.
- Nothing outside this repository imports `pkg/agent/handshake`, `lib/`, `pkg/storage`, `pkg/oidc`, `pkg/health`, `pkg/version`, `core/message/*`, or `crypto/vault`.
- The module is versioned v1.x; Go semantic import versioning means an import-path change is a v2 module (`github.com/sage-x-project/sage/v2`), not a minor release.

---

## Decision 1. 4-phase `handshake`: remove

**Evidence.** No binary, example or external repository reaches `pkg/agent/handshake`; its only non-test importer is `internal/session_creator.go`, whose only user is `handshake/server_test.go`. Both external consumers establish sessions with `hpke` (1-RTT). The package has no replay check on Invitation/Request/Complete, starts a cleanup goroutine with no non-test stop, and duplicates `KeyIDBinder`, signature verification and session-config defaults. Docs already call HPKE the recommended path.

**Proposal.** Mark `pkg/agent/handshake` and `internal/session_creator.go` `Deprecated` in the next minor release (v1.6.0) and delete them in the one after (v1.7.0), together with `core/message` (its `ControlHeader` is implemented only by handshake messages) and `dedupe/order/validator`. Fold the useful part, the duplicated `KeyIDBinder`, into `session`.

**Cost.** Agents whose on-chain record has no KEM (X25519) key cannot use HPKE-Base and today could in principle use the 4-phase flow with only an Ed25519 signing key. No consumer does this, and `sage-did register` already supports KEM key registration, so the cost is documentation: "a KEM key is required for encrypted sessions".

**Alternative.** Harden and keep (add nonce/timestamp checks, `Stop()`, shared verifier): about the same effort as deleting, and leaves two protocols to maintain and audit for a path nobody uses.

---

## Decision 2. Import paths: keep `pkg/agent/*` (Option A), restructure inside it

**Evidence.** Two active consumers import 11 `pkg/agent/*` packages across 40+ files. The previous path move (flat -> `pkg/agent`, v1.0) broke `sage-adk` permanently. A flatten now would be a v2 module: every consumer must change its module path and every import line at once, and v1 keeps receiving dependabot/security updates in parallel until they do.

**Proposal.** Keep the current import paths for the whole plan. Apply Phases 0-3 and 5-6 unchanged; execute Phase 4 only for packages that are new or private (`pkg/agent/did/a2a`, `internal/{app,cli,config,testutil}`, `pkg/blockchain/ethereum`). Treat the naming overlap (`core/message` vs `transport`, the `agent/` prefix) as accepted debt, and record it in `docs/ARCHITECTURE.md`. Revisit a `/v2` flatten only if a breaking API redesign of `did` is scheduled anyway; then ship it as one v2 release with a migration guide, not as alias packages.

**Cost.** The tree keeps a redundant `agent/` level and the `core/` grouping that no longer means "core" once `message/*` is gone.

**Alternative.** Option B now (flatten + alias packages for one release). Rejected: alias packages cannot cover interface-implementer relationships across the old and new paths without duplicating types, and the v1.0 precedent shows downstream repos are not migrated promptly.

**Consequence for the plan.** Three items in the design must become deprecations instead of deletions, because external code uses them:

| Symbol | External use | Change |
|---|---|---|
| `crypto.SetKeyGenerators`, `SetStorageConstructors`, `SetFormatConstructors` | `sage-a2a-go` (3 files) | keep as `Deprecated` no-ops for two releases after `crypto.Manager`/`internal/cryptoinit` are removed; document `keys.Generate*` |
| `ethereum.EthereumClient`, `NewEthereumClient` | both consumers, as `did.Resolver` and for `RegistryConfig` | keep the type; reimplement it as a wrapper over `AgentCardClient` so reads and writes both target AgentCardRegistry (see Decision 5); mark `Deprecated` in favour of `NewAgentCardClient` |
| `did.Resolver` (6 methods) | both consumers pass an `EthereumClient` where a `Resolver` is expected | trim only by adding the new small interfaces (`Registry`, `Resolver`, `Lister`) and making `EthereumClient`/`AgentCardClient` satisfy both old and new for one release |

---

## Decision 3. `lib/` (cgo): remove

**Evidence.** `lib/export.go` exports `SageVersion` (hard-coded 1.3.1), `SageInit` and `SageCleanup` (no-ops). No C header is generated, no consumer exists in the repository or the organisation, CI never builds it, and `Dockerfile:30` runs `make build-lib || true`. `docs/dev/architecture.md` describes `libsage_crypto` as a separate Rust project; the Rust SDK builds its own `libsage_client.so`. `docs/BUILD.md` documents `libsage.a/.h/.so` outputs that do not correspond to any exported API.

**Proposal.** Delete `lib/`, the 15 `build-lib*` Makefile targets, the `package` dependency on `build-lib-all`, the Dockerfile line, and the `docs/BUILD.md` section. If a C ABI is wanted later, write the exported function list first (sign, verify, resolve, session open/seal/open) and add it as a new `lib/` with a CI job on Linux.

**Cost.** The release bundle no longer contains a (useless) static library; `docs/BUILD.md` and `INSTALL.md` need a paragraph removed.

**Alternative.** Keep and implement a real API now. Rejected: no consumer has asked for it, and the multi-language SDKs (Decision 4) do not link it.

---

## Decision 4. Multi-language SDKs: mark experimental, stop claiming interop

**Evidence.** Python, Rust and Java clients call `/debug/kem-pub`, `/debug/server-did`, `/debug/register-agent`, `/health` and `/v1/a2a:sendMessage`; no Go code in this repository or in any organisation repository serves those routes (`api/openapi.yaml` describes them, nothing implements them). None of the four implements RFC 9421; their "HPKE" is an ad-hoc X25519+AES-GCM construction and the Java variant uses the raw shared secret as the key, so they do not even interoperate with each other. TypeScript ships all-zero session keys at version 1.0.0 and has no tests. None is built or tested in CI. Meanwhile the Go consumers (`sage-a2a-go`) are the SDK that is actually used.

**Proposal.** In v1.6.0: (a) add an "Experimental, not interoperable with the Go core" banner to each SDK README and to the "Multi-Language Bindings" section of the root README, removing the RFC 9421 / HPKE / blockchain claims; (b) set TypeScript to 0.x; (c) add the cheap CI jobs (`pytest`, `cargo test`, `mvn test`, `vitest --passWithNoTests`) so the code at least stays buildable; (d) keep the code in place. Do not implement interop until the missing server exists: the right sequence is Go reference server implementing `api/openapi.yaml` (or A2A over `sage-a2a-go`) first, then one SDK (Python, the most complete) against it, then the rest or deletion.

**Cost.** Marketing claims shrink; users who assumed the SDKs worked against a SAGE server learn that no such server ships. That is already true today.

**Alternative.** Implement RFC 9421 + real HPKE in four languages now. Rejected: without a server contract the work has nothing to test against, and the effort (four crypto stacks) exceeds the rest of the refactoring combined.

---

## Decision 5. Legacy `SageRegistryV2`: drop support, keep the client type as a wrapper

**Evidence.** README lists the Sepolia `SageRegistryV2` deployment under "Legacy Contracts" and says "deprecated, use AgentCardRegistry". No V2 Solidity source exists in `contracts/`; only a flattened verification note remains. `deployments/config` still defaults the Kaia network to a V2 address (`0x4Ba6…`) under the JSON key `SageRegistryV2`, and `cmd/deployment-verify` reads that key. `EthereumClient` loads the AgentCard ABI but packs V2 function signatures, so it can read AgentCardRegistry records and write to neither contract. Both external consumers construct `EthereumClient`, mostly to obtain a `did.Resolver`.

**Proposal.** Support AgentCardRegistry only. Reimplement `EthereumClient` as a thin, `Deprecated` wrapper over `AgentCardClient` (reads delegate to `GetAgentByDID`; `Register` performs commit-reveal or returns `ErrUseAgentCardClient` with a clear message). Delete `SageRegistryABI`/`SageRegistryV2.abi.json`, rename the config key to `AgentCardRegistry`, and replace the Kaia V2 default with either the AgentCard deployment address on Kaia or no default (explicit configuration required) if none exists. Update README "Legacy Contracts" to "not supported by the Go client since v1.6".

**Cost.** Anyone still pointing the Go client at the V2 Kaia or Sepolia contract loses reads (writes never worked). The Kaia AgentCard address must be confirmed before the preset is filled; until then Kaia requires explicit config.

**Alternative.** Keep a real V2 client bound to `SageRegistryABI`. Rejected: no V2 source in the repo to regenerate bindings from, no test coverage, and the README already declares it deprecated.

---

## Resulting plan adjustments

| Design item | Before | After these decisions |
|---|---|---|
| Phase 1.1 (`crypto.Manager`, `cryptoinit`) | delete | delete `Manager`/`cryptoinit`; keep `Set*Constructors` as deprecated no-ops for two releases |
| Phase 3.1 (`did` interfaces) | delete `Client`, `RegistryV4`, `ClientFactory`, `EthereumClient`, mock `Resolver` | delete `Client`, `RegistryV4`, `ClientFactory`, mock `Resolver`; keep `EthereumClient` as deprecated wrapper; keep `did.Resolver` shape for one release alongside the new small interfaces |
| Phase 4.3 (flatten) | optional | not done; paths stay `pkg/agent/*` |
| Phase 5 (`handshake`) | decide | deprecate v1.6.0, delete v1.7.0 |
| Phase 5 (`lib/`) | decide | delete in v1.6.0 |
| SDKs | decide | experimental banner + CI build jobs in v1.6.0; interop deferred until a server exists |
| Config presets (Phase 2.1) | add Sepolia | add Sepolia AgentCard addresses from README; Kaia preset only if an AgentCard deployment address is confirmed |

Release shape: v1.6.0 = Phase 0 defect fixes + deprecations + SDK banners + lib removal; v1.7.0 = Phases 1-3 + handshake removal; v1.8.0 = Phase 5 remainder + Phase 6.

## Facts vs. opinions

**Fact**
- `sage-a2a-go` (non-example packages) and `sage-multi-agent` compile against current `main` with a local replace; their imported SAGE symbols are listed in `FEATURE_MAP.md`.
- `sage-adk` imports `github.com/sage-x-project/sage/{crypto,did,core,config}`, paths that do not exist since v1.0.
- No repository in the organisation serves `/debug/kem-pub`, `/debug/server-did`, `/debug/register-agent` or `/v1/a2a:sendMessage`.
- No repository in the organisation imports `pkg/agent/handshake` or links `lib/`.
- `contracts/` contains no `SageRegistryV2` source; README marks the V2 deployment as deprecated.

**Opinion**
- [High] Removing `handshake` and `lib/` has no functional cost for any current consumer.
- [High] A `/v2` path flatten would repeat the v1.0 breakage for `sage-a2a-go` and `sage-multi-agent`; keeping paths is the right call while those repositories are the main users.
- [Mid] Wrapping `EthereumClient` over `AgentCardClient` preserves compilation for both consumers; behavioural differences in `Register` (commit-reveal needs three transactions) may still require small changes on their side.
- [Mid] The SDKs will not become interoperable without a reference server; marking them experimental is the honest state until that server is scoped.
- [Low] A Kaia AgentCard deployment may exist outside this repository; if so the preset can be filled in Phase 2.1.

---

## Decision 6 (2026-09-11). secp256k1 signatures use the Ethereum convention everywhere

**Maintainer decision.** SAGE must support Ethereum-based chains, so secp256k1 is the primary key type and its signatures must be verifiable by Ethereum tooling (`ecrecover`, wallets).

**Rule.** For secp256k1 keys every SAGE path hashes the message with Keccak-256, signs with RFC 6979 deterministic ECDSA, and emits `r || s || v` (65 bytes); verifiers accept 64-byte `r || s` as well. P-256 keeps SHA-256 with raw `r || s`. One implementation, `keys.SignSecp256k1Keccak` / `keys.VerifySecp256k1Keccak`, is used by `KeyPair`, the RFC 9421 envelope verifier, the RFC 9421 HTTP signer/verifier and (already) the HPKE verifier.

**Consequence for the spec.** The `es256k` entry of the algorithm table in `sage-spec` is defined as this profile; test vectors for it are generated from the Go implementation and must be reproduced by the Rust core.

