package config

import (
	"os"
	"strconv"

	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// Default values and constants
const (
	DefaultLogLevel         = "info"
	DefaultTimeoutMinutes   = 30
	DefaultMaxConcurrency   = 4
	DefaultMinTextThreshold = 10
	DefaultSkipExisting     = true
	DefaultEnableVerbose    = false
	DefaultOCRStrategy      = types.OCRStrategyInteractive
	DefaultContentType      = types.ContentTypeImage // Default to image content type
)

// Config holds application runtime configuration (no file persistence)
type Config struct {
	// Runtime settings only
	OCRStrategy      types.OCRStrategy `json:"-"`
	LLMTemplate      string            `json:"-"`
	ContentType      types.ContentType `json:"-"` // Document content type (text or image)
	SkipExisting     bool              `json:"-"`
	MaxConcurrency   int               `json:"-"`
	MinTextThreshold int               `json:"-"`
	TimeoutMinutes   int               `json:"-"`
	LogLevel         string            `json:"-"`
	EnableVerbose    bool              `json:"-"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		OCRStrategy:      DefaultOCRStrategy,
		LLMTemplate:      "",
		ContentType:      DefaultContentType,
		SkipExisting:     DefaultSkipExisting,
		MaxConcurrency:   DefaultMaxConcurrency,
		MinTextThreshold: DefaultMinTextThreshold,
		TimeoutMinutes:   DefaultTimeoutMinutes,
		LogLevel:         DefaultLogLevel,
		EnableVerbose:    DefaultEnableVerbose,
	}
}

// LoadConfigWithEnvOverrides creates config and applies environment variable overrides
func LoadConfigWithEnvOverrides() *Config {
	config := NewConfig()

	// Apply environment variable overrides for runtime settings
	if value := os.Getenv("DOC_TEXT_OCR_STRATEGY"); value != "" {
		config.OCRStrategy = types.OCRStrategy(value)
	}
	if value := os.Getenv("DOC_TEXT_LLM_TEMPLATE"); value != "" {
		config.LLMTemplate = value
	}
	if value := os.Getenv("DOC_TEXT_CONTENT_TYPE"); value != "" {
		config.ContentType = types.ContentType(value)
	}
	if value := os.Getenv("DOC_TEXT_SKIP_EXISTING"); value != "" {
		config.SkipExisting = value == "true" || value == "1" || value == "yes"
	}
	if value := os.Getenv("DOC_TEXT_MAX_CONCURRENCY"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			config.MaxConcurrency = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_MIN_TEXT_THRESHOLD"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			config.MinTextThreshold = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_TIMEOUT_MINUTES"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			config.TimeoutMinutes = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_LOG_LEVEL"); value != "" {
		config.LogLevel = value
	}
	if value := os.Getenv("DOC_TEXT_VERBOSE"); value != "" {
		config.EnableVerbose = value == "true" || value == "1" || value == "yes"
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Basic validation for runtime settings
	if c.MaxConcurrency < 1 {
		return utils.NewValidationError("max concurrency must be at least 1", nil)
	}
	if c.MaxConcurrency > 20 {
		return utils.NewValidationError("max concurrency should not exceed 20", nil)
	}
	if c.MinTextThreshold < 0 {
		return utils.NewValidationError("min text threshold must be non-negative", nil)
	}
	if c.TimeoutMinutes < 1 {
		return utils.NewValidationError("timeout must be at least 1 minute", nil)
	}

	return nil
}

// CreateFileManagers creates both intermediate and temporary file managers
func (c *Config) CreateFileManagers(inputFile, md5Hash string, log *logger.Logger) (interfaces.IntermediateFileManager, interfaces.TempFileManager) {
	intermediateManager := utils.NewIntermediateManager(inputFile, md5Hash, log)
	tempManager := utils.NewSimpleTempManager(inputFile, md5Hash, log)
	return intermediateManager, tempManager
}

// CreateIntermediateManager creates an intermediate file manager
func (c *Config) CreateIntermediateManager(inputFile, md5Hash string, log *logger.Logger) interfaces.IntermediateFileManager {
	return utils.NewIntermediateManager(inputFile, md5Hash, log)
}

// CreateTempFileManager creates a temporary file manager
func (c *Config) CreateTempFileManager(inputFile, md5Hash string, log *logger.Logger) interfaces.TempFileManager {
	return utils.NewSimpleTempManager(inputFile, md5Hash, log)
}

// GetOutputDir returns the output directory based on input file directory and MD5 hash
func (c *Config) GetOutputDir(inputFilePath, md5Hash string) string {
	inputDir := utils.NormalizePath(utils.JoinPath(inputFilePath, ".."))
	return utils.JoinPath(inputDir, md5Hash)
}

// GetTextFilePath returns the text file path for a given input file and MD5 hash
func (c *Config) GetTextFilePath(inputFilePath, md5Hash string) string {
	return utils.JoinPath(c.GetOutputDir(inputFilePath, md5Hash), "text.txt")
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	return &Config{
		OCRStrategy:      c.OCRStrategy,
		LLMTemplate:      c.LLMTemplate,
		ContentType:      c.ContentType,
		SkipExisting:     c.SkipExisting,
		MaxConcurrency:   c.MaxConcurrency,
		MinTextThreshold: c.MinTextThreshold,
		TimeoutMinutes:   c.TimeoutMinutes,
		LogLevel:         c.LogLevel,
		EnableVerbose:    c.EnableVerbose,
	}
}
