# Project review (2026-09-12)

A structured review of SAGE carried out before deciding what to do with the
deprecated handshake and nonce code and before the next refactoring steps.
Each document stands on its own and is meant to be loaded as context by a
later session; together they answer the eleven questions below. All were
written against `sage` `878932d`, `sage-spec` `a34dd49` and `rs-sage-core`
`206bbbb` on 2026-09-12. Every judgement carries a confidence label and
each document ends with a "Facts vs Opinions" block.

| Question | Document | What it holds |
|---|---|---|
| 1a. Documentation as a graph | `01-documentation-graph.md` | X-bar classification of every document changed since 2026-09-11 plus the specification and sibling READMEs (105 nodes), a 44-node concept graph, coverage per protocol layer, 18 contradictions with evidence |
| 1b. Code as a graph | `02-code-graph.md` | AST-derived package graph, the real call chains of every security feature with file:line, reachability from binaries, the Rust module map, the deprecated-symbol inventory |
| 2. History | `03-history-timeline.md` | Month-by-month timeline, the origin and fate of each security feature (4-phase handshake, HPKE, nonce and replay protection, sessions, registries, A2A), documentation staleness evidence, the 2026-09 cycle |
| 3. Purpose | `04-purpose-and-vision.md` | Problem statement, threat model as written, how the goals moved from 2025-07 to 2026-09, constraints, success criteria, divergences between documents |
| 4. Progress | `05-progress-assessment.md` | Capability matrix (specified, Go, Rust, live path, vectors, inspector, docs), progress figures with the rubric, the wiring gap, the 2026-09 cycle against the strategy |
| 5. Direction | `06-direction-and-stack-evaluation.md` | Each technology choice against its alternatives, where the stack is heavier or lighter than the purpose needs, overlapping mechanisms, ordered recommendations |
| 6. Research | `07-literature-review.md` | 45 cited works on agent security, decentralised identity, the standards used, blockchain trust registries and evaluation methods; gap analysis; candidate research questions |
| 7. Implementation | `08-implementation-evaluation.md` | Each technology as commonly implemented versus as implemented here (Go and Rust), findings with severity, how much runs on live paths |
| 8. Protocol | `09-protocol-flows.md` | RFC-style procedures and sequence diagrams for signed requests and responses, DID resolution and cards, the HPKE handshake, session records and on-chain registration; state machines; spec/Go/Rust/vector coverage |
| 9. Conformance | `10-inspector-test-matrix.md` | Every positive, negative and edge case an inspector must check, what `sage-inspector` covers today, missing checks and vectors with effort |
| 10. Repositories | `11-repository-split-and-research-plan.md` | Target repository set, code-reuse mechanisms, migration steps with gates and the removal schedule, research plan with experiments and metrics, open decisions |
| Follow-up: identity standards | `12-did-standards-research.md` | What a conformant identifier method must contain, how existing methods guarantee uniqueness and prove control, and what the identifier scheme is missing; grounded in the maintainer's own study of the standards |
| Follow-up: revocation | `13-revocation-research.md` | What immediate revocation can mean when the registry confirms in blocks, how other systems bound the delay, and the options with their cost per verified message |
| Follow-up: registry coupling | `14-registry-coupling-analysis.md` | Every place a chain, address format, key type or algorithm is decided in the two cores and the gateway; what the protocol actually needs from a registry; what breaks without a blockchain |
| Follow-up: design proposal | `15-identity-and-registry-design.md` | The identifier scheme, the chain-neutral registry model with profiles including one that is not a blockchain, the standards to adopt, how revocation is made immediate, and the decisions this needs |
| 11. Persistence | this directory | The documents are committed here so they survive the session; regenerate the graphs with `make codegraph` and the method in 01 |

## Reading order

Start with `05` for the state of the project and `06` for the decisions it
asks for; `11` turns those into a plan. The rest are the evidence.

## Headline findings

- The specification and its 26 test vectors are the only current protocol
  text; every handshake and session guide in `sage` disagrees with the code
  somewhere (01, 03).
- The HPKE handshake and the session record layer have no caller outside
  tests and examples, and no live path configures TLS, so the "end-to-end
  encrypted" claim rests on tests alone (02, 05, 08).
- The 4-phase handshake was superseded for three recorded reasons, at
  different times: round trips, a server that never read its nonce and
  timestamp, and no consumer (03). Removing it also removes `core/message`
  and the only implementer of `hpke.KeyIDBinder` (02).
- Two design claims are contradicted by published measurements: on-chain
  reputation as a trust signal and the cost and latency of an Ethereum
  registry on the critical path (07).
- The Go and Rust cores disagree on several verification decisions (unknown
  signature parameters, DER signatures, body limits, response binding,
  card key formats), which the inspector does not yet detect (09, 10).
