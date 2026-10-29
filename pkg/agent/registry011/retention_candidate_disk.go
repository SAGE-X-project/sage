package registry011

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

type retentionDisk struct {
	root                    *os.Root
	lock                    *os.File
	directory               string
	directoryInfo, lockInfo os.FileInfo
	syncFile                func(*os.File) error
}

type retentionRecord struct {
	Format, Namespace, Sequence, Previous, Instance, Operation string
	Root                                                       json.RawMessage
}

type retentionState struct {
	sequence   uint64
	tip        string
	used       int64
	roots      map[string]string
	operations map[string]bool
	last       retentionRecord
}

func openCandidateRetentionLog(config CandidateRetentionConfig, recovery *ProviderCandidateIdentity, operation string) (*CandidateRetentionLog, *CandidateRetentionPending, error) {
	if config.Directory == "" || !lowerHash(config.Namespace) || !lowerHash(config.AnchorDigest) || !lowerHash(config.ExpectedTip) ||
		config.MaxRecords <= 0 || config.MaxRecords > 65536 || config.MaxRecordBytes <= 0 || config.MaxRecordBytes > 128<<10 ||
		config.MaxStoredBytes <= 0 || config.MaxStoredBytes > 256<<20 || config.MaxRecordBytes > config.MaxStoredBytes {
		return nil, nil, ErrConfiguration
	}
	disk, err := openRetentionDisk(config.Directory)
	if err != nil {
		return nil, nil, err
	}
	accepted := false
	defer func() {
		if !accepted {
			_ = disk.close()
		}
	}()
	state, err := loadCandidateRetentionLog(disk, config)
	if err != nil {
		return nil, nil, err
	}
	if recovery == nil {
		if state.tip != config.ExpectedTip {
			return nil, nil, ErrRetentionConflict
		}
	} else {
		root, _, err := makeCandidateEnrollment(*recovery, config.AnchorDigest)
		want := state.last
		want.Previous, want.Operation, want.Instance, want.Root = config.ExpectedTip, operation, recovery.Instance, root
		if err != nil || state.sequence == 0 || !sameRetentionRecord(state.last, want) {
			return nil, nil, ErrRetentionConflict
		}
		f, err := disk.root.OpenFile(retentionName(state.sequence), os.O_RDWR, 0)
		if err != nil {
			return nil, nil, ErrCandidateRetention
		}
		if errors.Join(disk.syncFile(f), f.Close(), disk.sync()) != nil {
			return nil, nil, ErrCandidateRetention
		}
		confirmed, err := loadCandidateRetentionLog(disk, config)
		if err != nil || confirmed.tip != state.tip || !sameRetentionRecord(confirmed.last, state.last) {
			return nil, nil, ErrCandidateRetention
		}
	}
	a := &CandidateRetentionLog{disk: disk, config: config, state: state, pending: recovery != nil}
	a.owner = a
	accepted = true
	if recovery != nil {
		return a, &CandidateRetentionPending{a, recovery.Instance, operation, config.ExpectedTip, state.tip, string(state.last.Root), state.sequence}, nil
	}
	return a, nil, nil
}

func openRetentionDisk(directory string) (*retentionDisk, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	f, err := root.OpenFile("AUTHORITY.LOCK", os.O_RDWR, 0)
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	stat, e := f.Stat()
	path, p := root.Lstat("AUTHORITY.LOCK")
	if e != nil || p != nil || !stat.Mode().IsRegular() || !path.Mode().IsRegular() || !os.SameFile(stat, path) {
		_ = f.Close()
		_ = root.Close()
		return nil, ErrCandidateRetention
	}
	if err := retentionLock(f); err != nil {
		_ = f.Close()
		_ = root.Close()
		return nil, err
	}
	directoryInfo, err := root.Stat(".")
	if err != nil {
		_ = retentionUnlock(f)
		_ = f.Close()
		_ = root.Close()
		return nil, ErrCandidateRetention
	}
	disk := &retentionDisk{root: root, lock: f, directory: directory, directoryInfo: directoryInfo, lockInfo: stat, syncFile: (*os.File).Sync}
	if err := disk.checkIdentity(); err != nil {
		_ = disk.close()
		return nil, err
	}
	return disk, nil
}

func loadCandidateRetentionLog(disk *retentionDisk, config CandidateRetentionConfig) (retentionState, error) {
	state := retentionState{tip: config.AnchorDigest, roots: make(map[string]string), operations: make(map[string]bool)}
	if err := disk.checkIdentity(); err != nil {
		return state, err
	}
	f, err := disk.root.Open(".")
	if err != nil {
		return state, ErrCandidateRetention
	}
	names, e := f.Readdirnames(config.MaxRecords + 3)
	closed := f.Close()
	if (e != nil && e != io.EOF) || closed != nil || len(names) > config.MaxRecords+2 {
		return state, ErrCandidateRetention
	}
	var entries []string
	anchor, locked := false, false
	for _, name := range names {
		switch name {
		case "ANCHOR":
			anchor = true
		case "AUTHORITY.LOCK":
			locked = true
		default:
			entries = append(entries, name)
		}
	}
	if !anchor || !locked || len(entries) > config.MaxRecords {
		return state, ErrCandidateRetention
	}
	raw, err := disk.read("ANCHOR", min(config.MaxRecordBytes, config.MaxStoredBytes))
	if err != nil || providerCandidateHash("sage-retaining-anchor-v1", raw) != config.AnchorDigest {
		return state, ErrCandidateRetention
	}
	var selected struct{ Format, Namespace, Selection string }
	if retentionDecode(raw, &selected) != nil || selected.Format != "sage-retaining-authority-v1" || selected.Namespace != config.Namespace || !lowerHash(selected.Selection) {
		return state, ErrCandidateRetention
	}
	state.used = int64(len(raw))
	sort.Strings(entries)
	for index, name := range entries {
		if name != retentionName(uint64(index)+1) {
			return state, ErrCandidateRetention
		}
		raw, err := disk.read(name, min(config.MaxRecordBytes, config.MaxStoredBytes-state.used))
		if err != nil {
			return state, ErrCandidateRetention
		}
		var entry retentionRecord
		if retentionDecode(raw, &entry) != nil || entry.Format != "sage-retaining-enrollment-v1" || entry.Namespace != config.Namespace ||
			entry.Sequence != strconv.FormatUint(uint64(index)+1, 10) || entry.Previous != state.tip || !lowerHash(entry.Instance) || !lowerHash(entry.Operation) ||
			state.roots[entry.Instance] != "" || state.operations[entry.Operation] {
			return state, ErrCandidateRetention
		}
		root, err := candidateEnrollmentRoot(contentJSON(entry.Root), config.Namespace, entry.Instance)
		if err != nil || !bytes.Equal(root, entry.Root) {
			return state, ErrCandidateRetention
		}
		fields := contentJSON(root).(map[string]any)
		if fields["generation"] != "0" || fields["sequence"] != "0" || fields["fence"] != "1" || fields["status"] != "enrolling" {
			return state, ErrCandidateRetention
		}
		state.tip = providerCandidateHash("sage-retaining-entry-v1", raw)
		state.sequence, state.used = uint64(index)+1, state.used+int64(len(raw))
		state.roots[entry.Instance], state.operations[entry.Operation], state.last = string(root), true, entry
	}
	if err := disk.checkIdentity(); err != nil {
		return state, err
	}
	return state, nil
}

func (d *retentionDisk) read(name string, maximum int64) ([]byte, error) {
	info, err := d.root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return nil, ErrCandidateRetention
	}
	f, err := d.root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		_ = f.Close()
		return nil, ErrCandidateRetention
	}
	raw, e := io.ReadAll(io.LimitReader(f, maximum+1))
	closeErr := f.Close()
	if e != nil || closeErr != nil || int64(len(raw)) != info.Size() {
		return nil, ErrCandidateRetention
	}
	return raw, nil
}

func (d *retentionDisk) write(name string, raw []byte) error {
	if err := d.checkIdentity(); err != nil {
		return err
	}
	f, err := d.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, writeErr := f.Write(raw)
	if writeErr == nil && n != len(raw) {
		writeErr = io.ErrShortWrite
	}
	return errors.Join(writeErr, d.syncFile(f), f.Close(), d.sync(), d.checkIdentity())
}

func (d *retentionDisk) sync() error {
	f, err := d.root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(d.syncFile(f), f.Close())
}

func (d *retentionDisk) close() error {
	return errors.Join(retentionUnlock(d.lock), d.lock.Close(), d.root.Close())
}

// checkIdentity detects drift within the selected local view; it does not
// attest the host, protect against mount/backup rollback or exclude uncooperative
// writers. The qualified integration must independently establish those bounds.
func (d *retentionDisk) checkIdentity() error {
	selected, err := os.Lstat(d.directory)
	opened, openedErr := d.root.Stat(".")
	lockPath, pathErr := d.root.Lstat("AUTHORITY.LOCK")
	lockFD, fdErr := d.lock.Stat()
	if err != nil || openedErr != nil || pathErr != nil || fdErr != nil ||
		!selected.IsDir() || !opened.IsDir() || !lockPath.Mode().IsRegular() ||
		!lockFD.Mode().IsRegular() || !os.SameFile(d.directoryInfo, selected) ||
		!os.SameFile(d.directoryInfo, opened) || !os.SameFile(d.lockInfo, lockPath) ||
		!os.SameFile(d.lockInfo, lockFD) {
		return ErrCandidateRetention
	}
	return nil
}

func retentionName(sequence uint64) string { return fmt.Sprintf("entry-%020d", sequence) }

func retentionEncode(value any) ([]byte, error) { return json.Marshal(value) }

func retentionDecode(raw []byte, value any) error {
	if json.Unmarshal(raw, value) != nil {
		return ErrCandidateRetention
	}
	encoded, err := retentionEncode(value)
	if err != nil || !bytes.Equal(raw, encoded) {
		return ErrCandidateRetention
	}
	return nil
}
