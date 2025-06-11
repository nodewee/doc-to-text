package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/constants"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// EbookExtractor extracts text from e-books using Calibre
type EbookExtractor struct {
	name        string
	config      *config.Config
	logger      *logger.Logger
	tempManager interfaces.TempFileManager
}

// NewEbookExtractor creates a new e-book extractor
func NewEbookExtractor(cfg *config.Config, log *logger.Logger) interfaces.Extractor {
	return &EbookExtractor{
		name:   "ebook",
		config: cfg,
		logger: log,
	}
}

// findCalibrePath attempts to find the Calibre ebook-convert command
func (e *EbookExtractor) findCalibrePath() (string, error) {
	// Try to find ebook-convert using shell detection
	if foundPath, err := utils.DefaultPathUtils.FindExecutableInShell("ebook-convert"); err == nil {
		e.logger.Debug("Found ebook-convert at: %s", foundPath)
		return foundPath, nil
	}

	// Common installation paths based on platform
	platformConfig := constants.GetPlatformConfig()
	for _, path := range platformConfig.CalibrePaths {
		if utils.DefaultPathUtils.IsCommandAvailable(path) {
			e.logger.Debug("Found ebook-convert at common path: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("Calibre ebook-convert command not found. Please install Calibre first:\n" +
		"  - macOS: brew install calibre\n" +
		"  - Ubuntu/Debian: sudo apt-get install calibre\n" +
		"  - Windows: Download from https://calibre-ebook.com/download\n" +
		"  or visit: https://calibre-ebook.com for installation instructions")
}

// Extract extracts text from e-books using Calibre
func (e *EbookExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	e.logger.Info("Extracting text from e-book: %s", inputFile)

	// Find Calibre command at execution time
	calibrePath, err := e.findCalibrePath()
	if err != nil {
		return "", err
	}

	// Initialize temp manager if not already done
	if e.tempManager == nil {
		md5Hash, err := utils.CalculateFileMD5(inputFile)
		if err != nil {
			return "", fmt.Errorf("failed to calculate file hash: %w", err)
		}
		e.tempManager = e.config.CreateTempFileManager(inputFile, md5Hash, e.logger)
	}

	var result string
	err = e.tempManager.WithCleanup(func() error {
		// Create temporary output file
		outputFile, err := e.tempManager.CreateTempFile("ebook_output_", ".txt")
		if err != nil {
			return fmt.Errorf("failed to create temp output file: %w", err)
		}

		// Run Calibre ebook-convert
		cmd := exec.CommandContext(ctx, calibrePath, inputFile, outputFile)
		e.logger.Debug("Running Calibre command: %s", cmd.String())

		output, err := cmd.CombinedOutput()
		if err != nil {
			e.logger.Error("Calibre e-book conversion failed: %s", string(output))
			return fmt.Errorf("calibre e-book conversion failed: %w", err)
		}

		// Read the converted text
		content, err := os.ReadFile(outputFile)
		if err != nil {
			return fmt.Errorf("failed to read Calibre output: %w", err)
		}

		text := strings.TrimSpace(string(content))
		if len(text) < e.config.MinTextThreshold {
			return fmt.Errorf("extracted text is too short (%d chars, minimum %d)", len(text), e.config.MinTextThreshold)
		}

		result = text
		e.logger.Debug("E-book extraction successful: %d characters", len(text))
		return nil
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *EbookExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	return utils.IsEbookFile(fileInfo.Extension, fileInfo.MimeType)
}

// Name returns the name of the extractor
func (e *EbookExtractor) Name() string {
	return e.name
}
