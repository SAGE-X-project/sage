package memory

import (
	"testing"

	"github.com/sage-x-project/sage/pkg/storage"
	"github.com/sage-x-project/sage/pkg/storage/storagetest"
)

func TestMemoryStore_Conformance(t *testing.T) {
	storagetest.RunConformance(t, func(t *testing.T) storage.Store { return NewStore() })
}
