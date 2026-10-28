package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func historyOriginalPlan(t *testing.T, kind string, payload []byte) providerHistoryOriginalPlan {
	t.Helper()
	sum := sha256.Sum256(payload)
	ref := providerIntegrationObjectRef{kind, "0x" + hex.EncodeToString(sum[:]), uint64(len(payload))}
	p, err := newProviderHistoryOriginalPlan(ref, 32<<20)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func historyOriginalDisk(t *testing.T) *retentionDisk {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AUTHORITY.LOCK"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	d, err := openRetentionDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.close() })
	return d
}

func historyOriginalReader(t *testing.T, d *retentionDisk) *providerHistoryOriginalReader {
	t.Helper()
	r, err := newProviderHistoryOriginalReader(d)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestProviderHistoryOriginalFixedWorkBounds(t *testing.T) {
	for _, row := range []struct {
		bytes uint64
		size  int64
		valid bool
	}{{0, 0, false}, {1, 1, true}, {32 << 20, 32 << 20, true}, {32<<20 + 1, 0, false}, {^uint64(0), 0, false}} {
		if size, ok := providerHistoryOriginalSize(row.bytes); size != row.size || ok != row.valid {
			t.Fatal("invalid or overflowing byte count reached file offsets")
		}
	}
	for _, row := range []struct{ size, reads, writes uint64 }{
		{1, 2, 1}, {8191, 2, 1}, {8192, 2, 1}, {8193, 3, 2}, {32 << 20, 4097, 4096},
	} {
		p, err := newProviderHistoryOriginalPlan(providerIntegrationObjectRef{"response", admissionTestDigest("a"), row.size}, 32<<20)
		if err != nil {
			t.Fatal(err)
		}
		upper, err := p.upper()
		if err != nil || upper.readBytes != row.size+1 || upper.hashBytes != row.size || upper.readCalls != row.reads || upper.hashWrites != row.writes || upper.metadataCalls != 14 || upper.bufferBytes != 8193 {
			t.Fatalf("unexpected per-original bound: %+v, %v", upper, err)
		}
	}
}

func TestProviderHistoryOriginalRejectsUnboundedPlans(t *testing.T) {
	base := providerIntegrationObjectRef{"response", admissionTestDigest("a"), 1}
	for _, row := range []struct {
		name    string
		ref     providerIntegrationObjectRef
		maximum uint64
	}{
		{"no-cap", base, 0},
		{"oversized-cap", base, 32<<20 + 1},
		{"empty", providerIntegrationObjectRef{"response", base.digest, 0}, 1},
		{"oversized-object", providerIntegrationObjectRef{"response", base.digest, 2}, 1},
		{"no-digest", providerIntegrationObjectRef{"response", "", 1}, 1},
		{"noncanonical-digest", providerIntegrationObjectRef{"response", "0X" + base.digest[2:], 1}, 1},
		{"unknown-kind", providerIntegrationObjectRef{"exchange", base.digest, 1}, 1},
		{"long-request", providerIntegrationObjectRef{"request", base.digest, 16<<10 + 1}, 32 << 20},
		{"long-params", providerIntegrationObjectRef{"params", base.digest, 8<<10 + 1}, 32 << 20},
	} {
		t.Run(row.name, func(t *testing.T) {
			if p, err := newProviderHistoryOriginalPlan(row.ref, row.maximum); err == nil || p != (providerHistoryOriginalPlan{}) {
				t.Fatal("invalid plan must return no usable data")
			}
		})
	}
	for _, row := range []struct {
		kind string
		size uint64
	}{{"request", 16 << 10}, {"params", 8 << 10}, {"response", 32 << 20}, {"tls", 32 << 20}} {
		if _, err := newProviderHistoryOriginalPlan(providerIntegrationObjectRef{row.kind, base.digest, row.size}, 32<<20); err != nil {
			t.Fatal("exact ordinary-content cap refused")
		}
	}
	var zero providerHistoryOriginalPlan
	if _, err := zero.upper(); err == nil {
		t.Fatal("zero plan acquired a cost bound")
	}
	if err := json.Unmarshal([]byte(`{}`), &zero); err == nil {
		t.Fatal("JSON must not reconstruct a plan or permission")
	}
	p := historyOriginalPlan(t, "response", []byte("abc"))
	for _, size := range []int64{0, 2, 4, -1, 1 << 40} {
		changed := p
		changed.size = size
		if _, err := changed.upper(); err == nil {
			t.Fatal("a changed file offset reused an earlier byte reservation")
		}
	}
}

func TestProviderHistoryOriginalOwnedFileChunkBoundaries(t *testing.T) {
	for _, size := range []int{1, 8191, 8192, 8193, 3*8192 + 17} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			d := historyOriginalDisk(t)
			payload := bytes.Repeat([]byte("a"), size)
			p := historyOriginalPlan(t, "response", payload)
			name := providerObjectName(p.reference.kind, p.reference.digest)
			if err := os.WriteFile(filepath.Join(d.directory, name), payload, 0600); err != nil {
				t.Fatal(err)
			}
			// Separate invocations open and read the retained original again.
			// Neither invocation receives or reuses a prior hash result.
			first, second := historyOriginalReader(t, d), historyOriginalReader(t, d)
			for _, reader := range []*providerHistoryOriginalReader{first, second, first} {
				digest, err := reader.scan(context.Background(), p)
				if err != nil || digest != p.reference.digest {
					t.Fatalf("normal independent original read failed: %s, %v", digest, err)
				}
			}
			if got, err := os.ReadFile(filepath.Join(d.directory, name)); err != nil || !bytes.Equal(got, payload) {
				t.Fatal("read-only scan altered original bytes")
			}
		})
	}
}

func TestProviderHistoryOriginalOrdinaryKinds(t *testing.T) {
	for kind, raw := range map[string]string{"request": "{}", "params": "[]", "response": "abc", "tls": "{}"} {
		t.Run(kind, func(t *testing.T) {
			d := historyOriginalDisk(t)
			p := historyOriginalPlan(t, kind, []byte(raw))
			if kind == "response" && p.reference.digest != "0xba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
				t.Fatal("ordinary-content SHA-256 acquired a domain or separator prefix")
			}
			if err := os.WriteFile(filepath.Join(d.directory, providerObjectName(kind, p.reference.digest)), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if digest, err := historyOriginalReader(t, d).scan(context.Background(), p); err != nil || digest != p.reference.digest {
				t.Fatal("ordinary content hash semantics changed")
			}
		})
	}
}

func TestProviderHistoryOriginalRefusesFixedInvalidFiles(t *testing.T) {
	// Inert files and scalar states only; no hostile responder, callback,
	// replacement race, host-bypass or attack reproduction is constructed.
	for _, state := range []string{"missing", "directory", "empty", "short", "long", "digest"} {
		t.Run(state, func(t *testing.T) {
			d := historyOriginalDisk(t)
			p := historyOriginalPlan(t, "response", []byte("abc"))
			name := filepath.Join(d.directory, providerObjectName(p.reference.kind, p.reference.digest))
			var err error
			switch state {
			case "directory":
				err = os.Mkdir(name, 0700)
			case "empty":
				err = os.WriteFile(name, nil, 0600)
			case "short":
				err = os.WriteFile(name, []byte("ab"), 0600)
			case "long":
				err = os.WriteFile(name, []byte("abcd"), 0600)
			case "digest":
				err = os.WriteFile(name, []byte("abd"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			reader := historyOriginalReader(t, d)
			if digest, err := reader.scan(context.Background(), p); err == nil || digest != "" || !reader.failed {
				t.Fatal("invalid original produced a successful digest")
			}
			if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if err := os.WriteFile(name, []byte("abc"), 0600); err != nil {
				t.Fatal(err)
			}
			if digest, err := reader.scan(context.Background(), p); err == nil || digest != "" {
				t.Fatal("replacing a fixed invalid file restored a failed reader")
			}
		})
	}
	// Compare fixed metadata snapshots after fixture preparation, never a
	// concurrent filesystem replacement or a callback during a live read.
	dir := t.TempDir()
	firstPath, otherPath := filepath.Join(dir, "first"), filepath.Join(dir, "other")
	for _, name := range []string{firstPath, otherPath} {
		if err := os.WriteFile(name, []byte("abc"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.Stat(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(firstPath, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	changedTime, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		name          string
		before, after os.FileInfo
		size          int64
		valid         bool
	}{
		{"same", before, before, 3, true},
		{"missing-before", nil, before, 3, false},
		{"missing-after", before, nil, 3, false},
		{"empty-size", before, before, 0, false},
		{"negative-size", before, before, -1, false},
		{"large-size", before, before, 32<<20 + 1, false},
		{"directory-before", directory, before, 3, false},
		{"directory-after", before, directory, 3, false},
		{"different-size", before, before, 2, false},
		{"different-identity", before, other, 3, false},
		{"different-time", before, changedTime, 3, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			if providerHistoryOriginalSameFile(row.before, row.after, row.size) != row.valid {
				t.Fatal("file metadata drift was accepted or the original was refused")
			}
		})
	}
}

func TestProviderHistoryOriginalRejectsMissingContextOrDisk(t *testing.T) {
	p := historyOriginalPlan(t, "response", []byte("abc"))
	d := historyOriginalDisk(t)
	for _, disk := range []*retentionDisk{nil, {}, {root: d.root}, {lock: d.lock}} {
		if reader, err := newProviderHistoryOriginalReader(disk); err == nil || reader != nil {
			t.Fatal("incomplete disk accepted")
		}
	}
	reader := historyOriginalReader(t, d)
	for _, invalid := range []*providerHistoryOriginalReader{nil, {}, {owner: reader}} {
		if digest, err := invalid.scan(context.Background(), p); err == nil || digest != "" {
			t.Fatal("unowned reader accepted")
		}
	}
	var zero providerHistoryOriginalPlan
	if digest, err := reader.scan(context.Background(), zero); err == nil || digest != "" || reader.failed {
		t.Fatal("unbounded original acquired a reader")
	}
	if err := json.Unmarshal([]byte(`{}`), reader); err == nil {
		t.Fatal("JSON reconstructed a reader")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, row := range []struct {
		name   string
		ctx    context.Context
		failed bool
	}{{"missing-context", nil, false}, {"cancelled-context", ctx, true}} {
		t.Run(row.name, func(t *testing.T) {
			reader := historyOriginalReader(t, d)
			if digest, err := reader.scan(row.ctx, p); err == nil || digest != "" || reader.failed != row.failed {
				t.Fatal("invalid context acquired a read or lost its expected quarantine state")
			}
			if row.failed {
				if digest, err := reader.scan(context.Background(), p); err == nil || digest != "" {
					t.Fatal("failed reader was reused")
				}
			}
		})
	}
	closed := historyOriginalReader(t, d)
	if err := d.root.Close(); err != nil {
		t.Fatal(err)
	}
	if digest, err := closed.scan(context.Background(), p); err == nil || digest != "" || !closed.failed {
		t.Fatal("closed original directory accepted")
	}
}
