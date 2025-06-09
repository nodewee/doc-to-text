package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// TextFileExtractor handles plain text files
type TextFileExtractor struct {
	name string
}

// NewTextFileExtractor creates a new text file extractor
func NewTextFileExtractor() interfaces.Extractor {
	return &TextFileExtractor{
		name: "text-file",
	}
}

// Extract extracts text from plain text files
func (e *TextFileExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	content, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("error reading text file: %w", err)
	}

	return string(content), nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *TextFileExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	return utils.IsTextFile(fileInfo.Extension, fileInfo.MimeType)
}

// Name returns the name of the extractor
func (e *TextFileExtractor) Name() string {
	return e.name
}

// GetCacheKey returns a unique cache key for the file
func (e *TextFileExtractor) GetCacheKey(fileInfo *types.FileInfo) string {
	return fmt.Sprintf("text-%s", fileInfo.MD5Hash)
}
