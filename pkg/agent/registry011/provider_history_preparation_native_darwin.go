//go:build darwin

package registry011

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

type nativeHistoryAcquire func(context.Context, context.Context, int) (*compilermemory.Region, error)

type nativeHistoryOwner struct {
	region   *compilermemory.Region
	closed   bool
	closeErr error
}

func (o *nativeHistoryOwner) close() error {
	if !o.closed {
		o.closed = true
		if o.region != nil {
			o.closeErr = o.region.Close()
		}
	}
	return o.closeErr
}

type nativeHistoryPreparation struct {
	owner           *HistoryPreparationSession
	compiler        *providerHistorySourceCompiler
	cleanup         context.Context
	cleanupDeadline time.Time
	acquire         nativeHistoryAcquire
	schemaNodes     int
	root            nativeHistoryOwner
	policy          nativeHistoryOwner
	result          nativeHistoryOwner
	resultSpan      compilermemory.Span
	resultPlan      *providerHistorySourcePlan
}

// NewNativeHistoryPreparationSession selects the actual native DATA path. It
// retains the original cleanup context and compiler before input is consumed.
// Go DTOs, policy constructors and frozen config remain unqualified allocations;
// neither this session nor its diagnostic output grants execution authority.
func NewNativeHistoryPreparationSession(action, cleanup context.Context, origin time.Time, maximumInput, maximumResult int) (*HistoryPreparationSession, error) {
	return newNativeHistoryPreparationSession(action, cleanup, origin, maximumInput, maximumResult, compilermemory.Acquire)
}

// Only private failure units can replace acquisition. The public constructor
// always selects compilermemory.Acquire and never falls back to Go decoding.
func newNativeHistoryPreparationSession(action, cleanup context.Context, origin time.Time, maximumInput, maximumResult int, acquire nativeHistoryAcquire) (*HistoryPreparationSession, error) {
	if action == nil || cleanup == nil || acquire == nil {
		return nil, ErrConfiguration
	}
	actionDeadline, actionFinite := action.Deadline()
	cleanupDeadline, cleanupFinite := cleanup.Deadline()
	if !actionFinite || !cleanupFinite || cleanupDeadline.Before(actionDeadline) || cleanup.Err() != nil {
		return nil, ErrConfiguration
	}
	nodes, err := nativeHistorySchemaNodes(reflect.TypeFor[historyPreparationInput](), 0)
	if err != nil {
		return nil, err
	}
	session, err := NewHistoryPreparationSession(action, origin, maximumInput, maximumResult)
	if err != nil {
		return nil, err
	}
	session.native = &nativeHistoryPreparation{
		owner: session, compiler: session.compiler, cleanup: cleanup,
		cleanupDeadline: cleanupDeadline, acquire: acquire, schemaNodes: nodes,
	}
	return session, nil
}

func (n *nativeHistoryPreparation) acquireOwner(s *HistoryPreparationSession, owner *nativeHistoryOwner, capacity int) error {
	if n == nil || s == nil || s.compiler == nil || n.owner != s || n.compiler != s.compiler || owner == nil || owner.closed || owner.region != nil || capacity <= 0 {
		return ErrConfiguration
	}
	deadline, finite := n.cleanup.Deadline()
	if !finite || deadline != n.cleanupDeadline || n.cleanup.Err() != nil {
		return errors.Join(ErrHistoryPreparationCleanup, ErrConfiguration, n.cleanup.Err())
	}
	// Retain even a non-nil partial owner returned together with an error. The
	// checked session cleanup path must observe its cached Close result.
	var err error
	owner.region, err = n.acquire(s.compiler.ctx, n.cleanup, capacity)
	return err
}

func (n *nativeHistoryPreparation) closeTemporary() error {
	return errors.Join(n.root.close(), n.policy.close())
}

func (n *nativeHistoryPreparation) closeAll() error {
	err := errors.Join(n.root.close(), n.policy.close(), n.result.close())
	// Keep each owner's closed state and cached error, but drop the original
	// Go plan/compiler/context only after every required Close was attempted.
	n.resultPlan, n.compiler, n.owner, n.cleanup, n.acquire = nil, nil, nil, nil, nil
	n.resultSpan = compilermemory.Span{}
	if err == nil {
		return nil
	}
	return errors.Join(ErrHistoryPreparationCleanup, err)
}

func (n *nativeHistoryPreparation) compileLocked(s *HistoryPreparationSession, input []byte) error {
	decoded, inputDigest, err := n.decodeInput(s, input)
	if err != nil {
		return err
	}
	config, err := buildHistoryPreparationConfig(decoded)
	if err != nil {
		return err
	}
	plan, err := s.compiler.compile(config, decoded.Clock.Maximum, decoded.Clock.BaseUncertaintyMS, decoded.Clock.ReadWindowMS)
	if err != nil {
		return err
	}
	if err := n.storeResult(s, plan, inputDigest); err != nil {
		return err
	}
	s.plan, s.inputSHA256 = plan, inputDigest
	return nil
}
