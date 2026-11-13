package registryworker

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// These operation faults are unit DATA, not real native resource evidence.
func TestPreparationInputNativeFailuresKeepEveryCleanupError(t *testing.T) {
	for _, name := range []string{"success", "map-failure", "short-mapping", "lock-failure", "lock-unmap-failure", "unlock-failure", "unmap-failure", "unlock-unmap-failure"} {
		t.Run(name, func(t *testing.T) {
			mapping := make([]byte, 16)
			var calls []string
			fault := errors.New("native unit failure")
			secondFault := errors.New("native cleanup unit failure")
			operations := preparationMemoryOperations{
				page: 16,
				mapBytes: func(size int) ([]byte, error) {
					calls = append(calls, "map")
					if size != len(mapping) {
						t.Fatal("page rounding changed", size)
					}
					if name == "map-failure" {
						return nil, fault
					}
					if name == "short-mapping" {
						return mapping[:8], nil
					}
					return mapping, nil
				},
				lock: func(raw []byte) error {
					calls = append(calls, "lock")
					if len(raw) != len(mapping) || &raw[0] != &mapping[0] {
						t.Fatal("lock received another region")
					}
					if name == "lock-failure" || name == "lock-unmap-failure" {
						return fault
					}
					return nil
				},
				unlock: func(raw []byte) error {
					calls = append(calls, "unlock")
					if raw[0] != 0 {
						t.Fatal("raw input was not cleared before native release")
					}
					if name == "unlock-failure" || name == "unlock-unmap-failure" {
						return fault
					}
					return nil
				},
				unmap: func([]byte) error {
					calls = append(calls, "unmap")
					if name == "lock-unmap-failure" || name == "unmap-failure" || name == "unlock-unmap-failure" {
						return secondFault
					}
					return nil
				},
			}
			memory, err := acquirePreparationInputMemoryWith(8, operations)
			if name == "map-failure" {
				if memory != nil || !errors.Is(err, fault) || !reflect.DeepEqual(calls, []string{"map"}) {
					t.Fatal("failed mapping escaped or was retried", err, calls)
				}
				return
			}
			if name == "short-mapping" || name == "lock-failure" || name == "lock-unmap-failure" {
				if memory != nil || !errors.Is(err, ErrUnavailable) || calls[len(calls)-1] != "unmap" {
					t.Fatal("partial acquisition was not refused and reclaimed", err, calls)
				}
				if name != "short-mapping" && !errors.Is(err, fault) || name == "lock-unmap-failure" && !errors.Is(err, secondFault) {
					t.Fatal("native acquisition or cleanup error was lost", err)
				}
				return
			}
			if err != nil || memory == nil || &memory.data[0] != &mapping[0] {
				t.Fatal("successful lock did not retain the original mapping", err)
			}
			memory.data[0] = 7
			err = memory.release(context.Background())
			if name == "success" && err != nil || name != "success" && err == nil {
				t.Fatal("native release outcome changed", err)
			}
			if (name == "unlock-failure" || name == "unlock-unmap-failure") && !errors.Is(err, fault) {
				t.Fatal("unlock failure disappeared", err)
			}
			if (name == "unmap-failure" || name == "unlock-unmap-failure") && !errors.Is(err, secondFault) {
				t.Fatal("unmap failure disappeared", err)
			}
			if !reflect.DeepEqual(calls, []string{"map", "lock", "unlock", "unmap"}) {
				t.Fatal("native release was skipped or reordered", calls)
			}
		})
	}
}

func TestPreparationInputRefusesIncompleteMemoryOperations(t *testing.T) {
	for _, name := range []string{"bad-maximum", "bad-page", "missing-map", "missing-lock", "missing-unlock", "missing-unmap"} {
		t.Run(name, func(t *testing.T) {
			unexpected := false
			operations := preparationMemoryOperations{
				page: 16,
				mapBytes: func(int) ([]byte, error) {
					unexpected = true
					return nil, nil
				},
				lock: func([]byte) error { return nil }, unlock: func([]byte) error { return nil }, unmap: func([]byte) error { return nil },
			}
			maximum := 8
			switch name {
			case "bad-maximum":
				maximum = 0
			case "bad-page":
				operations.page = 0
			case "missing-map":
				operations.mapBytes = nil
			case "missing-lock":
				operations.lock = nil
			case "missing-unlock":
				operations.unlock = nil
			case "missing-unmap":
				operations.unmap = nil
			}
			if memory, err := acquirePreparationInputMemoryWith(maximum, operations); memory != nil || !errors.Is(err, ErrConfiguration) || unexpected {
				t.Fatal("invalid native operations reached allocation", err)
			}
		})
	}
}
