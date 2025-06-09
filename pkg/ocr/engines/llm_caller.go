package engines

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// LLMCallerEngine uses llm-caller for text extraction
type LLMCallerEngine struct {
	config              *config.Config
	logger              *logger.Logger
	intermediateManager interfaces.IntermediateFileManager
}

// NewLLMCallerEngine creates a new LLM caller engine
func NewLLMCallerEngine(cfg *config.Config, log *logger.Logger) interfaces.OCREngine {
	return &LLMCallerEngine{
		config: cfg,
		logger: log,
	}
}

// Name returns the name of the OCR tool
func (e *LLMCallerEngine) Name() string {
	return "llm-caller"
}

// GetDescription returns a description of the OCR tool
func (e *LLMCallerEngine) GetDescription() string {
	return "LLM Caller with configurable template"
}

// SupportsDirectPDF returns true for single-page PDF processing
func (e *LLMCallerEngine) SupportsDirectPDF() bool {
	return true
}

// IsAvailable checks if the OCR tool is available on the system
func (e *LLMCallerEngine) IsAvailable() bool {
	_, err := exec.LookPath(e.config.LLMCallerPath)
	return err == nil
}

// SetIntermediateManager sets the intermediate file manager for persistent storage
func (e *LLMCallerEngine) SetIntermediateManager(intermediateManager interfaces.IntermediateFileManager) {
	e.intermediateManager = intermediateManager
}

// ExtractTextFromPDF extracts text from PDF using llm-caller
func (e *LLMCallerEngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
	e.logger.Debug("LLM Caller: Processing PDF %s", pdfPath)

	// Ensure intermediate manager is available
	if e.intermediateManager == nil {
		return "", fmt.Errorf("intermediate manager not initialized")
	}

	// Create intermediate directory for LLM Caller results
	outputDir, err := e.intermediateManager.CreateIntermediateDir("llm_caller_results")
	if err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	// Check if LLM results already exist (resume capability)
	sanitizedFileName := utils.SanitizeFileName(filepath.Base(pdfPath))
	resultFile := filepath.Join(outputDir, fmt.Sprintf("%s_llm_result.txt", sanitizedFileName))
	resultFile = utils.NormalizePath(resultFile)

	if content, err := e.loadExistingLLMResults(resultFile); err == nil {
		e.logger.Progress("⏭️", "Loading cached LLM results: %s", resultFile)
		return content, nil
	}

	// Create a temporary output file for the result (in intermediate directory)
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_output.txt", sanitizedFileName))
	outputFile = utils.NormalizePath(outputFile)

	// Build the command based on configuration
	var cmd *exec.Cmd
	normalizedPDFPath := utils.NormalizePath(pdfPath)
	normalizedOutputFile := utils.NormalizePath(outputFile)

	if e.config.LLMTemplate != "" {
		// Use custom template if provided
		cmd = exec.CommandContext(ctx, e.config.LLMCallerPath,
			"--model", "anthropic:claude-3-5-sonnet-20241022",
			"--template", e.config.LLMTemplate,
			"--file", normalizedPDFPath,
			"--output", normalizedOutputFile)
	} else {
		// Use default template
		cmd = exec.CommandContext(ctx, e.config.LLMCallerPath,
			"--model", "anthropic:claude-3-5-sonnet-20241022",
			"--template", "analyze", // default template for document analysis
			"--file", normalizedPDFPath,
			"--output", normalizedOutputFile)
	}

	e.logger.Debug("Running LLM Caller command: %s", cmd.String())

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("LLM Caller failed: %s", string(output))
		return "", fmt.Errorf("llm-caller extraction failed: %w", err)
	}

	e.logger.Debug("LLM Caller completed successfully")

	// Read the output file
	content, err := os.ReadFile(normalizedOutputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM output: %w", err)
	}

	text := string(content)

	// Save the extracted text for future use
	if saveErr := os.WriteFile(resultFile, []byte(text), 0644); saveErr != nil {
		e.logger.Warn("Failed to save LLM results cache: %v", saveErr)
	}

	e.logger.Debug("LLM Caller: Extracted %d characters from PDF", len(text))

	return text, nil
}

// ExtractTextFromImage extracts text from image using llm-caller
func (e *LLMCallerEngine) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	e.logger.Debug("LLM Caller: Processing image %s", imagePath)

	// Ensure intermediate manager is available
	if e.intermediateManager == nil {
		return "", fmt.Errorf("intermediate manager not initialized")
	}

	// Create intermediate directory for LLM Caller results
	outputDir, err := e.intermediateManager.CreateIntermediateDir("llm_caller_image_results")
	if err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	// Check if LLM results already exist (resume capability)
	sanitizedFileName := utils.SanitizeFileName(filepath.Base(imagePath))
	resultFile := filepath.Join(outputDir, fmt.Sprintf("%s_llm_result.txt", sanitizedFileName))
	resultFile = utils.NormalizePath(resultFile)

	if content, err := e.loadExistingLLMResults(resultFile); err == nil {
		e.logger.Progress("⏭️", "Loading cached LLM results: %s", resultFile)
		return content, nil
	}

	// Create a output file for the result (in intermediate directory)
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_output.txt", sanitizedFileName))
	outputFile = utils.NormalizePath(outputFile)

	// Build the command for image processing
	var cmd *exec.Cmd
	normalizedImagePath := utils.NormalizePath(imagePath)
	normalizedOutputFile := utils.NormalizePath(outputFile)

	if e.config.LLMTemplate != "" {
		// Use custom template if provided
		cmd = exec.CommandContext(ctx, e.config.LLMCallerPath,
			"--model", "anthropic:claude-3-5-sonnet-20241022",
			"--template", e.config.LLMTemplate,
			"--file", normalizedImagePath,
			"--output", normalizedOutputFile)
	} else {
		// Use default template for image analysis
		cmd = exec.CommandContext(ctx, e.config.LLMCallerPath,
			"--model", "anthropic:claude-3-5-sonnet-20241022",
			"--template", "image-to-text", // default template for image OCR
			"--file", normalizedImagePath,
			"--output", normalizedOutputFile)
	}

	e.logger.Debug("Running LLM Caller command for image: %s", cmd.String())

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("LLM Caller image processing failed: %s", string(output))
		return "", fmt.Errorf("llm-caller image extraction failed: %w", err)
	}

	e.logger.Debug("LLM Caller image processing completed successfully")

	// Read the output file
	content, err := os.ReadFile(normalizedOutputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM image output: %w", err)
	}

	text := string(content)

	// Save the extracted text for future use
	if saveErr := os.WriteFile(resultFile, []byte(text), 0644); saveErr != nil {
		e.logger.Warn("Failed to save LLM results cache: %v", saveErr)
	}

	e.logger.Debug("LLM Caller: Extracted %d characters from image", len(text))

	return text, nil
}

// loadExistingLLMResults loads existing LLM results if available
func (e *LLMCallerEngine) loadExistingLLMResults(resultFile string) (string, error) {
	if _, err := os.Stat(resultFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no cached results found")
	}

	content, err := os.ReadFile(resultFile)
	if err != nil {
		return "", fmt.Errorf("failed to read cached results: %w", err)
	}

	return string(content), nil
}
