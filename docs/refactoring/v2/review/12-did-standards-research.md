# 12. Decentralised identifier standards: what SAGE should build on

Research note for the v2 refactoring review. Written 2026-09-13.

The question this note answers: agents will be numerous, each needs an identifier
that never collides with another even when agents are registered in different
blockchains or in registries that are not blockchains, and the holder of an
identifier must be provably its owner. SAGE wants to sit on the decentralised
identity standards rather than invent a scheme.

## 0. Sources and how they are marked

Every claim below says which class of source it came from.

**[study]** — the local standards study at
`/Users/kevin/work/github/0xmhha/study/projects/go-did/`, 3,187 lines written by
this project's maintainer. It pins the specification commits its line citations
refer to (`README.md`, provenance table) and archives the normative HTML with
SHA-256 digests under `standards/w3c/`. Every normative claim below about DID Core,
Controlled Identifiers or DID Resolution cites the study, not a web summary; two
claims were re-verified directly against the archived HTML.

**[web]** — fetched 2026-09-13, only for what the study does not cover: how other
methods guarantee uniqueness across registries, chain-agnostic identifiers, and
proposals aimed at autonomous agents.

The SAGE side is `sage-spec/spec/06-did-sage.md` (grammar, chains, resolution, key
proof of possession) and `spec/01-crypto.md`. `sage-spec/charter.md` does not exist;
`README.md` and `spec/00-overview.md` were read in its place. Archived normative
texts [study, `standards/README.md`]:

| Document | Status | Date | URL |
|---|---|---|---|
| Decentralized Identifiers (DIDs) v1.0 | W3C Recommendation | 2022-07-19 | https://www.w3.org/TR/2022/REC-did-core-20220719/ |
| Decentralized Identifiers (DIDs) v1.1 | W3C Candidate Recommendation Snapshot | 2026-03-05 | https://www.w3.org/TR/2026/CR-did-1.1-20260305/ |
| Controlled Identifiers (CIDs) v1.0 | W3C Recommendation | 2025-05-15 | https://www.w3.org/TR/2025/REC-cid-1.0-20250515/ |
| DID Resolution v1.0 | W3C Candidate Recommendation Draft | 2026-08-28 | https://www.w3.org/TR/2026/CRD-did-resolution-1.0-20260828/ |

---

## 1. The headline problem before anything else

**`did:sage` as specified cannot guarantee non-collision, and the collision is not
theoretical.** `06-did-sage.md` §2 states that networks within a chain
(`ethereum-mainnet`, `sepolia`, `goerli`, `solana-mainnet`, `solana-devnet`,
`solana-testnet`) "are configuration, not part of the DID". An agent registered at
address `0xabc…` in the Sepolia registry and a different agent at the same address
in the mainnet registry therefore bear the identical DID string
`did:sage:ethereum:0xabc…`. Two registries issue the same identifier and a
resolver cannot tell them apart from the identifier alone. [High] — direct from §2
with no intermediate inference.

The same defect makes the key proof of possession replayable across networks. The
challenge is `"SAGE-PoP:" || DID || ":" || hex(key_data)` (§4). Because the DID
does not name the network, a proof produced for a testnet registration is a
byte-identical valid proof for a mainnet registration of the same address and key,
and testnets are public and cheap to write to. [High] on the byte-identity; [Mid]
on exploitability, which turns on whether the registration path independently
binds `msg.sender` to the owner — the contracts were not read for this note.

A smaller second source: `ParseChain` accepts `eth`/`ethereum` and `sol`/`solana`
as aliases (§2), so two DID strings denote one agent. DID Resolution requires the
resolved document's `id` to string-match the DID that was resolved [study, `07`
§1], so an alias breaks the resolution contract unless one form is canonical. The
remedy exists and SAGE is not using it: `didDocumentMetadata.canonicalId` and
`equivalentId` [study, `07` §5]. [High].

The fix for the first defect is a solved problem — CAIP-2 chain identifiers, which
`did:pkh` already uses — and is the subject of §4.2.

---

## 2. What a specification must contain to call itself a DID method

Taken from the study, which derives the list from DID Core v1.1 §7 and from the
two specifications v1.1 delegates to.

DID Core §conformance defines five conforming entities, not four: a conforming
DID, a conforming DID document, a conforming producer, a conforming consumer, and
a conforming DID method [study, `00` §1 fact table and note m3]. A method
specification is judged as the fifth; an implementation must also satisfy the
other four to interoperate.

### 2.1 The mandatory contents of a method specification

| Obligation | What the specification must contain | Study reference |
|---|---|---|
| Method syntax | An ABNF for the method-specific identifier no looser than `idchar = ALPHA / DIGIT / "." / "-" / "_" / pct-encoded`, with colon-separated layering permitted but no trailing colon and no empty msid | `10` §1.1, §1.3 |
| Create | How a controller produces the DID and its initial document, and the authorisation for doing so | `00` FR-M1 |
| Read / Resolve | How a DID becomes a document, **including how the reader verifies the authenticity of the response** | `00` FR-M2 |
| Update | How an update is performed, **or an explicit statement that update is impossible** | `00` FR-M3 |
| Deactivate | How deactivation is performed, **or an explicit statement that it is impossible** | `00` FR-M4 |
| Authorisation | The authorisation and cryptographic procedure for every operation, naming which of `controller` / `authentication` / `capabilityInvocation` / out-of-band is used | `00` FR-M5 |
| Security considerations | Integrity protection and update authentication for every operation, plus documented handling of eavesdropping, replay, insertion, deletion, modification, denial of service, amplification and man-in-the-middle | `00` FR-P1, FR-P2 |
| Privacy considerations | Discussion of the applicable RFC 6973 §5 subsections is normatively required; correlation resistance and selective disclosure are informative | `00` FR-P4, note m2 |

Two points the study is careful about and online summaries get wrong. First,
"explicit statement that it is impossible" is a conforming answer for update and
deactivate — `did:key` and `did:pkh` take it [web, §4]. A method is not required to
support mutation; it is required to say so. Second, the method-specific identifier
is unique **within the method**; global uniqueness is a property of the whole DID,
and it is the method's job to make it so [study, `08` §1, correction m1]. A method
that admits two registries issuing the same method-specific identifier has failed
this, whatever the msid's internal entropy.

### 2.2 Registering the method name

The name is reserved by pull request to `w3c/did-extensions`, one JSON file per
method under `methods/`. The criteria: a human-readable description, a name that
avoids generic terms, an affirmation of no third-party IP infringement, no
unreasonable legal/security/moral/privacy issues, a link to a persistent
specification, and a JSON-LD context with versioned protected terms. "Any
submission to the registries that meet all the criteria listed above will be
accepted" — criteria-based, neither discretionary nor a rubber stamp.
[web: *Decentralized Identifier Extensions*, W3C,
https://www.w3.org/TR/did-extensions/, fetched 2026-09-13.]

**`sage` is not registered.** Direct check on 2026-09-13: `methods/sage.json`
returns 404 while `agent`, `aip`, `ans`, `opena2a`, `wba`, `trail`, `indy`, `pkh`,
`ethr`, `key`, `web`, `peer`, `ion` and `webvh` all return 200; the directory
listing shows roughly 260 names, with `safe` and `salt` present and no `sage`
between them [web, https://raw.githubusercontent.com/w3c/did-extensions/main/methods/].
Nothing currently stops another party from taking the name. [High]

The entries also show what a name commits you to: `did:pkh` records its registry as
"Ledger-independent generative DID method based on CAIP-10 keypair expressions",
`did:indy` as "Any Hyperledger Indy Ledger". The `verifiableDataRegistry` field is
where a multi-registry method declares itself.

### 2.3 Where `did:sage` stands against the list

| Obligation | `did:sage` today | Severity |
|---|---|---|
| Method syntax | Grammar exists (`06` §1) but is **looser than the DID ABNF**: `identifier = 1*( unreserved / ":" )` admits `~`, which is in RFC 3986 `unreserved` but not in DID `idchar`; and it has no `pct-encoded` production | [Important] |
| Create | Described operationally (3-phase commit-reveal, project CLAUDE.md) but not in the method spec | [Important] |
| Read / Resolve | Returns `AgentMetadataV4`, not a DID document (`06` §3, open item O-8). Response authenticity is not specified | [Critical] |
| Update | Not stated, and not stated to be impossible | [Important] |
| Deactivate | `is_active` boolean exists; the deactivation procedure and its authorisation are not specified | [Important] |
| Authorisation | Not stated per operation. The registry record has an `owner` field, but which verification relationship governs an update is undefined | [Important] |
| Security considerations | Absent from `06`. The cross-convention hash in the PoP is flagged as open item O-1 but not analysed as a threat | [Important] |
| Privacy considerations | Absent from `06`. The study's own risk table rates a fixed public per-subject DID as high correlation risk [study, `11` §5] | [Important] |
| Registered name | Not registered | [Recommended] |

[High] on every row; each is a direct comparison of two documents in front of me.

---

## 3. How the data model and the resolution contract constrain a non-blockchain registry

This is the part SAGE needs most, because the stated goal includes registries that
are not blockchains. The good news from the study is that **DID Core requires no
blockchain at all** [study, `11` §3.1]. The constraints that do bite are about the
shape of the answer, not about where it is stored.

### 3.1 What the document must look like, whatever the registry

DID Core v1.1 delegates verification methods, services and verification
relationships to Controlled Identifiers 1.0 [study, `00` §1 fact B; `06` header].
A registry-agnostic profile must therefore produce, for every backend:

- `id` present at the top level and equal to the URL the document was retrieved
  from; a mismatch SHOULD render the document invalid [study, `06` §1, citing CID
  §2.1.1 lines 785 and 808–812].
- Each verification method carrying `id`, `type` (`Multikey` or `JsonWebKey`),
  `controller`, and exactly one piece of key material — `publicKeyJwk` and
  `publicKeyMultibase` together are prohibited [study, `06` §2, §2.1, CID line 1306].
- Services carrying `type` and `serviceEndpoint`; `id` is optional in v1.1 and
  required in v1.0, and duplicate service `id` values are forbidden [study, `06`
  §8; `09` §2.2].
- Consumers producing an error on a non-conforming document — a hard MUST, not a
  SHOULD [study, `06` §1, CID line 461].

The binding between a key and its controller is not asserted, it is computed. The
CID "Retrieve Verification Method" algorithm is a twelve-step bidirectional check
[study, `06` §3]: the document must reach the method through a verification
relationship, the method's `controller` must point back at the document's URL, and
the method's absolute `id` must equal the identifier that was dereferenced. A
`controller` field on its own is a mere claim that may be false (CID line 1245).
This is the standards' answer to "the owner must be verifiable", and it is
structural rather than cryptographic: the cryptography proves possession of a key,
the algorithm proves that key is the one the document authorises for this purpose.

### 3.2 What the resolver must do, whatever the registry

`resolve(did, resolutionOptions) → « didResolutionMetadata, didDocument,
didDocumentMetadata »`. A conforming resolver does not change this signature
[study, `07` §1]. Three consequences for a non-blockchain profile:

1. **Deactivation is not an error.** A deactivated DID yields `didDocument = null`,
   `didDocumentMetadata.deactivated = true`, no error, HTTP 410 [study, `07` §1.1,
   §4]. Any backend — a database row, a file, a directory entry — must express
   "existed, now deactivated" distinctly from "never existed" (404).
2. **Errors are RFC 9457 Problem Details** with `type` a URL under
   `https://www.w3.org/ns/did#`, from a closed set of nine codes plus two inherited
   from CID, each with a fixed HTTP status [study, `07` §4]. Under v1.0 the same
   errors are bare camelCase strings and only five are defined [study, `09` §2.4];
   a profile wanting both goes through an adapter, not two implementations.
3. **The HTTP binding is fully specified** and registry-neutral:
   `GET /1.0/identifiers/{id}` is MUST, TLS required, `accept` travels as the HTTP
   `Accept` header, and the media types are `application/did`,
   `application/did-resolution`, `application/did-url-dereferencing` [study, `07`
   §6]. **A non-blockchain SAGE registry needs no new invention here — this is the
   profile.** [High]

The study flags that the resolution specification is still a draft with internal
contradictions and names the positions it adopts: `contentType` belongs in
`didResolutionMetadata` (T3), `deactivated` is boolean not string (T4), and service
endpoint composition follows RFC 3986 §5 rather than the specification's own worked
example (T2) [study, `08` §2.2]. A SAGE profile should adopt the same positions so
that it is not diverging twice.

### 3.3 The one place the study says to slow down

The study's release policy makes DID Core **1.0 the operational default** and marks
1.1 explicitly experimental until it reaches Recommendation [study, `11` §1, §6.1].
It also lists the v1.1 JSON-LD context as a release blocker:
`https://www.w3.org/ns/did/v1.1` is not yet published (returns 300 at CR stage), so
the pinned SHA-256 in `assets/did-v1.1.jsonld` must be re-pinned on publication
[study, `10` §3]. A SAGE profile targeting 1.1 only inherits that blocker.

---

## 4. How existing methods guarantee uniqueness, and the chain-agnostic answer

All [web], fetched 2026-09-13; the study puts specific methods out of scope
[study, `00` §4].

| Method | Uniqueness mechanism | Ownership proof | Network to resolve | Update / deactivate | Chain binding |
|---|---|---|---|---|---|
| `did:key` | Identifier **is** the key: `multibase(multicodec(type, raw key))`. Collision = key collision | Sign with the key the identifier encodes | No — purely generative | Neither supported, explicitly | None |
| `did:web` | Borrowed from DNS plus path | None specified; "delegated to implementations" | Yes — HTTPS GET | Edit or remove `did.json` | None |
| `did:pkh` | CAIP-10 account id: chain namespace + chain reference + address | Left to the chain's wallet; not mandated | No — deterministic offline expansion | Neither supported | Chain-agnostic by construction |
| `did:ethr` | secp256k1 address entropy (~2^-160); network segment disambiguates the same address on different chains | ERC-1056 `owner`, proven by signing a transaction or meta-transaction | Yes — JSON-RPC, replaying contract events | `changeOwner` rotates; `owner = 0x0` deactivates irreversibly | EVM-family; requires ERC-1056 deployed |
| `did:ion` (Sidetree) | Suffix = hash of the canonicalised create-operation suffix data; self-certifying | Commit-reveal against update and recovery key commitments, JWS-signed | Yes for published DIDs (anchor + IPFS); long-form DIDs self-resolve | Update, recover and deactivate all defined; deactivate needs the recovery key | Anchoring layer is pluggable (Bitcoin for ION) |
| `did:peer` | Cryptographic entropy or a hash of the genesis document; numalgo 0–4 | Possession of the encoded keys | No — resolved from local storage | No update; rotation is by issuing a new DID | None; pairwise use only |
| `did:webvh` | SCID = `base58btc(multihash(JCS(inception log entry), SHA-256))`, embedded in the DID | Data Integrity proof on every log entry, with optional key pre-rotation commitments | Yes — HTTPS GET of `did.jsonl`, or a watcher | Append a log entry; `"deactivated": true`, or empty `updateKeys` | None; web-hosted with a verifiable history |
| `did:indy` | msid = base58 of the first 16 bytes of SHA-256 of the initial verification key, self-certifying and enforced by the ledger; a namespace segment names the network | NYM ledger transaction signed by the current verkey | Yes — Indy ledger read | verkey rotation by NYM | One method, many Indy networks, disambiguated by namespace |

Sources, all fetched 2026-09-13: *did:key Method v0.9*, W3C CCG,
https://w3c-ccg.github.io/did-key-spec/ · *did:web Method Specification*, W3C CCG,
https://w3c-ccg.github.io/did-method-web/ · *did:pkh Method Specification* (draft),
W3C CCG, https://github.com/w3c-ccg/did-pkh/blob/main/did-pkh-method-draft.md ·
*ETHR DID Method Specification*, DIF,
https://github.com/decentralized-identity/ethr-did-resolver/blob/master/doc/did-method-spec.md ·
*Sidetree v1.0.1*, DIF Ratified Specification, https://identity.foundation/sidetree/spec/ ·
*Peer DID Method Specification v1.0 draft*, DIF, https://identity.foundation/peer-did-method-spec/ ·
*The did:webvh DID Method v1.0 (Editors Draft)*, DIF, https://identity.foundation/didwebvh/next/ ·
*Indy DID Method Specification*, Hyperledger / LF Decentralized Trust,
https://hyperledger.github.io/indy-did-method/ — registry entry read directly, the
specification only through a search summary, so that row is [Mid].

### 4.1 The pattern worth copying

Three strategies appear, and they trade against each other.

**Self-certifying identifiers** (`key`, `pkh`, `peer`, `ion`, `webvh`, `indy`)
derive the identifier from key material or a hash of the genesis state. Uniqueness
needs no coordination and the identifier proves its own provenance. The cost is
that the identifier cannot be chosen, and for `key`, `pkh` and `peer` rotation is
impossible — the identifier dies with the key. `ion`, `webvh` and `indy` buy
rotation back by committing the identifier to the *inception* state and proving an
authenticated chain of updates from it.

**Namespaced registry identifiers** (`ethr`, `pkh`, `indy`) put the registry's own
name inside the identifier, so two registries cannot collide. The cost is a
governance dependency: someone must run the namespace registry, and a namespace
renamed or forked breaks identifiers.

**Borrowed namespaces** (`web`) reuse DNS — cheapest to adopt, weakest ownership
story, since `did:web` specifies no authentication or authorisation mechanism.

SAGE needs the second combined with enough of the first to make ownership provable
without trusting the registry operator. `did:webvh` is the closest existing shape:
a self-certifying SCID plus a location plus an append-only signed log. [Mid] —
SAGE's on-chain anchoring may make the log redundant.

### 4.2 CAIP-2 and CAIP-10

*CAIP-2 Blockchain ID Specification*, ChainAgnostic (CASA), Final, created
2019-12-05, updated 2021-08-25,
https://github.com/ChainAgnostic/CAIPs/blob/main/CAIPs/caip-2.md, and *CAIP-10
Account ID Specification*, CASA, Final, created 2020-03-13, updated 2022-10-23,
https://github.com/ChainAgnostic/CAIPs/blob/main/CAIPs/caip-10.md:

```
chain_id        = namespace ":" reference      ; [-a-z0-9]{3,8} : [-_a-zA-Z0-9]{1,32}, max 41 chars
account_id      = chain_id ":" account_address ; account_address = [-.%a-zA-Z0-9]{1,128}
```

Examples: `eip155:1` (Ethereum mainnet), `eip155:11155111` (Sepolia),
`bip122:000000000019d6689c085ae165831e93`, `cosmos:cosmoshub-3`;
`eip155:1:0xab16a96D359eC26a11e2C2b3d8f8B8942d5Bfcdb`.

`did:pkh` is then simply `"did:pkh:" account_id`. Its own account of why this works
names exactly the property SAGE lacks: "networks (i.e., EVMs) and specific chains
have to be specified separately and explicitly", which is what "prevents address
collisions across different blockchains".

Two cautions, both from the specifications' own text [High]. CAIP-2 defines only
the shape and is silent on who may claim a namespace — registration lives in a
separate CASA `namespaces` repository. And it does not address uniqueness *within*
a chain; that rests on the chain's own address format.

**Applied to SAGE**: `did:sage:eip155:11155111:0xabc…` and
`did:sage:eip155:1:0xabc…` are different identifiers, and the Sepolia proof of
possession no longer verifies against the mainnet challenge; `solana:…` covers the
Solana side. The cost is a breaking change to the grammar and to every stored
identifier — a MAJOR version bump under `sage-spec/spec/00-overview.md` §5.

### 4.3 Per-registry cryptography

The different chains bring different signature conventions: secp256k1 over a
Keccak-256 digest on Ethereum, Ed25519 on Solana (`sage-spec/spec/01-crypto.md`
§2). Two things the standards say about this.

`did:pkh` handles it by giving each CAIP namespace its own verification method type
— `EcdsaSecp256k1RecoveryMethod2020` for EVM chains, `Ed25519VerificationKey` for
Solana, P-256 variants for Tezos [web]. The chain prefix selects the cryptography;
the document need not be uniform across chains.

But CID 1.0 constrains the available key encodings. Verified directly against the
archived normative HTML (`standards/w3c/controlled-identifiers-1.0-20250515.html`,
digest `c5e9bef…4ac9` per `standards/README.md`): `secp256k1` does not occur
anywhere in the document, and `X25519` occurs exactly once, in a `JsonWebKey`
example as `{"kty":"OKP","crv":"X25519"}`. The Multikey table covers Ed25519
`0xed01`, P-256 `0x8024`, P-384 `0x8124`, BLS12-381 G2 `0xeb01`, SM2 `0x8624`
[study, `06` §5, audited verbatim per `08` §1]. **So SAGE's two most important key
types — secp256k1 for Ethereum identity, X25519 for the HPKE KEM — have no
`Multikey` encoding in CID 1.0 and must be expressed as `JsonWebKey`**
(`kty:EC, crv:secp256k1` per RFC 8812; `kty:OKP, crv:X25519`). [High] on the
absence; [Mid] on JsonWebKey being the only conforming route, since a future
revision or a registered extension type could add them.

---

## 5. Identity proposals aimed at autonomous agents

All [web], fetched 2026-09-13. Adoption status matters more than content here,
because most of this is a year old or less.

| Proposal | Body | Status | Relation to DIDs |
|---|---|---|---|
| ERC-8004 "Trustless Agents" | Ethereum EIP process | **Draft** per the canonical `ERCS/erc-8004.md`, created 2025-08-13 | Identity Registry is ERC-721; `agentRegistry` is `{namespace}:{chainId}:{identityRegistry}`, a CAIP-2-shaped value. DIDs and ENS names are *optional* endpoints in the registration file, not mandated |
| A2A Protocol v1.0 | Agentic AI Foundation | Released v1.0 | Agent Card is a JSON metadata document; §8.4 defines Agent Card signing with canonicalisation and verification. Does not reference DIDs or verifiable credentials in the core |
| Agent Identity Protocol (AIP) | IETF, `draft-singla-agent-identity-protocol-03` | **Individual Internet-Draft**, no IETF standing; revision 2026-06-10, expires 2026-12-12 | Defines its own `did:aip` method (SHA-256 of an Ed25519 key). Delegation is a chain of signed "Principal Tokens" in an `aip_chain` array inside an `AIP+JWT` |
| `did:trail` (TRAIL) | TRAIL Protocol | Registered in the W3C method registry | Separate identifier types for organisations, agents and self-signed identities |
| `did:wba`, `did:opena2a`, `did:agent`, `did:ans` | Various | Registered in the W3C method registry | Agent-oriented method names already taken |

Sources: https://eips.ethereum.org/EIPS/eip-8004 and
https://raw.githubusercontent.com/ethereum/ERCs/master/ERCS/erc-8004.md ·
https://a2a-protocol.org/latest/specification/ ·
https://datatracker.ietf.org/doc/draft-singla-agent-identity-protocol/ · registry
entries under https://raw.githubusercontent.com/w3c/did-extensions/main/methods/.

A discrepancy worth recording: secondary sources claim ERC-8004 reached Final
ratification with a mainnet deployment on 2026-01-29 (for example
https://eco.com/support/en/articles/14730445-erc-8004-trustless-agent-identity),
while the canonical EIP source still says `status: Draft` as of 2026-09-13.
**Treat it as Draft.**

On delegation from a human or organisation to an agent there is no adopted
standard. The surveyed position is that the multi-hop case — A delegates to B
delegates to C, with every resource server able to trace the chain back to the
originating human — has no production-ready answer and that the IETF work is early
[web: *AI Identity: Standards, Gaps, and Research Directions for AI Agents*,
Otsuka, Toyoda and Leung, arXiv:2604.23280, 2026-04-25,
https://arxiv.org/abs/2604.23280, naming recursive delegation accountability as one
of five structural gaps].

The standards SAGE already has do more here than the agent-specific drafts. CID 1.0
separates `authentication` ("who is this?") from `capabilityInvocation` and
`capabilityDelegation` ("is this allowed?"), recommending different verification
methods for each [study, `06` §7]. Verifiable Credentials 2.0 is a Recommendation
(W3C, 2025-05-15, https://www.w3.org/TR/vc-data-model-2.0/), does not require DIDs
— "[DIDs] are not necessary for verifiable credentials to be useful" — but pairs
naturally with `assertionMethod`, and its optional `credentialStatus` is the hook
for revoking a delegation [web]. **Expressing organisation-to-agent delegation as a
VC over the agent's DID is available today; `did:aip`'s JWT chain is not.** [Mid]

---

## 6. Requirement against mechanism against SAGE

| Requirement | Mechanism the standards offer | Source | Does `did:sage` meet it |
|---|---|---|---|
| Unique identifier across registries | CAIP-2 chain id inside the identifier (`did:pkh`), or a network namespace segment (`did:ethr`, `did:indy`) | [web] | **No.** Network is configuration, not part of the DID; mainnet and testnet DIDs are the same string (`06` §2) |
| Unique identifier without a registry | Self-certifying identifier: key material (`did:key`) or hash of inception state (`did:ion`, `did:webvh`, `did:indy`) | [web] | **No.** The identifier is an account address plus an optional nonce; nothing binds it to key material |
| Verifiable ownership | CID Retrieve Verification Method, twelve steps, bidirectional document↔method binding; `controller` alone is a claim that may be false | [study, `06` §3] | **Partly.** The PoP signature proves key possession (`06` §4) but no document exists to bind the key to a relationship, and the PoP is not bound to the network |
| Works without a blockchain | DID Core mandates no ledger; DID Resolution's HTTP binding (`GET /1.0/identifiers/{id}`, TLS, three media types) is the registry-neutral profile | [study, `11` §3.1; `07` §6] | **No profile exists.** Resolution is defined only for the on-chain registry record |
| Per-registry cryptography | Per-namespace verification method types (`did:pkh`); CID key material is `Multikey` XOR `JsonWebKey` | [web]; [study, `06` §2.1, §5] | **No.** Key types are integers 0/1/2 with raw bytes (`06` §3); secp256k1 and X25519 have no CID Multikey encoding, so `JsonWebKey` is required |
| Revocation | CID `revoked` (XSD dateTimeStamp) per verification method; a referenced method missing from the latest document counts as revoked | [study, `06` §2.2, §7] | **No.** `is_active` deactivates the whole agent; individual keys have only a `verified` boolean |
| Rotation | ERC-1056 `changeOwner`; Sidetree update/recovery commitments; `did:webvh` pre-rotation hashes; Indy verkey rotation | [web] | **Not specified.** No update operation is defined in `06` |
| Expiry | CID `expires`; proofs are not verified after it | [study, `06` §2.2] | **No** equivalent field |
| Deactivation semantics | `didDocument = null` + `didDocumentMetadata.deactivated = true` + HTTP 410, and **not** an error | [study, `07` §1.1, §4] | **No.** `06` §3 says inactive agents MUST NOT be trusted but returns the record regardless |
| Resolution to a standard document | `resolve()` with a fixed signature; document per CID 1.0; media type `application/did` | [study, `07` §1; `00` fact C] | **No.** Returns `AgentMetadataV4` (`06` §3, open item O-8). Zero occurrences of `DIDDocument` in `pkg/` |
| Standard error reporting | RFC 9457 Problem Details, nine codes plus two from CID, fixed HTTP statuses | [study, `07` §4] | **Not specified** |
| Canonical form for aliases | `didDocumentMetadata.canonicalId` and `equivalentId` | [study, `07` §5] | **No.** `eth`/`ethereum` and `sol`/`solana` aliases produce distinct strings for one agent |
| Method name reserved | Pull request to `w3c/did-extensions` | [web] | **No.** `methods/sage.json` returns 404 |

---

## 7. What this implies for the refactoring

In the order the work should be done, stated as options with their costs rather
than as a single recommendation.

**First, close the collision.** Put the registry into the identifier. Either adopt
CAIP-2 wholesale (`did:sage:eip155:11155111:0xabc…`), which buys the existing
namespace governance and puts SAGE alongside `did:pkh`, at the cost of the
readable `ethereum` label; or keep SAGE's chain vocabulary and add an explicit
network segment (`did:sage:ethereum-sepolia:0xabc…`), a smaller edit that means
maintaining a private namespace registry with no path to CAIP-consuming tools.
Either is a MAJOR version bump. Doing neither leaves the PoP replay open.

**Second, write the document projection (O-8) as the registry-neutral contract, not
an on-chain add-on.** The CID mapping is mechanical: `keys[]` → a
`verificationMethod` array of `JsonWebKey` entries, with `keyAgreement` for the
X25519 KEM key and `authentication`/`assertionMethod` for the signing keys;
`endpoint` → `service`; `owner` → `controller`; `is_active` →
`didDocumentMetadata.deactivated`. Once that exists, a non-blockchain registry is an
implementation of the same `resolve()` contract behind the same HTTP binding, not a
second design.

**Third, replace the single `is_active` flag with per-key state.** CID gives
`expires` and `revoked` per verification method, with semantics: no retroactive
effect, only the latest document version counts, and a method absent from the latest
document is treated as revoked [study, `06` §2.2, §7]. This is what makes rotation
expressible — today SAGE can only deactivate a whole agent.

**Fourth, reserve the name.** A pull request; five agent-oriented names are taken.

The one thing not to do is let the agent-specific drafts set direction. ERC-8004 is
a Draft that does not mandate DIDs; AIP is an individual Internet-Draft with no
standing that invents its own method; A2A v1.0 signs agent cards but does not speak
DID. The stable ground — DID Core 1.0, CID 1.0 and VC 2.0, all Recommendations — is
what the local study already mapped, and it answers more of SAGE's question than
any of them. [Mid]

---

## Facts vs Opinions

**Fact** — verified directly against a document or a fetch.

- `06-did-sage.md` §2 excludes the network from the DID, so mainnet and Sepolia
  registrations at the same address share one DID string; `ParseChain` accepts
  `eth`/`ethereum` and `sol`/`solana` as aliases.
- The PoP challenge (§4) is `"SAGE-PoP:" || DID || ":" || hex(key_data)`, with no
  network component.
- `06-did-sage.md` §3 states resolution returns agent metadata, not a W3C DID
  document (open item O-8). `grep -rn "DIDDocument" pkg/` returns zero non-test
  matches.
- The study archives DID Core 1.0 (REC 2022-07-19), DID Core 1.1 (CR Snapshot
  2026-03-05), CID 1.0 (REC 2025-05-15) and DID Resolution 1.0 (CRD 2026-08-28)
  with SHA-256 digests (`standards/README.md`).
- Controlled Identifiers 1.0 contains zero occurrences of `secp256k1`; `X25519`
  occurs once, in a `JsonWebKey` example. Verified by grep of the archived HTML.
- DID `idchar = ALPHA / DIGIT / "." / "-" / "_" / pct-encoded` (study `10` §1.1);
  `did:sage` `identifier = 1*( unreserved / ":" )` (SAGE `06` §1). RFC 3986
  `unreserved` includes `~`, which is not an `idchar`. Under DID Resolution,
  `deactivated` is not an error: `didDocument = null`, metadata
  `deactivated = true`, HTTP 410 (study `07` §1.1, §4).
- `methods/sage.json` in `w3c/did-extensions` returns 404; `agent`, `aip`, `ans`,
  `opena2a`, `wba`, `trail`, `indy`, `pkh`, `ethr`, `key`, `web`, `peer`, `ion`
  and `webvh` all return 200 (fetched 2026-09-13).
- `ERCS/erc-8004.md` carries `status: Draft`, created 2025-08-13.
  `draft-singla-agent-identity-protocol-03` is an individual Internet-Draft with
  no IETF standing, revised 2026-06-10. VC Data Model 2.0 is a W3C Recommendation
  dated 2025-05-15 and states DIDs are not necessary for verifiable credentials.
- `sage-spec/charter.md` does not exist.

**Opinion** — inference.

- [High] `did:sage` does not satisfy the DID Core requirement of global uniqueness,
  because two registries can issue the same string; and the PoP signature is
  byte-identical across networks, so a testnet proof is a valid mainnet proof.
- [High] `did:sage` fails at least six of the eight mandatory contents of a method
  specification in §2.1 (read authenticity, update, deactivate, authorisation,
  security considerations, privacy considerations).
- [High] SAGE's secp256k1 and X25519 keys cannot be `Multikey` under CID 1.0 and
  must use `JsonWebKey`.
- [High] The DID Resolution HTTP binding is a complete, registry-neutral profile; a
  non-blockchain SAGE registry needs no new resolution design.
- [Mid] `did:webvh` is the closest existing shape to what SAGE needs, though
  on-chain anchoring may make its update log redundant.
- [Mid] CAIP-2 is preferable to a private network vocabulary because the namespace
  governance and tooling already exist; the cost is readable chain labels and a
  MAJOR version bump.
- [Mid] Organisation-to-agent delegation as a VC over the agent's DID is a better
  bet than the agent-specific JWT chains, VC 2.0 being a Recommendation.
- [Mid] The `did:indy` row in §4 is less reliable than the others; only its registry
  entry was fetched directly.
- [Mid] Exploitability of the cross-network PoP replay depends on whether the
  registry contract independently binds the transaction sender to the owner; the
  contracts were not read for this note.
- [Low] Registering the method name is urgent. Five agent-oriented names are taken,
  but no evidence was found of anyone contesting `sage` specifically.
