package config

import (
	"path/filepath"
	"strings"
)

func (cfg *Config) GetPkgPath(dir string) string {
	return cfg.GetDirPkgPath(filepath.Dir(dir))
}

func (cfg *Config) GetDirPkgPath(dir string) string {
	if strings.HasPrefix(dir, cfg.WorkDir) {
		dir = strings.TrimPrefix(dir, cfg.WorkDir)
		dir = strings.TrimPrefix(dir, "vendor/")
		return dir
	}
	if strings.HasPrefix(dir, cfg.GoRoot) {
		return strings.TrimPrefix(dir, cfg.GoRoot)
	}

	return dir
}
