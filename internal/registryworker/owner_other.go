//go:build !darwin

package registryworker

import (
	"context"
	"os"
	"os/exec"
)

func platformSupported() bool { return false }

func platformCommand(context.Context, string, string, Limits) (*exec.Cmd, error) {
	return nil, ErrUnsupported
}

func platformReadiness([]byte, Limits) bool { return false }

func platformSignal(*os.ProcessState) int { return 0 }
