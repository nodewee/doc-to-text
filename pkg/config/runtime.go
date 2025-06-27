package config

import (
	"os"
	"strconv"

	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"
)

// Config holds application runtime configuration
type Config struct {
	OCRStrategy      types.OCRStrategy
	LLMTemplate      string
	ContentType      types.ContentType
	SkipExisting     bool
	MaxConcurrency   int
	MinTextThreshold int
	TimeoutMinutes   int
	LogLevel         string
	EnableVerbose    bool
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		OCRStrategy:      types.OCRStrategyInteractive,
		LLMTemplate:      "",
		ContentType:      types.ContentTypeImage,
		SkipExisting:     true,
		MaxConcurrency:   4,
		MinTextThreshold: 10,
		TimeoutMinutes:   30,
		LogLevel:         "info",
		EnableVerbose:    false,
	}
}

// LoadConfigWithEnvOverrides creates config and applies environment variable overrides
func LoadConfigWithEnvOverrides() *Config {
	config := NewConfig()

	// Apply environment variable overrides
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
		config.SkipExisting = value == "true" || value == "1"
	}
	if value := os.Getenv("DOC_TEXT_MAX_CONCURRENCY"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal > 0 {
			config.MaxConcurrency = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_MIN_TEXT_THRESHOLD"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal >= 0 {
			config.MinTextThreshold = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_TIMEOUT_MINUTES"); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal > 0 {
			config.TimeoutMinutes = intVal
		}
	}
	if value := os.Getenv("DOC_TEXT_LOG_LEVEL"); value != "" {
		config.LogLevel = value
	}
	if value := os.Getenv("DOC_TEXT_VERBOSE"); value != "" {
		config.EnableVerbose = value == "true" || value == "1"
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 20 {
		return utils.NewValidationError("max concurrency must be between 1 and 20", nil)
	}
	if c.MinTextThreshold < 0 {
		return utils.NewValidationError("min text threshold must be non-negative", nil)
	}
	if c.TimeoutMinutes < 1 {
		return utils.NewValidationError("timeout must be at least 1 minute", nil)
	}
	return nil
}

// CreateFileManager creates a unified file manager
func (c *Config) CreateFileManager(inputFile, md5Hash string, log *logger.Logger) *utils.FileManager {
	return utils.NewFileManager(inputFile, md5Hash, log)
}
