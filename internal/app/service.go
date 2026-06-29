package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/d-init-d/d-research-cli/internal/auth"
	"github.com/d-init-d/d-research-cli/internal/browser"
	"github.com/d-init-d/d-research-cli/internal/config"
	"github.com/d-init-d/d-research-cli/internal/doctor"
	"github.com/d-init-d/d-research-cli/internal/event"
	"github.com/d-init-d/d-research-cli/internal/kb"
	"github.com/d-init-d/d-research-cli/internal/log"
	"github.com/d-init-d/d-research-cli/internal/paths"
	"github.com/d-init-d/d-research-cli/internal/search"
	"github.com/d-init-d/d-research-cli/internal/store"
)

type Service struct {
	CWD        string
	ConfigPath string
	Auth       *auth.Store
	Redactor   *log.Redactor
}

func NewService(cwd string) *Service {
	return &Service{
		CWD:      cwd,
		Auth:     auth.NewStore(),
		Redactor: log.NewRedactor(),
	}
}

func (s *Service) LoadConfig() (config.Config, config.Sources, error) {
	sources := config.ResolveSources(s.CWD, s.ConfigPath)
	cfg, err := config.Load(sources)
	return cfg, sources, err
}

func (s *Service) SaveGlobalConfig(cfg config.Config) error {
	path, err := paths.GlobalConfigPath()
	if err != nil {
		return err
	}
	return config.Save(path, cfg)
}

func (s *Service) SaveProjectConfig(cfg config.Config) error {
	return config.Save(paths.ProjectConfigPath(s.CWD), cfg)
}

func (s *Service) KnownCredentialRefs(cfg config.Config) []string {
	refs := []string{}
	if cfg.Models.Default.CredentialRef != "" {
		refs = append(refs, cfg.Models.Default.CredentialRef)
	} else if cfg.Models.Default.Provider != "" {
		refs = append(refs, auth.CredentialRefFor(cfg.Models.Default.Provider))
	}
	for _, p := range cfg.Search.Providers {
		if p.CredentialRef != "" {
			refs = append(refs, p.CredentialRef)
		}
	}
	return refs
}

func (s *Service) SearchManager(cfg config.Config) (*search.Manager, error) {
	http := search.NewHTTPClient(15 * time.Second)
	return search.NewManager(cfg.Search, func(p config.SearchProvider) (search.Provider, error) {
		key := ""
		if p.CredentialRef != "" {
			var err error
			key, err = s.Auth.Get(p.CredentialRef)
			if err != nil {
				return nil, err
			}
		}
		return search.BuildProvider(p, http, key)
	})
}

func (s *Service) TestAllSearch(ctx context.Context, cfg config.Config) ([]search.TestReport, error) {
	http := search.NewHTTPClient(10 * time.Second)
	var reports []search.TestReport
	for _, p := range cfg.Search.Providers {
		prov, err := search.BuildProvider(p, http, "")
		if err != nil {
			reports = append(reports, search.TestReport{ProviderID: p.ID, Available: false, Error: err.Error()})
			continue
		}
		reports = append(reports, search.TestProvider(ctx, prov))
	}
	return reports, nil
}

func (s *Service) Doctor() (doctor.Report, error) {
	cfg, _, err := s.LoadConfig()
	if err != nil {
		return doctor.Report{}, err
	}
	return doctor.Run(cfg, s.CWD), nil
}

func (s *Service) Workspace() (*store.Workspace, error) {
	return store.Open(s.CWD)
}

func (s *Service) EventBus() *event.Bus {
	return event.NewBus(s.Redactor)
}

func (s *Service) BrowserService() (*browser.Service, error) {
	return browser.NewService()
}

func (s *Service) KBStatus() kb.Status {
	return kb.StatusOf(s.CWD)
}

func (s *Service) KBExport(path string) error {
	return kb.Export(s.CWD, path)
}

func (s *Service) PrintJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(s.Redactor.RedactJSON(data)))
	return nil
}

func (s *Service) EnsureGlobalConfig() error {
	path, err := paths.GlobalConfigPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return config.Save(path, config.Default())
}