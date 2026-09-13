# 13. Revocation research: what "immediate" can mean, and how other systems get there

Date: 2026-09-13. Scope: external research supporting the charter decision that revocation of a key, and deactivation of an agent, must take effect immediately. Method: every work below was located by web search and its numbers confirmed by fetching the primary page on 2026-09-13; nothing is cited from memory. Confidence marks ([High]/[Mid]/[Low]) apply to the judgement they precede, not to the cited work. Items the research could not confirm are marked "unverified" rather than dropped.

---

## 1. The requirement this report serves

The charter was renumbered on 2026-09-13 while this report was being written. The revocation requirement asked about as R-8 is now **R-9**; R-8 is now the key-selection rule. As the charter currently reads (`sage/.sage-spec/charter.md` §6):

> R-9 — Revocation takes effect immediately. A verifier must not accept a message authenticated by a key that is revoked, or by any key of a deactivated agent, at the moment it verifies; caching must not extend the life of a revoked key. The text states the point at which a registry's state counts as observed. Tested by: vector for the verdict after revocation; measurement of the delay between a revocation becoming observable and the first rejection.

Charter §9 records this as decided, "whatever its cache holds", and leaves the design stage to define *the point at which a registry's state counts as observed*, "because a registry that confirms in blocks cannot be read instantaneously". That clause is the whole subject of this report.

The decision as worded is achievable, but only because it contains that escape hatch. A verifier can be forbidden to add a grace period of its own; it cannot be given a registry that answers instantaneously. R-9 measures "the delay between a revocation becoming **observable** and the first rejection" — a verifier property, which can be driven to zero — not the delay between submission and rejection, which is a chain property and cannot. Sections 2 and 6 quantify the second number.

Two facts about the present code bear on the decision. `spec/06-did-sage.md` §3 already says a record with `is_active` false "MUST NOT be trusted" but says nothing about how fresh the reader's copy must be. And `pkg/agent/did/README.md` lines 241 and 708 both advertise "Cache TTL: 5 minutes" for `MultiChainResolver`, while `grep -ril cache` over the non-test Go sources in `pkg/agent/did/` matches only a comment in `types_v4.go` — **no resolver cache is implemented today** [High]. That is convenient: a caching bound can be specified without breaking a deployed cache, because there is not one. The README line is wrong either way. A third detail matters for §6: `AgentKey` in `types_v4.go` carries `Type`, `KeyData`, `Verified` and `CreatedAt` but no revocation flag, so revocation is expressed by a key's *absence* from the record — and absence is exactly what a positive cache preserves [High].

---

## 2. What "immediate" can mean when the source of truth is a blockchain

Between an owner deciding to revoke and a verifier acting on it there are four delays: the transaction must reach a block; the block must reach the node the verifier reads; the reader must decide whether that block is safe enough to act on; and the reader must not be answering from an older copy. Only the last is under the verifier's control. "Immediate" can therefore only mean "the fourth delay is zero, and the third is a stated choice".

### 2.1 Ethereum

| Quantity | Value |
|---|---|
| Slot | 12 seconds |
| Epoch | 32 slots = 6.4 minutes |
| Checkpoint | first slot of each epoch |
| Finalisation | a justified checkpoint is finalised when the next checkpoint is justified |
| Measured `latest` → `finalized` gap | **16.0 minutes (p50 over 24 h), measured 2026-09-13** |

- "Ethereum glossary", ethereum.org, accessed 2026-09-13. https://ethereum.org/en/glossary/ — "Epoch: A period of 32 slots, each slot being 12 seconds, totalling 6.4 minutes"; "Slot: A period of time (12 seconds) in which new blocks can be proposed".
- "Proof-of-stake (PoS)", ethereum.org developer docs, accessed 2026-09-13. https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/ — checkpoint pairs, two-thirds supermajority, justification then finalisation; a finalised block "cannot be reverted or changed without a majority slashing of stakers".
- "JSON-RPC API", ethereum.org developer docs, accessed 2026-09-13. https://ethereum.org/en/developers/docs/apis/json-rpc/ — the block parameter accepts `latest` "for the latest proposed block", `safe` "for the latest safe head block", `finalized` "for the latest finalized block".
- MariusVanDerWijden, "core, eth, rpc: implement safe rpc block", go-ethereum PR #25165, GitHub, 2022. https://github.com/ethereum/go-ethereum/pull/25165 — the change that introduced the `safe` tag, mapped to the justified checkpoint.
- "L1 finality benchmark", OpenChainBench, measured 2026-09-13. https://openchainbench.com/benchmarks/l1-finality — Ethereum 16.0 min p50 over 24 h, by polling `eth_getBlockByNumber("latest")` and `eth_getBlockByNumber("finalized")` every 10 s and reporting the timestamp delta.
- "Ethereum Reorgs After The Merge", Paradigm, 2021-07. https://www.paradigm.xyz/2021/07/ethereum-reorgs-after-the-merge — why shallow single-slot reorgs persist under proof of stake.

The measured 16.0 minutes exceeds the "two epochs, 12.8 minutes" figure usually quoted, and should [High]: 12.8 minutes is the gap between a checkpoint being justified and being finalised, whereas a particular block also waits out the remainder of its own epoch, giving a wall-clock range of roughly 12.8 to 19.2 minutes with a median in between.

| Read at | Staleness for a revocation | Risk taken |
|---|---|---|
| `latest` | ~12 s after inclusion | the block may be reorganised out; the revocation is seen, then unseen |
| `safe` (justified) | ~6.4 min | reverting requires forking a justified checkpoint |
| `finalized` | ~16 min (p50, 2026-09-13) | reverting requires slashing a third of the stake |

For revocation the reorg risk runs in the safe direction [High]. A reorg that removes a revocation makes a verifier reject a message it would otherwise accept — it fails closed. A reorg that removes a *registration* makes a verifier accept a key that is not in the canonical record, which is the dangerous direction. This asymmetry is the strongest argument in this report for splitting the commitment level by what is being read (§6.4).

### 2.2 Solana

| Quantity | Value |
|---|---|
| Slot time | **350 ms since 2026-08-22** (previously 400 ms) |
| `processed` | node's most recent processed block; "can still be rolled back" |
| `confirmed` | voted on by a supermajority (>2/3) of stake |
| `finalized` | maximum lockout; ≥31 further confirmed blocks on top |
| Measured finality | **9.9 seconds (p50 over 24 h), measured 2026-09-13** |

- "Terminology", Solana documentation, accessed 2026-09-13. https://solana.com/docs/references/terminology — "Confirmed: a block that has received a super majority of ledger votes"; "Finalized: when nodes representing 2/3rd of the stake have a common root".
- "RPC API", Solana documentation, accessed 2026-09-13. https://solana.com/docs/rpc — `processed` is "the node's most recent processed block. This is the newest view, but it can still be rolled back".
- "What are Solana Commitment Levels?", Helius, accessed 2026-09-13. https://www.helius.dev/blog/solana-commitment-levels — finalized means "at least 31 further confirmed blocks have been built on top of it"; reports "No confirmed block has reverted in Solana's five-year history".
- "Solana cuts mainnet slot time to 350 milliseconds in first step toward 200ms goal", The Block, 2026-08-22. https://www.theblock.co/news/ecosystems/2026-08-22-solana-cuts-mainnet-slot-time-to-350-milliseconds-in-first-step-toward-200ms-goal-412521 — SIMD-0525, merged 2026-05-14, activated 2026-08-22; four staged 50 ms cuts planned toward 200 ms.
- "Alpenglow", Solana, accessed 2026-09-13. https://solana.com/upgrades/alpenglow — Votor plus BLS-aggregated votes, targeting roughly 150 ms finality; governance vote passed with 98.27% support in September 2025, activation targeted for October 2026 via Agave 4.3.

Both Solana numbers are moving, and both move in SAGE's favour [Mid]. A specification written now should name the commitment level and let the profile carry the measurement, which is what R-13 already asks for.

### 2.3 What a verifier can observe, and how fast

| Chain | Fastest honest observation | Irreversible observation |
|---|---|---|
| Ethereum | ~12 s (one slot, read at `latest`) | ~16 min (`finalized`, p50, 2026-09-13) |
| Solana | ~0.35–1 s (`processed`/`confirmed`) | ~10 s (`finalized`, 2026-09-13) |

Neither floor is affected by anything the SAGE text says. What the text controls is the *extra* delay the verifier adds (the cache — fixed at zero by charter §9) and which column it must read.

One cost is easy to miss: on Ethereum, reading at `finalized` is a different query, not merely a staler one, and is not served from the node's hot path the way `latest` is [Mid]. Vendor-published EVM read latencies for common methods run from about 15 ms to about 120 ms p50 depending on provider, measured 2026-08-08 to 2026-08-13 across four regions — though the publisher is one of the benchmarked providers and reports itself fastest, so treat the spread, not the winner, as the usable fact [Mid].

- "RPC provider benchmarks", Alchemy, measurement window 2026-08-08 to 2026-08-13, accessed 2026-09-13. https://www.alchemy.com/benchmarks

---

## 3. How other systems make revocation take effect quickly

### 3.1 Certificate revocation lists

A CA signs a periodic list of revoked serial numbers; the relying party caches it until `nextUpdate` and checks locally (RFC 5280 §5). Since Ballot SC-063v4, CRL publication is mandatory for CAs in the Baseline Requirements. BR v2.1.6 §4.9.7 requires a subscriber CRL to be reissued at least every **7 days** where the certificate carries an OCSP pointer and every **4 days** otherwise, and **within 24 hours** of recording a revocation. Worst case: a revocation may be up to 24 hours from publication, and a party holding a cached CRL may rely on data 4–7 days old. Cost to the verifier: a periodic file download plus a local lookup — near zero per verification, but bandwidth grows with the CA. Failure mode: historically fail-open; Chrome does not fetch per-certificate CRLs at all.

- CA/Browser Forum, "Baseline Requirements for the Issuance and Management of Publicly-Trusted TLS Server Certificates", v2.1.6, 2025-07-21, §4.9.7 and §4.9.9. https://cabforum.org/working-groups/server/baseline-requirements/documents/CA-Browser-Forum-TLS-BR-2.1.6.pdf

The aggregated push variants are the interesting part for SAGE. Mozilla's CRLite compresses the whole Web PKI's revocation set, built from CT logs, into a filter cascade pushed to Firefox on roughly a 12-hour cadence, enabled for all desktop users from Firefox 137. Chrome's CRLSet pushes a curated subset through the component updater and is described by Google as an emergency blocklist, not a complete feed; no update SLA is published (unverified).

- "CRLite: Fast, private, and comprehensive certificate revocation checking in Firefox", Mozilla Hacks, 2025-08. https://hacks.mozilla.org/2025/08/crlite-fast-private-and-comprehensive-certificate-revocation-checking-in-firefox/
- "CRLSets", Chromium security documentation, accessed 2026-09-13. https://chromium.googlesource.com/playground/chromium-org-site/+/refs/heads/main/Home/chromium-security/crlsets.md

Relevance [High]: CRLite and CRLSet distribute the *revocations*, not the good state. That is the one caching shape compatible with R-9, and §6.4 recommends it.

### 3.2 OCSP

A signed per-certificate status response carrying `thisUpdate`/`producedAt`/`nextUpdate` (RFC 6960). BR §4.9.9 caps subscriber response validity between 8 hours and **10 days**, and since 2025-01-15 requires an authoritative response within 15 minutes of issuance. Worst case handed to a relying party: a compliant response up to 10 days old. Cost: one round trip per uncached check, plus the privacy leak of telling the CA which site is being visited. Failure mode: soft-fail in every mainstream browser — unreachable responder means the certificate is treated as good.

OCSP is being withdrawn. Ballot SC-063v4 ("Make OCSP Optional, Require CRLs, and Incentivize Automation") passed in August 2023, effective 2024-03-15 (the ballot page did not return fetchable content this session; corroborated by an oss-security post of 2024-03-12 — treat the ballot page itself as unverified). Let's Encrypt then shut its responders down: Must-Staple requests began failing 2025-01-30, OCSP URLs were dropped from certificates 2025-05-07, and the service reached end of life 2025-08-06.

- Santesson, S., et al. "X.509 Internet PKI Online Certificate Status Protocol — OCSP", RFC 6960, IETF, 2013-06. https://www.rfc-editor.org/rfc/rfc6960
- "Intent to End OCSP Service", Let's Encrypt, 2024-12-05. https://letsencrypt.org/2024/12/05/ending-ocsp
- "OCSP Service Has Reached End of Life", Let's Encrypt, 2025-08-06. https://letsencrypt.org/2025/08/06/ocsp-service-has-reached-end-of-life
- "CA/Browser Forum Ballot SC-063v4", CA/Browser Forum, 2023-07-14. https://cabforum.org/2023/07/14/ballot-sc-063-v4make-ocsp-optional-require-crls-and-incentivize-automation/

### 3.3 OCSP stapling and Must-Staple

The server fetches the signed status itself and delivers it in the TLS handshake (`status_request`, RFC 6066); RFC 7633's Must-Staple extension obliges a conforming client to hard-fail without one. Staleness bound: unchanged — whatever the OCSP response's window is, up to 10 days. Cost to the verifier: essentially zero extra network work, one signature check on data already in the handshake. Failure mode: without Must-Staple, the same soft-fail; with it, hard-fail, but only Firefox enforces it and fewer than 1% of certificates carry it.

Why it never became universal: Must-Staple is baked in at issuance and cannot be turned off, so a single stapling misconfiguration takes the site dark for every visitor; Chrome and Safari never shipped enforcement; and the industry chose short lifetimes instead. These reasons come from secondary retrospectives rather than a CA/Browser Forum primary source [Mid].

- Eastlake, D. "Transport Layer Security (TLS) Extensions: Extension Definitions", RFC 6066, IETF, 2011-01. https://www.rfc-editor.org/rfc/rfc6066
- Hallam-Baker, P. "X.509v3 TLS Feature Extension", RFC 7633, IETF, 2015-10. https://www.rfc-editor.org/rfc/rfc7633
- "The Slow Death of OCSP", Feisty Duck newsletter issue 121, accessed 2026-09-13. https://www.feistyduck.com/newsletter/issue_121_the_slow_death_of_ocsp

### 3.4 Short-lived certificates as a replacement for revocation

Ballot SC-081v3, proposed by Apple and approved **2025-04-11** (29 in favour, 0 against, 5 abstentions), puts maximum TLS certificate validity on a schedule: 398 days until 2026-03-15, **200 days** to 2027-03-14, **100 days** to 2029-03-14, and **47 days** from 2029-03-15, with validation-data reuse shrinking to 10 days on the same date. Let's Encrypt moved faster on its own: first six-day certificate issued 2025-02-20, general availability of the 160-hour `shortlived` profile (and IP-address certificates, which must be short-lived) on 2026-01-15, and the default lifetime cut from 90 to 45 days announced 2025-12-02.

Worst-case delay: the certificate's own lifetime. Cost to the verifier: zero — it checks `notAfter`, which it already does. Failure mode: there is no status source to be unreachable; the dependency moves to the CA's issuance path, and an ACME outage takes the site dark rather than leaving a possibly-revoked certificate in use.

- "Ballot SC081v3: Introduce Schedule of Reducing Validity and Data Reuse Periods", CA/Browser Forum, 2025-04-11. https://cabforum.org/2025/04/11/ballot-sc081v3-introduce-schedule-of-reducing-validity-and-data-reuse-periods/
- "6-day and IP Address Certificates are Generally Available", Let's Encrypt, 2026-01-15. https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability
- "From 90 to 45", Let's Encrypt, 2025-12-02. https://letsencrypt.org/2025/12/02/from-90-to-45

Relevance [High]: this is the industry's verdict on the question SAGE is asking. Given a decade to make revocation work, the Web PKI's answer was to shorten lifetimes until revocation stopped mattering. SAGE cannot copy it directly — a registered agent identity is not a certificate with an expiry — but §5.2's delegated keys are the same move.

### 3.5 Certificate transparency

RFC 9162 logs every issued certificate in an append-only Merkle tree and issues Signed Certificate Timestamps as promises of inclusion. The RFC states plainly that "the logs do not themselves prevent misissuance": CT is detection, not revocation, and a CT-compliant certificate may be revoked, expired or compromised. The RFC deliberately sets no numeric Maximum Merge Delay; Chrome's log policy does, requiring an MMD of at most 4 hours for RFC 6962-style logs and at most 1 minute for static-ct-api logs. Cost to the verifier: two or three SCT signature checks against known log keys, no live query. Failure mode: if Chrome cannot refresh its log list for 70 days, CT enforcement disables itself — fail-open.

- Laurie, B., Messeri, E., Stradling, R. "Certificate Transparency Version 2.0", RFC 9162, IETF, 2021-12. https://www.rfc-editor.org/rfc/rfc9162
- "Chrome Certificate Transparency Log Policy", Google, accessed 2026-09-13. https://googlechrome.github.io/CertificateTransparency/log_policy.html

### 3.6 Credentials, tokens and tickets

**W3C Bitstring Status List v1.0** (Recommendation, 2025-05-15) publishes a compressed bitstring as a status credential; each credential names its index, and the verifier fetches the list once and reads one bit. Herd anonymity is bought with a minimum list length of 131,072. It has an optional `ttl` in milliseconds, but the specification says it "does not override or replace the validity period" — it is advisory, not a cache bound, so the worst-case delay is whatever the implementer's refetch period happens to be, and the behaviour when the list is unreachable is undefined by the specification [High].

**OAuth token revocation and introspection.** RFC 7009 revokes a token at the authorisation server, but RFC 7009 §2.1 itself warns that for self-contained tokens "there could be a propagation delay… some servers know about the invalidation while others do not". For a JWT access token verified locally, the worst-case delay is therefore the token's whole remaining lifetime; making revocation effective requires RFC 7662 introspection on every request, whose §4 states the trade-off explicitly — a less aggressive cache with a short timeout is more up to date "at cost of increased network traffic". The only hard rule is that a response carrying `exp` MUST NOT be cached past it. Behaviour when the introspection endpoint is down is left to the resource server.

**Push-based session revocation.** OpenID Connect Back-Channel Logout 1.0 (Final) posts a signed Logout Token directly to each relying party, bypassing the browser; the OpenID Shared Signals Framework with CAEP and RISC generalises this to security events, all three approved as Final Specifications on 2025-09-02. Delivery is best-effort HTTP: a relying party that is down when the event fires keeps the session until it expires locally, so the token lifetime remains the real backstop [Mid].

**Kerberos.** RFC 4120 tickets are self-contained and valid to their expiry without further KDC contact, so short lifetimes *are* the revocation mechanism. MIT Kerberos `max_life` and the Active Directory default domain policy both default to **10 hours** for user and service tickets. Disabling an account mid-ticket does not invalidate tickets already issued; the service verifies locally with the session key at zero per-request cost, and an unreachable KDC stops new tickets, not existing ones [Mid — the mid-ticket behaviour is documented by vendor and community sources rather than RFC 4120 itself].

- "Bitstring Status List v1.0", W3C Recommendation, 2025-05-15. https://www.w3.org/TR/vc-bitstring-status-list/
- Lodderstedt, T., Dronia, S., Scurtescu, M. "OAuth 2.0 Token Revocation", RFC 7009, IETF, 2013-08. https://www.rfc-editor.org/rfc/rfc7009
- Richer, J. "OAuth 2.0 Token Introspection", RFC 7662, IETF, 2015-10. https://www.rfc-editor.org/rfc/rfc7662
- "OpenID Connect Back-Channel Logout 1.0", OpenID Foundation, Final. https://openid.net/specs/openid-connect-backchannel-1_0.html
- "OpenID Shared Signals Framework 1.0", OpenID Foundation, Final Specification, 2025-09-02. https://openid.net/specs/openid-sharedsignals-framework-1_0-final.html
- Neuman, C., Yu, T., Hartman, S., Raeburn, K. "The Kerberos Network Authentication Service (V5)", RFC 4120, IETF, 2005-07. https://www.rfc-editor.org/rfc/rfc4120

### 3.7 What the comparison shows

| Mechanism | Worst-case delay | Verifier cost per check | If the status source is unreachable |
|---|---|---|---|
| CRL | ≤24 h to publish; 4–7 d cached | periodic download + local lookup | fail-open in practice |
| CRLite / CRLSet | ~12 h (CRLite); unpublished (CRLSet) | local set lookup, no network | under-blocks silently |
| OCSP | ≤10 days | one round trip, or amortised | soft-fail (fail-open) |
| Stapling / Must-Staple | ≤10 days | ~zero, one signature | soft-fail; hard-fail under Must-Staple (<1% of certs) |
| Short-lived certificates | the certificate lifetime (47 d by 2029; 160 h at Let's Encrypt) | ~zero | not applicable; risk moves to issuance availability |
| Certificate transparency | not a status feed (detection only) | 2–3 signature checks | enforcement self-disables after 70 d |
| Bitstring Status List | unbounded by the specification | one list fetch, one bit | undefined |
| OAuth introspection | token lifetime without it; zero with it | one round trip per request | undefined |
| Kerberos | ticket lifetime (10 h default) | zero | existing tickets keep working |

Every mechanism here is either *bounded and stale* or *unbounded and free* [High]. Not one of them achieves immediacy without a per-verification round trip, and the two that come closest — Must-Staple and introspection-on-every-request — are the two the industry has most conspicuously failed to adopt, for the same reason in both cases: they make the verifier's availability depend on a third party's.

---

## 4. Event-driven invalidation

The obvious way to keep a cache and still be fast is to subscribe to registry events and drop the entry when a revocation is logged. The research is mostly about why that guarantee is weaker than it appears.

### 4.1 What a node delivers

Ethereum's `eth_subscribe` offers `logs` and `newHeads` over WebSocket, and its reorg behaviour is documented and is the good kind: "In case of a chain reorganization previous sent logs that are on the old chain will be resent with the removed property set to true", with the new chain's logs emitted as well — so the same transaction's log can arrive more than once and consumers must be idempotent. `newHeads` emits only the last header of the new chain, so several headers may arrive for one height. The polling alternative is worse: filters made with `eth_newFilter` are garbage-collected after a default five minutes (`cfg.Timeout = 5 * time.Minute` in `eth/filters/filter_system.go`), after which polling returns "filter not found" and the consumer has lost its position.

- "Real-time Events", go-ethereum documentation, accessed 2026-09-13. https://geth.ethereum.org/docs/interacting-with-geth/rpc/pubsub
- go-ethereum, `eth/filters/filter_system.go`, GitHub master, accessed 2026-09-13. https://github.com/ethereum/go-ethereum/blob/master/eth/filters/filter_system.go
- "Understanding Ethereum's filter not found error", Chainstack documentation, accessed 2026-09-13. https://docs.chainstack.com/docs/understanding-ethereums-filter-not-found-error-and-how-to-fix-it

### 4.2 The delivery guarantee is the problem

**A node gives no replay.** Events emitted while a subscriber is disconnected are not redelivered on resubscription; the subscription resumes from the present. The clearest evidence that this is the raw behaviour is that a provider had to build a compensating layer over it — Alchemy's SDK re-sends events missed during a disconnection and states its own limit: beyond a gap of 120 blocks (about 20 minutes), events are still lost.

- alchemy-web3 README, GitHub, accessed 2026-09-13. https://github.com/alchemyplatform/alchemy-web3/blob/master/README.md
- "Subscription API Overview", Alchemy documentation, accessed 2026-09-13. https://www.alchemy.com/docs/reference/subscription-api — 100 to 2,000 concurrent connections by tier, up to 1,000 subscriptions per connection.

So delivery is at-most-once with silent loss, and recovery is a range query the client must issue itself. A verifier whose subscription drops must record the last block it fully processed, backfill with `eth_getLogs(fromBlock, toBlock)` on reconnect, and — the part that matters for R-9 — **treat its cache as stale for the whole gap**, because it cannot distinguish "no revocation happened" from "I was not listening" [High]. No primary document prescribes that fail-closed rule; it follows from the absence of replay.

### 4.3 Solana

`logsSubscribe`, `accountSubscribe` and `programSubscribe` take a commitment, and the commitment decides what a notification means: `processed` may be for a slot the cluster later abandons, `confirmed` has a supermajority behind it, `finalized` has maximum lockout. For volume the production answer is not the WebSocket RPC at all but a Geyser plugin streaming from validator memory over gRPC.

- "logsSubscribe" / "accountSubscribe" / "programSubscribe", Solana documentation, accessed 2026-09-13. https://solana.com/docs/rpc/websocket/logssubscribe
- "Commitment Status", Agave documentation, accessed 2026-09-13. https://docs.anza.xyz/consensus/commitments
- rpcpool/yellowstone-grpc, GitHub, accessed 2026-09-13. https://github.com/rpcpool/yellowstone-grpc

### 4.4 Indexers and light clients

Indexers bound the reorg problem rather than solving it. graph-node keeps a block pointer, reverts dynamic data sources on a reorg and tracks `reorg_count`, `current_reorg_depth` and `max_reorg_depth`; its `ETHEREUM_REORG_THRESHOLD` defaults to **250** blocks, documented as the "maximum expected reorg size, if a larger reorg happens, subgraphs might process inconsistent data". Subsquid buffers unfinalized "hot" blocks and rolls back database changes for orphaned ones, warning explicitly that external side effects cannot be rolled back and that irreversible actions should wait for a finality threshold.

- "Environment variables", graph-node, GitHub master, accessed 2026-09-13. https://github.com/graphprotocol/graph-node/blob/master/docs/environment-variables.md
- "RPC ingestion and reorgs", SQD documentation, accessed 2026-09-13. https://docs.sqd.ai/sdk/resources/unfinalized-blocks/

Light clients make the distinction structural. The Altair light-client store holds two heads: a `finalized_header`, advanced only on a two-thirds sync-committee signature plus a Merkle proof of the finalized checkpoint, and an `optimistic_header`, advanced on participation alone with no finality proof; `LightClientFinalityUpdate` and `LightClientOptimisticUpdate` are separate messages. Helios implements this to turn an untrusted RPC endpoint into a locally verified one rooted in a recent consensus checkpoint.

- "Altair light client — sync protocol", ethereum/consensus-specs, GitHub master, accessed 2026-09-13. https://github.com/ethereum/consensus-specs/blob/master/specs/altair/light-client/sync-protocol.md
- a16z/helios, GitHub, accessed 2026-09-13. https://github.com/a16z/helios

Relevance [High]: the light-client store is the shape a SAGE resolver should have — two heads, with the text saying which decisions may be taken against which. That is a better primitive than one cache with a TTL, and it is the same split §6.4 recommends.

---

## 5. Designs that avoid the question

### 5.1 The identifier carries the key

`did:key` derives the identifier from the public key, so resolution is a local computation and no registry is consulted. The cost is total: the specification states the method "cannot be updated or deactivated", does not support key rotation, and "provides no mechanism for deactivating or revoking", recommending it for ephemeral use only. Relevance [High]: this is the pure form of a trade SAGE has already refused — charter §9 decision 2 makes registration a prerequisite, and R-10 requires rotation without changing the identifier — so it is available only as a subordinate construction (§5.2).

KERI keeps self-certification while adding rotation: an autonomic identifier is derived from an initial key, every key change is an event in a signed key event log, pre-rotation commits to the hash of the next key before use, and witnesses countersign so that a signer presenting two histories is detectably duplicitous. Revocation is a rotation event. Relevance [Mid]: KERI is the closest published answer to "a registry that is not a blockchain", which charter R-3 now explicitly requires a profile for. It does not make revocation instantaneous, but it moves the floor from consensus finality to one network round trip and makes withholding detectable. Worth the R-3 profile study; out of scope for R-9.

- "The did:key Method v0.9", W3C Credentials Community Group, accessed 2026-09-13. https://w3c-ccg.github.io/did-key-spec/
- Smith, S. M. "Key Event Receipt Infrastructure (KERI)", arXiv:1907.02143, 2019-07-03. https://arxiv.org/abs/1907.02143

Adjacent, and relevant to R-14: W3C DID Resolution v1 (Candidate Recommendation Draft, 2026-08-28) requires that "if a DID has been deactivated, DID document metadata MUST include this property with the boolean value `true`" — so the standard machinery has a place to carry deactivation, but it defines no cache-control or freshness directive for resolution results [High].

- "Decentralized Identifier Resolution (DID Resolution) v1.0", W3C Candidate Recommendation Draft, 2026-08-28. https://www.w3.org/TR/did-resolution/

### 5.2 Capabilities with short expiry

UCAN carries `nbf`/`exp` in the token, so the natural revocation story is expiry; a separate revocation sub-specification adds a CID-keyed list and is RECOMMENDED rather than required. Biscuit verifies offline through a public-key signature chain and supports TTL caveats, but puts revocation outside the token — a `revocation_id` the application must check against externally managed state. Macaroons, the original of the family, attenuate authority by appending caveats chained into an HMAC so any holder can narrow a token without contacting the issuer, and their answer to revocation is likewise a short expiry caveat.

- "UCAN Specification v1.0.0", ucan-wg, GitHub, accessed 2026-09-13. https://github.com/ucan-wg/spec
- "Revocation", Biscuit documentation, accessed 2026-09-13. https://biscuitsec.org/docs/guides/revocation
- Birgisson, A., Politz, J. G., Erlingsson, Ú., Taly, A., Vrable, M., Lentczner, M. "Macaroons: Cookies with Contextual Caveats for Decentralized Authorization in the Cloud", NDSS 2014, San Diego, 2014-02. https://theory.stanford.edu/~ataly/Papers/macaroons.pdf

SPIFFE/SPIRE applies the same idea to workload X.509 identities with automatic rotation partway through a short SVID lifetime; the research could not settle the documented default TTL (sources gave both 1 hour and 6 hours), so no number is quoted here (unverified).

Relevance [High]: the pattern recurring in all of these is **make the credential expire faster than harm accumulates, and accept that anything shorter than the TTL needs a list after all**. The consequence for SAGE is that a delegated-key design does not remove the registry; it moves it off the per-message path and onto the per-delegation path.

### 5.3 A freshness proof travelling with the message

Three established shapes attach evidence of the authority's state to the message so the verifier checks a signature instead of making a query. OCSP stapling is the web-PKI form (§3.3). Certificate transparency is the log form: the log signs a tree head and the merge delay bounds how stale it may be (§3.5). For a chain the analogue is already a standard method: EIP-1186's `eth_getProof` returns an account or storage value with a Merkle-Patricia proof against the `stateRoot` in a named block header, so a sender can attach `(block number, block hash, proof)` and let the verifier check, with no RPC of its own, that the registry held exactly that record at exactly that block.

- Kilian, S., Bogensperger, S., et al. "EIP-1186: RPC-Method to get Merkle Proofs — eth_getProof", Ethereum Improvement Proposals, 2018-06-24. https://eips.ethereum.org/EIPS/eip-1186

Relevance [High], with one hard limit [High]: **a proof of inclusion is not a proof of currency.** A storage proof shows the record at block N; it cannot show that no revocation happened at block N+1, and a revoked sender will keep stapling its last favourable proof. The proof converts the problem into "how old may N be?", which is the caching bound restated, and the verifier now needs a current block reference from somewhere to judge it. This is the same weakness a replayed but unexpired OCSP response has, and it is why §6 scores the stapled-freshness option as no better than its bound rather than as a way out of it.

---

## 6. Recommendation space for SAGE

### 6.1 What the options are scored against

R-9 forbids caching from extending the life of a revoked key. One property decides most of the table [High]: **the registry's answer to "is this key revoked?" is a negative, and negatives cannot be cached safely.** A cached record proves the key existed at time T and proves nothing about T+1. That is why CRLs, status lists and OCSP all carry an explicit `nextUpdate`, and why the Web PKI's failure mode is soft-fail. SAGE may choose a different failure mode, because unlike a browser it is not obliged to keep the connection working.

### 6.2 The options

| # | Option | Guarantee | Latency and cost per verified message | If the registry is unreachable | Specification complexity |
|---|---|---|---|---|---|
| 1 | Verify against the latest **finalised** state on every message | Revocation effective one finalisation after it is mined: ~16 min Ethereum, ~10 s Solana. No staleness beyond that | One RPC per message; 15–120 ms p50 for common EVM reads, and a `finalized`-pinned read is not on the node's hot path, so treat that as a floor | Nothing verifies. Hard fail-closed; the verifier is as available as its RPC provider | Low — one sentence naming the commitment level per profile |
| 2 | Verify against **latest** state on every message | Revocation effective one block after mining: ~12 s Ethereum, ~0.35 s Solana. A reorg can un-see a revocation (fails closed) or a registration (fails open) | One RPC per message, the cheapest form | Same as 1 | Low, plus a paragraph on what a reorg means in each direction |
| 3 | Cache with a bound plus event-driven invalidation | Revocation effective within the bound; usually far sooner, but the *guarantee* is the bound, never the subscription (§4.2) | Amortised to a cache hit — microseconds — plus one subscription per verifier and a re-read every bound | Cache serves until the bound expires, then fail-closed. The only option with graceful degradation, and that window is what R-9 forbids | Medium-high: the bound, resubscribe-and-backfill, reorg handling and the meaning of "observed" all become normative |
| 4 | Registry-signed freshness statement attached to the message | Only as good as the statement's validity window — the same bound as option 3, moved onto the wire (§5.3) | No RPC for the record, but the verifier still needs a current block reference to judge the statement's age | The verifier can still act: the evidence travelled with the message. The only option that survives a registry outage | High, and it needs an entity that does not exist — a signer whose own key is subject to R-9 |
| 5 | Short-lived delegated keys | Compromise window bounded by the delegation lifetime, not the registry. Revoking the *root* key still needs one of 1–4 to take effect before the delegation expires | No per-message registry work; one registry read per delegation issued | Existing delegations keep working to expiry; new ones cannot be issued — Kerberos's failure mode exactly (§3.6) | High: a second key hierarchy, a delegation format, its own proof of possession, and a second thing for R-8's selection rule to choose among |

### 6.3 How the options score against R-9 as written

| Option | Satisfies "caching must not extend the life of a revoked key"? |
|---|---|
| 1 latest finalised state | Yes, trivially — there is no cache |
| 2 latest state | Yes, trivially — there is no cache |
| 3 bounded cache plus events | **No**, on a strict reading: the bound *is* an extension |
| 4 stapled freshness statement | **No**, for the same reason — the statement's window is a grace period wearing a different hat |
| 5 short-lived delegation | Not by itself; it bounds damage rather than making revocation immediate |

This is the finding the design stage must confront [High]: charter §9 has already ruled out the two options (3 and 4) that every comparable system in §3 converged on, leaving the two that put a network round trip on every verification. That is a coherent position — it is the position a payment system takes — but taking it knowingly matters, because it makes R-36 ("a responder does public-key or registry work only after the cheap checks it can make first") load-bearing, and it makes a verifier's availability equal to its RPC provider's.

### 6.4 Recommendations

1. **Keep the immediacy rule and define "observed" as a read the verifier performs for this message.** Options 1 and 2 are the only ones that meet R-9 unqualified, and one RPC per message is affordable for agent traffic in a way it is not for web browsing [Mid]: an agent call already costs tens or hundreds of milliseconds of model and tool work, so a 15–120 ms registry read is a large but not disqualifying fraction, whereas a page load makes thousands of such checks.
2. **Split the commitment level by what is read.** Read revocation and activation state at `latest`, where a reorg fails closed, and read the key material a signature is checked against at `finalized`, where a reorg would otherwise fail open (§2.1). On Ethereum this takes revocation latency from about 16 minutes to about 12 seconds while weakening nothing [Mid]. Cost: one sentence per profile and one extra vector.
3. **Permit a negative cache only, never a positive one.** A verifier may remember "revoked" indefinitely; it may never remember "not revoked". This is the only caching rule consistent with R-9's text, it is cheap to specify, and it is what CRLSets and CRLite actually do (§3.1) — they distribute revocations, not good state. It also repairs the gap noted in §1: because `AgentKey` has no revoked flag, revocation shows up as absence, which a positive cache silently preserves.
4. **Require fail-closed on a subscription gap if event-driven invalidation is ever permitted as an optimisation.** Even as a pure accelerator over options 1 or 2, the rule from §4.2 must be normative: record the last fully processed block, backfill by range query, and treat the gap as stale.
5. **Treat option 5 as a separate, later requirement.** Short-lived delegated keys answer a different question — how to keep a long-lived key out of a long-lived agent process — and folding them into R-9 would make immediacy depend on a key hierarchy that does not exist yet.
6. **Publish R-9's and R-13's numbers as a matrix, not a scalar.** "Delay between a revocation becoming observable and the first rejection" is a verifier property and should measure zero. "Delay between submitting a revocation and it becoming observable" is a chain property, differs by three orders of magnitude between Ethereum and Solana, and will change again when Alpenglow activates.
7. **Fix `pkg/agent/did/README.md`.** It documents a 5-minute resolver cache TTL that the code does not implement. Whichever option is chosen, that line is either wrong or a specification violation.

---

## Facts vs Opinions

**Fact** — verified on 2026-09-13 by fetching the cited source.
- Ethereum: 12-second slots, 32-slot (6.4-minute) epochs; `latest`/`safe`/`finalized` are defined block parameters; measured `latest`→`finalized` gap 16.0 min p50 over 24 h.
- Solana: slot time cut from 400 ms to 350 ms on 2026-08-22 by SIMD-0525; `finalized` requires ≥31 further confirmed blocks; measured finality 9.9 s p50 over 24 h; Alpenglow targets ~150 ms, activation targeted October 2026.
- CA/Browser Forum BR v2.1.6 §4.9.7: subscriber CRLs reissued at least every 7 days (4 without an OCSP pointer) and within 24 hours of a revocation; §4.9.9 caps OCSP response validity at 10 days.
- Ballot SC-081v3 approved 2025-04-11: 398 → 200 → 100 → 47 days by 2029-03-15.
- Let's Encrypt: first 6-day certificate 2025-02-20; 160-hour `shortlived` profile generally available 2026-01-15; OCSP end of life 2025-08-06.
- RFC 9162 states "the logs do not themselves prevent misissuance" and sets no numeric MMD; Chrome's log policy requires ≤4 h (RFC 6962-style) or ≤1 min (static-ct-api).
- W3C Bitstring Status List v1.0 is a Recommendation of 2025-05-15; its `ttl` "does not override or replace the validity period".
- RFC 7009 §2.1 warns of propagation delay for self-contained tokens; RFC 7662 §4 states the cache-freshness/traffic trade-off.
- MIT Kerberos and Active Directory both default user and service ticket lifetimes to 10 hours.
- go-ethereum re-sends reorganised logs with `removed: true`; `eth_newFilter` filters time out after a default 5 minutes.
- Alchemy's SDK loses events when a disconnection gap exceeds 120 blocks (~20 minutes).
- graph-node's `ETHEREUM_REORG_THRESHOLD` defaults to 250 blocks.
- The Altair light-client store keeps separate `finalized_header` and `optimistic_header`.
- `did:key` "cannot be updated or deactivated"; W3C DID Resolution v1 (CR Draft, 2026-08-28) requires `deactivated: true` in DID document metadata.
- EIP-1186 `eth_getProof` returns a Merkle-Patricia proof of an account or storage slot against a block's `stateRoot`.
- In this repository, no resolver cache is implemented in `pkg/agent/did/` despite `README.md` documenting a 5-minute TTL; `AgentKey` has no revocation field.

**Opinion** — inference.
- [High] Every surveyed mechanism is either bounded-and-stale or unbounded-and-free; none achieves immediacy without a per-verification round trip.
- [High] Negatives cannot be safely cached, so options 3 and 4 fail R-9 as written, and charter §9 has ruled out the designs the rest of the industry converged on.
- [High] A proof of inclusion at block N is not a proof of currency, so a stapled freshness statement reintroduces the caching bound rather than removing it.
- [High] For revocation, an Ethereum reorg fails closed; for registration it fails open — which is what makes the split in recommendation 2 safe.
- [High] With no replay guarantee, a verifier cannot distinguish "no revocation" from "not listening", so a subscription gap must be treated as stale.
- [Mid] One registry read per message is affordable for agent traffic because the surrounding model and tool work already costs more than the read.
- [Mid] Reading revocation state at `latest` and key material at `finalized` cuts Ethereum revocation latency from ~16 min to ~12 s without weakening the key binding.
- [Mid] Solana's numbers will keep improving (slot-time schedule, Alpenglow), so the specification should name commitment levels and leave numbers to per-profile measurements.
- [Mid] KERI deserves consideration in the R-3 non-blockchain registry profile study, not in the R-9 decision.
- [Low] Chrome's CRLSet update cadence and SPIFFE/SPIRE's default SVID TTL could not be confirmed from primary sources and are omitted rather than estimated; the CA/Browser Forum SC-063v4 ballot page and the "Chrome 19 disabled online revocation checks" claim were corroborated only by secondary sources.
