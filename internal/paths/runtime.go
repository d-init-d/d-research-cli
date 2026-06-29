package paths

import (
	"os"
	"path/filepath"
)

const (
	RuntimeDirName       = "runtime"
	ResearchSkillSubdir  = "d-research-skill"
	AlephSubdir          = "aleph"
	ScriptsSubdir        = "scripts"
)

// RuntimeRoot resolves the pinned runtime directory without depending on CWD.
// Resolution order: D_RESEARCH_RUNTIME env → adjacent to executable → walk up from CWD.
func RuntimeRoot() (string, error) {
	if env := os.Getenv("D_RESEARCH_RUNTIME"); env != "" {
		if ok, err := hasRuntimeMarkers(env); err != nil {
			return "", err
		} else if ok {
			return filepath.Clean(env), nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(dir, RuntimeDirName)
			if ok, _ := hasRuntimeMarkers(candidate); ok {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, RuntimeDirName)
			if ok, _ := hasRuntimeMarkers(candidate); ok {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return filepath.Join(RuntimeDirName), nil
}

func hasRuntimeMarkers(root string) (bool, error) {
	skill := filepath.Join(root, ResearchSkillSubdir)
	aleph := filepath.Join(root, AlephSubdir)
	skillInfo, skillErr := os.Stat(skill)
	alephInfo, alephErr := os.Stat(aleph)
	if skillErr != nil && alephErr != nil {
		if os.IsNotExist(skillErr) && os.IsNotExist(alephErr) {
			return false, nil
		}
		return false, skillErr
	}
	if skillInfo != nil && skillInfo.IsDir() {
		return true, nil
	}
	if alephInfo != nil && alephInfo.IsDir() {
		return true, nil
	}
	return false, nil
}

func ResearchSkillRoot() (string, error) {
	root, err := RuntimeRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ResearchSkillSubdir), nil
}

func AlephRoot() (string, error) {
	root, err := RuntimeRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, AlephSubdir), nil
}

func ScriptsRoot() (string, error) {
	if env := os.Getenv("D_RESEARCH_SCRIPTS"); env != "" {
		return filepath.Clean(env), nil
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(dir, ScriptsSubdir)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, ScriptsSubdir)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ScriptsSubdir, nil
}

func ScriptPath(name string) (string, error) {
	root, err := ScriptsRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}