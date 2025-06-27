package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"
)

// DefaultFileProcessor implements FileProcessor interface
type DefaultFileProcessor struct {
	config       *config.Config
	logger       *logger.Logger
	factory      interfaces.ExtractorFactory
	fileManager  *utils.FileManager
	errorHandler *utils.ErrorHandler
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(cfg *config.Config, log *logger.Logger) interfaces.FileProcessor {
	processor := &DefaultFileProcessor{
		config:       cfg,
		logger:       log,
		errorHandler: utils.NewErrorHandler(),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration validation failed: %v", err)
	}

	// Create and set default factory
	processor.factory = NewExtractorFactory(cfg, log)

	// Setup error recovery strategies
	processor.setupErrorRecovery()

	log.Info("File processor initialized with configuration")
	log.Info("Runtime settings applied from environment variables and command line")
	log.Info("Skip existing: %v", cfg.SkipExisting)
	log.Info("Output directory: using input file directory with MD5 hash")
	log.Info("Max concurrency: %d", cfg.MaxConcurrency)
	log.Info("Min text threshold: %d", cfg.MinTextThreshold)

	return processor
}

// setupErrorRecovery configures error recovery strategies
func (p *DefaultFileProcessor) setupErrorRecovery() {
	// Register file I/O error recovery
	p.errorHandler.RegisterRecoveryStrategy(utils.ErrorTypeIO, func(err error) error {
		p.logger.Warn("I/O error detected, checking file accessibility")
		return nil // Allow retry with different approach
	})

	// Register timeout recovery
	p.errorHandler.RegisterRecoveryStrategy(utils.ErrorTypeTimeout, func(err error) error {
		p.logger.Warn("Timeout detected, will retry with extended timeout")
		return nil // Allow retry
	})
}

// ProcessFile processes a file and returns extracted text
func (p *DefaultFileProcessor) ProcessFile(ctx context.Context, inputFile, outputFile string) (*interfaces.ExtractionResult, error) {
	startTime := time.Now()

	p.logger.Progress("📋", "=== Starting file processing ===")
	p.logger.Progress("📂", "Input file: %s", inputFile)
	p.logger.Progress("📤", "Output file: %s", outputFile)

	// Validate input file
	if err := p.validateInputFile(inputFile); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeValidation, "input file validation failed")
	}

	// Get file information with retry
	var fileInfo *types.FileInfo
	err := utils.WithRetry(func() error {
		info, infoErr := utils.GetFileInfo(inputFile)
		if infoErr != nil {
			return utils.WrapError(infoErr, utils.ErrorTypeIO, "failed to get file info")
		}
		fileInfo = info
		return nil
	}, constants.DefaultMaxRetries, p.errorHandler)

	if err != nil {
		p.logger.Error("Failed to get file information: %v", err)
		return nil, err
	}

	p.logger.Info("File analysis completed:")
	p.logger.Info("  Extension: %s", fileInfo.Extension)
	p.logger.Info("  MIME type: %s", fileInfo.MimeType)
	p.logger.Info("  Size: %d bytes", fileInfo.Size)
	p.logger.Info("  MD5 hash: %s", fileInfo.MD5Hash)
	p.logger.Info("  Media type: %s", fileInfo.MediaType)

	// Check file size limits
	maxFileSize := int64(100 * 1024 * 1024) // 100MB
	if fileInfo.Size > maxFileSize*2 {
		p.logger.Warn("File size (%d bytes) exceeds recommended limit (%d bytes)", fileInfo.Size, maxFileSize)
	} else if fileInfo.Size > maxFileSize {
		p.logger.Warn("Large file detected (%d MB), processing may take longer", fileInfo.Size/(1024*1024))
	}

	// Skip existing file if enabled
	if p.config.SkipExisting {
		if result, err := p.loadExistingResult(outputFile, inputFile); err == nil {
			p.logger.ProgressAlways("⏭️", "Output file already exists, skipping extraction")
			return result, nil
		}
	}

	// Process with resource management
	return p.processWithResourceManagement(ctx, inputFile, outputFile, fileInfo, startTime)
}

// validateInputFile validates the input file
func (p *DefaultFileProcessor) validateInputFile(inputFile string) error {
	if inputFile == "" {
		return utils.NewValidationError("input file path cannot be empty", nil)
	}

	// Check if file exists and get file info
	fileInfo, err := os.Stat(inputFile)
	if os.IsNotExist(err) {
		return utils.NewNotFoundError(fmt.Sprintf("file not found: %s", inputFile), err)
	}
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, "failed to get file information")
	}

	// Check if input path is a directory
	if fileInfo.IsDir() {
		return utils.NewValidationError(fmt.Sprintf("input path '%s' is a directory, please specify a file", inputFile), nil)
	}

	// Check if file is readable
	if file, err := os.Open(inputFile); err != nil {
		return utils.NewPermissionError(fmt.Sprintf("permission denied: %s", inputFile), err)
	} else {
		file.Close()
	}

	return nil
}

// loadExistingResult loads existing processing results if available
func (p *DefaultFileProcessor) loadExistingResult(outputFile, inputFile string) (*interfaces.ExtractionResult, error) {
	if _, err := os.Stat(outputFile); err == nil {
		p.logger.Info("Output file already exists, skipping extraction")
		p.logger.Info("Loading existing content from: %s", outputFile)

		content, err := os.ReadFile(outputFile)
		if err != nil {
			return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to read existing output file")
		}

		return &interfaces.ExtractionResult{
			Text:          string(content),
			Source:        inputFile,
			ExtractorUsed: "cached",
			ProcessTime:   0,
		}, nil
	}
	return nil, utils.NewNotFoundError("existing result not found", nil)
}

// processWithResourceManagement processes file with proper resource management
func (p *DefaultFileProcessor) processWithResourceManagement(ctx context.Context, inputFile, outputFile string, fileInfo *types.FileInfo, startTime time.Time) (*interfaces.ExtractionResult, error) {
	var result *interfaces.ExtractionResult

	// Initialize file manager if not already done
	if p.fileManager == nil {
		p.fileManager = p.config.CreateFileManager(inputFile, fileInfo.MD5Hash, p.logger)
	}

	err := p.fileManager.WithCleanup(func() error {
		// Create extractors with fallback options
		p.logger.Debug("Creating extractor chain with fallback options...")
		extractors, err := p.factory.CreateExtractorWithFallbacks(fileInfo)
		if err != nil {
			return utils.WrapError(err, utils.ErrorTypeUnsupported, "no suitable extractor found")
		}

		p.logger.Info("Created extraction chain with %d extractors", len(extractors))

		// Try each extractor in sequence with retry logic
		extractionResult, err := p.attemptExtractionWithFallbacks(ctx, inputFile, extractors)
		if err != nil {
			return err
		}

		// Save to output file if specified
		if outputFile != "" {
			if err := p.saveToFileWithRetry(extractionResult.Text, outputFile); err != nil {
				return utils.WrapError(err, utils.ErrorTypeIO, "failed to save output file")
			}
			p.logger.ProgressAlways("💾", "Text saved to: %s", outputFile)
		}

		// Set processing time
		extractionResult.ProcessTime = time.Since(startTime).Milliseconds()
		result = extractionResult
		return nil
	})

	if err != nil {
		return nil, err
	}

	p.logger.ProgressAlways("✅", "Text extraction completed successfully in %dms", result.ProcessTime)
	p.logger.Progress("✅", "=== File processing completed ===")

	return result, nil
}

// attemptExtractionWithFallbacks tries each extractor with fallback support
func (p *DefaultFileProcessor) attemptExtractionWithFallbacks(ctx context.Context, inputFile string, extractors []interfaces.Extractor) (*interfaces.ExtractionResult, error) {
	var lastError error
	var attemptedExtractors []string
	fallbackUsed := false

	for i, extractor := range extractors {
		extractorName := extractor.Name()
		attemptedExtractors = append(attemptedExtractors, extractorName)

		// Important extraction attempt information always shown
		p.logger.ProgressAlways("🔍", "Attempting extraction with %s (attempt %d/%d)", extractorName, i+1, len(extractors))

		// Mark as fallback if not the first extractor
		if i > 0 {
			fallbackUsed = true
			p.logger.Warn("Primary extractor failed, trying fallback: %s", extractorName)
		}

		// Extract text with retry mechanism
		extractResult, err := p.extractWithRetry(ctx, extractor, inputFile, extractorName)
		if err != nil {
			lastError = err

			// Handle specific error types with recovery strategies
			if appErr, ok := err.(*utils.AppError); ok {
				if err := p.errorHandler.Handle(appErr, true); err == nil {
					// If recovery was successful, continue to next extractor
					continue
				}
			}

			p.logger.Warn("Extractor '%s' failed: %v", extractorName, err)
			continue
		}

		// Validate extracted text
		if len(extractResult) < p.config.MinTextThreshold {
			lastError = utils.NewValidationError(
				fmt.Sprintf("extracted text too short (%d characters)", len(extractResult)),
				nil,
			)
			p.logger.Warn("Extractor '%s' returned insufficient text (%d chars)", extractorName, len(extractResult))
			continue
		}

		// Success! Create and return result
		result := &interfaces.ExtractionResult{
			Text:                extractResult,
			Source:              inputFile,
			ExtractorUsed:       extractorName,
			FallbackUsed:        fallbackUsed,
			AttemptedExtractors: attemptedExtractors,
		}

		// Log success
		if fallbackUsed {
			p.logger.ProgressAlways("✅", "Fallback extractor '%s' succeeded!", extractorName)
		} else {
			p.logger.ProgressAlways("✅", "Primary extractor '%s' succeeded!", extractorName)
		}

		// Log extraction details
		p.logExtractionResult(result, len(extractResult))
		return result, nil
	}

	// If we reach here, all extractors failed
	if lastError == nil {
		lastError = utils.NewUnsupportedError("no text extracted from any available extractor", nil)
	}

	return nil, utils.WrapError(lastError, utils.ErrorTypeOCR,
		fmt.Sprintf("all extractors failed for file: %s", inputFile))
}

// extractWithRetry extracts text with retry mechanism
func (p *DefaultFileProcessor) extractWithRetry(ctx context.Context, extractor interfaces.Extractor, inputFile, extractorName string) (string, error) {
	var extractedText string

	err := utils.WithRetry(func() error {
		text, extractErr := extractor.Extract(ctx, inputFile)
		if extractErr != nil {
			return utils.WrapError(extractErr, utils.ErrorTypeOCR, fmt.Sprintf("extractor '%s' failed", extractorName))
		}
		extractedText = text
		return nil
	}, constants.DefaultMaxRetries, p.errorHandler)

	return extractedText, err
}

// saveToFileWithRetry saves text content to a file with retry logic
func (p *DefaultFileProcessor) saveToFileWithRetry(text, outputFile string) error {
	return utils.WithRetry(func() error {
		return p.saveToFile(text, outputFile)
	}, constants.DefaultMaxRetries, p.errorHandler)
}

// saveToFile saves text content to a file
func (p *DefaultFileProcessor) saveToFile(text, outputFile string) error {
	p.logger.Debug("Attempting to save %d characters to file: %s", len(text), outputFile)

	// Validate output file path
	if outputFile == "" {
		return utils.NewValidationError("output file path cannot be empty", nil)
	}

	// Check if the output path is actually a directory
	if info, err := os.Stat(outputFile); err == nil && info.IsDir() {
		return utils.NewValidationError(fmt.Sprintf("output path '%s' is a directory, please specify a file", outputFile), nil)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	p.logger.Debug("Creating output directory: %s", outputDir)
	if err := os.MkdirAll(outputDir, constants.DefaultDirPermission); err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, fmt.Sprintf("failed to create output directory '%s'", outputDir))
	}

	p.logger.Debug("Output directory created/verified: %s", outputDir)

	// Check if we have write permission to the directory
	testFile := filepath.Join(outputDir, ".write_test_"+filepath.Base(outputFile))
	if err := os.WriteFile(testFile, []byte("test"), constants.DefaultFilePermission); err != nil {
		return utils.WrapError(err, utils.ErrorTypePermission, fmt.Sprintf("no write permission for output directory '%s'", outputDir))
	}
	os.Remove(testFile) // Clean up test file immediately

	// Write text to file
	p.logger.Debug("Writing %d characters to file: %s", len(text), outputFile)
	if err := os.WriteFile(outputFile, []byte(text), constants.DefaultFilePermission); err != nil {
		// Provide more detailed error information
		errorMsg := fmt.Sprintf("failed to write file '%s'", outputFile)
		if os.IsPermission(err) {
			errorMsg += " (permission denied)"
		} else if os.IsExist(err) {
			errorMsg += " (file already exists)"
		} else if os.IsNotExist(err) {
			errorMsg += " (directory does not exist)"
		}
		errorMsg += fmt.Sprintf(" - underlying error: %v", err)

		return utils.WrapError(err, utils.ErrorTypeIO, errorMsg)
	}

	p.logger.Debug("File written successfully: %s", outputFile)
	return nil
}

// SetOutputPath sets custom output path (kept for interface compatibility)
func (p *DefaultFileProcessor) SetOutputPath(outputPath string) {
	p.logger.Debug("SetOutputPath called with: %s (interface compatibility)", outputPath)
}

// SetExtractorFactory sets the extractor factory to use
func (p *DefaultFileProcessor) SetExtractorFactory(factory interfaces.ExtractorFactory) {
	p.factory = factory
	p.logger.Info("Extractor factory updated")
}

// logExtractionResult logs extraction result details
func (p *DefaultFileProcessor) logExtractionResult(result *interfaces.ExtractionResult, textLen int) {
	p.logger.Info("Text extraction completed:")
	p.logger.Info("  Extracted text length: %d characters", textLen)
	p.logger.Info("  Extractor used: %s", result.ExtractorUsed)
	p.logger.Info("  Fallback used: %v", result.FallbackUsed)
	p.logger.Info("  Attempted extractors: %v", result.AttemptedExtractors)
}
