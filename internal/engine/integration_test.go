package engine

import (
	"testing"
	"time"
)

func TestCoordinator(t *testing.T) {
	t.Run("TestCoordinator", func(t *testing.T) {
		coordinator := NewCoordinator(CoordinatorConfig{
			MaxParallelEngines: 4,
			Timeout:            1 * time.Minute,
		})

		if coordinator == nil {
			t.Errorf("Coordinator is nil")
		}
	})
}
