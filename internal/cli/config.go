package cli

import (
	"fmt"

	"github.com/d-init-d/d-research-cli/internal/config"
	"github.com/d-init-d/d-research-cli/internal/paths"
	"github.com/spf13/cobra"
)

func (r *Root) configCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Quản lý config"}
	cmd.AddCommand(r.configPathCmd())
	cmd.AddCommand(r.configShowCmd())
	cmd.AddCommand(r.configValidateCmd())
	return cmd
}

func (r *Root) configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Hiển thị đường dẫn config",
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := paths.GlobalConfigPath()
			if err != nil {
				return err
			}
			fmt.Println("global:", global)
			fmt.Println("project:", paths.ProjectConfigPath(r.svc.CWD))
			return nil
		},
	}
}

func (r *Root) configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Hiển thị config đã merge",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, sources, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			if config.ContainsSecret(cfg) {
				return fmt.Errorf("resolved config contains secret-like fields")
			}
			_ = sources
			return r.svc.PrintJSON(cfg)
		},
	}
}

func (r *Root) configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate config",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			issues := config.Validate(cfg)
			if len(issues) > 0 {
				return fmt.Errorf("%v", issues)
			}
			fmt.Println("valid")
			return nil
		},
	}
}