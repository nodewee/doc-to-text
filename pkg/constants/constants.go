package constants

import "time"

// Application constants
const (
	AppName = "doc-to-text"
)

// File processing constants
const (
	DefaultFilePermission = 0644
	DefaultDirPermission  = 0755
	DefaultMaxRetries     = 3
	DefaultTimeout        = 30 * time.Minute
)

// OCR processing constants
const (
	PDFPageFilePattern  = "page_%d.pdf"
	PDFPageTextPattern  = "page_%d.txt"
	PDFPageImagePattern = "page_%d.png"
)

// File type groups
var (
	ImageExtensions = []string{
		"jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "tif",
	}

	DocumentExtensions = []string{
		"pdf", "doc", "docx", "rtf", "odt", "ppt", "pptx", "xls", "xlsx",
		"html", "htm", "mhtml", "mht",
	}

	TextExtensions = []string{
		"txt", "md", "markdown", "xml", "json", "csv",
	}

	EbookExtensions = []string{
		"epub", "mobi",
	}
)
