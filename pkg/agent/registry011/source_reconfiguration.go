package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

var ErrReconfiguration = errors.New("registry011: reconfiguration candidate refused")

// ReconstructionLimits bounds one complete retained prefix. Every inspection
// and readback pass applies the same byte/batch ceiling; the number of passes
// is fixed by planning/rebuild. These are not an operating policy or activation.
type ReconstructionLimits struct {
	MaxBatches int
	MaxBytes   int64
}

// ReconfigurationPlan describes a candidate on the SAME accepted branch. It
// binds both configurations and checkpoints, but neither consumes its nonce
// nor enrolls Authority. Only a separately qualified administrative CAS store
// can consume the nonce, retain this audit and fence the old generation.
type ReconfigurationPlan struct {
	Format, Authority, Nonce, Reason string
	PreviousPolicy, NextPolicy       string
	PreviousConfig, NextConfig       string
	Previous, Next                   JournalCheckpoint
}

// ReconfigurationAudit must be durably installed with the old/new generation
// and checkpoints before activation. Returning it does not perform that step.
type ReconfigurationAudit struct {
	Plan                       ReconfigurationPlan
	AuthorityKey, Signature    string
	PreviousConfig, NextConfig JournalConfig
}

const reconfigurationFormat = "sage-registry-reconfiguration-candidate-v1"

func reconstructionConfig(previous, next JournalConfig, authority []byte, nonce, reason string, limits ReconstructionLimits) ([]byte, []byte, error) {
	a, _, e := journalConfiguration(previous)
	b, _, f := journalConfiguration(next)
	if e != nil || f != nil || !replayPrimeEdPoint(authority) || !lowerHash(nonce) || !reconfigurationText(reason, 1024) ||
		!reconfigurationText(previous.PolicyGeneration, 256) || !reconfigurationText(next.PolicyGeneration, 256) ||
		limits.MaxBatches <= 0 || limits.MaxBatches > 4096 || limits.MaxBytes <= 0 || limits.MaxBytes > 128<<20 ||
		previous.Registry != next.Registry || previous.Agent != next.Agent || previous.GenesisHash != next.GenesisHash || previous.SourceID != next.SourceID ||
		previous.CodeIdentity != next.CodeIdentity || previous.ABIIdentity != next.ABIIdentity || previous.Deployment != next.Deployment ||
		previous.Observation.ClockID != next.Observation.ClockID || previous.PolicyGeneration == next.PolicyGeneration {
		return nil, nil, ErrReconfiguration
	}
	return a, b, nil
}

func reconfigurationText(s string, maximum int) bool {
	if len(s) == 0 || len(s) > maximum || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func sourceSHA(raw []byte) string { h := sha256.Sum256(raw); return "0x" + hex.EncodeToString(h[:]) }

// ReconfigurationSigningMessage returns a domain-separated CLOSED local JSON
// message, not a SAGE wire/RFC8785 message. There is no signing/key-loading API.
func ReconfigurationSigningMessage(p ReconfigurationPlan) ([]byte, error) {
	if p.Format != reconfigurationFormat || !lowerHash(p.Authority) || !lowerHash(p.Nonce) || !reconfigurationText(p.Reason, 1024) ||
		!reconfigurationText(p.PreviousPolicy, 256) || !reconfigurationText(p.NextPolicy, 256) || p.PreviousPolicy == p.NextPolicy ||
		!lowerHash(p.PreviousConfig) || !lowerHash(p.NextConfig) || !lowerHash(p.Previous.Digest) || !lowerHash(p.Next.Digest) ||
		!validHeader(p.Previous.Anchor) || p.Previous.Anchor != p.Next.Anchor || p.Previous.Sequence != p.Next.Sequence || p.Previous.Version != p.Next.Version {
		return nil, ErrReconfiguration
	}
	b, e := json.Marshal(p)
	if e != nil {
		return nil, e
	}
	return append([]byte(reconfigurationFormat+"\x00"), b...), nil
}

// PlanJournalReconfiguration reads the externally acknowledged prefix without
// changing its LOCK/PENDING/QUARANTINED markers. A caller-supplied PUBLIC key is
// only a verification key; this does not enroll an administrator or stop a writer.
func PlanJournalReconfiguration(ctx context.Context, directory string, previous, next JournalConfig, expected JournalCheckpoint, authority []byte, nonce, reason string, limits ReconstructionLimits) (ReconfigurationPlan, error) {
	var none ReconfigurationPlan
	a, b, e := reconstructionConfig(previous, next, authority, nonce, reason, limits)
	if e != nil || ctx == nil {
		return none, ErrReconfiguration
	}
	tip, e := reconstructPrefix(ctx, directory, previous, next, expected, limits, nil)
	if e != nil {
		return none, e
	}
	return ReconfigurationPlan{reconfigurationFormat, sourceSHA(authority), nonce, reason, previous.PolicyGeneration, next.PolicyGeneration, journalDigest(a), journalDigest(b), expected, tip}, nil
}

// RebuildReplayJournal verifies the exact recomputed signed candidate BEFORE
// creating a separate journal. The old directory is never modified, including
// crashed locks/refusal markers. All original accepted evidence is replayed under
// BOTH policies. Corrupt/missing history and unacknowledged tails cannot be reset.
//
// Success returns an UNACTIVATED, unconfirmed candidate and audit. Activation
// still needs enrolled authority, durable anti-replay/CAS, old-generation fencing,
// fresh code/state/finality/clock checks and external checkpoint retention. Never
// clear quarantine or set SourceReady merely because this function succeeded.
func RebuildReplayJournal(ctx context.Context, original, destination string, previous, next JournalConfig, expected JournalCheckpoint, authority []byte, plan ReconfigurationPlan, signature []byte, limits ReconstructionLimits) (*ReplayJournal, ReconfigurationAudit, error) {
	var audit ReconfigurationAudit
	want, e := PlanJournalReconfiguration(ctx, original, previous, next, expected, authority, plan.Nonce, plan.Reason, limits)
	if e != nil || want != plan {
		return nil, audit, errors.Join(ErrReconfiguration, e)
	}
	message, e := ReconfigurationSigningMessage(want)
	if e != nil || !replayStrictEd(authority, message, signature) {
		return nil, audit, ErrReconfiguration
	}
	j, e := CreateReplayJournal(ctx, destination, next)
	if e != nil {
		return nil, audit, e
	}
	deny := func(e error) (*ReplayJournal, ReconfigurationAudit, error) {
		return nil, ReconfigurationAudit{}, errors.Join(j.fail(e), j.Close())
	}
	tip, e := reconstructPrefix(ctx, original, previous, next, expected, limits, func(entry journalEntry) error {
		_, e := j.Append(ctx, entry.Batch, entry.Started, entry.Completed)
		return e
	})
	if e != nil || tip != want.Next {
		return deny(errors.Join(ErrReconfiguration, e))
	}
	// Planning is not a snapshot lock. Reinspect the complete original prefix
	// and inventory after the copy, refusing concurrent append/change. A later
	// activation STILL needs an independent generation fence, not this readback.
	check, e := PlanJournalReconfiguration(ctx, original, previous, next, expected, authority, plan.Nonce, plan.Reason, limits)
	got, f := j.Checkpoint()
	if e != nil || f != nil || check != want || got != want.Next {
		return deny(errors.Join(ErrReconfiguration, e, f))
	}
	audit = ReconfigurationAudit{want, base64.RawURLEncoding.EncodeToString(authority), base64.RawURLEncoding.EncodeToString(signature), previous, next}
	return j, audit, nil
}
