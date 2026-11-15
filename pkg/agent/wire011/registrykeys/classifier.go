package registrykeys

import (
	"context"
	"fmt"
	"strings"
)

// Classifier marks every DID of this Registry that has a record, in any
// state, as a SAGE peer. A deactivated agent is therefore still verified and
// refused, never downgraded to an unprotected peer. DIDs of other methods or
// Registries have no SAGE evidence here.
type Classifier struct {
	Source     RecordSource
	RegistryID string
}

func (c *Classifier) RequiresSAGE(ctx context.Context, peerDID, _ string) (bool, error) {
	if c == nil || c.Source == nil || c.RegistryID == "" {
		return false, fmt.Errorf("registrykeys: classifier not configured")
	}
	agentID, ok := strings.CutPrefix(peerDID, "did:sage:"+c.RegistryID+":")
	if !ok || agentID == "" || strings.ContainsAny(agentID, ":#/") {
		return false, nil
	}
	present, _, err := c.Source.Record(ctx, agentID)
	if err != nil {
		return false, fmt.Errorf("registrykeys: read record: %w", err)
	}
	return present, nil
}
