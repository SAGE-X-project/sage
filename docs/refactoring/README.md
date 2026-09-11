# docs/refactoring

Artifacts produced for the 2026-09 dependency update and refactoring design.

| Path | What |
|---|---|
| `BACKLOG.md` | Tracking list of every finding (governance, security wiring, contracts, refactoring, docs, strategy) with ID, severity, source and status. Update here when an item lands. |
| `STRATEGY.md` | Final synthesis: protocol-first multi-repository target (spec + Go reference + Rust FFI/WASM core + SDKs + gateway + contracts), determinism requirements, threat coverage, migration sequence, versioning and licensing policy. Start here. |
| `SECURITY_WIRING_AUDIT.md` | Which security controls sit on the real request paths, which are dead code, executed proof tests, determinism analysis, prioritised wiring fixes. |
| `SUPPLY_CHAIN_AUDIT.md` | CI/release/governance audit with Scorecard-style checklist and P0/P1/P2 remediation, multi-repo version policy. |
| `DOCS_GRAPH.md` | All 139 documents classified as X-bar style nodes (head/type/specifier/complement/adjuncts) with freshness verdicts, cluster graph, stale list, gaps, contradictions. |
| `REFACTORING_DESIGN.md` | Target architecture, SSOT table, call conventions, phased migration plan, gates, open decisions. |
| `DECISIONS.md` | Proposed answers to the five open decisions (handshake, import paths, cgo lib, SDKs, V2 registry) with evidence from external consumers. |
| `FEATURE_MAP.md` | What each binary, example and external consumer actually reaches; feature-by-feature working/broken status. |
| `analysis/01-crypto.md` | `pkg/agent/crypto/**`, `internal/cryptoinit` |
| `analysis/02-did-blockchain-config.md` | `pkg/agent/did/**`, `pkg/blockchain/**`, `deployments/config`, `cmd/sage-did` |
| `analysis/03-handshake-hpke-session-transport.md` | `pkg/agent/{handshake,hpke,session,transport}/**`, `internal/session_creator.go` |
| `analysis/04-core-storage-health-oidc-internal.md` | `pkg/agent/core/**`, `pkg/{storage,health,oidc,version}`, `internal/{logger,metrics}` |
| `analysis/05-cmd-lib-tests-sdk-contracts-ci.md` | `cmd/*`, `lib/`, `tests/`, `tools/`, `examples/`, `sdk/`, `contracts/`, Makefile and workflows |
| `graph/entrypoints.md` | Cobra command inventory and call/ref reachability from every entry point; pkg packages no binary reaches |
| `graph/summary.md` | Code graph metrics: package table, import graph (mermaid), cycles, layer violations, interfaces/implementers, hot functions, duplicates, dead-code candidates |
| `graph/graph.json` | Full node/edge list (regenerated, not committed) |
| `PR_LOG.md` | Per-PR verification log for the dependency update |

Regenerate the graph:

```bash
cd tools/codegraph
go run . -dir ../.. -out ../../docs/refactoring/graph
# with a delta against a previous run
go run . -dir ../.. -out ../../docs/refactoring/graph -diff /path/to/previous/graph.json
```

`tools/codegraph` is its own Go module so the main module does not depend on `golang.org/x/tools`.
