package cli

import (
	"fmt"

	"github.com/d-init-d/d-research-cli/internal/auth"
	"github.com/spf13/cobra"
)

func (r *Root) modelCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "model", Short: "Cấu hình LLM model"}
	cmd.AddCommand(r.modelProvidersCmd())
	cmd.AddCommand(r.modelConfigureCmd())
	cmd.AddCommand(r.modelTestCmd())
	return cmd
}

func (r *Root) modelProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "Danh sách provider được hỗ trợ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.svc.PrintJSON([]string{"openai", "anthropic", "gemini", "openrouter", "ollama"})
		},
	}
}

func (r *Root) modelConfigureCmd() *cobra.Command {
	var provider, model, credentialRef, baseURL string
	var global bool
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Cấu hình model mặc định",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			if provider != "" {
				cfg.Models.Default.Provider = provider
			}
			if model != "" {
				cfg.Models.Default.Model = model
			}
			if credentialRef != "" {
				cfg.Models.Default.CredentialRef = credentialRef
			} else if provider != "" {
				cfg.Models.Default.CredentialRef = auth.CredentialRefFor(provider)
			}
			if baseURL != "" {
				cfg.Models.Default.BaseURL = baseURL
			}
			if global {
				return r.svc.SaveGlobalConfig(cfg)
			}
			return r.svc.SaveProjectConfig(cfg)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider id")
	cmd.Flags().StringVar(&model, "model", "", "model name")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "credential ref")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "optional base URL")
	cmd.Flags().BoolVar(&global, "global", false, "save to global config")
	return cmd
}

func (r *Root) modelTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Kiểm tra credential model mặc định",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			ref := cfg.Models.Default.CredentialRef
			if ref == "" {
				ref = auth.CredentialRefFor(cfg.Models.Default.Provider)
			}
			if err := r.svc.Auth.Test(ref); err != nil {
				return err
			}
			fmt.Println("credential ok for", cfg.Models.Default.Provider, cfg.Models.Default.Model)
			return nil
		},
	}
}