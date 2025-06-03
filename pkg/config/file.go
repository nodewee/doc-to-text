package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/utils"
)

const (
	ConfigFileName = "config.json"
	AppDirName     = ".doc-to-text"
)

// ConfigFile represents the JSON configuration file structure
type ConfigFile struct {
	// External tool paths only
	LLMCallerPath   string `json:"llm_caller_path"`
	SuryaOCRPath    string `json:"surya_ocr_path"`
	CalibrePath     string `json:"calibre_path"`
	PandocPath      string `json:"pandoc_path"`
	GhostscriptPath string `json:"ghostscript_path"`
}

// GetConfigDir returns the user configuration directory (~/.doc-to-text)
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to get user home directory")
	}

	appConfigDir := filepath.Join(homeDir, AppDirName)
	return appConfigDir, nil
}

// GetConfigFilePath returns the full path to the configuration file
func GetConfigFilePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, ConfigFileName), nil
}

// LoadConfig loads configuration from file or creates default if not exists
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigFilePath()
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to get config file path")
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		return createDefaultConfigFile(configPath)
	}

	// Load existing config file
	return loadConfigFromFile(configPath)
}

// createDefaultConfigFile creates a default configuration file with auto-detected tools
func createDefaultConfigFile(configPath string) (*Config, error) {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, constants.DefaultDirPermission); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to create config directory")
	}

	// Create default config with empty tool paths (will be auto-detected)
	configFile := &ConfigFile{
		LLMCallerPath:   "", // Will be auto-detected
		SuryaOCRPath:    "", // Will be auto-detected
		CalibrePath:     "", // Will be auto-detected
		PandocPath:      "", // Will be auto-detected
		GhostscriptPath: "", // Will be auto-detected
	}

	// Auto-detect tool paths
	detectAndUpdateToolPaths(configFile)

	// Save to file
	if err := saveConfigFile(configPath, configFile); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to save default config file")
	}

	fmt.Printf("✅ Created default configuration file: %s\n", configPath)
	if hasDetectedTools(configFile) {
		fmt.Printf("🔍 Auto-detected available tools\n")
	}

	// Convert to Config struct
	return configFileToConfig(configFile), nil
}

// loadConfigFromFile loads configuration from an existing file
func loadConfigFromFile(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to read config file")
	}

	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeConversion, "failed to parse config file")
	}

	return configFileToConfig(&configFile), nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigFilePath()
	if err != nil {
		return err
	}

	configFile := configToConfigFile(config)
	return saveConfigFile(configPath, configFile)
}

// saveConfigFile saves ConfigFile to disk
func saveConfigFile(configPath string, configFile *ConfigFile) error {
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeConversion, "failed to marshal config")
	}

	if err := os.WriteFile(configPath, data, constants.DefaultFilePermission); err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, "failed to write config file")
	}

	return nil
}

// detectAndUpdateToolPaths auto-detects tool paths and updates the config
func detectAndUpdateToolPaths(configFile *ConfigFile) {
	detectedPaths := make(map[string]string)
	platformConfig := constants.GetPlatformConfig()

	// Define tools to detect with their platform-specific possible names and paths
	toolsToDetect := map[string][]string{
		"llm-caller":  {utils.DefaultPathUtils.GetExecutableName("llm-caller")},
		"surya_ocr":   {utils.DefaultPathUtils.GetExecutableName("surya_ocr")},
		"pandoc":      append([]string{utils.DefaultPathUtils.GetExecutableName("pandoc")}, platformConfig.PandocPaths...),
		"ghostscript": append(getGhostscriptPossibleNames(), platformConfig.GhostscriptPaths...),
		"calibre":     append([]string{utils.DefaultPathUtils.GetExecutableName("ebook-convert")}, platformConfig.CalibrePaths...),
	}

	for toolKey, possiblePaths := range toolsToDetect {
		for _, pathOrName := range possiblePaths {
			var detectedPath string
			var err error

			// First try as a direct path
			if filepath.IsAbs(pathOrName) {
				// Handle wildcard paths (e.g., for Ghostscript on Windows)
				if strings.Contains(pathOrName, "*") {
					if expandedPath := expandWildcardPath(pathOrName); expandedPath != "" {
						detectedPath = expandedPath
					}
				} else if utils.DefaultPathUtils.IsExecutable(pathOrName) {
					detectedPath = pathOrName
				}
			} else {
				// Try to find in PATH
				detectedPath, err = exec.LookPath(pathOrName)
			}

			if err == nil && detectedPath != "" && utils.DefaultPathUtils.IsExecutable(detectedPath) {
				// Normalize the path for consistency
				normalizedPath := utils.NormalizePath(detectedPath)
				detectedPaths[toolKey] = normalizedPath
				break
			}
		}
	}

	// Update config with detected paths
	if path, ok := detectedPaths["llm-caller"]; ok {
		configFile.LLMCallerPath = path
	}
	if path, ok := detectedPaths["surya_ocr"]; ok {
		configFile.SuryaOCRPath = path
	}
	if path, ok := detectedPaths["pandoc"]; ok {
		configFile.PandocPath = path
	}
	if path, ok := detectedPaths["ghostscript"]; ok {
		configFile.GhostscriptPath = path
	}
	if path, ok := detectedPaths["calibre"]; ok {
		configFile.CalibrePath = path
	}
}

// getGhostscriptPossibleNames returns possible names for Ghostscript based on platform
func getGhostscriptPossibleNames() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"gs.exe", "gswin64c.exe", "gswin32c.exe"}
	default:
		return []string{"gs", "ghostscript"}
	}
}

// expandWildcardPath expands paths with wildcards (mainly for Windows)
func expandWildcardPath(pattern string) string {
	// Simple wildcard expansion for Ghostscript paths
	if strings.Contains(pattern, "*") {
		// Extract the directory part before the wildcard
		dir := filepath.Dir(pattern)
		if baseDir := strings.Split(dir, "*")[0]; baseDir != "" {
			matches, err := filepath.Glob(pattern)
			if err == nil && len(matches) > 0 {
				// Return the first match that is executable
				for _, match := range matches {
					if utils.DefaultPathUtils.IsExecutable(match) {
						return match
					}
				}
			}
		}
	}
	return ""
}

// getCalibePossiblePaths returns possible Calibre paths based on OS
func getCalibePossiblePaths() []string {
	platformConfig := constants.GetPlatformConfig()
	return platformConfig.CalibrePaths
}

// hasDetectedTools checks if any tools were auto-detected
func hasDetectedTools(configFile *ConfigFile) bool {
	return configFile.LLMCallerPath != "" ||
		configFile.SuryaOCRPath != "" ||
		configFile.PandocPath != "" ||
		configFile.GhostscriptPath != "" ||
		configFile.CalibrePath != ""
}

// configFileToConfig converts ConfigFile to Config
func configFileToConfig(cf *ConfigFile) *Config {
	return &Config{
		LLMCallerPath:   cf.LLMCallerPath,
		SuryaOCRPath:    cf.SuryaOCRPath,
		CalibrePath:     cf.CalibrePath,
		PandocPath:      cf.PandocPath,
		GhostscriptPath: cf.GhostscriptPath,
		// Runtime settings with defaults
		OCRStrategy:      DefaultOCRStrategy,
		ContentType:      DefaultContentType,
		SkipExisting:     DefaultSkipExisting,
		MaxConcurrency:   DefaultMaxConcurrency,
		MinTextThreshold: DefaultMinTextThreshold,
		TimeoutMinutes:   DefaultTimeoutMinutes,
		LogLevel:         DefaultLogLevel,
		EnableVerbose:    DefaultEnableVerbose,
	}
}

// configToConfigFile converts Config to ConfigFile
func configToConfigFile(c *Config) *ConfigFile {
	return &ConfigFile{
		LLMCallerPath:   c.LLMCallerPath,
		SuryaOCRPath:    c.SuryaOCRPath,
		CalibrePath:     c.CalibrePath,
		PandocPath:      c.PandocPath,
		GhostscriptPath: c.GhostscriptPath,
	}
}

// GetConfigValue gets a specific configuration value by key
func GetConfigValue(key string) (interface{}, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	switch key {
	case "llm_caller_path":
		return config.LLMCallerPath, nil
	case "surya_ocr_path":
		return config.SuryaOCRPath, nil
	case "calibre_path":
		return config.CalibrePath, nil
	case "pandoc_path":
		return config.PandocPath, nil
	case "ghostscript_path":
		return config.GhostscriptPath, nil
	default:
		return nil, utils.NewValidationError(fmt.Sprintf("unknown config key: %s", key), nil)
	}
}

// SetConfigValue sets a specific configuration value by key
func SetConfigValue(key string, value interface{}) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	switch key {
	case "llm_caller_path":
		if v, ok := value.(string); ok {
			config.LLMCallerPath = v
		} else {
			return utils.NewValidationError("llm_caller_path must be a string", nil)
		}
	case "surya_ocr_path":
		if v, ok := value.(string); ok {
			config.SuryaOCRPath = v
		} else {
			return utils.NewValidationError("surya_ocr_path must be a string", nil)
		}
	case "calibre_path":
		if v, ok := value.(string); ok {
			config.CalibrePath = v
		} else {
			return utils.NewValidationError("calibre_path must be a string", nil)
		}
	case "pandoc_path":
		if v, ok := value.(string); ok {
			config.PandocPath = v
		} else {
			return utils.NewValidationError("pandoc_path must be a string", nil)
		}
	case "ghostscript_path":
		if v, ok := value.(string); ok {
			config.GhostscriptPath = v
		} else {
			return utils.NewValidationError("ghostscript_path must be a string", nil)
		}
	default:
		return utils.NewValidationError(fmt.Sprintf("unknown config key: %s", key), nil)
	}

	// Save the updated config
	return SaveConfig(config)
}

// ListConfigKeys returns all available configuration keys
func ListConfigKeys() []string {
	return []string{
		"llm_caller_path",
		"surya_ocr_path",
		"calibre_path",
		"pandoc_path",
		"ghostscript_path",
	}
}
