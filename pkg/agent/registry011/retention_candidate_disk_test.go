//go:build linux || darwin

package registry011

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Refusal scenarios are finite test-only store bytes and private-field states.
// There is no hostile runtime, exploit, rollback tool or operating-store access.
func TestCandidateRetentionClosedPrefixRefusals(t *testing.T) {
	for _, mode := range []string{"unknown-file", "gap", "partial", "entry-format", "entry-namespace", "entry-sequence", "entry-previous", "entry-instance", "entry-operation", "root-status", "root-generation", "root-unknown", "root-fence", "root-noninitial", "unknown-member", "duplicate-member", "whitespace", "duplicate-instance", "duplicate-operation", "anchor-hash", "anchor-format", "anchor-namespace", "anchor-selection", "anchor-extra", "anchor-missing", "anchor-directory", "entry-directory", "record-limit", "entry-size", "stored-size"} {
		t.Run(mode, func(t *testing.T) {
			cfg := retentionTestConfig(t)
			a := retentionTestOpen(t, cfg)
			d := retentionTestEnroll(t, a)
			if err := a.Close(); err != nil {
				t.Fatal(err)
			}
			cfg.ExpectedTip = d.Tip
			entryPath := filepath.Join(cfg.Directory, retentionName(1))
			raw, err := os.ReadFile(entryPath)
			if err != nil {
				t.Fatal(err)
			}
			var entry retentionRecord
			if retentionDecode(raw, &entry) != nil {
				t.Fatal("fixture entry")
			}
			switch mode {
			case "unknown-file":
				err = os.WriteFile(filepath.Join(cfg.Directory, "unknown"), []byte("x"), 0600)
			case "gap":
				err = os.Rename(entryPath, filepath.Join(cfg.Directory, retentionName(2)))
			case "partial":
				raw = []byte(`{"Format":`)
			case "entry-format":
				entry.Format = "unknown"
			case "entry-namespace":
				entry.Namespace = "0x" + strings.Repeat("9", 64)
			case "entry-sequence":
				entry.Sequence = "01"
			case "entry-previous":
				entry.Previous = d.Tip
			case "entry-instance":
				entry.Instance = "invalid"
			case "entry-operation":
				entry.Operation = "invalid"
			case "root-status", "root-generation", "root-unknown", "root-fence", "root-noninitial":
				root := contentJSON(entry.Root).(map[string]any)
				switch mode {
				case "root-status":
					root["status"] = "active"
				case "root-generation":
					root["generation"] = "1"
				case "root-fence":
					root["fence"] = "2"
				case "root-noninitial":
					root = enrollmentTestRoot(t, false, "enrolling")
				default:
					root["unexpected"] = true
				}
				entry.Root = providerPolicyTestBytes(t, root)
				if mode == "root-fence" || mode == "root-noninitial" {
					valid, err := candidateEnrollmentRoot(root, cfg.Namespace, entry.Instance)
					if err != nil || !bytes.Equal(valid, entry.Root) {
						t.Fatal("fixture must pass the root schema before the initial-enrollment check", err)
					}
				}
			case "unknown-member":
				raw = append(append([]byte{}, raw[:len(raw)-1]...), []byte(`,"Unknown":true}`)...)
			case "duplicate-member":
				raw = append(append([]byte{}, raw[:len(raw)-1]...), []byte(`,"Format":"sage-retaining-enrollment-v1"}`)...)
			case "whitespace":
				raw = append(raw, '\n')
			case "duplicate-instance", "duplicate-operation":
				second := entry
				second.Sequence = "2"
				second.Previous = d.Tip
				if mode == "duplicate-instance" {
					second.Operation = "0x" + strings.Repeat("9", 64)
				} else {
					second.Instance = "0x" + strings.Repeat("9", 64)
					root := contentJSON(second.Root).(map[string]any)
					root["instance_key"] = second.Instance
					second.Root = providerPolicyTestBytes(t, root)
				}
				b, e := retentionEncode(second)
				if e != nil {
					t.Fatal(e)
				}
				err = os.WriteFile(filepath.Join(cfg.Directory, retentionName(2)), b, 0600)
			case "anchor-hash":
				cfg.AnchorDigest = "0x" + strings.Repeat("9", 64)
			case "anchor-format", "anchor-namespace", "anchor-selection", "anchor-extra":
				anchor, er := os.ReadFile(filepath.Join(cfg.Directory, "ANCHOR"))
				if er != nil {
					t.Fatal(er)
				}
				var m map[string]any
				if json.Unmarshal(anchor, &m) != nil {
					t.Fatal("anchor fixture")
				}
				switch mode {
				case "anchor-format":
					m["Format"] = "unknown"
				case "anchor-namespace":
					m["Namespace"] = "0x" + strings.Repeat("9", 64)
				case "anchor-selection":
					m["Selection"] = ""
				case "anchor-extra":
					m["Unknown"] = true
				}
				// The altered anchor digest is explicitly supplied to reach the
				// closed schema check; it authenticates no external authority.
				anchor, er = retentionEncode(m)
				if er != nil {
					t.Fatal(er)
				}
				err = os.WriteFile(filepath.Join(cfg.Directory, "ANCHOR"), anchor, 0600)
				cfg.AnchorDigest = enrollmentTestHash("sage-retaining-anchor-v1", anchor)
			case "anchor-missing":
				err = os.Remove(filepath.Join(cfg.Directory, "ANCHOR"))
			case "anchor-directory":
				err = os.Remove(filepath.Join(cfg.Directory, "ANCHOR"))
				if err == nil {
					err = os.Mkdir(filepath.Join(cfg.Directory, "ANCHOR"), 0700)
				}
			case "entry-directory":
				err = os.Remove(entryPath)
				if err == nil {
					err = os.Mkdir(entryPath, 0700)
				}
			case "record-limit":
				cfg.MaxRecords = 1
				err = os.WriteFile(filepath.Join(cfg.Directory, retentionName(2)), []byte("x"), 0600)
			case "entry-size":
				cfg.MaxRecordBytes = int64(len(raw) - 1)
			case "stored-size":
				anchor, er := os.ReadFile(filepath.Join(cfg.Directory, "ANCHOR"))
				if er != nil {
					t.Fatal(er)
				}
				cfg.MaxStoredBytes = int64(len(anchor) + len(raw) - 1)
				cfg.MaxRecordBytes = int64(len(raw))
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(mode, "entry-") && mode != "entry-size" && mode != "entry-directory" || strings.HasPrefix(mode, "root-") {
				raw, err = retentionEncode(entry)
				if err != nil {
					t.Fatal(err)
				}
			}
			if mode == "partial" || mode == "unknown-member" || mode == "duplicate-member" || mode == "whitespace" || strings.HasPrefix(mode, "entry-") && mode != "entry-directory" && mode != "entry-size" || strings.HasPrefix(mode, "root-") {
				if err := os.WriteFile(entryPath, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if owner, err := OpenCandidateRetentionLog(cfg); err == nil || owner != nil {
				if owner != nil {
					_ = owner.Close()
				}
				t.Fatal("invalid prefix accepted")
			}
		})
	}
}

func TestCandidateRetentionAppendBoundsAndIdentityFencing(t *testing.T) {
	for _, mode := range []string{"record-bytes", "total-bytes", "record-count", "directory-identity", "lock-identity", "lock-path", "root-closed", "changed-prefix"} {
		t.Run(mode, func(t *testing.T) {
			cfg := retentionTestConfig(t)
			if mode == "record-bytes" {
				cfg.MaxRecordBytes = 512
			}
			if mode == "total-bytes" {
				cfg.MaxStoredBytes = 512
				cfg.MaxRecordBytes = 512
			}
			a := retentionTestOpen(t, cfg)
			c := enrollmentTestContext(t)
			id, _ := c.Identity()
			read, err := a.ReadInstance(id.Instance)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "record-count":
				a.config.MaxRecords = 0
			case "directory-identity":
				a.disk.directoryInfo, err = os.Stat(t.TempDir())
			case "lock-identity":
				a.disk.lockInfo, err = os.Stat(filepath.Join(cfg.Directory, "ANCHOR"))
			case "lock-path":
				a.disk.directory = t.TempDir()
			case "root-closed":
				err = a.disk.root.Close()
			case "changed-prefix":
				err = os.WriteFile(filepath.Join(cfg.Directory, "unknown"), []byte("finite test fixture"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if pending, err := a.EnrollInstance(read, c, "0x"+strings.Repeat("8", 64)); err == nil || pending != nil {
				t.Fatal("invalid local append")
			}
			if mode != "changed-prefix" {
				if _, err := os.Stat(filepath.Join(cfg.Directory, retentionName(1))); !os.IsNotExist(err) {
					t.Fatal("refused append wrote an entry")
				}
			}
		})
	}
	for _, operation := range []string{"read", "diagnostic", "append"} {
		t.Run(operation, func(t *testing.T) {
			a := retentionTestOpen(t, retentionTestConfig(t))
			c := enrollmentTestContext(t)
			id, _ := c.Identity()
			read, err := a.ReadInstance(id.Instance)
			if err != nil {
				t.Fatal(err)
			}
			a.disk.lockInfo, err = os.Stat(filepath.Join(a.config.Directory, "ANCHOR"))
			if err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "read":
				_, err = a.ReadInstance(id.Instance)
			case "diagnostic":
				_, err = read.Diagnostic()
			case "append":
				_, err = a.EnrollInstance(read, c, "0x"+strings.Repeat("8", 64))
			}
			if err == nil || !a.fenced {
				t.Fatal("identity drift not fenced")
			}
		})
	}
}

func TestCandidateRetentionDiskPrimitiveErrors(t *testing.T) {
	cfg := retentionTestConfig(t)
	disk, err := openRetentionDisk(cfg.Directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = disk.close() })
	for _, name := range []string{"absent", "AUTHORITY.LOCK"} {
		if raw, err := disk.read(name, 1); err == nil || raw != nil {
			t.Fatal("invalid bounded read")
		}
	}
	if err := disk.write("ANCHOR", []byte("x")); err == nil {
		t.Fatal("exclusive append replaced anchor")
	}
	if raw, err := retentionEncode(make(chan int)); err == nil || raw != nil {
		t.Fatal("unencodable metadata")
	}
	var value struct{ A string }
	for _, raw := range [][]byte{nil, []byte(`{`), []byte(`{"A":"x","A":"x"}`), []byte(`{"A":"x","B":null}`), []byte(`{"A":"\u0078"}`)} {
		if err := retentionDecode(raw, &value); err == nil {
			t.Fatal("non-exact local schema")
		}
	}
	if err := retentionDecode([]byte(`{"A":"x"}`), &value); err != nil {
		t.Fatal(err)
	}
	if err := disk.root.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := disk.read("ANCHOR", cfg.MaxRecordBytes); err == nil {
		t.Fatal("closed root read")
	}
	if disk.sync() == nil {
		t.Fatal("closed directory sync")
	}
	if disk.write("entry", []byte("x")) == nil {
		t.Fatal("closed root write")
	}
}

func TestCandidateRetentionCloseFailureRetained(t *testing.T) {
	a := retentionTestOpen(t, retentionTestConfig(t))
	if err := a.disk.lock.Close(); err != nil {
		t.Fatal(err)
	}
	first := a.Close()
	if first == nil || a.Close() != first {
		t.Fatal("close error not cached")
	}
}

func TestCandidateRetentionExactStorageBounds(t *testing.T) {
	cfg := retentionTestConfig(t)
	c := enrollmentTestContext(t)
	id, _ := c.Identity()
	root, _, err := makeCandidateEnrollment(id, cfg.AnchorDigest)
	if err != nil {
		t.Fatal(err)
	}
	entry := retentionRecord{"sage-retaining-enrollment-v1", cfg.Namespace, "1", cfg.AnchorDigest, id.Instance, "0x" + strings.Repeat("8", 64), root}
	raw, err := retentionEncode(entry)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := os.ReadFile(filepath.Join(cfg.Directory, "ANCHOR"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxRecords = 1
	cfg.MaxRecordBytes = int64(len(raw))
	cfg.MaxStoredBytes = int64(len(anchor) + len(raw))
	a := retentionTestOpen(t, cfg)
	d := retentionTestEnroll(t, a)
	stored, err := os.ReadFile(filepath.Join(cfg.Directory, retentionName(1)))
	if err != nil || !bytes.Equal(stored, raw) {
		t.Fatal("exact entry", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.ExpectedTip = d.Tip
	_ = retentionTestOpen(t, cfg)
}

func TestCandidateRetentionMissingPermanentLockRefuses(t *testing.T) {
	cfg := retentionTestConfig(t)
	lock := filepath.Join(cfg.Directory, "AUTHORITY.LOCK")
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if owner, err := OpenCandidateRetentionLog(cfg); err == nil || owner != nil {
		t.Fatal("missing ownership inode accepted")
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("ownership inode was recreated")
	}
}

// Finite malformed path fixtures only. No concurrent hostile process, writer
// or operating directory is created or modified by these refusal units.
func TestCandidateRetentionSelectedPathMustBeRegular(t *testing.T) {
	for _, mode := range []string{"lock-link", "directory-link"} {
		t.Run(mode, func(t *testing.T) {
			cfg := retentionTestConfig(t)
			switch mode {
			case "lock-link":
				name := filepath.Join(cfg.Directory, "AUTHORITY.LOCK")
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("ANCHOR", name); err != nil {
					t.Fatal(err)
				}
			case "directory-link":
				name := filepath.Join(t.TempDir(), "selected")
				if err := os.Symlink(cfg.Directory, name); err != nil {
					t.Fatal(err)
				}
				cfg.Directory = name
			}
			if owner, err := OpenCandidateRetentionLog(cfg); err == nil || owner != nil {
				if owner != nil {
					_ = owner.Close()
				}
				t.Fatal("linked ownership path accepted")
			}
		})
	}
}
