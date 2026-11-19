# CI version suites

The root module holds code of several spec versions: the draft.1 code that
the `core` module replaces, and the 0.10 and 0.11 packages. CI tests each
version as a suite so that work on one version is checked by that version's
tests and does not wait for unrelated ones (decision D27).

## Which package belongs to which version

`.github/ci/versions.json` lists package patterns per version. A package
belongs to the first version whose pattern matches; draft.1 takes every
package that no other version names, including version-neutral packages.
`python3 .github/scripts/ci_versions.py check` fails when a package has no
version, so a new package must be classified in the same pull request.

The `core` module is the draft.1.1 implementation. It has its own workflow,
`core.yml`, which runs on every push, pull request and manual run.

## What runs

| Event | draft.1 suite | 0.10 and 0.11 jobs |
|---|---|---|
| Pull request into a version branch (`spec/**`, `dev`) | always | when affected |
| Pull request into `main`, any push, manual run | always | always |

- draft.1 suite: `core.yml` (core tests, fuzz, lint, budget), `draft.1 Go
  Tests` (the draft.1 packages with the race detector), Lint, Build,
  Contract Bindings, and Spec Vectors against sage-spec pinned at the commit
  in `test.yml`.
- 0.10 and 0.11 jobs: Native Resource Primitives, Registry Fixture Check,
  Registry Source Shard 0-8, Registry Source Checks and Go Tests. Go Tests
  runs every package except registry011 (the shards run it) and holds the
  coverage gates and evidence of 0.10 and 0.11, so the chain runs when
  either version is affected.

## How a pull request is judged

The selector builds the package graph with `go list -test -deps` for each
build configuration in `graph_configs` (linux, darwin and windows, with the
`registry_native_limits` and `e2e` tags) and merges them. A version is
affected when the pull request changes one of its packages or a package one
of its packages imports, directly, transitively or from a test.

A changed file maps to a package as follows:

- a Go file to the package of its own directory; a Go file in a directory
  that is not a package (a new, moved or removed package) selects every
  suite;
- any other file to the package whose directory is its nearest ancestor
  (testdata, embedded files, C sources);
- files matching `no_root_effect` (`core/`, `docs/`, `core.yml`, top-level
  Markdown) change no root package;
- any other file outside a package directory (scripts, configuration, other
  workflows) and any file in `all_suites_when_changed` (`go.mod`, `go.sum`,
  the selector and `test.yml`) selects every suite.

The changed files come from `git diff --no-renames --name-only -z`, so a
moved file counts at both paths and file names are not quoted.

If the job `Select version suites` fails, every gated job still runs and
fails at its first step; a required check such as Go Tests is never skipped
because the selection failed. The selector's tests are
`.github/scripts/test_ci_versions.py`.
