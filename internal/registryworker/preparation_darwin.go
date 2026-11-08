//go:build darwin

package registryworker

import (
	"errors"
	"os"
	"syscall"
)

func inheritedPreparationPipe() (*os.File, error) {
	// Inherited descriptors are blocking. Set NONBLOCK before NewFile so Go
	// registers this pipe with its poller: Close can then interrupt ReadByte.
	// Only this fixed child's inherited descriptor is touched, never a PID or
	// caller-selected descriptor. No native resource qualification is implied.
	if err := syscall.SetNonblock(5, true); err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	return os.NewFile(5, "sage-worker-preparation"), nil
}
