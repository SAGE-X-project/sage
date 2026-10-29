package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderFrontierCandidateRefusesIncompleteOrUnfundedOpening(t *testing.T) {
	for _, mode := range []string{"custody", "evidence", "anchor", "extra-entry", "failed-clock", "setup-cap", "claim-cap", "clock-cap", "setup-clock-cap"} {
		t.Run(mode, func(t *testing.T) {
			u := providerFrontierUnitFixture(t)
			config := u.s.config
			switch mode {
			case "setup-cap":
				config.bounds.setupRetained = 1
			case "claim-cap":
				config.bounds.maximumMetadataPerCommit = 1000
			case "setup-clock-cap":
				config.bounds.setupMaximumClockReads = 1
			}
			config.profile = providerIntegrationProfile(config.bounds, config.capture)
			empty, err := newProviderFrontierInventory("", nil, config.bounds.maximumEntries, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			anchor := providerIntegrationJSON(map[string]string{"format": "sage-provider-custody-anchor-v1", "namespace": config.identity.Namespace, "instance_key": config.identity.Instance, "profile": config.profile, "initial_objects_root": empty.digest})
			if mode == "anchor" {
				anchor = []byte(`{}`)
			}
			if mode == "claim-cap" && len(anchor) > 1000 {
				t.Fatal("claim-cap fixture must retain a readable normal anchor")
			}
			config.anchor = providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
			config.expectedTip, config.expectedSetupTip = config.anchor, config.anchor
			root := t.TempDir()
			config.custodyDirectory, config.evidenceDirectory = filepath.Join(root, "custody"), filepath.Join(root, "evidence")
			for i, path := range []string{config.custodyDirectory, config.evidenceDirectory} {
				if mode == "custody" && i == 0 || mode == "evidence" && i == 1 {
					continue
				}
				if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, "AUTHORITY.LOCK"), nil, 0600) != nil || os.WriteFile(filepath.Join(path, "ANCHOR"), anchor, 0600) != nil {
					t.Fatal("normal isolated candidate fixture provisioning")
				}
			}
			if mode == "extra-entry" && os.WriteFile(filepath.Join(config.custodyDirectory, "ordinary-existing-record"), []byte("finite unit tail"), 0600) != nil {
				t.Fatal("finite existing-tail unit data")
			}
			maximum := uint64(256)
			if mode == "clock-cap" {
				maximum = 1
			}
			clock, err := newProviderOwnedClock(config.policy, maximum, 100, 100)
			if err != nil || !config.bounds.valid() {
				t.Fatal("structurally valid finite opening fixture", err)
			}
			if mode == "failed-clock" {
				clock.failed = true
			}
			if s, err := openProviderFrontierCandidate(config, clock); err == nil || s != nil {
				t.Fatal("incomplete/unfunded candidate opening succeeded", mode)
			}
			if mode == "clock-cap" || mode == "setup-clock-cap" {
				if _, err := os.Stat(filepath.Join(config.custodyDirectory, providerCustodyName("setup-intent", 0))); err != nil {
					t.Fatal("a refused setup lost its original charged claim", err)
				}
			}
			// These roots are unqualified test subjects. Refusal never means
			// authenticated absence, recovery permission or an operating Source.
		})
	}
}

func TestProviderFrontierCustodyRefusesInvalidDeltaBeforeImmutableWrites(t *testing.T) {
	for _, mode := range []string{"empty", "raw", "duplicate", "existing-reference", "unknown-kind", "missing-body", "resident", "physical", "clock"} {
		t.Run(mode, func(t *testing.T) {
			f, u, r := providerFrontierWriterFixture(t)
			s := f.store
			record := u.records[0]
			fresh := append([]providerIntegrationObject(nil), record.fresh...)
			h := providerFrontierNextFileHead(t, f, record)
			switch mode {
			case "empty":
				fresh = nil
			case "raw":
				fresh[0].raw = ""
			case "duplicate":
				fresh = []providerIntegrationObject{fresh[0], fresh[0]}
			case "existing-reference":
				ref := fresh[0]
				s.objects[ref.kind+":"+ref.digest] = providerIntegrationObjectRef{ref.kind, ref.digest, uint64(len(ref.raw))}
			case "unknown-kind":
				fresh[0].kind = "unknown"
			case "missing-body":
				r.body = nil
			case "resident":
				// Crossed cached DATA must refuse; no selected profile is reset.
				r.indexes.bounds.maximumResidentBytes = 1
			case "physical":
				s.stored = s.config.bounds.maximumStoredBytes
			case "clock":
				s.clock.failed = true
			}
			if !s.valid() || !checkProviderHighwaterMain(f.pin, h, s.config, record.intent, record.commit, record.frontier) {
				t.Fatal("finite delta must reach the post-ownership preflight")
			}
			stored, head, objects := s.stored, f.pinRaw, len(s.objects)
			if _, err := r.retain(s, record.inventory, fresh, record.intent, record.commit, record.frontier, h); err == nil || !s.failed || !f.failed || !r.failed {
				t.Fatal("invalid/unsupported delta remained reusable", mode, err)
			}
			if s.stored != stored || s.sequence != 0 || len(s.objects) != objects || f.pinRaw != head || f.pending != nil || !f.readExact("HEAD", []byte(head)) {
				t.Fatal("early refusal changed retained originals or accounting", mode)
			}
		})
	}
}

func TestProviderFrontierCustodyUncertainTailRetainsChargeAndOriginalBytes(t *testing.T) {
	for _, mode := range []string{"intent", "object", "inventory"} {
		t.Run(mode, func(t *testing.T) {
			f, u, r := providerFrontierWriterFixture(t)
			s, record := f.store, u.records[0]
			disk, name := s.custody, providerCustodyName("intent", 1)
			switch mode {
			case "object":
				disk, name = s.evidence, providerObjectName(record.fresh[0].kind, record.fresh[0].digest)
			case "inventory":
				disk, name = s.evidence, "inventory-"+record.inventory.digest[2:]
			}
			// Finite crash-tail UNIT state in ordinary owned roots. No helper
			// process, malicious filesystem producer or host bypass is used.
			original := []byte("finite pre-existing unit tail")
			if disk.write(name, original) != nil {
				t.Fatal("normal exclusive tail fixture")
			}
			stored, head := s.stored, f.pinRaw
			h := providerFrontierNextFileHead(t, f, record)
			if _, err := r.retain(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h); err == nil || !s.failed || !f.failed || !r.failed || f.pending == nil || s.stored <= stored || s.sequence != 0 || f.pinRaw != head {
				t.Fatal("uncertain retained tail was refunded, adopted or retried", mode)
			}
			raw, err := disk.read(name, 64<<10)
			if err != nil || !bytes.Equal(raw, original) || !f.readExact("HEAD", []byte(head)) {
				t.Fatal("refusal replaced an existing original", mode, err)
			}
		})
	}
}

func providerFrontierRebindUnitIntent(t *testing.T, id ProviderCandidateIdentity, record providerFrontierUnitRecord, kind string, badClock bool) providerFrontierUnitRecord {
	t.Helper()
	value := contentJSON(record.intent).(map[string]any)
	if kind != "" {
		value["kind"] = kind
	}
	if badClock {
		value["clock"] = nil
	}
	record.intent = providerIntegrationJSON(value)
	record.commit = providerFrontierCommit(record.intent)
	f, err := parseProviderFrontierRecord(record.frontier, id)
	if err != nil {
		t.Fatal("finite canonical record input", err)
	}
	f.commit = providerCandidateHash("sage-provider-custody-commit-v2", record.commit)
	record.frontier, err = f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestProviderFrontierRetentionPreflightRejectsUnfundedOrWrongPhaseData(t *testing.T) {
	for _, mode := range []string{"inventory", "clock", "setup", "head-read", "wrong-phase", "pre-admission-overflow", "pre-admission-payload", "pending", "unsupported-live-kind"} {
		t.Run(mode, func(t *testing.T) {
			f, u, r := providerFrontierWriterFixture(t)
			s, record := f.store, u.records[0]
			if mode == "pre-admission-overflow" || mode == "pre-admission-payload" || mode == "pending" || mode == "unsupported-live-kind" {
				h := providerFrontierNextFileHead(t, f, record)
				if _, err := r.retain(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h); err != nil {
					t.Fatal("normal retained E0", err)
				}
				record = u.records[1]
			}
			if mode == "pending" || mode == "unsupported-live-kind" {
				h := providerFrontierNextFileHead(t, f, record)
				if _, err := r.retain(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, h); err != nil {
					t.Fatal("normal retained T", err)
				}
				consumption := ledgerTestConsumption(t, u.ledger.lastTicket, nil, 0)
				if _, err := u.ledger.AppendCandidateConsumption(consumption); err != nil {
					t.Fatal(err)
				}
				u.transitionData(t, "consume", "consumption", []byte(consumption.raw), 0)
				record = u.records[2]
			}
			inventory := record.inventory
			switch mode {
			case "inventory":
				inventory = nil
			case "clock":
				record = providerFrontierRebindUnitIntent(t, s.config.identity, record, "", true)
			case "setup":
				s.setups = nil
			case "head-read":
				s.activeMeter.Store(&providerWorkMeter{failed: true})
			case "wrong-phase":
				record = providerFrontierRebindUnitIntent(t, s.config.identity, record, "settle", false)
			case "pre-admission-overflow":
				r.preI.metadata = ^uint64(0)
			case "pre-admission-payload":
				r.preI.payload = ^uint64(0)
			case "pending":
				r.body.ledger.continuation.pending = &providerLedgerBoundary{}
			case "unsupported-live-kind":
				record = providerFrontierRebindUnitIntent(t, s.config.identity, record, "verify", false)
			}
			h := providerFrontierNextFileHead(t, f, record)
			if inventory != nil {
				if _, err := providerFrontierRetentionCost(s.config.identity, inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds); err != nil {
					t.Fatal("finite data must reach beyond the byte-quote preflight", mode, err)
				}
			}
			stored, head, state := s.stored, f.pinRaw, r.current
			if r.preflightRetention(s, inventory, record.fresh, record.intent, record.commit, record.frontier, h) == nil {
				t.Fatal("invalid phase/quota data passed preflight", mode)
			}
			s.activeMeter.Store(nil)
			if s.stored != stored || f.pinRaw != head || r.current != state || !f.readExact("HEAD", []byte(head)) {
				t.Fatal("refused pure preflight changed retained state")
			}
		})
	}
	if providerFrontierSealQuotaFits(nil, 1, 1) {
		t.Fatal("absent ledger supplied a seal quota")
	}
}
