// Local Inspector adapter for bounded web Registry lifecycle checks.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type request struct {
	Action               string `json:"action"`
	DID                  string `json:"did"`
	Previous             string `json:"previous"`
	Candidate            string `json:"candidate"`
	PreviousNow          int64  `json:"previous_now"`
	CandidateNow         int64  `json:"candidate_now"`
	Operation            string `json:"operation"`
	ExpectedVersion      string `json:"expected_version"`
	FixtureActor         string `json:"fixture_actor"`
	FixtureScope         string `json:"fixture_scope"`
	FixtureAuthenticated bool   `json:"fixture_authenticated"`
	History              []struct {
		Envelope  string `json:"envelope"`
		At        int64  `json:"at"`
		Operation string `json:"operation"`
	} `json:"history"`
}

// This authority is a local Inspector fixture, not a transport authenticator.
type fixtureAuthority struct {
	actor, scope  string
	authenticated bool
}

func (a fixtureAuthority) AuthenticatedActor(context.Context) (string, error) {
	if !a.authenticated {
		return "", errors.New("fixture credentials absent")
	}
	return a.actor, nil
}

func (a fixtureAuthority) Delegated(_ context.Context, controller, actor, _ string, operation, _ string) (bool, error) {
	return controller == "operator" && actor == "assistant" && a.scope == operation, nil
}

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 150001))
	if err != nil || len(raw) > 150000 {
		os.Exit(2)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var req request
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	candidate, err := base64.RawURLEncoding.DecodeString(req.Candidate)
	if err != nil {
		os.Exit(2)
	}
	switch req.Action {
	case "create":
		err = registry010.CheckWebRegistryCreationShape010(candidate, req.DID, req.CandidateNow)
	case "transition":
		previous, parseErr := base64.RawURLEncoding.DecodeString(req.Previous)
		if parseErr != nil {
			os.Exit(2)
		}
		err = registry010.CheckWebRegistryTransitionShape010(
			previous, candidate, req.DID, req.PreviousNow, req.CandidateNow, req.Operation)
	case "history":
		history := make([]registry010.WebRegistryHistoryEntry010, 0, len(req.History))
		for _, item := range req.History {
			envelope, parseErr := base64.RawURLEncoding.DecodeString(item.Envelope)
			if parseErr != nil {
				os.Exit(2)
			}
			history = append(history, registry010.WebRegistryHistoryEntry010{
				Envelope: envelope, At: item.At, Operation: item.Operation,
			})
		}
		err = registry010.CheckWebRegistryHistoryContinuity010(
			history, candidate, req.DID, req.CandidateNow)
	case "admission-create":
		authority := fixtureAuthority{req.FixtureActor, req.FixtureScope, req.FixtureAuthenticated}
		err = registry010.CheckWebRegistryCreationAdmission010(
			context.Background(), authority, candidate, req.DID, req.CandidateNow)
	case "admission-transition":
		previous, parseErr := base64.RawURLEncoding.DecodeString(req.Previous)
		if parseErr != nil {
			os.Exit(2)
		}
		authority := fixtureAuthority{req.FixtureActor, req.FixtureScope, req.FixtureAuthenticated}
		err = registry010.CheckWebRegistryMutationAdmission010(
			context.Background(), authority, previous, candidate, req.DID,
			req.CandidateNow, req.ExpectedVersion, req.Operation)
	default:
		os.Exit(2)
	}
	verdict := "TRANSITION_ACCEPT"
	if errors.Is(err, registry010.ErrSizeExceeded010) {
		verdict = "SIZE_EXCEEDED"
	} else if errors.Is(err, registry010.ErrStale) {
		verdict = "RECORD_STALE"
	} else if errors.Is(err, registry010.ErrRejected) {
		verdict = "WRITE_REJECTED"
	} else if err != nil {
		verdict = "RECORD_INVALID"
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"verdict": verdict}) != nil {
		os.Exit(2)
	}
}
