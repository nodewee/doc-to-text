package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version information variables - set by main.go
var (
	version   = "dev"
	gitCommit = "none"
	buildTime = "unknown"
	buildBy   = "unknown"
)

// SetVersionInfo sets the version information from main.go
func SetVersionInfo(v, commit, buildTimeParam, buildByParam string) {
	version = v
	gitCommit = commit
	buildTime = buildTimeParam
	buildBy = buildByParam
}

// GetVersionInfo returns the current version information
func GetVersionInfo() (string, string, string, string) {
	return version, gitCommit, buildTime, buildBy
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: "A CLI tool for extracting text from various document formats with configurable OCR capabilities.\n\n" +
		"Version Information:\n" +
		"- Version\n" +
		"- Git Commit\n" +
		"- Build Time\n" +
		"- Built By\n" +
		"- Go Version\n" +
		"- OS/Architecture",
	Run: func(cmd *cobra.Command, args []string) {
		showVersionInfo()
	},
}

// showVersionInfo displays comprehensive version information
func showVersionInfo() {
	fmt.Printf("📄 Doc Text Extractor\n")
	fmt.Printf("=======================\n\n")

	// Application information
	fmt.Printf("🔖 Version Information:\n")
	fmt.Printf("  Version:     %s\n", version)
	fmt.Printf("  Git Commit:  %s\n", gitCommit)
	fmt.Printf("  Build Time:  %s\n", buildTime)
	fmt.Printf("  Built By:    %s\n", buildBy)
	fmt.Printf("\n")

	// Runtime information
	fmt.Printf("⚙️ Runtime Information:\n")
	fmt.Printf("  Go Version:  %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Compiler:    %s\n", runtime.Compiler)
	fmt.Printf("\n")

	// Additional build info
	if version != "dev" && !strings.Contains(version, "dev") && !strings.Contains(version, "+") {
		fmt.Printf("🚀 Release Information:\n")
		fmt.Printf("  This is a release build\n")
		fmt.Printf("  Release notes: https://github.com/yourorg/doc-to-text/releases/tag/%s\n", version)
	} else {
		fmt.Printf("🔧 Development Information:\n")
		fmt.Printf("  This is a development build\n")
		fmt.Printf("  Not for production use\n")
	}
}

func init() {
	// Add version command to root
	rootCmd.AddCommand(versionCmd)
}
