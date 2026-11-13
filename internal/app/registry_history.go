package app

import (
	"context"

	"github.com/sage-x-project/sage/internal/registryrunner"
	"github.com/sage-x-project/sage/internal/registryworker"
)

// PrepareRegistryHistory connects application composition to the fixed owned
// preparation worker. Returned values are diagnostic DATA, not an activated
// Source, complete resource qualification or an Agent dispatch permission.
func PrepareRegistryHistory(ctx context.Context, configuration registryworker.Configuration, input []byte) (registryrunner.Result, error) {
	return registryrunner.Prepare(ctx, configuration, input)
}
