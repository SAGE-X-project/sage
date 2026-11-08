// Package registryrunner composes the fixed worker and public preparation API.
// Library packages do not import this application-level process owner.
package registryrunner

import (
	"context"
	"errors"

	"github.com/sage-x-project/sage/internal/registryworker"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

// Result keeps native attempt evidence separate from non-authority DATA.
// It never asserts full-profile feasibility or permission to collect history.
type Result struct {
	Worker   registryworker.Result
	Prepared registry011.HistoryPreparationSummary
}

// Prepare consumes one fixed native attempt with the caller's explicit limits.
// It does not select, default or reduce a resource profile. The caller must
// supply journal and live preparation channels, including a cleanup reserve.
func Prepare(ctx context.Context, configuration registryworker.Configuration, input []byte) (out Result, err error) {
	if !configuration.RequireJournal || !configuration.RequirePreparation {
		return out, registryworker.ErrConfiguration
	}
	owner, err := registryworker.New(ctx, configuration)
	if err != nil {
		return out, err
	}
	out.Worker, err = owner.Run(input)
	if err != nil {
		return out, err
	}
	// Original process exit, readiness, retained readback, terminal binding and
	// parent cleanup have all passed before ordinary output reaches this point.
	prepared, err := registry011.DecodeHistoryPreparationSummary(ctx, out.Worker.Output, input, configuration.Limits.ResultBytes)
	if err != nil || prepared.OriginUnixNano != out.Worker.PreparationWindow.Origin.UnixNano() ||
		prepared.DeadlineUnixNano != out.Worker.PreparationWindow.ActionDeadline.UnixNano() {
		out.Worker.Output = nil
		return out, errors.Join(registryworker.ErrUnavailable, err)
	}
	out.Prepared = prepared
	return out, nil
}
