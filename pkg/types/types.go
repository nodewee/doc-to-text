package types

// MediaType represents different types of media files
type MediaType string

const (
	DocumentMediaType MediaType = "document"
	VideoMediaType    MediaType = "video"
	ImageMediaType    MediaType = "image"
	AudioMediaType    MediaType = "audio"
)

// OCRStrategy represents different OCR tools
type OCRStrategy string

const (
	OCRStrategyLLMCaller   OCRStrategy = "llm-caller"
	OCRStrategySuryaOCR    OCRStrategy = "surya_ocr"
	OCRStrategyInteractive OCRStrategy = "interactive"
)

// ContentType represents the type of content in a document
type ContentType string

const (
	ContentTypeText  ContentType = "text"  // Document contains text content, try Calibre first
	ContentTypeImage ContentType = "image" // Document contains image content, use OCR directly
)

// FileInfo contains basic information about a file
type FileInfo struct {
	MD5Hash    string    `json:"md5_hash"`
	SHA256Hash string    `json:"sha256_hash"`
	Extension  string    `json:"extension"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	MediaType  MediaType `json:"media_type"`
}
