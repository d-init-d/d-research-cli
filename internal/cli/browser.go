package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func (r *Root) browserCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "browser", Short: "Playwright browser runtime"}
	cmd.AddCommand(r.browserStatusCmd())
	cmd.AddCommand(r.browserDoctorCmd())
	cmd.AddCommand(r.browserInstallCmd())
	cmd.AddCommand(r.browserRepairCmd())
	return cmd
}

func (r *Root) browserStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Trạng thái browser runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			svc, err := r.svc.BrowserService()
			if err != nil {
				return err
			}
			return r.svc.PrintJSON(svc.Doctor())
		},
	}
}

func (r *Root) browserDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Chẩn đoán Node/Python/Playwright/Chromium",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.browserStatusCmd().RunE(cmd, args)
		},
	}
}

func (r *Root) browserInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Cài Chromium đã pin vào LOCALAPPDATA",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			svc, err := r.svc.BrowserService()
			if err != nil {
				return err
			}
			return svc.Install(context.Background())
		},
	}
}

func (r *Root) browserRepairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Sửa cài đặt Chromium",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.browserInstallCmd().RunE(cmd, args)
		},
	}
}