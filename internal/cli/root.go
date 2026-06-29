package cli

import (
	"fmt"
	"os"

	"github.com/d-init-d/d-research-cli/internal/app"
	"github.com/d-init-d/d-research-cli/internal/version"
	"github.com/spf13/cobra"
)

type Root struct {
	svc        *app.Service
	headless   bool
	configPath string
}

func New() *cobra.Command {
	cwd, _ := os.Getwd()
	r := &Root{svc: app.NewService(cwd)}
	root := &cobra.Command{
		Use:   "d-research",
		Short: "D Research CLI — nghiên cứu sâu Windows-first",
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.headless {
				return fmt.Errorf("headless mode requires `run` subcommand")
			}
			return r.runTUI()
		},
	}
	root.PersistentFlags().StringVar(&r.configPath, "config", "", "override config path")
	root.PersistentFlags().BoolVar(&r.headless, "headless", false, "headless mode")
	root.AddCommand(r.runCmd())
	root.AddCommand(r.authCmd())
	root.AddCommand(r.modelCmd())
	root.AddCommand(r.searchCmd())
	root.AddCommand(r.browserCmd())
	root.AddCommand(r.kbCmd())
	root.AddCommand(r.configCmd())
	root.AddCommand(r.doctorCmd())
	root.Version = version.String()
	return root
}

func (r *Root) refreshService() {
	cwd, _ := os.Getwd()
	r.svc = app.NewService(cwd)
	r.svc.ConfigPath = r.configPath
}

func (r *Root) Execute() error {
	return New().Execute()
}