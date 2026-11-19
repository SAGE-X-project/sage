"""Tests for ci_versions.py. Run: python3 .github/scripts/test_ci_versions.py"""

import json
import pathlib
import sys
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import ci_versions  # noqa: E402

M = "github.com/sage-x-project/sage"
CONFIG = json.loads((pathlib.Path(__file__).resolve().parents[1] / "ci" / "versions.json").read_text())


def pkg(rel, deps=(), d=None):
    path = M if rel == "." else f"{M}/{rel}"
    return ci_versions.Package(path, d or rel, [f"{M}/{x}" for x in deps])


GRAPH = [
    pkg("pkg/agent/crypto"),
    pkg("pkg/agent/internal/rfc8785"),
    pkg("pkg/agent/did", ["pkg/agent/crypto"]),
    pkg("pkg/agent/hpke", ["pkg/agent/registry010"]),
    pkg("pkg/agent/registry010", ["pkg/agent/crypto"]),
    pkg("examples/registry-media010", ["pkg/agent/did"]),
    pkg("pkg/agent/registry011", ["pkg/agent/internal/rfc8785"]),
    pkg("pkg/agent/registry011/internal/compilermemory"),
    pkg("internal/qualification/nativelimits"),
    pkg("cmd/sage-did", ["pkg/agent/registry011"]),
]


class ClassifyTest(unittest.TestCase):
    def test_every_package_has_one_version(self):
        got = ci_versions.classify(CONFIG, GRAPH, M)
        self.assertEqual(got[f"{M}/pkg/agent/registry011/internal/compilermemory"], "0.11")
        self.assertEqual(got[f"{M}/internal/qualification/nativelimits"], "0.11")
        self.assertEqual(got[f"{M}/examples/registry-media010"], "0.10")
        self.assertEqual(got[f"{M}/pkg/agent/hpke"], "draft.1")
        self.assertEqual(got[f"{M}/cmd/sage-did"], "draft.1")

    def test_patterns_are_anchored(self):
        cfg = {"versions": {"0.11": ["pkg/agent/wire011"], "draft.1": ["..."]}}
        got = ci_versions.classify(cfg, [pkg("pkg/agent/wire011x")], M)
        self.assertEqual(got[f"{M}/pkg/agent/wire011x"], "draft.1")

    def test_unclassified_package_fails(self):
        cfg = {"versions": {"0.11": ["pkg/agent/registry011"]}}
        with self.assertRaises(SystemExit):
            ci_versions.classify(cfg, [pkg("pkg/agent/crypto")], M)


class ParseListTest(unittest.TestCase):
    def test_external_test_packages_merge_into_their_package(self):
        lines = [
            f"{M}/pkg/agent/crypto\t/r/pkg/agent/crypto\t",
            f"{M}/pkg/agent/crypto [{M}/pkg/agent/crypto.test]\t/r/pkg/agent/crypto\t{M}/pkg/agent/did",
            f"{M}/pkg/agent/crypto_test [{M}/pkg/agent/crypto.test]\t/r/pkg/agent/crypto\t{M}/pkg/agent/registry010 [{M}/pkg/agent/crypto.test]",
            f"{M}/pkg/agent/crypto.test\t/r/pkg/agent/crypto\t{M}/pkg/agent/crypto_test [{M}/pkg/agent/crypto.test]",
        ]
        got = ci_versions.parse_list(lines, M, {}, root="/r")
        self.assertEqual(sorted(got), [f"{M}/pkg/agent/crypto"])
        p = got[f"{M}/pkg/agent/crypto"]
        self.assertEqual(p.dir, "pkg/agent/crypto")
        self.assertEqual(p.deps, [f"{M}/pkg/agent/did", f"{M}/pkg/agent/registry010"])


class SelectTest(unittest.TestCase):
    def suites(self, files, event="pull_request", base="spec/0.9.0-draft.1.1"):
        return ci_versions.select(CONFIG, GRAPH, M, files, event, base)

    def test_core_only_runs_draft1_only(self):
        self.assertEqual(self.suites(["core/did/did.go", ".github/workflows/core.yml", "docs/x.md", "core/go.sum"]),
                         {"draft1": True, "v010": False, "v011": False})

    def test_legacy_change_reaches_dependants(self):
        # crypto -> did -> examples/registry-media010 (0.10), and crypto -> registry010
        self.assertEqual(self.suites(["pkg/agent/crypto/keys.go"]),
                         {"draft1": True, "v010": True, "v011": False})

    def test_shared_rfc8785_reaches_011(self):
        self.assertEqual(self.suites(["pkg/agent/internal/rfc8785/rfc8785_test.go"]),
                         {"draft1": True, "v010": False, "v011": True})

    def test_registry011_change_runs_011(self):
        got = self.suites(["pkg/agent/registry011/replay.go"])
        self.assertTrue(got["v011"])
        self.assertFalse(got["v010"])

    def test_testdata_maps_to_its_package(self):
        self.assertTrue(self.suites(["pkg/agent/registry011/testdata/fixture.json"])["v011"])

    def test_infrastructure_change_runs_everything(self):
        for f in ["go.mod", ".github/ci/versions.json", ".github/workflows/test.yml"]:
            self.assertEqual(self.suites([f]), {"draft1": True, "v010": True, "v011": True}, f)

    def test_unknown_go_directory_fails_safe(self):
        # A removed or new package directory that is not in the graph.
        self.assertEqual(self.suites(["pkg/agent/registry011/newpkg/x.go"]),
                         {"draft1": True, "v010": True, "v011": True})

    def test_main_and_push_run_everything(self):
        every = {"draft1": True, "v010": True, "v011": True}
        self.assertEqual(self.suites(["core/did/did.go"], base="main"), every)
        self.assertEqual(self.suites(["core/did/did.go"], event="push"), every)
        self.assertEqual(self.suites(["core/did/did.go"], event="workflow_dispatch"), every)

    def test_files_outside_packages_run_everything(self):
        every = {"draft1": True, "v010": True, "v011": True}
        for f in [".github/scripts/registry_test_shards.py", ".github/workflows/registry-node-evidence.yml",
                  ".gitleaks.toml", "Makefile", "scripts/gen-bindings.sh"]:
            self.assertEqual(self.suites([f]), every, f)

    def test_files_without_root_effect(self):
        self.assertEqual(self.suites(["docs/dev/x.md", "README.md", "core/go.mod", ".github/workflows/core.yml"]),
                         {"draft1": True, "v010": False, "v011": False})

    def test_nested_markdown_is_not_free(self):
        # "*.md" covers top-level files only; a .md file in a package
        # directory belongs to that package.
        self.assertTrue(self.suites(["pkg/agent/registry011/README.md"])["v011"])

    def test_changed_file_list_formats(self):
        import tempfile
        with tempfile.NamedTemporaryFile("wb", delete=False) as f:
            f.write(b"a/b.go\0docs/\xc3\xa9.md\0")
        self.assertEqual(ci_versions.read_changed(f.name), ["a/b.go", "docs/\u00e9.md"])
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            f.write("a/b.go\ndocs/x.md\n")
        self.assertEqual(ci_versions.read_changed(f.name), ["a/b.go", "docs/x.md"])

    def test_no_changes_runs_draft1(self):
        self.assertEqual(self.suites([]), {"draft1": True, "v010": False, "v011": False})


if __name__ == "__main__":
    unittest.main()
