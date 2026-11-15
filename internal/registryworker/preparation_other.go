//go:build !darwin

package registryworker

import "os"

func inheritedPreparationPipe() (*os.File, error) {
	return nil, ErrUnsupported
}
