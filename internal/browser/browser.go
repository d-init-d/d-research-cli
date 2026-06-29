package browser

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/d-init-d/d-research-cli/internal/paths"
)

type Status struct {
	NodeOK        bool   `json:"node_ok"`
	NodeVersion   string `json:"node_version,omitempty"`
	PythonOK      bool   `json:"python_ok"`
	PythonVersion string `json:"python_version,omitempty"`
	ChromiumOK    bool   `json:"chromium_ok"`
	ChromiumPath  string `json:"chromium_path,omitempty"`
	WorkerOK      bool   `json:"worker_ok"`
	WorkerPath    string `json:"worker_path,omitempty"`
}

type WorkerRequest struct {
	ID      string         `json:"id"`
	Command string         `json:"command"`
	URL     string         `json:"url,omitempty"`
	Timeout int            `json:"timeout_ms,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type WorkerResponse struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Title   string `json:"title,omitempty"`
	Text    string `json:"text,omitempty"`
	Error   string `json:"error,omitempty"`
	Blocker string `json:"blocker,omitempty"`
}

type Service struct {
	workerPath string
	browserDir string
}

func NewService() (*Service, error) {
	browserDir, err := paths.BrowserDir()
	if err != nil {
		return nil, err
	}
	workerPath, err := paths.ScriptPath("playwright_worker.mjs")
	if err != nil {
		return nil, err
	}
	return &Service{
		workerPath: workerPath,
		browserDir: browserDir,
	}, nil
}

func (s *Service) WorkerPath() string {
	return s.workerPath
}

func (s *Service) Doctor() Status {
	st := Status{WorkerPath: s.WorkerPath()}
	if v, err := runVersion("node", "-v"); err == nil {
		st.NodeOK = true
		st.NodeVersion = strings.TrimSpace(v)
	}
	if v, err := runVersion("python", "--version"); err == nil {
		st.PythonOK = true
		st.PythonVersion = strings.TrimSpace(v)
	}
	if _, err := os.Stat(s.WorkerPath()); err == nil {
		st.WorkerOK = true
	}
	chromium := filepath.Join(s.browserDir, "chromium")
	if info, err := os.Stat(chromium); err == nil && info.IsDir() {
		st.ChromiumOK = true
		st.ChromiumPath = chromium
	}
	return st
}

func runVersion(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *Service) Install(ctx context.Context) error {
	script, err := paths.ScriptPath("install_chromium.mjs")
	if err != nil {
		return err
	}
	if _, err := os.Stat(script); os.IsNotExist(err) {
		return errors.New("install script missing; runtime not provisioned")
	}
	cmd := exec.CommandContext(ctx, "node", script, s.browserDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *Service) Navigate(ctx context.Context, url string, timeoutMS int) (WorkerResponse, error) {
	req := WorkerRequest{
		ID:      fmt.Sprintf("nav-%d", time.Now().UnixNano()),
		Command: "navigate",
		URL:     url,
		Timeout: timeoutMS,
		Options: map[string]any{
			"headless":     true,
			"browser_path": s.browserDir,
		},
	}
	return s.invoke(ctx, req)
}

func (s *Service) invoke(ctx context.Context, req WorkerRequest) (WorkerResponse, error) {
	cmd := exec.CommandContext(ctx, "node", s.WorkerPath())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return WorkerResponse{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return WorkerResponse{}, err
	}
	if err := cmd.Start(); err != nil {
		return WorkerResponse{}, err
	}
	enc := json.NewEncoder(stdin)
	if err := enc.Encode(req); err != nil {
		return WorkerResponse{}, err
	}
	_ = stdin.Close()
	scanner := bufio.NewScanner(stdout)
	var resp WorkerResponse
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return WorkerResponse{}, err
		}
	}
	_ = cmd.Wait()
	return resp, nil
}