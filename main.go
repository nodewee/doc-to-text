package main

import (
	"doc-to-text/cmd"
	"os"
)

// Version information - these will be set during build time via ldflags
var (
	Version   = "dev"     // Application version (e.g., "v1.2.3")
	GitCommit = "none"    // Git commit hash
	BuildTime = "unknown" // Build timestamp
	BuildBy   = "unknown" // Builder information
)

func main() {
	// Pass version information to the command system
	cmd.SetVersionInfo(Version, GitCommit, BuildTime, BuildBy)

	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
