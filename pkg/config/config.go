package config

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Tool paths
	DefaultLLMCallerPath    = "llm-caller"
	DefaultSuryaOCRPath     = "surya_ocr"
	DefaultPandocPath       = "pandoc"
	DefaultGhostscriptPath  = "gs"
	DefaultCalibrePathMacOS = "/Applications/calibre.app/Contents/MacOS/ebook-convert"
	DefaultCalibrePathLinux = "ebook-convert"
)

// Config holds application configuration
type Config struct {
	// External tool paths
	LLMCallerPath   string `json:"llm_caller_path"`
	SuryaOCRPath    string `json:"surya_ocr_path"`
	CalibrePath     string `json:"calibre_path"`
	PandocPath      string `json:"pandoc_path"`
	GhostscriptPath string `json:"ghostscript_path"`

	// Runtime settings (not persisted to file)
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

// DefaultConfig returns the configuration by loading from file or creating default
func DefaultConfig() *Config {
	// Try to load config from file first
	config, err := LoadConfig()
	if err != nil {
		// If loading fails, create a basic default config
		fmt.Printf("Warning: Failed to load config file, using basic defaults: %v\n", err)

		return &Config{
			LLMCallerPath:    "",
			SuryaOCRPath:     "",
			CalibrePath:      "",
			PandocPath:       "",
			GhostscriptPath:  "",
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

	return config
}

// LoadConfigWithEnvOverrides loads config from file and applies environment variable overrides
func LoadConfigWithEnvOverrides() *Config {
	config := DefaultConfig()

	// Apply environment variable overrides for tool paths
	if value := os.Getenv("LLM_CALLER_PATH"); value != "" {
		config.LLMCallerPath = value
	}
	if value := os.Getenv("SURYA_OCR_PATH"); value != "" {
		config.SuryaOCRPath = value
	}
	if value := os.Getenv("CALIBRE_PATH"); value != "" {
		config.CalibrePath = value
	}
	if value := os.Getenv("PANDOC_PATH"); value != "" {
		config.PandocPath = value
	}
	if value := os.Getenv("GHOSTSCRIPT_PATH"); value != "" {
		config.GhostscriptPath = value
	}

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
	validator := NewConfigValidator()
	return validator.Validate(c)
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
	inputDir := filepath.Dir(inputFilePath)
	return filepath.Join(inputDir, md5Hash)
}

// GetTextFilePath returns the text file path for a given input file and MD5 hash
func (c *Config) GetTextFilePath(inputFilePath, md5Hash string) string {
	return filepath.Join(c.GetOutputDir(inputFilePath, md5Hash), "text.txt")
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	return &Config{
		LLMCallerPath:    c.LLMCallerPath,
		SuryaOCRPath:     c.SuryaOCRPath,
		CalibrePath:      c.CalibrePath,
		PandocPath:       c.PandocPath,
		GhostscriptPath:  c.GhostscriptPath,
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

// String returns a string representation of the configuration
func (c *Config) String() string {
	return fmt.Sprintf("Config{OCRStrategy: %s, LogLevel: %s, Verbose: %v}",
		c.OCRStrategy, c.LogLevel, c.EnableVerbose)
}
