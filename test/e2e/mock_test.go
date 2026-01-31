package e2e

import (
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/internal/mock"
)

func TestMockPlugin(t *testing.T) {
	t.Run("TestMockPlugin", func(t *testing.T) {
		plugin := mock.NewMockPlugin("test", "test", nil)
		if plugin == nil {
			t.Errorf("Plugin is nil")
		}
	})
}
