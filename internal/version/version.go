package version

// Build metadata injected at compile time via -ldflags.
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return Version + "+" + Commit
}