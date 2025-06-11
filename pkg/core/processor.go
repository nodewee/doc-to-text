package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/constants"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// DefaultFileProcessor implements FileProcessor interface
type DefaultFileProcessor struct {
	config       *config.Config
	logger       *logger.Logger
	factory      interfaces.ExtractorFactory
	tempManager  interfaces.TempFileManager
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

	log.Info("File processor initialized with configuration:")
	log.Info("  Runtime settings applied from environment variables and command line")
	log.Info("  Skip existing: %v", cfg.SkipExisting)
	log.Info("  Output directory: using input file directory with MD5 hash")
	log.Info("  Max concurrency: %d", cfg.MaxConcurrency)
	log.Info("  Min text threshold: %d", cfg.MinTextThreshold)

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

	p.logger.Info("=== Starting file processing ===")
	p.logger.Info("Input file: %s", inputFile)
	p.logger.Info("Output file: %s", outputFile)

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
	if err := p.validateFileSize(fileInfo.Size); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeValidation, "file size validation failed")
	}

	// Check if output file already exists and skip if configured
	if p.config.SkipExisting && outputFile != "" {
		if result, err := p.loadExistingResult(outputFile, inputFile); err == nil {
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

	// Check if file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return utils.NewNotFoundError(fmt.Sprintf("input file not found: %s", inputFile), err)
	}

	// Check if file is readable
	if file, err := os.Open(inputFile); err != nil {
		return utils.NewPermissionError(fmt.Sprintf("cannot read input file: %s", inputFile), err)
	} else {
		file.Close()
	}

	return nil
}

// validateFileSize validates that the file size is within acceptable limits
func (p *DefaultFileProcessor) validateFileSize(size int64) error {
	if size > constants.MaxFileSize {
		return utils.NewValidationError(
			fmt.Sprintf("file size (%d bytes) exceeds maximum limit (%d bytes)",
				size, constants.MaxFileSize), nil)
	}

	if size > constants.WarnFileSizeLimit {
		p.logger.Warn("Large file detected (%d bytes), processing may take longer", size)
	}

	return nil
}

// loadExistingResult loads existing processing results if available
func (p *DefaultFileProcessor) loadExistingResult(outputFile, inputFile string) (*interfaces.ExtractionResult, error) {
	if _, err := os.Stat(outputFile); err == nil {
		p.logger.Info("⏭️ Output file already exists, skipping extraction,")
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

// processWithResourceManagement handles the main processing with proper resource management
func (p *DefaultFileProcessor) processWithResourceManagement(ctx context.Context, inputFile, outputFile string, fileInfo *types.FileInfo, startTime time.Time) (*interfaces.ExtractionResult, error) {
	var result *interfaces.ExtractionResult

	// Initialize temp manager if not already done
	if p.tempManager == nil {
		p.tempManager = p.config.CreateTempFileManager(inputFile, fileInfo.MD5Hash, p.logger)
	}

	err := p.tempManager.WithCleanup(func() error {
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
			p.logger.Progress("💾", "Text saved to: %s", outputFile)
		}

		// Set processing time
		extractionResult.ProcessTime = time.Since(startTime).Milliseconds()
		result = extractionResult
		return nil
	})

	if err != nil {
		return nil, err
	}

	p.logger.Progress("✅", "Text extraction completed successfully in %dms", result.ProcessTime)
	p.logger.Info("=== File processing completed ===")

	return result, nil
}

// attemptExtractionWithFallbacks tries each extractor with fallback support
func (p *DefaultFileProcessor) attemptExtractionWithFallbacks(ctx context.Context, inputFile string, extractors []interfaces.Extractor) (*interfaces.ExtractionResult, error) {
	var text string
	var lastError error
	var extractorUsed string
	var attemptedExtractors []string
	fallbackUsed := false

	// Get file extension to check if it's a PDF
	ext := strings.ToLower(filepath.Ext(inputFile))
	isPDF := ext == ".pdf"

	// Check if user explicitly chose a specific OCR strategy
	userChosenOCR := p.config.OCRStrategy != types.OCRStrategyInteractive

	for i, extractor := range extractors {
		extractorName := extractor.Name()
		attemptedExtractors = append(attemptedExtractors, extractorName)

		p.logger.Progress("🔍", "Attempting extraction with %s (attempt %d/%d)", extractorName, i+1, len(extractors))

		if i > 0 {
			fallbackUsed = true
			p.logger.Warn("Primary extractor failed, trying fallback: %s", extractorName)
		}

		// Attempt extraction with retry
		err := utils.WithRetry(func() error {
			extractedText, extractErr := extractor.Extract(ctx, inputFile)
			if extractErr != nil {
				return utils.WrapError(extractErr, utils.ErrorTypeOCR, fmt.Sprintf("extractor '%s' failed", extractorName))
			}

			// Validate extracted text
			if len(extractedText) < p.config.MinTextThreshold {
				return utils.NewValidationError(
					fmt.Sprintf("extracted text below minimum threshold (%d chars)", p.config.MinTextThreshold), nil)
			}

			text = extractedText
			return nil
		}, constants.DefaultMaxRetries, p.errorHandler)

		if err != nil {
			p.logger.Warn("Extractor '%s' failed: %v", extractorName, err)
			lastError = err

			// Special handling for image content type with PDF files
			if isPDF && p.config.ContentType == types.ContentTypeImage && extractorName == "ocr" {
				p.logger.Error("OCR failed for image-based PDF content. No fallback will be attempted.")
				return &interfaces.ExtractionResult{
					Source:              inputFile,
					ExtractorUsed:       "",
					Error:               fmt.Sprintf("OCR extraction failed for image-based PDF: %v", err),
					FallbackUsed:        false,
					AttemptedExtractors: attemptedExtractors,
				}, err
			}

			// If user explicitly chose an OCR strategy and OCR failed, don't fallback
			if userChosenOCR && extractorName == "ocr" {
				p.logger.Error("OCR failed for user-chosen strategy (%s). No fallback will be attempted.", p.config.OCRStrategy)
				return &interfaces.ExtractionResult{
					Source:              inputFile,
					ExtractorUsed:       "",
					Error:               fmt.Sprintf("OCR extraction failed for chosen strategy %s: %v", p.config.OCRStrategy, err),
					FallbackUsed:        false,
					AttemptedExtractors: attemptedExtractors,
				}, err
			}

			continue
		}

		// Success!
		extractorUsed = extractorName
		if fallbackUsed {
			p.logger.Progress("✅", "Fallback extractor '%s' succeeded!", extractorName)
		} else {
			p.logger.Progress("✅", "Primary extractor '%s' succeeded!", extractorName)
		}
		break
	}

	// If all extractors failed
	if text == "" && lastError != nil {
		p.logger.Error("All extraction methods failed. Last error: %v", lastError)
		return &interfaces.ExtractionResult{
			Source:              inputFile,
			ExtractorUsed:       "",
			Error:               fmt.Sprintf("all extractors failed, last error: %v", lastError),
			FallbackUsed:        fallbackUsed,
			AttemptedExtractors: attemptedExtractors,
		}, lastError
	}

	// Log extraction results
	textLen := len(text)
	p.logger.Info("Text extraction completed:")
	p.logger.Info("  Extracted text length: %d characters", textLen)
	p.logger.Info("  Extractor used: %s", extractorUsed)
	p.logger.Info("  Fallback used: %v", fallbackUsed)
	p.logger.Info("  Attempted extractors: %v", attemptedExtractors)

	return &interfaces.ExtractionResult{
		Text:                text,
		Source:              inputFile,
		ExtractorUsed:       extractorUsed,
		FallbackUsed:        fallbackUsed,
		AttemptedExtractors: attemptedExtractors,
	}, nil
}

// saveToFileWithRetry saves text content to a file with retry logic
func (p *DefaultFileProcessor) saveToFileWithRetry(text, outputFile string) error {
	return utils.WithRetry(func() error {
		return p.saveToFile(text, outputFile)
	}, constants.DefaultMaxRetries, p.errorHandler)
}

// saveToFile saves text content to a file
func (p *DefaultFileProcessor) saveToFile(text, outputFile string) error {
	p.logger.Debug("Creating output directory...")

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, constants.DefaultDirPermission); err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, "failed to create output directory")
	}

	p.logger.Debug("Output directory created/verified: %s", outputDir)

	// Write text to file
	p.logger.Debug("Writing %d characters to file...", len(text))
	if err := os.WriteFile(outputFile, []byte(text), constants.DefaultFilePermission); err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, "failed to write text file")
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
