package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/utils"
)

// === LLM Caller 引擎 ===

// LLMCallerEngine 使用LLM Caller进行OCR识别
type LLMCallerEngine struct {
	config      *config.Config
	logger      *logger.Logger
	fileManager *utils.FileManager
}

// NewLLMCallerEngine 创建LLM Caller引擎
func NewLLMCallerEngine(cfg *config.Config, log *logger.Logger, fm *utils.FileManager) interfaces.OCREngine {
	return &LLMCallerEngine{
		config:      cfg,
		logger:      log,
		fileManager: fm,
	}
}

func (e *LLMCallerEngine) Name() string {
	return "LLM Caller"
}

func (e *LLMCallerEngine) GetDescription() string {
	return "LLM-based OCR engine using external AI models"
}

func (e *LLMCallerEngine) SupportsDirectPDF() bool {
	return true
}

func (e *LLMCallerEngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
	// 检查缓存
	if e.fileManager != nil {
		outputPath := e.fileManager.GetOCRDataPath()
		if content, err := os.ReadFile(outputPath); err == nil {
			e.logger.Progress("⏭️", "Loading cached LLM results")
			return string(content), nil
		}
	}

	llmCallerPath, err := e.findLLMCallerPath()
	if err != nil {
		return "", err
	}

	// 创建输出目录
	outputDir, err := e.fileManager.CreateIntermediateDir("llm_caller_results")
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 确定模板
	template := e.config.LLMTemplate
	if template == "" {
		template = "analyze"
	}

	// 执行LLM Caller
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_output.txt", utils.SanitizeFileName(filepath.Base(pdfPath))))
	cmd := exec.CommandContext(ctx, llmCallerPath,
		"call", template,
		"--var", fmt.Sprintf("file:file:%s", pdfPath),
		"-o", outputFile)

	// 捕获标准错误输出
	var stderrBuilder strings.Builder
	cmd.Stderr = &stderrBuilder

	if err := cmd.Run(); err != nil {
		stderrOutput := strings.TrimSpace(stderrBuilder.String())
		if stderrOutput != "" {
			return "", fmt.Errorf("LLM caller execution failed (%v) with stderr: %s", err, stderrOutput)
		}
		return "", fmt.Errorf("LLM caller execution failed: %w", err)
	}

	// 读取结果
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM results: %w", err)
	}

	text := strings.TrimSpace(string(content))

	// 保存缓存
	if e.fileManager != nil {
		os.WriteFile(e.fileManager.GetOCRDataPath(), []byte(text), 0644)
	}

	return text, nil
}

func (e *LLMCallerEngine) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	// 检查缓存
	if e.fileManager != nil {
		outputPath := e.fileManager.GetOCRDataPath()
		if content, err := os.ReadFile(outputPath); err == nil {
			e.logger.Progress("⏭️", "Loading cached LLM results")
			return string(content), nil
		}
	}

	llmCallerPath, err := e.findLLMCallerPath()
	if err != nil {
		return "", err
	}

	// 创建输出目录
	outputDir, err := e.fileManager.CreateIntermediateDir("llm_caller_image_results")
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 读取图像并转换为base64
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	imageFormat := strings.TrimPrefix(filepath.Ext(imagePath), ".")
	if imageFormat == "" {
		imageFormat = "png"
	}

	base64String := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/%s;base64,%s", imageFormat, base64String)

	// 确定模板
	template := e.config.LLMTemplate
	if template == "" {
		template = "image-to-text"
	}

	// 执行LLM Caller
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_output.txt", utils.SanitizeFileName(filepath.Base(imagePath))))
	cmd := exec.CommandContext(ctx, llmCallerPath,
		"call", template,
		"--var", fmt.Sprintf("image_url:text:%s", dataURL),
		"-o", outputFile)

	// 捕获标准错误输出
	var stderrBuilder strings.Builder
	cmd.Stderr = &stderrBuilder

	if err := cmd.Run(); err != nil {
		stderrOutput := strings.TrimSpace(stderrBuilder.String())
		if stderrOutput != "" {
			return "", fmt.Errorf("LLM caller execution failed (%v) with stderr: %s", err, stderrOutput)
		}
		return "", fmt.Errorf("LLM caller execution failed: %w", err)
	}

	// 读取结果
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM results: %w", err)
	}

	text := strings.TrimSpace(string(content))

	// 保存缓存
	if e.fileManager != nil {
		os.WriteFile(e.fileManager.GetOCRDataPath(), []byte(text), 0644)
	}

	return text, nil
}

// findLLMCallerPath 查找LLM Caller路径
func (e *LLMCallerEngine) findLLMCallerPath() (string, error) {
	// Try to find llm-caller using shell detection
	if utils.IsCommandAvailable("llm-caller") {
		return "llm-caller", nil
	}

	// Common installation paths
	commonPaths := []string{
		"llm-caller",
		"/usr/local/bin/llm-caller",
		"/usr/bin/llm-caller",
		"/opt/homebrew/bin/llm-caller",
	}

	for _, path := range commonPaths {
		if utils.IsCommandAvailable(path) {
			e.logger.Debug("Found llm-caller at: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("LLM Caller not found. Please install llm-caller")
}

// === Surya OCR 引擎 ===

// SuryaOCREngine 使用Surya OCR进行文本识别
type SuryaOCREngine struct {
	config      *config.Config
	logger      *logger.Logger
	fileManager *utils.FileManager
}

// SuryaOCRResult Surya OCR结果结构
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

// NewSuryaOCREngine 创建Surya OCR引擎
func NewSuryaOCREngine(cfg *config.Config, log *logger.Logger, fm *utils.FileManager) interfaces.OCREngine {
	return &SuryaOCREngine{
		config:      cfg,
		logger:      log,
		fileManager: fm,
	}
}

func (e *SuryaOCREngine) Name() string {
	return "Surya OCR"
}

func (e *SuryaOCREngine) GetDescription() string {
	return "Surya OCR engine for multilingual text recognition"
}

func (e *SuryaOCREngine) SupportsDirectPDF() bool {
	return true
}

func (e *SuryaOCREngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
	// 检查缓存
	if e.fileManager != nil {
		outputPath := e.fileManager.GetOCRDataPath()
		if content, err := os.ReadFile(outputPath); err == nil {
			e.logger.Progress("⏭️", "Loading cached Surya results")
			return string(content), nil
		}
	}

	suryaPath, err := e.findSuryaOCRPath()
	if err != nil {
		return "", err
	}

	// 创建输出目录
	outputDir, err := e.fileManager.CreateIntermediateDir("surya_ocr_results")
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 执行Surya OCR
	cmd := exec.CommandContext(ctx, suryaPath, pdfPath, "--output_dir", outputDir)

	// 捕获标准错误输出和标准输出
	var stderrBuilder strings.Builder
	var stdoutBuilder strings.Builder
	cmd.Stderr = &stderrBuilder
	cmd.Stdout = &stdoutBuilder

	if err := cmd.Run(); err != nil {
		stderrOutput := strings.TrimSpace(stderrBuilder.String())
		stdoutOutput := strings.TrimSpace(stdoutBuilder.String())

		errorMsg := fmt.Sprintf("Surya execution failed: %v", err)
		if stderrOutput != "" {
			errorMsg += "\nStderr: " + stderrOutput
		}
		if stdoutOutput != "" {
			errorMsg += "\nStdout: " + stdoutOutput
		}

		return "", fmt.Errorf(errorMsg)
	}

	// 解析结果
	text, err := e.parseOCRResults(outputDir, utils.SanitizeFileName(filepath.Base(pdfPath)))
	if err != nil {
		return "", fmt.Errorf("failed to parse Surya results: %w", err)
	}

	// 保存缓存
	if e.fileManager != nil {
		os.WriteFile(e.fileManager.GetOCRDataPath(), []byte(text), 0644)
	}

	return text, nil
}

func (e *SuryaOCREngine) ExtractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	// 检查缓存
	if e.fileManager != nil {
		outputPath := e.fileManager.GetOCRDataPath()
		if content, err := os.ReadFile(outputPath); err == nil {
			e.logger.Progress("⏭️", "Loading cached Surya results")
			return string(content), nil
		}
	}

	suryaPath, err := e.findSuryaOCRPath()
	if err != nil {
		return "", err
	}

	// 创建输出目录
	outputDir, err := e.fileManager.CreateIntermediateDir("")
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 执行Surya OCR
	cmd := exec.CommandContext(ctx, suryaPath, imagePath, "--output_dir", outputDir)

	// 捕获标准错误输出和标准输出
	var stderrBuilder strings.Builder
	var stdoutBuilder strings.Builder
	cmd.Stderr = &stderrBuilder
	cmd.Stdout = &stdoutBuilder

	if err := cmd.Run(); err != nil {
		stderrOutput := strings.TrimSpace(stderrBuilder.String())
		stdoutOutput := strings.TrimSpace(stdoutBuilder.String())

		errorMsg := fmt.Sprintf("Surya execution failed: %v", err)
		if stderrOutput != "" {
			errorMsg += "\nStderr: " + stderrOutput
		}
		if stdoutOutput != "" {
			errorMsg += "\nStdout: " + stdoutOutput
		}

		return "", fmt.Errorf(errorMsg)
	}

	// 解析结果
	text, err := e.parseOCRResults(outputDir, utils.SanitizeFileName(filepath.Base(imagePath)))
	if err != nil {
		return "", fmt.Errorf("failed to parse Surya results: %w", err)
	}

	// 保存缓存
	if e.fileManager != nil {
		os.WriteFile(e.fileManager.GetOCRDataPath(), []byte(text), 0644)
	}

	return text, nil
}

func (e *SuryaOCREngine) findSuryaOCRPath() (string, error) {
	// Try to find surya_ocr using shell detection
	if utils.IsCommandAvailable("surya_ocr") {
		return "surya_ocr", nil
	}

	// Common installation paths
	commonPaths := []string{
		"surya_ocr",
		"/usr/local/bin/surya_ocr",
		"/usr/bin/surya_ocr",
		"/opt/homebrew/bin/surya_ocr",
	}

	for _, path := range commonPaths {
		if utils.IsCommandAvailable(path) {
			e.logger.Debug("Found surya_ocr at: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("Surya OCR not found. Please install with: pip install surya-ocr")
}

func (e *SuryaOCREngine) parseOCRResults(outputDir, fileName string) (string, error) {
	// 查找JSON结果文件
	// sub dir name is image name (no extension)
	subDirName := strings.TrimSuffix(utils.SanitizeFileName(filepath.Base(fileName)), filepath.Ext(fileName))
	jsonFile := filepath.Join(outputDir, subDirName, "results.json")
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		// 尝试其他可能的文件名
		files, err := os.ReadDir(outputDir)
		if err != nil {
			return "", err
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				jsonFile = filepath.Join(outputDir, file.Name())
				break
			}
		}
	}

	// 读取并解析JSON
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return "", fmt.Errorf("failed to read JSON results: %w", err)
	}

	var results SuryaOCRResult
	if err := json.Unmarshal(data, &results); err != nil {
		return "", fmt.Errorf("failed to parse JSON results: %w", err)
	}

	// 提取文本
	var allText strings.Builder
	for _, pages := range results {
		for _, page := range pages {
			for _, line := range page.TextLines {
				if line.Text != "" {
					allText.WriteString(line.Text)
					allText.WriteString("\n")
				}
			}
		}
	}

	return allText.String(), nil
}
