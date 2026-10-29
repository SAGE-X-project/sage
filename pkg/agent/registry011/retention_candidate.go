package registry011

import (
	"bytes"
	"errors"
	"strconv"
	"sync"
)

var ErrCandidateRetention = errors.New("registry011: candidate retention unavailable")
var ErrRetentionConflict = errors.New("registry011: candidate retention conflict")
var ErrRetentionPlatform = errors.New("registry011: retaining platform unsupported")

// CandidateRetentionConfig describes a proposed Agent-local retention boundary.
// Namespace, AnchorDigest and ExpectedTip are supplied comparison inputs, NOT
// authenticated authority or proof of absence. An operating integration needs
// independently authenticated custody and a monotone external high-water CAS.
// No directory/anchor provisioning, policy selection or external ACK exists.
// File Sync, writer exclusion and rollback assumptions need separate host/store
// qualification; advisory locking does not exclude noncooperating writers.
type CandidateRetentionConfig struct {
	Directory, Namespace, AnchorDigest, ExpectedTip string
	MaxRecords                                      int
	MaxRecordBytes, MaxStoredBytes                  int64
}

// CandidateRetentionLog retains bounded, initial-enrollment candidate records.
// Each open may append at most one pending record. Local durability is not a
// final ACK: no API permits work, publishes a Source or updates external custody.
// Normal reopen compares the complete local tip to the supplied expected tip.
// That equality does not authenticate the external selection or its authority.
type CandidateRetentionLog struct {
	mu       sync.Mutex
	owner    *CandidateRetentionLog
	disk     *retentionDisk
	config   CandidateRetentionConfig
	state    retentionState
	closed   bool
	fenced   bool
	pending  bool
	closeErr error
}

// CandidateRetentionRead is an owner/prefix/instance-bound local observation.
// Its empty root is NOT authoritative absence. Unknown/partial state refuses.
type CandidateRetentionRead struct {
	owner    *CandidateRetentionLog
	tip      string
	instance string
	root     string
	sequence uint64
}

// CandidateRetentionPending describes one locally retained, unacknowledged
// operation. Even after recovery it is pending external authenticated custody.
// No Source, work, clock or message authority API accepts this diagnostic.
type CandidateRetentionPending struct {
	owner                         *CandidateRetentionLog
	instance, operation, previous string
	tip, root                     string
	sequence                      uint64
}

// CandidateRetentionDiagnostic is copied local evidence, never a final ACK.
type CandidateRetentionDiagnostic struct {
	Namespace, Instance, Operation, PreviousTip, Tip, RootDigest string
	State                                                        string
	Sequence                                                     uint64
	Root                                                         []byte
}

func (*CandidateRetentionLog) UnmarshalJSON([]byte) error     { return ErrCandidateRetention }
func (*CandidateRetentionRead) UnmarshalJSON([]byte) error    { return ErrCandidateRetention }
func (*CandidateRetentionPending) UnmarshalJSON([]byte) error { return ErrCandidateRetention }

// OpenCandidateRetentionLog never creates the directory or its ANCHOR.
// It rejects a different expected tip even if the retained prefix is valid.
// Callers cannot infer authoritative enrollment from a successful local open.
func OpenCandidateRetentionLog(config CandidateRetentionConfig) (*CandidateRetentionLog, error) {
	a, _, err := openCandidateRetentionLog(config, nil, "")
	return a, err
}

// RecoverCandidateRetentionEnrollment re-syncs only an identical pending final
// enrollment after an interrupted local acknowledgement. ExpectedTip must name
// its exact predecessor. The result stays pending, permits no further append,
// and does not advance or reconcile the independent external high-water.
// Partial/extra tails refuse without truncation, guessed absence or reset.
func RecoverCandidateRetentionEnrollment(config CandidateRetentionConfig, candidate *ProviderCandidateContext, operation string) (*CandidateRetentionLog, *CandidateRetentionPending, error) {
	id, err := candidate.Identity()
	if err != nil || id.Generation != 0 || id.Namespace != config.Namespace || !lowerHash(operation) {
		return nil, nil, ErrConfiguration
	}
	return openCandidateRetentionLog(config, &id, operation)
}

// ReadInstance validates the entire selected bounded prefix before reading an
// instance. A new policy, Source ID or candidate path cannot change that key.
func (a *CandidateRetentionLog) ReadInstance(instance string) (*CandidateRetentionRead, error) {
	if a == nil {
		return nil, ErrCandidateRetention
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.usable() || !lowerHash(instance) {
		return nil, ErrCandidateRetention
	}
	if err := a.revalidate(); err != nil {
		return nil, err
	}
	return &CandidateRetentionRead{a, a.state.tip, instance, a.state.roots[instance], a.state.sequence}, nil
}

// EnrollInstance compares a local empty-instance observation and appends one
// enrolling candidate. This is a local comparison, NOT an authoritative absence
// CAS or final enrollment ACK. Existing instances and operation IDs refuse.
// The pending result does not select policy or permit any protected work.
func (a *CandidateRetentionLog) EnrollInstance(read *CandidateRetentionRead, candidate *ProviderCandidateContext, operation string) (*CandidateRetentionPending, error) {
	if a == nil {
		return nil, ErrCandidateRetention
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.usable() || a.pending {
		return nil, ErrCandidateRetention
	}
	if read == nil || read.owner != a || read.tip != a.state.tip || read.sequence != a.state.sequence || read.root != "" || !lowerHash(operation) {
		return nil, ErrRetentionConflict
	}
	id, err := candidate.Identity()
	if err != nil || id.Namespace != a.config.Namespace || id.Instance != read.instance || id.Generation != 0 {
		return nil, ErrConfiguration
	}
	if err := a.revalidate(); err != nil {
		return nil, err
	}
	if a.state.roots[id.Instance] != "" || a.state.operations[operation] || a.state.sequence >= uint64(a.config.MaxRecords) {
		return nil, ErrRetentionConflict
	}
	root, _, err := makeCandidateEnrollment(id, a.config.AnchorDigest)
	if err != nil {
		return nil, ErrCandidateRetention
	}
	record := retentionRecord{
		Format: "sage-retaining-enrollment-v1", Namespace: a.config.Namespace,
		Sequence: strconv.FormatUint(a.state.sequence+1, 10), Previous: a.state.tip,
		Instance: id.Instance, Operation: operation, Root: root,
	}
	raw, err := retentionEncode(record)
	if err != nil || int64(len(raw)) > a.config.MaxRecordBytes || int64(len(raw)) > a.config.MaxStoredBytes-a.state.used {
		return nil, ErrCandidateRetention
	}
	previous := a.state.tip
	tip := providerCandidateHash("sage-retaining-entry-v1", raw)
	if err := a.disk.write(retentionName(a.state.sequence+1), raw); err != nil {
		a.fenced = true
		return nil, ErrCandidateRetention
	}
	next, err := loadCandidateRetentionLog(a.disk, a.config)
	if err != nil || next.tip != tip || next.sequence != a.state.sequence+1 || next.roots[id.Instance] != string(root) {
		a.fenced = true
		return nil, ErrCandidateRetention
	}
	a.state, a.pending = next, true
	return &CandidateRetentionPending{a, id.Instance, operation, previous, tip, string(root), next.sequence}, nil
}

func (a *CandidateRetentionLog) revalidate() error {
	next, err := loadCandidateRetentionLog(a.disk, a.config)
	if err != nil || next.tip != a.state.tip || next.sequence != a.state.sequence || next.used != a.state.used {
		a.fenced = true
		return ErrCandidateRetention
	}
	return nil
}

func (a *CandidateRetentionLog) usable() bool {
	return a.owner == a && a.disk != nil && !a.closed && !a.fenced
}

// Close releases the OS lock without deleting its inode or any retained data.
func (a *CandidateRetentionLog) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.owner != a {
		return ErrCandidateRetention
	}
	if a.closed {
		return a.closeErr
	}
	a.closed = true
	a.closeErr = a.disk.close()
	return a.closeErr
}

func (s *CandidateRetentionRead) Diagnostic() (CandidateRetentionDiagnostic, error) {
	if s == nil || s.owner == nil {
		return CandidateRetentionDiagnostic{}, ErrCandidateRetention
	}
	a := s.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.usable() || s.tip != a.state.tip || s.sequence != a.state.sequence {
		return CandidateRetentionDiagnostic{}, ErrRetentionConflict
	}
	if err := a.revalidate(); err != nil {
		return CandidateRetentionDiagnostic{}, err
	}
	return CandidateRetentionDiagnostic{Namespace: a.config.Namespace, Instance: s.instance, Tip: s.tip, Sequence: s.sequence, State: "local-observation", Root: []byte(s.root)}, nil
}

func (r *CandidateRetentionPending) Diagnostic() (CandidateRetentionDiagnostic, error) {
	if r == nil || r.owner == nil || r.owner.owner != r.owner || r.root == "" {
		return CandidateRetentionDiagnostic{}, ErrCandidateRetention
	}
	return CandidateRetentionDiagnostic{r.owner.config.Namespace, r.instance, r.operation, r.previous, r.tip, providerCandidateHash("sage-provider-enrollment-v1", []byte(r.root)), "pending-external-ack", r.sequence, []byte(r.root)}, nil
}

func sameRetentionRecord(a, b retentionRecord) bool {
	return a.Format == b.Format && a.Namespace == b.Namespace && a.Sequence == b.Sequence && a.Previous == b.Previous && a.Instance == b.Instance && a.Operation == b.Operation && bytes.Equal(a.Root, b.Root)
}
