package hpke

const TaskHPKEComplete = "hpke/complete@v1"

type InfoBuilder interface {
	BuildInfo(ctxID, initDID, respDID string) []byte
	BuildExportContext(ctxID string) []byte
}

const (
	hpkeSuiteID    = "hpke-base+x25519+hkdf-sha256"
	combinerID     = "e2e-x25519-hkdf-v1"  // Combines HPKE exporter output with (ephC, ephS) ECDH secret
	infoLabel      = "sage/hpke-info|v1"   // Domain label used for the HPKE info transcript
	exportCtxLabel = "sage/hpke-export|v1" // Domain label used for the HPKE export context
)

type DefaultInfoBuilder struct{}
