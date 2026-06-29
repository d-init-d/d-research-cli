package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/d-init-d/d-research-cli/internal/auth"
	"github.com/d-init-d/d-research-cli/internal/config"
	"github.com/spf13/cobra"
)

func (r *Root) searchCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "search", Short: "Cấu hình search provider"}
	cmd.AddCommand(r.searchProvidersCmd())
	cmd.AddCommand(r.searchConfigureCmd())
	cmd.AddCommand(r.searchTestCmd())
	return cmd
}

func (r *Root) searchProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "Danh sách search provider đã cấu hình",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			return r.svc.PrintJSON(cfg.Search.Providers)
		},
	}
}

func (r *Root) searchConfigureCmd() *cobra.Command {
	var id, typ, credentialRef, baseURL string
	var priority int
	var global bool
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Thêm/sửa search provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			if id == "" || typ == "" {
				return fmt.Errorf("--id and --type required")
			}
			p := config.SearchProvider{ID: id, Type: typ, Priority: priority, CredentialRef: credentialRef, BaseURL: baseURL}
			if credentialRef == "" && typ == "brave" {
				p.CredentialRef = auth.SearchCredentialRef(id)
			}
			replaced := false
			for i, existing := range cfg.Search.Providers {
				if existing.ID == id {
					cfg.Search.Providers[i] = p
					replaced = true
					break
				}
			}
			if !replaced {
				cfg.Search.Providers = append(cfg.Search.Providers, p)
			}
			if global {
				return r.svc.SaveGlobalConfig(cfg)
			}
			return r.svc.SaveProjectConfig(cfg)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "provider id")
	cmd.Flags().StringVar(&typ, "type", "", "duckduckgo|searxng|brave|google_cse")
	cmd.Flags().IntVar(&priority, "priority", 10, "priority (lower first)")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "credential ref")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for searxng/google_cse")
	cmd.Flags().BoolVar(&global, "global", false, "save global config")
	return cmd
}

func (r *Root) searchTestCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test search provider availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if all {
				reports, err := r.svc.TestAllSearch(ctx, cfg)
				if err != nil {
					return err
				}
				return r.svc.PrintJSON(reports)
			}
			mgr, err := r.svc.SearchManager(cfg)
			if err != nil {
				return err
			}
			start := time.Now()
			res, providerID, err := mgr.Search(ctx, "d-research connectivity probe 2026")
			report := map[string]any{
				"provider_id": providerID,
				"latency_ms":  time.Since(start).Milliseconds(),
				"available":   err == nil,
				"results":     len(res),
			}
			if err != nil {
				report["error"] = err.Error()
			}
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "test all configured providers")
	return cmd
}