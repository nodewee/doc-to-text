package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// SuryaOCREngine uses surya_ocr for text extraction
type SuryaOCREngine struct {
	config              *config.Config
	logger              *logger.Logger
	intermediateManager interfaces.IntermediateFileManager
}

// SuryaOCRResult represents the structure of surya_ocr JSON output
type SuryaOCRResult map[string][]SuryaPageResult

type SuryaPageResult struct {
	TextLines []SuryaTextLine `json:"text_lines"`
	Languages interface{}     `json:"languages"`
	ImageBbox []float64       `json:"image_bbox"`
	Page      int             `json:"page"`
}

type SuryaTextLine struct {
	Text       string      `json:"text"`
	Confidence float64     `json:"confidence"`
	Polygon    [][]float64 `json:"polygon"`
	Bbox       []float64   `json:"bbox"`
}

// NewSuryaOCREngine creates a new Surya OCR engine
func NewSuryaOCREngine(cfg *config.Config, log *logger.Logger) interfaces.OCREngine {
	return &SuryaOCREngine{
		config: cfg,
		logger: log,
	}
}

// Name returns the name of the OCR tool
func (e *SuryaOCREngine) Name() string {
	return "surya_ocr"
}

// GetDescription returns a description of the OCR tool
func (e *SuryaOCREngine) GetDescription() string {
	return "Surya OCR (local OCR tool)"
}

// SupportsDirectPDF returns true since we now want to process single-page PDFs directly
func (e *SuryaOCREngine) SupportsDirectPDF() bool {
	return true
}

// IsAvailable checks if the OCR tool is available on the system
func (e *SuryaOCREngine) IsAvailable() bool {
	_, err := exec.LookPath(e.config.SuryaOCRPath)
	return err == nil
}

// SetIntermediateManager sets the intermediate file manager for persistent storage
func (e *SuryaOCREngine) SetIntermediateManager(intermediateManager interfaces.IntermediateFileManager) {
	e.intermediateManager = intermediateManager
}

// ExtractTextFromPDF extracts text from PDF using Surya OCR
func (e *SuryaOCREngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
	e.logger.Debug("Starting Surya OCR extraction from PDF: %s", pdfPath)

	// Ensure intermediate manager is available
	if e.intermediateManager == nil {
		return "", fmt.Errorf("intermediate manager not initialized")
	}

	// Create intermediate output directory for Surya OCR results
	outputDir, err := e.intermediateManager.CreateIntermediateDir("surya_ocr_results")
	if err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	e.logger.Debug("Created intermediate output directory: %s", outputDir)

	// Check if OCR results already exist (resume capability)
	sanitizedFileName := utils.SanitizeFileName(filepath.Base(pdfPath))
	resultFile := filepath.Join(outputDir, fmt.Sprintf("%s_ocr.txt", sanitizedFileName))
	resultFile = utils.NormalizePath(resultFile)

	if content, err := e.loadExistingOCRResults(resultFile); err == nil {
		e.logger.Info("⏭️ Loading cached OCR results: %s", resultFile)
		return content, nil
	}

	// Run surya_ocr command with normalized paths
	normalizedPDFPath := utils.NormalizePath(pdfPath)
	normalizedOutputDir := utils.NormalizePath(outputDir)

	cmd := exec.CommandContext(ctx, e.config.SuryaOCRPath,
		"--output_dir", normalizedOutputDir,
		normalizedPDFPath)

	e.logger.Debug("Running Surya OCR command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("Surya OCR command failed: %s", string(output))
		return "", fmt.Errorf("surya OCR extraction failed: %w", err)
	}

	e.logger.Debug("Surya OCR command completed successfully")

	// Parse the JSON output and cache the result
	text, err := e.parseOCRResults(normalizedOutputDir, sanitizedFileName)
	if err != nil {
		return "", fmt.Errorf("failed to parse OCR results: %w", err)
	}

	// Save the extracted text for future use
	if saveErr := os.WriteFile(resultFile, []byte(text), 0644); saveErr != nil {
		e.logger.Warn("Failed to save OCR results cache: %v", saveErr)
	}

	e.logger.Debug("Extracted %d characters using Surya OCR", len(text))

	return text, nil
}

// ExtractTextFromImage extracts text from image using Surya OCR
func (e *SuryaOCREngine) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	e.logger.Debug("Starting Surya OCR extraction from image: %s", imagePath)

	// Ensure intermediate manager is available
	if e.intermediateManager == nil {
		return "", fmt.Errorf("intermediate manager not initialized")
	}

	// Create intermediate output directory for Surya OCR results
	outputDir, err := e.intermediateManager.CreateIntermediateDir("surya_ocr_image_results")
	if err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	e.logger.Debug("Created intermediate output directory: %s", outputDir)

	// Check if OCR results already exist (resume capability)
	sanitizedFileName := utils.SanitizeFileName(filepath.Base(imagePath))
	resultFile := filepath.Join(outputDir, fmt.Sprintf("%s_ocr.txt", sanitizedFileName))
	resultFile = utils.NormalizePath(resultFile)

	if content, err := e.loadExistingOCRResults(resultFile); err == nil {
		e.logger.Info("⏭️ Loading cached OCR results: %s", resultFile)
		return content, nil
	}

	// Run surya_ocr command for image with normalized paths
	normalizedImagePath := utils.NormalizePath(imagePath)
	normalizedOutputDir := utils.NormalizePath(outputDir)

	cmd := exec.CommandContext(ctx, e.config.SuryaOCRPath,
		"--output_dir", normalizedOutputDir,
		normalizedImagePath)

	e.logger.Debug("Running Surya OCR command for image: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.logger.Error("Surya OCR image command failed: %s", string(output))
		return "", fmt.Errorf("surya OCR image extraction failed: %w", err)
	}

	e.logger.Debug("Surya OCR image command completed successfully")

	// Parse the JSON output and cache the result
	text, err := e.parseOCRResults(normalizedOutputDir, sanitizedFileName)
	if err != nil {
		return "", fmt.Errorf("failed to parse OCR results: %w", err)
	}

	// Save the extracted text for future use
	if saveErr := os.WriteFile(resultFile, []byte(text), 0644); saveErr != nil {
		e.logger.Warn("Failed to save OCR results cache: %v", saveErr)
	}

	e.logger.Debug("Extracted %d characters from image using Surya OCR", len(text))

	return text, nil
}

// loadExistingOCRResults loads existing OCR results if available
func (e *SuryaOCREngine) loadExistingOCRResults(resultFile string) (string, error) {
	if _, err := os.Stat(resultFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no cached results found")
	}

	content, err := os.ReadFile(resultFile)
	if err != nil {
		return "", fmt.Errorf("failed to read cached results: %w", err)
	}

	return string(content), nil
}

// parseOCRResults parses Surya OCR's JSON output and extracts text
func (e *SuryaOCREngine) parseOCRResults(outputDir, fileName string) (string, error) {
	// Look for the results.json file in the subdirectory named after the input file
	// Remove file extension from fileName for the subdirectory name
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	resultsPath := filepath.Join(outputDir, baseName, "results.json")
	resultsPath = utils.NormalizePath(resultsPath)

	// Check if results file exists
	if _, err := os.Stat(resultsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("no results.json found at %s", resultsPath)
	}

	// Read and parse JSON
	content, err := os.ReadFile(resultsPath)
	if err != nil {
		return "", fmt.Errorf("error reading results.json: %w", err)
	}

	var results SuryaOCRResult
	err = json.Unmarshal(content, &results)
	if err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	// Extract text from all pages and text lines
	var textBuilder strings.Builder

	// Iterate through all file results
	for fileName, pages := range results {
		e.logger.Debug("Processing OCR results for file: %s", fileName)

		for _, page := range pages {
			// Extract text from all text lines on this page
			for _, textLine := range page.TextLines {
				if strings.TrimSpace(textLine.Text) != "" {
					textBuilder.WriteString(textLine.Text)
					textBuilder.WriteString("\n")
				}
			}
		}
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if extractedText == "" {
		return "", fmt.Errorf("no text found in OCR results")
	}

	return extractedText, nil
}
