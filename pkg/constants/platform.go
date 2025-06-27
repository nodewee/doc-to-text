package constants

import "runtime"

// Platform-specific tool configurations
type PlatformConfig struct {
	CalibrePaths     []string
	GhostscriptPaths []string
	PandocPaths      []string
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
			},
			GhostscriptPaths: []string{
				"gs.exe", "gswin64c.exe", "gswin32c.exe",
			},
			PandocPaths: []string{
				"pandoc.exe",
				"C:\\Program Files\\Pandoc\\pandoc.exe",
			},
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
				"gs", "/usr/local/bin/gs", "/opt/homebrew/bin/gs",
			},
			PandocPaths: []string{
				"pandoc", "/usr/local/bin/pandoc", "/opt/homebrew/bin/pandoc",
			},
		}
	default: // Linux and other Unix-like systems
		return &PlatformConfig{
			CalibrePaths: []string{
				"ebook-convert",
				"/usr/bin/ebook-convert",
				"/usr/local/bin/ebook-convert",
				"/snap/bin/ebook-convert",
			},
			GhostscriptPaths: []string{
				"gs", "/usr/bin/gs", "/usr/local/bin/gs",
			},
			PandocPaths: []string{
				"pandoc", "/usr/bin/pandoc", "/usr/local/bin/pandoc",
			},
		}
	}
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// GetDefaultTempDir returns the platform-appropriate temporary directory
func GetDefaultTempDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\Temp"
	}
	return "/tmp"
}
