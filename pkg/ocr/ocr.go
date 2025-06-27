package ocr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"
)

// OCRExtractor unified OCR processor
type OCRExtractor struct {
	name        string
	config      *config.Config
	logger      *logger.Logger
	fileManager *utils.FileManager
}

// NewOCRExtractor creates a new OCR extractor
func NewOCRExtractor(cfg *config.Config, log *logger.Logger) interfaces.Extractor {
	return &OCRExtractor{
		name:   "ocr",
		config: cfg,
		logger: log,
	}
}

// Extract extracts text from files
func (e *OCRExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	// Initialize file manager
	if err := e.initialize(inputFile); err != nil {
		return "", err
	}

	// Check cache
	if cachedText, found := e.checkCache(); found {
		return cachedText, nil
	}

	// Select OCR engine
	engine, err := e.selectOCREngine()
	if err != nil {
		return "", err
	}

	e.logger.ProgressAlways("🔍", "Using OCR engine: %s", engine.Name())

	// Get file information
	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to get file info")
	}

	// Process based on file type
	var text string
	if fileInfo.Extension == "pdf" {
		text, err = e.processPDF(ctx, inputFile, engine)
	} else if utils.IsImageFile(fileInfo.Extension) {
		text, err = e.processImage(ctx, inputFile, engine)
	} else {
		return "", fmt.Errorf("unsupported file type for OCR: %s", fileInfo.Extension)
	}

	if err != nil {
		return "", err
	}

	// Validate text
	if len(text) < e.config.MinTextThreshold {
		return "", fmt.Errorf("extracted text too short (%d characters)", len(text))
	}

	// Save cache
	e.saveCache(text)

	return text, nil
}

// initialize initializes the OCR extractor
func (e *OCRExtractor) initialize(inputFile string) error {
	if e.fileManager != nil {
		return nil
	}

	fileInfo, err := utils.GetFileInfo(inputFile)
	if err != nil {
		return err
	}

	e.fileManager = utils.NewFileManager(inputFile, fileInfo.MD5Hash, e.logger)
	return e.fileManager.EnsureBaseDir()
}

// checkCache checks for cached results
func (e *OCRExtractor) checkCache() (string, bool) {
	if !e.config.SkipExisting || e.fileManager == nil {
		return "", false
	}

	textFilePath := e.fileManager.GetTextFilePath()
	if content, err := os.ReadFile(textFilePath); err == nil {
		e.logger.Progress("📄", "Loading cached OCR results from: %s", textFilePath)
		return string(content), true
	}

	return "", false
}

// saveCache saves results to cache
func (e *OCRExtractor) saveCache(text string) {
	if e.fileManager == nil {
		return
	}

	textFilePath := e.fileManager.GetTextFilePath()
	if err := os.WriteFile(textFilePath, []byte(text), 0644); err != nil {
		e.logger.Warn("Failed to save cache: %v", err)
	}
}

// processPDF processes PDF files using page-by-page approach
func (e *OCRExtractor) processPDF(ctx context.Context, inputFile string, engine interfaces.OCREngine) (string, error) {
	// Use page-by-page processing for better progress display, caching and error handling
	e.logger.Progress("📄", "Using page-by-page processing for better control")
	return e.processPDFByPages(ctx, inputFile, engine)
}

// processImage processes image files
func (e *OCRExtractor) processImage(ctx context.Context, inputFile string, engine interfaces.OCREngine) (string, error) {
	return engine.ExtractTextFromImage(ctx, inputFile)
}

// processPDFByPages processes PDF page by page (simplified sequential version)
func (e *OCRExtractor) processPDFByPages(ctx context.Context, inputFile string, engine interfaces.OCREngine) (string, error) {
	// Create pages directory
	pagesDir := e.fileManager.GetPagesDir()
	if err := utils.EnsureDir(pagesDir); err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to create pages directory")
	}

	e.logger.Progress("📂", "Created pages directory: %s", pagesDir)

	// Check for existing split page files
	existingPageCount := e.countPages(pagesDir)
	var totalPages int
	var err error

	if existingPageCount > 0 {
		e.logger.Progress("⏭️", "Found %d existing page files, resuming from there", existingPageCount)
		totalPages = existingPageCount
	} else {
		// Split PDF into individual pages
		e.logger.ProgressAlways("✂️", "Splitting PDF into individual pages...")
		totalPages, err = e.splitPDFIntoPages(ctx, inputFile, pagesDir)
		if err != nil {
			return "", utils.WrapError(err, utils.ErrorTypeOCR, "failed to split PDF into pages")
		}
		e.logger.ProgressAlways("✅", "Successfully split PDF into %d pages", totalPages)
	}

	if totalPages == 0 {
		return "", utils.NewOCRError("no pages found in PDF", nil)
	}

	e.logger.ProgressAlways("🔄", "Processing %d pages with OCR engine: %s", totalPages, engine.Name())

	// Process each page sequentially
	var allText strings.Builder
	var errors []error
	successCount := 0

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		// Only show detailed page processing in verbose mode
		e.logger.Progress("📄", "Processing page %d/%d", pageNum, totalPages)

		pageText, err := e.processPageWithProgress(ctx, pageNum, totalPages, engine)
		if err != nil {
			e.logger.Warn("Failed to process page %d: %v", pageNum, err)
			errors = append(errors, fmt.Errorf("page %d failed: %w", pageNum, err))
			continue
		}

		if pageText != "" {
			allText.WriteString(fmt.Sprintf("--- Page %d ---\n", pageNum))
			allText.WriteString(pageText)
			allText.WriteString("\n\n")
			successCount++
		}

		// Show progress every 10 pages or at milestones
		if pageNum%10 == 0 || pageNum == totalPages || pageNum == 1 {
			e.logger.ProgressAlways("📈", "Pages completed: %d/%d (%.1f%%)",
				pageNum, totalPages, float64(pageNum)/float64(totalPages)*100)
		} else {
			e.logger.Progress("📈", "Pages completed: %d/%d (%.1f%%)",
				pageNum, totalPages, float64(pageNum)/float64(totalPages)*100)
		}
	}

	e.logger.ProgressAlways("📊", "Processing completed: %d/%d pages successful", successCount, totalPages)

	finalText := strings.TrimSpace(allText.String())
	if finalText == "" {
		return "", utils.NewOCRError("no text extracted from any page", nil)
	}

	e.logger.ProgressAlways("✨", "Text extraction completed, total length: %d characters", len(finalText))

	return finalText, nil
}

// processPageWithProgress processes a single page with progress tracking
func (e *OCRExtractor) processPageWithProgress(ctx context.Context, pageNum, totalPages int, engine interfaces.OCREngine) (string, error) {
	// Check for cached page text first
	pageTextPath := e.fileManager.GetPageTextPath(pageNum)
	if content, err := os.ReadFile(pageTextPath); err == nil {
		e.logger.Progress("⏭️", "Loaded cached text for page %d/%d", pageNum, totalPages)
		return string(content), nil
	}

	// Get page PDF path
	pagePDFPath := e.fileManager.GetPagePDFPath(pageNum)
	if _, err := os.Stat(pagePDFPath); os.IsNotExist(err) {
		return "", fmt.Errorf("page PDF not found: %s", pagePDFPath)
	}

	var text string
	var err error

	// Check if engine supports direct PDF processing
	if engine.SupportsDirectPDF() {
		text, err = engine.ExtractTextFromPDF(ctx, pagePDFPath)
	} else {
		// Convert PDF page to image first
		pageImagePath := e.fileManager.GetPageImagePath(pageNum)
		e.logger.Progress("🖼️", "Converting page %d/%d to image", pageNum, totalPages)

		if err := e.convertPDFToImage(ctx, pagePDFPath, pageImagePath); err != nil {
			return "", utils.WrapError(err, utils.ErrorTypeConversion,
				fmt.Sprintf("failed to convert page %d to image", pageNum))
		}

		text, err = engine.ExtractTextFromImage(ctx, pageImagePath)
	}

	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeOCR,
			fmt.Sprintf("failed to extract text from page %d", pageNum))
	}

	// Save page text to cache
	if text != "" {
		if writeErr := os.WriteFile(pageTextPath, []byte(text), 0644); writeErr != nil {
			e.logger.Warn("Failed to cache page %d text: %v", pageNum, writeErr)
		}
	}

	e.logger.Progress("✅", "Completed page %d/%d, extracted %d characters", pageNum, totalPages, len(text))

	return text, nil
}

// splitPDFIntoPages splits a PDF into individual page files
func (e *OCRExtractor) splitPDFIntoPages(ctx context.Context, inputFile, outputDir string) (int, error) {
	gsPath, err := e.findGhostscriptPath()
	if err != nil {
		return 0, utils.WrapError(err, utils.ErrorTypeSystem, "Ghostscript not found")
	}

	// Use Ghostscript to split PDF into pages
	cmd := exec.CommandContext(ctx, gsPath,
		"-sDEVICE=pdfwrite",
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		fmt.Sprintf("-sOutputFile=%s", filepath.Join(outputDir, constants.PDFPageFilePattern)),
		inputFile)

	if err := cmd.Run(); err != nil {
		return 0, utils.WrapError(err, utils.ErrorTypeConversion, "failed to split PDF with Ghostscript")
	}

	// Count the generated pages
	pageCount := e.countPages(outputDir)
	return pageCount, nil
}

// convertPDFToImage converts a PDF page to an image
func (e *OCRExtractor) convertPDFToImage(ctx context.Context, pdfPath, imagePath string) error {
	gsPath, err := e.findGhostscriptPath()
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeSystem, "Ghostscript not found")
	}

	cmd := exec.CommandContext(ctx, gsPath,
		"-sDEVICE=png16m",
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		"-r300", // 300 DPI for good quality
		fmt.Sprintf("-sOutputFile=%s", imagePath),
		pdfPath)

	if err := cmd.Run(); err != nil {
		return utils.WrapError(err, utils.ErrorTypeConversion,
			fmt.Sprintf("failed to convert PDF to image: %s", pdfPath))
	}

	return nil
}

// countPages counts existing page files in a directory
func (e *OCRExtractor) countPages(dir string) int {
	count := 0
	for i := 1; i <= 10000; i++ { // Reasonable upper limit
		pageFile := filepath.Join(dir, fmt.Sprintf(constants.PDFPageFilePattern, i))
		if _, err := os.Stat(pageFile); err == nil {
			count++
		} else {
			break
		}
	}
	return count
}

// findGhostscriptPath finds the Ghostscript executable
func (e *OCRExtractor) findGhostscriptPath() (string, error) {
	platformConfig := constants.GetPlatformConfig()
	for _, path := range platformConfig.GhostscriptPaths {
		if utils.IsCommandAvailable(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("Ghostscript not found. Please install Ghostscript")
}

// selectOCREngine selects an appropriate OCR engine
func (e *OCRExtractor) selectOCREngine() (interfaces.OCREngine, error) {
	// If strategy is set, use it directly
	if e.config.OCRStrategy != types.OCRStrategyInteractive {
		return e.createOCREngine(e.config.OCRStrategy)
	}

	// Interactive selection
	strategy, err := e.promptUserSelection()
	if err != nil {
		return nil, err
	}

	return e.createOCREngine(strategy)
}

// promptUserSelection prompts user to select OCR strategy
func (e *OCRExtractor) promptUserSelection() (types.OCRStrategy, error) {
	fmt.Printf("\n🔍 OCR Tool Selection\n")
	fmt.Println("===================")

	strategies := e.getAvailableStrategies()
	if len(strategies) == 0 {
		return "", fmt.Errorf("no OCR tools available")
	}

	fmt.Printf("Available OCR tools:\n")
	for i, strategy := range strategies {
		engine, err := e.createOCREngine(strategy)
		if err == nil {
			fmt.Printf("  %d. %s - %s\n", i+1, engine.Name(), engine.GetDescription())
		}
	}

	fmt.Printf("\nSelect OCR tool (1-%d): ", len(strategies))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(strategies) {
		return "", fmt.Errorf("invalid selection: %s", input)
	}

	selected := strategies[selection-1]
	fmt.Printf("✅ Selected: %s\n", selected)

	// Handle LLM Caller template selection
	if selected == types.OCRStrategyLLMCaller {
		template, err := e.promptForLLMTemplate(reader)
		if err != nil {
			return "", err
		}
		e.config.LLMTemplate = template
	}

	return selected, nil
}

// promptForLLMTemplate prompts for LLM template
func (e *OCRExtractor) promptForLLMTemplate(reader *bufio.Reader) (string, error) {
	fmt.Printf("\nLLM Template (e.g., qwen-vl-ocr): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read template input: %w", err)
	}

	template := strings.TrimSpace(input)
	if template == "" {
		return "", fmt.Errorf("LLM template cannot be empty")
	}

	return template, nil
}

// getAvailableStrategies returns list of available OCR strategies
func (e *OCRExtractor) getAvailableStrategies() []types.OCRStrategy {
	var strategies []types.OCRStrategy

	if e.isLLMCallerAvailable() {
		strategies = append(strategies, types.OCRStrategyLLMCaller)
	}
	if e.isSuryaOCRAvailable() {
		strategies = append(strategies, types.OCRStrategySuryaOCR)
	}

	return strategies
}

// createOCREngine creates an OCR engine based on strategy
func (e *OCRExtractor) createOCREngine(strategy types.OCRStrategy) (interfaces.OCREngine, error) {
	switch strategy {
	case types.OCRStrategyLLMCaller:
		return NewLLMCallerEngine(e.config, e.logger, e.fileManager), nil
	case types.OCRStrategySuryaOCR:
		return NewSuryaOCREngine(e.config, e.logger, e.fileManager), nil
	default:
		return nil, fmt.Errorf("unsupported OCR strategy: %s", strategy)
	}
}

// isLLMCallerAvailable checks if LLM Caller is available
func (e *OCRExtractor) isLLMCallerAvailable() bool {
	return utils.IsCommandAvailable("llm-caller")
}

// isSuryaOCRAvailable checks if Surya OCR is available
func (e *OCRExtractor) isSuryaOCRAvailable() bool {
	return utils.IsCommandAvailable("surya_ocr")
}

// SupportsFile checks if this extractor supports the file type
func (e *OCRExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	return fileInfo.Extension == "pdf" || utils.IsImageFile(fileInfo.Extension)
}

// Name returns the extractor name
// Name 返回提取器名称
func (e *OCRExtractor) Name() string {
	return e.name
}
