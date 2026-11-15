package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"time"
)

// This is the concrete private pipeline behind a future qualified Agent
// selection. No exported constructor takes candidate/fixture authority. Its
// existence, a normal fixture run or an ACK does not select operating policy.
type providerIntegration struct {
	mu, command              sync.Mutex
	owner                    *providerIntegration
	store                    *providerCustodyStore
	policy                   *ProviderPolicyCandidate
	context                  *ProviderCandidateContext
	clock                    *providerOwnedClock
	primitive                providerPrimitiveBounds
	anchor, root             *CandidateEnrollmentRoot
	intent                   *CandidateAdmissionIntent
	ledger                   *CandidateAdmissionLedger
	first                    *providerIntegrationRequest
	firstTicket              *CandidateAdmissionTicket
	firstExchange, bootstrap string
	lastExchange, pin        string
	firstResult              string
	selected                 FinalizedHeader
	preparations             int
	failed                   bool
	meters                   map[string]*providerWorkMeter
}

type providerIntegrationPermit struct {
	owner   *providerIntegration
	ticket  *CandidateAdmissionTicket
	ack     *providerCustodyAcknowledgment
	root    *CandidateEnrollmentRoot
	request *providerIntegrationRequest
	clock   ClockSample
	meter   *providerWorkMeter
	used    bool
}

func (*providerIntegration) UnmarshalJSON([]byte) error       { return errProviderIntegration }
func (*providerIntegrationPermit) UnmarshalJSON([]byte) error { return errProviderIntegration }

// No reopen may create a fresh admission, identity, initial clock or permit.
// A complete retained prefix can be inspected by the store, but recovery of a
// consumed operation never calls this initial-only live constructor.
func newProviderIntegration(store *providerCustodyStore, policy *ProviderPolicyCandidate, candidate *ProviderCandidateContext, primitive providerPrimitiveBounds) (*providerIntegration, error) {
	if store == nil || store.config.bounds.history != nil || store.history != nil {
		return nil, ErrConfiguration
	}
	id, err := candidate.Identity()
	if err != nil || policy == nil || policy.raw == "" || !store.valid() || store.enrollment != "" || store.sequence != 0 || store.config.identity != id || id.Policy != policy.policy || !primitive.capture.valid() || primitive.capture != store.config.capture || primitive.maximumCreatedObjects < 32 || primitive.maximumCreatedObjects > uint64(store.config.bounds.maximumEntries) {
		return nil, ErrConfiguration
	}
	c := &providerIntegration{store: store, policy: policy, context: candidate, clock: store.clock, primitive: primitive, meters: make(map[string]*providerWorkMeter)}
	c.owner = c
	var success bool
	defer func() {
		if !success {
			c.failed = true
		}
	}()
	initial := providerIntegrationJSON(map[string]any{"format": "sage-provider-enrollment-v2", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": "0", "sequence": "0", "previous_root": nil, "status": "enrolling", "checkpoint": nil, "budget_root": nil, "transition_root": nil, "fence": "1"})
	c.anchor, err = ParseCandidateEnrollmentRoot(candidate, initial, 64<<10)
	if err != nil {
		return nil, errProviderIntegration
	}
	c.root = c.anchor
	setup, err := store.read(store.evidence, providerCustodyName("setup-claim", uint64(store.config.attempt)), int64(store.config.bounds.maximumMetadataPerCommit))
	setupCommit, commitErr := store.read(store.custody, providerCustodyName("setup-commit", uint64(store.config.attempt)), int64(store.config.bounds.maximumMetadataPerCommit))
	objects, objectErr := c.objects([]providerIntegrationObjectInput{{"enrollment", initial}, {"setup", setup}, {"setup-commit", setupCommit}})
	if err != nil || commitErr != nil || objectErr != nil {
		return nil, errProviderIntegration
	}
	operation := providerCandidateHash("sage-provider-enroll-operation-v1", []byte(store.config.anchor+id.Instance+id.Policy+id.SourceID))
	ack, err := store.commit("enroll", operation, c.root, objects)
	if err != nil || !store.confirms(ack, operation, c.root) {
		return nil, errProviderIntegration
	}
	if err := c.makeIntent(ack.commit); err != nil {
		return nil, err
	}
	c.first, err = newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil || !c.first.firstFinalized() {
		return nil, errProviderIntegration
	}
	c.firstTicket, err = c.admitRPC(c.first, true)
	if err != nil {
		return nil, err
	}
	success = true
	return c, nil
}

func (c *providerIntegration) makeIntent(authorityRoot string) error {
	if c == nil || c.store == nil || c.store.config.bounds.history != nil || c.store.history != nil {
		return errProviderIntegration
	}
	p := contentJSON([]byte(c.policy.raw)).(map[string]any)
	limits := p["limits"].(map[string]any)
	duration, _ := providerCandidateUint(limits["bootstrap"].(map[string]any)["deadline_ms"], true)
	clock := c.store.setupOrigin
	if clock.WallMS < clock.UncertaintyMS || duration > math.MaxInt64-uint64(clock.WallMS-clock.UncertaintyMS) {
		return errProviderIntegration
	}
	deadline := uint64(clock.WallMS-clock.UncertaintyMS) + duration
	limitsBytes := providerIntegrationJSON(limits)
	limitsDigest := sha256.Sum256(limitsBytes)
	id := c.store.config.identity
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-admission-intent-v1", "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": "1", "admission_id": providerCandidateHash("sage-provider-admission-id-v1", []byte(authorityRoot+id.Instance)), "enrollment_anchor_root": c.anchor.digest, "prior_published_checkpoint": nil, "initial_clock": providerClockObject(clock), "deadline_wall_ms": strconv.FormatUint(deadline, 10), "limits_digest": "0x" + hex.EncodeToString(limitsDigest[:]), "authority_metadata_root": authorityRoot})
	intent, err := ParseCandidateAdmissionIntent(c.policy, c.context, c.anchor, raw, 64<<10)
	if err != nil {
		return errProviderIntegration
	}
	c.intent = intent
	return nil
}

type providerIntegrationObjectInput struct {
	kind string
	raw  []byte
}

func (c *providerIntegration) objects(inputs []providerIntegrationObjectInput) ([]providerIntegrationObject, error) {
	if c == nil || c.store == nil || c.store.config.bounds.history != nil || c.store.history != nil || len(inputs) == 0 || len(inputs) > c.store.config.bounds.maximumEntries {
		return nil, errProviderIntegration
	}
	objects := make([]providerIntegrationObject, 0, len(inputs))
	for _, input := range inputs {
		domain, known := providerIntegrationDomain(input.kind)
		if !known || !c.store.event(providerHashEvent, uint64(len(input.raw)+len(domain))) || !c.store.event(providerCopyEvent, uint64(len(input.raw))) || domain != "" && !c.store.event(providerCodecEvent, uint64(len(input.raw))) {
			return nil, errProviderIntegration
		}
		object, err := newProviderIntegrationObject(input.kind, input.raw, c.store.config.bounds.maximumObjectBytes)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (c *providerIntegration) valid() bool {
	return c != nil && c.owner == c && !c.failed && c.store.valid() && c.policy != nil && c.intent != nil && c.root != nil && c.clock != nil
}

// First-operation observers still use valid/currentClock. Generic admission,
// clients and local commands must refuse the whole-program profile before
// touching a meter, nil generic cache, clock, serializer or retaining writer.
func (c *providerIntegration) legacyAllowed() bool {
	return c != nil && c.store != nil && c.store.config.bounds.history == nil && c.store.history == nil && c.valid()
}

// All preparation/validation is also bounded by a fully charged finite setup
// envelope, even when a proposed ticket is refused before it can fund work.
// A new handle cannot replenish it: initial enrollment has already been used.
func (c *providerIntegration) prepare() error {
	if !c.legacyAllowed() || c.preparations >= c.store.config.bounds.maximumCommits {
		return errProviderIntegration
	}
	c.preparations++
	return nil
}

func (c *providerIntegration) currentClock() (ClockSample, error) {
	if c == nil || c.store == nil {
		return ClockSample{}, errProviderIntegration
	}
	if c.store.config.bounds.history != nil {
		o, meter := c.store.firstOperation, c.store.activeMeter.Load()
		if o == nil || o.integration != c || !o.selectedLocked() || meter == nil || meter.firstOperation != o || meter.integration != c {
			return ClockSample{}, errProviderIntegration
		}
	}
	clock, err := c.store.sample()
	if err != nil || !candidateLedgerClockAdvance(c.intent.clock, providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), c.intent.deadline) {
		return ClockSample{}, errProviderIntegration
	}
	return clock, nil
}

// A gated clone never escapes its concrete adapter, and cannot be nested into
// the older acquisition constructor. The existing nil-gate V1 path is intact.
func (c *providerIntegration) readClient() (*ReadRPC, error) {
	if !c.legacyAllowed() || c.primitive.capture.BodyBytes <= 0 {
		return nil, errProviderIntegration
	}
	client, err := NewSepoliaReadRPC(RPCTransportLimits{c.primitive.capture.Timeout, c.primitive.capture.BodyBytes, c.primitive.capture.JSONDepth})
	if err != nil {
		return nil, err
	}
	client.acquisition = &rpcAcquisition{integration: c}
	return client, nil
}

func (c *providerIntegration) execute(ctx context.Context, method string, params any, maximum int) (json.RawMessage, json.RawMessage, error) {
	if c == nil {
		return nil, nil, errProviderIntegration
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx == nil || ctx.Err() != nil || maximum <= 0 || c.prepare() != nil {
		return nil, nil, errProviderIntegration
	}
	request, err := newProviderIntegrationRequest(method, params)
	if err != nil {
		return nil, nil, err
	}
	primitive, err := c.operationPrimitive(request)
	if err != nil {
		return nil, nil, err
	}
	var ticket *CandidateAdmissionTicket
	if c.firstExchange == "" {
		if !request.firstFinalized() || request.inputDigest != c.first.inputDigest || c.firstTicket == nil || c.ledger.lastConsumption != nil {
			return nil, nil, errProviderIntegration
		}
		ticket = c.firstTicket
	} else {
		ticket, err = c.admitRPC(request, false)
		if err != nil {
			return nil, nil, err
		}
	}
	permit, err := c.consume(ticket, request)
	if err != nil {
		c.failed = true
		return nil, nil, err
	}
	defer c.restoreMeter(permit.meter)
	// Set the irreversible one-use mark before dial, TLS or any request bytes.
	if !c.use(permit) {
		c.failed = true
		return nil, nil, errProviderIntegration
	}
	remaining := permit.ticket.deadline - uint64(permit.clock.WallMS) - uint64(permit.clock.UncertaintyMS)
	child, cancel := context.WithTimeout(ctx, min(primitive.capture.Timeout, time.Duration(remaining)*time.Millisecond))
	defer cancel()
	conn, dialErr := providerIntegrationDial(child, primitive.capture)
	var sample *ProviderTransportSample
	if dialErr == nil {
		limits := primitive.capture
		limits.BodyBytes = min(limits.BodyBytes, maximum)
		if c.store.config.bounds.continuation == nil {
			sample, err = providerIntegrationCapture(child, conn, limits, request, permit.meter)
		} else {
			sample, err = providerIntegrationCaptureWithBounds(child, conn, limits, request, permit.meter, *c.store.config.bounds.tls)
		}
		_ = conn.Close()
	} else {
		err = dialErr
	}
	return c.finishRPC(permit, sample, err)
}
