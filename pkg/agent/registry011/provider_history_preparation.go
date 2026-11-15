package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

const historyPreparationResultFormat = "sage-registry-history-prepared-data-v1"

// ErrHistoryPreparationCleanup preserves native cleanup refusal independently
// of parsing/compilation errors. A diagnostic result must not survive it.
var ErrHistoryPreparationCleanup = errors.New("registry history preparation: native cleanup refused")

// This private seam permits lifecycle fault units without exposing a backend
// selector. Public native construction fixes the actual platform implementation.
type nativeHistoryImplementation interface {
	compileLocked(*HistoryPreparationSession, []byte) error
	readResultLocked(*HistoryPreparationSession) (HistoryPreparationSummary, error)
	closeTemporary() error
	closeAll() error
}

// HistoryPreparationSummary is detached diagnostic DATA. Even a successful
// compile grants no native resources, private derivation, store access, source
// readiness, custody acknowledgement or message-dispatch permission.
type HistoryPreparationSummary struct {
	Format             string                    `json:"format"`
	InputSHA256        string                    `json:"input_sha256"`
	Identity           ProviderCandidateIdentity `json:"identity"`
	ManifestSHA256     string                    `json:"manifest_sha256"`
	AssignedWork       [11]uint64                `json:"assigned_work"`
	AssignedClockReads uint64                    `json:"assigned_clock_reads"`
	OriginUnixNano     int64                     `json:"origin_unix_nano"`
	DeadlineUnixNano   int64                     `json:"deadline_unix_nano"`
	SourceReady        bool                      `json:"source_ready"`
}

// HistoryPreparationSession owns one original compiler before input decoding.
// The embedding worker must independently enforce this context's process
// lifetime and fund preparation. This constructor is not resource admission.
type HistoryPreparationSession struct {
	self                        *HistoryPreparationSession
	mu                          sync.Mutex
	compiler                    *providerHistorySourceCompiler
	plan                        *providerHistorySourcePlan
	inputSHA256                 string
	maximumInput, maximumResult int
	used                        atomic.Bool
	closed                      bool
	native                      nativeHistoryImplementation
	cleanupErr                  error
}

func (*HistoryPreparationSession) UnmarshalJSON([]byte) error { return ErrConfiguration }

// NewHistoryPreparationSession retains the original attempt's origin and
// finite action context. Calling it after preparing the input cannot qualify
// that earlier work. The fixed child creates it before reading standard input.
func NewHistoryPreparationSession(ctx context.Context, origin time.Time, maximumInput, maximumResult int) (*HistoryPreparationSession, error) {
	if maximumInput <= 0 || maximumInput > 64<<20 || maximumResult <= 0 || maximumResult > 64<<20 {
		return nil, ErrConfiguration
	}
	compiler, err := newProviderHistorySourceCompilerAtOrigin(ctx, origin)
	if err != nil {
		return nil, err
	}
	session := &HistoryPreparationSession{compiler: compiler, maximumInput: maximumInput, maximumResult: maximumResult}
	session.self = session
	return session, nil
}

// Prepare preserves the one-call diagnostic API. It compiles and consumes the
// same session's private selection without exposing a plan or execution grant.
func (s *HistoryPreparationSession) Prepare(input []byte) ([]byte, error) {
	if err := s.CompileOwned(input); err != nil {
		return nil, err
	}
	return s.FinishDiagnostic()
}

// CompileOwned consumes the original attempt before copying or parsing input.
// Its detached, unqualified plan stays private to this same session until
// FinishDiagnostic or Close. The caller may release its input after return.
// This does not reserve compiler memory, open a store or issue a derivation.
func (s *HistoryPreparationSession) CompileOwned(input []byte) (err error) {
	if s == nil || s.self != s {
		return ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.used.CompareAndSwap(false, true) || s.closed || s.compiler == nil {
		return ErrConfiguration
	}
	defer func() {
		var temporaryErr error
		if s.native != nil {
			temporaryErr = s.native.closeTemporary()
		}
		if err != nil || temporaryErr != nil {
			cleanupErr := s.discardLocked()
			if temporaryErr != nil && cleanupErr == nil {
				cleanupErr = errors.Join(ErrHistoryPreparationCleanup, temporaryErr)
				s.cleanupErr = cleanupErr
			}
			if cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	if err := s.compiler.contextError(); err != nil {
		return err
	}
	if len(input) == 0 || len(input) > s.maximumInput {
		return ErrConfiguration
	}
	if s.native != nil {
		if err := s.native.compileLocked(s, input); err != nil {
			return err
		}
		if s.plan == nil || s.compiler == nil {
			return ErrConfiguration
		}
		return nil
	}
	owned := bytes.Clone(input)
	decoded, err := decodeHistoryPreparationInput(s.compiler.ctx, owned)
	if err != nil {
		return fmt.Errorf("registry history preparation input: %w", err)
	}
	config, err := buildHistoryPreparationConfig(decoded)
	if err != nil {
		return fmt.Errorf("registry history preparation configuration: %w", err)
	}
	plan, err := s.compiler.compile(config, decoded.Clock.Maximum, decoded.Clock.BaseUncertaintyMS, decoded.Clock.ReadWindowMS)
	if err != nil {
		return fmt.Errorf("registry history preparation compilation: %w", err)
	}
	digest := sha256.Sum256(owned)
	s.plan, s.inputSHA256 = plan, hex.EncodeToString(digest[:])
	return nil
}

// FinishDiagnostic consumes the retained plan once, under its original context
// and deadline. A failure discards it without retry or a fresh clock. The output
// remains detached DATA and cannot reconstruct the private plan or permission.
func (s *HistoryPreparationSession) FinishDiagnostic() (output []byte, err error) {
	if s == nil || s.self != s {
		return nil, ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.used.Load() {
		s.used.Store(true)
		if cleanupErr := s.discardLocked(); cleanupErr != nil {
			return nil, errors.Join(ErrConfiguration, cleanupErr)
		}
		return nil, ErrConfiguration
	}
	if s.closed || s.compiler == nil || s.plan == nil {
		if cleanupErr := s.discardLocked(); cleanupErr != nil {
			return nil, errors.Join(ErrConfiguration, cleanupErr)
		}
		return nil, ErrConfiguration
	}
	defer func() {
		if cleanupErr := s.discardLocked(); cleanupErr != nil {
			output, err = nil, errors.Join(err, cleanupErr)
		}
	}()
	if err := s.compiler.contextError(); err != nil {
		return nil, err
	}
	var summary HistoryPreparationSummary
	if s.native != nil {
		summary, err = s.native.readResultLocked(s)
		if err != nil {
			return nil, err
		}
	} else {
		plan := s.plan
		summary = HistoryPreparationSummary{
			Format: historyPreparationResultFormat, InputSHA256: s.inputSHA256,
			Identity: plan.config.identity, ManifestSHA256: plan.config.bounds.history.collection.digest,
			AssignedWork: plan.source.assigned.work, AssignedClockReads: plan.source.assigned.clocks,
			OriginUnixNano: s.compiler.origin.UnixNano(), DeadlineUnixNano: plan.compilerDeadline.UnixNano(),
			SourceReady: false,
		}
	}
	if summary.SourceReady {
		return nil, ErrConfiguration
	}
	output, err = json.Marshal(summary)
	if err != nil || len(output) > s.maximumResult {
		return nil, ErrConfiguration
	}
	if err := s.compiler.contextError(); err != nil {
		return nil, err
	}
	return output, nil
}

// Close permanently discards this session's references. It waits for an active
// CompileOwned or FinishDiagnostic call; cancellation still uses the original
// context and the fixed parent's process stop. Dropping Go references does not
// certify immediate physical memory release. Close is idempotent.
func (s *HistoryPreparationSession) Close() error {
	if s == nil || s.self != s {
		return ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.used.Store(true)
	return s.discardLocked()
}

func (s *HistoryPreparationSession) discardLocked() error {
	if s.closed {
		return s.cleanupErr
	}
	s.closed = true
	if s.native != nil {
		s.cleanupErr = s.native.closeAll()
	}
	s.compiler, s.plan, s.inputSHA256 = nil, nil, ""
	return s.cleanupErr
}

// DecodeHistoryPreparationSummary checks a bounded diagnostic result. Matching
// hashes and timestamps cannot turn serialized DATA into a live permission.
func DecodeHistoryPreparationSummary(ctx context.Context, raw, input []byte, maximum int) (HistoryPreparationSummary, error) {
	var empty HistoryPreparationSummary
	if maximum <= 0 || maximum > 64<<20 || len(raw) == 0 || len(raw) > maximum {
		return empty, ErrConfiguration
	}
	if err := validateHistoryPreparationJSON(ctx, raw, reflect.TypeFor[HistoryPreparationSummary]()); err != nil {
		return empty, err
	}
	var out HistoryPreparationSummary
	if err := json.Unmarshal(raw, &out); err != nil {
		return empty, ErrConfiguration
	}
	digest := sha256.Sum256(input)
	if out.Format != historyPreparationResultFormat || out.InputSHA256 != hex.EncodeToString(digest[:]) ||
		out.SourceReady || out.OriginUnixNano <= 0 || out.DeadlineUnixNano <= out.OriginUnixNano ||
		!lowerHash(out.ManifestSHA256) || !lowerHash(out.Identity.Policy) ||
		!lowerHash(out.Identity.Instance) || !lowerHash(out.Identity.Provider) ||
		!lowerHash(out.Identity.Namespace) || !lowerHash(out.Identity.SourceID) {
		return empty, ErrConfiguration
	}
	return out, nil
}
