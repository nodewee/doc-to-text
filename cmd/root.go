package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/constants"
	"doc-to-text/pkg/core"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/types"
	"doc-to-text/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	outputPath  string
	ocrStrategy string
	llmTemplate string
	contentType string
	verbose     bool
	showVersion bool
)

// AppHandler 封装应用程序主要处理逻辑
type AppHandler struct {
	config    *config.Config
	logger    *logger.Logger
	processor interfaces.FileProcessor
}

// NewAppHandler 创建应用程序处理器
func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

// ProcessFile 处理文件的主要入口
func (h *AppHandler) ProcessFile(inputFile string) error {
	// 初始化配置和组件
	if err := h.initialize(inputFile); err != nil {
		return err
	}

	// 处理文件
	result, err := h.processFile(inputFile)
	if err != nil {
		return err
	}

	// 显示结果
	h.displayResults(result)
	return nil
}

// initialize 初始化应用程序组件
func (h *AppHandler) initialize(inputFile string) error {
	// 验证输入文件路径
	_, err := filepath.Abs(inputFile)
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeValidation, "error resolving file path")
	}

	// 加载配置
	h.config = config.LoadConfigWithEnvOverrides()
	h.applyCommandLineOverrides()

	// 验证配置
	if err := h.config.Validate(); err != nil {
		return utils.WrapError(err, utils.ErrorTypeValidation, "configuration validation failed")
	}

	// 创建日志器和处理器
	h.logger = logger.NewLogger(h.config.LogLevel, h.config.EnableVerbose)
	h.processor = core.NewFileProcessor(h.config, h.logger)

	return nil
}

// applyCommandLineOverrides 应用命令行参数覆盖
func (h *AppHandler) applyCommandLineOverrides() {
	if ocrStrategy != "" {
		h.config.OCRStrategy = types.OCRStrategy(ocrStrategy)

		// 验证LLM模板参数
		if h.config.OCRStrategy == types.OCRStrategyLLMCaller && llmTemplate == "" {
			log.Fatalf("Error: --llm_template is required when using --ocr llm-caller")
		}
		h.config.LLMTemplate = llmTemplate
	}

	if contentType != "" {
		h.config.ContentType = types.ContentType(contentType)
	} else if ocrStrategy == "" {
		// 当没有指定 ocr 参数，也没有指定 content-type 参数时，智能检测文件类型
		// 对于纯文本和HTML文档，直接使用默认设置，不询问用户
		if h.shouldSkipContentTypePrompt() {
			// 对于纯文本和HTML文档，使用默认的 image 类型（实际上不会用到OCR）
			h.config.ContentType = types.ContentTypeImage
			// 注意：此时logger还未初始化，所以不能调用logger方法
		} else {
			// 对于其他文档类型（如PDF），交互询问用户
			selectedContentType, err := h.promptForContentType()
			if err != nil {
				log.Fatalf("Error selecting content type: %v", err)
			}
			h.config.ContentType = selectedContentType
		}
	}

	// 应用 verbose 参数覆盖
	if verbose {
		h.config.EnableVerbose = true
	}
}

// shouldSkipContentTypePrompt 检查是否应该跳过 content type 询问
// 对于纯文本文档和HTML文档，不需要询问 content type
func (h *AppHandler) shouldSkipContentTypePrompt() bool {
	// 这个方法在初始化之前调用，所以需要临时获取文件信息
	if len(os.Args) < 2 {
		return false
	}

	inputFile := os.Args[len(os.Args)-1] // 获取最后一个参数作为输入文件
	if inputFile == "" || strings.HasPrefix(inputFile, "-") {
		return false // 如果是选项参数，不是文件路径
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(inputFile))
	if ext != "" && ext[0] == '.' {
		ext = ext[1:] // 移除点号
	}

	// 检查是否为纯文本文档或HTML文档
	switch ext {
	case "txt", "md", "markdown", "json", "xml", "csv", "py", "js", "ts", "c", "cpp", "h", "java", "sh":
		// 纯文本文档
		return true
	case "html", "htm", "mhtml", "mht":
		// HTML文档
		return true
	case "epub", "mobi":
		// 电子书文档
		return true
	default:
		// 其他文档类型（如PDF、图片等）需要询问
		return false
	}
}

// processFile 执行文件处理逻辑
func (h *AppHandler) processFile(inputFile string) (*interfaces.ExtractionResult, error) {
	absPath, _ := filepath.Abs(inputFile)

	// 确定输出路径
	outputFilePath, err := h.determineOutputPath(absPath)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "error determining output path")
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(h.config.TimeoutMinutes)*time.Minute)
	defer cancel()

	// 执行文件处理（带重试）
	var result *interfaces.ExtractionResult
	err = utils.WithRetry(func() error {
		var processErr error
		result, processErr = h.processor.ProcessFile(ctx, absPath, outputFilePath)
		if processErr != nil {
			return utils.WrapError(processErr, utils.ErrorTypeOCR, "file processing failed")
		}

		if result.Error != "" {
			return utils.NewOCRError(result.Error, nil)
		}
		return nil
	}, constants.DefaultMaxRetries, nil)

	return result, err
}

// determineOutputPath 确定输出文件路径
func (h *AppHandler) determineOutputPath(inputPath string) (string, error) {
	if outputPath != "" {
		return filepath.Abs(outputPath)
	}

	// 使用MD5哈希路径
	md5Hash, err := utils.CalculateFileMD5(inputPath)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "error calculating MD5 hash")
	}

	return h.config.GetTextFilePath(inputPath, md5Hash), nil
}

// displayResults 显示处理结果
func (h *AppHandler) displayResults(result *interfaces.ExtractionResult) {
	fmt.Printf("✅ Text extracted successfully\n")
	fmt.Printf("📊 Extractor used: %s\n", result.ExtractorUsed)
	fmt.Printf("⏱️  Processing time: %dms\n", result.ProcessTime)

	if result.FallbackUsed {
		fmt.Printf("⚠️  Fallback extraction was used\n")
		fmt.Printf("🔄 Attempted extractors: %v\n", result.AttemptedExtractors)
	}

	if len(result.Text) > 0 {
		fmt.Printf("📝 Extracted text length: %d characters\n", len(result.Text))
		h.showTextPreview(result.Text)
	}
}

// showTextPreview 显示文本预览
func (h *AppHandler) showTextPreview(text string) {
	if len(text) > 200 {
		preview := text[:200]
		if lastNewline := strings.LastIndex(preview, "\n"); lastNewline > 0 {
			preview = preview[:lastNewline]
		}
		fmt.Printf("📄 Preview:---\n%s...\n---\n", preview)
	}
}

// promptForContentType 交互式询问用户选择content-type
func (h *AppHandler) promptForContentType() (types.ContentType, error) {
	fmt.Println("\n📄 Content Type Selection")
	fmt.Println("==========================")
	fmt.Println("Please select the content type of your document:")
	fmt.Println("  1. image - Document contains image content, use OCR directly")
	fmt.Println("  2. text  - Document contains text content, try Calibre first")
	fmt.Printf("\nSelect content type (1-2) [default: 1 (image)]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	// 默认选择 image (选项1)
	if input == "" || input == "1" {
		fmt.Println("✅ Selected: image")
		return types.ContentTypeImage, nil
	} else if input == "2" {
		fmt.Println("✅ Selected: text")
		return types.ContentTypeText, nil
	} else {
		fmt.Println("❌ Invalid choice, using default: image")
		return types.ContentTypeImage, nil
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "doc-to-text [input_file]",
	Short: "A CLI tool for extracting text from various document formats",
	Long: `A comprehensive CLI tool for extracting text content from various document formats. Supports PDF, E-books (EPUB/MOBI), HTML/MHTML, images, and text files.

Features:
- Advanced OCR with configurable tools (LLM Caller, Surya OCR)
- E-book conversion using Calibre
- HTML/MHTML text extraction with built-in parser
- Text file direct reading

OCR Tools:
- llm-caller: Use LLM Caller with specified template (requires --llm_template)
- surya_ocr: Use Surya OCR (local OCR tool)
- (not specified): Prompt user to select OCR tool interactively

Content Type Selection:
- When neither --ocr nor --content-type is specified, the tool will interactively prompt you to select the content type
- Default selection is 'image' (press Enter for quick selection)

Examples:
  doc-to-text document.pdf                                        # Interactive content type and OCR selection
  doc-to-text document.pdf --ocr llm-caller --llm_template qwen-vl-ocr  # Extract text using LLM Caller
  doc-to-text document.pdf --ocr surya_ocr                       # Extract text using Surya OCR
  doc-to-text document.pdf --content-type text                   # Try Calibre first, then auto-select OCR if failed
  doc-to-text document.pdf --content-type image                  # Use OCR directly for image-based PDF
  doc-to-text ebook.epub                                          # Convert e-book to text
  doc-to-text image.png                                           # Extract text from image using OCR
  doc-to-text document.pdf -o ./output.txt                       # Extract text to specific file path
  doc-to-text document.pdf --verbose                             # Extract text with verbose progress output
  doc-to-text document.pdf -v --ocr surya_ocr                    # Extract text with verbose output using Surya OCR`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle version flag
		if showVersion {
			fmt.Printf("doc-to-text %s\n", version)
			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		handler := NewAppHandler()
		if err := handler.ProcessFile(args[0]); err != nil {
			if appErr, ok := err.(*utils.AppError); ok {
				log.Fatalf("Error (%s): %s", appErr.Type, appErr.Message)
			} else {
				log.Fatalf("Error: %v", err)
			}
		}
	},
}

// NewRootCmd creates and returns the root command
func NewRootCmd() *cobra.Command {
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add flags to root command
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "",
		"Output file path (default: {input_file_directory}/{md5_hash}/text.txt)")
	rootCmd.Flags().StringVar(&ocrStrategy, "ocr", "",
		"OCR tool (llm-caller, surya_ocr, interactive)")
	rootCmd.Flags().StringVar(&llmTemplate, "llm_template", "",
		"LLM template for LLM Caller OCR")
	rootCmd.Flags().StringVar(&contentType, "content-type", "",
		"Content type of the document (text, image). Default: image. 'text' tries Calibre first, then auto-selects OCR if failed (no interaction). 'image' uses OCR directly.")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable verbose output to show progress information")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false,
		"Show version information")
}
