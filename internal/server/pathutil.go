package server

import (
	"fmt"
	"path/filepath"
	"strings"
)

func SafePath(root, reqPath string) (string, error) {
	if filepath.IsAbs(reqPath) {
		return "", fmt.Errorf("absolute path not allowed: %s", reqPath)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("cannot resolve root: %w", err)
	}
	joined := filepath.Join(absRoot, reqPath)
	cleaned := filepath.Clean(joined)
	if !strings.HasPrefix(cleaned, absRoot+string(filepath.Separator)) && cleaned != absRoot {
		return "", fmt.Errorf("path traversal detected: %s", reqPath)
	}
	return cleaned, nil
}
