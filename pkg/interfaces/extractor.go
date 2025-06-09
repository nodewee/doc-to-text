package interfaces

import (
	"context"

	"github.com/nodewee/doc-to-text/pkg/types"
)

// Extractor defines the interface for text extraction
type Extractor interface {
	// Extract extracts text from the given file
	Extract(ctx context.Context, inputFile string) (string, error)

	// SupportsFile checks if this extractor supports the given file type
	SupportsFile(fileInfo *types.FileInfo) bool

	// Name returns the name of the extractor
	Name() string
}

// OCRExtractor defines the interface for OCR-based extractors
type OCRExtractor interface {
	Extractor

	// ExtractWithOCR performs OCR extraction and returns structured data
	ExtractWithOCR(ctx context.Context, inputFile string) (map[string]interface{}, error)
}

// CacheableExtractor defines interface for extractors that support caching
type CacheableExtractor interface {
	Extractor

	// GetCacheKey returns a unique cache key for the file
	GetCacheKey(fileInfo *types.FileInfo) string
}

// ExtractorFactory creates extractors based on file type
type ExtractorFactory interface {
	// CreateExtractor creates an appropriate extractor for the file
	CreateExtractor(fileInfo *types.FileInfo) (Extractor, error)

	// CreateExtractorWithFallbacks creates extractors with fallback options
	CreateExtractorWithFallbacks(fileInfo *types.FileInfo) ([]Extractor, error)

	// RegisterExtractor registers a new extractor
	RegisterExtractor(name string, extractor Extractor)

	// ListExtractors returns all registered extractors
	ListExtractors() []string
}

// ExtractorMiddleware defines middleware for extraction process
type ExtractorMiddleware interface {
	// Process wraps the extraction process
	Process(next ExtractorFunc) ExtractorFunc
}

// ExtractorFunc is a function type for extraction
type ExtractorFunc func(ctx context.Context, inputFile string) (string, error)

// ExtractionResult holds the result of text extraction
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

// FileProcessor handles the overall file processing workflow
type FileProcessor interface {
	// ProcessFile processes a file and returns extracted text
	ProcessFile(ctx context.Context, inputFile, outputFile string) (*ExtractionResult, error)

	// SetOutputPath sets custom output path
	SetOutputPath(outputPath string)

	// SetExtractorFactory sets the extractor factory to use
	SetExtractorFactory(factory ExtractorFactory)
}
