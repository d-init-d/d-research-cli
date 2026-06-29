package cli

import (
	"fmt"
	"path/filepath"

	"github.com/d-init-d/d-research-cli/internal/kb"
	"github.com/spf13/cobra"
)

func (r *Root) kbCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "kb", Short: "Quản lý Aleph KB"}
	cmd.AddCommand(r.kbStatusCmd())
	cmd.AddCommand(r.kbValidateCmd())
	cmd.AddCommand(r.kbRepairCmd())
	cmd.AddCommand(r.kbExportCmd())
	return cmd
}

func (r *Root) kbStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Trạng thái ./kb",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			return r.svc.PrintJSON(r.svc.KBStatus())
		},
	}
}

func (r *Root) kbValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate KB schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			st := r.svc.KBStatus()
			if !st.Valid {
				return fmt.Errorf("invalid kb: %s", st.Error)
			}
			fmt.Println("valid")
			return nil
		},
	}
}

func (r *Root) kbRepairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Tạo skeleton KB nếu thiếu (không ghi đè KB lỗi)",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			st := r.svc.KBStatus()
			if st.Exists && !st.Empty && !st.Valid {
				return fmt.Errorf("kb invalid; manual repair required: %s", st.Error)
			}
			if st.Valid {
				fmt.Println("kb already valid")
				return nil
			}
			if err := kb.CreateSkeleton(r.svc.CWD); err != nil {
				return err
			}
			fmt.Println("kb skeleton created")
			return nil
		},
	}
}

func (r *Root) kbExportCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Xuất KB ZIP (không chứa secret/cache)",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			if out == "" {
				out = filepath.Join("research-output", "kb-export.zip")
			}
			if err := r.svc.KBExport(out); err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output zip path")
	return cmd
}