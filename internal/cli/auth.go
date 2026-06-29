package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/d-init-d/d-research-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func (r *Root) authCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Quản lý credential"}
	cmd.AddCommand(r.authLoginCmd())
	cmd.AddCommand(r.authListCmd())
	cmd.AddCommand(r.authLogoutCmd())
	cmd.AddCommand(r.authTestCmd())
	return cmd
}

func (r *Root) authLoginCmd() *cobra.Command {
	var provider, ref string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Lưu credential vào Windows Credential Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			if ref == "" && provider != "" {
				ref = auth.CredentialRefFor(provider)
			}
			if ref == "" {
				return fmt.Errorf("--provider or --ref required")
			}
			fmt.Fprintf(os.Stderr, "Secret cho %s: ", ref)
			secret, err := readSecret()
			if err != nil {
				return err
			}
			return r.svc.Auth.Set(ref, secret)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "llm provider id")
	cmd.Flags().StringVar(&ref, "ref", "", "credential ref e.g. llm/openrouter")
	return cmd
}

func (r *Root) authListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Liệt kê credential (không hiển thị secret)",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			cfg, _, err := r.svc.LoadConfig()
			if err != nil {
				return err
			}
			entries, err := r.svc.Auth.List(r.svc.KnownCredentialRefs(cfg))
			if err != nil {
				return err
			}
			return r.svc.PrintJSON(entries)
		},
	}
}

func (r *Root) authLogoutCmd() *cobra.Command {
	var ref string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Xóa credential",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			if ref == "" {
				return fmt.Errorf("--ref required")
			}
			return r.svc.Auth.Delete(ref)
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "", "credential ref")
	return cmd
}

func (r *Root) authTestCmd() *cobra.Command {
	var ref string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Kiểm tra credential ref",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			if ref == "" {
				return fmt.Errorf("--ref required")
			}
			if err := r.svc.Auth.Test(ref); err != nil {
				return err
			}
			fmt.Println("ok")
			return nil
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "", "credential ref")
	return cmd
}

func readSecret() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		fmt.Fprintln(os.Stderr)
		return strings.TrimSpace(string(b)), nil
	}
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text()), sc.Err()
	}
	return "", fmt.Errorf("no secret provided")
}