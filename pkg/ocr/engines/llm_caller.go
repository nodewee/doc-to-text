package engines

import (
	"context"
	"encoding/base64"
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
	return "LLM Caller (OCR with configurable AI models)"
}

// SupportsDirectPDF returns true for single-page PDF processing
func (e *LLMCallerEngine) SupportsDirectPDF() bool {
	return true
}

// findLLMCallerPath attempts to find the llm-caller command
func (e *LLMCallerEngine) findLLMCallerPath() (string, error) {
	// Try to find llm-caller using shell detection
	if foundPath, err := utils.DefaultPathUtils.FindExecutableInShell("llm-caller"); err == nil {
		e.logger.Debug("Found llm-caller at: %s", foundPath)
		return foundPath, nil
	}

	// Common installation paths
	commonPaths := []string{
		"llm-caller",
		"/usr/local/bin/llm-caller",
		"/usr/bin/llm-caller",
		"/opt/homebrew/bin/llm-caller",
		"/home/linuxbrew/.linuxbrew/bin/llm-caller",
	}

	for _, path := range commonPaths {
		if utils.DefaultPathUtils.IsCommandAvailable(path) {
			e.logger.Debug("Found llm-caller at common path: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("llm-caller command not found. Please install llm-caller first:\n" +
		"  pip install llm-caller\n" +
		"  or visit: https://github.com/nodewee/llm-caller for installation instructions")
}

// SetIntermediateManager sets the intermediate file manager for persistent storage
func (e *LLMCallerEngine) SetIntermediateManager(intermediateManager interfaces.IntermediateFileManager) {
	e.intermediateManager = intermediateManager
}

// ExtractTextFromPDF extracts text from PDF using llm-caller
func (e *LLMCallerEngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
	e.logger.Debug("LLM Caller: Processing PDF %s", pdfPath)

	// Find llm-caller command at execution time
	llmCallerPath, err := e.findLLMCallerPath()
	if err != nil {
		return "", err
	}

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

	// Determine template to use
	template := e.config.LLMTemplate
	if template == "" {
		template = "analyze" // default template for document analysis
	}

	// Build command: llm-caller call <template> --var file:file:<file> -o <output>
	cmd = exec.CommandContext(ctx, llmCallerPath,
		"call", template,
		"--var", fmt.Sprintf("file:file:%s", normalizedPDFPath),
		"-o", normalizedOutputFile)

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

	// Find llm-caller command at execution time
	llmCallerPath, err := e.findLLMCallerPath()
	if err != nil {
		return "", err
	}

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

	// Determine template to use
	template := e.config.LLMTemplate
	if template == "" {
		template = "image-to-text" // default template for image OCR
	}

	// Build command: llm-caller call <template> --var image_url:text:<data_string> -o <output>
	// "本地图像链接示例(image_url)格式：data:image/{format};base64,{base64_string}）"

	// 获取图片格式
	imageFormat := filepath.Ext(normalizedImagePath)
	if imageFormat == "" {
		imageFormat = "png"
	}

	// 读取图片文件
	imageData, err := os.ReadFile(normalizedImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// 将图片数据转换为 base64 字符串
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)
	imageUrl := fmt.Sprintf("data:image/%s;base64,%s", imageFormat, imageBase64)

	cmd = exec.CommandContext(ctx, llmCallerPath,
		"call", template,
		"--var", fmt.Sprintf("image_url:text:%s", imageUrl),
		"-o", normalizedOutputFile)

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
