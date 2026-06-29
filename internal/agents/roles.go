package agents

// Fixed role capabilities per v0.1 plan.
const (
	Coordinator     = "coordinator"
	Planner         = "planner"
	Searcher        = "searcher"
	Browser         = "browser"
	Extractor       = "extractor"
	Verifier        = "verifier"
	Synthesizer     = "synthesizer"
	CausalBuilder   = "causal_builder"
	Challenger      = "challenger"
)

var ResearchRoles = []string{
	Coordinator, Planner, Searcher, Browser, Extractor, Verifier, Synthesizer,
}

var SimulationRoles = []string{
	CausalBuilder, Challenger,
}

func AllRoles(simulation bool) []string {
	if simulation {
		out := append([]string{}, ResearchRoles...)
		return append(out, SimulationRoles...)
	}
	return ResearchRoles
}