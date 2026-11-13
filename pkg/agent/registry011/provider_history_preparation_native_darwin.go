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
	manifest        nativeHistoryOwner
	result          nativeHistoryOwner
	resultSpan      compilermemory.Span
	resultPlan      *providerHistorySourcePlan
	selection       *nativeHistoryOwnedSelection
}

// NewNativeHistoryPreparationSession selects the actual native DATA path. It
// retains the original cleanup context and compiler before input is consumed.
// Its retained input/policy/manifest DATA stays native. Temporary Go sections,
// the original numeric compiler, clock and bootstrap remain separate resource
// obligations. Neither this session nor diagnostics grant execution authority.
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
	if n.selection != nil && n.selection.plan != nil {
		bindingErr := n.selection.checkRetainedPlan(n.selection.plan)
		numbersErr := n.selection.numbers.close()
		if bindingErr == nil {
			return numbersErr
		}
		// A failed exact-plan binding cannot preserve retained owners.
		rootErr, policyErr, manifestErr := n.root.close(), n.policy.close(), n.manifest.close()
		return errors.Join(bindingErr, numbersErr, rootErr, policyErr, manifestErr)
	}
	rootErr, policyErr, manifestErr := n.root.close(), n.policy.close(), n.manifest.close()
	var numbersErr error
	if n.selection != nil {
		numbersErr = n.selection.numbers.close()
	}
	return errors.Join(rootErr, policyErr, manifestErr, numbersErr)
}

func (n *nativeHistoryPreparation) closeAll() error {
	rootErr, policyErr, manifestErr := n.root.close(), n.policy.close(), n.manifest.close()
	var numbersErr error
	if n.selection != nil {
		numbersErr = n.selection.numbers.close()
	}
	resultErr := n.result.close()
	err := errors.Join(rootErr, policyErr, manifestErr, numbersErr, resultErr)
	if n.selection != nil {
		if n.selection.plan != nil && n.selection.plan.retained == n.selection {
			n.selection.plan.retained = nil
		}
		n.selection.closed = true
		n.selection.session, n.selection.preparer, n.selection.compiler, n.selection.policy = nil, nil, nil, nil
		n.selection.document = compilermemory.JSONDocument{}
		n.selection.input, n.selection.inputHash = compilermemory.Span{}, ""
		n.selection.plan, n.selection.rootOwner, n.selection.policyOwner = nil, nil, nil
		n.selection.manifestOwner, n.selection.resultOwner = nil, nil
		n.selection.manifestSpan, n.selection.resultSpan = compilermemory.Span{}, compilermemory.Span{}
	}
	// Keep each owner's closed state and cached error, but drop the original
	// Go plan/compiler/context only after every required Close was attempted.
	n.resultPlan, n.compiler, n.owner, n.cleanup, n.acquire = nil, nil, nil, nil, nil
	n.resultSpan = compilermemory.Span{}
	n.selection = nil
	if err == nil {
		return nil
	}
	return errors.Join(ErrHistoryPreparationCleanup, err)
}

func (n *nativeHistoryPreparation) compileLocked(s *HistoryPreparationSession, input []byte) error {
	selection, err := n.decodeOwnedSelection(s, input)
	if err != nil {
		return err
	}
	plan, err := s.compiler.compileSelection(selection)
	if err != nil {
		return err
	}
	s.plan, s.inputSHA256 = plan, selection.inputHash
	return nil
}
