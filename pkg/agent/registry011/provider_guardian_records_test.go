package registry011

import "testing"

func providerGuardianRecordFixture(t *testing.T) (*providerScanCodec, providerGuardianHeadData, providerGuardianAttemptData) {
	t.Helper()
	config, _ := providerGuardianConfigFixture(t)
	x := mustProviderBindingCodec(t, config)
	clock := CandidateClockDiagnostic{ClockID: "candidate-clock", Epoch: admissionTestDigest("c"), WallMS: 1000000, MonotonicMS: 10, UncertaintyMS: 1}
	deadline, _, ok := x.deadline(clock)
	if !ok {
		t.Fatal("original fixed DATA deadline")
	}
	head := providerGuardianHeadData{origin: admissionTestDigest("a"), e0: admissionTestDigest("b"), frontier: admissionTestDigest("c"), h0: admissionTestDigest("d"), setup: admissionTestDigest("e"), clock: clock, deadline: deadline}
	raw, err := x.guardianHead(head)
	if err != nil {
		t.Fatal(err)
	}
	start := clock
	start.WallMS++
	start.MonotonicMS++
	v := providerGuardianAttemptData{origin: head.origin, previous: x.hash(providerGuardianHeadDomain, raw), main: admissionTestDigest("f"), frontier: head.frontier, setup: head.setup, target: admissionTestDigest("1"), admission: admissionTestDigest("2"), purpose: "bootstrap-admission", ordinal: 1, deadline: deadline, clock: clock, start: start}
	v.operation, err = x.guardianOperation(v)
	if err != nil {
		t.Fatal(err)
	}
	return x, head, v
}

func TestProviderGuardianClosedRecordsBindOriginalClockAndSeparatePurpose(t *testing.T) {
	x, before, v := providerGuardianRecordFixture(t)
	for _, purpose := range []string{"bootstrap-admission", "rpc-admission", "local-admission", "publication-inspection", "ledger-page-seal"} {
		t.Run(purpose, func(t *testing.T) {
			attempt := v
			attempt.purpose = purpose
			var err error
			attempt.operation, err = x.guardianOperation(attempt)
			if err != nil {
				t.Fatal(err)
			}
			intent, err := x.guardianAttempt(attempt)
			parsed, parseErr := x.parseGuardianAttempt(intent)
			claim, claimErr := x.guardianClaim(intent, attempt)
			commit, commitErr := x.guardianCommit(intent, claim, attempt)
			after := before
			after.ordinal, after.previous, after.admission, after.lastStart = attempt.ordinal, attempt.previous, attempt.admission, attempt.start
			after.claim, after.commit = x.hash(providerGuardianClaimDomain, claim), x.hash(providerGuardianCommitDomain, commit)
			head, headErr := x.guardianHead(after)
			if err != nil || parseErr != nil || parsed != attempt || claimErr != nil || commitErr != nil || headErr != nil || len(head) == 0 {
				t.Fatal("closed original chain failed", err, parseErr, claimErr, commitErr, headErr)
			}
			if string(claim) == string(commit) || before.ordinal != 0 || after.clock != before.clock || after.deadline != before.deadline {
				t.Fatal("record roles or original admission clock were replaced")
			}
		})
	}
	if (new(providerGuardianOwner)).UnmarshalJSON(nil) == nil || providerGuardianPurpose("arbitrary-work") {
		t.Fatal("DATA created an owner or unknown command")
	}
}

func TestProviderGuardianDataRejectsCrossedPredecessorClockAndUnknownFields(t *testing.T) {
	x, before, v := providerGuardianRecordFixture(t)
	for _, mode := range []string{"ordinal", "purpose", "origin", "previous", "main", "frontier", "setup", "target", "admission", "deadline", "epoch", "time", "expired"} {
		t.Run(mode, func(t *testing.T) {
			attempt := v
			switch mode {
			case "ordinal":
				attempt.ordinal = 0
			case "purpose":
				attempt.purpose = "unknown"
			case "origin":
				attempt.origin = ""
			case "previous":
				attempt.previous = ""
			case "main":
				attempt.main = ""
			case "frontier":
				attempt.frontier = ""
			case "setup":
				attempt.setup = ""
			case "target":
				attempt.target = ""
			case "admission":
				attempt.admission = ""
			case "deadline":
				attempt.deadline++
			case "epoch":
				attempt.start.Epoch = admissionTestDigest("f")
			case "time":
				attempt.start.MonotonicMS = -1
			case "expired":
				attempt.start.WallMS += 300001
				attempt.start.MonotonicMS += 300001
			}
			if _, err := x.guardianAttempt(attempt); err == nil {
				t.Fatal("inconsistent finite attempt accepted", mode)
			}
		})
	}
	intent, _ := x.guardianAttempt(v)
	for _, mode := range []string{"unknown", "operation", "upper", "selected", "clock", "ordinal", "missing"} {
		t.Run("parse-"+mode, func(t *testing.T) {
			m := contentJSON(intent).(map[string]any)
			switch mode {
			case "unknown":
				m["unknown"] = "inert DATA"
			case "operation":
				m["operation_id"] = admissionTestDigest("f")
			case "upper":
				m["claim_upper"] = map[string]any{}
			case "selected":
				m["selected_guardian_head"] = admissionTestDigest("f")
			case "clock":
				m["original_clock"] = nil
			case "ordinal":
				m["ordinal"] = "-1"
			case "missing":
				delete(m, "origin_digest")
			}
			if _, err := x.parseGuardianAttempt(providerIntegrationJSON(m)); err == nil {
				t.Fatal("noncanonical or crossed attempt parsed")
			}
		})
	}
	if _, err := x.guardianClaim([]byte(`{}`), v); err == nil {
		t.Fatal("claim accepted a different intent")
	}
	if _, err := x.guardianCommit(intent, []byte(`{}`), v); err == nil {
		t.Fatal("commit accepted a different claim")
	}
	for _, mode := range []string{"genesis-previous", "genesis-start", "missing-fixed", "future-clock", "deadline"} {
		head := before
		switch mode {
		case "genesis-previous":
			head.previous = admissionTestDigest("f")
		case "genesis-start":
			head.lastStart = v.start
		case "missing-fixed":
			head.e0 = ""
		case "future-clock":
			head.ordinal = 1
		case "deadline":
			head.deadline++
		}
		if _, err := x.guardianHead(head); err == nil {
			t.Fatal("invalid head accepted", mode)
		}
	}
}
