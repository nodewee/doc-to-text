package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/types"
)

// File extension sets for different media types
var (
	TextExtensions = map[string]bool{
		"txt": true, "md": true, "markdown": true, "xml": true, "json": true, "csv": true, "py": true, "js": true,
		"ts": true, "c": true, "cpp": true, "h": true, "java": true, "sh": true,
	}

	DocumentExtensions = map[string]bool{
		"pdf": true, "doc": true, "docx": true, "rtf": true, "odt": true,
		"ppt": true, "pptx": true, "xls": true, "xlsx": true, "mhtml": true, "mht": true,
		"html": true, "htm": true, "epub": true, "mobi": true,
	}

	VideoExtensions = map[string]bool{
		"mp4": true, "avi": true, "mov": true, "mkv": true, "wmv": true,
		"flv": true, "webm": true, "m4v": true, "mpg": true, "mpeg": true,
	}

	ImageExtensions = map[string]bool{
		"jpg": true, "jpeg": true, "png": true, "gif": true, "bmp": true,
		"svg": true, "webp": true, "tiff": true, "tif": true,
	}

	AudioExtensions = map[string]bool{
		"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true,
		"m4a": true, "wma": true,
	}

	// E-book extensions
	EbookExtensions = map[string]bool{
		"epub": true, "mobi": true,
	}

	// MIME type prefixes
	TextMimePrefixes = []string{
		"text/", "application/json", "application/xml", "application/csv",
	}

	DocumentMimePrefixes = []string{
		"application/pdf", "application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml",
		"message/rfc822", "multipart/related",
	}

	EbookMimeTypes = []string{
		"application/epub+zip", "application/x-mobipocket-ebook",
	}
)

// GetFileInfo extracts basic information about a file
func GetFileInfo(filePath string) (*types.FileInfo, error) {
	// Get file stats
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting file stats: %w", err)
	}

	// Calculate hashes
	md5Hash, sha256Hash, err := calculateHashes(filePath)
	if err != nil {
		return nil, fmt.Errorf("error calculating hashes: %w", err)
	}

	// Get file extension
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))

	// Get MIME type
	mimeType, err := getMimeType(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting MIME type: %w", err)
	}

	// Determine media type
	mediaType := determineMediaType(extension, mimeType)

	return &types.FileInfo{
		MD5Hash:    md5Hash,
		SHA256Hash: sha256Hash,
		Extension:  extension,
		MimeType:   mimeType,
		Size:       stat.Size(),
		MediaType:  mediaType,
	}, nil
}

// IsTextFile determines if a file is a text file
func IsTextFile(extension, mimeType string) bool {
	// Check by extension
	if TextExtensions[strings.ToLower(extension)] {
		return true
	}

	// Exclude HTML files from text processing even if they have text/* MIME type
	extension = strings.ToLower(extension)
	if extension == "html" || extension == "htm" || extension == "mhtml" || extension == "mht" {
		return false
	}

	// Check by MIME type
	for _, prefix := range TextMimePrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			// Additional check to exclude HTML MIME types
			if strings.Contains(mimeType, "html") {
				return false
			}
			return true
		}
	}

	return false
}

// IsEbookFile determines if a file is an e-book
func IsEbookFile(extension, mimeType string) bool {
	// Check by extension
	if EbookExtensions[strings.ToLower(extension)] {
		return true
	}

	// Check by MIME type
	for _, mimePattern := range EbookMimeTypes {
		if strings.Contains(mimeType, mimePattern) {
			return true
		}
	}

	return false
}

// NeedsOCR determines if a file needs OCR or can have text directly extracted
func NeedsOCR(extension, mimeType string) bool {
	extension = strings.ToLower(extension)

	// Text files can be read directly
	if IsTextFile(extension, mimeType) {
		return false
	}

	// MHTML files can have text extracted directly
	if extension == "mhtml" || extension == "mht" {
		return false
	}

	// Images always need OCR
	if ImageExtensions[extension] {
		return true
	}

	// PDF files should try OCR first, then fallback to direct extraction
	if extension == "pdf" {
		return true
	}

	// E-books should be converted using calibre, not OCR
	if IsEbookFile(extension, mimeType) {
		return false
	}

	// Other document types may need OCR depending on content
	return true
}

// determineMediaType determines the media type based on extension and MIME type
func determineMediaType(extension, mimeType string) types.MediaType {
	extension = strings.ToLower(extension)

	// Check by extension first
	if DocumentExtensions[extension] || TextExtensions[extension] {
		return types.DocumentMediaType
	}
	if VideoExtensions[extension] {
		return types.VideoMediaType
	}
	if ImageExtensions[extension] {
		return types.ImageMediaType
	}
	if AudioExtensions[extension] {
		return types.AudioMediaType
	}

	// Fall back to MIME type checking
	for _, prefix := range DocumentMimePrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return types.DocumentMediaType
		}
	}
	for _, mimePattern := range EbookMimeTypes {
		if strings.Contains(mimeType, mimePattern) {
			return types.DocumentMediaType
		}
	}
	for _, prefix := range TextMimePrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return types.DocumentMediaType
		}
	}
	if strings.HasPrefix(mimeType, "video/") {
		return types.VideoMediaType
	}
	if strings.HasPrefix(mimeType, "image/") {
		return types.ImageMediaType
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return types.AudioMediaType
	}

	// Default to document for unknown types
	return types.DocumentMediaType
}

// calculateHashes calculates MD5 and SHA256 hashes for a file
func calculateHashes(filePath string) (string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	// Calculate MD5
	md5Cmd := exec.Command("md5", "-q", filePath)
	md5Output, err := md5Cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error calculating MD5: %w", err)
	}
	md5Hash := strings.TrimSpace(string(md5Output))

	// Calculate SHA256
	sha256Cmd := exec.Command("shasum", "-a", "256", filePath)
	sha256Output, err := sha256Cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error calculating SHA256: %w", err)
	}
	sha256Hash := strings.Fields(strings.TrimSpace(string(sha256Output)))[0]

	return md5Hash, sha256Hash, nil
}

// getMimeType gets the MIME type of a file using the file command
func getMimeType(filePath string) (string, error) {
	cmd := exec.Command("file", "--mime-type", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting MIME type: %w", err)
	}

	// Parse output: "filename: mime/type"
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected output format from file command")
	}

	return strings.TrimSpace(parts[1]), nil
}

// IsImageFile determines if a file is an image file
func IsImageFile(extension string) bool {
	return ImageExtensions[strings.ToLower(extension)]
}
