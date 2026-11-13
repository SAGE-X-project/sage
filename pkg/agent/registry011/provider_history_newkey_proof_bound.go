package registry011

import "reflect"

// One original new-key verification under the selected Registry/agent and
// observed parser/owned-state field bounds. Counts alone do not establish
// that caller route or bind a production factory. These explicit logical,
// decoded and proof-attempt rows do not certify native curve CPU or RAM.
func providerHistoryNewKeyProofUpper(added, retained uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if added > 128 || retained > 128 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	key := uint64(reflect.TypeFor[replayKey]().Size())
	// Registry58, agent64, name32 and valid alg7 give challenge<=218;
	// strict verification therefore admits at most32+218+64=314 bytes.
	// One bundle: binary43/86, challenge binary43 and strict codec314.
	decoded := a.product(486, added)
	// The first Ed loop and second X loop are disjoint on immutable Alg.
	// KEM lookup still inspects up to every retained key for each addition.
	outer := a.product(added, a.sum(a.product(2, key), 122027))
	search := a.product(a.product(added, retained), a.sum(key, 522))
	units := a.sum(133, outer, search) // the DID join occurs even with no additions
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[2], work[7], work[10] = decoded, added, units
	return work, true
}
