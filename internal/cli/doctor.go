package cli

import (
	"github.com/spf13/cobra"
)

func (r *Root) doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Chẩn đoán hệ thống",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			rep, err := r.svc.Doctor()
			if err != nil {
				return err
			}
			if err := r.svc.PrintJSON(rep); err != nil {
				return err
			}
			if !rep.OK {
				return errDoctorFailed
			}
			return nil
		},
	}
}

var errDoctorFailed = doctorExitError{}

type doctorExitError struct{}

func (doctorExitError) Error() string { return "doctor found issues" }