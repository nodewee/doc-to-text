package ocr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/constants"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/ocr/engines"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// OCRExtractor handles PDF and image files using configurable OCR strategies
type OCRExtractor struct {
	name                string
	config              *config.Config
	logger              *logger.Logger
	selector            interfaces.OCRSelector
	tool                interfaces.OCREngine
	intermediateManager interfaces.IntermediateFileManager // For persistent intermediate files
	tempFileManager     interfaces.TempFileManager         // For temporary files
	errorHandler        *utils.ErrorHandler
}

// NewOCRExtractor creates a new OCR extractor
func NewOCRExtractor(cfg *config.Config, log *logger.Logger) interfaces.OCRExtractor {
	return &OCRExtractor{
		name:         "ocr",
		config:       cfg,
		logger:       log,
		selector:     NewOCRSelector(cfg, log),
		errorHandler: utils.NewErrorHandler(),
	}
}

// Extract extracts text using OCR
func (e *OCRExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	// 初始化组件
	if err := e.initialize(inputFile); err != nil {
		return "", err
	}

	// 检查缓存
	if cachedText, found := e.checkCache(inputFile); found {
		return cachedText, nil
	}

	// 执行OCR提取
	result, err := e.performOCR(ctx, inputFile)
	if err != nil {
		return "", err
	}

	// 提取并验证文本
	text, err := e.extractAndValidateText(result)
	if err != nil {
		return "", err
	}

	return text, nil
}

// initialize 初始化OCR组件
func (e *OCRExtractor) initialize(inputFile string) error {
	// 获取文件信息
	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeIO, "failed to get file info")
	}

	// 初始化文件管理器
	if e.intermediateManager == nil || e.tempFileManager == nil {
		e.intermediateManager, e.tempFileManager = e.config.CreateFileManagers(inputFile, fileInfo.MD5Hash, e.logger)
	}

	// 初始化OCR工具
	if e.tool == nil {
		tool, err := e.selector.SelectOCRStrategy(e.config.OCRStrategy)
		if err != nil {
			return utils.WrapError(err, utils.ErrorTypeOCR, "failed to select OCR tool")
		}
		e.tool = tool

		// Set the intermediate manager for the OCR tool to support resume capability
		if suryaEngine, ok := e.tool.(*engines.SuryaOCREngine); ok {
			suryaEngine.SetIntermediateManager(e.intermediateManager)
		}
		if llmEngine, ok := e.tool.(*engines.LLMCallerEngine); ok {
			llmEngine.SetIntermediateManager(e.intermediateManager)
		}
	}

	return nil
}

// checkCache 检查是否有缓存的结果
func (e *OCRExtractor) checkCache(inputFile string) (string, bool) {
	if !e.config.SkipExisting {
		return "", false
	}

	// Use intermediate manager for final text file
	textFilePath := e.intermediateManager.GetTextFilePath()
	if content, err := os.ReadFile(textFilePath); err == nil {
		e.logger.Progress("📄", "Loading cached text: %s", textFilePath)
		return string(content), true
	}

	return "", false
}

// performOCR 执行OCR处理
func (e *OCRExtractor) performOCR(ctx context.Context, inputFile string) (map[string]interface{}, error) {
	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to get file info")
	}

	e.logger.Progress("🔍", "Starting OCR processing: %s", filepath.Base(inputFile))

	ext := strings.ToLower(fileInfo.Extension)
	switch ext {
	case "pdf":
		return e.extractFromPDFViaSinglePages(ctx, inputFile, fileInfo)
	default:
		if utils.IsImageFile(ext) {
			return e.processImage(ctx, inputFile, fileInfo)
		}
		return nil, utils.NewUnsupportedError(fmt.Sprintf("unsupported file type for OCR: %s", ext), nil)
	}
}

// processImage 处理图像文件
func (e *OCRExtractor) processImage(ctx context.Context, inputFile string, fileInfo *types.FileInfo) (map[string]interface{}, error) {
	e.logger.Progress("🖼️", "Processing image: %s", filepath.Base(inputFile))

	text, err := e.tool.ExtractTextFromImage(ctx, inputFile)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeOCR, "image OCR failed")
	}

	return map[string]interface{}{
		"text":   text,
		"source": inputFile,
		"tool":   e.tool.GetDescription(),
	}, nil
}

// extractAndValidateText 提取并验证文本
func (e *OCRExtractor) extractAndValidateText(result map[string]interface{}) (string, error) {
	text, err := e.extractTextFromOCRData(result)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeConversion, "failed to extract text from OCR result")
	}

	// 验证文本长度
	if len(strings.TrimSpace(text)) < e.config.MinTextThreshold {
		return "", utils.NewOCRError(
			fmt.Sprintf("extracted text below minimum threshold (%d characters)", e.config.MinTextThreshold),
			nil)
	}

	return text, nil
}

// ExtractWithOCR performs OCR extraction and returns structured data
func (e *OCRExtractor) ExtractWithOCR(ctx context.Context, inputFile string) (map[string]interface{}, error) {
	return e.performOCR(ctx, inputFile)
}

// setupErrorRecovery configures error recovery strategies
func (e *OCRExtractor) setupErrorRecovery() {
	// Register timeout recovery - retry with shorter timeout
	e.errorHandler.RegisterRecoveryStrategy(utils.ErrorTypeTimeout, func(err error) error {
		e.logger.Warn("Timeout detected, will retry with shorter processing window")
		return nil // Allow retry
	})

	// Register OCR error recovery - try alternative engines
	e.errorHandler.RegisterRecoveryStrategy(utils.ErrorTypeOCR, func(err error) error {
		e.logger.Warn("OCR error detected, alternative strategies will be attempted")
		return nil // Allow fallback to other extractors
	})
}

// extractFromPDFViaSinglePages extracts text from PDF by splitting into single-page PDFs first
func (e *OCRExtractor) extractFromPDFViaSinglePages(ctx context.Context, inputFile string, fileInfo *types.FileInfo) (map[string]interface{}, error) {
	e.logger.Progress("📄", "Processing PDF document: %s", filepath.Base(inputFile))

	// Use intermediate manager for persistent pages directory
	pagesDir := e.intermediateManager.GetPagesDir()
	if _, err := e.intermediateManager.CreateIntermediateDir("pages"); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "failed to create pages directory")
	}

	e.logger.Progress("📁", "Created pages directory: %s", pagesDir)

	// Split PDF into single-page PDFs with error handling
	var actualPageCount int
	err := utils.WithRetry(func() error {
		count, splitErr := e.splitPDFIntoSinglePages(ctx, inputFile, pagesDir)
		if splitErr != nil {
			return splitErr
		}
		if count <= 0 {
			return utils.NewValidationError("PDF has no pages", nil)
		}
		actualPageCount = count
		return nil
	}, 3, e.errorHandler)

	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeConversion, "failed to split PDF into single pages")
	}

	// Process each single-page PDF individually with OCR
	e.logger.Info("Processing %d single-page PDFs with OCR", actualPageCount)
	var allText strings.Builder
	processedPages := 0

	for pageNum := 1; pageNum <= actualPageCount; pageNum++ {
		pagePDFPath := e.intermediateManager.GetPagePDFPath(pageNum)
		pageTextPath := e.intermediateManager.GetPageTextPath(pageNum)
		displayPageNum := pageNum // Page numbers now match display numbers (1-based)

		e.logger.Progress("📄", "Processing page %d / %d ...", displayPageNum, actualPageCount)

		// Check if page text already exists (resume capability)
		if content, err := e.loadExistingPageText(pageTextPath, displayPageNum); err == nil {
			allText.WriteString(content)
			allText.WriteString("\n")
			processedPages++
			continue
		}

		// Check if single-page PDF exists
		if _, err := os.Stat(pagePDFPath); os.IsNotExist(err) {
			e.logger.Warn("Single-page PDF %d not found, skipping", displayPageNum)
			continue
		}

		// Process single-page PDF with OCR
		pageText, err := e.processPageWithRetry(ctx, pagePDFPath, displayPageNum)
		if err != nil {
			e.logger.Warn("Failed to extract text from page %d: %v", displayPageNum, err)
			continue
		}

		// Format and save page text
		formattedPageText := fmt.Sprintf("--- Page %d ---\n%s", displayPageNum, pageText)

		// Save page text for resume capability
		if saveErr := os.WriteFile(pageTextPath, []byte(formattedPageText), 0644); saveErr != nil {
			e.logger.Warn("Failed to save page %d text: %v", displayPageNum, saveErr)
		}

		allText.WriteString(formattedPageText)
		allText.WriteString("\n")
		processedPages++
	}

	e.logger.Progress("✅", "Processed %d/%d pages successfully using %s", processedPages, actualPageCount, e.tool.Name())

	result := map[string]interface{}{
		"text":       allText.String(),
		"page_count": actualPageCount,
		"pages_dir":  pagesDir,
		"source":     "pdf_single_pages",
		"tool":       e.tool.Name(),
	}

	return result, nil
}

// loadExistingPageText loads existing page text if available
func (e *OCRExtractor) loadExistingPageText(pageTextPath string, displayPageNum int) (string, error) {
	if _, err := os.Stat(pageTextPath); err == nil {
		e.logger.Info("Page %d text already exists, loading...", displayPageNum)
		content, err := os.ReadFile(pageTextPath)
		if err != nil {
			return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to read existing page text")
		}
		return string(content), nil
	}
	return "", utils.NewNotFoundError("page text not found", nil)
}

// processPageWithRetry processes a page with retry logic
func (e *OCRExtractor) processPageWithRetry(ctx context.Context, pagePDFPath string, displayPageNum int) (string, error) {
	e.logger.Info("OCRing page %d ...", displayPageNum)
	var pageText string
	err := utils.WithRetry(func() error {
		text, err := e.processPagePDFWithOCR(ctx, pagePDFPath, displayPageNum)
		if err != nil {
			return utils.WrapError(err, utils.ErrorTypeOCR, fmt.Sprintf("OCR failed for page %d", displayPageNum))
		}

		// Validate page text meets minimum requirements
		if len(strings.TrimSpace(text)) == 0 {
			return utils.NewOCRError(fmt.Sprintf("no text found in page %d", displayPageNum), nil)
		}

		pageText = text
		return nil
	}, 2, e.errorHandler)

	if err != nil {
		return "", err
	}

	return pageText, nil
}

// splitPDFIntoSinglePages splits a PDF into individual single-page PDFs using Ghostscript
// Uses a simpler approach: let Ghostscript split all pages, then count the resulting files
func (e *OCRExtractor) splitPDFIntoSinglePages(ctx context.Context, pdfPath, outputDir string) (int, error) {
	e.logger.Debug("Starting PDF splitting process")
	e.logger.Debug("Input PDF: %s", pdfPath)
	e.logger.Debug("Output directory: %s", outputDir)

	// Ensure output directory exists before running gs command
	e.logger.Debug("Creating output directory: %s", outputDir)
	if err := utils.EnsureDir(outputDir); err != nil {
		return 0, utils.WrapError(err, utils.ErrorTypeIO, "failed to create output directory for PDF splitting")
	}

	// Check if pages already exist (resume capability)
	existingCount := e.countExistingPages(outputDir)
	if existingCount > 0 {
		e.logger.Info("Found %d existing page files, resuming from there", existingCount)
		return existingCount, nil
	}

	// Use Ghostscript to split all pages at once with pattern-based output
	// Normalize paths for cross-platform compatibility
	normalizedPDFPath := utils.NormalizePath(pdfPath)
	normalizedOutputDir := utils.NormalizePath(outputDir)
	outputPattern := filepath.Join(normalizedOutputDir, "page_%d.pdf")

	// Build Ghostscript command with proper path handling
	var args []string
	args = append(args, "-dNOPAUSE", "-dBATCH", "-sDEVICE=pdfwrite")
	args = append(args, "-dFirstPage=1", "-dLastPage=999999") // Use a very high number to ensure all pages are processed

	// On Windows, we might need to quote the output file pattern if it contains spaces
	if constants.IsWindows() {
		args = append(args, fmt.Sprintf("-sOutputFile=%s", outputPattern))
	} else {
		args = append(args, fmt.Sprintf("-sOutputFile=%s", outputPattern))
	}

	args = append(args, normalizedPDFPath)

	cmd := exec.CommandContext(ctx, e.config.GhostscriptPath, args...)

	e.logger.Info("Splitting PDF into individual pages...")
	e.logger.Debug("Command: %s %s", e.config.GhostscriptPath, strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just a "page range exceeds" error, which is expected
		outputStr := string(output)
		if !strings.Contains(outputStr, "exceeds page count") && !strings.Contains(outputStr, "Invalid page") {
			return 0, utils.WrapError(err, utils.ErrorTypeSystem, fmt.Sprintf("failed to split PDF: %s", outputStr))
		}
		e.logger.Debug("Ghostscript completed with expected page range warning: %s", outputStr)
	}

	// Count the actual number of files created
	pageCount := e.countExistingPages(outputDir)

	if pageCount == 0 {
		return 0, utils.NewValidationError("no pages were extracted from PDF", nil)
	}

	e.logger.Info("Successfully split PDF into %d page files", pageCount)
	return pageCount, nil
}

// countExistingPages counts the number of existing page files in the output directory
func (e *OCRExtractor) countExistingPages(outputDir string) int {
	normalizedOutputDir := utils.NormalizePath(outputDir)
	count := 0
	for i := 1; i <= 10000; i++ { // Start from 1 since Ghostscript uses 1-based indexing
		pageFile := filepath.Join(normalizedOutputDir, fmt.Sprintf("page_%d.pdf", i))
		pageFile = utils.NormalizePath(pageFile)

		if _, err := os.Stat(pageFile); err == nil {
			// Verify the file has reasonable size (not empty)
			if stat, err := os.Stat(pageFile); err == nil && stat.Size() > 100 {
				count++
			}
		} else {
			// If we hit a missing file, assume we've found all pages
			break
		}
	}
	return count
}

// processPagePDFWithOCR processes a single-page PDF with OCR
func (e *OCRExtractor) processPagePDFWithOCR(ctx context.Context, pagePDFPath string, pageNum int) (string, error) {
	// Use the OCR tool to extract text from the single-page PDF
	pageText, err := e.tool.ExtractTextFromPDF(ctx, pagePDFPath)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeOCR, fmt.Sprintf("OCR failed for page PDF %d", pageNum))
	}

	// Clean up the text
	pageText = strings.TrimSpace(pageText)
	if pageText == "" {
		e.logger.Info("No text found in page PDF %d", pageNum)
		return "", nil
	}

	e.logger.Debug("Extracted %d characters from page PDF %d", len(pageText), pageNum)
	return pageText, nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *OCRExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	ext := strings.ToLower(fileInfo.Extension)
	return ext == "pdf" || utils.IsImageFile(ext)
}

// Name returns the name of the extractor
func (e *OCRExtractor) Name() string {
	return e.name
}

// GetCacheKey returns a unique cache key for the file
func (e *OCRExtractor) GetCacheKey(fileInfo *types.FileInfo) string {
	engineName := "unknown"
	if e.tool != nil {
		engineName = e.tool.Name()
	}
	return fmt.Sprintf("ocr-%s-%s", engineName, fileInfo.MD5Hash)
}

// extractTextFromOCRData extracts text from OCR result data
func (e *OCRExtractor) extractTextFromOCRData(result map[string]interface{}) (string, error) {
	if text, exists := result["text"]; exists {
		if textStr, ok := text.(string); ok {
			return textStr, nil
		}
	}
	return "", utils.NewConversionError("no text found in OCR result", nil)
}

// SetOCREngine allows setting a specific OCR tool (for testing or manual selection)
func (e *OCRExtractor) SetOCREngine(tool interfaces.OCREngine) {
	e.tool = tool
	e.logger.Info("OCR tool manually set to: %s", tool.GetDescription())
}
