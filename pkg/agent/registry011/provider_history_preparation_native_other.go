//go:build !darwin

package registry011

import (
	"context"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// NewNativeHistoryPreparationSession refuses an unavailable platform. Callers
// must explicitly select the separate Go diagnostic API if they intend to use
// it; native refusal never selects that API or starts a preparation attempt.
func NewNativeHistoryPreparationSession(action, cleanup context.Context, origin time.Time, maximumInput, maximumResult int) (*HistoryPreparationSession, error) {
	return nil, compilermemory.ErrUnsupported
}
