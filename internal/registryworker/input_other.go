//go:build !darwin

package registryworker

func preparationInputNativeMemory() (preparationMemoryOperations, error) {
	return preparationMemoryOperations{}, ErrUnsupported
}
