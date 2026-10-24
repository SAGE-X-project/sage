package registry011

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
)

// This is a local trusted directory boundary, not protection against its writer,
// hard links, mounts or a compromised host. Root confines path resolution. Fixed
// names, exclusive writes, regular-file checks and bounds reject ambiguous data.
// Successful Sync semantics of the actual filesystem need separate enrollment.
type journalDisk struct {
	root     *os.Root
	syncFile func(*os.File) error
}

func openJournalDisk(directory string, create bool) (*journalDisk, error) {
	r, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	d := &journalDisk{r, (*os.File).Sync}
	if create {
		f, e := r.Open(".")
		if e != nil {
			_ = r.Close()
			return nil, e
		}
		names, e := f.Readdirnames(1)
		closeErr := f.Close()
		if len(names) != 0 || (e != nil && e != io.EOF) || closeErr != nil {
			_ = r.Close()
			return nil, ErrJournal
		}
	}
	// Never steal a lock, including one left by a crashed process. An explicit
	// independently authenticated operator procedure is outside this API.
	if err = d.write("LOCK", []byte("locked\n")); err != nil {
		_ = r.Close()
		return nil, err
	}
	return d, nil
}

func (d *journalDisk) sync() error {
	f, err := d.root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(d.syncFile(f), f.Close())
}

func (d *journalDisk) write(name string, data []byte) error {
	f, err := d.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, e := f.Write(data)
	if e == nil && n != len(data) {
		e = io.ErrShortWrite
	}
	return errors.Join(e, d.syncFile(f), f.Close(), d.sync())
}

func (d *journalDisk) remove(name string) error {
	if err := d.root.Remove(name); err != nil {
		return err
	}
	return d.sync()
}

func (d *journalDisk) read(name string, maximum int64) ([]byte, error) {
	s, err := d.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() || s.Size() <= 0 || s.Size() > maximum {
		return nil, ErrJournal
	}
	f, err := d.root.Open(name)
	if err != nil {
		return nil, err
	}
	b, e := io.ReadAll(io.LimitReader(f, maximum+1))
	closeErr := f.Close()
	if e != nil || closeErr != nil || int64(len(b)) != s.Size() {
		return nil, errors.Join(ErrJournal, e, closeErr)
	}
	return b, nil
}

func (d *journalDisk) batches(maximum int) ([]string, error) {
	f, err := d.root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Bound directory inventory before allocating an unbounded slice.
	names, err := f.Readdirnames(maximum + 4)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(names) >= maximum+4 {
		return nil, ErrJournal
	}
	var batches []string
	for _, name := range names {
		if name == "LOCK" || name == "CONFIG" {
			continue
		}
		if !strings.HasPrefix(name, "batch-") || len(name) != 26 {
			return nil, ErrJournal
		}
		batches = append(batches, name)
	}
	if len(batches) > maximum {
		return nil, ErrJournal
	}
	sort.Strings(batches)
	return batches, nil
}

func (d *journalDisk) close() error {
	return errors.Join(d.remove("LOCK"), d.root.Close())
}

// Deterministic, CLOSED local storage JSON is not an RFC8785/wire contract.
// Bound container depth before typed parsing; exact re-encoding rejects unknown
// or duplicate fields, alternate escaping/number spelling and trailing values.
func journalDecode(raw []byte, out any, config JournalConfig) error {
	if journalScan(raw, config) != nil || json.Unmarshal(raw, out) != nil {
		return ErrJournal
	}
	canonical, err := json.Marshal(out)
	if err != nil || !bytes.Equal(raw, canonical) {
		return ErrJournal
	}
	return nil
}

func journalScan(raw []byte, config JournalConfig) error {
	// Streaming token parsing does not construct generic arrays or maps. Bound
	// configured collection sizes BEFORE the typed decoder allocates slices.
	limits := map[string]int{"Blocks": config.Replay.MaxBlocks, "History": int(config.Replay.History.MaxItems), "Parents": int(config.Observation.MaxAncestryHeaders), "Transactions": config.Replay.MaxTransactions, "Receipts": config.Replay.MaxTransactions, "Executions": config.Replay.MaxTransactions, "Trace": config.Replay.MaxTransactions, "Calls": config.Replay.MaxFrames, "Logs": config.Replay.MaxLogs, "Topics": 4}
	type container struct {
		kind         json.Delim
		key          string
		wantKey      bool
		count, limit int
	}
	stack := []container{}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	for tokens := 0; tokens < 1<<22; tokens++ {
		token, err := d.Token()
		if err == io.EOF {
			if len(stack) == 0 {
				return nil
			}
			return ErrJournal
		}
		if err != nil {
			return ErrJournal
		}
		delim, isDelim := token.(json.Delim)
		if isDelim && (delim == '}' || delim == ']') {
			if len(stack) == 0 {
				return ErrJournal
			}
			stack = stack[:len(stack)-1]
			continue
		}
		key := ""
		if len(stack) > 0 {
			p := &stack[len(stack)-1]
			if p.kind == '{' && p.wantKey {
				s, ok := token.(string)
				if !ok {
					return ErrJournal
				}
				p.key = s
				p.wantKey = false
				p.count++
				if p.count > 32 {
					return ErrJournal
				}
				continue
			}
			if p.kind == '[' {
				p.count++
				if p.count > p.limit {
					return ErrJournal
				}
			} else {
				key = p.key
				p.wantKey = true
			}
		}
		if isDelim {
			if len(stack) >= 96 {
				return ErrJournal
			}
			stack = append(stack, container{kind: delim, wantKey: delim == '{', limit: limits[key]})
		}
	}
	return ErrJournal
}
