# 03. History timeline: how SAGE and its documentation evolved

Written 2026-09-12 from read-only `git log` queries on `main` (253 commits since
2025-07-05, tags v1.0.0 .. v1.5.2) and the files in the tree at `878932d`.
Hashes are short; dates are author dates (`--date=short`). Judgements carry
[High]/[Mid]/[Low]. Facts and opinions are separated at the end.

Method: `git log --reverse --date=short`, `git log --diff-filter=A -- <path>`
for creation, `git log --all` to see the pre-merge `dev` branch history behind
the squashed release PRs, `git log --follow` for renamed documents, commit
bodies, `CHANGELOG.md`, `docs/refactoring/*` and the documents themselves.

---

## 1. Timeline by month

| Month | Tag | Notable commits (hash, date, subject) | Added / changed / removed |
|---|---|---|---|
| 2025-07 | - | `6d36735` 07-05 Initial commit; `4b1ffe7` 07-08 feat: implement SAGE core functionality with CLI tools and MCP integration (#10) | Added `core/rfc9421`, `crypto`, `did`, `docs/core`, `docs/crypto`, `docs/did`. On the `dev` branch: `a1e5a48` 07-31 "feature: add handshake package", `57021ec` 07-31 "feature: add message package" (first `core/message/nonce`). |
| 2025-08 | - | `85595ea` 08-11 "3. Handshake (#11)" (dev); `55cdf6d` 08-17 Release: SAGE Registry V2 and enhanced features (#17); `1567112` 08-24 SageRegistryV2 security enhancements (#19) | `main` receives the 4-phase `handshake/`, `session/` (with `NonceCache`), `core/message/nonce`, `SageRegistry.sol` + `SageRegistryV2.sol`, `docs/dev/security-design.md`. |
| 2025-09 | - | dev: `363a481` 09-27 "feature: add RFC 9180", `a2974b0` 09-28 "feature: support HPKE"; `b6f40c2` 09-28 Fix test infrastructure and improve error handling (#25) | `#25` brings the `hpke` package, `docs/handshake/` (4-phase `handshake-en/ko.md`, `cryptographic-*.md`) and `docs/dev/SAGE-secure-session-communication-ko.md` (HPKE 2-step) to `main` in one commit. |
| 2025-10 (1-11) | v1.0.0 (10-11) | `695ad58` 10-06 handshake cache & integration test (#27); `25b49a6` 10-06 Release: merge dev (#28); `9e325f6` 10-11 Release: SAGE v1.0 (#33); `ad4f81f` 10-11 Prepare v1.0.0 (#63) | Packages moved under `pkg/agent/*`; `SageRegistryV3.sol` appears; `docs/audit/`, `docs/handshake/hpke-based-handshake-*.md`, `docs/handshake/README.md`, `docs/INDEX.md`, `docs/test/`, `docs/overview/` created. |
| 2025-10 (12-18) | v1.0.1 (10-12), v1.0.2 (10-14), v1.0.3 (10-18), v1.1.0 (10-18) | `aca9347` 10-12 HPKE handshake security improvement (#66) (#68); `6695144` 10-17 Remove A2A implementation from SAGE core (#90); `69ee16c` 10-18 Multi-Key Registry V4 (#95); `614dbb8` 10-18 docs: Remove all archived documentation | HPKE pinning + allowed suites; gRPC A2A transport removed (to `sage-a2a-go`); `SageRegistryV4.sol` + `types_v4.go` + `a2a.go` (A2A agent card); `SageRegistry.sol`, `SageRegistryV3.sol` deleted. |
| 2025-10 (19-26) | v1.1.1 (10-19), v1.2.0 (10-22), v1.3.0 (10-24), v1.3.1 (10-26), v1.4.0 (10-27) | `a071b08` 10-19 A2A integration docs (#96); `b5b8cda` 10-22 100% specification coverage (#110); `5e3b279` 10-26 Release v1.3.1 docs (#112); `965aa4f` 10-26 Hardhat v3 + governance tests (#114) | `SAGE_A2A_INTEGRATION_GUIDE.md`; `a2a_proof.go`, `key_proof.go`; `SageRegistryV2.sol` deleted (#110); `docs/adr/` (3 ADRs), `docs/test/sections/`; `AgentCardRegistry.sol` added and `SageRegistryV4.sol` deleted (#114). |
| 2025-10 (27-31) | v1.5.0 (10-30) | `0c29697` 10-28 AgentCardRegistry three-phase registration (#122); `0107563` 10-28 docs: Remove archive folder; `7111367` 10-28 KME key storage with X25519 verification (#123); `620d6a0` 10-30 PR #118 security enhancements (#124) | Go `AgentCardClient` (commit -> register -> activate), `sage-did commit/register/activate`; `AGENTCARD_MIGRATION_GUIDE.md`; RFC 9421 `BodyIntegrityValidator` (Content-Digest); HPKE `CompositeVerifier` (ECDSA + Ed25519). |
| 2025-11 | v1.5.1 (11-02), v1.5.2 (11-02) | `ac8b067` 11-02 Go 1.25.2 + contract tooling (#136); `da50b62` 11-02 KME -> KEM rename, HPKE client sync (#137); `d78e803` 11-03 deploy and verify contracts on Sepolia (#139) | Sepolia addresses recorded. Last commit before a 3.5-month gap. |
| 2026-02 | - | `7e4bfad`..`3556ecb` 02-19/20 (9 commits, no PR numbers) | Dependabot/Slither alert fixes; `3556ecb` restructures README, fixes badge versions, adds CHANGELOG `[Unreleased]`. Then a 6.7-month gap. |
| 2026-09-11 | - | 27 Dependabot commits (`91fe084`..`4f43301`); `a7ed062` code graph tool + refactoring design; `6f1415b` security wiring audit, supply-chain audit, docs graph; `8ab096e` wire replay, body binding, session keys (#221); `e7c713c` session seq/replay window/rekey (#265); `6bf8d90` response signing (#266); `84621cc` A2A proof bound to on-chain DID (#264) | Refactoring cycle opens: `docs/refactoring/` (12 files), CI hardening, security fixes on the paths real consumers use. |
| 2026-09-12 | - | `c1b8f67` recipient binding (#267); `f35feb2` JCS canonical JSON (#268); `2c48ceb` one replay guard (#281); `f068212` archive superseded docs (#294); `eff53f8` remove cgo lib, random tester, unused message packages (#295); `b04477b` contracts to `sage-contracts` (#306); `878932d` trim session/chain/storage surfaces (#322) | `docs/archive/2026-09/` (16 files); `docs/refactoring/v2/`; `handshake` + `internal/sessioninit` marked Deprecated; `contracts/` removed; `pkg/vectors`. |

Observations on the shape of the history:

1. Everything up to v1.5.2 was merged through large squashed release PRs
   (`#17`, `#28`, `#33`) whose bodies replay the whole `dev` history, so the
   `main` creation date of a feature is often weeks after its `dev` commit.
   Where the two differ this document names both. [High]
2. There are two dead periods (2025-11-03 to 2026-02-19 and 2026-02-20 to
   2026-09-11); 99 of the 253 commits were authored on 2026-09-11/12. [High]
3. `CHANGELOG.md:548` dates v1.1.0 as `2024-10-18`; the tag is 2025-10-18. [High]

---

## 2. Security features: origin, rationale, later changes, status

### 4-phase handshake (`pkg/agent/handshake`)

- Introduced: `a1e5a48` 2025-07-31 on `dev` ("feature: add handshake package",
  author sujine2), aggregated in `85595ea` 2025-08-11 "3. Handshake (#11)",
  reached `main` in `55cdf6d` 2025-08-17 (#17) as `handshake/`; moved to
  `pkg/agent/handshake` in `9e325f6` 2025-10-11 (#33). Peer cache and
  singleflight resolver added in `695ad58` 2025-10-06 (#27).
- Documents created with it: `docs/handshake/handshake-en.md` and `-ko.md`
  (`b6f40c2` 2025-09-28), `docs/dev/security-design.md` (`55cdf6d`),
  `docs/assets/SAGE-handshake.png`.
- Recorded rationale (first version, `git show b6f40c2:docs/handshake/handshake-en.md`):
  "It extends the existing A2A protocol and performs the handshake over gRPC"
  (line 7); "encrypts request/response payloads with the peer's DID public key
  to block man-in-the-middle attacks" (line 11); "Uses an X25519 ephemeral
  exchange to derive a shared secret, then derives signing and encryption keys
  for the session" (line 12). The four phases (Invitation, Request, Response,
  Complete) bind a random `kid` that "later becomes the keyId field in HTTP
  Message Signatures (RFC 9421)" (line 36). No document records a comparison
  with alternatives; the design is presented, not argued. [High]
- Why it was introduced [Mid]: it was the first session-establishment design,
  written before the HPKE package existed (dev 2025-07-31 vs 2025-09-27), and
  it reused only primitives already in the repository (Ed25519 identity keys
  from the DID registry, X25519, HKDF). The "bootstrap encryption with the
  peer's DID key" step exists because the DID document at that time carried
  only a signing key, not a KEM key; the KEM key field arrived with
  AgentCardRegistry (`7111367` 2025-10-28).
- Why it was superseded [High]: (1) round trips: HPKE doc
  `docs/handshake/hpke-based-handshake-en.md:21-22` "Compact 1-RTT exchange.
  The base mode finishes in two messages"; `docs/adr/002-hpke-selection-rationale.md:39`
  "Session establishment without multiple round trips" and `:53` "Single
  round-trip key agreement". (2) forward secrecy framing: `hpke-based-handshake-en.md:14-16`
  (exporter secret HKDF-combined with an ephemeral-ephemeral DH). (3) the
  2026-09 audit found the 4-phase server never validates freshness:
  `docs/refactoring/SECURITY_WIRING_AUDIT.md:262` "`handshake/server.go`
  contains no read of `Nonce` or `Timestamp` (grep)", and
  `docs/refactoring/DECISIONS.md:16` "No binary, example or external repository
  reaches `pkg/agent/handshake` ... Both external consumers establish sessions
  with `hpke` (1-RTT)". The alternative "harden and keep" was rejected at
  `DECISIONS.md:22` as "about the same effort as deleting, and leaves two
  protocols to maintain and audit for a path nobody uses".
- Note on sequencing: HPKE landed on `main` in the same commit as the 4-phase
  documentation (`b6f40c2` 2025-09-28), so no tagged release ever shipped the
  4-phase handshake alone; v1.0.0's CHANGELOG lists both (`CHANGELOG.md:806-830`).
  `docs/handshake/README.md:31` still rates both "Stable" while the package is
  Deprecated. [High]
- Status: **deprecated** since `eff53f8` 2026-09-12 (#295):
  `pkg/agent/handshake/doc.go:20-27` "Deprecated: ... scheduled for removal
  (docs/refactoring/BACKLOG.md, D-06). Its server does not validate the Nonce
  and Timestamp fields". Planned removal: "deprecate v1.6, remove v1.7"
  (`docs/refactoring/BACKLOG.md:76`, D-06; `DECISIONS.md:18`). Only non-test
  importer: `internal/session_creator.go` (package `sessioninit`, itself
  Deprecated, `internal/session_creator.go:19-23`).

### HPKE 1-RTT handshake (`pkg/agent/hpke`)

- Introduced: dev `363a481` 2025-09-27 "feature: add RFC 9180", `a2974b0`
  2025-09-28 "feature: support HPKE", `50f3d13` 2025-09-28 "clearing
  directional keys and sanitizing error messages"; on `main` via `b6f40c2`
  2025-09-28 (#25).
- Documents: `docs/dev/SAGE-secure-session-communication-ko.md` (`b6f40c2`),
  `docs/handshake/hpke-based-handshake-en/ko.md`, `hpke-detailed-ko.md`,
  `docs/handshake/README.md` (`9e325f6` 2025-10-11), `docs/adr/002-hpke-selection-rationale.md`
  (`5e3b279` 2025-10-26, i.e. one month after the code; six alternatives
  listed at `:207-315`).
- Later changes: `aca9347` 2025-10-12 (#66/#68) "add pinning and allowsuites"
  (CHANGELOG v1.0.2 `:716-720` calls it "Critical security improvements" without
  detail); `620d6a0` 2025-10-30 (#124) `CompositeVerifier` for secp256k1 +
  Ed25519 ("HPKE only supported Ed25519, blocking Ethereum agent
  communication"); `da50b62` 2025-11-02 (#137) KME -> KEM naming. 2026-09:
  `c1b8f67` (#267) server rejects an init whose `respDid` is not its own
  (`pkg/agent/hpke/server.go:255`); `f35feb2` (#268) response signed over RFC
  8785 canonical JSON ("previously the client re-encoded a Go struct, which no
  other language could reproduce"); `3f0902e` (#269) single `ErrInitRejected`
  to the peer (`server.go:64-73`); `2c48ceb` (#281) `ServerOpts.ReplayGuard`
  replaces the O(n) `nonceStore`; `aa03576` (#284) verifiers delegate to
  `keys.VerifySignature` and are Deprecated; `9827cec` (#303) test vectors.
- Status: **active**, the only session-establishment path for new code.

### Nonce and replay protection

Three generations, each documented before it was wired:

1. 2025-08 (`55cdf6d`, from dev `57021ec` 07-31 and `f022374` 08-10):
   `core/message/nonce.Manager` (global TTL map behind the RFC 9421 envelope
   `Verifier`) and `session.NonceCache` (`keyid -> nonce -> expiry`) with
   `Manager.ReplayGuardSeenOnce`. `docs/dev/security-design.md:227` records the
   intent: "each request is identified by (kid, nonce) and the server's nonce
   cache rejects reuse". The 2026-09 audit found the cache had "zero callers"
   (`SECURITY_WIRING_AUDIT.md:69`) and that `nonce.Manager`'s check-then-mark
   was non-atomic and global (`:105`).
2. 2025-09-28 (`b6f40c2`): HPKE server `nonceStore` keyed `ctxID|nonce` with a
   +-2 minute timestamp window; correct but O(n) per call (`:33`).
3. 2025-10-11 v1.0.0 CHANGELOG claims "Timestamp and nonce validation for
   replay attack prevention" for RFC 9421 (`CHANGELOG.md:803`) and "Nonce cache
   for replay attack prevention" (`:829`); `hpke-based-handshake-en.md:224`
   says RFC 9421 requests "reject duplicate Signature-Input nonces via
   ReplayGuardSeenOnce(kid, nonce)". The audit's test showed the HTTP verifier
   (the path the MCP examples use) "accepted the same nonce-bearing request 3x"
   (`SECURITY_WIRING_AUDIT.md:258`).

Why it was added [High]: the threat model in `docs/dev/security-design.md:57-60`
("재전송 공격 ... 타임스탬프 + nonce + 메시지 ID 검증") and the RFC 9421
`nonce`/`created` parameters; the design assumed the session `kid` doubles as
the RFC 9421 `keyid`, so replay state was placed next to sessions.

How it changed shape (2026-09) [High]:

- `8ab096e` 09-11 (#221): `HTTPVerifier.VerifyRequest` records nonces per
  `keyid`, bounds `created` by a clock-skew window, enforces
  `RequiredComponents`; `StrictHTTPVerificationOptions` requires a nonce and
  body digest; `nonce.Manager` gains atomic `CheckAndMark` and `Close`.
- `e7c713c` 09-11 (#265): session ciphertexts gain an 8-byte sequence number
  as AEAD associated data and a 1024-entry sliding window
  (`pkg/agent/session/session.go:111-131`, `ErrReplayedMessage`,
  `ErrStaleMessage`). Wire format `nonce || ciphertext` -> `seq || nonce || ciphertext`.
- `2c48ceb` 09-12 (#281): one contract, `session.ReplayGuard`
  (`CheckAndMark(scope, nonce)`, `pkg/agent/session/replay.go:26-38`) with
  `MemoryReplayGuard`; scopes are keyid (HTTP), agent DID (envelope), HPKE
  context id, session key id. `NonceCache`, `rfc9421.NonceReplayGuard` and
  `NewVerifierWithNonceManager` remain as Deprecated wrappers
  (`pkg/agent/session/nonce.go:22-27`, `pkg/agent/core/rfc9421/verifier.go:61-69`).
- `eff53f8` 09-12 (#295): `core/message/{dedupe,order,validator}` deleted
  (no importers); `core/message/nonce` kept only behind the deprecated adapter.

What the current code still depends on [High]: `pkg/agent/core/rfc9421/verifier.go`
is the sole importer of `core/message/nonce` (for `nonceManagerGuard`);
`verifier_http.go:64-82`, `response_signer.go:77`, `hpke/server.go:56,116-117`
and `session/manager.go:37,52` all take `session.ReplayGuard`. Persistent
`storage.NonceStore` (memory, postgres) implements a different interface and
has no production caller (`BACKLOG.md:54`, B-16).

### Session key schedule and rekey (`pkg/agent/session`)

- Introduced with the handshake (`55cdf6d` 2025-08-17): HKDF-Extract from the
  ECDH shared secret and handshake salt into a session seed, then encryption
  and signing keys (`session.go:56,231,325,362`). Directional keys and key
  zeroisation on close: dev `50f3d13` 2025-09-28.
- Documentation before code: "automatic key rotation" was claimed in
  `CLAUDE.md` while "there is no rekey. Session ends at MaxMessages"
  (`SECURITY_WIRING_AUDIT.md:71`).
- `8ab096e` (#221): sessions created from an HPKE exporter secret had an
  all-zero signing key and nil AEAD; now derived from the exporter.
- `e7c713c` (#265): generation `g > 0` = HKDF-SHA256(seed, salt = session id,
  info = "sage-session-rekey-v1" || direction || g); `DefaultRekeyInterval`
  256 (`session.go:118-134`); only current and previous generation cached.
- `878932d` (#322): `Manager` returns `*SecureSession`; `Session` interface
  reduced to five methods and Deprecated.
- Status: **active**; the `docs/test/sections/SECTION_7_SESSION.md` (last
  2025-11-02) does not mention sequence numbers, windows or rekey.

### RFC 9421 HTTP message signatures (`pkg/agent/core/rfc9421`)

- Introduced `4b1ffe7` 2025-07-08 (#10), with `docs/core/rfc9421-en/ko.md`
  and `docs/core/rfc-9421-test.md` the same day.
- 2025-10: `8be4d6a` (#81) nil check; `620d6a0` (#124) `BodyIntegrityValidator`
  for Content-Digest ("Attacker could modify request body but leave
  Content-Digest header unchanged").
- 2026-09: `8ab096e` (#221) replay, skew, required components; `9cf8286`
  (#222) secp256k1 uses Keccak-256 on every path (rfc9421 used SHA-256,
  `SECURITY_WIRING_AUDIT.md:245`; older secp256k1 signatures no longer verify);
  `6bf8d90` (#266) response signing with `;req` binding and `ResponseSigner`
  middleware; `c1b8f67` (#267) `ExpectedDID`/`ExpectedKeyID`/`ExpectedAuthorities`;
  `f35feb2` (#268) P-256 detection and low-S; `3f0902e` (#269) digest buffer
  bound; `2c48ceb` (#281) `ReplayGuard`; `aa03576` (#284) single verifier.
- Status: **active**, "the only verification path shipped consumers use"
  (`8ab096e` body).

### DID registry contracts: SageRegistry V2 -> V4 -> AgentCardRegistry

| Step | Commit | Added | Removed | Rationale recorded |
|---|---|---|---|---|
| V1/V2 | `55cdf6d` 2025-08-17 (#17); `1567112` 2025-08-24 (#19) | `SageRegistry.sol`, `SageRegistryV2.sol`, `docs/audit/*` (2025-10-11) | - | Audit package described "SageRegistryV2 + UUPS contracts and HMAC-based signatures" (archive banner, `docs/archive/2026-09/audit/README.md:3`) |
| V3 | `9e325f6` 2025-10-11 (#33) | `SageRegistryV3.sol` | - | `642cb87` skips it in coverage for "stack depth issues"; never referenced by Go code found |
| V4 | `69ee16c` 2025-10-18 (#95), v1.1.0 | `SageRegistryV4.sol`, `types_v4.go`, `a2a.go`, `contracts/MULTI_KEY_DESIGN.md` | `SageRegistry.sol`, `SageRegistryV3.sol` | "multi-key storage to support multi-chain AI agents ... as per Google A2A protocol requirements"; "Fix P0 Critical Ed25519 bypass in SageRegistry.sol" (commit body) |
| V2 removed | `b5b8cda` 2025-10-22 (#110) | `a2a_proof.go`, `key_proof.go` | `SageRegistryV2.sol`, `SageRegistryTest.sol` | - |
| AgentCardRegistry | `965aa4f` 2025-10-26 (#114) contract; `0c29697` 2025-10-28 (#122) Go client; `7111367` (#123) KEM key; v1.5.0 | `AgentCardRegistry.sol`, `AgentCardStorage.sol`, `AgentCardVerifyHook.sol`, `sage-did commit/register/activate`, `docs/AGENTCARD_MIGRATION_GUIDE.md` | `SageRegistryV4.sol`, `clientv4.go` | `AGENTCARD_MIGRATION_GUIDE.md:21` "Vulnerable to front-running attacks"; `:33-34` "Commit-reveal prevents front-running attacks ... Sybil Resistance: Activation delay and stake requirement"; CHANGELOG `:267-271` |
| Sepolia | `d78e803` 2025-11-03 (#139) | deployed addresses | - | - |
| 2026-09 | `d0af34f` (#223) ERC-8004 admin owner-only; `9328cdc` (#261) resolve only verified keys, honour revocation; `0a5ab21` (#274) binding drift check; `b04477b` (#306) contracts moved to `sage-contracts`; `d7e131d` (#321) | `.contracts-version` (`2ff992e`), `AgentCardClient` as the Ethereum chain client | `contracts/` directory | `CHANGELOG.md:18`: `EthereumClient` "packed SageRegistryV2 function signatures against the AgentCardRegistry ABI, so Register, Update and Deactivate could never succeed"; `Register` now returns `ErrCommitRevealRequired` |

Status: AgentCardRegistry **active** (external repository, pinned commit);
V2 client type kept as a Deprecated wrapper (`DECISIONS.md:70`, decision 5).
The KEM key field was "present in v1.3.1 but removed in v1.4.0" and restored in
v1.5.0 (`CHANGELOG.md:261`), so DID documents from v1.4.0 lack it. [High]

### A2A agent card (`pkg/agent/did/a2a.go`, `a2a_proof.go`)

- The gRPC A2A transport was removed first (`6695144` 2025-10-17, #90: "SAGE
  core should only define transport interfaces ... maintained separately in
  sage-a2a-go"). The card types came with V4 (`69ee16c` 2025-10-18: "Only
  export verified keys in A2A cards"); DID helpers and
  `SAGE_A2A_INTEGRATION_GUIDE.md` in `a071b08` 2025-10-19 (#96); proofs in
  `b5b8cda` 2025-10-22 (#110).
- 2026-09: `84621cc` (#264) `VerifyA2ACardProofWithDID` ("anyone could list
  their own key, sign, and obtain a card that 'verifies' for a DID they do not
  control"); `f35feb2` (#268) proofs over JCS canonical card; `555c8be` (#316)
  one `AgentMetadata` type, `AgentMetadataV4` a deprecated alias; guide
  archived by `f068212` (#294) because it was "built on SageRegistryV4,
  EthereumClientV4 and pkg/verifier, none of which exist".
- Status: **active**.

---

## 3. Documentation history

First and last commit per directory (`git log --follow`, tracked `docs/**/*.md`,
101 files):

| Directory | Files | First | Last | Notes |
|---|---|---|---|---|
| `docs/*.md` (root) | 13 | 2025-10-11 | 2026-09-12 | `INDEX.md` now generated (#291); `CODING_GUIDELINES.md`, `DATABASE.md` untouched since 2025-10-11 |
| `docs/core` | 4 | 2025-07-08 | 2026-09-12 | `README.md` untouched since 2025-07-08; `rfc-9421-test.md` since 2025-10-11 |
| `docs/crypto` | 2 | 2025-07-08 | 2025-10-11 | code (`pkg/agent/crypto/keys`) last changed 2026-09-12 |
| `docs/did` | 2 | 2025-07-08 | 2026-09-12 | |
| `docs/dev` | 2 | 2025-08-17 | 2025-11-02 | `SAGE-secure-session-communication-ko.md` last 2025-10-11 |
| `docs/handshake` | 9 | 2025-09-28 | 2026-09-12 | `handshake-en.md` never edited after creation; `README.md` links a non-existent `hpke-detailed-en.md` (`README.md:16,99,117`) |
| `docs/cli` | 2 | 2025-10-11 | 2025-10-11 | no `sage-did commit/register/activate` (added 2025-10-28), no `sage-verify` fold (2026-09-12) |
| `docs/contracts` | 3 | 2025-10-11 | 2025-10-28 | contracts left the repository 2026-09-12 |
| `docs/overview` | 8 | 2025-10-11 | 2026-09-12 | 4 of 8 Korean parts last touched 2025-10-11/12 |
| `docs/test` | 15 | 2025-10-11 | 2026-09-12 | `sections/SECTION_7_SESSION.md`, `SECTION_8_HPKE.md` last 2025-11-02 |
| `docs/adr` | 3 | 2025-10-26 | 2026-09-12 | written after the code they justify; dates corrected in #294 |
| `docs/refactoring` | 17 | 2026-09-11 | 2026-09-12 | analysis, audits, backlog, decisions |
| `docs/refactoring/v2` | 5 | 2026-09-12 | 2026-09-12 | repo plan, licensing, Rust alignment, version policy |
| `docs/archive/2026-09` | 16 | 2026-09-12 | 2026-09-12 | see below; original creation 2025-10-11 .. 2025-10-28 |

Documents whose description predates the change it describes (staleness
evidence; each pairs a doc's last commit with the package's last commit,
`git log -1 -- <pkg>`, all 2026-09-12):

| Document (last commit) | Describes | Divergence | Confidence |
|---|---|---|---|
| `docs/handshake/handshake-en.md` (2025-09-28) | 4-phase handshake as a live protocol | package Deprecated, server never checked nonce/timestamp | [High] |
| `docs/handshake/README.md:31` (2026-09-12) | "Maturity: Stable / Stable" | one column is Deprecated; line 12 of the same file says so | [High] |
| `docs/cli/cli-guide-en.md`, `-ko.md` (2025-10-11) | CLI commands | commit-reveal commands (2025-10-28) and `sage-verify deployment` (#298) absent | [High] |
| `docs/crypto/crypto-en.md`, `-ko.md` (2025-10-11) | key formats and signing | secp256k1 Keccak convention (#222), P-256/low-S (#268), fixed-length ECDSA encoding, `keys.KeyID` (#282) absent | [High] |
| `docs/test/sections/SECTION_7_SESSION.md` (2025-11-02) | session tests | no sequence number, replay window or rekey (#265) | [High] |
| `docs/test/sections/SECTION_8_HPKE.md` (2025-11-02) | HPKE tests | no JCS-canonical response, `respDid` check, `ErrInitRejected` (#267-#269) | [High] |
| `docs/DATABASE.md` (2025-10-11) | storage backends | sentinel errors (#284), unexported store types (#322) absent | [Mid] |
| `docs/contracts/*` (2025-10-28) | in-repo contracts | contracts moved to `sage-contracts` (#306) | [High] |
| `CHANGELOG.md` v1.0.0 (2025-10-11) | "nonce validation for replay attack prevention" (RFC 9421) | HTTP path had no replay check until #221 (2026-09-11) | [High] |
| `CLAUDE.md` (untracked) | "automatic key rotation" | no rekey until #265 | [High] |

The 2026-09 archive move (`f068212` 2026-09-12, #294): sixteen documents moved
to `docs/archive/2026-09/` with a banner ("Archived on 2026-09-12 (formerly
...). This document is kept for history only: ...") and links rewritten;
origins traced with `--follow`: `audit/*` from `9e325f6` (2025-10-11),
`SAGE_A2A_INTEGRATION_GUIDE.md` from `69ee16c`/`a071b08` (2025-10-18/19),
`dev/*`, `test/*`, `planning/*`, `maintenance/DOCUMENTATION_AUDIT_2025-10-26.md`,
`contracts/contracts-submodule-setup.md`. The commit body also records "twenty
statements across the tree contradicted the code". This is the third archive
episode: `614dbb8` 2025-10-18 "Remove all archived documentation" and
`0107563`/`333c15a` 2025-10-28 "Remove archive folder" deleted earlier archives
outright, so pre-2025-10-28 history survives only in git. [High]

---

## 4. The 2026-09 refactoring cycle (99 commits since 2026-09-10)

Grouped by theme, one line each; `#n` is the pull request.

Supply chain and dependencies

- `91fe084`..`4f43301` 27 Dependabot commits (#170-#214, rebased in one batch); `207d748` #245 and `5d09d8c` #255 apply the remaining batches; `5319e3e` #259 redis image; `0e81f2e` #293 Java SDK group.
- `66d5157`, `67f7ba7` (vitest 5), `14283ce` (Alpine packages), `e98b42b` (withdrawn Trivy) direct fixes for open alerts.
- `c277b7e` #256 Go 1.26.8, `go mod verify`, `--ignore-scripts`, govulncheck; `5f176f0` #257 base images by digest, cgo build dropped; `5493589` #262 builder image follows the toolchain minor.
- `fe710a4` #292 Dependabot ignores majors for experimental SDKs; `df275a2` #301 LGPL-3.0 metadata aligned; `9b0f7a4` #285 SDKs marked experimental and built in CI.

Security wiring (the audited paths)

- `8ab096e` #221 replay, skew, required components on the HTTP path; exporter-derived session keys; DID manager wiring; CLI key loading.
- `9cf8286` #222 secp256k1 Keccak convention everywhere; `1ebd12a` #286 accept high-S on verify.
- `d0af34f` #223 ERC-8004 administration owner-only; `9328cdc` #261 only verified keys, revocation honoured; `4a06adf` #263 mock resolver removed.
- `84621cc` #264 A2A proof bound to on-chain DID; `e7c713c` #265 session seq/window/rekey; `6bf8d90` #266 response signing.
- `c1b8f67` #267 recipient binding; `f35feb2` #268 JCS, P-256, low-S; `3f0902e` #269 body limits, origin checks, header override stop; `ba5b36c` #270 config locking, nil options.

Single sources of truth and layering (`REFACTORING_DESIGN.md` phases 1-5)

- `f61995e` #275 telemetry to `pkg/telemetry`; `ea6309d` #276 `crypto.Manager` removed; `6fe1e51` #277 metrics interface, health logger; `a3dd4a1` #278 EVM provider under `pkg/blockchain/ethereum`; `e954265` #279 no `init()` registration.
- `55c70f0` #280 one wire codec, chain parser, version; `2c48ceb` #281 one replay guard; `6c7d45e` #282 one key id and address derivation; `8217472` #283 one network preset table; `aa03576` #284 one signature verifier, `KeyIDBinder`, storage sentinels.
- `eff53f8` #295 cgo lib, random tester, unused message packages removed; `handshake` deprecated; `403f1d4` #298 helpers under `internal`, `deployment-verify` folded into `sage-verify`; `81e9a40` #299 Windows disk probe.
- `555c8be` #316 one `AgentMetadata`; `332e2b1` #320 trimmed `did` interfaces, one creator registry; `d7e131d` #321 `AgentCardClient` as the Ethereum chain client; `878932d` #322 session/chain/storage surfaces trimmed.

Repository split and cross-repository policy

- `9827cec` #303 `sage-vectors` generator; `c2e3603` #305 vectors verified against `sage-spec` in CI; `5ab9c7e` #307 `pkg/vectors`.
- `b04477b` #306 contracts moved to `sage-contracts`, `.contracts-version` pin, bindings from published ABI.
- `d88fc5b` #300 v2 repository plan and licence review; `c60d1e9` #302 A-16 closed, Actions policy blocker; `ba8d9b7` #318 version policy and compatibility matrix.

Rust core alignment

- `a3e9287` #309 `rs-sage-core` alignment plan; `274ff7d` #310 progress through did/A2A; `6bcb806` #317 alignment recorded complete at vector level, live interoperability (F-03b) left open.

CI

- `fca9c5c` least-privilege permissions, manual-only load tests, security policy; `60381d6` #224 every Action pinned to a commit SHA; `414e086` #254 scanners blocking and pinned.
- `9bfa54f` #271 reproducible builds, signed and attested releases; `0fdaaf7` #272 depguard layer rules and storage conformance suite; `0a5ab21` #274 bindings generated from artifacts with drift check.
- `b9489bf` #297 feature verification points at session/rfc9421 replay tests; `f660cd2` #296 code owners, pre-commit, Go version gate; `58cf20c` #319 shorter PR checks, stable required job names.

Documentation of the cycle

- `a7ed062` code graph tool, PR log, refactoring design; `2f15617` feature map, decisions; `6f1415b` strategy, security wiring audit, supply-chain audit, docs graph; `8bcdd67` backlog with IDs.
- `004c9ba` #287 examples do what their READMEs claim; `2e3f5af` #291 generated `docs/INDEX.md`; `f068212` #294 archive and contradiction fixes; `beac19a` #308 F-01/02/05/07 completion recorded.

---

## 5. Facts vs Opinions

**Fact** (verified in git or the tree)

- 4-phase handshake: dev `a1e5a48` 2025-07-31, `main` `55cdf6d` 2025-08-17; Deprecated by `eff53f8` 2026-09-12; removal planned "deprecate v1.6, remove v1.7" (`BACKLOG.md:76`); only non-test importer `internal/session_creator.go`.
- HPKE: dev `363a481` 2025-09-27, `main` `b6f40c2` 2025-09-28, the same commit that added `docs/handshake/handshake-en.md`; ADR-002 dated 2025-10-26 was created in `5e3b279` on that date.
- `handshake/server.go` never read `Nonce` or `Timestamp` (`SECURITY_WIRING_AUDIT.md:262`); `HTTPVerifier.VerifyRequest` had no replay check before `8ab096e` (`:258`).
- Replay protection existed three times (`nonce.Manager`, `session.NonceCache`, `hpke.nonceStore`) until `2c48ceb` 2026-09-12 defined `session.ReplayGuard`; `core/message/nonce` is now imported only by `pkg/agent/core/rfc9421/verifier.go`.
- Session rekey and sequence numbers did not exist before `e7c713c` 2026-09-11; the wire format changed in that commit.
- Registry contract sources: V1/V3 deleted `69ee16c` 2025-10-18, V2 deleted `b5b8cda` 2025-10-22, V4 deleted `965aa4f` 2025-10-26, all contracts moved out `b04477b` 2026-09-12.
- `docs/handshake/README.md:16` links `hpke-detailed-en.md`, which does not exist; `:31` rates the 4-phase handshake "Stable".
- `docs/cli/*.md` (last 2025-10-11) do not mention `sage-did commit`.
- `CHANGELOG.md:548` reads `[1.1.0] - 2024-10-18`.
- 99 commits since 2026-09-10; no commits between 2025-11-03 and 2026-02-19 or between 2026-02-20 and 2026-09-11.

**Opinion**

- [High] The 4-phase handshake was superseded for three independent reasons recorded at different times: round-trip count (docs, 2025-09/10), lack of freshness checks (audit, 2026-09-11), and absence of any consumer (decisions, 2026-09-11). Removing it in v1.7 as planned carries low compatibility risk because nothing outside the repository imports it (`DECISIONS.md:9`).
- [High] Replay protection changed shape because it was originally designed around the session `kid` and never connected to the HTTP verifier; the 2026-09 fix moved the state to an interface scoped by the verifier's own identifier rather than by session.
- [High] The documentation set has a "claim precedes implementation" pattern (v1.0.0 CHANGELOG nonce validation, automatic key rotation, `ReplayGuardSeenOnce` in the HPKE guide). The 2026-09 audits were the first time claims were tested against callers.
- [Mid] The 4-phase design's bootstrap encryption with the peer's DID signing key exists because the registry then carried no KEM key; once `AgentCardRegistry` stored X25519 keys (2025-10-28) the HPKE Base mode became the natural replacement. No document states this dependency explicitly.
- [Mid] Nine documents listed in section 3 describe code that changed on 2026-09-11/12 and were not updated; the CLI and crypto guides are the highest-impact ones for users because they are the documents `docs/INDEX.md` points new users to.
- [Low] `SageRegistryV3.sol` (2025-10-11 to 2025-10-18) appears to have been an intermediate that never had a Go client; no commit body explains it.
