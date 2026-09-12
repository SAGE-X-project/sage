# docs/refactoring/v2

Working documents for the second phase of the refactoring: splitting SAGE into
protocol, core, contract, gateway and inspection repositories. Every document
produced from 2026-09-12 onward lives here; the first-phase artifacts in
`docs/refactoring/` (audits, design, backlog) stay where they are and are
referenced, not duplicated.

| Path | What |
|---|---|
| `REPO_PLAN.md` | Final repository plan: naming rule, assessment of `rs-sage-core`, role of each repository, dependency direction, execution order, open decisions. Start here. |
| `LICENSING.md` | Licence review of `sage` and the new repositories; per-repository licence recommendation and the fixes required before any release. |
| `RS_SAGE_CORE_ALIGNMENT.md` | F-03: divergences of `rs-sage-core` from sage-spec by module, and the ordered pull-request plan to close them. |

Naming rule used throughout this directory:

- `sage` is the Go reference core. When older documents say "sage-core" in the
  Go sense, they mean this repository.
- `rs-sage-core` is the Rust core. Older documents that say "sage-core (Rust)"
  mean this repository.
- `sage-spec`, `sage-contracts`, `sage-gateway`, `sage-inspector` are the
  repositories created on 2026-09-11 and 2026-09-12 under
  `github.com/SAGE-X-project`.
