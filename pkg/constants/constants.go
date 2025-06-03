package constants

import "time"

// Application constants
const (
	AppName = "doc-to-text"
	// Note: AppVersion is now managed via build-time ldflags injection in main.go
	// Use cmd.GetVersionInfo() to get the current version at runtime
)

// Semantic versioning constants
const (
	// Version format components
	VersionFormat    = "v%d.%d.%d"    // Standard semantic version format
	PreReleaseFormat = "v%d.%d.%d-%s" // Version with pre-release identifier
	DevVersionSuffix = "dev"          // Development build suffix

	// Version comparison constants
	VersionMajor = 1 // Current major version
	VersionMinor = 0 // Current minor version
	VersionPatch = 0 // Current patch version
)

// File processing constants
const (
	// Default file permissions
	DefaultFilePermission = 0644
	DefaultDirPermission  = 0755

	// Text processing
	DefaultMinTextLength     = 10
	DefaultPageTextHeader    = "--- Page %d ---"
	DefaultTextFileExtension = ".txt"

	// Retry and timeout settings
	DefaultMaxRetries      = 3
	DefaultOCRRetries      = 2
	DefaultTimeoutDuration = 30 * time.Minute
	DefaultShortTimeout    = 5 * time.Minute

	// Concurrency limits
	MaxConcurrentPages    = 10
	DefaultWorkerPoolSize = 4
)

// File size limits (in bytes)
const (
	MaxFileSize       = 100 * 1024 * 1024 // 100MB
	MaxImageSize      = 50 * 1024 * 1024  // 50MB
	WarnFileSizeLimit = 10 * 1024 * 1024  // 10MB
)

// Progress and logging
const (
	ProgressUpdateInterval = 100 * time.Millisecond
	LogBufferSize          = 1000
)

// OCR processing constants
const (
	// Image conversion settings
	DefaultImageDPI     = 300
	DefaultImageQuality = 90
	DefaultImageFormat  = "jpeg"

	// PDF processing
	PDFPageFilePattern  = "page_%d.pdf"
	PDFPageTextPattern  = "page_%d.txt"
	PDFPageImagePattern = "page_%d.png"

	// OCR result validation
	MinValidTextLength   = 1
	MaxEmptyPagesAllowed = 5
)

// Error messages
const (
	ErrInvalidFile       = "invalid or corrupted file"
	ErrUnsupportedFormat = "unsupported file format"
	ErrOCRFailed         = "OCR processing failed"
	ErrNoTextFound       = "no text content found"
	ErrPermissionDenied  = "permission denied"
	ErrInsufficientSpace = "insufficient disk space"
)

// File type groups
var (
	ImageExtensions = []string{
		"jpg", "jpeg", "png", "gif", "bmp",
		"svg", "webp", "tiff", "tif",
	}

	DocumentExtensions = []string{
		"pdf", "doc", "docx", "rtf", "odt",
		"ppt", "pptx", "xls", "xlsx",
		"html", "htm", "mhtml", "mht",
	}

	TextExtensions = []string{
		"txt", "md", "markdown", "xml", "json",
		"csv", "py", "js", "ts", "c", "cpp",
		"h", "java", "sh",
	}

	EbookExtensions = []string{
		"epub", "mobi",
	}
)

// MIME type patterns
var (
	TextMimePatterns = []string{
		"text/", "application/json", "application/xml",
		"application/csv",
	}

	DocumentMimePatterns = []string{
		"application/pdf", "application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml",
		"message/rfc822", "multipart/related",
	}

	EbookMimePatterns = []string{
		"application/epub+zip", "application/x-mobipocket-ebook",
	}
)
