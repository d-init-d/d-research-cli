package host

import (
	"context"
	"fmt"

	"github.com/d-init-d/d-research-cli/internal/config"
	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

// LLMFactory builds an agentcore ChatModel from resolved config + secret.
type LLMFactory struct {
	APIKey string
}

func (f LLMFactory) Model(cfg config.ModelRef) (agentcore.ChatModel, error) {
	if f.APIKey == "" {
		return nil, fmt.Errorf("missing API key for provider %s", cfg.Provider)
	}
	opts := []llm.ModelOption{llm.WithAPIKey(f.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, llm.WithBaseURL(cfg.BaseURL))
	}
	return llm.NewModel(cfg.Provider, cfg.Model, opts...)
}

// RunPlanner is the integration point for agentcore-driven planning.
// v0.1 keeps a deterministic fallback plan when LLM is unavailable.
func RunPlanner(ctx context.Context, model agentcore.ChatModel, prompt string) (string, error) {
	if model == nil {
		return "", fmt.Errorf("planner model not configured")
	}
	agent := agentcore.NewAgent(
		agentcore.WithModel(model),
		agentcore.WithSystemPrompt("You are the D Research Planner. Produce a concise research plan outline."),
	)
	if err := agent.Prompt(ctx, "Research question: "+prompt); err != nil {
		return "", err
	}
	agent.WaitForIdle()
	return "planner complete", nil
}