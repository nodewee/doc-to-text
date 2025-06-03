package core

import (
	"fmt"
	"strings"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/ocr"
	"doc-to-text/pkg/providers"
	"doc-to-text/pkg/types"
)

// DefaultExtractorFactory implements ExtractorFactory
type DefaultExtractorFactory struct {
	extractors map[string]interfaces.Extractor
	config     *config.Config
	logger     *logger.Logger
}

// NewExtractorFactory creates a new extractor factory
func NewExtractorFactory(cfg *config.Config, log *logger.Logger) interfaces.ExtractorFactory {
	factory := &DefaultExtractorFactory{
		extractors: make(map[string]interfaces.Extractor),
		config:     cfg,
		logger:     log,
	}

	// Register default extractors
	factory.registerDefaultExtractors()

	return factory
}

// CreateExtractor creates an appropriate extractor for the file
func (f *DefaultExtractorFactory) CreateExtractor(fileInfo *types.FileInfo) (interfaces.Extractor, error) {
	f.logger.Debug("Looking for extractor for file type: %s (MIME: %s)", fileInfo.Extension, fileInfo.MimeType)

	// Try to find a specific extractor that supports this file
	for name, extractor := range f.extractors {
		if extractor.SupportsFile(fileInfo) {
			f.logger.Debug("Selected extractor '%s' for file type %s", extractor.Name(), fileInfo.Extension)
			return extractor, nil
		} else {
			f.logger.Debug("Extractor '%s' does not support file type %s", name, fileInfo.Extension)
		}
	}

	return nil, fmt.Errorf("no suitable extractor found for file type: %s (MIME: %s)",
		fileInfo.Extension, fileInfo.MimeType)
}

// CreateExtractorWithFallbacks creates extractors with fallback options
func (f *DefaultExtractorFactory) CreateExtractorWithFallbacks(fileInfo *types.FileInfo) ([]interfaces.Extractor, error) {
	var extractors []interfaces.Extractor
	ext := strings.ToLower(fileInfo.Extension)

	f.logger.Debug("Creating extractor chain with fallbacks for file type: %s", ext)

	// Define extraction strategy based on file type
	switch {
	case ext == "txt" || ext == "md" || ext == "json" || ext == "xml":
		// Text files - only text extractor
		if textExtractor, exists := f.extractors["text"]; exists {
			extractors = append(extractors, textExtractor)
			f.logger.Debug("Added text extractor for file type: %s", ext)
		}

	case ext == "html" || ext == "htm" || ext == "mhtml" || ext == "mht":
		// HTML files - HTML extractor, then calibre fallback
		if htmlExtractor, exists := f.extractors["html"]; exists {
			extractors = append(extractors, htmlExtractor)
			f.logger.Debug("Added HTML extractor for file type: %s", ext)
		}
		if calibreExtractor, exists := f.extractors["calibre"]; exists {
			extractors = append(extractors, calibreExtractor)
			f.logger.Debug("Added Calibre fallback extractor for file type: %s", ext)
		}

	case ext == "epub" || ext == "mobi":
		// E-books - ebook extractor first
		if ebookExtractor, exists := f.extractors["ebook"]; exists {
			extractors = append(extractors, ebookExtractor)
			f.logger.Debug("Added ebook extractor for file type: %s", ext)
		}

	case ext == "pdf":
		// PDF files - strategy depends on content type
		if f.config.ContentType == types.ContentTypeText {
			// Text content type: try Calibre first, then OCR if failed
			f.logger.Debug("PDF with text content type: trying Calibre first, then OCR fallback")
			if calibreExtractor, exists := f.extractors["calibre"]; exists {
				extractors = append(extractors, calibreExtractor)
				f.logger.Debug("Added Calibre extractor for text-based PDF: %s", ext)
			}
			if ocrExtractor, exists := f.extractors["ocr"]; exists {
				extractors = append(extractors, ocrExtractor)
				f.logger.Debug("Added OCR extractor as fallback for text-based PDF: %s", ext)
			}
		} else {
			// Image content type (default): use OCR directly, no fallback
			f.logger.Debug("PDF with image content type: using OCR directly")
			if ocrExtractor, exists := f.extractors["ocr"]; exists {
				extractors = append(extractors, ocrExtractor)
				f.logger.Debug("Added OCR extractor for image-based PDF: %s", ext)
			}
		}

	case ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "bmp":
		// Images - only OCR
		if ocrExtractor, exists := f.extractors["ocr"]; exists {
			extractors = append(extractors, ocrExtractor)
			f.logger.Debug("Added OCR extractor for file type: %s", ext)
		}

	default:
		// Unknown types - try OCR first, then calibre fallback
		f.logger.Debug("Unknown file type %s, trying fallback extractors", ext)
		if ocrExtractor, exists := f.extractors["ocr"]; exists {
			extractors = append(extractors, ocrExtractor)
			f.logger.Debug("Added OCR extractor as fallback for unknown type: %s", ext)
		}
		if calibreExtractor, exists := f.extractors["calibre"]; exists {
			extractors = append(extractors, calibreExtractor)
			f.logger.Debug("Added Calibre fallback extractor for unknown type: %s", ext)
		}
	}

	if len(extractors) == 0 {
		return nil, fmt.Errorf("no suitable extractors found for file type: %s", ext)
	}

	f.logger.Info("Created extraction chain with %d extractors for file type: %s", len(extractors), ext)
	return extractors, nil
}

// RegisterExtractor registers a new extractor
func (f *DefaultExtractorFactory) RegisterExtractor(name string, extractor interfaces.Extractor) {
	f.extractors[name] = extractor
	f.logger.Debug("Registered extractor: %s", name)
}

// ListExtractors returns all registered extractors
func (f *DefaultExtractorFactory) ListExtractors() []string {
	names := make([]string, 0, len(f.extractors))
	for name := range f.extractors {
		names = append(names, name)
	}
	return names
}

// registerDefaultExtractors registers the default set of extractors
func (f *DefaultExtractorFactory) registerDefaultExtractors() {
	f.logger.Debug("Registering default providers...")

	// Text file extractor
	f.RegisterExtractor("text", providers.NewTextFileExtractor())

	// HTML/MHTML extractor
	f.RegisterExtractor("html", providers.NewHTMLExtractor())

	// OCR extractor for PDFs and images
	f.RegisterExtractor("ocr", ocr.NewOCRExtractor(f.config, f.logger))

	// E-book extractor
	f.RegisterExtractor("ebook", providers.NewEbookExtractor(f.config, f.logger))

	// Calibre fallback extractor
	f.RegisterExtractor("calibre", providers.NewCalibreFallbackExtractor(f.config, f.logger))

	f.logger.Info("Registered %d extractors: %v", len(f.extractors), f.ListExtractors())
}

// GetExtractorPriority returns the priority order for extractors based on file type
func (f *DefaultExtractorFactory) GetExtractorPriority(fileInfo *types.FileInfo) []string {
	ext := strings.ToLower(fileInfo.Extension)

	switch {
	case ext == "txt" || ext == "md" || ext == "json" || ext == "xml":
		return []string{"text"}
	case ext == "html" || ext == "htm" || ext == "mhtml" || ext == "mht":
		return []string{"html"}
	case ext == "epub" || ext == "mobi":
		return []string{"ebook"}
	case ext == "pdf":
		return []string{"ocr"}
	case ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "bmp":
		return []string{"ocr"}
	default:
		// Try OCR as fallback for unknown types
		return []string{"ocr", "text"}
	}
}
