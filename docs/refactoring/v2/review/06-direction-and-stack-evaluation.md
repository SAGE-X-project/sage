# 06. Direction and stack evaluation: does the chosen technology fit the purpose?

Status: review synthesis, 2026-09-12. Read-only. Same tree as `05-progress-assessment.md` (`sage` `878932d`, `sage-spec` `a34dd49`, `rs-sage-core` `206bbbb`, `sage-gateway` `4e72668`, `sage-inspector` `05b890d`).

Sources: `04` (purpose), `07` (literature; cited as `07` x.y), `08` (implementation against common practice; cited as `08` §n / finding n.m), `09` (flows), `10` (inspector matrix), `02` (code graph), `03` (history), `BACKLOG.md`, `STRATEGY.md` §2-§6, `v2/REPO_PLAN.md`, `v2/VERSION_POLICY.md`, `v2/LICENSING.md`. Code facts that carry a verdict were re-checked by grep on 2026-09-12 and are marked "verified". Confidence [High]/[Mid]/[Low]; severity [critical]/[major]/[minor].

The purpose the stack is measured against (`04` §1, `STRATEGY.md:19`): requests and responses between agents and MCP servers that cannot be forged, replayed or altered; identity that a third party can resolve and check for revocation; encrypted sessions; deterministic bytes across implementations; adoption by unmodified agent clients.

Verdict scale: **keep** (the choice fits and the open work is execution), **keep with change** (fits, but a named change is needed for it to serve the purpose), **reconsider** (a documented alternative serves the purpose better or the choice is not earning its cost).

---

## 1. Per-choice evaluation

### 1.1 Go reference core under LGPL-3.0

- Problem solved: one implementation that the two external Go consumers import (`STRATEGY.md` §1), that generates the vectors, and that the CLIs, gateway and inspector are built on; Go is normative where the spec is silent (`00-overview.md` §6, cited in `09` §2a).
- Alternatives: Apache-2.0 relicensing (`STRATEGY.md:176`; rejected pending contributor consent, `LICENSING.md` §5); Rust as the sole reference core (implied by `STRATEGY.md` §9 alternative 2); Go `c-shared` as the SDK base (rejected, `STRATEGY.md` §9).
- Evidence for: consumers build against main (`STRATEGY.md` §1 fact); all 26 vectors generated here and consumed by Rust; layer gates green (`02` §1); `INSTALL.md` documents LGPL relinking (`01` §3).
- Evidence against: the Go vector check is self-referential, so a Go regression that regenerates vectors passes Go CI (`08` 8.2 [major]); "Go normative when silent" turns Go deviations into spec (`@request-target`, verified `canonicalizer.go:232-241`; check order in `09` §2a); LGPL is the reason `STRATEGY.md` §7 gave for SDK friction, and the SDKs are now planned on the MIT/Apache Rust core instead.
- Verdict: **keep with change** [High]. Keep Go as reference and LGPL as decided; remove the self-reference by ingesting a Rust-generated vector set and `rejected` lists (`08` §8 recommendation), and move each "Go is normative" gap into spec text as it is found (`10` §3 lists them). Trade-offs: Apache relicense buys embedding freedom at the cost of a consent round over every past contributor; Rust-only reference removes one core to maintain but orphans the Go consumers and the gateway.

### 1.2 Rust core for FFI and WASM (`rs-sage-core`)

- Problem solved: non-Go SDKs without embedding the Go runtime; browser/WASM reach (`STRATEGY.md` §2.3).
- Alternatives: Go `c-shared` (rejected: runtime in host, no browser path, `STRATEGY.md` §9); native implementation per language against the spec (allowed per SDK, not default, §9); TinyGo WASM (not discussed in any document).
- Evidence for: 26/26 vectors in CI (`ci.yml:30-35` verified); `cbindgen` header drift-checked; WASM build; ~52 `extern "C"` entries (`08` §7); MIT OR Apache-2.0; module layout mirrors Go (`02` §3).
- Evidence against: no consumer and no live exchange with Go (F-03b Open; VERSION_POLICY §4 rule 4 keeps it `0.x`); HPKE absent from FFI/WASM (verified: no `hpke` file under `src/ffi`), so the SDK path cannot open sessions; no `catch_unwind` (verified; `08` 7.2 [major]); key files without 0600 (`08` 1.1 [major]); responder trusts `sender_did` as an argument (`09` §4 c9); behavioural divergences from Go in ten places (`10` facts); it diverged once already for ten months (`REPO_PLAN.md` §2); CI red on main (BACKLOG F-08 note).
- Verdict: **keep with change** [High]. The choice is right for the reason `STRATEGY.md` §9 gives; the change is to make F-03b the gate for any SDK work (already policy) and to expose HPKE/session through FFI before that gate is attempted. Trade-offs: per-language native implementations avoid FFI packaging but multiply the cryptographic surface to audit (the four in-tree SDKs show the failure mode, `08` 7.4); Go `c-shared` gives one core but the runtime and browser problems recorded in §9.

### 1.3 Ethereum/Solana registries with commit-reveal-activate

- Problem solved: an identity any verifier can resolve and check for revocation without a CA; front-running of first-come registrations (`07` 3.14, 3.15; `AGENTCARD_MIGRATION_GUIDE.md:21,33` per `03` §2).
- Alternatives: `did:ethr`/ERC-1056 (event-based, no registration, `08` §6); ERC-8004 identity as ERC-721 (no commit-reveal, `07` 2.3); `did:web` or an ANS-style PKI directory (`07` 2.7); F3B-style encrypted mempool (`07` 3.16); pre-shared trusted-key tables (the gateway already does this: "Static entries win over the chain", `04` §5).
- Evidence for: resolver enforces `is_active` and `verified`, honours revocation (B-06 merged; `09` §2b); commit-reveal is sound in the modelled setting (`07` 3.15); the chain-free Verifier level exists and is what the gateway ships (`00-overview.md` §4 verified).
- Evidence against: Ethereum has the highest cost and latency of benchmarked ledger DID methods and SAGE adds three transactions plus a 1-60 min window and a 1 h delay (`07` 4.4, 3.15; `08` 6.6); the on-chain `verified` flag for Ed25519 is a length check (`AgentCardRegistry.sol:439-442` verified; `08` 6.1 [major]); PoP not bound to chain id or registry (`08` 6.2 [major]); the CLI commit path cannot succeed (`08` 6.5); the Solana program has placeholder IDs and no CI build (C-04); the ERC-8004 reputation registry is deployed but contradicted as a trust signal (`07` 2.4, 4.6); Sepolia still runs the pre-fix validation registry (`REPO_PLAN.md` §5); no document states that the chain is optional while the README leads with "blockchain-based" (`04` §5, `README.md:12` verified).
- Verdict: **keep with change** for Ethereum AgentCardRegistry; **reconsider** Solana and the ERC-8004 reputation/validation registries [High]. The change: state the chain as the anchor for `did:sage` and as optional for verification (Verifier level with static keys), earn the `verified` flag (PoP in `key_policy.go`, chain-bound), and stop presenting ERC-8004 reputation as an input. Trade-offs: `did:web` costs nothing and needs no wallet but rests on DNS/CA trust and gives no commit-reveal; an ANS-style directory gives fast revocation at the cost of running a CA; keeping Solana costs a second on-chain program to audit for zero current consumers.

### 1.4 RFC 9421 HTTP Message Signatures

- Problem solved: per-message provenance and integrity that survives proxies and needs only headers on the client side; the only mechanism shipped consumers use (`03` §2 RFC 9421 status; `08` §2 exercised).
- Alternatives: SigV4-style HMAC (no non-repudiation, no nonce, `08` §2 common practice); mTLS (transport-bound, no per-message proof through intermediaries); draft-cavage as Mastodon used (`08` §2); a JSON-RPC `_meta` envelope for MCP (`STRATEGY.md` §2.1, unwritten).
- Evidence for: IETF Proposed Standard with vectors (`07` 3.1); request and response signing with `;req` binding live in the gateway (`09` §2a); strict options close body-swap and replay (B-01, B-09 merged); three request vectors and one response vector pass in both cores.
- Evidence against: two interoperability-breaking deviations, `@request-target` with method and re-serialised `@signature-params` dropping `tag` (`08` 2.1, 2.2 [major], verified); hand-rolled Structured Fields parser (`08` 2.3); `X-SAGE-DID` MUST unchecked in the core (`09` §4 a6); no negative vectors (`10` §3); it covers HTTP only, and the MCP servers that Claude Code, Codex and Hermes launch are mostly stdio, which has no HTTP headers (gateway README lists stdio bridging as not done, verified).
- Verdict: **keep** [High], with two conditions: fix 2.1 and 2.2 before any external peer (cost low, breaks only Go signatures over `@request-target`, which nothing shipped covers, `08` §2), and add the MCP binding for stdio (see 1.10). Trade-offs: adopting `yaronf/httpsign` or `dunglas/httpsfv` removes a parser class of bugs for one dependency and one vector re-check; mTLS would be simpler to operate but cannot bind a tool result to a server DID through a proxy.

### 1.5 HPKE 1-RTT handshake instead of TLS, Noise or mTLS

- Problem solved: an application-layer session between two DIDs, independent of transport and proxies, using the responder's registry KEM key, one round trip, forward secrecy from `ephC`/`ephS` (`docs/handshake/hpke-based-handshake-en.md:14-22` per `03` §2; ADR-002).
- Alternatives: TLS 1.3 with raw public keys (misbinding hazard, `07` 3.3); Noise NK/IK (verified patterns and tooling, `07` 3.6-3.8); HPKE Auth mode instead of Base + signatures (`08` §3 common practice); mTLS (endpoint-bound); KEMTLS-style KEM authentication (`07` 5.3).
- Evidence for: RFC 9180 is a proven primitive and the suite is exactly the one with RFC vectors (`07` 3.4, 3.5); mainstream libraries in both cores (`08` §3); sound combiner with all-zero rejection; thorough tamper/replay/downgrade tests (`08` §8); six derivation vectors; shape is NK plus SIGMA-style signatures, a known-good family (`08` §3 structural assessment [Mid]).
- Evidence against: not on any live path (verified; `05` §3 rank 1); the composition (HPKE base + ephemeral DH + Ed25519 + ack tag) has no formal analysis (`07` §6 "Unsupported"); DID/key-hash misbinding property unverified (`07` 3.3); cookie after public-key work (O-6, verified `server.go:147,159`, [major]); traffic keys specified and unused (`08` 3.2); no initiator key confirmation (`08` §3); first message has NK's replay property, mitigated by the per-context nonce guard and 2-minute window only.
- Verdict: **keep with change** [Mid]. Keep the design; frame it as a Noise-like pattern so the existing verification tooling applies, run a Verifpal/ProVerif model before spec 1.0 (`08` §8: "a few dozen lines"), fix the cookie order, and decide whether it is a product (wire into the gateway) or experimental (label it). Trade-offs: moving to Noise proper gives verified patterns and libraries but re-specifies the wire format (MAJOR bump) and loses the HPKE exporter alignment with OHTTP/MLS practice; TLS RPK gives ubiquity but no path through HTTP proxies and the misbinding problem; Auth-mode HPKE removes one signature but requires the initiator's static KEM key in the registry, which `did:sage` records do not carry for initiators.

### 1.6 ChaCha20-Poly1305 session layer (seq header, 1024 window, rekey per 256)

- Problem solved: an AEAD record layer with replay and reordering rejection after the handshake (B-08; spec 05).
- Alternatives: AES-256-GCM mandatory (`STRATEGY.md:62`, overturned by `REPO_PLAN.md` §4); TLS/QUIC nonce = IV XOR seq (`07` 3.11; `08` 4.3); Noise `REKEY(k)` chaining (`08` 4.2); reuse of the TLS record layer.
- Evidence for: RFC 4303/9001/9147 pattern for the window (`07` 3.9-3.12); constant-time in software on every platform; both cores agree on the vectors; random nonce is safe at 256 records per key (`08` opinions [Low]).
- Evidence against: 0 % live (`05` §2); directional entry points bypass lifetime limits (`08` 4.1 [major], verified); no forward secrecy between generations (`08` 4.2); 20 bytes per record of overhead where TLS/QUIC spend 0 (`08` 4.3); shared and directional keys share one counter and window against spec 05 §2 (`08` 4.4); Go and Rust `EncryptAndSign` records are not interoperable (`08` 4.5); Rust minimum record length differs (`09` §4 d3); ChaCha20-Poly1305 is absent from WebCrypto, which was the reason `STRATEGY.md` §2.1 preferred AES-GCM for SDK reach.
- Verdict: **keep** [High] for the AEAD (the decision is recorded and vectored; the WebCrypto argument is moot once TypeScript binds the Rust WASM); **keep with change** for the record format: fix the lifetime checks now (no wire change), and if a v2 record is ever opened, use counter-derived nonces and chained rekeys. Trade-offs: AES-GCM mandatory restores WebCrypto/JCA reach at the cost of timing dependence on platforms without AES-NI; IV XOR seq saves 12 bytes per record and removes one uniqueness mechanism at the cost of a MAJOR spec bump and losing the "nonce independent of seq" property the vectors pin.

### 1.7 JCS (RFC 8785) for signed JSON

- Problem solved: one byte form for the A2A card proof and the HPKE ack envelope so that any language reproduces the signed bytes (`STRATEGY.md` §3; B-11 merged).
- Alternatives: sign the transmitted bytes and forbid re-serialisation (no canonicaliser; every intermediary must preserve bytes); CBOR/COSE (not discussed in any document); maintained libraries `gowebpki/jcs` and `serde_jcs` (`08` §5).
- Evidence for: four vectors including RFC Appendix A and Unicode sorting pass in both cores; both signed objects verify over received bytes re-canonicalised (`09` §2c step 7).
- Evidence against: RFC 8785 is an Independent Submission (`07` 3.2); two hand-written parsers; Go accepts duplicate keys and lone surrogates that Rust rejects (`08` 5.1 [minor]; `10` x-12); Rust re-serialises a typed struct for the HPKE envelope, so a Go responder adding a member breaks Rust clients (`08` 3.3); the number and Unicode hazards `07` 3.2 warns about have no edge vectors.
- Verdict: **keep** [High]; replace both parsers with the maintained libraries and add the `jcs/edge-cases` rejected vector (`10` §3). Trade-off: two dependencies and one vector re-verification against roughly 600 lines removed (`08` §5); "sign bytes as sent" would remove the canonicaliser entirely but JSON-RPC proxies and MCP hosts routinely re-serialise, which is the case the gateway must survive.

### 1.8 The `did:sage` method

- Problem solved: a chain-anchored identifier that resolves to a key list with `verified` flags, a KEM key and an active flag; DID Core shape for interoperability with DID tooling (ADR-003 per `04` §2).
- Alternatives: `did:ethr` (ERC-1056 events, existing resolvers, `08` §6); `did:pkh` (generative, no rotation); `did:web`; ERC-8004 ERC-721 identity with off-chain registration JSON (`07` 2.3); `did:key` (rejected in ADR-003 for lacking anchoring).
- Evidence for: the record carries what the handshake needs (KEM key) and what the verifier needs (`verified`, `is_active`), and the Go resolver enforces both (`09` §2b); five vectors for grammar and PoP; Rust parses the same grammar.
- Evidence against: no DID method specification or DID Document projection, so standard resolvers cannot consume it (`07` 2.1 gap; `08` 6.3; O-8); identifier accepts any bytes (`08` 6.4); key selection, `#key-n` binding, cache TTL and registration timing are silent in the spec (`09` §4 b4; `09` opinions); one method string spans two chains with different key conventions; Rust has no chain resolver, so a Rust peer cannot resolve a `did:sage` at all (`02` §3); the method's value over `did:ethr` (commit-reveal, KEM key, `verified`) is exactly the part `08` §6 finds partly unearned.
- Verdict: **keep with change** [High]. Publish the method specification (resolve, update, deactivate, DID Document projection, key selection, fragment semantics) as an addendum to spec 06, and give Rust a resolver interface that the gateway's cache can back. Trade-offs: `did:ethr` brings existing resolvers and libraries but no KEM key, no `verified` semantics and no commit-reveal; `did:web` is trivial for enterprises but depends on DNS and TLS trust and cannot express revocation beyond document removal.

### 1.9 A2A agent card format with proof

- Problem solved: publish an agent's keys and endpoint in the shape Google A2A clients discover, with a signature that binds the card to a DID (`03` §2 A2A; spec 07).
- Alternatives: the DID Document itself as the card (W3C); a Verifiable Credential attestation (`07` 2.2); ERC-8004 registration JSON (`07` 2.3); A2A's default unsigned card with transport OAuth (`07` 1.3, 1.4).
- Evidence for: A2ABreak and the MAESTRO analysis motivate signed cards and identity bound to every message (`07` 1.3, 1.6); chain cross-check exists (`VerifyA2ACardProofWithDID`, B-07); one vector; gRPC A2A transport correctly moved out (`03` §2).
- Evidence against: `created` sits inside `proof` and is unsigned and unchecked (`08` §6); Go misses two spec MUSTs (hex/base58 equality, key type vs proof type; `09` §4 b7; `10` b-n9, b-n10); only an Ed25519 vector; 2019/2020 suite names; `ValidateA2ACardWithProofAndDID` has no caller (`02` §1); `did/a2a` move blocked by import direction (D-05).
- Verdict: **keep** [High]; close the two MUSTs and add the secp256k1 and rejected vectors (`10` §3). Trade-off: a DID Document projection (1.8) could serve both A2A discovery and DID tooling from one object; a VC-based attestation would make `verified` portable across registries at the cost of a second signature format to support.

### 1.10 Gateway proxies for MCP clients

- Problem solved: adoption by clients that are configured, not programmed (Codex, Claude Code, Hermes): one binary, one config line, no SDK (`STRATEGY.md` §2.5).
- Alternatives: SDK middleware inside each client (needs client code changes); server-side library only (protects one direction); MCP's own OAuth authorisation (bearer tokens; orthogonal, `07` 1.4); a stdio wrapper (gateway roadmap).
- Evidence for: the skeleton verifies and signs, resolves against static keys or chain with a TTL cache, signs responses, ships recipes and tests (`sage-gateway/pkg/gateway/{keyfile,resolve,sign,verify}` verified; BACKLOG F-05); it is the only live consumer of the RFC 9421 path (`08` §2 exercised); the literature supports signing tool results and descriptions for in-transit alteration (`07` 1.2, 1.8, 5.6).
- Evidence against: HTTP only; no stdio bridge, no HPKE, no TLS, no rate limiting, no capability checks (README:57-58 verified; `08` §7); label chosen by map iteration (`08` 2.5); recipes not executed in any review (`01` §3 [Mid]); the strategy gate "an unmodified MCP client exchanges signed calls and results" is not automated (`04` §4); the MCP binding it implements is not in the spec, so the gateway is the de facto profile (`01` opinions); signed-yet-malicious tool content is not addressed and cannot be (`07` 1.7, 1.8).
- Verdict: **keep** [High]; it is the shortest path to the purpose's adoption clause. Changes: stdio bridge and a spec chapter for the MCP binding (`_meta` envelope vs transport headers is the decision `STRATEGY.md` §10 deferred), TLS policy, and an automated end-to-end gate. Trade-offs: SDK-only adoption avoids an extra local process but requires every client to be programmed; the gateway adds a hop and a locally trusted process, which is the same trust the client already places in its MCP servers.

### 1.11 Separate specification repository with vectors

- Problem solved: "deterministic verification cannot live in a library" (`STRATEGY.md:45`); one normative text and a byte-exact conformance oracle for every core (`07` 5.2, 5.4, 5.5).
- Alternatives: keep the spec under `docs/` in `sage`; an IETF-style Internet-Draft; a formal model as the primary artefact (`07` 5.1-5.3).
- Evidence for: the nine chapters are the only CURRENT protocol text anywhere (`01` §5); every cited Go file and constant matches the tree (`01` §3 notes); 26 vectors run in three CIs; the Rust alignment was driven by them (`REPO_PLAN.md` §2 vs `RS_SAGE_CORE_ALIGNMENT.md`).
- Evidence against: untagged, consumed at `ref: main` (VERSION_POLICY §5; `test.yml:182` verified); negative vectors only for `did/parse` (`10` facts); chapter 08 has no vector; the MCP binding is missing; the spec describes unused key material (04 §4) and a derivation the HPKE path skips (05 §1) (`09` opinions); eight open items O-1..O-8; Go's own check is self-referential (`08` 8.2).
- Verdict: **keep** [High]. Tag `1.0.0-draft.1`, pin the three checkouts, add the `rejected` schema and a Rust-generated set, resolve O-1..O-8 and write the MCP chapter before `1.0.0`. Trade-offs: an in-repo spec is simpler to keep in sync but loses the "no implementation imports the spec" property and the neutral licence; a formal model first would give proofs but no wire-level oracle for SDK authors.

---

## 2. Cross-cutting findings

### Where the stack is heavier than the purpose needs

| Weight | Evidence | Cost it carries | Conf. |
|---|---|---|---|
| Blockchain presented as the critical path | `README.md:12` "blockchain-based" (verified); 2025 diagrams put the chain at layer 1 (`04` §5); yet the Verifier conformance level needs no chain and the gateway prefers static keys (`00-overview.md` §4; `04` §5) | operators believe a wallet, RPC and three transactions are prerequisites for signature verification; onboarding delay of minutes to an hour (`07` 3.15) for a property (name front-running resistance) most deployments do not need | [High] |
| Two on-chain programs and three ERC-8004 registries | Solana program with placeholder IDs, no CI build (C-04); ERC-8004 reputation and validation registries deployed on Sepolia with no Go caller and a contradicted trust model (`07` 2.4; `REPO_PLAN.md` §5) | audit and redeploy surface for zero live consumers | [High] |
| Two cores plus four SDK reimplementations | `02` §3; `08` 7.4; D-08 | six cryptographic implementations in the organisation, one already diverged (TS SHA-256) | [High] |
| Deprecated surfaces still compiled | `handshake` (37 exported symbols), `core/message/nonce`, `session.NonceCache`, `crypto/vault`, `EthereumClient`, `hpke.*Verifier` (`02` §4) | audit surface and documentation drift (`01` §6 rows 1, 14) until 1.8.0 by policy | [High] |
| Algorithms with no production caller | RSA (mislabelled PSS, `08` 1.2), P-256 (verify-only vectors, randomised signing), `pkg/storage/postgres`, `pkg/oidc`, `pkg/health` | each is a spec open item or a dependency with no consumer | [Mid] |

### Where the stack is lighter than the threat model needs

| Gap | Threat-model statement | Evidence | Sev. | Conf. |
|---|---|---|---|---|
| No transport confidentiality on any live path | `ARCHITECTURE.md` assumes TLS at endpoints; README claims E2E encryption (`04` §1) | no `tls.Config` anywhere (verified); HPKE/session unwired; `http://` registered (`08` 7.1) | [major] | [High] |
| Identity trust flags unearned | "impersonation: DID-based identity with on-chain verification" (`04` §1) | Ed25519 `verified` by length check; PoP unchecked at resolve and not chain-bound (`08` 6.1, 6.2) | [major] | [High] |
| MCP over stdio uncovered | `STRATEGY.md` §4 rows "malicious tool result", "confused deputy" | gateway HTTP only; no MCP binding in spec | [major] | [High] |
| Verifier DoS | `STRATEGY.md` §4 "bounded, TTL-evicting nonce store", cookie before work | guards unbounded in memory (`09` §3); cookie after resolution (O-6); no rate limit (B-15) | [major] | [High] |
| No delegation chain | `07` 2.8, 1.6: multi-hop identity loss is the A2A failure mode | envelope carries sender and target DIDs only | gap, not defect | [High] |
| No formal analysis, no parser fuzzing in CI | new handshake composition (`07` §6); RFC 9421/JCS/DID parsers hand-written | `08` 8.3, 8.4; no fuzz workflow (verified) | [major] | [High] |
| Replay state lost on restart | replay rows of every threat model | in-memory guards only; persistent store unwired (B-16) | [minor] | [High] |

### Where two mechanisms overlap

| Mechanism A | Mechanism B | Relation | Recommendation | Conf. |
|---|---|---|---|---|
| RFC 9421 nonce guard (scope keyid, 5 min) | HPKE init nonce guard (scope ctx, 10 min); session seq window (1024); unused `NonceCache`, `storage.NonceStore` | four replay mechanisms with three scopes, one contract (`02` §2a); they protect different layers, so the overlap is in state stores, not semantics | keep the layered guards; delete the two unused ones (B-16); share one bounded store per process | [High] |
| HPKE exporter (spec 04 §3) | HPKE traffic keys and channel binding (04 §4) | second KDF over the same secret, vectored, never used (`08` 3.2; `09` §4 c5) | remove from spec and cores while still a draft (MAJOR bump is free before 1.0) | [High] |
| Combined seed used verbatim | `DeriveSessionSeed` salted derivation (spec 05 §1) | the vector tests a function the HPKE path skips (`09` §4 d1) | spec says which path uses which; or the code adopts one | [High] |
| Responder `sigB64` over the JCS envelope | ack tag HMAC over the transcript | both authenticate the responder; the tag additionally proves possession of the derived key (key confirmation), the signature binds the DID | keep both, document the distinct role of each in spec 04 §5 | [Mid] |
| Single-key schedule (`sage-session-keys-v1`) + `EncryptAndSign` HMAC | directional keys + AEAD | two key sets and a MAC-over-AEAD path on one session object with one counter (`08` 4.4, 4.5) | make the directional AEAD path the only record path; drop the MAC path from the spec | [High] |
| `WireMessage.signature` over the payload | RFC 9421 signature over the HTTP request | on the gateway path a handshake init is signed twice (`09` §2c step 2; spec 08 §1 vs 04) | spec states which signature is authoritative per transport | [Mid] |
| `X-SAGE-DID` header | `keyid` DID | duplicate identity carrier; core ignores the header (`09` §4 a6) | enforce equality in the core or drop the header from the profile | [High] |
| Deprecated 4-phase handshake | HPKE 1-RTT | superseded for three recorded reasons (`03` §2) | remove at the policy date; delete the docs that still rate it Stable | [High] |

---

## 3. Recommendations, ordered by impact

Each item names the decision the maintainer has to take, at most two options, with cost (S under a day, M days, L weeks) and the trade-off.

1. **Decide what "end-to-end encrypted" means on the live path.** Option A: wire HPKE + session into the gateway (gateway-to-gateway "HPKE session mode") with a Go-Rust exchange in CI (F-03b); cost L; trade-off: the largest single piece of work, but it makes rows 6, 7 and rank 1 of `05` real. Option B: reword `README.md:12`, `ARCHITECTURE.md` and CLAUDE.md to "authenticated, tamper-evident, replay-protected; confidentiality by TLS; HPKE sessions experimental"; cost S; trade-off: honest today, but it concedes half of the stated purpose until A happens. [High]

2. **State that the chain is optional for verification and required only for `did:sage` anchoring.** Option A: keep "blockchain-based" as the lead and require chain resolution for every DID; cost 0; trade-off: preserves the 2025 positioning and blocks adoption by anyone without an RPC and wallet. Option B: lead with the protocol, document the Verifier level with static keys, present the registry as the anchor for `did:sage`; cost S (docs) plus a documented static-key format; trade-off: the chain stops being the headline feature. [High]

3. **Fix the two interoperability-breaking RFC 9421 deviations and adopt an RFC 8941 parser before spec 1.0** (`08` 2.1, 2.2). Cost S-M. Decision: hand-rolled parser (no dependency, edge cases stay) vs `httpsign`/`httpsfv` (one dependency, vector re-check). [High]

4. **Specify the MCP binding and build the stdio bridge.** Decision: `_meta.sage` envelope over JCS of `method/params/result/id` (works on stdio and HTTP; needs JSON-RPC awareness) vs transport headers (HTTP only, reuses RFC 9421 unchanged). Cost M-L. Trade-off: the envelope covers the clients the strategy names; headers cover only HTTP MCP servers. [High]

5. **Earn the `verified` flag.** Verify PoP in `key_policy.go`, bind the challenge to `chainid || registry` (spec O-1). Cost M; one verification per resolved key, amortised by the gateway cache. Decision: off-chain PoP enforced by every resolver vs an on-chain Ed25519 oracle (`AgentCardRegistry.sol:441` comment); trade-off: resolver-side is immediate and chain-agnostic; on-chain is uniform but needs an oracle or TEE. [High]

6. **Model the handshake and fix its order.** Verifpal/ProVerif model of spec 04 with the misbinding property (`07` 3.3), cookie before resolution (O-6), traffic-keys section removed. Cost S-M. Decision: remove 04 §4 now (MAJOR bump while draft) vs keep as informative annex; trade-off: removal simplifies both cores and the vectors, the annex preserves a channel-binding hook nobody uses. [Mid]

7. **Make the vectors adversarial and two-directional.** `rejected` schema for rfc9421/hpke/jcs, init/ack envelope verify vectors, a Rust-generated set that Go ingests, pinned spec tag in the three CIs. Cost M. Decision: rejected lists in the existing JSON schema vs a mutation suite in the inspector (`10` §3 P3); trade-off: JSON lists are consumed by both cores' CI today, the mutation suite tests more but only Go. [High]

8. **Cut weight.** Remove `handshake`, `core/message/nonce` and the dead exports at the policy date (decision: honour the two-minor rule in VERSION_POLICY §4 vs D-06's 1.7; `02` §4 shows the conflict); rename or drop RSA (O-3); leave the Solana preset empty and stop advertising ERC-8004 reputation until a consumer and a Sybil filter exist (`07` 2.4). Cost S each. Trade-off: earlier removal breaks nothing known (`DECISIONS.md:9`) but breaks the written policy. [High]

9. **SDK path.** Decision: archive `sdk/*` now and create SDK repositories only after F-03b (as `REPO_PLAN.md` §1 already says) vs keep the four experimental trees building. Cost S to archive. Trade-off: archiving removes the diverged reimplementations and the maintenance of four CI jobs; keeping them preserves example code that no server serves. [Mid]

10. **Cheap correctness fixes with no decision needed:** session lifetime checks on directional entry points (`08` 4.1), `catch_unwind` in every Rust FFI entry (7.2), 0600 on Rust key files (1.1), WebSocket write mutex (7.3), inspector `-did`/`-authority` inputs and replay across captures (`10` §3 P1). Cost S each. [High]

---

## Facts vs Opinions

**Facts** (verified in the tree on 2026-09-12 or read in the cited documents)

- No package under `pkg`, `cmd`, `examples`, `internal` calls `hpke.NewServer`/`NewClient`; `pkg/agent/transport` has no RFC 9421 call; `pkg`, `cmd`, `internal` and `sage-gateway/cmd` have no TLS configuration.
- `pkg/agent/hpke/server.go` verifies the sender (`:147`) before the cookie (`:159`); `canonicalizer.go:232-241` prefixes `@request-target` with the method.
- `AgentCardRegistry.sol:439-442` marks Ed25519 keys verified after `signature.length == 64`; `key_policy.go` never calls `VerifyKeyProofOfPossession`.
- `rs-sage-core/src/ffi` has no HPKE file; `src/` has no `catch_unwind`; its CI runs the 26 `sage-spec` vectors; the spec is consumed at `ref: main` by `sage` CI.
- `sage-gateway` implements `keyfile`, `resolve`, `sign`, `verify`; its README lists HPKE sessions, stdio bridging, capability checks and rate limiting as not done.
- `00-overview.md` §4 defines the Verifier level as `crypto`, `jcs`, `rfc9421`, `did` with no session suite; `README.md:12` opens with "A blockchain-based security framework".
- `STRATEGY.md:62` recommended AES-256-GCM and `:176` Apache-2.0; `REPO_PLAN.md` §4 and `LICENSING.md` §5 decided ChaCha20-Poly1305 and LGPL-3.0.
- `07` found no formal analysis of the SAGE handshake or session layer, and reports Ethereum as the highest-cost ledger for DID lifecycle operations and ERC-8004 reputation as non-functional as deployed.
- No workflow in `sage` runs fuzzing; the only `rejected` vector list is `did/parse`.

**Opinions**

- [High] The stack fits the purpose; the mismatch is in what is presented as central. Signing over HTTP, JCS, the spec-with-vectors and the gateway are the parts doing the work today and each earns a keep. The blockchain is presented as the foundation but functions as an optional anchor, and the E2E layer is presented as delivered but is not on any live path.
- [High] The two changes that matter most are not cryptographic: decide whether the E2E claim is a product or a label, and decide whether the chain is a requirement or an option. Every other recommendation is execution.
- [High] Overlaps are cheap to remove while the spec is a draft (traffic keys, MAC path, seed derivation, header duplicate); after 1.0 each becomes a MAJOR bump.
- [Mid] The HPKE-based handshake is a defensible design that a Noise-style formal model would most likely confirm; until that model exists the project should not claim more than "built from proven primitives".
- [Mid] Solana and the ERC-8004 reputation registry cost more than they return at the current stage; deferring both loses nothing a consumer uses.
- [Low] Adopting maintained RFC 8941 and JCS parsers would remove more latent risk than any single finding in `08`; the vectors make the switch checkable in one CI run.
