//go:build !darwin

package compilermemory

func mapRegion(int) ([]byte, error) { return nil, ErrUnsupported }
func lockRegion([]byte) error      { return ErrUnsupported }
func unlockRegion([]byte) error    { return ErrUnsupported }
func unmapRegion([]byte) error     { return ErrUnsupported }
