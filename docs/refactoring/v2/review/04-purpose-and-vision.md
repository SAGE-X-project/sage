# 04. Purpose and vision: what SAGE says it is for

Status: review, 2026-09-12. Read-only synthesis of the project's own statements
of purpose. Paths are relative to `sage/` unless prefixed with a sibling
repository name. Line numbers refer to the working tree on 2026-09-12;
historical README quotes carry the commit hash. Interpretations are marked
[High]/[Mid]/[Low]; unmarked sentences are quotations or direct restatements.

## 1. Problem statement

### What problem

The current README defines SAGE as "a blockchain-based security framework for
AI agent communication — providing end-to-end encrypted, authenticated
channels between AI agents using decentralized identity (DID), HPKE key
agreement (RFC 9180), and HTTP Message Signatures (RFC 9421)"
(`README.md:12`). `docs/ARCHITECTURE.md:17` and the first Korean guide
(`docs/overview/DETAILED_GUIDE_PART1_KO.md:23`) use the same phrase,
"blockchain-based security framework". The Korean guide adds the layer
argument: SAGE "operates at a different purpose and a different layer than
TLS" (`PART1_KO.md:47`) and HPKE "does not replace TLS; it is application-layer
encryption added on top of TLS" (`PART1_KO.md:132`).

The 2026-09 strategy restates the purpose from the maintainer in different
terms: "when an AI agent calls another agent, calls an MCP server, or is
called by one, the request and the response must not be forgeable, replayable
or silently altered, and any agent client should be able to adopt that
protection without re-implementing cryptography. Signing and verification are
deterministic operations, so the protocol must be specified so that every
implementation produces and accepts exactly the same bytes"
(`docs/refactoring/STRATEGY.md:19`). The specification repository compresses
this to one sentence: "SAGE secures messages between AI agents"
(`sage-spec/spec/00-overview.md:5`).

The five properties named in the task (identity, authenticity,
confidentiality, replay protection, on-chain trust) map onto the ADRs:
ADR-003 opens with "For agents to establish trust and authenticate each other,
they need verifiable digital identities" (`docs/adr/003-did-method-selection.md:15`)
and lists "How to prevent man-in-the-middle attacks? How to ensure keys
haven't been compromised?" (`:36-37`); ADR-002 opens with messages that
"must be protected from eavesdropping, tampering, and unauthorized access"
(`docs/adr/002-hpke-selection-rationale.md:15`) and requires "Resistance to
replay attacks" (`:49`); on-chain trust is the "Verifiability" consequence,
"Anyone can verify agent identity on blockchain" (`003:270`).

### For whom

Only two documents name a target user. `CLAUDE.md:7` (untracked working
file) states "Target User: AI 에이전트 시스템 개발자, MCP/A2A 프로토콜 사용자"
(AI agent system developers, MCP/A2A protocol users). The strategy names
concrete adopters: "agent clients (Codex, Claude Code, Hermes and others)"
(`STRATEGY.md:8`), and the gateway is "The adoption path for agent clients
that cannot or should not link a library" (`STRATEGY.md:83`). The Korean
guides target "beginner programmers to intermediate developers"
(`PART1_KO.md:3`). No README revision in the 28-commit history has ever
contained a target-audience, roadmap, vision or threat-model section
(verified by the git sweep in §2). [High] The audience was implicit until
2026-09; the MCP-client framing is new.

### Threat model as written

Three documents state a threat model; they overlap but are not identical.

| Source | Protected against | Explicitly not protected / assumed |
|---|---|---|
| `docs/adr/002-hpke-selection-rationale.md:394-405` | Eavesdropping, message tampering, impersonation, replay ("nonces + timestamps"), forward secrecy | Traffic analysis, denial of service ("must handle at application layer"), key compromise, side-channel attacks |
| `docs/ARCHITECTURE.md:461-476` | MITM ("HPKE forward secrecy"), replay ("Nonce-based deduplication and timestamp validation"), impersonation ("DID-based identity with on-chain verification"), key compromise ("Ephemeral keys limit blast radius"), message tampering (AEAD) | Assumes "Blockchain registries are trusted", "Agent endpoints are authenticated via TLS"; not mitigated: endpoint compromise, side channels, "Denial of Service (application-layer DoS protection needed)" |
| `docs/overview/DETAILED_GUIDE_PART1_KO.md:446-457` | Scenario 1 identity impersonation (신원 사칭), scenario 2 man-in-the-middle, scenario 3 replay ("same operation executed multiple times, e.g. money transfer") | Not stated; TLS limitations at `:176-217` (no true E2E, no non-repudiation) are the motivation |
| `docs/refactoring/STRATEGY.md:131-140` | Body tampering, replay of a signed request or tool result, DID impersonation, malicious tool result injected into an agent, confused deputy via MCP callback, downgrade to unsigned traffic, key compromise/rotation, DoS on the verifier | Column "Today" records that most of these were "Not covered" on the live HTTP path before the 2026-09 wiring |

Three points follow. First, the 2025-10 documents describe threats against
agent-to-agent channels; the 2026-09 strategy adds MCP-specific threats
(tool-result injection, confused deputy, downgrade) that appear nowhere in
the ADRs or guides. Second, DoS moves from "not protected" (ADR-002, ARCHITECTURE) to
"bounded, TTL-evicting nonce store" and gateway size limits
(`STRATEGY.md:140`), while `PART4_KO.md:414` already described an optional
cookie/puzzle check. Third, the strategy's own audit found that "the
protection the project promises is not yet enforced on the paths consumers
actually use" (`STRATEGY.md:37`), with executed evidence: an RFC 9421 request
"still verifies after the body is replaced" and "HPKE sessions carry an
all-zero signing key" (`STRATEGY.md:27`). The unreleased changelog records
the fixes (`CHANGELOG.md:42-47`). [High] The threat model was aspirational
until 2026-09 for the HTTP path.

## 2. Vision and goals over time

| Date | Document | Stated goal |
|---|---|---|
| 2025-07-05 | `6d36735:README.md:2` | "this repo support sage(secure agent guarantee engine) library." (entire file) |
| 2025-08-24 | `1567112:README.md:10` | "a comprehensive blockchain-based security framework for AI agent communication. It implements RFC 9421 ... with decentralized identity (DID) management on multiple blockchains." Licence badge MIT (`:5`). |
| 2025-10-07 | `docs/overview/DETAILED_GUIDE_PART1_KO.md:23,660-697` | Same "framework" definition; five-layer stack with "Layer 1: Blockchain Layer" as the base. |
| 2025-10-11 | `CHANGELOG.md:794` | "first production-ready release ... a complete blockchain-based security framework for AI agent communication with end-to-end encryption, decentralized identity management, and RFC-compliant message signatures." Quality gates: "Minimum 70% code coverage", "90%+ coverage for critical paths" (`:911-915`). |
| 2025-10-11 | `docs/archive/2026-09/planning/PERFORMANCE_OPTIMIZATION_ROADMAP.md:31,131-149` | "currently in a production-deployable state"; goals are throughput, latency, memory; targets such as allocations 38 to under 10, throughput 8,700 to over 15,000 msg/s. Status "planning stage", never approved (`:1016-1018`). |
| 2025-10-26 | `docs/adr/001..003` | Transport-agnostic security layer (`001:43-48`); HPKE for E2E with forward secrecy and quantum-evolvable design (`002:17-22`); decentralised, verifiable, multi-chain identity (`003:17-22`); CA and OAuth rejected as "Too centralized" (`003:369,392`). |
| 2025-10-30 | `73d8291:README.md:14` | "provides end-to-end encrypted, authenticated communication channels ... using DID management, HPKE-based key agreement, and RFC 9421". Multi-language bindings presented as working (`:690-788`). |
| 2026-02-20 | `3556ecb:README.md:12` | Condensed to the current one-sentence definition; networks widened to "BSC, Base, Arbitrum, Optimism — EVM-compatible deployment ready" (`:325-336`). |
| undated (log stamp 2025-10-10) | `docs/ARCHITECTURE.md:659-665` | Future enhancements: multi-party sessions, key rotation without interruption, quantum-resistant cryptography, zero-knowledge identity, cross-chain bridges. |
| 2026-09-11 | `docs/refactoring/STRATEGY.md:8,19,43-45` | Tamper-evident agent-to-agent and agent-to-MCP calls, adoptable by Codex/Claude Code/Hermes; "protocol first, cores second, bindings third"; "The deterministic guarantee ... cannot live in a library; it lives in a specification plus conformance vectors". |
| 2026-09-12 | `docs/refactoring/v2/REPO_PLAN.md:58-66` | Six repositories: `sage` (Go reference core), `rs-sage-core` (Rust, C ABI, WASM), `sage-spec`, `sage-contracts`, `sage-gateway`, `sage-inspector`; SDK repositories deferred. |
| 2026-09-12 | `sage-spec/README.md:3-5` | "the wire-level rules that every SAGE implementation must follow so that agents built on different cores interoperate." |
| 2026-09-12 | `sage-gateway/README.md:3-8` | Signatures "in front of and behind HTTP services such as MCP servers, so agents that cannot sign or verify themselves still get authenticated, tamper-evident, replay-protected calls." |

Reading the table top to bottom: the noun changes from "library" (2025-07) to
"framework" (2025-08 through today's README and CLAUDE.md), and in 2026-09 to
a protocol with several implementations, of which the Go module is "the
reference implementation" (`README.md:363`). The object of protection widens
from agent-to-agent channels to agent-to-MCP calls. The blockchain moves from
the foundation layer of every diagram to a layer the specification declares
"Out of scope" (`sage-spec/spec/00-overview.md:20-22`). [High] The 2026-09
documents are the first to state goals as gates rather than feature lists.

The task mentions a "2025-10 contest era". The git sweep found no commit,
README revision or document mentioning a contest, hackathon or award;
"Kaia" appears only as a supported chain and in the acknowledgements
(`73d8291:README.md:640,900`). [High] If such a context existed it is not
recorded in this repository.

## 3. Non-goals and constraints

| Constraint | Where stated | Note |
|---|---|---|
| Licence LGPL-3.0 for the Go core; contracts MIT | `README.md:423-431`; `docs/refactoring/v2/LICENSING.md:5,74-78`; `REPO_PLAN.md:114` ("Decided 2026-09-12: Go repositories stay LGPL-3.0") | History: `LICENSE` was GPL-2.0 text from 2025-07-05 while the README claimed MIT; both became LGPL-3.0 in `25b49a6` (2025-10-06). `STRATEGY.md:176` recommended Apache-2.0 everywhere; `LICENSING.md:22-26` rejected relicensing the Go core pending contributor consent (A-16). |
| Per-repository licences | `LICENSING.md:76-80` | `rs-sage-core` MIT OR Apache-2.0, `sage-spec` Apache-2.0, gateway/inspector LGPL-3.0 (`sage-gateway/README.md:114`, `sage-inspector/README.md:57`). |
| SDKs under `sdk/` follow LGPL-3.0 until moved; then "the licence of the Rust core they bind to" | `README.md:433` | |
| Go import paths stay `pkg/agent/*`; no `/v2` module | `docs/refactoring/DECISIONS.md:26-34`; `VERSION_POLICY.md:11,44` ("a `/v2` module path, which is not planned"); `STRATEGY.md:166` | Justified by two external consumers and the v1.0 breakage of `sage-adk` (`DECISIONS.md:8,28`). |
| Deprecations live two minor releases | `VERSION_POLICY.md:44` | |
| Go 1.26 required | `README.md:47`; `CHANGELOG.md:35` | |
| Chain scope | `README.md:326-335`: Ethereum, Kaia "Production deployment", BSC/Base/Arbitrum/Optimism "deployment ready", Solana "In development"; `CLAUDE.md:9` same | `PART6C_KO.md:1290-1292` lists Solana Mainnet/Devnet as supported; see §5. |
| Only AgentCardRegistry supported; `SageRegistryV2` dropped | `DECISIONS.md:70-78`; `README.md:359` ("Legacy contracts are deprecated") | |
| 4-phase handshake removed; HPKE 1-RTT only | `DECISIONS.md:14-22`; `CHANGELOG.md:84-85` | Cost accepted: "a KEM key is required for encrypted sessions" (`DECISIONS.md:20`). |
| Language SDKs "experimental and not yet interoperable with the Go core" | `README.md:363-371`; `DECISIONS.md:58-66`; `CHANGELOG.md:24` | SDK repositories "deferred until `rs-sage-core` publishes a C header and a WASM artifact" (`REPO_PLAN.md:16-17`). |
| cgo `lib/` removed | `DECISIONS.md:46-54`; `CHANGELOG.md:82` | |
| Spec out of scope: contracts, gateway, client integrations, storage, metrics, operations | `sage-spec/spec/00-overview.md:20-22` | |
| Rust core is "pure crypto and protocol"; on-chain reads belong to Go core and gateway | `REPO_PLAN.md:49` | Marked "Recommended: drop", confidence Mid in the source. |
| Gateway "adds nothing to the protocol" | `sage-gateway/README.md:8` | Roadmap items are stdio MCP bridge, capability checks, HPKE session mode (`:105-110`). |
| Not mitigated: endpoint compromise, side channels, DoS | `docs/ARCHITECTURE.md:473-476`; `002:401-405` | |
| Offline agents: "direct communication is not possible" | `PART6C_KO.md:1351-1353` | |
| Release artefacts unsigned until Sigstore/SLSA land | `VERSION_POLICY.md:48` | |

[Mid] The only non-goal stated before 2026-09 is the ADR/ARCHITECTURE "not
mitigated" list; every other constraint above was written in the last four
days, so the older documents should be read as having no explicit non-goals.

## 4. Success criteria

### As written

- 2025-10: "85+ feature tests with 100% pass rate" (`CHANGELOG.md:898`);
  "Minimum 70% code coverage requirement", "90%+ coverage for critical paths"
  (`:911-915`); v1.5.0 "Solidity Tests: 202/202 passing" (`:511`).
- 2025-10 performance roadmap: allocation, latency and throughput targets with
  sprint milestones (`PERFORMANCE_OPTIMIZATION_ROADMAP.md:131-149,925-975`);
  archived as predating the current session layer (`:3`).
- 2026-09 strategy, per migration step (`STRATEGY.md:150-157`): governance P0
  list green; security wiring audit closed with external consumers still
  building; "Go passes 100 % of its own vectors; negative vectors present";
  bindings regenerated from CI match committed ones; "codegraph gates (0 layer
  violations, 0 exact duplicates)"; Rust core "passes spec vectors; ABI
  documented"; "Python SDK talks to `sage-gateway` end to end"; "an unmodified
  MCP client exchanges signed calls and results through the wrapper".
- Conformance definition: "A conforming implementation of a layer MUST pass
  the vectors of that layer and of every layer below it"
  (`sage-spec/spec/00-overview.md:54-55`), with three levels, Verifier, Peer,
  Reference (`:59-63`).
- Rust core "reaches `1.0.0` when a CI job exchanges an HPKE handshake and
  RFC 9421 messages with the Go core" (`VERSION_POLICY.md:45`).

### Measurable today

| Criterion | Mechanism | Evidence | State |
|---|---|---|---|
| Go core reproduces the spec vectors | `spec-vectors` job runs `go run ./cmd/sage-vectors check -dir .sage-spec/vectors` | `.github/workflows/test.yml:169-192`; six suites in `sage-spec/vectors/` | Measurable; checkout is `ref: main`, not a spec tag (`test.yml:182`; open action in `VERSION_POLICY.md:55`) |
| Rust core reproduces the vectors | Vectors in `rs-sage-core` CI | `REPO_PLAN.md:61` "F-03 done at vector level" | Measurable at vector level; not verified in this review |
| Live Go/Rust interoperability | CI job exchanging a handshake | `VERSION_POLICY.md:38` "no live Go/Rust exchange yet (F-03b)" | Not measurable yet |
| Layer boundaries | `codegraph` with `layer-baseline.txt`; `depguard` | `.github/workflows/test.yml:112-114`; `docs/refactoring/README.md:39-40` | Measurable |
| Bindings match the pinned contracts ABI | `make bindings-check` against `.contracts-version` | `README.md:306-315` | Measurable; pin is a commit, tag `v1.5.0` pending (`VERSION_POLICY.md:54`) |
| Coverage and race | `go test -race -coverprofile` | `test.yml:42-45` | Measured, no threshold enforced in the workflow that was read [Mid] |
| Gateway round trip, replay, authority binding | Gateway tests | `sage-gateway/README.md:99-103` | Measurable in that repository; not run here |
| MCP client end-to-end through the wrapper | Recipes only | `sage-gateway/README.md:77`; strategy gate at `STRATEGY.md:157` | Not automated |
| SDK interoperability | Deferred | `REPO_PLAN.md:66` | Not measurable; nothing to measure |

[High] The only cross-implementation success criterion that exists as a
mechanism today is the vector run; every interoperability claim before
2026-09 (SDK sections of the 2025-10 README, `73d8291:README.md:690-788`) had
no measurement behind it, which `DECISIONS.md:60` documents.

## 5. Divergences

| Topic | Statement A | Statement B | Reading |
|---|---|---|---|
| Library, framework, protocol or service | "library" (`6d36735:README.md:2`); "framework" (`README.md:12`, `ARCHITECTURE.md:17`, `CLAUDE.md:5`) | "cannot live in a library; it lives in a specification plus conformance vectors" (`STRATEGY.md:45`); `sage` is "Reference core" (`REPO_PLAN.md:60`); gateway is a service that "adds nothing to the protocol" (`sage-gateway/README.md:8`) | [High] Not contradictory once layered: the protocol is the product from 2026-09 on, the Go module is one implementation, the gateway is the deployable service. The README and CLAUDE.md have not been reworded to say so. |
| Blockchain required or optional | Every 2025 diagram places the chain at the base (`PART1_KO.md:691`); `ARCHITECTURE.md:462` assumes "Blockchain registries are trusted"; `PART6B_KO.md:39` lists RPC access as a prerequisite; ADR-003 rejects `did:key` because it "Lacks blockchain anchoring for trust" (`003:418`) | `sage-spec/spec/00-overview.md:20` puts registry contracts out of scope; the gateway verifies "against a trusted-key table and/or the on-chain registry" with "Static entries win over the chain" (`sage-gateway/README.md:12,40`); `PART6C_KO.md:1379-1381` proposes a Phase 1 with "RFC 9421 signature verification only" and no chain; the Verifier conformance level needs `did` vectors but no chain (`00-overview.md:61`) | [High] Diverges. The chain is required for `did:sage` resolution and registration, optional for signature verification with pre-shared keys. No document states this split explicitly. |
| Handshake shape | "1-RTT" (`CLAUDE.md:60`, `README.md:149`) | 4-phase Invitation/Request/Response/Complete in `ARCHITECTURE.md:159-171`, `PART6A_KO.md:2033`, `PART6C_KO.md:1622`; `STRATEGY.md:32` records the contradiction | [High] Resolved in code by Decision 1 (`DECISIONS.md:14-18`); the guides and ARCHITECTURE still describe the removed flow. |
| Solana status | "In development" (`README.md:331`, `CLAUDE.md:9`, `PART1_KO.md:324`, `PART3_KO.md:318`) | "Solana (Mainnet, Devnet)" supported (`PART6C_KO.md:1290-1292`); "Solana Program (Rust-based on-chain registry)" as a component (`ARCHITECTURE.md:47`); `chain.Presets` includes `solana-devnet`, `solana-mainnet` (`CHANGELOG.md:26`) | [Mid] Diverges; the README wording is the conservative one. |
| Session AEAD | `STRATEGY.md:62` recommends "AES-256-GCM as mandatory ... ChaCha20-Poly1305 optional" | `REPO_PLAN.md:84` fixes "ChaCha20-Poly1305 as the mandatory AEAD"; `sage-spec/spec/00-overview.md:48` | [High] The strategy's [Mid] opinion was overturned one day later; STRATEGY §2.1 is stale on this point. |
| Licence direction | `STRATEGY.md:176` "relicense the Go core to Apache-2.0" | `LICENSING.md:22-26` and `REPO_PLAN.md:114` keep LGPL-3.0 | [High] Superseded; the 2025-10 README's MIT badge and the GPL-2.0 `LICENSE` file are a second, older inconsistency already closed in `25b49a6`. |
| Production readiness | v1.0.0 "first production-ready release" (`CHANGELOG.md:794`); roadmap "production-deployable state" (`PERFORMANCE_OPTIMIZATION_ROADMAP.md:31`); Kaia "Production deployment" (`README.md:329`) | `STRATEGY.md:27,37` finds body-swap verification, zero session keys and unprotected contract admin functions; `CHANGELOG.md:44` "Existing Sepolia deployments must be redeployed" | [High] The 2025 production claims were not backed by the security wiring the project itself later audited. |
| SDK claims | 2025-10 README "Multi-Language Bindings" as a feature (`73d8291:README.md:690-788`); `CLAUDE.md:44` lists `sdk/` without caveat | `README.md:363-371` "experimental and not yet interoperable" | [High] README corrected; CLAUDE.md not. |
| DoS | Not mitigated (`ARCHITECTURE.md:476`, `002:403`) | Optional cookie/puzzle (`PART4_KO.md:414`); bounded nonce store and gateway size limits planned (`STRATEGY.md:140`); "rate limiting before verification" listed as not done (`sage-gateway/README.md:57`) | [Mid] Consistent if read as "partial, optional"; no document says which DoS class is in scope. |
| Guide currency | `PART3_KO.md:3`, `PART5_KO.md:3`, `PART6B_KO.md:3` carry 2026-09-12 status notes marking their bodies superseded | `PART1_KO.md`, `PART6A_KO.md`, `PART6C_KO.md` and `ARCHITECTURE.md` carry no note and still describe `SageRegistryV2` and the 4-phase handshake | [High] Inconsistent labelling within the same directory. |

## Facts vs Opinions

**Fact**
- The first README (`6d36735`, 2025-07-05) is two lines and calls SAGE a "library"; from `1567112` (2025-08-24) every README calls it a "blockchain-based security framework"; the current sentence dates from `3556ecb` (2026-02-20).
- No README revision, commit message or document in `sage` mentions a contest, hackathon or award.
- `LICENSE` was GPL-2.0 text while the README badge said MIT until `25b49a6` (2025-10-06) switched both to LGPL-3.0; the contracts carve-out is MIT.
- Three documents state a threat model: `docs/adr/002-hpke-selection-rationale.md:394-405`, `docs/ARCHITECTURE.md:461-476`, `docs/refactoring/STRATEGY.md:131-140`; the first two exclude DoS and side channels, the third adds MCP-specific threats.
- `sage-spec/spec/00-overview.md:20-22` declares registry contracts, gateway, client integrations, storage, metrics and operations out of scope of the protocol.
- `.github/workflows/test.yml:169-192` runs `sage-vectors check` against `sage-spec` at `ref: main`; `sage-spec/vectors/` holds six suite files; `VERSION_POLICY.md:38` records no live Go/Rust exchange.
- `README.md:363-371` and `CHANGELOG.md:24` mark the four SDKs experimental and non-interoperable; `CLAUDE.md:44` lists them without that caveat.
- `STRATEGY.md:62` recommended AES-256-GCM as the mandatory AEAD; `REPO_PLAN.md:84` and the spec fix ChaCha20-Poly1305.

**Opinion**
- [High] The purpose has been stable in substance since 2025-08 (authenticated, encrypted, replay-protected agent messaging with chain-anchored identity); what changed in 2026-09 is the delivery form (specification plus vectors, several cores, a gateway) and the addition of MCP traffic as a first-class object of protection.
- [High] The largest unresolved divergence is whether the blockchain is a requirement or an option. The code and the gateway already support a chain-free verifier path; no purpose document says so, and the README's opening sentence still leads with "blockchain-based".
- [High] `docs/ARCHITECTURE.md`, `CLAUDE.md` and the three Korean guides without status notes describe the pre-2026-09 system and should be relabelled or rewritten before they are cited as statements of purpose.
- [Mid] The 2025 "production-ready" wording should be treated as a release label, not a security claim, given the project's own 2026-09 audit findings.
- [Low] The reference to a "2025-10 contest era" in the task probably reflects context outside this repository (organisation, sibling repositories, or private communication); nothing here supports or refutes it.
