package paths

import (
	"os"
	"path/filepath"
)

const (
	AppName           = "d-research"
	KeyringService    = "d-research-cli"
	GlobalConfigDir   = "d-research"
	ProjectMetaDir    = ".d-research"
	ResearchOutputDir = "research-output"
	SimulationDir     = "simulation"
	KBDir             = "kb"
)

func GlobalConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, GlobalConfigDir, "config.json"), nil
}

func LocalAppDataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	dir := filepath.Join(base, GlobalConfigDir)
	return dir, ensureDir(dir)
}

func BrowserDir() (string, error) {
	local, err := LocalAppDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(local, "browsers")
	return dir, ensureDir(dir)
}

func CacheDir() (string, error) {
	local, err := LocalAppDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(local, "cache")
	return dir, ensureDir(dir)
}

func LogDir() (string, error) {
	local, err := LocalAppDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(local, "logs")
	return dir, ensureDir(dir)
}

func ProjectConfigPath(cwd string) string {
	return filepath.Join(cwd, ProjectMetaDir, "config.json")
}

func ProjectStateDir(cwd string) string {
	return filepath.Join(cwd, ProjectMetaDir)
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}