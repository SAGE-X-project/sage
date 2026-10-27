package registry011

import "testing"

// Only isolated owned files and finite inert meter/permit states are used.
// Failure cases do not attack a host, change existing bytes or dispatch work.
func TestProviderOwnedIORefusesUnfundedOrUnavailablePrimitives(t *testing.T) {
	for _, mode := range []string{"read-size", "read-budget", "missing-original", "write-budget", "existing-original"} {
		t.Run(mode, func(t *testing.T) {
			s, policy, candidate, primitive := providerStoreFixture(t)
			c, err := newProviderIntegration(s, policy, candidate, primitive)
			if err != nil {
				t.Fatal(err)
			}
			m := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{9: 1024, 10: 1 << 20}})
			s.activeMeter.Store(m)
			defer s.activeMeter.Store(nil)
			original, err := s.custody.read("ANCHOR", 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "read-size":
				_, err = s.read(s.custody, "ANCHOR", 0)
			case "read-budget":
				m.upper[10] = 0
				_, err = s.read(s.custody, "ANCHOR", 64<<10)
			case "missing-original":
				_, err = s.read(s.custody, "not-retained", 64<<10)
			case "write-budget":
				m.upper[9] = 0
				err = s.write(s.custody, "not-written", []byte("finite"))
			case "existing-original":
				err = s.write(s.custody, "ANCHOR", []byte("finite"))
			}
			if err == nil || mode != "read-size" && !m.failed {
				t.Fatal("unobserved/unfunded IO remained eligible to complete")
			}
			retained, err := s.custody.read("ANCHOR", 64<<10)
			if err != nil || string(retained) != string(original) {
				t.Fatal("refusal changed an existing original", err)
			}
		})
	}
}

func TestProviderOwnedMeterCannotBorrowForeignOrNestedAttribution(t *testing.T) {
	for _, mode := range []string{"foreign", "authority", "closed", "failed", "nested"} {
		t.Run(mode, func(t *testing.T) {
			s, policy, candidate, primitive := providerStoreFixture(t)
			c, err := newProviderIntegration(s, policy, candidate, primitive)
			if err != nil {
				t.Fatal(err)
			}
			parent := newProviderWorkMeter(c, providerOperationReservation{})
			switch mode {
			case "foreign":
				parent.integration = &providerIntegration{}
			case "authority":
				parent.authority = true
			case "closed":
				parent.closed = true
			case "failed":
				parent.failed = true
			case "nested":
				parent.parent = &providerWorkMeter{}
			}
			s.activeMeter.Store(parent)
			defer s.activeMeter.Store(nil)
			if value, err := c.attachMeter(providerOperationReservation{}); err == nil || value != nil || s.activeMeter.Load() != parent {
				t.Fatal("invalid parent supplied attribution for another operation")
			}
		})
	}
}

func TestProviderOwnedMeterRestorationPoisonsUnknownOwnership(t *testing.T) {
	for _, mode := range []string{"nil", "foreign-meter", "foreign-active", "wrong-ticket", "valid-parent"} {
		t.Run(mode, func(t *testing.T) {
			s, policy, candidate, primitive := providerStoreFixture(t)
			c, err := newProviderIntegration(s, policy, candidate, primitive)
			if err != nil {
				t.Fatal(err)
			}
			parent := newProviderWorkMeter(c, providerOperationReservation{})
			m := newProviderWorkMeter(c, providerOperationReservation{})
			m.ticket, m.parent = admissionTestDigest("a"), parent
			active := m
			switch mode {
			case "nil":
				m = nil
			case "foreign-meter":
				m.integration = &providerIntegration{}
			case "foreign-active":
				active = newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{})
			case "wrong-ticket":
				active = newProviderWorkMeter(c, providerOperationReservation{})
				active.authority, active.ticket = true, admissionTestDigest("b")
			}
			s.activeMeter.Store(active)
			defer s.activeMeter.Store(nil)
			c.restoreMeter(m)
			if mode == "valid-parent" {
				if c.failed || s.activeMeter.Load() != parent {
					t.Fatal("exact owned parent was not restored")
				}
			} else if !c.failed || s.activeMeter.Load() != active {
				t.Fatal("unknown attribution was adopted or left usable")
			}
		})
	}
}
