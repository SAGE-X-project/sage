# 07. Literature review: where SAGE sits in the research landscape

Date: 2026-09-12. Scope: the SAGE protocol as described in `README.md`, `sage-spec/spec/00-overview.md` and `docs/refactoring/STRATEGY.md` (sections 1, 3, 4). Method: every work below was located by web search and its metadata confirmed by fetching the abstract page, RFC datatracker entry, or W3C/EIP page on 2026-09-12. Nothing is cited from memory. Confidence marks ([High]/[Mid]/[Low]) apply to the relevance judgement, not to the cited work itself.

The problem the review targets, restated from the three project documents: an agent that calls another agent or an MCP server needs (1) the caller and callee to be identified by a key that a third party can resolve and check for revocation (`did:sage`, on-chain AgentCardRegistry, ERC-8004 registries); (2) requests and responses that cannot be forged, replayed or altered (RFC 9421 over JCS-canonical bodies, `content-digest`, nonce plus `created`/`expires` window); (3) an encrypted session established in one round trip from HPKE plus an ephemeral X25519 exchange and Ed25519 signatures, with a monotonic-counter replay window over ChaCha20-Poly1305; (4) deterministic bytes so that Go and Rust cores and thin SDKs verify identically (byte-exact vectors). STRATEGY.md section 4 lists the threats: body tampering, replay, DID impersonation, malicious tool results, confused deputy via MCP call-backs, downgrade to unsigned, key compromise/rotation, verifier DoS.

Format per work: citation line; Summary; Shows; Relation to SAGE with confidence.

---

## 1. Security of LLM-based multi-agent systems and agent protocols

**1.1** Kong, D., Lin, S., Xu, Z., et al. "A Survey of LLM-Driven AI Agent Communication: Protocols, Security Risks, and Defense Countermeasures." arXiv:2506.19676, 2025. https://arxiv.org/abs/2506.19676
- Summary: Defines agent communication, sorts it into three classes (user-agent, agent-agent, agent-environment) in a three-layer architecture, and walks MCP and A2A through the layers listing vulnerabilities and countermeasures per layer, with experiments and a maintained resource list.
- Shows: the field treats transport-level authentication, message integrity and replay as one layer among several; most attacks it catalogues live above that layer (prompt and tool semantics).
- Relation [High]: supports SAGE's positioning as the transport/identity layer, and makes explicit that SAGE does not address the semantic layer (prompt injection content). Useful as the taxonomy to cite when scoping SAGE's threat model.

**1.2** Hou, X., Zhao, Y., Wang, S., Wang, H. "Model Context Protocol (MCP): Landscape, Security Threats, and Future Research Directions." arXiv:2503.23278, 2025; ACM TOSEM, DOI 10.1145/3796519. https://arxiv.org/abs/2503.23278
- Summary: Lifecycle model of an MCP server (creation, deployment, operation, maintenance; 16 activities) and a threat taxonomy across four attacker types (malicious developer, external attacker, malicious user, security flaw) with 16 scenarios and per-phase safeguards.
- Shows: MCP's threat surface includes server impersonation, tool-description tampering and unauthenticated servers; the paper recommends integrity and provenance controls for tool metadata.
- Relation [High]: supports STRATEGY.md section 4 "malicious tool result injected into an agent" and "confused deputy": signing tool results and tool descriptions with the server's DID is one of the safeguards this taxonomy calls for.

**1.3** Habler, I., Huang, K., Narajala, V. S., Kulkarni, P. "Building A Secure Agentic AI Application Leveraging A2A Protocol." arXiv:2504.16902, 2025. https://arxiv.org/abs/2504.16902
- Summary: Applies the MAESTRO threat-modelling framework to Google A2A, listing spoofing, task replay, privilege escalation and prompt injection, with emphasis on Agent Card management, task integrity and authentication.
- Shows: A2A's own security relies on transport-level OAuth/API-key schemes declared in the Agent Card; the card itself and task messages are the assets to protect.
- Relation [High]: directly motivates SAGE's signed A2A cards (`GenerateA2ACard` with key proof) and signed task envelopes. Also a checklist against which a SAGE paper can show which MAESTRO risks it closes.

**1.4** Louck, Y., Stulman, A., Dvir, A. "Improving Google A2A Protocol: Protecting Sensitive Data and Mitigating Unintended Harms in Multi-Agent Systems." arXiv:2505.12490, 2025. https://arxiv.org/abs/2505.12490
- Summary: Identifies token-lifetime, customer-authentication, over-broad scope and consent gaps in A2A; proposes consent orchestration, ephemeral scoped tokens and direct user-to-service channels; reports empirical reduction of data leakage.
- Shows: A2A authorization as shipped is bearer-token based, and short-lived scoped credentials measurably reduce harm.
- Relation [Mid]: partially orthogonal (authorization, not authentication) but argues for SAGE binding a session to a short-lived key id (`kid`) rather than to a long-lived bearer token. Suggests an evaluation: leakage under compromised token vs compromised session key.

**1.5** Anbiaee, Z., Rabbani, M., Mirani, M., Piya, G., Opushnyev, I., Ghorbani, A., Dadkhah, S. "Security Threat Modeling for Emerging AI-Agent Protocols: A Comparative Analysis of MCP, A2A, Agora, and ANP." arXiv:2602.11327, 2026. https://arxiv.org/abs/2602.11327
- Summary: Qualitative risk framework with twelve protocol-level risks across creation, operation and update phases, compared across MCP, A2A, Agora and ANP.
- Shows: the recurring "design-induced" gap is missing validation and attestation of executable components (tools, agent capabilities); none of the four protocols mandates cryptographic attestation of what an agent advertises.
- Relation [High]: supports SAGE's on-chain registration of the agent card and key hash; contradicts nothing, but note that SAGE also does not attest the code behind a capability (see the Binding Agent ID entry in the note after section 2, and 5.6, for works that try).

**1.6** Lotfi, A., Rahman, M. M., Karim, I., Bertino, E. "A2ABreak: Systematic Security Analysis of the A2A Protocol." arXiv:2609.10871, 2026. https://arxiv.org/abs/2609.10871
- Summary: Extracts a finite-state machine from the A2A natural-language specification and finds 11 vulnerabilities exploitable by a specification-compliant adversary: cross-client context injection via unprotected context identifiers, credential harvesting through identity loss in multi-hop delegation chains, exfiltration via rogue agents advertising unattested capabilities. Reports 73.3% precision against expert review, versus zero confirmed findings from an LLM-only baseline.
- Shows: A2A as specified does not bind context/task identifiers to an authenticated principal, and does not carry identity across delegation hops.
- Relation [High]: the strongest external evidence for SAGE's design choices (identity bound to every message, `kid` bound to a DID, direction and target in the envelope). Also contradicts any claim that A2A-compliance alone is a security property. Its method (spec-to-FSM, then property checking) is a reusable evaluation for SAGE's own specification.

**1.7** Lee, D., Tiwari, M. "Prompt Infection: LLM-to-LLM Prompt Injection within Multi-Agent Systems." arXiv:2410.07283, 2024 (ESORICS 2025 Workshops). https://arxiv.org/abs/2410.07283
- Summary: Demonstrates self-replicating prompts that spread agent-to-agent like a virus; the attack succeeds even when inter-agent messages are restricted; proposes "LLM Tagging" (marking message provenance) as a partial defence.
- Shows: message provenance is a useful mitigation, but authenticated transport does not stop infection when the sender is itself compromised.
- Relation [High]: bounds SAGE's claims. SAGE gives cryptographic provenance (who sent a message), which is the substrate LLM Tagging needs, but SAGE cannot claim to stop prompt injection. A paper should state this limit explicitly.

**1.8** Wang, Z., Gao, Y., Wang, Y., et al. "MCPTox: A Benchmark for Tool Poisoning Attack on Real-World MCP Servers." arXiv:2508.14925, 2025. https://arxiv.org/abs/2508.14925
- Summary: 45 live MCP servers, 353 tools, 1,312 adversarial cases in 10 risk categories; tool-poisoning success up to 72.8% (o1-mini), refusal rates under 3%.
- Shows: tool metadata is executed as instruction by current agents; the attack needs no transport compromise, only a malicious or altered tool description.
- Relation [Mid]: supports signing tool descriptions with the server DID (STRATEGY section 4) as a way to make alteration detectable, but shows a signed-yet-malicious description is still effective. Provides a benchmark a SAGE gateway could be run against to measure "altered-in-transit" versus "malicious-at-source" cases.

---

## 2. Decentralised identity for autonomous agents

**2.1** Sporny, M., Guy, A., Sabadello, M., Reed, D. (eds.). "Decentralized Identifiers (DIDs) v1.0." W3C Recommendation, 19 July 2022. https://www.w3.org/TR/did-core/
- Summary: Defines DID syntax, DID documents (verification methods, services), the controller model and the abstract resolution contract that DID methods implement.
- Shows: a method-agnostic document model that SAGE's `did:sage` resolver can target.
- Relation [High]: SAGE's `did:sage:<chain>:<addr>` and its resolver follow this model; a spec-level claim of DID Core conformance requires a published DID method specification for `did:sage` (resolution, update, deactivate), which the repository does not yet contain.

**2.2** Sporny, M., Thibodeau, T., Herman, I., Cohen, G., Jones, M. B. (eds.). "Verifiable Credentials Data Model v2.0." W3C Recommendation, 15 May 2025. https://www.w3.org/TR/vc-data-model-2.0/
- Summary: Issuer/holder/verifier model with embedded (Data Integrity) or enveloping (JOSE/COSE) proofs.
- Shows: a standard way to express third-party attestations about a subject key.
- Relation [Mid]: SAGE's `verified` flag on an agent card is an ad-hoc attestation; VCs are the standard equivalent (issuer = verifier hook or auditor). Adoption would make the "verified" claim portable across registries.

**2.3** De Rossi, M., Crapis, D., Ellis, J., Reppel, E. "ERC-8004: Trustless Agents." Ethereum Improvement Proposals, Draft, created 13 August 2025. https://eips.ethereum.org/EIPS/eip-8004
- Summary: Identity Registry (ERC-721 + URIStorage, agent URI to a registration JSON listing A2A/MCP endpoints and trust models), Reputation Registry (`giveFeedback()` by any client, no signature required, fixed-point score plus tags), Validation Registry (validator contracts: stake-secured re-execution, zkML, TEE oracles). Security section acknowledges Sybil inflation of reputation.
- Shows: an ERC-721 token as agent identity with off-chain card, and reputation as open, unsigned on-chain feedback.
- Relation [High]: SAGE deploys ERC-8004 registries alongside AgentCardRegistry. Where ERC-8004 leaves the card unsigned and feedback unauthenticated, SAGE adds key-proof-of-possession and commit-reveal; that difference is the claim to evaluate.

**2.4** Xiong, X., Li, Z., Wei, W., Wang, Q., Knottenbelt, W., Wang, Z. "Can Trustless Agents Be Trusted? An Empirical Study of the ERC-8004 Decentralized AI Agent Ecosystem." arXiv:2606.26028, 2026. https://arxiv.org/abs/2606.26028
- Summary: Measures ERC-8004 registrations on three chains: only 3-15% keep a valid endpoint; reputation values are non-comparable and unverifiable; 59-91% of reviewers show coordinated fraudulent activity; after filtering, most agents have no trustworthy feedback.
- Shows: the deployed ERC-8004 Reputation Registry "cannot function as a trust signal" as deployed.
- Relation [High]: contradicts using ERC8004ReputationRegistry scores as a trust input without Sybil-resistant filtering. Supports SAGE's staked commit-reveal registration as a cost on placeholder registrations, but that effect is unmeasured. Its measurement method (endpoint liveness, reviewer clustering) is reusable.

**2.5** Rodriguez Garzon, S., Vaziry, A., Kuzu, E. M., Gehrmann, D. E., Varkan, B., Gaballa, A., Küpper, A. "AI Agents with Decentralized Identifiers and Verifiable Credentials." arXiv:2511.02841, 2025; ICAART 2026. https://arxiv.org/abs/2511.02841
- Summary: Gives each LLM agent a ledger-anchored DID plus issued VCs; agents prove DID control at dialogue start and establish cross-domain trust. Prototype works but shows limits when the LLM itself drives the security procedure.
- Shows: DID+VC authentication at the start of agent dialogue is feasible; delegating the protocol steps to the LLM is unreliable.
- Relation [High]: supports SAGE's architecture decision to put the protocol in a library/gateway outside the model. Suggests an evaluation: compare protocol correctness when steps are executed by code versus by the model.

**2.6** Xu, M., Liu, X., Guo, Y., Liu, C., Zhang, Y., Cheng, X. "AgentDID: Trustless Identity Authentication for AI Agents." arXiv:2604.25189, 2026. https://arxiv.org/abs/2604.25189
- Summary: DID/VC-based identity with a challenge-response step in which a verifier checks the agent's execution conditions at interaction time; throughput evaluated with many concurrent agents.
- Shows: scalable DID authentication for agent populations is achievable, and throughput is a reportable metric.
- Relation [Mid]: closest peer system to SAGE's identity layer; a SAGE paper should compare handshake cost and verifier throughput against it. SAGE lacks the state-verification step.

**2.7** Huang, K., Narajala, V. S., Habler, I., Sheriff, A. "Agent Name Service (ANS): A Universal Directory for Secure AI Agent Discovery and Interoperability." arXiv:2505.10609, 2025. https://arxiv.org/abs/2505.10609
- Summary: DNS-inspired agent directory with PKI certificates for identity, registration/renewal lifecycle, capability-aware resolution and protocol adapters for A2A/MCP/ACP; JSON-Schema messages and a threat analysis.
- Shows: discovery and naming are being designed around X.509-style PKI, not blockchains.
- Relation [Mid]: alternative to SAGE's on-chain registry. A comparison axis for the paper: revocation latency, censorship resistance and operator cost of ANS-style CA registries versus on-chain registries (see section 4).

**2.8** South, T., Marro, S., Hardjono, T., Mahari, R., Whitney, C. D., Greenwood, D., Chan, A., Pentland, A. "Authenticated Delegation and Authorized AI Agents." arXiv:2501.09674, 2025. https://arxiv.org/abs/2501.09674
- Summary: Framework for authenticated, authorized and auditable delegation from humans to agents, extending OAuth 2.0 / OpenID Connect with agent credentials and metadata so that an agent's authority chain is verifiable.
- Shows: agent identity is incomplete without a delegation chain to a principal.
- Relation [High]: SAGE identifies agents but does not express who an agent acts for; A2ABreak's multi-hop identity loss (1.6) is exactly this gap. Candidate extension: carry a delegation credential in the signed envelope.

Additional identity works located (metadata verified, not reviewed in depth): Chan, A., Kolt, N., Wills, P., et al. "IDs for AI Systems." arXiv:2406.12137, 2024, https://arxiv.org/abs/2406.12137 (policy case for agent identifiers); Otsuka, T., Toyoda, K., Leung, A. "AI Identity: Standards, Gaps, and Research Directions for AI Agents." arXiv:2604.23280, 2026, https://arxiv.org/abs/2604.23280 (five gaps incl. recursive delegation accountability); Lin, Z., Zhang, S., Liao, G., Tao, D., Wang, T. "Binding Agent ID." arXiv:2512.17538, 2025, https://arxiv.org/abs/2512.17538 (binds operator and code hash via zkVM proofs); Braendgaard, P., Torstensson, J. "ERC-1056: Ethereum Lightweight Identity." EIP, 2018, https://eips.ethereum.org/EIPS/eip-1056 (`did:ethr` registry).

---

## 3. Standards the design builds on, and their analyses

### Signatures and canonicalisation

**3.1** Backman, A., Richer, J., Sporny, M. "HTTP Message Signatures." RFC 9421, IETF Proposed Standard, February 2024. https://datatracker.ietf.org/doc/rfc9421/
- Summary: Signature base built from covered components (derived components such as `@method`, `@authority`, `@path`, plus fields such as `content-digest`), with `created`, `expires`, `nonce`, `keyid`, `alg` parameters; multiple independently confirmed test vectors in the RFC.
- Shows: the signer chooses the covered set; integrity of the body is only covered if `content-digest` is included and separately verified.
- Relation [High]: STRATEGY.md section 3 records that SAGE currently verifies a signed request after body replacement; the RFC's own model explains why (uncovered component) and the fix (mandatory component set, verifier-side policy). The RFC's vectors are the natural first layer of SAGE's conformance suite.

**3.2** Rundgren, A., Jordan, B., Erdtman, S. "JSON Canonicalization Scheme (JCS)." RFC 8785, Independent Submission (Informational), June 2020. https://datatracker.ietf.org/doc/rfc8785/
- Summary: Canonical JSON via ECMAScript number serialisation, I-JSON subset and sorted property names.
- Shows: a deterministic byte form exists for JSON, but the document has no IETF standards standing.
- Relation [High]: supports the determinism requirement (STRATEGY section 3, "sign the bytes that are sent"). Risk to note in a paper: JCS number formatting and Unicode sorting are the classic cross-language divergence points; vectors must cover them.

**3.3** Moustafa, M., Sethi, M., Aura, T. "Misbinding Raw Public Keys to Identities in TLS." arXiv:2411.09770, 2024; Secure IT Systems (NordSec), Springer, 2025. https://arxiv.org/abs/2411.09770
- Summary: ProVerif analysis showing that TLS raw-public-key authentication permits identity misbinding (a key gets associated with the wrong endpoint identity); gives mitigations.
- Shows: authenticating a key is not the same as authenticating an identity; the binding must be inside the signed transcript.
- Relation [High]: SAGE authenticates raw keys resolved from a DID document. The handshake must include both DIDs and the resolved key hash in the signed/exported context (`info`/`exportCtx`) or it is exposed to the same misbinding. This is a concrete property to verify formally (section 5).

### HPKE and one-round-trip handshakes

**3.4** Barnes, R., Bhargavan, K., Lipp, B., Wood, C. "Hybrid Public Key Encryption." RFC 9180, IRTF CFRG Informational, February 2022. https://datatracker.ietf.org/doc/rfc9180/
- Summary: KEM/KDF/AEAD composition with base, PSK, auth and auth_psk modes, a secret export interface, and test vectors for DHKEM(X25519, HKDF-SHA256) and others.
- Shows: single-shot public-key encryption with an exporter suitable for deriving session keys.
- Relation [High]: SAGE uses base mode plus its own signatures for authentication, not `auth` mode. The RFC vectors cover SAGE's suite exactly and belong in the `hpke` conformance layer.

**3.5** Alwen, J., Blanchet, B., Hauck, E., Kiltz, E., Lipp, B., Riepel, D. "Analysing the HPKE Standard." EUROCRYPT 2021; IACR ePrint 2020/1499. https://eprint.iacr.org/2020/1499
- Summary: Computational proofs (CryptoVerif) for HPKE's authenticated KEM under Gap-DH with a random-oracle KDF, plus composition theorems from AKEM and DEM to the full scheme; results hold for the NIST curves, X25519 and X448.
- Shows: HPKE as a primitive is proven; HPKE inside a larger handshake is not.
- Relation [High]: supports the primitive choice. It does not cover SAGE's composition (HPKE base + ephemeral X25519 + Ed25519 signature + ack tag), so a SAGE paper cannot inherit these guarantees without its own analysis.

**3.6** Perrin, T. "The Noise Protocol Framework." Revision 34, 11 July 2018. https://noiseprotocol.org/noise.html
- Summary: Handshake patterns (NK, XK, IK, ...) built from DH tokens, each with stated authentication and identity-hiding levels; one-round-trip patterns where the responder's static key is known in advance.
- Shows: the design space SAGE's handshake occupies (responder static known via DID, ephemeral-ephemeral for forward secrecy, initiator authentication by signature rather than static DH).
- Relation [Mid]: SAGE's handshake resembles NK plus signatures. Framing it as a Noise-like pattern lets the project reuse the Noise security levels vocabulary and the two tools below.

**3.7** Kobeissi, N., Nicolas, G., Bhargavan, K. "Noise Explorer: Fully Automated Modeling and Verification for Arbitrary Noise Protocols." IEEE EuroS&P 2019; IACR ePrint 2018/766. https://eprint.iacr.org/2018/766
- Summary: Generates ProVerif models for any Noise pattern; verified 57 patterns and produced per-message security reports.
- Shows: automated symbolic verification of one-round-trip DH handshakes is routine.
- Relation [High]: suggests the evaluation method: model the SAGE handshake in ProVerif and report per-message confidentiality/authentication levels.

**3.8** Girol, G., Hirschi, L., Sasse, R., Jackson, D., Cremers, C., Basin, D. "A Spectral Analysis of Noise: A Comprehensive, Automated, Formal Analysis of Diffie-Hellman Protocols." USENIX Security 2020; IACR ePrint 2024/1226. https://eprint.iacr.org/2024/1226
- Summary: Vacarme computes, with Tamarin, the strongest threat model under which each Noise pattern is secure; finds differences between patterns thought equivalent and missing assumptions in the Noise specification.
- Shows: informal security levels can be wrong; a threat-model lattice is the precise way to state what a handshake achieves.
- Relation [High]: same evaluation method for SAGE, with a stronger result form (strongest adversary tolerated: compromised ephemeral, compromised static, KCI).

### Replay windows and robust channels

**3.9** Kent, S. "IP Encapsulating Security Payload (ESP)." RFC 4303, IETF Proposed Standard, December 2005. https://datatracker.ietf.org/doc/rfc4303/
- Summary: Anti-replay by monotonically increasing sequence numbers and a receiver sliding window (minimum 32, default 64) whose right edge is the highest validated number; extended sequence numbers for 64-bit counters.
- Shows: the reference design for counter-based replay protection.
- Relation [High]: `sage-spec` 05-session names a "seq header, replay window"; this RFC gives the exact semantics to specify (window size, left-edge rejection, validate-before-advance).

**3.10** Zhang, X., Tsou, T. "IPsec Anti-Replay Algorithm without Bit Shifting." RFC 6479, Independent Submission, January 2012. https://datatracker.ietf.org/doc/rfc6479/
- Summary: Window as M circular blocks of N bits so that sliding updates an index rather than shifting bits.
- Shows: large windows are cheap.
- Relation [Mid]: implementation guidance for SAGE's window and for replacing the unbounded nonce store noted in STRATEGY.md section 4 (verifier DoS).

**3.11** Thomson, M., Turner, S. "Using TLS to Secure QUIC." RFC 9001, IETF Proposed Standard, May 2021. https://datatracker.ietf.org/doc/rfc9001/
- Summary: AEAD nonce = IV XOR packet number; per-algorithm confidentiality and integrity limits (ChaCha20-Poly1305: 2^36 forgery attempts before key change).
- Shows: deriving the nonce from the counter removes nonce storage and makes replay detection equal to duplicate-packet-number detection.
- Relation [High]: supports STRATEGY.md section 3 "session AEAD nonce derived from a monotonic counter with direction in AAD"; the integrity limits give SAGE a concrete key-rotation trigger to specify.

**3.12** Rescorla, E., Tschofenig, H., Modadugu, N. "The Datagram Transport Layer Security (DTLS) Protocol Version 1.3." RFC 9147, IETF Proposed Standard, April 2022. https://datatracker.ietf.org/doc/rfc9147/
- Summary: Sliding receive window per epoch with a bit-mask check referencing IPsec; no fixed window size; 2^36 failed-authentication limit for AES-GCM and ChaCha20-Poly1305.
- Shows: the same window design carried into a modern AEAD record layer.
- Relation [Mid]: SAGE's WebSocket transport is reliable and ordered, so a window is only needed if SAGE allows reordering; the spec should say which.

**3.13** Fischlin, M., Günther, F., Janson, C. "Robust Channels: Handling Unreliable Networks in the Record Layers of QUIC and DTLS 1.3." IACR ePrint 2020/718, 2020. https://eprint.iacr.org/2020/718
- Summary: Defines channel robustness for sliding-window record layers; shows robustness + integrity + IND-CPA gives a robust analogue of IND-CCA; the analysis produced the concrete forgery limits adopted by the IETF.
- Shows: the formal security notion for a replay-windowed AEAD channel.
- Relation [High]: this is the definition a SAGE paper should prove or cite for its session layer, rather than asserting "replay protection".

### Commit-reveal against front-running

**3.14** Daian, P., Goldfeder, S., Kell, T., Li, Y., Zhao, X., Bentov, I., Breidenbach, L., Juels, A. "Flash Boys 2.0: Frontrunning, Transaction Reordering, and Consensus Instability in Decentralized Exchanges." arXiv:1904.05234, 2019 (IEEE S&P 2020). https://arxiv.org/abs/1904.05234
- Summary: Measures arbitrage bots front-running users on Ethereum, introduces miner extractable value and priority gas auctions.
- Shows: any valuable first-come registration on a public chain will be front-run.
- Relation [High]: the threat that motivates SAGE's commit-reveal registration of names and DIDs.

**3.15** Canidio, A., Danos, V. "Commit-Reveal Schemes Against Front-Running Attacks." Tokenomics 2022, OASIcs vol. 110, 2023. DOI 10.4230/OASIcs.Tokenomics.2022.7. https://drops.dagstuhl.de/entities/document/10.4230/OASIcs.Tokenomics.2022.7
- Summary: Game-theoretic model showing a commit-reveal protocol removes exploitative front-running while keeping benign competition, at the cost of two-stage messaging and delay.
- Shows: commit-reveal is sound against front-running in the modelled setting; its cost is latency.
- Relation [High]: supports the mechanism and names the trade-off SAGE pays (1-60 min commit window, 1 h activation delay). A paper should measure the delay's effect on onboarding.

**3.16** Zhang, H., Merino, L.-H., Estrada-Galiñanes, V., Ford, B. "Flash Freezing Flash Boys: Countering Blockchain Front-Running." IEEE ICDCS Workshops (DINPS), 2022. https://bford.info/pub/sec/f3b/
- Summary: Encrypts transaction content, revealed by a secret-management committee after commitment; 0.1-2.2 s added latency depending on committee size.
- Shows: an alternative with seconds of latency instead of minutes, at the cost of a committee.
- Relation [Mid]: alternative design point for SAGE's registration; useful comparison in the trade-off table.

---

## 4. Blockchain-anchored PKI, trust registries and reputation

**4.1** Kubilay, M. Y., Kiraz, M. S., Mantar, H. A. "CertLedger: A New PKI Model with Certificate Transparency Based on Blockchain." IACR ePrint 2018/1071, 2018. https://eprint.iacr.org/2018/1071
- Summary: Validation, storage and revocation of TLS certificates on a blockchain; domain owners prove certificate existence directly; compared with AKI, ARPKI, DTKI.
- Shows: on-chain PKI removes OCSP dependence and split-world attacks.
- Relation [Mid]: same argument SAGE makes for on-chain agent keys (revocation visible to every resolver). CertLedger's comparison table is a template.

**4.2** Wang, Z., Lin, J., Cai, Q., Wang, Q., Zha, D., Jing, J. "Blockchain-Based Certificate Transparency and Revocation Transparency." IEEE TDSC 19(1):681-697, 2022 (workshop version FC 2018 BITCOIN). https://ieeexplore.ieee.org/document/9050433/
- Summary: Certificates and revocation status published as blockchain transactions; web servers get cooperative control over their certificates.
- Shows: costs are "reasonable" in storage, validation delay and incentives; the open problem is scalable revocation for lightweight clients.
- Relation [Mid]: SAGE's resolver cache plus on-chain revocation is the lightweight client case this work leaves open; revocation-to-rejection latency is the metric.

**4.3** Fdhila, W., Stifter, N., Kostal, K., Saglam, C., Sabadello, M. "Methods for Decentralized Identities: Evaluation and Insights." BPM 2021 Blockchain Forum; IACR ePrint 2021/1087. https://eprint.iacr.org/2021/1087
- Summary: Evaluates DID methods against W3C-derived criteria (decentralisation, self-sovereignty, operations).
- Shows: an evaluation rubric for DID methods.
- Relation [Mid]: `did:sage` should be scored on the same rubric before claiming advantages over `did:ethr` or `did:web`.

**4.4** Satybaldy, A., Tylinski, K., Xu, J. "Decentralized Identity in Practice: Benchmarking Latency, Cost, and Privacy." arXiv:2601.20716, 2026. https://arxiv.org/abs/2601.20716
- Summary: Benchmarks `did:ethr`, `did:hedera`, `did:xrpl` with reference SDKs: Ethereum has off-chain creation but the highest latency and cost for on-chain lifecycle operations; XRPL fixed low fees but more metadata leakage; Hedera lowest latency.
- Shows: a concrete benchmark protocol (latency, fee, on-chain metadata exposure) for ledger DID methods.
- Relation [High]: directly reusable for `did:sage` on Ethereum, Kaia and Solana; the paper's Ethereum numbers are the baseline SAGE's three-phase registration will exceed.

**4.5** Zhu, L., Li, Y., Wang, T., et al. "Blockchain Empowered Trustworthy Agent Networks: Foundations, Taxonomy, and Future Directions." arXiv:2608.04626, 2026. https://arxiv.org/abs/2608.04626
- Summary: Five-dimensional trust taxonomy (entity/capability, authorization/ delegation, information/provenance, coordination, accountability/settlement); blockchain as a shared trust layer that complements, not replaces, existing security.
- Shows: the field's framing of what an on-chain layer contributes to agent networks.
- Relation [High]: SAGE covers entity trust and provenance; the taxonomy exposes authorization/delegation and settlement as out of scope, which a paper should say.

**4.6** Chishti, M. S., Oyinloye, D. P., Li, J. "AgentReputation: A Decentralized Agentic AI Reputation Framework." arXiv:2605.00073, 2026. https://arxiv.org/abs/2605.00073
- Summary: Three-layer design (execution, reputation service, tamper-proof storage), context-conditioned reputation cards, explicit verification regimes, and a policy engine for risk-based verification escalation.
- Shows: reputation must be context-specific and tied to a verification regime to be meaningful.
- Relation [Mid]: argues against the single scalar score of ERC-8004 that SAGE deploys (see 2.4). Candidate direction if SAGE keeps a reputation registry.

**4.7** Alqithami, S. "Autonomous Agents on Blockchains: Standards, Execution Models, and Trust Boundaries." arXiv:2601.04583, 2026. https://arxiv.org/abs/2601.04583
- Summary: Survey of agents that read and write chain state; threat model covering prompt injection, key compromise and collusion; proposes a Transaction Intent Schema and Policy Decision Record with benchmarks.
- Shows: key compromise of an agent's signing key is the dominant on-chain risk.
- Relation [Mid]: reinforces that SAGE's rotation/revocation path must be enforced by the resolver (STRATEGY section 4 notes it currently is not).

---

## 5. Evaluation methodologies in this literature

**5.1** Cremers, C., Horvat, M., Hoyland, J., Scott, S., van der Merwe, T. "A Comprehensive Symbolic Analysis of TLS 1.3." ACM CCS 2017, pp. 1773-1788. DOI 10.1145/3133956.3134063. https://dl.acm.org/doi/10.1145/3133956.3134063
- Summary: Modular Tamarin model of TLS 1.3 draft 21 across all handshake modes; verified the claimed properties and influenced the final standard.
- Shows: a symbolic model of a full handshake with unbounded sessions is tractable and changes standards.
- Relation [High]: model for a SAGE handshake analysis; the size of SAGE's handshake (two or four messages) makes this cheaper than TLS.

**5.2** Bhargavan, K., Blanchet, B., Kobeissi, N. "Verified Models and Reference Implementations for the TLS 1.3 Standard Candidate." IEEE S&P 2017, pp. 483-502. DOI 10.1109/SP.2017.26. https://www.semanticscholar.org/paper/338d4815de02be38990db8cff9f96ef8e6959c80
- Summary: Symbolic (ProVerif) and computational (CryptoVerif) models developed hand in hand with a reference implementation of TLS 1.3 draft 18.
- Shows: model and implementation can be kept aligned; the reference implementation is the artefact that makes the proof apply to real bytes.
- Relation [High]: matches `sage-spec`'s "Go core is normative, vectors bind text to code" stance; the extra step is deriving the model from the same source as the vectors.

**5.3** Celi, S., Hoyland, J., Stebila, D., Wiggers, T. "A Tale of Two Models: Formal Verification of KEMTLS via Tamarin." ESORICS 2022; IACR ePrint 2022/1111. https://eprint.iacr.org/2022/1111
- Summary: Two Tamarin models of KEMTLS: one derived from the TLS 1.3 model (fidelity), one from the multi-stage key exchange proof (property granularity); found flaws in pen-and-paper claims.
- Shows: modelling choices decide which properties you can state; two models are sometimes needed.
- Relation [High]: KEMTLS authenticates by KEM rather than signatures, the same family as HPKE-based authentication; its models are a starting point for SAGE.

**5.4** Basin, D., Foster, N., McMillan, K. L., Namjoshi, K. S., Nita-Rotaru, C., Smith, J. M., Zave, P., Zuck, L. D. "It Takes a Village: Bridging the Gaps between Current and Formal Specifications for Protocols." arXiv:2509.13208, 2025. https://arxiv.org/abs/2509.13208
- Summary: Position paper on why RFC-style prose specifications diverge from formal ones and how to reconcile them, with recent success cases.
- Shows: the argument for spec-first, vector-backed protocol work.
- Relation [Mid]: cites the same problem STRATEGY.md section 3 identifies (33 documented contradictions between docs and code); useful framing for the spec repository.

**5.5** C2SP. "Project Wycheproof: test vectors for cryptographic libraries." Community repository (originally Google), ongoing. https://github.com/C2SP/wycheproof
- Summary: JSON test vectors with schemas that target known attacks, specification inconsistencies and implementation bugs across common primitives (ECDSA, EdDSA, X25519, AEADs).
- Shows: negative and edge-case vectors, not only positive ones, are what catch cross-implementation bugs.
- Relation [High]: SAGE's `crypto` layer (Ed25519, secp256k1 low-S/RFC 6979, X25519, ChaCha20-Poly1305) should run Wycheproof vectors in both cores; the `verify-only` vectors in `sage-spec` follow the same idea.

**5.6** Zhou, Z. "Governing Dynamic Capabilities: Cryptographic Binding and Reproducibility Verification for AI Agent Tool Use." arXiv:2603.14332, 2026. https://arxiv.org/abs/2603.14332
- Summary: Binds tool descriptions cryptographically (Ed25519/SHA-256; BBS+ with Groth16 for selective disclosure) and verifies that agents executed the claimed actions; formal proofs plus experiments on 9 models and 5-20 agent pipelines, reporting detection of all attack scenarios with zero false positives.
- Shows: an evaluation design that mixes formal argument with an empirical multi-agent attack suite and reports detection/false-positive rates.
- Relation [High]: the closest empirical template for evaluating a SAGE gateway that signs MCP tool descriptions and results (STRATEGY section 4).

Also relevant as method: A2ABreak (1.6) for specification-level analysis, Xiong et al. (2.4) for on-chain ecosystem measurement, Satybaldy et al. (4.4) for DID benchmarks, and the RFC 9421 / RFC 9180 vectors (3.1, 3.4) for conformance.

---

## 6. Gap analysis: which SAGE claims have research support

| SAGE claim (source) | Support in literature | Status | Confidence |
|---|---|---|---|
| Identity must be bound to every agent message; A2A alone does not do it (STRATEGY s4) | 1.3, 1.5, 1.6 | Supported | [High] |
| Signing tool descriptions/results with the server DID mitigates tampering (STRATEGY s4) | 1.2, 1.8, 5.6 | Supported for in-transit alteration; not for malicious-at-source | [High] |
| SAGE protects against prompt injection across agents | 1.7, 1.8 | Not supported; provenance only | [High] |
| HPKE is a sound primitive for the handshake (README) | 3.4, 3.5 | Supported for the primitive | [High] |
| The SAGE handshake (HPKE base + ephC + Ed25519 + ackTag) is secure as a composition | none found | Unsupported; no analysis exists | [High] |
| DID-resolved raw keys are safely bound to identity | 3.3 | At risk unless DIDs and key hash are in the signed context; unverified | [Mid] |
| Replay protection by nonce cache / seq window (README, spec 05) | 3.9-3.13 | Design pattern supported; SAGE's instance unverified and, per STRATEGY s1, not on the live path | [High] |
| Counter-derived AEAD nonce with direction in AAD (STRATEGY s3) | 3.11, 3.13 | Supported | [High] |
| Commit-reveal defeats registration front-running (README) | 3.14, 3.15 | Supported in model; latency cost unmeasured for SAGE | [High] |
| On-chain registry gives revocation visible to all resolvers | 4.1, 4.2 | Supported in principle; revocation latency and light-client cost unmeasured | [Mid] |
| Ethereum registry cost/latency is acceptable | 4.4 | Contradicted for lifecycle ops (highest cost/latency of benchmarked ledgers); SAGE adds three transactions and delays | [Mid] |
| ERC-8004 reputation is a usable trust signal | 2.3, 2.4, 4.6 | Contradicted as deployed | [High] |
| Byte-exact vectors give cross-language determinism (spec 00) | 3.1, 3.2, 5.2, 5.5 | Supported as method; JCS and ECDSA determinism are the known hazard points | [High] |
| Delegation / who the agent acts for | 2.8, 1.6 | Gap: SAGE has no delegation chain | [High] |
| `did:sage` conforms to DID Core | 2.1, 4.3 | Gap: no DID method specification exists | [High] |

---

## 7. Candidate research questions and the evaluation each needs

1. Is the SAGE handshake secure against the strongest Noise-style adversary (ephemeral and static compromise, KCI, identity misbinding)? Evaluation: Tamarin or ProVerif model derived from `sage-spec` 04/06 (methods 3.7, 3.8, 5.1, 5.3); report the threat-model lattice; include the misbinding property from 3.3. Pass: proofs for the properties the README claims (mutual authentication, forward secrecy, key confirmation).

2. Does the session layer satisfy channel robustness (3.13) with the specified window and counter-derived nonces? Evaluation: proof sketch against the Fischlin-Günther-Janson definitions plus a differential test harness (replay, reorder, forgery-count) run on both cores with shared vectors. Pass: identical accept/reject decisions across cores.

3. Which A2A and MCP threats are closed by SAGE, measured rather than argued? Evaluation: re-run A2ABreak's 11 findings (1.6) and the MAESTRO list (1.3) against a SAGE-wrapped A2A deployment; run MCPTox (1.8) through a SAGE gateway, separating altered-in-transit from malicious-at-source. Report per-threat closed/open with detection and false-positive rates as in 5.6.

4. What do on-chain registration and revocation cost, and how fast does revocation reach verifiers? Evaluation: the Satybaldy et al. protocol (4.4) for `did:sage` on Ethereum, Kaia and Solana: gas, fee, wall-clock latency per phase, metadata exposure; plus revocation-to-first-rejection latency under resolver caching.

5. Does staked commit-reveal reduce placeholder and Sybil registrations relative to ERC-8004's open registry? Evaluation: replicate the Xiong et al. measurement (2.4) on SAGE's Sepolia registries (endpoint liveness, reviewer clustering) and compare rates.

6. Do byte-exact vectors keep two cores and thin SDKs interoperable over time? Evaluation: vector suite coverage (positive, negative, Wycheproof-style edge cases for JCS numbers, ECDSA low-S, HPKE info strings), number of divergences caught per release, and a cross-implementation matrix.

7. Can delegation be added to the signed envelope without breaking one-round-trip setup? Evaluation: extend the envelope with a South et al. (2.8) style credential; re-run the multi-hop identity-loss case from 1.6; measure added bytes and verification time.

---

## 8. Facts vs Opinions

**Fact**
- All 45 works cited above were located by web search on 2026-09-12 and their title, authors, year and URL confirmed by fetching the abstract, datatracker, W3C or EIP page. Two fetches returned HTTP 403 (usenix.org, dl.acm.org); those two works were confirmed through the IACR ePrint mirror and dblp/publisher search results.
- No published formal analysis of the SAGE handshake or session layer was found.
- Xiong et al. (2.4) report that 3-15% of ERC-8004 registrations keep a valid endpoint and that 59-91% of reviewers show coordinated fraudulent activity.
- Satybaldy et al. (4.4) report Ethereum has the highest latency and cost for on-chain DID lifecycle operations among Ethereum, Hedera and XRPL.
- RFC 8785 (JCS) is an Independent Submission, not an IETF standard; RFC 9180 is an IRTF Informational document; RFC 9421 is an IETF Proposed Standard.
- STRATEGY.md section 1 records that SAGE's replay and body-integrity controls are implemented but not enforced on the live request path.

**Opinion**
- [High] The single most valuable evaluation for a SAGE paper is a symbolic model of the handshake including the DID/key-hash binding, because the composition is novel and misbinding (3.3) is the plausible failure mode.
- [High] SAGE should not claim prompt-injection protection; it should claim cryptographic provenance and state that limit, citing 1.7.
- [High] ERC8004ReputationRegistry should not be presented as a trust signal until a Sybil-resistant filter is specified and measured (2.4, 4.6).
- [Mid] Framing the handshake as a Noise-like pattern (NK plus signatures) will make the security claims easier for reviewers to place and easier to verify with existing tooling.
- [Mid] The three-phase registration's minute-to-hour delays are a stronger onboarding cost than any benchmarked DID method; the paper needs to justify the window sizes or offer an F3B-style faster alternative.
- [Low] Adding a VC-based delegation credential to the envelope is feasible in one round trip; this is untested.
