# docs/refactoring

Artifacts produced for the 2026-09 dependency update and refactoring design.

| Path | What |
|---|---|
| `REFACTORING_DESIGN.md` | Target architecture, SSOT table, call conventions, phased migration plan, gates, open decisions. Start here. |
| `analysis/01-crypto.md` | `pkg/agent/crypto/**`, `internal/cryptoinit` |
| `analysis/02-did-blockchain-config.md` | `pkg/agent/did/**`, `pkg/blockchain/**`, `deployments/config`, `cmd/sage-did` |
| `analysis/03-handshake-hpke-session-transport.md` | `pkg/agent/{handshake,hpke,session,transport}/**`, `internal/session_creator.go` |
| `analysis/04-core-storage-health-oidc-internal.md` | `pkg/agent/core/**`, `pkg/{storage,health,oidc,version}`, `internal/{logger,metrics}` |
| `analysis/05-cmd-lib-tests-sdk-contracts-ci.md` | `cmd/*`, `lib/`, `tests/`, `tools/`, `examples/`, `sdk/`, `contracts/`, Makefile and workflows |
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
