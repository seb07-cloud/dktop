package version

// Version information set at build time via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns the full version string
func String() string {
	return Version
}

// Full returns the full version with commit and date
func Full() string {
	return Version + " (" + Commit + ") " + BuildDate
}
