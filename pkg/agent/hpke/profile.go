package hpke

import "github.com/sage-x-project/sage/pkg/agent/session"

// profile holds the byte labels and field values one SAGE version selects for
// the HPKE completion, its record session and its HTTP binding. The chapter
// 04, 05 and 08 constructions are identical across versions; only these values
// change (sage-spec profiles/sage-0.11.0/README.md: "only explicitly selected
// values change").
//
// An endpoint's profile is fixed when it is constructed. Every message it
// accepts must carry that profile's values, so a 0.11.0 endpoint never
// accepts a 0.10.0 message or the reverse, and nothing falls back.
type profile struct {
	version      string // envelope version, binding v, x-sage-version, completion v
	httpTag      string // RFC 9421 tag parameter
	wireRequest  string // chapter 08 request signing domain
	wireResponse string // chapter 08 response signing domain
	hpkeInfo     string
	hpkeExport   string
	hpkeComplete string
	combiner     string
	ack          string
	session      string
	initTask     string
	completeTask string
	records      func(seed, th []byte, initiator bool) (*session.RecordSession010, error)
}

var profile010 = &profile{
	version:      "0.10.0",
	httpTag:      "sage-0.10.0",
	wireRequest:  "sage-wire-request|0.10.0\n",
	wireResponse: "sage-wire-response|0.10.0\n",
	hpkeInfo:     "sage-hpke-info|0.10.0\n",
	hpkeExport:   "sage-hpke-export|0.10.0\n",
	hpkeComplete: "sage-hpke-complete|0.10.0\n",
	combiner:     "sage-hpke-combiner|0.10.0",
	ack:          "sage-hpke-ack|0.10.0",
	session:      "sage-session|0.10.0",
	initTask:     "hpke/init@0.10.0",
	completeTask: "hpke/complete@0.10.0",
	records:      session.NewRecordSession010,
}

// profile011 takes every value from sage-spec profiles/sage-0.11.0/selection.json
// (domain_entries and field_values). The HPKE suite and combiner identifiers
// are retained namespaces and stay unchanged.
var profile011 = &profile{
	version:      "0.11.0",
	httpTag:      "sage-0.11.0",
	wireRequest:  "sage-wire-request|0.11.0\n",
	wireResponse: "sage-wire-response|0.11.0\n",
	hpkeInfo:     "sage-hpke-info|0.11.0\n",
	hpkeExport:   "sage-hpke-export|0.11.0\n",
	hpkeComplete: "sage-hpke-complete|0.11.0\n",
	combiner:     "sage-hpke-combiner|0.11.0",
	ack:          "sage-hpke-ack|0.11.0",
	session:      "sage-session|0.11.0",
	initTask:     "hpke/init@0.11.0",
	completeTask: "hpke/complete@0.11.0",
	records:      session.NewRecordSession011,
}
