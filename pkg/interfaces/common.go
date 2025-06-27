package interfaces

import (
	"context"

	"doc-to-text/pkg/types"
)

// === 核心接口 ===

// Extractor 文本提取器接口
type Extractor interface {
	// Extract 从文件中提取文本
	Extract(ctx context.Context, inputFile string) (string, error)
	// SupportsFile 检查是否支持该文件类型
	SupportsFile(fileInfo *types.FileInfo) bool
	// Name 返回提取器名称
	Name() string
}

// ExtractorFactory 提取器工厂接口
type ExtractorFactory interface {
	// CreateExtractorWithFallbacks 创建带备选的提取器链
	CreateExtractorWithFallbacks(fileInfo *types.FileInfo) ([]Extractor, error)
	// RegisterExtractor 注册新的提取器
	RegisterExtractor(name string, extractor Extractor)
}

// FileProcessor 文件处理器接口
type FileProcessor interface {
	// ProcessFile 处理文件并返回提取结果
	ProcessFile(ctx context.Context, inputFile, outputFile string) (*ExtractionResult, error)
}

// === OCR相关接口 ===

// OCREngine OCR引擎接口
type OCREngine interface {
	// Name 返回OCR工具名称
	Name() string
	// ExtractTextFromImage 从图像提取文本
	ExtractTextFromImage(ctx context.Context, imagePath string) (string, error)
	// ExtractTextFromPDF 从PDF提取文本
	ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error)
	// SupportsDirectPDF 是否支持直接处理PDF
	SupportsDirectPDF() bool
	// GetDescription 返回工具描述
	GetDescription() string
}

// === 数据结构 ===

// ExtractionResult 提取结果
type ExtractionResult struct {
	Text                string                 `json:"text"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Source              string                 `json:"source"`
	ExtractorUsed       string                 `json:"extractor_used"`
	ProcessTime         int64                  `json:"process_time_ms"`
	Error               string                 `json:"error,omitempty"`
	FallbackUsed        bool                   `json:"fallback_used,omitempty"`
	AttemptedExtractors []string               `json:"attempted_extractors,omitempty"`
}
