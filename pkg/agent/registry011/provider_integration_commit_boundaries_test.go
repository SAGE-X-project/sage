package registry011

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Initial enrollment data for an isolated owned store. No I, ticket, network
// operation, selected principal or protected-use authority is created here.
func providerInitialCommitUnit(t *testing.T) (*providerCustodyStore, *CandidateEnrollmentRoot, []providerIntegrationObject, string) {
	t.Helper()
	s, policy, candidate, primitive := providerStoreFixture(t)
	id := s.config.identity
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-enrollment-v2", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": "0", "sequence": "0", "previous_root": nil, "status": "enrolling", "checkpoint": nil, "budget_root": nil, "transition_root": nil, "fence": "1"})
	root, err := ParseCandidateEnrollmentRoot(candidate, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	setup, err := s.read(s.evidence, providerCustodyName("setup-claim", 0), 64<<10)
	commit, otherErr := s.read(s.custody, providerCustodyName("setup-commit", 0), 64<<10)
	c := &providerIntegration{store: s, policy: policy, context: candidate, primitive: primitive}
	objects, objectErr := c.objects([]providerIntegrationObjectInput{{"enrollment", raw}, {"setup", setup}, {"setup-commit", commit}})
	if err != nil || otherErr != nil || objectErr != nil {
		t.Fatal("normal initial object closure", err, otherErr, objectErr)
	}
	operation := providerCandidateHash("sage-provider-enroll-operation-v1", []byte(s.config.anchor+id.Instance+id.Policy+id.SourceID))
	return s, root, objects, operation
}

func TestProviderIntegrationInitialCommitRequiresCompleteBoundedOriginals(t *testing.T) {
	for _, mode := range []string{"normal", "empty", "entries", "object-size", "missing-root", "existing-size", "existing-original", "duplicate", "clock", "phase", "metadata", "physical"} {
		t.Run(mode, func(t *testing.T) {
			s, root, objects, operation := providerInitialCommitUnit(t)
			beforeStored, beforeTip := s.stored, s.tip
			switch mode {
			case "empty":
				objects = nil
			case "entries":
				objects = make([]providerIntegrationObject, s.config.bounds.maximumEntries+1)
			case "object-size":
				objects[0].raw = strings.Repeat("x", int(s.config.bounds.maximumObjectBytes)+1)
			case "missing-root":
				objects = objects[1:]
			case "existing-size", "existing-original":
				object := objects[0]
				n := uint64(len(object.raw))
				if mode == "existing-size" {
					n++
				}
				// Inert inconsistent index input is refused before any new write.
				s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, n}
			case "duplicate":
				objects = append(objects, objects[0])
			case "clock":
				s.clock.failed = true
			case "phase":
				root.sequence = 1
			case "metadata":
				s.config.bounds.maximumMetadataPerCommit = 1
			case "physical":
				s.stored = s.config.bounds.maximumStoredBytes
				beforeStored = s.stored
			}
			ack, err := s.commit("enroll", operation, root, objects)
			if mode == "normal" {
				if err != nil || !s.confirms(ack, operation, root) || s.sequence != 1 || s.replay.intent != nil || s.replay.ledger != nil {
					t.Fatal("normal initial custody did not reconstruct E0 alone", err)
				}
				return
			}
			if err == nil || ack != nil || s.sequence != 0 || s.tip != beforeTip || s.stored != beforeStored || s.last != nil {
				t.Fatal("incomplete bounded input wrote or acknowledged a new prefix", mode)
			}
			if _, err := os.Stat(filepath.Join(s.config.custodyDirectory, providerCustodyName("intent", 1))); !os.IsNotExist(err) {
				t.Fatal("refusal preceded immutable writes", err)
			}
		})
	}
}

func TestProviderIntegrationInitialCommitPreservesPreexistingUnitTail(t *testing.T) {
	for _, mode := range []string{"intent", "object", "inventory", "pending", "commit"} {
		t.Run(mode, func(t *testing.T) {
			s, root, objects, operation := providerInitialCommitUnit(t)
			inventory, err := newProviderIntegrationInventory(s.objectsRoot, objects, s.config.bounds.maximumEntries, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			directory, name := s.config.custodyDirectory, providerCustodyName(mode, 1)
			switch mode {
			case "object":
				directory, name = s.config.evidenceDirectory, providerObjectName(objects[0].kind, objects[0].digest)
			case "inventory":
				directory, name = s.config.evidenceDirectory, "inventory-"+inventory.digest[2:]
			case "pending":
				directory = s.config.evidenceDirectory
			}
			path := filepath.Join(directory, name)
			const tail = "finite ordinary unit crash tail"
			if err := os.WriteFile(path, []byte(tail), 0600); err != nil {
				t.Fatal(err)
			}
			stored, tip := s.stored, s.tip
			ack, err := s.commit("enroll", operation, root, objects)
			if err == nil || ack != nil || !s.failed || s.sequence != 0 || s.tip != tip || s.stored != stored || s.last != nil {
				t.Fatal("uncertain legacy commit became a reusable owner or ACK", mode)
			}
			raw, err := os.ReadFile(path)
			if err != nil || string(raw) != tail || s.setupTip == s.config.anchor {
				t.Fatal("unit tail or precharged setup was replaced", err)
			}
		})
	}
}
