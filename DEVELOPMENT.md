# Development Guide

Development information for the Doc Text Extractor project.

## 📚 Documentation Links

- **🏷️ [Version Management](VERSIONING.md)** - Build system, versioning, and release process
- **🚀 [Quick Start](QUICKSTART.md)** - Installation and basic usage  
- **📖 [User Guide](README.md)** - Complete usage documentation

## 🏗️ Architecture Overview

The project follows clean architecture with modular design and clear separation of concerns.

### Project Structure

```
doc-to-text/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command with content-type logic
│   ├── config.go          # Configuration management commands
│   └── version.go         # Version display with build-time injection
├── pkg/
│   ├── config/            # Configuration with auto-detection
│   ├── constants/         # Platform-specific constants
│   ├── logger/            # Structured logging
│   ├── interfaces/        # Clean interfaces design
│   ├── types/             # OCR strategies and content types
│   ├── utils/             # Cross-platform utilities
│   ├── core/              # Business logic
│   │   ├── factory.go     # Extractor factory with fallback chains
│   │   └── processor.go   # File processing workflow
│   ├── providers/         # Format-specific extractors
│   └── ocr/               # OCR system
│       ├── extractor.go   # Main OCR coordinator
│       ├── selector.go    # Interactive tool selection
│       └── engines/       # OCR engine implementations
├── main.go                # Entry point with version injection
└── build.sh               # Build script (see VERSIONING.md)
```

## 🎯 Key Design Patterns

### 1. Strategy Pattern for OCR Tools

```go
type OCREngine interface {
    Name() string
    ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error)
    ExtractTextFromImage(ctx context.Context, imagePath string) (string, error)
    SupportsDirectPDF() bool
    IsAvailable() bool
}
```

### 2. Factory Pattern with Fallback Chains

```go
func (f *DefaultExtractorFactory) CreateExtractorWithFallbacks(fileInfo *types.FileInfo) ([]interfaces.Extractor, error) {
    switch {
    case ext == "pdf":
        if f.config.ContentType == types.ContentTypeText {
            // Text-first: Calibre → OCR fallback
            extractors = append(extractors, calibreExtractor, ocrExtractor)
        } else {
            // Image-first: OCR only
            extractors = append(extractors, ocrExtractor)
        }
    }
}
```

### 3. Interactive Selection System

```go
func (s *DefaultOCRSelector) SelectOCRStrategy(strategy types.OCRStrategy) (interfaces.OCREngine, error) {
    if strategy == types.OCRStrategyInteractive {
        return s.PromptUserSelection()
    }
    // ... direct selection logic
}
```

## 🔧 Core Components

### Configuration System (`pkg/config/`)

**Auto-detection with platform awareness:**

```go
func detectAndUpdateToolPaths(configFile *ConfigFile) {
    platformConfig := constants.GetPlatformConfig()
    
    toolsToDetect := map[string][]string{
        "surya_ocr":   {utils.DefaultPathUtils.GetExecutableName("surya_ocr")},
        "ghostscript": append(getGhostscriptPossibleNames(), platformConfig.GhostscriptPaths...),
        "calibre":     append([]string{utils.DefaultPathUtils.GetExecutableName("ebook-convert")}, platformConfig.CalibrePaths...),
    }
}
```

### OCR System Architecture (`pkg/ocr/`)

**Page-by-page processing with resume capability:**

```go
func (e *OCRExtractor) extractFromPDFViaSinglePages(ctx context.Context, inputFile string, fileInfo *types.FileInfo) (map[string]interface{}, error) {
    // Split PDF into single pages
    totalPages, err := e.splitPDFIntoSinglePages(ctx, inputFile, pagesDir)
    
    // Process each page with resume capability
    for pageNum := 1; pageNum <= totalPages; pageNum++ {
        pageTextPath := e.intermediateManager.GetPageTextPath(pageNum)
        if existingText, err := e.loadExistingPageText(pageTextPath, pageNum); err == nil {
            continue // Skip already processed pages
        }
    }
}
```

### Content Type Logic (`cmd/root.go`)

**Smart content type detection:**

```go
func (h *AppHandler) shouldSkipContentTypePrompt() bool {
    ext := strings.ToLower(filepath.Ext(inputFile))
    switch ext {
    case "txt", "md", "html", "epub":
        return true // Skip prompt for text documents
    default:
        return false // Prompt for PDFs and images
    }
}
```

## 🚀 Adding New Features

### Adding a New OCR Engine

1. **Implement the OCREngine interface** in `pkg/ocr/engines/`:

```go
type NewOCREngine struct {
    config *config.Config
    logger *logger.Logger
}

func (e *NewOCREngine) ExtractTextFromPDF(ctx context.Context, pdfPath string) (string, error) {
    // Implementation with intermediate file support
    if e.intermediateManager != nil {
        resultFile := e.intermediateManager.GetOCRDataPath()
        if existingText, err := e.loadExistingResults(resultFile); err == nil {
            return existingText, nil
        }
    }
    // ... OCR logic
}
```

2. **Register in selector** (`pkg/ocr/selector.go`):

```go
selector.engines[types.OCRStrategyNewEngine] = engines.NewNewOCREngine(cfg, log)
```

3. **Add strategy type** to `pkg/types/types.go`:

```go
const OCRStrategyNewEngine OCRStrategy = "new_engine"
```

### Adding a New File Type Extractor

1. **Implement the Extractor interface** in `pkg/providers/`:

```go
func (e *NewFormatExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
    return fileInfo.Extension == "newformat"
}
```

2. **Register in factory** (`pkg/core/factory.go`):

```go
f.RegisterExtractor("newformat", providers.NewNewFormatExtractor(f.config, f.logger))
```

## 🧪 Testing Strategy

### Key Test Areas

1. **OCR Engine Integration**: Test with mock engines
2. **Content Type Logic**: Verify fallback chains
3. **Cross-platform Paths**: Test tool detection
4. **Resume Capability**: Test interrupted processing

### Example Test Structure

```go
func TestOCRExtractor_ResumeCapability(t *testing.T) {
    // Create partially processed state
    // Verify resume continues from correct page
    // Test completion detection
}
```

## 🔍 Code Quality Standards

### Project-Specific Guidelines

1. **Error Wrapping**: Use `utils.WrapError` for context
2. **Logging**: Use structured logging with progress indicators
3. **Platform Handling**: Use `constants.GetPlatformConfig()`
4. **Path Operations**: Use `utils.DefaultPathUtils` for cross-platform paths

### Critical Error Handling

```go
func (e *OCRExtractor) processPageWithRetry(ctx context.Context, pagePDFPath string, displayPageNum int) (string, error) {
    return e.errorHandler.WithRetrySimple(func() error {
        text, err := e.processPagePDFWithOCR(ctx, pagePDFPath, displayPageNum)
        if err != nil {
            return utils.WrapError(err, utils.ErrorTypeOCR, "page processing failed")
        }
        return nil
    })
}
```

## 🔧 Development Environment

### Initial Setup

```bash
# Clone and setup
git clone <repository-url>
cd doc-to-text
go mod download

# Install development dependencies
brew install ghostscript pandoc calibre  # macOS
pip install surya-ocr

# Build and test
go build -o doc-to-text .
go test ./...
```

### Development Workflow

For build commands and version management, see **[VERSIONING.md](VERSIONING.md)**:

```bash
# Quick development build (see VERSIONING.md for details)
./build.sh local

# Install for local testing  
./install-bin.sh
```

### Debugging

```bash
# Enable debug logging
DOC_TEXT_LOG_LEVEL=debug ./doc-to-text document.pdf

# Test OCR engine detection
./doc-to-text config list

# Test specific components
go test ./pkg/ocr -v
go test ./pkg/config -v
```

## 📋 Project-Specific Conventions

### Configuration Management

- **Tool paths**: Stored in JSON config file with auto-detection
- **Runtime settings**: Environment variables only
- **Platform awareness**: Uses `constants.GetPlatformConfig()`

### File Organization

- **Intermediate files**: `{input_dir}/{md5_hash}/`
- **Page processing**: `pages/page_N.pdf` and `pages/page_N.txt`
- **OCR results**: Persistent in intermediate directories for resume capability

### OCR Processing Flow

1. **File type detection**: Determine processing strategy based on extension
2. **Tool selection**: Interactive prompt or configured OCR engine
3. **PDF page splitting**: Split into single-page PDFs using Ghostscript
4. **Parallel processing**: Process pages concurrently with worker pools
5. **Resume handling**: Skip already completed pages automatically
6. **Text aggregation**: Combine page results with page headers

### Error Recovery Strategy

- **Retry logic**: Automatic retry for transient failures
- **Graceful fallback**: OCR → Calibre → Text extraction chains
- **State preservation**: Intermediate files enable process resumption
- **Context propagation**: Proper error context through utils.WrapError

## 📚 Key Interfaces and Dependencies

### Core Interfaces

- `interfaces.OCREngine`: OCR tool abstraction with availability detection
- `interfaces.Extractor`: File format extractor with support detection
- `interfaces.ExtractorFactory`: Extractor creation with fallback chains
- `interfaces.FileProcessor`: Main processing workflow with error handling

### External Dependencies

- **Ghostscript**: PDF processing, page splitting, format conversion
- **Pandoc**: Office document conversion (DOC, PPT, XLS to text)
- **Calibre**: E-book text extraction (EPUB, MOBI to text)
- **OCR Tools**: Surya OCR (fast), LLM Caller (AI-powered, configurable)

### Platform Considerations

The project uses platform-aware tool detection and path handling:

- **Windows**: Executable extensions (`.exe`), program files paths
- **macOS**: Application bundles, Homebrew paths
- **Linux**: Package manager paths, snap packages

All cross-platform logic is centralized in `pkg/constants/platform.go` and `pkg/utils/path.go`. 