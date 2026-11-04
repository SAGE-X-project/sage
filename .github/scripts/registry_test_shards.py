"""Retain and verify a complete, disjoint Registry test partition in CI."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess


PACKAGE = "github.com/sage-x-project/sage/pkg/agent/registry011"
SOURCE_PREFIX = "pkg/agent/registry011/"
SHARDS = 5
FORMAT = "sage-registry-test-partition-v2"
ROOT = re.compile(r"(?:Test|Example|Fuzz)\w*\Z")
PROVIDER_PREFIXES = (
    "TestProvider", "TestNode", "TestReplay", "TestRootGas", "TestErigon",
    "TestNethermind", "TestGasProfile", "TestStateGas",
)
ENV_NAMES = (
    "GITHUB_SHA", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_JOB",
    "GITHUB_REF", "GITHUB_REPOSITORY",
)
CONTROLS = (
    "TestProviderHistoryMemberReplayCopyOwnsBehaviorOutsideSemanticRoot",
    "TestProviderHistoryNativeBlockEncodingPreservesCompleteRecordedBytes",
    "TestProviderHistoryNativeBlockEncodingKeepsFailedOriginalAllowance",
    "TestProviderHistoryResultsBindExactOriginalsToEveryOrdinal",
    "TestProviderHistoryResultPackRevalidatesContextOnEveryCall",
    "TestProviderHistoryResultPackKeepsLastMemberBindings",
    "TestProviderHistoryProgramKeepsRecoveryAndSelectionBindings",
    "TestProviderHistoryProgramRefusesReplacementOriginalAndSelectedInputs",
    "TestProviderBufferedHeadKeepsBytesEventsAndReaderPosition",
    "TestProviderBufferedHeadKeepsExactInsufficientWorkPrefix",
    "TestProviderBufferedHeadRechecksOwnerOnEveryCall",
    "TestProviderHistoryCollectionRecordedManifestKeepsJournalIdentity",
    "TestProviderHistoryRecoveryUnusedSealsRejectUnexpectedState",
)
RECOVERY_ROOT = "TestProviderHistoryRecoveryGroupReconstructsConsumedSuccessfulAndFailedPrefixesTwice"
CPU_ROOT = "TestProviderHistoryMemberDispatchRetainsOneSeveralAndMaximumOrdinals"
FINAL_REFUSAL_ROOT = "TestProviderHistoryFinalReaderRefusesUnfundedReadWithoutExecutionBorrow"


def require(condition, message):
    if not condition:
        raise ValueError(message)


def encoded(value):
    return (json.dumps(value, sort_keys=True, indent=2) + "\n").encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def record(raw):
    def pairs(items):
        value = {}
        for key, item in items:
            require(key not in value, "duplicate JSON object member")
            value[key] = item
        return value
    return json.loads(raw, object_pairs_hook=pairs)


def identity(role):
    require(role in ("source", "provider"), "unknown evidence role")
    paths = [
        "pkg/agent/registry011", "go.mod", "go.sum",
        ".github/workflows/test.yml", ".github/workflows/registry-node-evidence.yml",
        "docs/dev/registry-source-checks.md",
        ".github/scripts/registry_test_shards.py",
        ".github/scripts/test_registry_test_shards.py",
    ]
    if role == "source":
        paths += ["docs/dev/agent-embedding.md", "docs/INDEX.md", ".gitleaks.toml"]
    tracked = subprocess.check_output(["git", "ls-files", "-z", *paths]).decode().split("\0")[:-1]
    require(tracked and len(tracked) == len(set(tracked)), "invalid source inventory")
    return {
        "context": {name: os.environ[name] for name in ENV_NAMES},
        "revision": subprocess.check_output(["git", "rev-parse", "HEAD"]),
        "tree": subprocess.check_output(["git", "rev-parse", "HEAD^{tree}"]),
        "compiler": subprocess.check_output(["go", "version"]),
        "source": "".join(f"{digest(Path(name).read_bytes())}  {name}\n" for name in tracked).encode(),
    }


def events(raw):
    return [record(line) for line in raw.decode().splitlines() if line.strip()]


def check_context(raw, current, job):
    context = record(raw)
    require(set(context) == set(ENV_NAMES) and all(isinstance(v, str) and v for v in context.values()), "invalid context schema")
    for name in ("GITHUB_SHA", "GITHUB_RUN_ID", "GITHUB_REF", "GITHUB_REPOSITORY"):
        require(context[name] == current["context"][name], "evidence context mismatch: " + name)
    attempt = context["GITHUB_RUN_ATTEMPT"]
    require(attempt.isdecimal() and str(int(attempt)) == attempt and context["GITHUB_JOB"] == job and
            1 <= int(attempt) <= int(current["context"]["GITHUB_RUN_ATTEMPT"]), "evidence job or attempt mismatch")
    return context


def discovery(raw, role):
    require(role in ("source", "provider"), "unknown evidence role")
    roots = []
    completed = 0
    for event in events(raw):
        require(event.get("Package") == PACKAGE, "discovery package mismatch")
        require(event.get("Action") != "fail", "failed test discovery")
        require(not event.get("Test"), "discovery executed a named test")
        if event.get("Action") == "pass":
            completed += 1
        if event.get("Action") == "output":
            line = event.get("Output", "").strip()
            if ROOT.fullmatch(line):
                roots.append(line)
    require(completed == 1 and roots and len(roots) == len(set(roots)), "incomplete or duplicate discovery")
    selected = sorted(name for name in roots if role == "source" or name.startswith(PROVIDER_PREFIXES))
    require(selected, "empty selected package test set")
    return selected


def shard_number(name):
    if name == RECOVERY_ROOT:
        return 4
    if name.startswith("TestProviderHistoryCollection"):
        return 0
    if name.startswith("TestProviderHistoryRecovery"):
        return 1
    if name == FINAL_REFUSAL_ROOT:
        # Keep this complete owned readback fixture off the long-running
        # ordinary history worker. No child, observer or quota is shared.
        return 1
    if name.startswith("TestProviderHistoryFinal"):
        # Keep each original-read scenario whole, and leave the isolated
        # maximum-recovery process on worker 4. Full-hash modulo four is
        # the low two bits of its final byte, independent of discovery order.
        return hashlib.sha256(name.encode("utf-8")).digest()[-1] % (SHARDS - 1)
    if name.startswith("TestProviderHistory"):
        # Spread the remaining history roots without depending on discovery
        # order, Python's randomized hash seed or timing from an earlier run.
        return 2 + (hashlib.sha256(name.encode("utf-8")).digest()[0] & 1)
    return 3


def plan(raw, role, index):
    require(type(index) is int and 0 <= index < SHARDS, "invalid shard index")
    roots = discovery(raw, role)
    selected = [name for name in roots if shard_number(name) == index]
    require(selected, "empty test partition")
    pattern = "^(" + "|".join(re.escape(name) for name in selected) + ")$"
    require(len(pattern.encode()) <= 100000, "test selector exceeds argument bound")
    return {"format": FORMAT, "role": role, "index": index, "shards": SHARDS,
            "all_roots": roots, "selected_roots": selected,
            "discovery_sha256": digest(raw), "run_pattern": pattern}


def validate_tests(raw, selected):
    started, ended = set(), set()
    package_passes = 0
    for event in events(raw):
        require(event.get("Package") == PACKAGE, "test package mismatch")
        action, name = event.get("Action"), event.get("Test")
        require(action != "fail", "partition contains a failed test or package")
        if name:
            require(name.split("/", 1)[0] in selected, "test outside selected partition")
            if action == "run":
                require(name not in started, "repeated named test execution")
                started.add(name)
            if action in ("pass", "skip"):
                require(name in started and name not in ended, "unmatched or duplicate named result")
                ended.add(name)
        elif action == "pass":
            package_passes += 1
    roots = {name for name in started if "/" not in name}
    require(package_passes == 1 and started == ended and roots == set(selected), "incomplete test partition")
    return len(started)


def coverage(raw):
    lines = raw.decode().splitlines()
    require(len(lines) > 1 and lines[0] == "mode: atomic", "missing atomic coverage")
    out = {}
    for line in lines[1:]:
        location, statements, count = line.split()
        filename, span = location.rsplit(":", 1)
        require(filename.startswith(PACKAGE + "/") and re.fullmatch(r"\d+\.\d+,\d+\.\d+", span), "invalid coverage location")
        require(location not in out and statements.isdecimal() and count.isdecimal(), "duplicate or invalid coverage block")
        statements, count = int(statements), int(count)
        require(statements >= 0 and 0 <= count < 1 << 63, "coverage counter outside bound")
        out[location] = (statements, count)
    return out


def check_controls(folder):
    require((folder / "cpu-profile.exit").read_bytes() == b"0\n", "original maximum race control failed")
    validate_tests((folder / "regression-controls.jsonl").read_bytes(), CONTROLS)
    raw = (folder / "cpu-profile-tests.jsonl").read_bytes()
    require(validate_tests(raw, [CPU_ROOT]) == 2, "maximum race control omitted its selected child")
    selected = [e for e in events(raw) if e.get("Test") == CPU_ROOT + "/32"]
    require(sum(e["Action"] == "run" for e in selected) == 1 and sum(e["Action"] == "pass" for e in selected) == 1,
            "maximum race control did not pass ordinal 32")
    expected = {"registry-cpu.test", "cpu-profile.pprof"}
    seen = set()
    for line in (folder / "cpu-profile.sha256").read_text().splitlines():
        want, filename = line.split("  ", 1)
        name = Path(filename).name
        require(name in expected and name not in seen and digest((folder / name).read_bytes()) == want, "CPU diagnostic digest mismatch")
        seen.add(name)
    require(seen == expected, "incomplete CPU diagnostic identity")


def check_source_copy(folder, source):
    expected = {}
    for line in source.decode().splitlines():
        want, name = line.split("  ", 1)
        if name.startswith(SOURCE_PREFIX):
            relative = name[len(SOURCE_PREFIX):]
            if relative.endswith((".go", ".json")) or relative.startswith("testdata/"):
                expected[relative] = want
    root = folder / "source"
    require(root.is_dir() and not root.is_symlink(), "invalid source snapshot directory")
    files = [p for p in root.rglob("*") if p.is_file()]
    require(expected and {str(p.relative_to(root)) for p in files} == set(expected), "incomplete source snapshot")
    require(not any(p.is_symlink() for p in root.rglob("*")), "source snapshot contains a link")
    for name, want in expected.items():
        require(digest((root / name).read_bytes()) == want, "source snapshot bytes mismatch")


def inspect_shards(directory, role, current):
    require(role in ("source", "provider"), "unknown evidence role")
    prefix = "registry-source-shard-" if role == "source" else "provider-transport-shard-"
    require({p.name for p in directory.iterdir()} == {prefix + str(i) for i in range(SHARDS)}, "missing or extra shard artifact")
    all_roots, observed, merged, logs, records = None, set(), None, [], []
    job = "registry_source_shard" if role == "source" else "provider-capture-shard"
    for index in range(SHARDS):
        folder = directory / (prefix + str(index))
        require(folder.is_dir() and not folder.is_symlink(), "invalid shard directory")
        context = check_context((folder / "context.json").read_bytes(), current, job)
        for filename, key in (("revision.txt", "revision"), ("tree.txt", "tree"), ("compiler.txt", "compiler"), ("source.sha256", "source")):
            require((folder / filename).read_bytes() == current[key], "shard identity mismatch: " + filename)
        check_source_copy(folder, current["source"])
        if index == 0:
            check_controls(folder)
        raw_discovery = (folder / "test-discovery.jsonl").read_bytes()
        expected = plan(raw_discovery, role, index)
        require((folder / "test-plan.json").read_bytes() == encoded(expected) and (folder / "run-pattern.txt").read_text() == expected["run_pattern"] + "\n", "changed test plan")
        if all_roots is None:
            all_roots = expected["all_roots"]
        require(all_roots == expected["all_roots"] and not observed.intersection(expected["selected_roots"]), "test discovery disagreement or overlapping partitions")
        observed.update(expected["selected_roots"])
        raw_tests = (folder / "tests.jsonl").read_bytes()
        named = validate_tests(raw_tests, expected["selected_roots"])
        rows = coverage((folder / "coverage.out").read_bytes())
        source_names = {line.split("  ", 1)[1] for line in current["source"].decode().splitlines()}
        for location in rows:
            filename = location.rsplit(":", 1)[0]
            relative = filename[len(PACKAGE) + 1:]
            require(SOURCE_PREFIX + relative in source_names and "/" not in relative and not relative.endswith("_test.go"), "coverage names untracked or non-production source")
        if merged is None:
            merged = rows
        else:
            require(rows.keys() == merged.keys(), "coverage block inventory mismatch")
            for location, (statements, count) in rows.items():
                old_statements, old_count = merged[location]
                require(statements == old_statements and count + old_count < 1 << 63, "coverage shape mismatch or counter overflow")
                merged[location] = (statements, count + old_count)
        logs.append(raw_tests)
        records.append({"index": index, "context": context, "roots": expected["selected_roots"],
                        "named_runs": named, "tests_sha256": digest(raw_tests),
                        "coverage_sha256": digest((folder / "coverage.out").read_bytes()),
                        "discovery_sha256": digest(raw_discovery)})
    require(observed == set(all_roots), "partition does not cover the complete selected package")
    merged_coverage = ("mode: atomic\n" + "".join(f"{loc} {st} {count}\n" for loc, (st, count) in sorted(merged.items()))).encode()
    joined_tests = b"".join(log if log.endswith(b"\n") else log + b"\n" for log in logs)
    receipt = {"format": FORMAT, "role": role, "shards": SHARDS, "package_passes": SHARDS,
               "all_roots": all_roots, "records": records,
               "tests_sha256": digest(joined_tests), "coverage_sha256": digest(merged_coverage)}
    return receipt, joined_tests, merged_coverage


def assemble(directory, output, role, current):
    receipt, tests, counts = inspect_shards(directory, role, current)
    require(not output.exists(), "aggregation output already exists")
    output.mkdir(parents=True)
    shutil.copytree(directory, output / "shards")
    prefix = "registry-source-shard-" if role == "source" else "provider-transport-shard-"
    first = directory / (prefix + "0")
    for path in first.iterdir():
        if path.name not in ("context.json", "tests.jsonl", "coverage.out", "test-plan.json", "test-discovery.jsonl", "run-pattern.txt"):
            if path.is_dir():
                shutil.copytree(path, output / path.name)
            else:
                shutil.copyfile(path, output / path.name)
    context = dict(current["context"])
    context["GITHUB_JOB"] = "registry_source_checks" if role == "source" else "provider-capture"
    (output / "context.json").write_bytes(encoded(context))
    (output / "tests.jsonl").write_bytes(tests)
    (output / "coverage.out").write_bytes(counts)
    (output / "test-partition.json").write_bytes(encoded(receipt))
    return receipt


def verify(output, role, current):
    receipt, tests, counts = inspect_shards(output / "shards", role, current)
    require((output / "tests.jsonl").read_bytes() == tests and (output / "coverage.out").read_bytes() == counts and
            (output / "test-partition.json").read_bytes() == encoded(receipt), "aggregation differs from original shard evidence")
    job = "registry_source_checks" if role == "source" else "provider-capture"
    check_context((output / "context.json").read_bytes(), current, job)
    for filename, key in (("revision.txt", "revision"), ("tree.txt", "tree"), ("compiler.txt", "compiler"), ("source.sha256", "source")):
        require((output / filename).read_bytes() == current[key], "aggregate source identity mismatch")
    return receipt


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("plan", "assemble", "verify"))
    parser.add_argument("role", choices=("source", "provider"))
    parser.add_argument("directory", type=Path)
    parser.add_argument("--index", type=int)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args(argv)
    if args.command == "plan":
        require(args.index is not None and args.output is None, "plan needs only a shard index")
        value = plan((args.directory / "test-discovery.jsonl").read_bytes(), args.role, args.index)
        (args.directory / "test-plan.json").write_bytes(encoded(value))
        (args.directory / "run-pattern.txt").write_text(value["run_pattern"] + "\n")
        return value
    require(args.index is None, "aggregate commands cannot select a shard")
    current = identity(args.role)
    if args.command == "assemble":
        require(args.output is not None, "assemble needs a fresh output directory")
        value = assemble(args.directory, args.output, args.role, current)
    else:
        require(args.output is None, "verify cannot rewrite evidence")
        value = verify(args.directory, args.role, current)
    print(json.dumps(value, sort_keys=True))
    return value


if __name__ == "__main__":
    main()
