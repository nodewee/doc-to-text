package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/logger"
)

// FileManager统一管理中间文件和临时文件
// 目录结构: {input_file_dir}/{md5_hash}/
//
//	├── text.txt           # 最终输出文本
//	├── pages/             # PDF页面文件
//	│   ├── page_1.pdf
//	│   └── page_1.txt
//	├── ocr_data.json      # OCR结果数据
//	└── temp/              # 临时文件
type FileManager struct {
	inputFile  string
	md5Hash    string
	baseDir    string
	tempFiles  []string
	tempDirs   []string
	mu         sync.RWMutex
	logger     *logger.Logger
	cleanupFns []func() error
}

// NewFileManager 创建新的文件管理器
func NewFileManager(inputFile, md5Hash string, log *logger.Logger) *FileManager {
	inputDir := filepath.Dir(inputFile)
	baseDir := filepath.Join(inputDir, md5Hash)

	return &FileManager{
		inputFile: NormalizePath(inputFile),
		md5Hash:   md5Hash,
		baseDir:   NormalizePath(baseDir),
		logger:    log,
	}
}

// EnsureBaseDir 确保基础目录存在
func (fm *FileManager) EnsureBaseDir() error {
	return EnsureDir(fm.baseDir)
}

// GetBasePath 返回基础路径
func (fm *FileManager) GetBasePath() string {
	return fm.baseDir
}

// GetPath 返回相对于基础目录的路径
func (fm *FileManager) GetPath(relativePath string) string {
	return NormalizePath(filepath.Join(fm.baseDir, relativePath))
}

// === 中间文件管理 ===

// CreateIntermediateDir 创建中间文件目录
func (fm *FileManager) CreateIntermediateDir(name string) (string, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	sanitizedName := SanitizeFileName(name)
	dirPath := fm.GetPath(sanitizedName)

	if err := EnsureDir(dirPath); err != nil {
		return "", fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	fm.logger.Debug("Created intermediate directory: %s", dirPath)
	return dirPath, nil
}

// GetTextFilePath 返回最终文本文件路径
func (fm *FileManager) GetTextFilePath() string {
	return fm.GetPath("text.txt")
}

// GetOCRDataPath 返回OCR数据文件路径
func (fm *FileManager) GetOCRDataPath() string {
	return fm.GetPath("ocr_data.json")
}

// GetPagesDir 返回页面文件目录
func (fm *FileManager) GetPagesDir() string {
	return fm.GetPath("pages")
}

// GetPagePDFPath 返回指定页面PDF路径
func (fm *FileManager) GetPagePDFPath(pageNum int) string {
	return fm.GetPath(filepath.Join("pages", fmt.Sprintf(constants.PDFPageFilePattern, pageNum)))
}

// GetPageTextPath 返回指定页面文本路径
func (fm *FileManager) GetPageTextPath(pageNum int) string {
	return fm.GetPath(filepath.Join("pages", fmt.Sprintf(constants.PDFPageTextPattern, pageNum)))
}

// GetPageImagePath 返回指定页面图像路径
func (fm *FileManager) GetPageImagePath(pageNum int) string {
	return fm.GetPath(filepath.Join("pages", fmt.Sprintf(constants.PDFPageImagePattern, pageNum)))
}

// === 临时文件管理 ===

// CreateTempDir 创建临时目录
func (fm *FileManager) CreateTempDir(prefix string) (string, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if err := fm.EnsureBaseDir(); err != nil {
		return "", fmt.Errorf("failed to ensure base directory: %w", err)
	}

	sanitizedPrefix := SanitizeFileName(prefix)
	if sanitizedPrefix == "" {
		sanitizedPrefix = "temp"
	}

	tempDir, err := os.MkdirTemp("", sanitizedPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	fm.tempDirs = append(fm.tempDirs, tempDir)
	fm.logger.Debug("Created temp directory: %s", tempDir)
	return tempDir, nil
}

// CreateTempFile 创建临时文件
func (fm *FileManager) CreateTempFile(prefix, suffix string) (string, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if err := fm.EnsureBaseDir(); err != nil {
		return "", fmt.Errorf("failed to ensure base directory: %w", err)
	}

	sanitizedPrefix := SanitizeFileName(prefix)
	if sanitizedPrefix == "" {
		sanitizedPrefix = "temp"
	}

	tempDir := fm.GetPath("temp")
	if err := EnsureDir(tempDir); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	file, err := os.CreateTemp(tempDir, sanitizedPrefix+"*"+suffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	tempFile := NormalizePath(file.Name())
	fm.tempFiles = append(fm.tempFiles, tempFile)
	fm.logger.Debug("Created temp file: %s", tempFile)
	return tempFile, nil
}

// RegisterCleanupFunc 注册清理函数
func (fm *FileManager) RegisterCleanupFunc(fn func() error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.cleanupFns = append(fm.cleanupFns, fn)
}

// WithCleanup 执行函数并自动清理临时文件
func (fm *FileManager) WithCleanup(fn func() error) error {
	defer func() {
		if err := fm.CleanupTemp(); err != nil {
			fm.logger.Error("Temporary file cleanup failed: %v", err)
		}
	}()
	return fn()
}

// CleanupTemp 清理临时文件（保留中间文件用于缓存）
func (fm *FileManager) CleanupTemp() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	var errors []error

	// 执行自定义清理函数
	for _, fn := range fm.cleanupFns {
		if err := fn(); err != nil {
			errors = append(errors, err)
			fm.logger.Warn("Cleanup function failed: %v", err)
		}
	}

	// 清理临时文件
	for _, file := range fm.tempFiles {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp file %s: %w", file, err))
			fm.logger.Warn("Failed to remove temporary file: %s, error: %v", file, err)
		} else {
			fm.logger.Debug("Removed temporary file: %s", file)
		}
	}

	// 清理临时目录
	for _, dir := range fm.tempDirs {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp dir %s: %w", dir, err))
			fm.logger.Warn("Failed to remove temporary directory: %s, error: %v", dir, err)
		} else {
			fm.logger.Debug("Removed temporary directory: %s", dir)
		}
	}

	// 清理slice
	fm.tempFiles = fm.tempFiles[:0]
	fm.tempDirs = fm.tempDirs[:0]
	fm.cleanupFns = fm.cleanupFns[:0]

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// Cleanup 清理所有文件（包括中间文件）
func (fm *FileManager) Cleanup() error {
	fm.logger.Debug("Intermediate file cleanup skipped (files preserved for caching)")
	return fm.CleanupTemp()
}
