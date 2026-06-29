package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/d-init-d/d-research-cli/internal/host"
	"github.com/spf13/cobra"
)

func (r *Root) runCmd() *cobra.Command {
	var prompt, promptFile string
	var eventsFormat string
	var resume bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Chạy workflow nghiên cứu",
		RunE: func(cmd *cobra.Command, args []string) error {
			r.refreshService()
			text := prompt
			if promptFile != "" {
				data, err := os.ReadFile(promptFile)
				if err != nil {
					return err
				}
				text = string(data)
			}
			if text == "" {
				return fmt.Errorf("--prompt or --prompt-file is required for headless run")
			}
			ws, err := r.svc.Workspace()
			if err != nil {
				return err
			}
			bus := r.svc.EventBus()
			var out *os.File
			if eventsFormat == "jsonl" {
				out = os.Stdout
			}
			h := host.New(ws, bus, host.Options{Headless: true, EventsOut: out, Resume: resume, Mode: "research"})
			return h.Run(context.Background(), text)
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "research prompt")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "path to prompt file")
	cmd.Flags().StringVar(&eventsFormat, "events", "", "events output format: jsonl")
	cmd.Flags().BoolVar(&resume, "resume", true, "resume from checkpoint")
	return cmd
}