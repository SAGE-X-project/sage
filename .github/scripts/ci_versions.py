"""Spec-version suites for CI (decision D27).

Every package of the root module belongs to one spec version
(.github/ci/versions.json). A pull request into a version branch always runs
the draft.1 suite; it runs the 0.10 or 0.11 suite only when a changed file
belongs to, or is imported (directly, transitively or by tests, in any of the
graph_configs) by, a package of that version. Pushes, pull requests into main
and manual runs run every suite. When in doubt the selector runs every suite.

Commands:
  check                       every package is classified
  packages --version V        list the packages of version V (for go test)
  select --event E --base B --changed FILE [--output PATH]
                              FILE holds NUL- or newline-separated paths
                              (git diff --no-renames --name-only -z); writes
                              draft1=, v010=, v011= for GitHub outputs
"""

import argparse
import json
import os
import pathlib
import re
import subprocess
import sys
from dataclasses import dataclass, field

ROOT = pathlib.Path(__file__).resolve().parents[2]
SUITES = {"draft.1": "draft1", "0.10": "v010", "0.11": "v011"}


@dataclass
class Package:
    path: str  # import path
    dir: str  # directory relative to the repository root, "." for the root
    deps: list = field(default_factory=list)  # import paths, transitive, test imports included


def pattern_regex(pattern):
    """Go-style package patterns: "..." is everything, "x/..." is x and
    below, "*" matches within one path element."""
    if pattern == "...":
        return re.compile(r".*")
    base, sub = (pattern[:-4], True) if pattern.endswith("/...") else (pattern, False)
    rx = "".join("[^/]*" if c == "*" else re.escape(c) for c in base)
    return re.compile(rx + (r"(/.*)?" if sub else "") + r"$")


def relative(path, module):
    if path == module:
        return "."
    if not path.startswith(module + "/"):
        sys.exit(f"package outside the module: {path}")
    return path[len(module) + 1:]


def classify(config, packages, module):
    """Map every import path to its version; exit if one is unclassified."""
    rules = [(version, [pattern_regex(p) for p in patterns]) for version, patterns in config["versions"].items()]
    out = {}
    for p in packages:
        rel = relative(p.path, module)
        for version, regexes in rules:
            if any(rx.match(rel) for rx in regexes):
                out[p.path] = version
                break
        else:
            sys.exit(f"package {p.path} has no spec version in .github/ci/versions.json")
    return out


def owning_package(filename, by_dir):
    """The package whose directory is the nearest proper ancestor of a
    non-Go file (testdata, embedded files), or None. The root package owns
    only the files directly in the root directory."""
    parts = pathlib.PurePosixPath(filename).parts
    for i in range(len(parts) - 1, -1, -1):
        d = "/".join(parts[:i]) or "."
        if d in by_dir and (d != "." or i == len(parts) - 1):
            return by_dir[d]
    return None


def select(config, packages, module, files, event, base):
    every = {name: True for name in SUITES.values()}
    if event != "pull_request" or base == "main":
        return every
    if any(f in config.get("all_suites_when_changed", []) for f in files):
        return every
    versions = classify(config, packages, module)
    by_dir = {p.dir: p for p in packages}
    free = [pattern_regex(p) for p in config.get("no_root_effect", [])]
    changed = set()
    for f in files:
        if any(rx.match(f) for rx in free):
            continue
        parts = pathlib.PurePosixPath(f).parts
        if f.endswith(".go"):
            # A Go file belongs to the package of its own directory; a Go
            # file anywhere else is a new or removed package.
            d = "/".join(parts[:-1]) or "."
            if d not in by_dir:
                return every
            changed.add(by_dir[d].path)
            continue
        p = owning_package(f, by_dir)
        if p is None:
            return every  # scripts, configuration, other workflows: run all
        changed.add(p.path)
    affected = set(changed)
    for p in packages:
        if changed.intersection(p.deps):
            affected.add(p.path)
    out = {name: False for name in SUITES.values()}
    out["draft1"] = True
    for path in affected:
        out[SUITES[versions[path]]] = True
    return out


def path_of_test(path):
    """The package an external test package belongs to."""
    return path[: -len("_test")] if path.endswith("_test") else path


def parse_list(lines, module, merged, root=ROOT):
    """Merge `go list -test -f` lines into merged, keyed by import path."""
    for line in lines:
        path, directory, deps = (line.split("\t") + ["", ""])[:3]
        path = path.split(" ", 1)[0]  # "p [p.test]" is p with its test imports
        if path.endswith(".test"):
            continue  # the generated test main
        # An external test package (package p_test, import path
        # "p_test [p.test]") is part of p: its imports count as p's.
        path = path_of_test(path)
        rel = os.path.relpath(directory, root).replace(os.sep, "/")
        p = merged.setdefault(path, Package(path, rel, []))
        found = {path_of_test(d.split(" ", 1)[0]) for d in deps.split() if d.startswith(module)}
        p.deps = sorted(set(p.deps) | found - {path})
    return merged


def load_packages(module, configs):
    """The union of the package graphs of every build configuration."""
    fmt = "{{.ImportPath}}\t{{.Dir}}\t{{join .Deps \" \"}}"
    merged = {}
    for cfg in configs:
        env = dict(os.environ, GOOS=cfg["GOOS"], CGO_ENABLED="0", GOFLAGS="-mod=readonly")
        cmd = ["go", "list", "-test", "-f", fmt]
        if cfg.get("tags"):
            cmd += ["-tags", cfg["tags"]]
        lines = subprocess.check_output(cmd + ["./..."], cwd=ROOT, text=True, env=env).splitlines()
        parse_list(lines, module, merged)
    return list(merged.values())


def read_changed(path):
    data = pathlib.Path(path).read_bytes().decode("utf-8", "surrogateescape")
    sep = "\0" if "\0" in data else "\n"
    return [f for f in (x.strip("\n") for x in data.split(sep)) if f]


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("check")
    pk = sub.add_parser("packages")
    pk.add_argument("--version", required=True, choices=list(SUITES))
    se = sub.add_parser("select")
    se.add_argument("--event", required=True)
    se.add_argument("--base", default="")
    se.add_argument("--changed", required=True)
    se.add_argument("--output")
    args = ap.parse_args()
    config = json.loads((ROOT / ".github/ci/versions.json").read_text())
    module = subprocess.check_output(["go", "list", "-m"], cwd=ROOT, text=True).strip()
    packages = load_packages(module, config["graph_configs"])
    versions = classify(config, packages, module)
    if args.cmd == "check":
        counts = {v: sum(1 for x in versions.values() if x == v) for v in config["versions"]}
        print(json.dumps(counts))
    elif args.cmd == "packages":
        print("\n".join(sorted(p for p, v in versions.items() if v == args.version)))
    else:
        result = select(config, packages, module, read_changed(args.changed), args.event, args.base)
        text = "".join(f"{k}={'true' if v else 'false'}\n" for k, v in result.items())
        print(text, end="")
        if args.output:
            with open(args.output, "a") as f:
                f.write(text)


if __name__ == "__main__":
    main()
