package jsplugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SGoerzen/LRSolid/internal/plugins"
)

// LoadDir reads every *.js file in dir and registers them with registry.
// The plugin ID is the filename without the .js extension.
// Returns the number of plugins loaded.
func LoadDir(dir string, registry *plugins.Registry) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("jsplugin: read dir %s: %w", dir, err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".js")
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return count, fmt.Errorf("jsplugin: read %s: %w", e.Name(), err)
		}
		p, err := New(id, string(src))
		if err != nil {
			return count, fmt.Errorf("jsplugin: load %s: %w", e.Name(), err)
		}
		registry.Register(p)
		count++
	}
	return count, nil
}
