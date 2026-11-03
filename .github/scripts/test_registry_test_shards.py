"""Scenario tests for CI evidence partitioning; no Registry execution fixture."""

from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import registry_test_shards as m


ROOTS = [
    "TestProviderHistoryCollectionNormal", "TestProviderHistoryRecoveryNormal",
    "TestProviderHistoryNormal", "TestProviderNormal", "ExampleNormal",
    m.RECOVERY_ROOT,
]


def stream(rows):
    return b"".join((json.dumps(row) + "\n").encode() for row in rows)


def discovery(roots=ROOTS):
    return stream([{"Action": "output", "Package": m.PACKAGE, "Output": name + "\n"}
                   for name in roots] + [{"Action": "pass", "Package": m.PACKAGE}])


def results(roots, child=None):
    rows = []
    for name in roots:
        rows.append({"Action": "run", "Package": m.PACKAGE, "Test": name})
        if child:
            rows += [{"Action": action, "Package": m.PACKAGE, "Test": name + child}
                     for action in ("run", "pass")]
        rows.append({"Action": "pass", "Package": m.PACKAGE, "Test": name})
    return stream(rows + [{"Action": "pass", "Package": m.PACKAGE}])


class EvidenceFixture(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.current = {
            "context": dict(zip(m.ENV_NAMES, ["merge", "123", "2", "consumer", "refs/pull/1/merge", "owner/repo"])),
            "revision": b"merge\n", "tree": b"tree\n", "compiler": b"go version fixture\n",
            "source": (m.digest(b"package registry011\n") + "  pkg/agent/registry011/unit.go\n").encode(),
        }

    def fixture(self, role="source", roots=ROOTS):
        directory = self.root / role
        directory.mkdir()
        prefix = "registry-source-shard-" if role == "source" else "provider-transport-shard-"
        job = "registry_source_shard" if role == "source" else "provider-capture-shard"
        for index in range(m.SHARDS):
            folder = directory / (prefix + str(index))
            folder.mkdir()
            context = dict(self.current["context"], GITHUB_JOB=job)
            (folder / "context.json").write_bytes(m.encoded(context))
            for filename, key in (("revision.txt", "revision"), ("tree.txt", "tree"), ("compiler.txt", "compiler"), ("source.sha256", "source")):
                (folder / filename).write_bytes(self.current[key])
            (folder / "source").mkdir()
            (folder / "source/unit.go").write_bytes(b"package registry011\n")
            raw = discovery(roots)
            value = m.plan(raw, role, index)
            (folder / "test-discovery.jsonl").write_bytes(raw)
            (folder / "test-plan.json").write_bytes(m.encoded(value))
            (folder / "run-pattern.txt").write_text(value["run_pattern"] + "\n")
            (folder / "tests.jsonl").write_bytes(results(value["selected_roots"]))
            (folder / "coverage.out").write_text(f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 {index}\n{m.PACKAGE}/unit.go:3.1,4.1 2 0\n")
            if index == 0:
                (folder / "cpu-profile.exit").write_bytes(b"0\n")
                (folder / "regression-controls.jsonl").write_bytes(results(m.CONTROLS))
                (folder / "cpu-profile-tests.jsonl").write_bytes(results([m.CPU_ROOT], "/32"))
                for name in ("registry-cpu.test", "cpu-profile.pprof"):
                    (folder / name).write_bytes(name.encode())
                (folder / "cpu-profile.sha256").write_text("".join(m.digest(name.encode()) + "  /fixture/" + name + "\n" for name in ("registry-cpu.test", "cpu-profile.pprof")))
                (folder / "source/nested").mkdir()
                (folder / "source/nested").rmdir()
        return directory

    def test_complete_source_and_provider_partitions_preserve_every_selected_root(self):
        for role in ("source", "provider"):
            with self.subTest(role=role):
                directory = self.fixture(role)
                output = self.root / (role + "-merged")
                expected = m.assemble(directory, output, role, self.current)
                actual = m.verify(output, role, self.current)
                self.assertEqual(actual, expected)
                roots = sorted(set(ROOTS) if role == "source" else set(ROOTS) - {"ExampleNormal"})
                self.assertEqual(actual["all_roots"], roots)
                self.assertEqual(actual["package_passes"], 5)
                self.assertIn(b" 1 10\n", (output / "coverage.out").read_bytes())
                self.assertTrue((output / "shards").is_dir())
                self.assertTrue((output / "source/unit.go").is_file())

    def test_maximum_recovery_has_its_own_complete_process_partition(self):
        neighbors = [m.RECOVERY_ROOT + "Extra", "TestProviderHistoryRecoveryOther"]
        for role in ("source", "provider"):
            with self.subTest(role=role):
                raw = discovery(ROOTS + neighbors)
                isolated = m.plan(raw, role, 4)
                self.assertEqual(isolated["selected_roots"], [m.RECOVERY_ROOT])
                self.assertEqual(isolated["shards"], 5)
                self.assertEqual(isolated["format"], "sage-registry-test-partition-v2")
                self.assertEqual(isolated["run_pattern"], "^(" + m.RECOVERY_ROOT + ")$")
                ordinary = m.plan(raw, role, 1)["selected_roots"]
                self.assertNotIn(m.RECOVERY_ROOT, ordinary)
                self.assertTrue(set(neighbors).issubset(ordinary))
                self.assertEqual(m.validate_tests(results([m.RECOVERY_ROOT], "/settled-maximum"),
                                                  isolated["selected_roots"]), 2)
        with self.assertRaises(ValueError):
            m.plan(discovery([name for name in ROOTS if name != m.RECOVERY_ROOT]), "source", 4)

    def test_unicode_compiled_roots_remain_in_the_complete_partition(self):
        unicode_tests = [
            "TestProviderΩ", "TestProviderHistoryCollectionΔ",
            "TestProviderHistoryRecovery復元", "TestProviderHistoryΕ",
        ]
        all_roots = ROOTS + unicode_tests + ["ExampleΩ", "Fuzz世界"]
        for role in ("source", "provider"):
            with self.subTest(role=role):
                expected = sorted(all_roots if role == "source" else
                                  [name for name in ROOTS if name != "ExampleNormal"] + unicode_tests)
                raw = discovery(all_roots)
                self.assertEqual(m.discovery(raw, role), expected)
                observed = []
                for index in range(m.SHARDS):
                    value = m.plan(raw, role, index)
                    observed.extend(value["selected_roots"])
                    for name in value["selected_roots"]:
                        self.assertRegex(name, value["run_pattern"])
                    self.assertEqual(m.validate_tests(results(value["selected_roots"], "/正常"),
                                                      value["selected_roots"]),
                                     2 * len(value["selected_roots"]))
                self.assertEqual(sorted(observed), expected)
                self.assertEqual(len(observed), len(set(observed)))
                directory = self.fixture(role, all_roots)
                output = self.root / (role + "-unicode-merged")
                value = m.assemble(directory, output, role, self.current)
                self.assertEqual(value["all_roots"], expected)
                self.assertEqual(m.verify(output, role, self.current), value)

    def test_incomplete_failed_or_replaced_evidence_is_never_aggregated(self):
        cases = [
            ("missing-shard", lambda d, f: (d / "registry-source-shard-3").rename(d / "wrong-shard")),
            ("wrong-context", lambda d, f: (f / "context.json").write_bytes(m.encoded(dict(self.current["context"], GITHUB_JOB="registry_source_shard", GITHUB_SHA="other")))),
            ("wrong-attempt", lambda d, f: (f / "context.json").write_bytes(m.encoded(dict(self.current["context"], GITHUB_JOB="registry_source_shard", GITHUB_RUN_ATTEMPT="3")))),
            ("wrong-job", lambda d, f: (f / "context.json").write_bytes(m.encoded(self.current["context"]))),
            ("wrong-revision", lambda d, f: (f / "revision.txt").write_bytes(b"other\n")),
            ("wrong-tree", lambda d, f: (f / "tree.txt").write_bytes(b"other\n")),
            ("wrong-compiler", lambda d, f: (f / "compiler.txt").write_bytes(b"other\n")),
            ("wrong-source", lambda d, f: (f / "source.sha256").write_bytes(b"other\n")),
            ("changed-copy", lambda d, f: (f / "source/unit.go").write_bytes(b"other")),
            ("missing-copy", lambda d, f: (f / "source/unit.go").unlink()),
            ("extra-copy", lambda d, f: (f / "source/extra.go").write_bytes(b"extra")),
            ("changed-plan", lambda d, f: (f / "test-plan.json").write_bytes(b"{}\n")),
            ("changed-selector", lambda d, f: (f / "run-pattern.txt").write_text(".*\n")),
            ("discovery-disagreement", lambda d, f: (f / "test-discovery.jsonl").write_bytes(discovery(ROOTS + ["TestExtra"]))),
            ("failed-package", lambda d, f: (f / "tests.jsonl").write_bytes(stream([{"Package": m.PACKAGE, "Action": "fail"}]))),
            ("missing-roots", lambda d, f: (f / "tests.jsonl").write_bytes(results([]))),
            ("wrong-package", lambda d, f: (f / "tests.jsonl").write_bytes(stream([{"Package": "other", "Action": "pass"}]))),
            ("missing-coverage", lambda d, f: (f / "coverage.out").write_bytes(b"mode: atomic\n")),
            ("coverage-shape", lambda d, f: (f / "coverage.out").write_text(f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 2 0\n{m.PACKAGE}/unit.go:3.1,4.1 2 0\n")),
            ("coverage-inventory", lambda d, f: (f / "coverage.out").write_text(f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 0\n")),
        ]
        for label, mutate in cases:
            with self.subTest(case=label):
                directory = self.fixture()
                mutate(directory, directory / "registry-source-shard-1")
                output = self.root / "refused"
                with self.assertRaises((ValueError, FileNotFoundError)):
                    m.assemble(directory, output, "source", self.current)
                self.assertFalse(output.exists())
                import shutil
                shutil.rmtree(directory)

    def test_maximum_race_and_original_regression_controls_are_required(self):
        for filename, raw in [
            ("cpu-profile.exit", b"1\n"),
            ("regression-controls.jsonl", results(m.CONTROLS[:-1])),
            ("cpu-profile-tests.jsonl", results([m.CPU_ROOT])),
            ("cpu-profile-tests.jsonl", results([m.CPU_ROOT], "/31")),
            ("cpu-profile.sha256", "".encode()),
            ("registry-cpu.test", b"changed"),
        ]:
            with self.subTest(file=filename, raw=raw[:20]):
                directory = self.fixture()
                (directory / "registry-source-shard-0" / filename).write_bytes(raw)
                with self.assertRaises(ValueError):
                    m.inspect_shards(directory, "source", self.current)
                import shutil
                shutil.rmtree(directory)

    def test_verifier_reopens_children_and_recomputes_merged_evidence(self):
        for file in ("tests.jsonl", "coverage.out", "test-partition.json", "revision.txt", "context.json", "shards/registry-source-shard-0/tests.jsonl"):
            with self.subTest(file=file):
                directory = self.fixture()
                output = self.root / "merged"
                m.assemble(directory, output, "source", self.current)
                (output / file).write_bytes(b"{}\n")
                with self.assertRaises((ValueError, KeyError)):
                    m.verify(output, "source", self.current)
                import shutil
                shutil.rmtree(directory)
                shutil.rmtree(output)

    def test_discovery_and_root_results_reject_omissions_duplicates_and_incomplete_runs(self):
        bad_discovery = [discovery([]), discovery(ROOTS + ROOTS[:1]), stream([{"Package": m.PACKAGE, "Action": "fail"}]), stream([{"Package": m.PACKAGE, "Action": "run", "Test": "TestExtra"}]), stream([{"Package": "other", "Action": "pass"}])]
        for raw in bad_discovery:
            with self.subTest(discovery=raw[:80]), self.assertRaises(ValueError):
                m.discovery(raw, "source")
        for index in (-1, 5, True):
            with self.subTest(index=index), self.assertRaises(ValueError):
                m.plan(discovery(), "source", index)
        with self.assertRaises(ValueError):
            m.plan(discovery(["TestOnly"]), "source", 0)
        with self.assertRaises(ValueError):
            m.plan(discovery(["TestProviderHistoryCollection" + "x" * 100001]), "source", 0)
        for raw in [results(["TestOther"]), results(["TestOne"]) + results(["TestOne"]), stream([{"Package": m.PACKAGE, "Action": "pass", "Test": "TestOne"}]), stream([{"Package": m.PACKAGE, "Action": "run", "Test": "TestOne"}, {"Package": m.PACKAGE, "Action": "pass"}])]:
            with self.subTest(results=raw[:80]), self.assertRaises(ValueError):
                m.validate_tests(raw, ["TestOne"])
        skipped = stream([{"Package": m.PACKAGE, "Action": "run", "Test": "TestOne"}, {"Package": m.PACKAGE, "Action": "skip", "Test": "TestOne"}, {"Package": m.PACKAGE, "Action": "pass"}])
        self.assertEqual(m.validate_tests(skipped, ["TestOne"]), 1)

    def test_context_and_coverage_domains_remain_closed(self):
        for context in [dict(self.current["context"], GITHUB_RUN_ATTEMPT="01"), dict(self.current["context"], GITHUB_RUN_ATTEMPT="0"), dict(self.current["context"], extra="unexpected"), dict(self.current["context"], GITHUB_JOB="")]:
            with self.subTest(context=context), self.assertRaises(ValueError):
                m.check_context(m.encoded(context), self.current, "consumer")
        with self.assertRaises(ValueError):
            m.record('{"x":1,"x":2}')
        for raw in [b"mode: set\n", b"mode: atomic\nother/file.go:1.1,2.1 1 0\n", f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 -1\n".encode(), f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 {1 << 63}\n".encode(), (f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 0\n" + f"{m.PACKAGE}/unit.go:1.1,2.1 1 0\n").encode()]:
            with self.subTest(coverage=raw), self.assertRaises(ValueError):
                m.coverage(raw)
        with self.assertRaises(ValueError):
            m.discovery(discovery(), "other")

    def test_individually_valid_coverage_counters_cannot_overflow_when_merged(self):
        directory = self.fixture()
        for index, count in [(0, (1 << 63) - 1), (1, 1)]:
            path = directory / ("registry-source-shard-" + str(index)) / "coverage.out"
            path.write_text(f"mode: atomic\n{m.PACKAGE}/unit.go:1.1,2.1 1 {count}\n{m.PACKAGE}/unit.go:3.1,4.1 2 0\n")
        with self.assertRaises(ValueError):
            m.inspect_shards(directory, "source", self.current)

    def test_cli_builds_plans_and_assembles_and_verifies_only_explicit_roles(self):
        directory = self.fixture()
        first = directory / "registry-source-shard-0"
        m.main(["plan", "source", str(first), "--index", "0"])
        output = self.root / "merged"
        with patch.object(m, "identity", return_value=self.current), redirect_stdout(io.StringIO()):
            self.assertEqual(m.main(["assemble", "source", str(directory), "--output", str(output)]), m.main(["verify", "source", str(output)]))
            for args in [["plan", "source", str(first)], ["plan", "source", str(first), "--index", "0", "--output", str(output)], ["assemble", "source", str(directory)], ["verify", "source", str(output), "--output", str(output)], ["verify", "source", str(output), "--index", "0"]]:
                with self.subTest(args=args), self.assertRaises(ValueError):
                    m.main(args)
        with self.assertRaises(ValueError):
            m.assemble(directory, output, "source", self.current)

    def test_identity_pins_the_tracked_source_and_current_native_compiler(self):
        path = self.root / "tracked.go"
        path.write_bytes(b"package fixture\n")
        def answer(command):
            if command[:3] == ["git", "ls-files", "-z"]:
                return (str(path) + "\0").encode()
            if command[:2] == ["git", "rev-parse"]:
                return b"revision\n"
            self.assertEqual(command, ["go", "version"])
            return b"go version fixture\n"
        with patch.object(m.subprocess, "check_output", side_effect=answer), patch.dict(os.environ, self.current["context"]):
            for role in ("source", "provider"):
                value = m.identity(role)
                self.assertEqual(value["compiler"], b"go version fixture\n")
                self.assertIn(m.digest(path.read_bytes()).encode(), value["source"])
            with self.assertRaises(ValueError):
                m.identity("other")


if __name__ == "__main__":
    unittest.main()
