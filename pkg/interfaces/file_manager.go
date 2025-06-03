package interfaces

// FileManager defines the interface for file and directory management
type FileManager interface {
	// EnsureBaseDir ensures the base directory exists
	EnsureBaseDir() error

	// GetBasePath returns the base path for file operations
	GetBasePath() string

	// GetPath returns a path under the base directory
	GetPath(relativePath string) string

	// Cleanup performs cleanup operations
	Cleanup() error
}

// IntermediateFileManager manages intermediate files that persist between runs
// These files are used for caching and resume functionality
type IntermediateFileManager interface {
	FileManager

	// CreateIntermediateDir creates a directory for intermediate files
	CreateIntermediateDir(name string) (string, error)

	// GetIntermediatePath returns a path for intermediate files
	GetIntermediatePath(relativePath string) string

	// GetPagesDir returns the directory for PDF page files
	GetPagesDir() string

	// GetPagePDFPath returns the path for a specific page PDF
	GetPagePDFPath(pageNum int) string

	// GetPageTextPath returns the path for a specific page text
	GetPageTextPath(pageNum int) string

	// GetPageImagePath returns the path for a specific page image
	GetPageImagePath(pageNum int) string

	// GetTextFilePath returns the final text output file path
	GetTextFilePath() string

	// GetOCRDataPath returns the path for OCR result data
	GetOCRDataPath() string
}

// TempFileManager manages temporary files that are cleaned up after processing
// These files are only used during the processing workflow
type TempFileManager interface {
	FileManager

	// CreateTempDir creates a temporary directory
	CreateTempDir(prefix string) (string, error)

	// CreateTempFile creates a temporary file
	CreateTempFile(prefix, suffix string) (string, error)

	// RegisterCleanupFunc registers a cleanup function
	RegisterCleanupFunc(fn func() error)

	// WithCleanup executes a function with automatic cleanup
	WithCleanup(fn func() error) error
}
