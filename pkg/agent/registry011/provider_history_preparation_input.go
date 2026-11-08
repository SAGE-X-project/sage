package registry011

import (
	"encoding/json"
	"time"
)

// This is local preparation DATA, not a protocol credential or selected
// operating profile. Every member is required by the bounded decoder. Integer
// tokens are checked and decoded directly into integer types without float64.
// Policy retains its original canonical bytes for the existing policy parser.
type historyPreparationInput struct {
	Format string `json:"format"`
	Policy json.RawMessage `json:"policy"`
	MaximumPolicyBytes int `json:"maximum_policy_bytes"`
	Namespace string `json:"namespace"`
	SourceID string `json:"source_id"`
	Generation string `json:"generation"`
	Store historyPreparationStore `json:"store"`
	Capture ProviderCaptureLimits `json:"capture"`
	Collection historyPreparationCollection `json:"collection"`
	Bounds historyPreparationBounds `json:"bounds"`
	Clock historyPreparationClock `json:"clock"`
}

type historyPreparationStore struct {
	CustodyDirectory string `json:"custody_directory"`
	EvidenceDirectory string `json:"evidence_directory"`
	Anchor string `json:"anchor"`
	ExpectedTip string `json:"expected_tip"`
	ExpectedSetupTip string `json:"expected_setup_tip"`
	ExpectedScanHead string `json:"expected_scan_head"`
	Profile string `json:"profile"`
}

type historyPreparationClock struct {
	Maximum uint64 `json:"maximum"`
	BaseUncertaintyMS int64 `json:"base_uncertainty_ms"`
	ReadWindowMS int64 `json:"read_window_ms"`
}

type historyPreparationCode struct {
	Name string `json:"name"`
	Address string `json:"address"`
	SHA256 string `json:"code_sha256"`
}

type historyPreparationCollection struct {
	Journal JournalConfig `json:"journal"`
	Codes [13]historyPreparationCode `json:"codes"`
	Encoding string `json:"block_encoding"`
	Gas string `json:"gas_profile"`
	MaximumSelected uint64 `json:"maximum_selected_height"`
	Blocks uint64 `json:"maximum_inclusive_blocks"`
	Native uint64 `json:"maximum_native_blocks"`
	Effects uint64 `json:"maximum_effect_transactions"`
	Versions uint64 `json:"maximum_history_versions"`
	PresentStops uint64 `json:"maximum_present_stops"`
	WindowBlocks uint64 `json:"window_blocks"`
	ContentBlocks uint64 `json:"content_blocks_per_group"`
	FirstRead historyPreparationFirstRead `json:"first_read"`
	TraceTimeout time.Duration `json:"trace_timeout_ns"`
}

type historyPreparationFirstRead struct {
	Objects uint64 `json:"objects"`
	Raw uint64 `json:"raw"`
}

type historyPreparationBounds struct {
	Ledger CandidateLedgerBounds `json:"ledger"`
	Continuation historyPreparationContinuation `json:"continuation"`
	Methods historyPreparationMethods `json:"methods"`
	TLS historyPreparationTLS `json:"tls"`
	Scan historyPreparationScan `json:"scan"`
	FirstAdmission historyPreparationAdmission `json:"first_admission"`
	History historyPreparationHistory `json:"history"`
	MaximumCommits int `json:"maximum_commits"`
	MaximumObjects int `json:"maximum_objects"`
	MaximumEntries int `json:"maximum_entries"`
	MaximumObjectBytes uint64 `json:"maximum_object_bytes"`
	MaximumStoredBytes uint64 `json:"maximum_stored_bytes"`
	MaximumMetadataPerCommit uint64 `json:"maximum_metadata_per_commit"`
	FileOverheadBytes uint64 `json:"file_overhead_bytes"`
	MaximumSetupRecoveries int `json:"maximum_setup_recoveries"`
	SetupWork candidateWorkVector `json:"setup_work"`
	SetupLive uint64 `json:"setup_live"`
	SetupRetained uint64 `json:"setup_retained"`
	SetupMaximumClockReads uint64 `json:"setup_maximum_clock_reads"`
	SetupMaximumDurationMS uint64 `json:"setup_maximum_duration_ms"`
}

type historyPreparationContinuation struct {
	MaximumRecords uint64 `json:"maximum_records"`
	PageRecords uint64 `json:"page_records"`
	MaximumPages uint64 `json:"maximum_pages"`
	MaximumPayloadBytes uint64 `json:"maximum_payload_bytes"`
	MaximumMetadataBytes uint64 `json:"maximum_metadata_bytes"`
	PagePayloadBytes uint64 `json:"page_payload_bytes"`
	PageMetadataBytes uint64 `json:"page_metadata_bytes"`
	MaximumResidentBytes uint64 `json:"maximum_resident_bytes"`
	CodecMemoryMultiplier uint64 `json:"codec_memory_multiplier"`
	TicketMemoryOverhead uint64 `json:"ticket_memory_overhead"`
	OperationMemoryOverhead uint64 `json:"operation_memory_overhead"`
	SegmentMemoryOverhead uint64 `json:"segment_memory_overhead"`
	ObjectMemoryOverhead uint64 `json:"object_memory_overhead"`
	MaximumCommits uint64 `json:"maximum_commits"`
	MaximumObjects uint64 `json:"maximum_objects"`
}

type historyPreparationMethod struct {
	Body int `json:"body"`
	Wire int `json:"wire"`
}

type historyPreparationMethods struct {
	Scalar historyPreparationMethod `json:"scalar"`
	Header historyPreparationMethod `json:"header"`
	Code historyPreparationMethod `json:"code"`
	State historyPreparationMethod `json:"state"`
	Block historyPreparationMethod `json:"block"`
	Receipts historyPreparationMethod `json:"receipts"`
	Gas historyPreparationMethod `json:"gas"`
	Execution historyPreparationMethod `json:"execution"`
}

type historyPreparationTLS struct {
	Chains int `json:"chains"`
	Certificates int `json:"certificates"`
	CertificateBytes int `json:"certificate_bytes"`
}

type historyPreparationScan struct {
	MaximumAttempts uint64 `json:"maximum_attempts"`
	MaximumRecords uint64 `json:"maximum_records"`
	MaximumObjects uint64 `json:"maximum_objects"`
	MaximumRawBytes uint64 `json:"maximum_raw_bytes"`
	MaximumPages uint64 `json:"maximum_pages"`
	MaximumChunks uint64 `json:"maximum_chunks"`
	ChunkRecords uint64 `json:"chunk_records"`
	ChunkObjects uint64 `json:"chunk_objects"`
	ChunkRawBytes uint64 `json:"chunk_raw_bytes"`
	MaximumResidentBytes uint64 `json:"maximum_resident_bytes"`
	FilenameEntryOverhead uint64 `json:"filename_entry_overhead"`
	MaximumFilenameBytes uint64 `json:"maximum_filename_bytes"`
	MaximumClockReads uint64 `json:"maximum_clock_reads"`
	ChunkDurationMS uint64 `json:"chunk_duration_ms"`
	TotalDurationMS uint64 `json:"total_duration_ms"`
	MaximumCatalogEntries uint64 `json:"maximum_catalog_entries"`
	MaximumCatalogNameBytes uint64 `json:"maximum_catalog_name_bytes"`
	MaximumPreIRetainedBytes uint64 `json:"maximum_pre_i_retained_bytes"`
	MaximumPreIClockReads uint64 `json:"maximum_pre_i_clock_reads"`
	ChunkWork candidateWorkVector `json:"chunk_work"`
	MaximumPreIWork candidateWorkVector `json:"maximum_pre_i_work"`
}

type historyPreparationAdmission struct {
	Work candidateWorkVector `json:"work"`
	Retained uint64 `json:"retained"`
	Live uint64 `json:"live"`
	Clocks uint64 `json:"clocks"`
	DurationMS uint64 `json:"duration_ms"`
}

type historyPreparationTotals struct {
	Work candidateWorkVector `json:"work"`
	Physical uint64 `json:"physical"`
	RetainedRaw uint64 `json:"retained_raw"`
	Payload uint64 `json:"payload"`
	Metadata uint64 `json:"metadata"`
	Objects uint64 `json:"objects"`
	Files uint64 `json:"files"`
	Clocks uint64 `json:"clocks"`
}

type historyPreparationStartup struct {
	Work candidateWorkVector `json:"work"`
	Retained uint64 `json:"retained"`
	Clocks uint64 `json:"clocks"`
}

type historyPreparationRecovery struct {
	Work candidateWorkVector `json:"work"`
	MaximumFiles uint64 `json:"maximum_files"`
	MaximumBytes uint64 `json:"maximum_bytes"`
}

type historyPreparationFinal struct {
	Read historyPreparationRecovery `json:"read"`
	ElapsedMS uint64 `json:"elapsed_ms"`
}

type historyPreparationHistory struct {
	Program string `json:"program"`
	Costs string `json:"costs"`
	Future historyPreparationTotals `json:"future"`
	Startup historyPreparationStartup `json:"startup"`
	Recovery historyPreparationRecovery `json:"recovery"`
	Final historyPreparationFinal `json:"final"`
	Stages uint64 `json:"stages"`
	Records uint64 `json:"records"`
	Commits uint64 `json:"commits"`
	CatalogEntries uint64 `json:"catalog_entries"`
	CatalogNameBytes uint64 `json:"catalog_name_bytes"`
	Resident uint64 `json:"resident"`
	DurationMS uint64 `json:"duration_ms"`
}
