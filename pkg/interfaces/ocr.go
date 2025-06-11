package interfaces

import (
	"context"

	"github.com/nodewee/doc-to-text/pkg/types"
)

// OCREngine defines the interface for different OCR implementations
type OCREngine interface {
	// Name returns the name of the OCR tool
	Name() string

	// ExtractTextFromImage extracts text from an image file
	ExtractTextFromImage(ctx context.Context, imagePath string) (string, error)

	// ExtractTextFromPDF extracts text directly from a PDF file (if supported)
	ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error)

	// SupportsDirectPDF returns true if the tool can process PDF files directly
	SupportsDirectPDF() bool

	// GetDescription returns a description of the OCR tool
	GetDescription() string
}

// OCRSelector handles the selection of OCR tool
type OCRSelector interface {
	// SelectOCRStrategy selects an OCR tool, either from config or interactively
	SelectOCRStrategy(strategy types.OCRStrategy) (OCREngine, error)

	// GetAvailableStrategies returns all available OCR strategies
	GetAvailableStrategies() []types.OCRStrategy

	// PromptUserSelection prompts user to select an OCR tool interactively
	PromptUserSelection() (types.OCRStrategy, error)
}
