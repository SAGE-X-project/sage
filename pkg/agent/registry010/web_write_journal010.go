package registry010

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

const webWriteJournalHeader010 = "sage-web-registry-writes|0.10.0|operator-1\n"
const webWriteJournalLimit010 = 64 * 1024 * 1024

type webWriteJournalBinding010 struct {
	DID    string `json:"did"`
	Source string `json:"source"`
}

// WebRegistryWriteJournal010 is a single-DID, single-writer local reference
// store. The caller pins the path, DID, source and credential-backed authority.
// A crash or uncertain I/O leaves the exclusive lock in place; an operator
// must establish exclusive ownership and inspect the file before recovery.
// Trusted filesystem integrity and malicious disk rollback remain outside it.
type WebRegistryWriteJournal010 struct {
	mu        sync.Mutex
	file      *os.File
	lock      string
	did       string
	source    string
	authority WebRegistryAdminAuthority010
	state     WebRegistryWriteState010
	size      int64
	failed    bool
}

func webWriteJournalDecode010(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidRecord010
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return ErrInvalidRecord010
	}
	return nil
}

func webWriteJournalStep010(previous, next WebRegistryWriteState010, source, did string) error {
	if next.Source != source || len(next.History) != len(previous.History)+1 ||
		(previous.Tombstoned || len(previous.Envelope) == 0 && len(previous.History) != 0) {
		return ErrInvalidRecord010
	}
	for index, entry := range previous.History {
		if !reflect.DeepEqual(entry, next.History[index]) {
			return ErrInvalidRecord010
		}
	}
	last := next.History[len(next.History)-1]
	if !bytes.Equal(last.Envelope, next.Envelope) ||
		CheckWebRegistryHistoryContinuity010(next.History, next.Envelope, did, last.At) != nil {
		return ErrInvalidRecord010
	}
	record, err := webTransitionRecordFromEnvelopeWithPolicy010(next.Envelope, did, last.At, false)
	if err != nil || next.Tombstoned != (record.State == "deactivated") {
		return ErrInvalidRecord010
	}
	if len(previous.History) == 0 {
		if len(previous.Envelope) != 0 || previous.Tombstoned || last.Operation != "create" {
			return ErrInvalidRecord010
		}
	} else if last.At < previous.History[len(previous.History)-1].At {
		return ErrInvalidRecord010
	}
	if webVerifyGrantStep010(previous, next, did) != nil {
		return ErrInvalidRecord010
	}
	return nil
}

func webWriteJournalRefresh010(raw []byte, now int64) ([]byte, error) {
	if now < 0 || now > 9007199254740986 {
		return nil, ErrInvalidRecord010
	}
	var stored struct {
		Record  json.RawMessage `json:"record"`
		Issued  int64           `json:"issued"`
		Expires int64           `json:"expires"`
	}
	if err := webWriteJournalDecode010(raw, &stored); err != nil || len(stored.Record) == 0 {
		return nil, ErrInvalidRecord010
	}
	return json.Marshal(struct {
		Record  json.RawMessage `json:"record"`
		Issued  int64           `json:"issued"`
		Expires int64           `json:"expires"`
	}{stored.Record, now, now + 5})
}

func webWriteJournalSyncDir010(path string) error {
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

// OpenWebRegistryWriteJournal010 creates state only with create=true. Ordinary
// restart requires the existing complete journal and no lock. The path and
// authority must come from trusted deployment configuration.
func OpenWebRegistryWriteJournal010(path, did, source string, authority WebRegistryAdminAuthority010, create bool) (*WebRegistryWriteJournal010, error) {
	if path == "" || did == "" || source == "" || authority == nil {
		return nil, ErrRejected
	}
	lock := path + ".lock"
	// #nosec G304 -- trusted deployment path; exclusive writer lock.
	guard, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err = guard.Close(); err != nil {
		_ = os.Remove(lock)
		return nil, err
	}
	if err = webWriteJournalSyncDir010(lock); err != nil {
		_ = os.Remove(lock)
		return nil, err
	}
	flags := os.O_RDWR | os.O_APPEND
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	// #nosec G304 -- caller pins the local path; restart never creates missing state.
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		_ = os.Remove(lock)
		return nil, err
	}
	journal := &WebRegistryWriteJournal010{
		file: file, lock: lock, did: did, source: source,
		authority: authority, state: WebRegistryWriteState010{Source: source},
	}
	fail := func(cause error) (*WebRegistryWriteJournal010, error) {
		_ = file.Close()
		journal.file = nil
		_ = os.Remove(lock)
		return nil, cause
	}
	failInitialWrite := func() (*WebRegistryWriteJournal010, error) {
		_ = file.Close()
		journal.file = nil
		// The initialization write may have reached disk; retain the lock.
		return nil, ErrUnreachable
	}
	if create {
		binding, marshalErr := json.Marshal(webWriteJournalBinding010{did, source})
		if marshalErr != nil {
			return fail(marshalErr)
		}
		initial := append([]byte(webWriteJournalHeader010), binding...)
		initial = append(initial, '\n')
		if written, writeErr := file.Write(initial); writeErr != nil || written != len(initial) {
			return failInitialWrite()
		}
		if err = file.Sync(); err != nil {
			return failInitialWrite()
		}
		if err = webWriteJournalSyncDir010(path); err != nil {
			return failInitialWrite()
		}
	}
	stat, err := file.Stat()
	if err != nil || stat.Size() > webWriteJournalLimit010 {
		return fail(ErrUnreachable)
	}
	journal.size = stat.Size()
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	reader := bufio.NewReader(file)
	if header, readErr := reader.ReadString('\n'); readErr != nil || header != webWriteJournalHeader010 {
		return fail(ErrInvalidRecord010)
	}
	bindingLine, err := reader.ReadBytes('\n')
	if err != nil {
		return fail(ErrInvalidRecord010)
	}
	var binding webWriteJournalBinding010
	if webWriteJournalDecode010(bindingLine, &binding) != nil || binding.DID != did || binding.Source != source {
		return fail(ErrInvalidRecord010)
	}
	for {
		line, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF && len(line) == 0 {
			break
		}
		if readErr != nil {
			return fail(ErrInvalidRecord010)
		}
		var next WebRegistryWriteState010
		if webWriteJournalDecode010(line, &next) != nil ||
			webWriteJournalStep010(journal.state, next, source, did) != nil {
			return fail(ErrInvalidRecord010)
		}
		journal.state = webCopyWriteState010(next)
	}
	return journal, nil
}

// Update holds the writer lock while it validates the latest durable state,
// invokes the policy once, and appends the entire replacement as one synced
// row. A write or sync failure quarantines the journal with an unknown commit
// outcome; no later writes may proceed without operator recovery.
func (j *WebRegistryWriteJournal010) Update(ctx context.Context, did string, now int64, decide func(WebRegistryWriteSnapshot010) (WebRegistryWriteState010, error)) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if ctx == nil || ctx.Err() != nil || j.failed || j.file == nil || did != j.did {
		return ErrUnreachable
	}
	snapshot := webCopyWriteState010(j.state)
	if len(snapshot.Envelope) != 0 {
		fresh, err := webWriteJournalRefresh010(snapshot.Envelope, now)
		if err != nil {
			return err
		}
		snapshot.Envelope = fresh
	}
	next, err := decide(WebRegistryWriteSnapshot010{State: snapshot, Authority: j.authority})
	if err != nil {
		return err
	}
	if ctx.Err() != nil || webWriteJournalStep010(j.state, next, j.source, j.did) != nil {
		return ErrUnreachable
	}
	line, err := json.Marshal(next)
	if err != nil {
		return ErrUnreachable
	}
	line = append(line, '\n')
	if j.size+int64(len(line)) > webWriteJournalLimit010 {
		return ErrUnreachable
	}
	n, err := j.file.Write(line)
	if err != nil || n != len(line) {
		j.failed = true
		return ErrUnreachable
	}
	if err = j.file.Sync(); err != nil {
		j.failed = true
		return ErrUnreachable
	}
	j.size += int64(n)
	j.state = webCopyWriteState010(next)
	return nil
}

// Inspect returns a copy of local state for diagnostics, never write authority.
func (j *WebRegistryWriteJournal010) Inspect() WebRegistryWriteState010 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return webCopyWriteState010(j.state)
}

// Close releases a clean writer lock. A quarantined store retains its lock.
func (j *WebRegistryWriteJournal010) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		if j.failed {
			return ErrUnreachable
		}
		return nil
	}
	err := j.file.Close()
	j.file = nil
	if err != nil || j.failed {
		return ErrUnreachable
	}
	if err = os.Remove(j.lock); err != nil {
		return err
	}
	if err = webWriteJournalSyncDir010(j.lock); err != nil {
		return err
	}
	return nil
}

var _ WebRegistryWriteStore010 = (*WebRegistryWriteJournal010)(nil)
