package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Normal isolated filesystem subjects only. These fixtures are unqualified:
// a file transition never selects an operating principal, clock or Source and
// returns no protected-message/dispatch acknowledgment or permission.
func providerFrontierFileFixture(t *testing.T) (*providerFrontierFiles, *providerFrontierUnit) {
	return providerFrontierFileFixtureWithClock(t, nil)
}

func providerFrontierFileFixtureWithClock(t *testing.T, owned *providerOwnedClock) (*providerFrontierFiles, *providerFrontierUnit) {
	t.Helper()
	u := providerFrontierUnitFixtureWithClock(t, owned)
	return providerFrontierFileFixtureFromUnit(t, owned, u)
}

func providerFrontierFileFixtureFromUnit(t *testing.T, owned *providerOwnedClock, u *providerFrontierUnit) (*providerFrontierFiles, *providerFrontierUnit) {
	t.Helper()
	config := u.s.config
	root := t.TempDir()
	config.custodyDirectory, config.evidenceDirectory = filepath.Join(root, "custody"), filepath.Join(root, "evidence")
	anchor, err := u.s.read(u.s.evidence, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{config.custodyDirectory, config.evidenceDirectory} {
		if os.Mkdir(name, 0700) != nil || os.WriteFile(filepath.Join(name, "AUTHORITY.LOCK"), nil, 0600) != nil || os.WriteFile(filepath.Join(name, "ANCHOR"), anchor, 0600) != nil {
			t.Fatal("fresh normal owned file fixture")
		}
	}
	custody, err := openRetentionDisk(config.custodyDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = custody.close() })
	evidence, err := openRetentionDisk(config.evidenceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = evidence.close() })
	empty, err := newProviderFrontierInventory("", nil, config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := &providerCustodyStore{config: config, custody: custody, evidence: evidence, clock: owned, tip: config.anchor, objectsRoot: empty.digest, setupOrigin: u.clock, setupTip: u.s.setupTip, setups: append([]providerSetupReference(nil), u.s.setups...), stored: config.bounds.setupRetained, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	s.owner = s
	claim := providerCustodySetup(config, u.clock)
	commit := providerSetupCommit(claim)
	if custody.write(providerCustodyName("setup-intent", 0), claim) != nil || evidence.write(providerCustodyName("setup-claim", 0), claim) != nil || custody.write(providerCustodyName("setup-commit", 0), commit) != nil {
		t.Fatal("normal finite original setup prefix")
	}
	f, err := newProviderFrontierFiles(s)
	if err != nil {
		t.Fatal(err)
	}
	return f, u
}

func TestProviderFrontierFilesInitializeAndRetainExactOldHighwater(t *testing.T) {
	f, u := providerFrontierFileFixture(t)
	initial, err := f.initialize()
	if err != nil || !f.initialized || f.failed || initial.metadata != 2*uint64(len(f.pinRaw)) || f.store.stored != f.store.config.bounds.setupRetained {
		t.Fatal("initial two exact copies within the pre-I envelope", err)
	}
	old, oldRaw := f.pin, f.pinRaw
	record, s := u.records[0], f.store
	mainCost, err := providerFrontierRetentionCost(s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds)
	if err != nil || !candidateCapacity(0, s.stored, mainCost.stored, s.config.bounds.maximumStoredBytes) {
		t.Fatal("finite normal main fixture", err)
	}
	s.stored += mainCost.stored
	for _, object := range record.fresh {
		if s.evidence.write(providerObjectName(object.kind, object.digest), []byte(object.raw)) != nil {
			t.Fatal("normal original object retention")
		}
		s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	if s.custody.write(providerCustodyName("intent", 1), record.intent) != nil || s.evidence.write("inventory-"+record.inventory.digest[2:], []byte(record.inventory.raw)) != nil || s.evidence.write(providerCustodyName("pending", 1), record.intent) != nil || s.custody.write(providerCustodyName("commit", 1), record.commit) != nil {
		t.Fatal("normal original main metadata retention")
	}
	replayed, err := newProviderFrontierReplay(s.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replayed.apply(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier); err != nil {
		t.Fatal("independently reconstructed original E0", err)
	}
	if s.custody.write(providerCustodyName("frontier", 1), record.frontier) != nil {
		t.Fatal("normal derived frontier retention")
	}
	after := old
	after.ordinal, after.sequence, after.previous = 1, 1, providerCandidateHash("sage-provider-custody-highwater-v1", []byte(oldRaw))
	after.commit, after.frontier = replayed.current.commit, providerCandidateHash("sage-provider-custody-frontier-v1", record.frontier)
	beforeStored := s.stored
	cost, err := f.advance(after, "main", record.intent, record.commit, record.frontier)
	if err != nil || f.failed || f.pin != after || s.stored != beforeStored+cost.stored || cost.metadata != 2*uint64(len(f.pinRaw)) || !f.readExact("HEAD", []byte(f.pinRaw)) || !f.readExact(providerCustodyName("highwater", 0), []byte(oldRaw)) || !f.readExact(providerCustodyName("highwater", 1), []byte(f.pinRaw)) {
		t.Fatal("exact candidate file transition/readback/old original", err)
	}
	if _, err := s.custody.root.Lstat(providerCustodyName("HEAD.pending", 1)); !os.IsNotExist(err) {
		t.Fatal("only the owned successful pending file must become HEAD", err)
	}
	// The file pin is metadata only. This fixture has no qualified principal,
	// original owned clock/scan artifact, operating ACK or protected Source.
	if replayed.body.intent != nil || replayed.body.ledger != nil {
		t.Fatal("E0 file transition manufactured a future I/ledger")
	}
}

func TestProviderFrontierFilesRefuseMalformedInputBeforeNewWrites(t *testing.T) {
	f, _ := providerFrontierFileFixture(t)
	if _, err := f.initialize(); err != nil {
		t.Fatal(err)
	}
	old := f.pinRaw
	if _, err := f.advance(providerFrontierHighwater{}, "unknown", nil, nil, nil); err == nil || !f.failed || !f.store.failed {
		t.Fatal("malformed inert transition kept an owner reusable")
	}
	raw, err := f.store.read(f.store.custody, "HEAD", candidateLedgerMetadataLimit)
	if err != nil || !bytes.Equal(raw, []byte(old)) {
		t.Fatal("format refusal changed a normal original HEAD", err)
	}
	if _, err := f.initialize(); err == nil {
		t.Fatal("poisoned candidate automatically reinitialized")
	}
	if _, err := newProviderFrontierFiles(nil); err == nil {
		t.Fatal("nil owner created filesystem state")
	}
}

func TestProviderFrontierFilesRefuseFailedClosedOrUnboundOwner(t *testing.T) {
	for _, state := range []string{"failed", "closed", "backlink", "owner"} {
		t.Run(state, func(t *testing.T) {
			f, _ := providerFrontierFileFixture(t)
			s := f.store
			s.mu.Lock()
			switch state {
			case "failed":
				s.failed = true
			case "closed":
				s.closed = true // Inert state refusal; cleanup still owns both disks.
			case "backlink":
				s.frontierFiles = nil
			case "owner":
				f.owner = nil
			}
			stored, retained := s.stored, s.retainedRaw
			s.mu.Unlock()
			if _, err := f.initialize(); err == nil {
				t.Fatal("invalid owner initialized a file transition")
			}
			if _, err := f.advance(providerFrontierHighwater{}, "main", nil, nil, nil); err == nil {
				t.Fatal("invalid owner advanced a file transition")
			}
			if s.stored != stored || s.retainedRaw != retained {
				t.Fatal("refused owner changed retention accounting")
			}
			if _, err := s.custody.root.Lstat("HEAD"); !os.IsNotExist(err) {
				t.Fatal("refused owner wrote a high-water head", err)
			}
		})
	}
}

func TestProviderFrontierFilesSerializeNormalInitializeAndClose(t *testing.T) {
	f, _ := providerFrontierFileFixture(t)
	start := make(chan struct{})
	initialized, closed := make(chan error, 1), make(chan error, 1)
	go func() {
		<-start
		_, err := f.initialize()
		initialized <- err
	}()
	go func() {
		<-start
		closed <- f.store.close()
	}()
	close(start)
	initErr, closeErr := <-initialized, <-closed
	if closeErr != nil || !f.store.closed || f.store.failed || f.failed {
		t.Fatal("normal close and initialization were not serialized", initErr, closeErr)
	}
	// Either entire initialization precedes close, or close refuses it before
	// its first write. Neither order creates an operating ACK or permission.
	if (initErr == nil) != f.initialized || f.initialized != (f.pinRaw != "") {
		t.Fatal("partial initialization was exposed across close")
	}
	if _, err := f.initialize(); err == nil {
		t.Fatal("closed store became reusable")
	}
}

func TestProviderHighwaterFileCostCountsTwoCopiesAndTwoAllocations(t *testing.T) {
	b, _ := providerFrontierBoundsFixture()
	for _, n := range []uint64{1, 1024, candidateLedgerMetadataLimit} {
		cost, ok := providerHighwaterFileCost(n, b)
		if !ok || cost.metadata != 2*n || cost.stored != 2*n+2*b.fileOverheadBytes || cost.payload != 0 || cost.objects != 0 {
			t.Fatal("head rename became a third payload copy or a refund")
		}
	}
	for _, n := range []uint64{0, candidateLedgerMetadataLimit + 1, ^uint64(0)} {
		if _, ok := providerHighwaterFileCost(n, b); ok {
			t.Fatal("unbounded high-water cost")
		}
	}
}
