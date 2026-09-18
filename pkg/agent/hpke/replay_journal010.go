package hpke

import (
	"bytes"
	"encoding/json"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const replayHeader010 = "sage-replay-denials|0.10.0\n"

// ReplayJournal010 is a bounded single-writer denial journal on trusted Linux/macOS
// storage. It does not restore sessions or defend against malicious disk rollback.
// Missing/partial files never become empty valid state on ordinary reopen.
type ReplayJournal010 struct {
	mu                sync.Mutex
	file              *os.File
	lock              string
	clock             registry010.Clock
	start, last       registry010.Stamp
	recovered, failed bool
	rows              int
	size              int
	seen              map[[4]string]int64
}
type replayRow010 struct {
	At        int64  `json:"at"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	ID        string `json:"id"`
	Nonce     string `json:"nonce"`
	Context   string `json:"context"`
	Expires   int64  `json:"expires"`
}

func rowReplay010(v Replay010, at int64) replayRow010 {
	return replayRow010{at, v.Sender, v.Recipient, v.ID, v.Nonce, v.Context, v.Expires}
}
func replayText010(s string) bool {
	for _, b := range []byte(s) {
		if b < 33 || b > 126 || strings.ContainsRune("\"\\<>&", rune(b)) {
			return false
		}
	}
	return true
}
func validReplayRow010(v replayRow010) bool {
	for _, s := range []string{v.Sender, v.Recipient, v.ID, v.Nonce} {
		if len(s) == 0 || len(s) > 1024 || !replayText010(s) {
			return false
		}
	}
	return len(v.Context) <= 1024 && replayText010(v.Context) && v.At >= 0 && v.At <= 9007199254740000 && v.Expires >= v.At-30 && v.Expires <= v.At+330
}
func replayKeys010(v replayRow010) [][4]string {
	k := [][4]string{{v.Sender, v.Recipient, "id", v.ID}, {v.Sender, v.Recipient, "nonce", v.Nonce}}
	if v.Context != "" {
		k = append(k, [4]string{v.Sender, "", "context", v.Context})
	}
	return k
}

// OpenReplayJournal010 requires a trusted clock and exclusive path ownership.
// create exclusively creates a new file and imposes a full 360-second UTC AND
// monotonic quarantine. Reopening an empty journal restarts quarantine. Recovery
// of a complete nonempty journal permits immediate use with nondecreasing UTC.
// A crash leaves .lock: an operator must establish ownership before removing it.
func OpenReplayJournal010(path string, create bool, clock registry010.Clock) (*ReplayJournal010, error) {
	if (runtime.GOOS != "linux" && runtime.GOOS != "darwin") || clock == nil {
		return nil, errCompletion010
	}
	now, e := clock.Now()
	if e != nil || now.Unix < 0 || now.MonoMS < 0 || now.Unix > 9007199254740000 {
		return nil, errCompletion010
	}
	lock := path + ".lock"
	// #nosec G304 -- trusted deployment path; exclusive single-writer creation.
	guard, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	if e = guard.Close(); e != nil {
		_ = os.Remove(lock)
		return nil, e
	}
	flags := os.O_RDWR | os.O_APPEND
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	// #nosec G304 -- trusted deployment path; missing state is never silently recreated.
	f, e := os.OpenFile(path, flags, 0600)
	if e != nil {
		_ = os.Remove(lock)
		return nil, e
	}
	j := &ReplayJournal010{file: f, lock: lock, clock: clock, start: now, last: now, seen: map[[4]string]int64{}}
	fail := func(e error) (*ReplayJournal010, error) { _ = j.Close(); return nil, e }
	if create {
		if _, e = f.WriteString(replayHeader010); e != nil {
			return fail(e)
		}
		if e = f.Sync(); e != nil {
			return fail(e)
		}
		// #nosec G304 -- sync the trusted parent directory before admitting new state.
		dir, x := os.Open(filepath.Dir(path))
		if x != nil {
			return fail(x)
		}
		x = dir.Sync()
		closeErr := dir.Close()
		if x != nil {
			return fail(x)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
	}
	st, e := f.Stat()
	if e != nil {
		return fail(e)
	}
	if st.Size() > 1048576 {
		return fail(errCompletion010)
	}
	// #nosec G304 -- bounded trusted journal, held exclusively.
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return fail(e)
	}
	data, e := io.ReadAll(io.LimitReader(f, 1048577))
	if e != nil {
		return fail(e)
	}
	if len(data) > 1048576 {
		return fail(errCompletion010)
	}
	if !bytes.HasPrefix(data, []byte(replayHeader010)) || !bytes.HasSuffix(data, []byte{'\n'}) {
		return fail(errCompletion010)
	}
	previous := int64(0)
	lines := bytes.Split(data[len(replayHeader010):], []byte{'\n'})
	for _, line := range lines[:len(lines)-1] {
		if len(line) == 0 {
			return fail(errCompletion010)
		}
		var v replayRow010
		if json.Unmarshal(line, &v) != nil {
			return fail(errCompletion010)
		}
		canonical, _ := json.Marshal(v)
		if !bytes.Equal(canonical, line) || !validReplayRow010(v) || v.At < previous || v.At > now.Unix || j.rows >= 4096 {
			return fail(errCompletion010)
		}
		if j.duplicate(v, v.At) {
			return fail(errCompletion010)
		}
		j.remember(v)
		j.rows++
		previous = v.At
	}
	j.size = len(data)
	j.recovered = j.rows > 0
	return j, nil
}
func (j *ReplayJournal010) sample() (registry010.Stamp, error) {
	if j.failed || j.file == nil {
		return registry010.Stamp{}, errCompletion010
	}
	t, e := j.clock.Now()
	if e != nil || t.Unix < j.last.Unix || t.MonoMS < j.last.MonoMS || t.Unix > 9007199254740000 {
		j.failed = true
		return t, errCompletion010
	}
	j.last = t
	return t, nil
}
func (j *ReplayJournal010) ready(t registry010.Stamp) bool {
	return j.recovered || (t.Unix-j.start.Unix >= 360 && t.MonoMS-j.start.MonoMS >= 360000)
}

// Ready samples trusted time; false cannot be overridden by a peer assertion.
func (j *ReplayJournal010) Ready() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	t, e := j.sample()
	return e == nil && j.ready(t)
}
func (j *ReplayJournal010) duplicate(v replayRow010, at int64) bool {
	for _, k := range replayKeys010(v) {
		if until, ok := j.seen[k]; ok && (k[2] == "context" || at <= until) {
			return true
		}
	}
	return false
}
func (j *ReplayJournal010) remember(v replayRow010) {
	for _, k := range replayKeys010(v) {
		j.seen[k] = v.Expires + 30
	}
}
func (j *ReplayJournal010) reserve(v Replay010, validate func() error) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	now, e := j.sample()
	if e != nil || !j.ready(now) {
		return errCompletion010
	}
	row := rowReplay010(v, now.Unix)
	if !validReplayRow010(row) || j.rows >= 4096 || j.duplicate(row, now.Unix) {
		return errCompletion010
	}
	data, e := json.Marshal(row)
	if e != nil {
		return e
	}
	data = append(data, '\n')
	if j.size+len(data) > 1048576 {
		return errCompletion010
	}
	// Durably stage denial before the endpoint's final gate. Failure never releases
	// plaintext or advances sequence, but this denial survives failed gates/restart.
	j.failed = true
	n, e := j.file.Write(data)
	if e != nil || n != len(data) {
		return errCompletion010
	}
	if j.file.Sync() != nil {
		return errCompletion010
	}
	j.remember(row)
	j.rows++
	j.size += len(data)
	j.failed = false
	if _, e = j.sample(); e != nil {
		return e
	}
	if validate != nil {
		return validate()
	}
	return nil
}
func (j *ReplayJournal010) Reserve(v Replay010) error { return j.reserve(v, nil) }

// ReserveRecord stages durable denial before the final gate. The callback must
// not reenter this journal; failure retains denial without accepting a record.
func (j *ReplayJournal010) ReserveRecord(v Replay010, validate func() error) error {
	if validate == nil || v.Context != "" {
		return errCompletion010
	}
	return j.reserve(v, validate)
}
func (j *ReplayJournal010) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	e := j.file.Close()
	j.file = nil
	x := os.Remove(j.lock)
	if e != nil {
		return e
	}
	return x
}
