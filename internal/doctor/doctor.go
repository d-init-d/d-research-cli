package doctor

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/d-init-d/d-research-cli/internal/browser"
	"github.com/d-init-d/d-research-cli/internal/config"
	"github.com/d-init-d/d-research-cli/internal/paths"
	"github.com/d-init-d/d-research-cli/internal/version"
)

const (
	StatusReady    = "ready"
	StatusDegraded = "degraded"
	StatusFailed   = "failed"
)

type Report struct {
	Version string  `json:"version"`
	OS      string  `json:"os"`
	Arch    string  `json:"arch"`
	Status  string  `json:"status"`
	Checks  []Check `json:"checks"`
	OK      bool    `json:"ok"`
}

type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Warning bool   `json:"warning,omitempty"`
}

func Run(cfg config.Config, cwd string) Report {
	rep := Report{
		Version: version.String(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	rep.Checks = append(rep.Checks, checkConfig(cfg))
	rep.Checks = append(rep.Checks, checkPaths()...)
	rep.Checks = append(rep.Checks, checkWorkspace(cwd))
	rep.Checks = append(rep.Checks, checkRuntime()...)
	browserSvc, _ := browser.NewService()
	st := browserSvc.Doctor()
	rep.Checks = append(rep.Checks, Check{Name: "node", OK: st.NodeOK, Detail: st.NodeVersion})
	rep.Checks = append(rep.Checks, Check{Name: "python", OK: st.PythonOK, Detail: st.PythonVersion})
	rep.Checks = append(rep.Checks, Check{Name: "playwright_worker", OK: st.WorkerOK, Detail: st.WorkerPath})
	rep.Checks = append(rep.Checks, Check{Name: "chromium", OK: st.ChromiumOK, Detail: st.ChromiumPath, Warning: !st.ChromiumOK})
	rep.Status, rep.OK = classify(rep.Checks)
	return rep
}

func classify(checks []Check) (status string, ok bool) {
	hasFailed := false
	hasDegraded := false
	for _, c := range checks {
		if c.OK {
			continue
		}
		switch c.Name {
		case "config", "workspace", "global_config_path", "local_app_data":
			hasFailed = true
		case "runtime_root", "d_research_skill", "aleph", "chromium", "node", "python", "playwright_worker":
			hasDegraded = true
		default:
			hasDegraded = true
		}
	}
	switch {
	case hasFailed:
		return StatusFailed, false
	case hasDegraded:
		return StatusDegraded, false
	default:
		return StatusReady, true
	}
}

func checkConfig(cfg config.Config) Check {
	issues := config.Validate(cfg)
	if config.ContainsSecret(cfg) {
		issues = append(issues, "config must not contain secrets")
	}
	if len(issues) > 0 {
		return Check{Name: "config", OK: false, Detail: strings.Join(issues, "; ")}
	}
	return Check{Name: "config", OK: true, Detail: "valid"}
}

func checkPaths() []Check {
	var out []Check
	global, err := paths.GlobalConfigPath()
	if err != nil {
		out = append(out, Check{Name: "global_config_path", OK: false, Detail: err.Error()})
	} else if _, err = os.Stat(global); err != nil {
		out = append(out, Check{Name: "global_config", OK: true, Detail: global + " (will be created on first configure)"})
	} else {
		out = append(out, Check{Name: "global_config", OK: true, Detail: global})
	}
	local, err := paths.LocalAppDataDir()
	out = append(out, Check{Name: "local_app_data", OK: err == nil, Detail: local})
	return out
}

func checkWorkspace(cwd string) Check {
	writable := cwd
	if err := os.MkdirAll(paths.ProjectStateDir(cwd), 0o755); err != nil {
		return Check{Name: "workspace", OK: false, Detail: err.Error()}
	}
	test := paths.ProjectStateDir(cwd) + string(os.PathSeparator) + ".doctor"
	if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
		return Check{Name: "workspace", OK: false, Detail: err.Error()}
	}
	_ = os.Remove(test)
	return Check{Name: "workspace", OK: true, Detail: fmt.Sprintf("writable %s", writable)}
}

func checkRuntime() []Check {
	root, err := paths.RuntimeRoot()
	if err != nil {
		return []Check{{Name: "runtime_root", OK: false, Detail: err.Error()}}
	}
	out := []Check{{Name: "runtime_root", OK: true, Detail: root}}
	skill, err := paths.ResearchSkillRoot()
	if err != nil {
		out = append(out, Check{Name: "d_research_skill", OK: false, Detail: err.Error()})
	} else if info, err := os.Stat(skill); err != nil || !info.IsDir() {
		out = append(out, Check{Name: "d_research_skill", OK: false, Detail: skill + " missing"})
	} else {
		out = append(out, Check{Name: "d_research_skill", OK: true, Detail: skill})
	}
	aleph, err := paths.AlephRoot()
	if err != nil {
		out = append(out, Check{Name: "aleph", OK: false, Detail: err.Error()})
	} else if info, err := os.Stat(aleph); err != nil || !info.IsDir() {
		out = append(out, Check{Name: "aleph", OK: false, Detail: aleph + " missing"})
	} else {
		out = append(out, Check{Name: "aleph", OK: true, Detail: aleph})
	}
	return out
}