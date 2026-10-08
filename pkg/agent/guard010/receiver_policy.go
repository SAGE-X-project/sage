package guard010

import (
	"context"
	"reflect"
)

// ReceiverMapping is protected receiver administration. Approved returns the
// provisioned policy descriptor and component manifest that trusted
// administration mapped to exactly this issuer and policy digest, and fails
// for unknown, retired or issuer-mismatched commitments. Authorize evaluates
// the permitted receiver operation for the complete final arguments. Both must
// be bounded, cancellation-aware and concurrency-safe. A descriptor supplied
// by a call never provisions or approves itself.
type ReceiverMapping interface {
	Approved(ctx context.Context, issuer, policyDigest string) (policy, manifest []byte, err error)
	Authorize(ctx context.Context, issuer, tool string, args []byte) error
}

// ReceiverPolicy adapts a ReceiverMapping for receiver verification. It is
// accepted only by VerifyReceivedIntent and the receiver-side gates built on
// it. As an IntentPolicy its Bindings always fails, so it cannot approve
// issuance or verify on behalf of a Client that holds the original.
type ReceiverPolicy struct{ mapping ReceiverMapping }

// NewReceiverPolicy wraps a required mapping.
func NewReceiverPolicy(m ReceiverMapping) (*ReceiverPolicy, error) {
	if absentPolicy(m) {
		return nil, ErrInvalid
	}
	return &ReceiverPolicy{mapping: m}, nil
}

// Bindings always fails: a receiver mapping holds no original commitment.
func (*ReceiverPolicy) Bindings(context.Context, string, string) (string, []byte, []byte, error) {
	return "", nil, nil, ErrInvalid
}

// Authorize delegates to the mapping's evaluator.
func (r *ReceiverPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	if r == nil || r.mapping == nil {
		return ErrInvalid
	}
	return r.mapping.Authorize(ctx, issuer, tool, args)
}

func absentPolicy(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}
