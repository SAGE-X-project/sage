#!/usr/bin/env python3
"""Generate docs/INDEX.md from the Markdown files in the repository.

The index is derived, not hand-maintained: every Markdown file under the
scanned roots is listed once, grouped by area, with its H1 as title and the
freshness verdict recorded in docs/refactoring/DOCS_GRAPH.md (section 4) when
one exists. Run `make docs-index` after adding or moving documentation; the
Lint job fails when the committed index differs from the generated one.
"""
from __future__ import annotations

import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
INDEX = ROOT / "docs" / "INDEX.md"
GRAPH = ROOT / "docs" / "refactoring" / "DOCS_GRAPH.md"

# Scanned roots (relative to the repository). node_modules and build output
# are skipped; generated code-graph output is skipped.
ROOTS = ["README.md", "INSTALL.md", "CONTRIBUTING.md", "SECURITY.md", "CHANGELOG.md",
         "docs", "api", "examples", "pkg",
         "internal", "deployments", "tools", "sdk"]
SKIP_PARTS = {"node_modules", "artifacts", "cache", "typechain-types", "dist", "target", "reports", "flattened"}
SKIP_FILES = {"docs/INDEX.md", "docs/refactoring/graph/delta.md"}

# (section title, description, predicate on the repo-relative path)
SECTIONS = [
    ("Getting started", "Entry points for new users and contributors.",
     lambda p: p in {"README.md", "INSTALL.md", "CONTRIBUTING.md", "SECURITY.md", "CHANGELOG.md",
                     "docs/BUILD.md", "docs/CODING_GUIDELINES.md", "docs/CI-CD.md", "docs/GO_VERSION_REQUIREMENT.md", "docs/CODE_REVIEW_CHECKLIST.md"}),
    ("Architecture and design", "System design, decisions and rationale.",
     lambda p: p.startswith("docs/adr/") or p.startswith("docs/dev/") or p in {"docs/ARCHITECTURE.md", "docs/API.md", "docs/KME_PUBLIC_KEY_INTEGRATION.md", "docs/AGENTCARD_MIGRATION_GUIDE.md", "docs/QUICKSTART_PR118.md", "docs/SAGE_A2A_INTEGRATION_GUIDE.md", "docs/PERFORMANCE_BENCHMARKS.md"}),
    ("Protocols: handshake, HPKE, RFC 9421", "Wire protocols and their cryptographic design.",
     lambda p: p.startswith("docs/handshake/") or p.startswith("docs/core/") or p.startswith("api/")),
    ("Go packages", "Package-level READMEs under pkg/ and internal/.",
     lambda p: (p.startswith("pkg/") or p.startswith("internal/")) and p.endswith("README.md")
     or p.startswith("docs/crypto/") or p.startswith("docs/did/")),
    ("Command-line tools", "sage-crypto, sage-did and sage-verify.",
     lambda p: p.startswith("docs/cli/")),
    ("Smart contracts", "Contract-facing guides; the contracts themselves live in github.com/SAGE-X-project/sage-contracts.",
     lambda p: p.startswith("docs/contracts/")),
    ("Examples", "Runnable demonstrations. Their READMEs state what each demo checks and what it does not.",
     lambda p: p.startswith("examples/")),
    ("Testing", "Test guides and specification verification records.",
     lambda p: p.startswith("docs/test/")),
    ("Operations", "Deployment configuration, containers, load and benchmark tooling.",
     lambda p: p.startswith("deployments/") or p.startswith("tools/") or p in {"docs/DEPLOYMENT.md", "docs/OPERATIONS.md", "docs/DATABASE.md"}),
    ("Language clients (experimental)", "Not interoperable with the Go core; see DECISIONS.md, Decision 4.",
     lambda p: p.startswith("sdk/")),
    ("Refactoring programme", "Analyses, decisions, backlog and the generated code graph.",
     lambda p: p.startswith("docs/refactoring/")),
    ("Detailed guides (Korean)", "Long-form series; several parts describe superseded APIs (see freshness).",
     lambda p: p.startswith("docs/overview/")),
    ("Planning, audit and maintenance records", "Historical material kept for reference.",
     lambda p: p.startswith("docs/planning/") or p.startswith("docs/audit/") or p.startswith("docs/maintenance/") or p.startswith("docs/archive/")),
]

CLI_TOOLS = [
    ("sage-crypto", "key generation, import/export (JWK, PEM), signing, verification and address derivation"),
    ("sage-did", "agent registration (commit -> register -> activate), resolution, key and A2A card management"),
    ("sage-verify", "health, blockchain and system checks; `deployment` prints the configuration and deployment record for a network and tests the RPC and registry code"),
]


def load_freshness() -> dict[str, tuple[str, str]]:
    """Return path -> (head, freshness) from the DOCS_GRAPH node table."""
    out: dict[str, tuple[str, str]] = {}
    if not GRAPH.exists():
        return out
    row = re.compile(r"^\| `([^`]+)` \| [^|]+ \| ([^|]+) \| [^|]+ \| \*\*([A-Z]+)\*\* \|")
    for line in GRAPH.read_text(encoding="utf-8").splitlines():
        m = row.match(line)
        if m:
            out[m.group(1).strip()] = (m.group(2).strip(), m.group(3).strip())
    return out


def title_of(path: Path) -> str:
    try:
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            if line.startswith("# "):
                t = line[2:].strip()
                t = re.sub(r"[*_`]", "", t)
                return t or path.stem
    except OSError:
        pass
    return path.stem


def scan() -> list[str]:
    files: set[str] = set()
    for root in ROOTS:
        p = ROOT / root
        if p.is_file():
            files.add(root)
            continue
        if not p.is_dir():
            continue
        for dirpath, dirnames, filenames in os.walk(p):
            rel_dir = Path(dirpath).relative_to(ROOT)
            if any(part in SKIP_PARTS for part in rel_dir.parts):
                dirnames[:] = []
                continue
            dirnames[:] = [d for d in dirnames if d not in SKIP_PARTS and not d.startswith(".")]
            for f in filenames:
                if f.endswith(".md"):
                    files.add(str(rel_dir / f))
    return sorted(f for f in files if f not in SKIP_FILES)


def render(files: list[str], fresh: dict[str, tuple[str, str]]) -> str:
    assigned: dict[str, list[str]] = {title: [] for title, _, _ in SECTIONS}
    other: list[str] = []
    for f in files:
        for title, _, pred in SECTIONS:
            if pred(f):
                assigned[title].append(f)
                break
        else:
            other.append(f)

    lines = [
        "# SAGE Documentation Index",
        "",
        "<!-- Generated by tools/scripts/gen-docs-index.py. Do not edit by hand: run `make docs-index`. -->",
        "",
        "Every Markdown document in the repository, grouped by area. **Freshness** is the",
        "verdict recorded in [`docs/refactoring/DOCS_GRAPH.md`](refactoring/DOCS_GRAPH.md)",
        "(section 4) on 2026-09-11: CURRENT matches the code, MIXED is partly outdated,",
        "STALE describes superseded behaviour; blank means the document was not assessed.",
        "The refactoring backlog ([`BACKLOG.md`](refactoring/BACKLOG.md), items E-04 to",
        "E-07) tracks the rewrite or retirement of MIXED and STALE documents.",
        "",
    ]
    for title, desc, _ in SECTIONS:
        items = assigned[title]
        if not items and title != "Command-line tools":
            continue
        lines += [f"## {title}", "", desc, ""]
        if title == "Command-line tools":
            lines += ["| Binary | Purpose |", "|---|---|"]
            for name, purpose in CLI_TOOLS:
                lines.append(f"| `{name}` (`cmd/{name}`) | {purpose} |")
            lines.append("")
            if items:
                lines += ["Guides:", ""]
        if items:
            lines += ["| Document | Title | Freshness |", "|---|---|---|"]
            for f in items:
                rel = os.path.relpath(ROOT / f, INDEX.parent).replace(os.sep, "/")
                head, verdict = fresh.get(f, ("", ""))
                t = title_of(ROOT / f)
                if head and head.lower() != t.lower():
                    t = f"{t} ({head})"
                lines.append(f"| [`{f}`]({rel}) | {t} | {verdict} |")
            lines.append("")
    if other:
        lines += ["## Other", "", "| Document | Title | Freshness |", "|---|---|---|"]
        for f in other:
            rel = os.path.relpath(ROOT / f, INDEX.parent).replace(os.sep, "/")
            head, verdict = fresh.get(f, ("", ""))
            lines.append(f"| [`{f}`]({rel}) | {title_of(ROOT / f)} | {verdict} |")
        lines.append("")
    return "\n".join(lines).rstrip("\n") + "\n"


def main() -> int:
    content = render(scan(), load_freshness())
    if "--check" in sys.argv:
        current = INDEX.read_text(encoding="utf-8") if INDEX.exists() else ""
        if current != content:
            print("docs/INDEX.md is out of date: run `make docs-index` and commit the result", file=sys.stderr)
            return 1
        print("docs/INDEX.md is up to date")
        return 0
    INDEX.write_text(content, encoding="utf-8")
    print(f"wrote {INDEX.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
