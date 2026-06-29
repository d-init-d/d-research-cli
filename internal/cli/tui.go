package cli

import (
	"github.com/d-init-d/d-research-cli/internal/tui"
)

func (r *Root) runTUI() error {
	r.refreshService()
	return tui.Run(r.svc)
}