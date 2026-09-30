package registry010

import "context"

// WebRegistryAdminAuthority010 is supplied by a trusted deployment binding.
// AuthenticatedActor must derive identity from verified transport credentials,
// never a caller-provided record or JSON field. Delegated must read a
// controller-authorized, operation-scoped management state for the expected
// version; this package does not define that state or its wire format.
type WebRegistryAdminAuthority010 interface {
	AuthenticatedActor(context.Context) (string, error)
	Delegated(context.Context, string, string, string, string, string) (bool, error)
}

func webAdminActor010(ctx context.Context, authority WebRegistryAdminAuthority010) (string, error) {
	if ctx == nil || ctx.Err() != nil || authority == nil {
		return "", ErrRejected
	}
	actor, err := authority.AuthenticatedActor(ctx)
	if err != nil || actor == "" || len(actor) > 256 || !webASCII010(actor, 1, 256) {
		return "", ErrRejected
	}
	return actor, nil
}

// CheckWebRegistryCreationAdmission010 checks a proposed creation against
// the deployment-authenticated controller. It does not reserve a name or
// commit an atomic write; admission must be repeated inside the transaction.
func CheckWebRegistryCreationAdmission010(ctx context.Context, authority WebRegistryAdminAuthority010, candidate []byte, did string, now int64) error {
	actor, err := webAdminActor010(ctx, authority)
	if err != nil {
		return err
	}
	if err := CheckWebRegistryCreationShape010(candidate, did, now); err != nil {
		return err
	}
	record, err := webTransitionRecordFromEnvelope010(candidate, did, now)
	if err != nil {
		return err
	}
	if actor != record.Controller {
		return ErrRejected
	}
	return nil
}

// CheckWebRegistryMutationAdmission010 checks expected-version and authority
// policy for one proposed mutation. A delegated actor may perform only the
// exact operation approved by the trusted management-plane authority.
// Authentication, delegation state, current version and the record write
// must be bound in one atomic deployment transaction; this predicate alone
// grants no durable write authority.
func CheckWebRegistryMutationAdmission010(ctx context.Context, authority WebRegistryAdminAuthority010, previous, candidate []byte, did string, now int64, expectedVersion, operation string) error {
	actor, err := webAdminActor010(ctx, authority)
	if err != nil {
		return err
	}
	before, err := webTransitionRecordFromEnvelope010(previous, did, now)
	if err != nil {
		return err
	}
	if expectedVersion != before.Version {
		return ErrStale
	}
	if actor != before.Controller {
		allowed, authErr := authority.Delegated(ctx, before.Controller, actor, did, operation, expectedVersion)
		if authErr != nil || !allowed || ctx.Err() != nil {
			return ErrRejected
		}
	}
	return CheckWebRegistryTransitionShape010(previous, candidate, did, now, now, operation)
}
