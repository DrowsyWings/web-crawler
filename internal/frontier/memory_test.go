package frontier

import "testing"

func TestMemoryConformance(t *testing.T) {
	runConformance(t, func() Frontier { return NewMemory(1000) })
}
