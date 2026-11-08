package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"
)

const historyPreparationResultFormat = "sage-registry-history-prepared-data-v1"

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
	compiler                    *providerHistorySourceCompiler
	maximumInput, maximumResult int
	used                        atomic.Bool
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

// Prepare consumes the session before copying, parsing or validating input.
// Refusal and cancellation cannot retry with new limits or a renewed lifetime.
// The original compiler freezes and compiles the config built from these exact
// bytes. No candidate store, private issuer or protected-use path is opened.
func (s *HistoryPreparationSession) Prepare(input []byte) ([]byte, error) {
	if s == nil || s.self != s || !s.used.CompareAndSwap(false, true) || s.compiler == nil {
		return nil, ErrConfiguration
	}
	if err := s.compiler.contextError(); err != nil {
		return nil, err
	}
	if len(input) == 0 || len(input) > s.maximumInput {
		return nil, ErrConfiguration
	}
	defer func() { s.compiler = nil }()
	owned := bytes.Clone(input)
	decoded, err := decodeHistoryPreparationInput(s.compiler.ctx, owned)
	if err != nil {
		return nil, fmt.Errorf("registry history preparation input: %w", err)
	}
	config, err := buildHistoryPreparationConfig(decoded)
	if err != nil {
		return nil, fmt.Errorf("registry history preparation configuration: %w", err)
	}
	plan, err := s.compiler.compile(config, decoded.Clock.Maximum, decoded.Clock.BaseUncertaintyMS, decoded.Clock.ReadWindowMS)
	if err != nil {
		return nil, fmt.Errorf("registry history preparation compilation: %w", err)
	}
	digest := sha256.Sum256(owned)
	summary := HistoryPreparationSummary{
		Format: historyPreparationResultFormat, InputSHA256: hex.EncodeToString(digest[:]),
		Identity: plan.config.identity, ManifestSHA256: plan.config.bounds.history.collection.digest,
		AssignedWork: plan.source.assigned.work, AssignedClockReads: plan.source.assigned.clocks,
		OriginUnixNano: s.compiler.origin.UnixNano(), DeadlineUnixNano: plan.compilerDeadline.UnixNano(),
		SourceReady: false,
	}
	output, err := json.Marshal(summary)
	if err != nil || len(output) > s.maximumResult {
		return nil, ErrConfiguration
	}
	if err := s.compiler.contextError(); err != nil {
		return nil, err
	}
	return output, nil
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
