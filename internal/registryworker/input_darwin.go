//go:build darwin

package registryworker

import (
	"os"
	"syscall"
)

func preparationInputNativeMemory() (preparationMemoryOperations, error) {
	return preparationMemoryOperations{
		page: os.Getpagesize(),
		mapBytes: func(size int) ([]byte, error) {
			return syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		},
		lock: syscall.Mlock, unlock: syscall.Munlock, unmap: syscall.Munmap,
	}, nil
}
