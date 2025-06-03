package constants

import (
	"runtime"
)

// Platform-specific constants
var (
	// Current operating system
	CurrentOS = runtime.GOOS

	// Platform-specific executable extensions
	ExecutableExt = getExecutableExtension()

	// Platform-specific path separators (though filepath.Join should be used)
	PathSeparator = getPathSeparator()

	// Platform-specific line endings
	LineEnding = getLineEnding()
)

// Platform-specific tool configurations
type PlatformConfig struct {
	CalibrePaths     []string
	GhostscriptPaths []string
	PandocPaths      []string
	DefaultShell     string
	TempDirPrefix    string
}

// GetPlatformConfig returns platform-specific configuration
func GetPlatformConfig() *PlatformConfig {
	switch runtime.GOOS {
	case "windows":
		return &PlatformConfig{
			CalibrePaths: []string{
				"ebook-convert.exe",
				"C:\\Program Files\\Calibre2\\ebook-convert.exe",
				"C:\\Program Files (x86)\\Calibre2\\ebook-convert.exe",
				"C:\\ProgramData\\chocolatey\\bin\\ebook-convert.exe",
			},
			GhostscriptPaths: []string{
				"gs.exe",
				"gswin64c.exe",
				"gswin32c.exe",
				"C:\\Program Files\\gs\\gs*\\bin\\gswin64c.exe",
				"C:\\Program Files (x86)\\gs\\gs*\\bin\\gswin32c.exe",
			},
			PandocPaths: []string{
				"pandoc.exe",
				"C:\\Program Files\\Pandoc\\pandoc.exe",
				"C:\\Program Files (x86)\\Pandoc\\pandoc.exe",
			},
			DefaultShell:  "cmd.exe",
			TempDirPrefix: "doc-to-text-",
		}
	case "darwin":
		return &PlatformConfig{
			CalibrePaths: []string{
				"/Applications/calibre.app/Contents/MacOS/ebook-convert",
				"ebook-convert",
				"/usr/local/bin/ebook-convert",
				"/opt/homebrew/bin/ebook-convert",
			},
			GhostscriptPaths: []string{
				"gs",
				"/usr/local/bin/gs",
				"/opt/homebrew/bin/gs",
				"/usr/bin/gs",
			},
			PandocPaths: []string{
				"pandoc",
				"/usr/local/bin/pandoc",
				"/opt/homebrew/bin/pandoc",
				"/usr/bin/pandoc",
			},
			DefaultShell:  "/bin/sh",
			TempDirPrefix: "doc-to-text-",
		}
	default: // Linux and other Unix-like systems
		return &PlatformConfig{
			CalibrePaths: []string{
				"ebook-convert",
				"/usr/bin/ebook-convert",
				"/usr/local/bin/ebook-convert",
				"/snap/bin/ebook-convert",
				"/opt/calibre/ebook-convert",
			},
			GhostscriptPaths: []string{
				"gs",
				"/usr/bin/gs",
				"/usr/local/bin/gs",
				"/bin/gs",
			},
			PandocPaths: []string{
				"pandoc",
				"/usr/bin/pandoc",
				"/usr/local/bin/pandoc",
				"/bin/pandoc",
			},
			DefaultShell:  "/bin/sh",
			TempDirPrefix: "doc-to-text-",
		}
	}
}

// getExecutableExtension returns the executable file extension for the current platform
func getExecutableExtension() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// getPathSeparator returns the path separator for the current platform
func getPathSeparator() string {
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

// getLineEnding returns the line ending for the current platform
func getLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsUnixLike returns true if running on a Unix-like system (macOS, Linux, etc.)
func IsUnixLike() bool {
	return runtime.GOOS != "windows"
}

// GetDefaultTempDir returns the platform-appropriate temporary directory
func GetDefaultTempDir() string {
	switch runtime.GOOS {
	case "windows":
		return "C:\\Windows\\Temp"
	default:
		return "/tmp"
	}
}

// NormalizePath normalizes a file path for the current platform
func NormalizePath(path string) string {
	// filepath.Clean handles most normalization, but we can add platform-specific logic if needed
	return path
}
