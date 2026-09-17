package registry010

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
)

const journalHeader = "sage-registry-watermarks|0.10.0\n"

// Watermark is persistent denial/rollback state, never positive authority.
type Watermark struct {
	Scope    Scope  `json:"scope"`
	Version  uint64 `json:"version"`
	Digest   string `json:"digest"`
	Terminal bool   `json:"terminal"`
}

// Journal is a single-writer durable append log on trusted local storage.
// The exclusive lock file is held until Close. A crash leaves the lock in place:
// an operator must establish exclusive ownership before removing it. Never
// automatically delete a stale lock or recreate lost state on ordinary restart.
type Journal struct {
	mu     sync.Mutex
	file   *os.File
	lock   string
	values map[Scope]Watermark
	failed bool
	size   int64
}

// OpenJournal creates new state only with create=true; reopening requires an
// existing complete journal. Paths, storage integrity and exclusive recovery are
// deployment obligations. It does not protect against malicious disk rollback.
func OpenJournal(path string, create bool) (*Journal, error) {
	lock := path + ".lock"
	l, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	if e = l.Close(); e != nil {
		_ = os.Remove(lock)
		return nil, e
	}
	flags := os.O_RDWR | os.O_APPEND
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, e := os.OpenFile(path, flags, 0600)
	if e != nil {
		_ = os.Remove(lock)
		return nil, e
	}
	j := &Journal{file: f, lock: lock, values: map[Scope]Watermark{}}
	fail := func(e error) (*Journal, error) { _ = j.Close(); return nil, e }
	if create {
		if _, e = f.WriteString(journalHeader); e != nil {
			return fail(e)
		}
		if e = f.Sync(); e != nil {
			return fail(e)
		}
	}
	stat, e := f.Stat()
	if e != nil {
		return fail(e)
	}
	if stat.Size() > 64*1024*1024 {
		return fail(errors.New("journal capacity"))
	}
	j.size = stat.Size()
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return fail(e)
	}
	reader := bufio.NewReader(f)
	h, e := reader.ReadString('\n')
	if e != nil || h != journalHeader {
		return fail(errors.New("invalid journal"))
	}
	for {
		line, e := reader.ReadBytes('\n')
		if e == io.EOF && len(line) == 0 {
			break
		}
		if e != nil {
			return fail(errors.New("incomplete journal"))
		}
		var w Watermark
		d := json.NewDecoder(bytes.NewReader(line))
		d.DisallowUnknownFields()
		if d.Decode(&w) != nil {
			return fail(errors.New("invalid journal row"))
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			return fail(errors.New("invalid journal row"))
		}
		if e = j.check(w); e != nil {
			return fail(e)
		}
		j.values[w.Scope] = w
	}
	return j, nil
}
func (j *Journal) check(w Watermark) error {
	if w.Scope.Registry == "" || w.Scope.DID == "" || w.Version == 0 || !hex32.MatchString(w.Digest) {
		return ErrRejected
	}
	if old, ok := j.values[w.Scope]; ok {
		if w.Version < old.Version || (old.Terminal && !w.Terminal) || (w.Version == old.Version && (w.Digest != old.Digest || w.Terminal != old.Terminal)) {
			return ErrStale
		}
	} else if len(j.values) >= 4096 {
		return errors.New("journal record capacity")
	}
	return nil
}
func (j *Journal) Advance(s Scope, v uint64, d string, t bool) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.failed || j.file == nil {
		return ErrUnreachable
	}
	w := Watermark{s, v, d, t}
	if e := j.check(w); e != nil {
		return e
	}
	if old, ok := j.values[s]; ok && old == w {
		return nil
	}
	b, e := json.Marshal(w)
	if e != nil {
		return e
	}
	b = append(b, '\n')
	if j.size+int64(len(b)) > 64*1024*1024 {
		return ErrUnreachable
	}
	n, e := j.file.Write(b)
	if e != nil || n != len(b) {
		j.failed = true
		return ErrUnreachable
	}
	if e = j.file.Sync(); e != nil {
		j.failed = true
		return ErrUnreachable
	}
	j.size += int64(n)
	j.values[s] = w
	return nil
}

// Get exposes denial-state instrumentation; it must never authorize a request.
func (j *Journal) Get(s Scope) (Watermark, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	v, ok := j.values[s]
	return v, ok
}
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	e := j.file.Close()
	j.file = nil
	r := os.Remove(j.lock)
	if e != nil {
		return e
	}
	return r
}
