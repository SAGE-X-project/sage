package registry011

import (
	"bytes"
	"errors"
	"sync"
)

var ErrProviderRetention = errors.New("registry011: candidate retaining namespace unavailable")
var ErrProviderRetentionConflict = errors.New("registry011: candidate retaining state conflict")

// CandidateNamespaceConfig contains proposed trusted Agent integration inputs,
// not authenticated configuration just because its digests parse. AnchorDigest
// and ExpectedTip MUST come from an independently retained selection boundary,
// never peer/RPC/prompt data or a missing candidate directory. No defaults exist.
// MaxBytes bounds each evidence file, at most 64 KiB. LOCK is a fixed 7 bytes.
// Operating authority/custody/filesystem qualification remains separate.
type CandidateNamespaceConfig struct {
	Directory, Namespace, Instance, AnchorDigest, ExpectedTip string
	MaxBytes                                                  int
}

// CandidateProviderNamespace owns only first-enrollment persistence. It never
// provisions an authority, steals a lock, activates policy, changes a generation
// or authorizes protected use. Its receipts are not accepted by any Source API.
// The directory must be trusted against external writers/mount/hard-link drift;
// os.Root confinement and matching hashes do not prove that trust assumption.
type CandidateProviderNamespace struct {
	mu       sync.Mutex
	disk     *journalDisk
	config   CandidateNamespaceConfig
	tip      string
	root     string
	closed   bool
	fenced   bool
	closeErr error
}

// CandidateNamespaceSnapshot is a private owner/tip-bound read, not proof of
// authenticated absence or a publication capability. Zero and JSON restoration
// refuse. It cannot be used by another namespace handle or after Close/failure.
type CandidateNamespaceSnapshot struct {
	owner     *CandidateProviderNamespace
	tip, root string
}

// CandidateNamespaceDiagnostic is copied, forgeable diagnostic data only.
type CandidateNamespaceDiagnostic struct {
	Namespace, Instance, Tip string
	Enrollment               []byte
}

// CandidateRetentionReceipt records this owned component's exact durable
// operation. It is NOT a qualified external acknowledgement or Source token.
type CandidateRetentionReceipt struct {
	owner       *CandidateProviderNamespace
	identity    ProviderCandidateIdentity
	oldTip, tip string
	root        string
}

// CandidateEnrollmentDiagnostic cannot select authority or authorize use.
type CandidateEnrollmentDiagnostic struct {
	Identity                        ProviderCandidateIdentity
	ExpectedOldTip, Tip, RootDigest string
	Enrollment                      []byte
	Fence                           uint64
}

func (*CandidateProviderNamespace) UnmarshalJSON([]byte) error { return ErrProviderRetention }
func (*CandidateNamespaceSnapshot) UnmarshalJSON([]byte) error { return ErrProviderRetention }
func (*CandidateRetentionReceipt) UnmarshalJSON([]byte) error  { return ErrProviderRetention }

// OpenCandidateProviderNamespace opens an existing pinned namespace. It does
// not create Directory or ANCHOR, infer first use, or reset an expected tip.
// A stale LOCK, missing/unknown/partial file or mismatching high-water pin
// refuses. The initial fixture provision procedure is not a production API.
func OpenCandidateProviderNamespace(config CandidateNamespaceConfig) (*CandidateProviderNamespace, error) {
	n, _, err := openCandidateProviderNamespace(config, nil)
	return n, err
}

// RecoverCandidateProviderEnrollment explicitly recovers only the identical
// retained first operation after a lost acknowledgement and released lock.
// It never truncates, overwrites, recreates evidence or bypasses a crash lock.
// Actual power-failure/stale-lock recovery and operating qualification remain
// unsupported. Returned data cannot activate the proposed policy.
func RecoverCandidateProviderEnrollment(config CandidateNamespaceConfig, candidate *ProviderCandidateContext) (*CandidateProviderNamespace, *CandidateRetentionReceipt, error) {
	id, err := candidate.Identity()
	if err != nil || id.Namespace != config.Namespace || id.Instance != config.Instance || id.Generation != 0 {
		return nil, nil, ErrConfiguration
	}
	return openCandidateProviderNamespace(config, &id)
}

// ReadCandidate reloads the bounded exact state before creating a private read.
func (n *CandidateProviderNamespace) ReadCandidate() (*CandidateNamespaceSnapshot, error) {
	if n == nil {
		return nil, ErrProviderRetention
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.usable() {
		return nil, ErrProviderRetention
	}
	state, err := loadCandidateNamespace(n.disk, n.config)
	if err != nil || state.tip != n.tip || string(state.root) != n.root {
		n.fenced = true
		return nil, ErrProviderRetention
	}
	return &CandidateNamespaceSnapshot{n, n.tip, n.root}, nil
}

// EnrollCandidate permits exactly one initial append under the supplied pins.
// It does not qualify the proposed policy/Source identity or grant authority.
func (n *CandidateProviderNamespace) EnrollCandidate(snapshot *CandidateNamespaceSnapshot, candidate *ProviderCandidateContext) (*CandidateRetentionReceipt, error) {
	if n == nil {
		return nil, ErrProviderRetention
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.usable() {
		return nil, ErrProviderRetention
	}
	if snapshot == nil || snapshot.owner != n || snapshot.tip != n.tip || snapshot.root != n.root {
		return nil, ErrProviderRetentionConflict
	}
	id, err := candidate.Identity()
	if err != nil || id.Namespace != n.config.Namespace || id.Instance != n.config.Instance || id.Generation != 0 {
		return nil, ErrConfiguration
	}
	state, err := loadCandidateNamespace(n.disk, n.config)
	if err != nil || state.tip != n.tip || string(state.root) != n.root {
		n.fenced = true
		return nil, ErrProviderRetention
	}
	if n.root != "" || n.tip != n.config.AnchorDigest {
		return nil, ErrProviderRetentionConflict
	}
	root, entry, err := makeCandidateEnrollment(id, n.tip)
	if err != nil || len(entry) > n.config.MaxBytes {
		return nil, ErrProviderRetention
	}
	oldTip := n.tip
	newTip := providerCandidateHash("sage-provider-namespace-entry-v1", entry)
	if n.disk.write("ENROLLMENT", entry) != nil {
		n.fenced = true
		return nil, ErrProviderRetention
	}
	confirmed, err := loadCandidateNamespace(n.disk, n.config)
	if err != nil || confirmed.tip != newTip || !bytes.Equal(confirmed.root, root) || !bytes.Equal(confirmed.entry, entry) {
		n.fenced = true
		return nil, ErrProviderRetention
	}
	n.tip, n.root = newTip, string(root)
	return &CandidateRetentionReceipt{n, id, oldTip, newTip, n.root}, nil
}

// Close releases only this handle's lock. It never deletes retained evidence.
// A failed close remains an error and cannot confirm a successful commit.
func (n *CandidateProviderNamespace) Close() error {
	if n == nil {
		return nil
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return n.closeErr
	}
	n.closed = true
	if n.disk != nil {
		n.closeErr = n.disk.close()
	}
	return n.closeErr
}

func (n *CandidateProviderNamespace) usable() bool { return n.disk != nil && !n.closed && !n.fenced }

func (s *CandidateNamespaceSnapshot) Diagnostic() (CandidateNamespaceDiagnostic, error) {
	if s == nil || s.owner == nil {
		return CandidateNamespaceDiagnostic{}, ErrProviderRetention
	}
	n := s.owner
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.usable() || s.tip != n.tip || s.root != n.root {
		return CandidateNamespaceDiagnostic{}, ErrProviderRetentionConflict
	}
	return CandidateNamespaceDiagnostic{n.config.Namespace, n.config.Instance, s.tip, []byte(s.root)}, nil
}

func (r *CandidateRetentionReceipt) Diagnostic() (CandidateEnrollmentDiagnostic, error) {
	if r == nil || r.owner == nil {
		return CandidateEnrollmentDiagnostic{}, ErrProviderRetention
	}
	// This is historical operation metadata, usable even after clean Close.
	// It is never a fresh observation or an accepted Source acknowledgement.
	return CandidateEnrollmentDiagnostic{r.identity, r.oldTip, r.tip, providerCandidateHash("sage-provider-enrollment-v1", []byte(r.root)), []byte(r.root), 1}, nil
}
