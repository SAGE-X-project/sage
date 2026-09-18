package guard010

import "testing"

func TestEmptyVerificationCannotCreateProjection(t *testing.T) {
	for _, v := range []*VerifiedIntent{nil, {}} {
		if _, e := reservationEntry(v); e == nil {
			t.Fatal("empty receipt accepted")
		}
	}
}
