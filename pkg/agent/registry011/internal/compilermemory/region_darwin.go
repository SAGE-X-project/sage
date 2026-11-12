package compilermemory

import "syscall"

func mapRegion(capacity int) ([]byte, error) {
	return syscall.Mmap(-1, 0, capacity, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
}

func lockRegion(data []byte) error   { return syscall.Mlock(data) }
func unlockRegion(data []byte) error { return syscall.Munlock(data) }
func unmapRegion(data []byte) error  { return syscall.Munmap(data) }
