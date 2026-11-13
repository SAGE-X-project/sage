package registryworker

import "errors"

// Production obtains these functions only from the fixed platform adapter.
// Unit fault injection cannot make a Registry resource or derivation witness.
type preparationMemoryOperations struct {
	page                int
	mapBytes            func(int) ([]byte, error)
	lock, unlock, unmap func([]byte) error
}

func acquirePreparationInputMemory(maximum int) (*preparationInputMemory, error) {
	operations, err := preparationInputNativeMemory()
	if err != nil {
		return nil, err
	}
	return acquirePreparationInputMemoryWith(maximum, operations)
}

func acquirePreparationInputMemoryWith(maximum int, operations preparationMemoryOperations) (*preparationInputMemory, error) {
	size, ok := preparationInputMappingBytes(maximum, operations.page)
	if !ok || operations.mapBytes == nil || operations.lock == nil || operations.unlock == nil || operations.unmap == nil {
		return nil, ErrConfiguration
	}
	memory, err := operations.mapBytes(size)
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	if len(memory) != size {
		return nil, errors.Join(ErrUnavailable, operations.unmap(memory))
	}
	if err := operations.lock(memory); err != nil {
		return nil, errors.Join(ErrUnavailable, err, operations.unmap(memory))
	}
	return &preparationInputMemory{data: memory, close: func() error {
		clear(memory)
		unlockErr := operations.unlock(memory)
		unmapErr := operations.unmap(memory)
		return errors.Join(unlockErr, unmapErr)
	}}, nil
}
