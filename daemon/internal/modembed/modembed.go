package modembed

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:mod
var ModFS embed.FS

// ReadModFile reads a file from the embedded mod directory
func ReadModFile(relPath string) ([]byte, error) {
	fullPath := "mod/" + relPath
	data, err := fs.ReadFile(ModFS, fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading embedded mod file %s: %w", fullPath, err)
	}
	return data, nil
}
