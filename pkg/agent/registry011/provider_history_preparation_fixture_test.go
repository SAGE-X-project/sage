package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"
)

// Reuse the original selection without weakening its limits, clearing a
// derivation or installing authority. Its helper supplies bounds and policy,
// not a bound public identity. Bind a separate test-only proposal before
// constructing the input; never overwrite an existing private identity.
// This is a small preparation fixture, never the full-history operating profile.
func historyPreparationInputFromOriginalTestAPI(t *testing.T) (historyPreparationInput, ProviderCandidateIdentity, string) {
	t.Helper()
	c := historySelectedSourceConfig(t)
	if c.identity != (ProviderCandidateIdentity{}) {
		t.Fatal("original selection already has an identity; refusing to rebind it")
	}
	b := c.bounds
	if c.attempt != 0 || c.policy == nil || b.guardian != nil || b.history == nil || b.history.derivation != nil ||
		b.continuation == nil || b.methods == nil || b.tls == nil || b.scan == nil || b.firstAdmission == nil ||
		!b.history.collection.validSnapshot() {
		t.Fatal("original test API did not supply an unminted preparation selection")
	}
	clock := historySelectedSourceClock(c)
	window := int64(clock.window / time.Millisecond)
	if clock.window%time.Millisecond != 0 || window <= 0 || clock.uncertainty <= window {
		t.Fatal("original test clock cannot supply a positive base uncertainty without changing its selection")
	}
	l, m, s, a, h := b.continuation, b.methods, b.scan, b.firstAdmission, b.history
	v := h.collection.input
	policy := c.policy.CanonicalBytes()
	proposal, err := ParseProviderPolicyCandidate(policy, len(policy))
	if err != nil {
		t.Fatalf("original test policy rejected before binding: %v", err)
	}
	// These digests identify only this test proposal. Generation zero does not
	// establish enrollment, authenticated absence or permission to send messages.
	namespace := sha256.Sum256([]byte("test:sage:registry011:history-preparation:namespace"))
	sourceID := sha256.Sum256([]byte("test:sage:registry011:history-preparation:source"))
	proposed, err := proposal.BindCandidate("0x"+hex.EncodeToString(namespace[:]), "0x"+hex.EncodeToString(sourceID[:]), "0")
	if err != nil {
		t.Fatalf("public test proposal rejected before transport: %v", err)
	}
	expectedIdentity, err := proposed.Identity()
	if err != nil {
		t.Fatal(err)
	}
	in := historyPreparationInput{
		Format: historyPreparationInputFormat, Policy: string(policy), MaximumPolicyBytes: len(policy),
		Namespace: expectedIdentity.Namespace, SourceID: expectedIdentity.SourceID, Generation: strconv.FormatUint(expectedIdentity.Generation, 10),
		Store: historyPreparationStore{
			CustodyDirectory: c.custodyDirectory, EvidenceDirectory: c.evidenceDirectory,
			Anchor: c.anchor, ExpectedTip: c.expectedTip, ExpectedSetupTip: c.expectedSetupTip,
			ExpectedScanHead: c.expectedScanHead, Profile: c.profile,
		},
		Capture: c.capture,
		Clock:   historyPreparationClock{Maximum: clock.maximum, BaseUncertaintyMS: clock.uncertainty - window, ReadWindowMS: window},
		Collection: historyPreparationCollection{
			Journal: v.journal, Encoding: v.encoding, Gas: v.gas, MaximumSelected: v.maximumSelected,
			Blocks: v.blocks, Native: v.native, Effects: v.effects, Versions: v.versions, PresentStops: v.presentStops,
			WindowBlocks: v.windowBlocks, ContentBlocks: v.contentBlocks,
			FirstRead: historyPreparationFirstRead{Objects: v.firstRead.objects, Raw: v.firstRead.raw}, TraceTimeout: v.traceTimeout,
		},
		Bounds: historyPreparationBounds{
			Ledger: b.ledger,
			Continuation: historyPreparationContinuation{
				MaximumRecords: l.maximumRecords, PageRecords: l.pageRecords, MaximumPages: l.maximumPages,
				MaximumPayloadBytes: l.maximumPayloadBytes, MaximumMetadataBytes: l.maximumMetadataBytes,
				PagePayloadBytes: l.pagePayloadBytes, PageMetadataBytes: l.pageMetadataBytes,
				MaximumResidentBytes: l.maximumResidentBytes, CodecMemoryMultiplier: l.codecMemoryMultiplier,
				TicketMemoryOverhead: l.ticketMemoryOverhead, OperationMemoryOverhead: l.operationMemoryOverhead,
				SegmentMemoryOverhead: l.segmentMemoryOverhead, ObjectMemoryOverhead: l.objectMemoryOverhead,
				MaximumCommits: l.maximumCommits, MaximumObjects: l.maximumObjects,
			},
			Methods: historyPreparationMethods{
				Scalar:    historyPreparationMethod{Body: m.scalar.body, Wire: m.scalar.wire},
				Header:    historyPreparationMethod{Body: m.header.body, Wire: m.header.wire},
				Code:      historyPreparationMethod{Body: m.code.body, Wire: m.code.wire},
				State:     historyPreparationMethod{Body: m.state.body, Wire: m.state.wire},
				Block:     historyPreparationMethod{Body: m.block.body, Wire: m.block.wire},
				Receipts:  historyPreparationMethod{Body: m.receipts.body, Wire: m.receipts.wire},
				Gas:       historyPreparationMethod{Body: m.gas.body, Wire: m.gas.wire},
				Execution: historyPreparationMethod{Body: m.execution.body, Wire: m.execution.wire},
			},
			TLS: historyPreparationTLS{Chains: b.tls.chains, Certificates: b.tls.certificates, CertificateBytes: b.tls.certificateBytes},
			Scan: historyPreparationScan{
				MaximumAttempts: s.maximumAttempts, MaximumRecords: s.maximumRecords, MaximumObjects: s.maximumObjects,
				MaximumRawBytes: s.maximumRawBytes, MaximumPages: s.maximumPages, MaximumChunks: s.maximumChunks,
				ChunkRecords: s.chunkRecords, ChunkObjects: s.chunkObjects, ChunkRawBytes: s.chunkRawBytes,
				MaximumResidentBytes: s.maximumResidentBytes, FilenameEntryOverhead: s.filenameEntryOverhead,
				MaximumFilenameBytes: s.maximumFilenameBytes, MaximumClockReads: s.maximumClockReads,
				ChunkDurationMS: s.chunkDurationMS, TotalDurationMS: s.totalDurationMS,
				MaximumCatalogEntries: s.maximumCatalogEntries, MaximumCatalogNameBytes: s.maximumCatalogNameBytes,
				MaximumPreIRetainedBytes: s.maximumPreIRetainedBytes, MaximumPreIClockReads: s.maximumPreIClockReads,
				ChunkWork: s.chunkWork, MaximumPreIWork: s.maximumPreIWork,
			},
			FirstAdmission: historyPreparationAdmission{Work: a.work, Retained: a.retained, Live: a.live, Clocks: a.clocks, DurationMS: a.durationMS},
			History: historyPreparationHistory{
				Program: h.program, Costs: h.costs,
				Future: historyPreparationTotals{Work: h.future.work, Physical: h.future.physical, RetainedRaw: h.future.retainedRaw,
					Payload: h.future.payload, Metadata: h.future.metadata, Objects: h.future.objects, Files: h.future.files, Clocks: h.future.clocks},
				Startup:  historyPreparationStartup{Work: h.startup.work, Retained: h.startup.retained, Clocks: h.startup.clocks},
				Recovery: historyPreparationRecovery{Work: h.recovery.work, MaximumFiles: h.recovery.maximumFiles, MaximumBytes: h.recovery.maximumBytes},
				Final: historyPreparationFinal{Read: historyPreparationRecovery{Work: h.final.read.work, MaximumFiles: h.final.read.maximumFiles,
					MaximumBytes: h.final.read.maximumBytes}, ElapsedMS: h.final.elapsedMS},
				Stages: h.stages, Records: h.records, Commits: h.commits, CatalogEntries: h.catalogEntries,
				CatalogNameBytes: h.catalogNameBytes, Resident: h.resident, DurationMS: h.durationMS,
			},
			MaximumCommits: b.maximumCommits, MaximumObjects: b.maximumObjects, MaximumEntries: b.maximumEntries,
			MaximumObjectBytes: b.maximumObjectBytes, MaximumStoredBytes: b.maximumStoredBytes,
			MaximumMetadataPerCommit: b.maximumMetadataPerCommit, FileOverheadBytes: b.fileOverheadBytes,
			MaximumSetupRecoveries: b.maximumSetupRecoveries, SetupWork: b.setupWork,
			SetupLive: b.setupLive, SetupRetained: b.setupRetained,
			SetupMaximumClockReads: b.setupMaximumClockReads, SetupMaximumDurationMS: b.setupMaximumDurationMS,
		},
	}
	for i, code := range v.codes {
		in.Collection.Codes[i] = historyPreparationCode(code)
	}
	// Check the original proposal before transport or negative-case mutation.
	// A non-nil private test value is not evidence of valid canonical policy.
	parsed, err := ParseProviderPolicyCandidate(policy, in.MaximumPolicyBytes)
	if err != nil {
		t.Fatalf("original test policy rejected before transport: %v", err)
	}
	bound, err := parsed.BindCandidate(in.Namespace, in.SourceID, in.Generation)
	if err != nil {
		_, generationValid := providerCandidateUint(in.Generation, false)
		t.Fatalf("original test policy binding rejected before transport (namespace_digest=%t source_id_digest=%t generation_canonical=%t): %v",
			providerCandidateDigest(in.Namespace), providerCandidateDigest(in.SourceID), generationValid, err)
	}
	identity, err := bound.Identity()
	if err != nil || identity != expectedIdentity {
		t.Fatalf("public test policy identity differs before transport: err=%v", err)
	}
	return in, expectedIdentity, h.collection.digest
}

func TestProviderHistoryPreparationCompilesOriginalUnmintedSelection(t *testing.T) {
	// The same test may export its bounded DATA to the fixed-child CI test.
	// A missing supported fixture is a failure, never a skip or refusal PASS.
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	session, err := NewHistoryPreparationSession(ctx, origin, 64<<20, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	in, expectedIdentity, expectedManifest := historyPreparationInputFromOriginalTestAPI(t)
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeHistoryPreparationInput(ctx, raw)
	if err != nil || decoded.Policy != in.Policy {
		t.Fatalf("original policy bytes changed during transport: err=%v", err)
	}
	output, err := session.Prepare(raw)
	if err != nil {
		t.Fatalf("original unminted selection did not prepare: %v", err)
	}
	summary, err := DecodeHistoryPreparationSummary(ctx, output, raw, 64<<10)
	if err != nil || summary.SourceReady || summary.Identity != expectedIdentity || summary.ManifestSHA256 != expectedManifest || summary.OriginUnixNano != origin.UnixNano() ||
		summary.DeadlineUnixNano != origin.Add(time.Minute).UnixNano() || summary.AssignedWork == [11]uint64{} {
		t.Fatalf("preparation DATA or original window missing: err=%v source_ready=%t identity_match=%t manifest_match=%t origin_match=%t deadline_match=%t work_assigned=%t",
			err, summary.SourceReady, summary.Identity == expectedIdentity, summary.ManifestSHA256 == expectedManifest,
			summary.OriginUnixNano == origin.UnixNano(), summary.DeadlineUnixNano == origin.Add(time.Minute).UnixNano(), summary.AssignedWork != [11]uint64{})
	}
	if output, err := session.Prepare(raw); err == nil || output != nil {
		t.Fatal("successful attempt was reused")
	}
	if path, export := os.LookupEnv("SAGE_HISTORY_PREPARATION_FIXTURE_PATH"); export {
		writeHistoryPreparationTestData(t, path, raw)
		reference, err := json.Marshal(struct {
			Identity       ProviderCandidateIdentity `json:"identity"`
			ManifestSHA256 string                    `json:"manifest_sha256"`
		}{expectedIdentity, expectedManifest})
		if err != nil {
			t.Fatal(err)
		}
		writeHistoryPreparationTestData(t, path+".expected.json", reference)
	}
}

func writeHistoryPreparationTestData(t *testing.T, path string, raw []byte) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	n, writeErr := file.Write(raw)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || n != len(raw) || syncErr != nil || closeErr != nil {
		t.Fatalf("preparation fixture export failed: write=%v sync=%v close=%v", writeErr, syncErr, closeErr)
	}
}

func TestProviderHistoryPreparationRejectsInvalidSelectedInputs(t *testing.T) {
	for name, change := range map[string]func(*historyPreparationInput){
		"format":     func(in *historyPreparationInput) { in.Format = "unknown" },
		"capture":    func(in *historyPreparationInput) { in.Capture.Timeout = 0 },
		"policy":     func(in *historyPreparationInput) { in.Policy = `{}` },
		"namespace":  func(in *historyPreparationInput) { in.Namespace = "missing" },
		"generation": func(in *historyPreparationInput) { in.Generation = "01" },
		"entries":    func(in *historyPreparationInput) { in.Bounds.MaximumEntries = 0 },
		"first-read": func(in *historyPreparationInput) { in.Collection.FirstRead.Raw = 0 },
		"code":       func(in *historyPreparationInput) { in.Collection.Codes[0].SHA256 = "missing" },
		"stages":     func(in *historyPreparationInput) { in.Bounds.History.Stages = 0 },
		"structure":  func(in *historyPreparationInput) { in.Bounds.MaximumMetadataPerCommit = 0 },
		"directory":  func(in *historyPreparationInput) { in.Store.CustodyDirectory = "" },
		"clock":      func(in *historyPreparationInput) { in.Clock.Maximum = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			origin := time.Now()
			ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
			defer cancel()
			session, err := NewHistoryPreparationSession(ctx, origin, 64<<20, 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			in, _, _ := historyPreparationInputFromOriginalTestAPI(t)
			change(&in)
			raw, err := json.Marshal(in)
			if err != nil {
				t.Fatal(err)
			}
			if output, err := session.Prepare(raw); err == nil || output != nil {
				t.Fatal("invalid original selection produced a result")
			} else if (name == "policy" || name == "namespace" || name == "generation") && !errors.Is(err, ErrProviderPolicyCandidate) {
				t.Fatalf("policy error identity lost: %v", err)
			}
			if output, err := session.Prepare(raw); err == nil || output != nil {
				t.Fatal("failed selection refunded its attempt")
			}
		})
	}
}
