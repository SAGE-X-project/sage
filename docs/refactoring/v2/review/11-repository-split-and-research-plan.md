# 11. Repository split and research plan

Status: planning document, 2026-09-12. No code was changed. Sources: `sage`
main `e98b42b` (`go list ./...`, `wc -l`), `sage-spec` `a34dd49`,
`sage-contracts` `d9f313b`, `sage-gateway` `4e72668`, `sage-inspector`
`05b890d`, `rs-sage-core` `206bbbb` (local checkouts, read-only), and the
documents `../REPO_PLAN.md`, `../VERSION_POLICY.md`, `../LICENSING.md`,
`../../STRATEGY.md`, `../../REFACTORING_DESIGN.md`, `../../DECISIONS.md`,
`../../BACKLOG.md`, `02-code-graph.md`, `03-history-timeline.md`,
`07-literature-review.md`, `08-implementation-evaluation.md`,
`09-protocol-flows.md`, `10-inspector-test-matrix.md`. Confidence marks:
[High] read in the cited file or measured; [Mid] one inference from cited
evidence; [Low] estimate. Effort figures are [Low] unless stated.

Goal restated: split the remaining monolith so that (a) each repository is
usable alone, (b) protocol behaviour is shared through the spec, vectors and
one FFI/WASM substrate instead of being copied, and (c) the resulting set can
be evaluated in a paper on efficiency and security.

---

## 1. Current state

### 1.1 What has already moved

| Repository | Content | Moved by | State (2026-09-12) |
|---|---|---|---|
| `sage-spec` | `spec/00-08`, `vectors/{crypto,jcs,rfc9421,hpke,session,did}.json` (26 vectors), Apache-2.0 | sage `#303` (`9827cec`, generator), `#305` (`c2e3603`, CI check), sage-spec `#1..#3` (`9f7c918`..`a34dd49`) | `1.0.0-draft.1`, untagged; Go and Rust CIs check out `main` (`VERSION_POLICY.md` §2 "to fix") |
| `sage-contracts` | `ethereum/` (Hardhat, 219 tests), `solana/`, `abi/` (10 ABIs), MIT | sage `#306` (`b04477b`); sage-contracts `ef304f3` (`git subtree split` of `contracts/`), `#1` (`d9f313b`) | untagged; `sage/.contracts-version` = `2ff992e`; `make bindings-check` drift gate |
| `sage-gateway` | `pkg/gateway/{keyfile,resolve,sign,verify}`, `cmd/sage-gateway`, recipes, LGPL-3.0 | sage-gateway `#1` (`4e72668`) | pins `sage v1.5.3-0.20260912041026-b04477bc0840` (`sage-gateway/go.mod:10`); RFC 9421 only, no HPKE (`08` §3 "Exercised") |
| `sage-inspector` | `pkg/inspect/{card,http,p256,vectors}.go`, `cmd/sage-inspector`, LGPL-3.0 | sage `#307` (`5ab9c7e`, `pkg/vectors` exported), sage-inspector `#1` (`05b890d`) | pins `sage v1.5.3-0.20260912042550-5ab9c7e46ef3` (`sage-inspector/go.mod:10`); no HPKE/session inspection (`10` §1 item 6) |
| `rs-sage-core` | crate `sage_crypto_core` 0.3.0, `include/sage_crypto.h` (cbindgen, CI drift check), `pkg/*.wasm` + `.d.ts`, MIT OR Apache-2.0 | rs-sage-core `#18`, `#23`, `#24`, `#25` (`aa57ab0`) | all 26 vectors pass in CI; 52 `extern "C"` entries, HPKE not exported, no `catch_unwind` (`08` §7); live Go/Rust exchange open (BACKLOG F-03b) |
| removed from `sage` | `lib/` (cgo), `cmd/random-test` + `tests/random`, `core/message/{dedupe,order,validator}`, `contracts/` | `#295` (`eff53f8`), `#306` | `handshake` and `internal/sessioninit` marked `Deprecated` (`pkg/agent/handshake/doc.go:22`, `internal/session_creator.go:22`) |

### 1.2 What still lives in `sage`

Measured with `find ... -name '*.go' | xargs wc -l` on `e98b42b` (non-test LOC
/ test LOC / files); cluster names follow `02` §1. The `go list` output has 60
packages, two of which are noise (`reports/bindings`, a 5,554-line exact copy
of the generated bindings, and `sdk/typescript/node_modules/.../flatted`;
`02` §1).

| Cluster | Packages | LOC (src / test / files) | Role today | Destination (section 2) |
|---|---|---|---|---|
| crypto | `pkg/agent/crypto`, `/keys`, `/formats`, `/storage`, `/rotation`, `/chain{,/ethereum,/solana}`, `/jcs`, `/vault` | 5,130 / 7,575 / 52 | primitives, key files, JCS, chain tables; `vault` has no importer (`08` §1.7) | `sage`; `vault` deleted |
| verification | `pkg/agent/core` (facade, 317), `core/rfc9421` (2,224), `core/message` (42), `core/message/nonce` (147) | 2,730 / 5,367 / 26 | RFC 9421 profile; `nonce` reachable only via deprecated `NewVerifierWithNonceManager` (`02` §2a) | `sage`; `message`, `nonce` deleted |
| identity | `pkg/agent/did`, `/ethereum`, `/solana` | 4,746 / 6,803 / 47 | DID grammar, resolver, key policy, A2A card, commit-reveal client | `sage` |
| channel | `pkg/agent/hpke`, `/session`, `/transport{,/http,/websocket}` | 4,802 / 6,696 / 43 | 1-RTT handshake, record layer, wire codec; no `cmd/` caller (`02` §2e) | `sage` |
| legacy | `pkg/agent/handshake`, `internal/session_creator.go` | 1,130 / 825 / 9 | 4-phase handshake, deprecated | deleted (D-06) |
| bindings | `pkg/blockchain/ethereum` (+ generated `contracts/agentcardregistry`) | 5,909 / 592 / 5 | RPC provider + abigen output from `sage-contracts/abi` | `sage` now; `sage-contracts` Go module later (step 8) |
| platform | `pkg/storage{,/memory,/postgres,/storagetest}`, `pkg/telemetry`, `pkg/health`, `pkg/oidc{,/auth0}`, `pkg/version` | 3,841 / 1,495 / 44 | `storage` and `oidc` have zero in-module consumers (`REFACTORING_DESIGN.md` §2); `health` used by `sage-verify` | `telemetry`, `version` stay; `health` to CLI; `storage`, `oidc` decision (section 6) |
| vectors | `pkg/vectors`, `cmd/sage-vectors` | 1,567 / 117 / 9 | generator and checker for `sage-spec` | `sage` (reference core generates) |
| CLI | `cmd/sage-crypto` (1,182), `cmd/sage-did` (2,962), `cmd/sage-verify` (410), `internal/{app,cli,config}` (1,356), `deployments/config` YAML | 5,910 / 1,126 / ~50 | binaries and composition root | `sage-cli` (new) |
| examples | `examples/{mcp-integration,a2a-integration,metrics-demo,...}` | 3,804 / 0 / 32 | demos; `a2a-integration` is `//go:build ignore` (`02` §1) | `sage-examples` (new) or nested module |
| tooling | `tools/codegraph` (1,124), `tools/benchmark` (3 bench files), `tools/loadtest` (k6), `tools/analyze` (243) | 1,367 / 618 / 52 | layer gate, benchmarks | `codegraph` stays; benchmarks and load test to `sage-bench` |
| tests | `tests/integration`, `internal/testutil` | 556 / 5,835 / 17 | Hardhat-backed integration tests | `sage` |
| SDK stubs | `sdk/{python,typescript,java,rust}` | 5,383 source lines (all languages) | experimental, not interoperable (D-08; `08` §7.4) | archived, replaced by `sage-sdk-*` on `rs-sage-core` |

### 1.3 Dependency direction between repositories

Arrows point from consumer to provider. Solid edges are `go.mod` pins or
committed refs; dotted edges are CI checkouts (`actions/checkout` of
`sage-spec` at `main`, `.github/workflows/test.yml:178-192`) or planned.

```mermaid
graph LR
  SPEC["sage-spec\n(text + vectors)"]
  CON["sage-contracts\n(Solidity, ABI)"]
  GO["sage\n(Go core, sage-vectors)"]
  RS["rs-sage-core\n(crate, C header, WASM)"]
  GW["sage-gateway"]
  INS["sage-inspector"]
  CLI["sage-cli (planned)"]
  EX["sage-examples (planned)"]
  BENCH["sage-bench (planned)"]
  SDK["sage-sdk-* (deferred)"]
  A2A["sage-a2a-go, sage-multi-agent\n(external, build on main)"]
  ADK["sage-adk\n(external, pre-v1.0 paths, broken)"]
  GO -. vectors CI .-> SPEC
  RS -. vectors CI .-> SPEC
  INS -. vectors CI .-> SPEC
  GO -->|.contracts-version 2ff992e| CON
  GW -->|go.mod b04477b| GO
  INS -->|go.mod 5ab9c7e| GO
  CLI -.-> GO
  CLI -.-> CON
  EX -.-> GO
  EX -.-> GW
  BENCH -.-> GO
  BENCH -.-> RS
  BENCH -.-> GW
  SDK -.-> RS
  A2A --> GO
  ADK -. replace ../../sage .-> GO
```

Facts behind the graph: `sage-a2a-go` and `sage-multi-agent` import eleven
`pkg/agent/*` packages and build against `main` (`DECISIONS.md` "Evidence");
`sage-adk/go.mod:10,75` requires `sage v0.0.0` with
`replace github.com/sage-x-project/sage => ../../sage` and its code imports
`github.com/sage-x-project/sage/{crypto,did,config,core/rfc,...}`, paths that
have not existed since v1.0.0 (`DECISIONS.md` "Evidence"; verified with grep
on the local checkout). No repository imports in the reverse direction
(`REPO_PLAN.md` §3). [High]

---

## 2. Target repository set

Licences follow `LICENSING.md` §4 (decided A-16: Go repositories stay
LGPL-3.0). Versioning follows `VERSION_POLICY.md` §1 and §4. "Owns spec
chapters" means the repository is the implementation the chapter is derived
from or the conformance level it must pass (`sage-spec/spec/00-overview.md`
§4: Verifier = crypto, jcs, rfc9421, did; Peer = Verifier + hpke, session;
Reference = all incl. verify-only regeneration). `sage-spec` owns every
chapter's text; the column lists which implementation is normative for it.

| Repository | Purpose (one sentence) | Public surface | Must not depend on | Licence | Versioning | Spec chapters | Reuse without the rest |
|---|---|---|---|---|---|---|---|
| `sage-spec` | Normative wire formats and golden vectors that every implementation must reproduce. | `spec/*.md`, `vectors/*.json`, planned `labels.json` and `formal/` | any code repository | Apache-2.0 | protocol SemVer, `1.0.0-draft.N` until frozen | all (text) | a third-party implementer in any language; a reviewer checking a paper's claims |
| `sage` | Go reference core: the library the vectors are generated from. | `pkg/agent/{crypto/**,core,core/rfc9421,did/**,hpke,session,transport/**}`, `pkg/blockchain/ethereum`, `pkg/vectors`, `pkg/telemetry`, `pkg/version`; `cmd/sage-vectors` | `internal/` of anything; `examples/`, `tests/`, `tools/` (depguard, `.golangci.yml:26-48`); no `cmd` other than the generator | LGPL-3.0 | Go module tags `v1.x`, no `/v2`; deprecations live two minors (§4 rule 3) | Reference for 01-08; normative where the text is silent (00 §6) | `sage-a2a-go`-style Go agents; any Go verifier; the gateway |
| `rs-sage-core` | Portable core with a C ABI and WASM build, the substrate for every non-Go SDK. | crate `sage_crypto_core`, `include/sage_crypto.h`, `pkg/sage_crypto_core{.js,.d.ts,_bg.wasm}` | on-chain reads (drop `blockchain`, `REPO_PLAN.md` §2), transports, the Go core | MIT OR Apache-2.0 | Cargo `0.x` until F-03b; C signature change = minor (§4 rule 4) | Peer level 01-07; not 08 (transport removed) nor 06 §3 on-chain resolution (host supplies a resolver) | Rust agents; Python/TS/Java via bindings; browser via WASM |
| `sage-contracts` | Registry, ERC-8004 and governance contracts with published ABIs and deployment records. | `abi/*.json`, `ethereum/deployments/*`, ABI tarball on `v*` tags; planned `bindings/go` module and `deployments/<network>.json` | either core | MIT | ABI SemVer; first tag `v1.5.0` (§5) | 06 §3 record shape (registry behaviour is out of spec scope, 00 §1) | contract auditors; a Go or Rust consumer that only needs bindings and addresses |
| `sage-gateway` | Zero-integration verifying/signing proxy and MCP wrapper for agent clients. | binary, container image; `pkg/gateway/*` (Go, importable but not stable) | `rs-sage-core`; `sage` `internal/` | LGPL-3.0 | tracks `sage` tags, never `replace` (§4 rule 5) | Verifier level: 03, 06 §3, 08 §3 | Codex / Claude Code / Hermes users with config only (`STRATEGY.md` §2.5) |
| `sage-inspector` | Conformance judge: vector runner, mutation suite, message inspector, cross-implementation diff. | binary; `pkg/inspect` | writing to any repository; network by default | LGPL-3.0 (Apache-2.0 possible if the `sage` import is dropped, `LICENSING.md` §4) | tracks `sage` tags | checks 01-03, 06 §4, 07 today; 04, 05 planned (`10` §3 P1) | every repository's CI; third-party implementers |
| `sage-cli` (new) | Operator binaries for keys, DID registration and deployment checks. | `sage-crypto`, `sage-did`, `sage-verify` binaries; GoReleaser artefacts | being imported by `sage`, `sage-gateway`, `sage-inspector` | LGPL-3.0 | tracks `sage` tags | none (06 §4 PoP and registration are exercised, not owned) | operators who never write Go |
| `sage-examples` (new, or nested module) | Runnable demos whose READMEs are tested in CI (E-01). | `go run` targets, README recipes | being imported | Apache-2.0 (copied into user code; links LGPL `sage`, see section 6 item 8) | untagged; pins `sage` and `sage-gateway` | none | newcomers; paper artefact appendix |
| `sage-bench` (new) | Research harness: benchmarks, interoperability matrix, adversarial suites, datasets and results. | `go test -bench` packages, `criterion` links, scenario scripts, CSV/JSON results, `Makefile` | being imported; production code | Apache-2.0 (same note) | untagged; results directories named by experiment and commit | none | reviewers reproducing the paper; other protocols reusing the baselines |
| `sage-sdk-{python,typescript,java}` (deferred, F-04) | Thin generated bindings over `rs-sage-core` plus HTTP/MCP middleware. | language packages | `sage` (Go); re-implemented crypto | MIT OR Apache-2.0 | package SemVer declaring the `rs-sage-core` version | Verifier or Peer via the substrate | application developers in each language |

### Split of `sage` itself

| Goes to | Packages / paths | Notes |
|---|---|---|
| stays: `sage` core library | `pkg/agent/crypto/{keys,formats,storage,rotation,chain/**,jcs}`, `pkg/agent/crypto` (contracts, algorithm table), `pkg/agent/core` + `core/rfc9421`, `pkg/agent/did/**`, `pkg/agent/hpke`, `pkg/agent/session`, `pkg/agent/transport/**`, `pkg/blockchain/ethereum/**`, `pkg/vectors`, `cmd/sage-vectors`, `pkg/telemetry`, `pkg/version`, `internal/testutil`, `tests/integration`, `tools/codegraph`, `tools/scripts` | Import paths unchanged (D-02). `pkg/telemetry` stays because Go links per package: a consumer that never imports it does not link Prometheus, although `go.sum` lists it [High]. `internal/app.RegisterDefaults` (43 LOC) stays only if `pkg/vectors` or the integration tests need it; otherwise it moves with the CLI [Mid]. |
| `sage-cli` | `cmd/sage-crypto`, `cmd/sage-did`, `cmd/sage-verify`, `internal/cli`, `internal/config`, `internal/app` (copy), `deployments/config/*.yaml`, `deployments/docker`, `pkg/health` (after a deprecation cycle), `scripts/` that only drive binaries | 5,910 non-test LOC, ~50 files. `pkg/health` has no external importer (`DECISIONS.md` "Evidence") but is exported; mark `Deprecated` in 1.6.0, delete from `sage` in 1.8.0. |
| `sage-examples` | `examples/**`, `examples/run-examples.sh` | 3,804 LOC, 32 files. Alternative: keep in `sage` as a nested module (`examples/go.mod`) so the main module stops carrying their imports (section 6 item 5). |
| `sage-bench` | `tools/benchmark`, `tools/loadtest`, `tools/analyze`, the `Benchmark*` functions listed in section 5 that are cross-implementation rather than unit-level | Unit benchmarks (`pkg/agent/hpke/hpke_bench_test.go`, 6 functions; `did/performance_test.go`, 8) stay with their packages. |
| archived (branch `archive/sdk-2026-09`, tag, then deleted) | `sdk/{python,typescript,java,rust}`, `api/openapi.yaml`, `sdk.yml` workflow | `STRATEGY.md` §2.4: "archived, not migrated: none of them implements the protocol"; `08` 7.4 counts four crypto re-implementations, one diverged (SHA-256 secp256k1). |
| deleted | `pkg/agent/handshake` (976 + 825 test), `internal/session_creator.go` (154), `pkg/agent/core/message` (42), `core/message/nonce` (147 + 362), `pkg/agent/crypto/vault` (407 + 324), `reports/bindings` (5,554), `keys.{Encrypt,Decrypt}WithEd25519Peer` (`02` §2c) | `02` §4 inventory and D-06. `vault` is "used by the keys tests" (BACKLOG D-06), so its deletion needs those tests rewritten first. `DeriveSessionSeed` / `NewSecureSessionWithParams` stay in `session` because `pkg/vectors/session.go:55,198` use them (`02` §2c). After deletion `hpke.KeyIDBinder` has no in-tree implementer; the server already generates `kid-<uuid>` itself (`server.go:329`). |

---

## 3. Code-reuse mechanisms

Principle (`STRATEGY.md` §2 "protocol first, cores second, bindings third"):
behaviour is shared by making both cores reproduce the same bytes, not by
sharing source. Where sharing source is possible (SDKs over one substrate)
it is done; where it is not (Go vs Rust), the duplicate is pinned by a
machine-checked artefact. The table lists every duplication found in `02`,
`08`, `09` and `10` and the mechanism that removes or pins it.

| # | Duplicated today | Where (Go / Rust / other) | Mechanism | Repository hosting the artefact | State |
|---|---|---|---|---|---|
| 1 | Domain-separation labels, suite ids, header names, JSON member names | `pkg/agent/hpke/common.go:39-59`, `session/session.go:365-431` / `rs/hpke/types.rs:55-65`, `rs/session/secure_session.rs:44-46` / gateway `sign.go:43` covered set | `sage-spec/labels.json` (one JSON: labels, suite strings, header names, task ids, default covered set) plus a test in each core asserting equality with its constants; codegen only if the list grows past a few dozen entries | `sage-spec` | proposed; today pinned only indirectly by `hpke/info-and-export-context` and `session/directional-with-aad` vectors |
| 2 | JCS (RFC 8785) | `pkg/agent/crypto/jcs/jcs.go` (225 LOC) / `rs/jcs/mod.rs`; Go accepts duplicate keys and lone surrogates that Rust rejects (`08` §5.1, `10` x-12) | (a) `jcs/edge-cases` vector with a `rejected` list (`10` §3), then (b) replace both with `gowebpki/jcs` and `serde_jcs` (`08` §5 recommendation, about 600 lines removed) | `sage-spec` (vector), both cores | proposed |
| 3 | Signature verification conventions (secp256k1 Keccak, low-S, DER, key id) | `keys.VerifySignature`, `keys/keyid.go:33-36` (compressed key for secp256k1 `ID()`, `08` §1.3) / `rs/crypto/signature.rs`, `keys.rs:100-107` | `crypto` vectors already pin sign/verify; add `der-encoded`, `low-s-boundary`, `secp256k1-leading-zero`, `pop-secp256k1-high-s` (`10` §3); fix Go `ID()` to hash the uncompressed key | `sage-spec`, `sage` | vectors exist for the positive path |
| 4 | RFC 9421 canonicaliser and Structured Fields parser | `rfc9421/parser.go`, `canonicalizer.go:232-241,300-326` / `rs/rfc9421/dictionary.rs:59-66`, `canonicalize.rs:60-67`; two interop-breaking deviations in Go (`08` §2.1, §2.2) | fix Go (`@request-target` without method; raw `Signature-Input` member in the base), add `rfc9421/request-rejected` vector list; optionally adopt an RFC 8941 parser | `sage`, `sage-spec` | open; must precede spec 1.0 (`08` §2 recommendation) |
| 5 | Replay guards | Go `session.ReplayGuard` (one contract after `#281`) / Rust `rfc9421/replay.rs` and `hpke/nonce_store.rs` (two) | behaviour pinned by the inspector's replay-over-capture-set check (`10` §3 P1) and a `hpke/init-rejected` vector; a Rust unification to one guard mirrors `#281` | `sage-inspector`, `sage-spec`, `rs-sage-core` | proposed |
| 6 | DID resolution | Go on-chain `AgentCardClient` / Rust `BlockchainDIDResolver` returns `Unsupported` (`08` §6) | not a duplication to keep: the C ABI takes a host-supplied resolver callback (`rs/hpke/types.rs` `DIDResolver`), so on-chain reads exist once, in Go and in the gateway; SDKs call the gateway or a host resolver | `rs-sage-core` (callback), `sage-gateway` (resolver + cache) | design decision recorded in `REPO_PLAN.md` §2 ("drop blockchain") |
| 7 | Contract bindings and addresses | `pkg/blockchain/ethereum/contracts/agentcardregistry` (abigen from pinned ABI), `reports/bindings` (copy), `chain.Presets` addresses, `sage-contracts/README.md` address table; `rs-sage-core/contracts/` directory exists (not inspected) [Mid] | delete `reports/bindings`; `sage-contracts` publishes `deployments/<network>.json` and, later, a Go module `bindings/go` regenerated on tag (`STRATEGY.md` §2.6) so bindings are build outputs consumed at a tag; Rust binding directory removed with the `blockchain` module | `sage-contracts` | `abi/` and drift check done (F-02); bindings module proposed |
| 8 | Vector generation and checking | Go `pkg/vectors` check is self-referential (`08` §8.2); Rust `tests/spec_vectors.rs` consumes Go output; no Rust-to-Go direction (`08` §8.1) | `sage-vectors check` accepts a foreign generator's files; `sage-spec` CI job runs Go-gen -> Rust-check and Rust-gen -> Go-check; inspector "cross-implementation mode" over FFI (`10` §3 P3) | `sage-spec` (CI), `sage-inspector` | one direction today |
| 9 | SDK substrate | four hand-written SDK crypto stacks in `sdk/*` (`08` §7.4) vs `rs-sage-core` C header (52 functions) and WASM package | SDKs become generated bindings: `uniffi` (Python, Kotlin, Swift), `wasm-bindgen` output already in `pkg/` (TypeScript), `jextract`/Panama from `sage_crypto.h` (Java); preconditions: `catch_unwind` on every entry (`08` 7.2), HPKE and session exported (`08` §7 "Not exposed"), 0600 key files (`08` 1.1) | `rs-sage-core`, `sage-sdk-*` | header and WASM exist; hardening open |
| 10 | Conformance runner in every CI | Go: `sage-vectors check`; Rust: own harness; gateway and inspector: checkout of `sage-spec` `main` | one step everywhere: `sage-inspector vectors -dir <spec>/vectors` (plus mutation suite) pinned to a spec tag; `rs-sage-core` runs it in FFI mode | `sage-inspector` | needs `10` §3 P1 items first |
| 11 | Compatibility matrix and `spec:` declaration | `VERSION_POLICY.md` §3 (one row) | move the matrix to `sage-spec/README.md` once it has two rows; each README states the spec version whose vectors its CI runs (§5 open actions) | `sage-spec`, every implementation | open |
| 12 | `AgentMetadata`, DID grammar, key id, wire codec, chain presets inside Go | seven metadata types, six DID parsers, nine key-id copies before the cycle (`REFACTORING_DESIGN.md` §3.3) | done inside `sage` (`#280`, `#282`, `#283`, `#316`); the single owners in §3.3 are the ones the CLI split must import rather than copy | `sage` | done |

Duplication that is accepted: two protocol cores (Go and Rust). The cost is
stated in `STRATEGY.md` §2.3 and §10; the mitigation is rows 1-5, 8 and 10
above plus the "spec first" rule (`VERSION_POLICY.md` §4 rule 1). [High]

---

## 4. Migration plan

Ordering rule: nothing is copied while broken (`STRATEGY.md` §5), external
consumers keep compiling, and each step is a separate PR series gated by CI.
Effort assumes one maintainer familiar with the tree. "Vectors" = `sage-spec`
vectors pass; "inspector" = `sage-inspector vectors` and the request/response
checks pass; "interop" = gateway `client` <-> gateway `serve` exchange in CI.

| Step | Scope | Preconditions | Mechanical procedure | Shims for external consumers | Gates | Size | Effort |
|---|---|---|---|---|---|---|---|
| 0 | Pin the baseline | none | tag `sage-contracts v1.5.0`; tag `sage-spec v1.0.0-draft.1`; switch `.contracts-version` to the tag and the three `actions/checkout` refs to the spec tag; add `spec:` lines to four READMEs (`VERSION_POLICY.md` §5) | none | all four CIs green on pinned refs | config only | hours |
| 1 | Release `sage v1.6.0` | step 0 | add missing `Deprecated:` markers (`nonce.NewManager`, `message.ControlHeader`, `02` §4; `pkg/health`, `pkg/storage`, `pkg/oidc` if section 6 item 4 decides removal); record the removal horizon exception (item 1); tag from `main` after CI | the shims already merged: `crypto.Set*Constructors` no-ops, `ethereum.EthereumClient` wrapper, `did.Resolver` shape kept, `RegistryV4`/`AgentMetadataV4` aliases (`DECISIONS.md` decision 2 table) | vectors; `sage-gateway` and `sage-inspector` build against the tag; `sage-a2a-go` builds (`DECISIONS.md` "Evidence") | none moved | 1 day |
| 2 | Hygiene deletions | step 1 | delete `reports/bindings`; exclude `reports/`, `node_modules` in `tools/codegraph`; move `tools/{benchmark,loadtest,analyze}` to `sage-bench` (step 9) or delete `analyze` | none (no importer) | `make codegraph-check` shows 58 packages, 0 exact duplicates | 5,554 + 1,367 LOC | half a day |
| 3 | `sage-cli` extraction | step 1 | on a clone: `git filter-repo --path cmd/sage-crypto --path cmd/sage-did --path cmd/sage-verify --path internal/cli --path internal/config --path internal/app --path deployments --path pkg/health` (multi-prefix history; `subtree split` handles one prefix); new module `github.com/sage-x-project/sage-cli`, `go.mod` requires `sage v1.6.0`, no `replace` (§4 rule 5); GoReleaser config copied from `sage`; in `sage`: remove the three `cmd/` dirs, `internal/{cli,config}`, `deployments/config` Go code in the same PR that tags 1.7.0; `pkg/health` deleted in 1.8.0 | binaries keep their names; `sage` release notes and `INSTALL.md` point to `sage-cli`; a `cmd/sage-did` stub cannot stay in `sage` without an import cycle, so none is kept | `sage-cli` CI builds against the `sage` tag; `sage-did resolve/commit/register/activate` integration test on Hardhat from the pinned contracts; `sage` codegraph shows no `cmd` cluster except `sage-vectors`; inspector unchanged | 5,910 src LOC, ~50 files | 2-3 days |
| 4 | Examples | step 1 | either `git filter-repo --path examples` into `sage-examples` with its own `go.mod` pinning `sage` and `sage-gateway`, or add `examples/go.mod` in place; CI runs `run-examples.sh` | none | examples build against tags; E-01 README claims re-checked | 3,804 LOC, 32 files | 1-2 days |
| 5 | Deprecated-code removal, `sage v1.7.0` | steps 1-3; decision on item 1 | delete `handshake`, `internal/session_creator.go`, `core/message`, `core/message/nonce`, `crypto/vault` (rewrite the `keys` tests that use it), `keys.{Encrypt,Decrypt}WithEd25519Peer`; delete `rfc9421.NewVerifierWithNonceManager` with `nonce`; archive `sdk/*` (branch + tag) and delete; drop `sdk.yml` | none needed: zero external importers for every deleted symbol (`DECISIONS.md` "Evidence"); `NewVerifierWithNonceManager` is the one exported symbol removed one minor early | vectors; `sage-a2a-go` and `sage-multi-agent` build; codegraph dead-code list empty; `docs/handshake/handshake-*.md` archived (`03` §3 staleness table) | 3,237 LOC (1,720 src) + 5,383 SDK lines | 1-2 days |
| 6 | `rs-sage-core` substrate hardening and F-03b | step 0 | `catch_unwind` macro on every `extern "C"` entry; export HPKE client/server and session records over FFI and WASM; 0600 key files; `Zeroize` on `PrivateKey`; CI job: Go `sage-gateway client` signs -> Rust verifier; Rust `HpkeClient` -> Go `hpke.Server` over HTTP; tag `1.0.0` when green (§4 rule 4) | none (Rust has no downstream yet) | live interop job green; header drift check; vectors both directions (row 8) | Rust only | 1-2 weeks [Low] |
| 7 | Conformance suite as the shared CI step | step 0; `10` §3 P1 | inspector: `-did`/`-authority` inputs, replay over capture set, exact `;req` set, `init`/`ack`/`record` commands; `sage-spec`: `request-rejected`, `hpke/init-rejected`, `jcs/edge-cases`, `session/stale-and-order` vectors; every repository's CI adds `sage-inspector vectors` at the spec tag | none | the eleven Go/Rust verdict differences listed in `10` "Facts" each covered by a vector; both cores pass | inspector + vectors | 1 week |
| 8 | Bindings module in `sage-contracts` (optional) | step 0, tag `v1.5.0` | `sage-contracts/bindings/go/go.mod` (module `github.com/sage-x-project/sage-contracts/bindings/go`) generated on tag by the existing `gen-bindings.sh`; `sage` imports it and keeps `pkg/blockchain/ethereum/contracts/agentcardregistry` as alias declarations for two minors | alias package | `make bindings-check` becomes a version comparison; `did/ethereum` tests green | 5,909 generated LOC leave `sage` | 2-3 days |
| 9 | `sage-bench` | step 1 (library tag), step 6 for FFI-based experiments | new repository: `bench/` (Go `testing.B` packages importing `sage`, `crypto/tls`, a Noise library), `interop/` (matrix driver over gateway, Rust FFI, inspector), `adversarial/` (A2ABreak, MAESTRO, MCPTox scenario scripts), `chain/` (Hardhat and Sepolia measurement scripts), `results/<experiment>/<commit>/` | none | each experiment reproducible from a `Makefile` target; results committed with the commits of every repository used | new | 1-2 weeks for the harness skeleton |
| 10 | `sage-sdk-*` (F-04) and `sage v1.8.0` | step 6 green | generated bindings per `STRATEGY.md` §2.4; `sage v1.8.0` removes the externally used shims (table below) | migration notes in the 1.8.0 changelog: `keys.Generate*`, `NewAgentCardClient`, `did.Registry`/`Resolver`/`Lister` | SDK CI runs the inspector conformance step; Python SDK end-to-end through `sage-gateway` (`STRATEGY.md` §5 step 6 gate) | new | weeks [Low] |

### Removal schedule aligned with `VERSION_POLICY.md`

Rule applied: §4 rule 3 (removal two minors after the marking release). Every
marker on `main` ships in 1.6.0, so the policy removal is 1.8.0 (`02` §4).
BACKLOG D-06 ("deprecate v1.6, remove v1.7") and two code comments say
1.7.0. This plan takes 1.7.0 only for symbols with zero external importers
and asks the maintainer to record that as an explicit exception in
`VERSION_POLICY.md` §4 (section 6 item 1); otherwise everything below moves
to 1.8.0.

| Release | Removed | Basis |
|---|---|---|
| v1.6.0 (tag current `main`) | nothing; all `Deprecated:` markers ship; CLI, examples, SDK stubs still present | policy rule 3; `CHANGELOG.md` "[Unreleased]" |
| v1.7.0 | `pkg/agent/handshake`, `internal/session_creator.go`, `core/message`, `core/message/nonce`, `rfc9421.NewVerifierWithNonceManager`, `crypto/vault`, `reports/bindings`, `sdk/*`, `cmd/{sage-crypto,sage-did,sage-verify}` + `internal/{cli,config,app}` (moved to `sage-cli`), `examples/` (moved) | D-06; zero external importers (`DECISIONS.md`); `cmd`/`internal` are not importable API |
| v1.8.0 | `crypto.Set*Constructors`, `crypto.New*`/`Generate*` wrappers, `rfc9421.NonceReplayGuard`, `did.GetRecommendedKeyType`/`ValidateKeyTypeForChain`/`GetRFC9421Algorithm`, `did.RegistryV4`, `did.AgentMetadataV4` + `ToAgentMetadata`/`FromAgentMetadata`, `ethereum.EthereumClient`/`NewEthereumClient`, `hpke.SignatureVerifier` family, `session.Session`, `session.NonceCache` family, `pkg/health` (from `sage`), bindings alias package if step 8 was taken | `02` §4 table; `DECISIONS.md` decision 2 consequences |

---

## 5. Research plan

Research questions RQ1-RQ7 are `07` §7 items 1-7; RQ8 and RQ9 are added
because the task asks for efficiency measurements and the reviews identified
an unmeasured DoS surface (`09` §3 "unbounded"; BACKLOG B-15). Baseline
libraries named here were not checked for API fit [Low]; the metric
definitions were.

| RQ | Question | Hypothesis | Experiment | Metric | Baseline | Repository | Prerequisite step |
|---|---|---|---|---|---|---|---|
| RQ1 | Is the 1-RTT handshake secure against Noise-style adversaries including identity misbinding? | Mutual authentication, forward secrecy from `ssE2E`, and DID binding hold; initiator key confirmation is absent by design (`08` §3 "Structural assessment") | Tamarin (and ProVerif cross-check) model derived from spec 04 §2-§6 and 06 §3; properties: agreement on (`initDid`, `respDid`, `enc`, `ephC`, `ephS`, seed), secrecy of seed under ephemeral and static compromise, misbinding per `07` 3.3 | properties proved / threat-model lattice (Vacarme style, `07` 3.8) | Noise NK (`07` 3.6) and KEMTLS models (`07` 5.3) | `sage-spec/formal/` (model derived from the same text as the vectors, `07` 5.2) | step 0 (spec tag) and resolution of spec open items O-6, O-7 (`08` §3), because both change the transcript |
| RQ2 | Does the record layer satisfy channel robustness (`07` 3.13)? | Yes for the 1024 window with counter-authenticated AAD; the random nonce is redundant | proof sketch against the FGJ definitions; differential harness feeding the same record streams (replay, reorder, stale, forgery, rekey boundary, 20-35 byte records) to Go and Rust through FFI | verdict agreement rate across cores; forgery attempts before rejection; bytes per record (SAGE 20 + 16 vs TLS 5 + 16, `08` §4.3) | DTLS 1.3 / QUIC record layer numbers (`07` 3.11, 3.12) | `sage-bench/adversarial/session`, vectors in `sage-spec` | steps 6, 7 |
| RQ3 | Which A2A and MCP threats does SAGE close, measured? | In-transit alteration and replay are closed; malicious-at-source is not (`07` §6) | re-run the A2ABreak 11 findings (`07` 1.6) and the MAESTRO list (`07` 1.3) against A2A traffic wrapped by `sage-gateway`; run MCPTox (`07` 1.8) through the gateway's MCP wrapper; split altered-in-transit vs malicious-at-source | per-threat closed/open; detection rate; false-positive rate (`07` 5.6 format) | unwrapped A2A and MCP | `sage-bench/adversarial/{a2a,mcp}`; scenarios require `sage-gateway` MCP stdio bridge (F-05 remainder) | step 9; F-05 bridge |
| RQ4 | What do registration and revocation cost, and how fast does revocation reach verifiers? | Three transactions plus the 1-60 min and 1 h windows exceed every benchmarked DID method (`07` 4.4); revocation latency is bounded by the gateway cache TTL (5 min / 30 s negative, `09` §2b) | Satybaldy et al. protocol (`07` 4.4): gas, fee, wall-clock per phase on Hardhat and Sepolia; revocation-to-first-rejection under cache TTLs; metadata exposure per phase | gas, USD-equivalent, seconds per phase; seconds to rejection | `did:ethr` numbers from `07` 4.4; ERC-8004 single `register()` | `sage-bench/chain` with `sage-contracts` deployments and `sage-cli` for the transactions | step 0; the `sage-did commit` key-binding fix (`09` §2e "CLI gap") |
| RQ5 | Does staked commit-reveal reduce placeholder and Sybil registrations? | Lower placeholder rate than the open ERC-8004 registries measured by `07` 2.4 | replicate the Xiong et al. measurement (endpoint liveness, reviewer clustering) on the SAGE Sepolia registries and public ERC-8004 registries | live-endpoint %, clustered-reviewer % | `07` 2.4 figures | `sage-bench/chain/ecosystem` (scripts + dataset snapshot) | none for data; C-01 redeploy before publishing conclusions |
| RQ6 | Do byte-exact vectors keep implementations interoperable over time? | Divergences appear per release and are caught by vectors before release | interoperability matrix Go core / Rust core / gateway / inspector / each SDK over every spec and implementation tag; count Go/Rust verdict differences per release (today eleven, `10` "Facts"); both generation directions | vector pass rate; divergences caught pre-release vs post-release; matrix cells green | the eleven current differences as the t0 row | `sage-spec` README (matrix), `sage-inspector` cross-implementation mode, `sage-bench/interop` driver | step 7; step 6 for Rust cells |
| RQ7 | Can delegation be carried in the signed envelope without a second round trip? | A South et al. credential (`07` 2.8) fits in the init payload with tens of bytes and one extra verification | extend the envelope in a spec draft; re-run the A2ABreak multi-hop identity-loss case; measure size and verify time | added bytes; verify latency delta; identity-loss case closed | RQ1 model without delegation; unwrapped A2A | `sage-spec` (draft), `sage-bench/adversarial/a2a` | RQ1 model; step 9 [Low] |
| RQ8 | What are the latency and byte overheads of SAGE versus TLS, mTLS and Noise? | 1-RTT HPKE handshake within a small factor of TLS 1.3 mTLS on localhost; per-record overhead 20 bytes above TLS; RFC 9421 signing adds header bytes plus one signature per request | Go `testing.B` harness: (a) handshake wall-clock p50/p99 for `hpke.Client.Initialize`/`Server.HandleMessage`, Go `crypto/tls` 1.3 with client certs, Noise NK/XX via a Go Noise implementation; (b) bytes on the wire per handshake and per record; (c) RFC 9421 sign/verify microseconds and header bytes per key type vs mTLS; Rust `criterion` equivalents (`rs-sage-core/benches/*`) for the cross-core column | microseconds p50/p99, bytes, CPU cycles | TLS 1.3 (mTLS), Noise NK/XX | `sage-bench/bench` (seeded from `tools/benchmark`, `hpke_bench_test.go`, `rs-sage-core/benches`) | step 1 (library tag); step 6 for the Rust column |
| RQ9 | How do the replay guards behave under adversarial unique-nonce floods? | Unbounded in-memory guards (`09` §3) grow linearly until TTL; a bounded RFC 6479-style window (`07` 3.10) holds memory constant with equal correctness | flood the gateway `serve` and `hpke.Server` with distinct nonces per keyid/ctx; measure RSS and verify throughput; then the same with a bounded guard and the cookie-first order (`08` 3.1) | RSS over time; requests/s; correctness on the RQ2 replay suite | current guard | `sage-bench/adversarial/replay`; fixes land in `sage` (B-14, B-15) | step 1 |

### Artefacts per experiment

| Artefact | Content | Host repository | Exists today |
|---|---|---|---|
| Formal models | Tamarin `.spthy`, ProVerif `.pv`, property list, lemma results | `sage-spec/formal/` | no (`07` "Facts": no published analysis) |
| Golden vectors and `rejected` lists | six suites today; `rejected` schema for rfc9421, hpke, jcs, session, did (`10` §3) | `sage-spec/vectors/` | partially (26 vectors, one `rejected` list) |
| Wycheproof subset | ECDSA/EdDSA/X25519/ChaCha20 edge cases (`07` 5.5) | `sage-bench/vectors/wycheproof/` (mirrored, pinned) | no |
| Benchmark harness | Go `testing.B` suites, Rust `criterion` links, TLS/Noise baselines, `Makefile` | `sage-bench/bench/` | seeds: `sage/tools/benchmark/*`, `pkg/agent/hpke/hpke_bench_test.go`, `rs-sage-core/benches/{crypto,hpke,session}_benchmarks.rs` |
| Interoperability matrix driver | scripts that build each repo at a tag and run the inspector conformance step and the live gateway/Rust exchange | `sage-bench/interop/` + `sage-inspector` cross-implementation mode | no (F-03b open) |
| Adversarial suites | A2ABreak case scripts, MAESTRO checklist, MCPTox runner config, session/replay mutators | `sage-bench/adversarial/` | partial: Go security tests (`08` §8) are unit-level only |
| On-chain measurement scripts and datasets | Hardhat and Sepolia phase timings, gas logs, ecosystem snapshot for RQ5 | `sage-bench/chain/` with `sage-contracts` deployments | no |
| Results | CSV/JSON per experiment, per commit tuple, with the compatibility-matrix row used | `sage-bench/results/` | no |

---

## 6. Risks and open decisions

| # | Decision or risk | Option A | Option B | Recommendation |
|---|---|---|---|---|
| 1 | Removal horizon conflict: policy says 1.8.0, D-06 and two code comments say 1.7.0 (`02` §4) | Follow the policy: every deprecated symbol, including `handshake`, leaves at 1.8.0. Trade-off: 1.7.0 still ships a handshake with no freshness checks (D7) and two extra `pkg` packages. | Record an exception in `VERSION_POLICY.md` §4 for symbols with zero external importers and remove them at 1.7.0. Trade-off: the policy gains a case that needs evidence (an import search) per exception. | B; the evidence exists in `DECISIONS.md` [High] |
| 2 | Extract `sage-cli` or keep `cmd/` in `sage` | Extract (step 3). Trade-off: one more repository to release; `sage` loses the only in-tree callers of `did.Manager` (`02` §2e), so behaviour regressions surface first in `sage-cli` CI. | Keep `cmd/` in `sage` as today. Trade-off: the core module keeps cobra, YAML config and `pkg/health` as dependencies; the "reusable core library" goal is met only by convention. | A, but only after step 1 tags the library, so the CLI pins a real version [Mid] |
| 3 | Where generated contract bindings live | `sage-contracts/bindings/go` module regenerated on tag (step 8). Trade-off: a Go module inside a Solidity repository; consumers pin two tags (`sage`, bindings). | Keep generation in `sage` from the pinned ABI (today). Trade-off: 5,909 generated lines and the abigen toolchain stay in the core module; drift check remains. | B now, A after the first two `sage-contracts` tags show the ABI is stable [Mid] |
| 4 | Fate of `pkg/storage`, `pkg/oidc` (zero in-module consumers) and `pkg/health` | Move `storage` to `sage-gateway` as the persistent replay guard after adapting `NonceStore` to `session.ReplayGuard` (B-16); delete `oidc` unless an external importer is found; `health` to `sage-cli`. Trade-off: PostgreSQL code moves to the repository that needs it; anyone importing `oidc` is broken at 1.8.0. | Keep all three in `sage` as platform packages. Trade-off: the core module carries `pgx`, `jwt` and Prometheus requirements that no protocol consumer needs. | A; verify with an organisation-wide import search before marking `oidc` deprecated [Mid] |
| 5 | Examples as a repository or a nested module | Separate `sage-examples` (step 4). Trade-off: examples can pin `sage-gateway` too; another CI to maintain. | Nested `examples/go.mod` in `sage`. Trade-off: cheaper; the examples cannot depend on `sage-gateway` without a cycle of pins on unreleased versions. | B until `sage-gateway` has a tag, then A [Mid] |
| 6 | Two cores diverge again (`STRATEGY.md` §10); Rust has no named owner (BACKLOG "Decisions still open") | Freeze `rs-sage-core` scope at Peer level and gate every spec change on both cores passing the vectors before tagging. Trade-off: spec velocity limited by the slower core. | Declare Rust the only SDK substrate and let the Go core lag on non-wire features. Trade-off: the Go reference could then fail vectors generated by Rust; conflicts with 00 §6 "Go is normative". | A; the "spec first" rule already exists, it needs the both-directions vector CI (row 8) to be enforceable [High] |
| 7 | Where the formal model lives | `sage-spec/formal/` next to the text it models. Trade-off: the spec repository gains a second toolchain (Tamarin) in CI or runs it manually. | `sage-bench/formal/`. Trade-off: the model can drift from the spec without the spec PR noticing. | A [Mid] |
| 8 | Licence of `sage-examples` and `sage-bench` | Apache-2.0 (meant to be copied). Trade-off: they link LGPL-3.0 `sage`; distributing binaries built from them triggers LGPL §4, which these repositories do not do, but this is a legal question the maintainers must confirm. | LGPL-3.0 like the gateway (`LICENSING.md` §4 reasoning). Trade-off: reviewers copying harness code into their own work inherit copyleft. | A pending a licence check; not legal advice [Low] |
| 9 | Measuring a moving target: `08` recommends spec-breaking fixes (`@request-target`, `@signature-params`, cookie order O-6, unused traffic keys O-7) and 00 §5 makes each a MAJOR bump | Land the four fixes before `sage-spec v1.0.0`, then start RQ1/RQ2/RQ8. Trade-off: the research plan waits for one more spec draft. | Start measurements on `1.0.0-draft.1` and re-run after the fixes. Trade-off: two result sets; the formal model must be redone once. | A [High] |
| 10 | `sage-adk` is broken since v1.0 and uses a local `replace` (`sage-adk/go.mod:75`) | Migrate `sage-adk` to `pkg/agent/*` in its own repository (no shim in `sage`; alias packages were rejected in `DECISIONS.md` decision 2). Trade-off: work in a repository outside this plan. | Archive `sage-adk`. Trade-off: loses the only "framework" consumer. | Out of scope for `sage`; note in the 1.6.0 release notes that no compatibility path exists [High] |

---

## Facts vs Opinions

**Fact** (measured on the local checkouts or read in the cited files)

- `go list ./...` in `sage` at `e98b42b` returns 60 packages; `reports/bindings` and `sdk/typescript/node_modules/flatted/golang/pkg/flatted` are among them.
- Non-test Go LOC by cluster (section 1.2) were measured with `wc -l`; the deletion set (`handshake`, `session_creator.go`, `core/message`, `nonce`, `vault`) totals 3,237 lines including tests.
- `sage-gateway/go.mod:10` pins `sage v1.5.3-0.20260912041026-b04477bc0840`; `sage-inspector/go.mod:10` pins `v1.5.3-0.20260912042550-5ab9c7e46ef3`; neither uses `replace`.
- `sage-adk/go.mod` requires `sage v0.0.0` with `replace github.com/sage-x-project/sage => ../../sage` and imports `github.com/sage-x-project/sage/{crypto,did,config,core/rfc,crypto/keys,crypto/formats,crypto/storage,did/ethereum}`.
- `sage-spec` has nine chapters (`spec/00-08`) and six vector files; `spec/00-overview.md` §4 defines the Verifier, Peer and Reference conformance levels; §6 makes the Go core normative where the text is silent.
- `sage-contracts` was created by `git subtree split` of `contracts/` (`ef304f3`) and publishes ten ABI files; it is untagged and `sage/.contracts-version` holds commit `2ff992e`.
- `rs-sage-core` ships `include/sage_crypto.h`, `pkg/sage_crypto_core_bg.wasm` and three criterion benches; `08` §7 counts 52 `extern "C"` functions with no HPKE export and no `catch_unwind`.
- `VERSION_POLICY.md` §4 rule 3 removes deprecated symbols two minors after marking; BACKLOG D-06 says "remove v1.7"; both are in the tree.
- `07` §8 records that no published formal analysis of the SAGE handshake or session layer was found; `10` "Facts" lists eleven inputs on which the Go and Rust cores give different verdicts.
- `sage-inspector` today derives `ExpectedDID` from the message and disables the replay check (`10` "Facts").

**Opinion**

- [High] The remaining split of `sage` is small in code (CLI ~5.9k LOC, examples ~3.8k, deletions ~3.2k) compared with the work that makes the set researchable: the both-directions vector CI, the inspector P1 items and the live Go/Rust exchange (steps 6-7). Those three should be scheduled before the CLI extraction if maintainer time is the constraint.
- [High] Constants and labels should be pinned by a machine-readable list in `sage-spec` with an equality test in each core; code generation is not needed at the current size.
- [High] The four spec-breaking fixes in `08` (§2.1, §2.2, §3.1, §3.2) must land before any measurement is published, because 00 §5 makes each a MAJOR bump and the formal model depends on the transcript.
- [Mid] Moving `pkg/storage` to the gateway as its persistent replay guard gives the dead `NonceStore` a real consumer and removes PostgreSQL from the core module; the alternative (delete) loses tested code.
- [Mid] The bindings module in `sage-contracts` is worth doing only once the ABI has been tagged at least twice without change; before that it adds a second tag to pin for no stability gain.
- [Low] Effort figures in section 4 are single-maintainer estimates without a calibration history in this repository; the F-03b and SDK steps in particular could be several times larger.
- [Low] The RQ8 baselines (Go `crypto/tls`, a Go Noise implementation) were not checked for API fit; the metric definitions are the part to hold fixed.
