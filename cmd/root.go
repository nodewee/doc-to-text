package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/constants"
	"github.com/nodewee/doc-to-text/pkg/core"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"

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

// AppHandler encapsulates application main processing logic
type AppHandler struct {
	config    *config.Config
	logger    *logger.Logger
	processor interfaces.FileProcessor
}

// NewAppHandler creates an application handler
func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

// ProcessFile is the main entry point for file processing
func (h *AppHandler) ProcessFile(inputFile string) error {
	// Initialize configuration and components
	if err := h.initialize(inputFile); err != nil {
		return err
	}

	// Process the file
	result, err := h.processFile(inputFile)
	if err != nil {
		return err
	}

	// Display results
	h.displayResults(result)
	return nil
}

// initialize initializes application components
func (h *AppHandler) initialize(inputFile string) error {
	// Validate input file path
	_, err := filepath.Abs(inputFile)
	if err != nil {
		return utils.WrapError(err, utils.ErrorTypeValidation, "error resolving file path")
	}

	// Load configuration with environment overrides (no file persistence)
	h.config = config.LoadConfigWithEnvOverrides()
	h.applyCommandLineOverrides()

	// Validate configuration
	if err := h.config.Validate(); err != nil {
		return utils.WrapError(err, utils.ErrorTypeValidation, "configuration validation failed")
	}

	// Create logger and processor
	h.logger = logger.NewLogger(h.config.LogLevel, h.config.EnableVerbose)
	h.processor = core.NewFileProcessor(h.config, h.logger)

	return nil
}

// applyCommandLineOverrides applies command line parameter overrides
func (h *AppHandler) applyCommandLineOverrides() {
	if ocrStrategy != "" {
		h.config.OCRStrategy = types.OCRStrategy(ocrStrategy)

		// Validate LLM template parameter
		if h.config.OCRStrategy == types.OCRStrategyLLMCaller && llmTemplate == "" {
			log.Fatalf("Error: --llm_template is required when using --ocr llm-caller")
		}
		h.config.LLMTemplate = llmTemplate
	}

	if contentType != "" {
		h.config.ContentType = types.ContentType(contentType)
	} else if ocrStrategy == "" {
		// When neither ocr nor content-type is specified, smart detection
		// For pure text and HTML documents, use default settings without prompting
		if h.shouldSkipContentTypePrompt() {
			// For pure text and HTML documents, use default image type (OCR won't actually be used)
			h.config.ContentType = types.ContentTypeImage
			// Note: logger is not initialized yet, so can't call logger methods
		} else {
			// For other document types (like PDF), ask user interactively
			selectedContentType, err := h.promptForContentType()
			if err != nil {
				log.Fatalf("Error selecting content type: %v", err)
			}
			h.config.ContentType = selectedContentType
		}
	}

	// Apply verbose parameter override
	if verbose {
		h.config.EnableVerbose = true
	}
}

// shouldSkipContentTypePrompt checks whether to skip content type prompting
// For pure text documents and HTML documents, no need to ask for content type
func (h *AppHandler) shouldSkipContentTypePrompt() bool {
	// This method is called before initialization, so we need to temporarily get file info
	if len(os.Args) < 2 {
		return false
	}

	inputFile := os.Args[len(os.Args)-1] // Get the last parameter as input file
	if inputFile == "" || strings.HasPrefix(inputFile, "-") {
		return false // If it's an option parameter, not a file path
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(inputFile))
	if ext != "" && ext[0] == '.' {
		ext = ext[1:] // Remove the dot
	}

	// Check if it's a pure text document or HTML document
	switch ext {
	case "txt", "md", "markdown", "json", "xml", "csv", "py", "js", "ts", "c", "cpp", "h", "java", "sh":
		// Pure text documents
		return true
	case "html", "htm", "mhtml", "mht":
		// HTML documents
		return true
	case "epub", "mobi":
		// E-book documents
		return true
	default:
		// Other document types (like PDF, images) need prompting
		return false
	}
}

// processFile executes file processing logic
func (h *AppHandler) processFile(inputFile string) (*interfaces.ExtractionResult, error) {
	absPath, _ := filepath.Abs(inputFile)

	// Determine output path
	outputFilePath, err := h.determineOutputPath(absPath)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeIO, "error determining output path")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(h.config.TimeoutMinutes)*time.Minute)
	defer cancel()

	// Execute file processing (with retry)
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

// determineOutputPath determines the output file path
func (h *AppHandler) determineOutputPath(inputPath string) (string, error) {
	if outputPath != "" {
		return filepath.Abs(outputPath)
	}

	// Use MD5 hash path
	md5Hash, err := utils.CalculateFileMD5(inputPath)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "error calculating MD5 hash")
	}

	return h.config.GetTextFilePath(inputPath, md5Hash), nil
}

// displayResults displays processing results
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

// showTextPreview displays text preview
func (h *AppHandler) showTextPreview(text string) {
	if len(text) > 200 {
		preview := text[:200]
		if lastNewline := strings.LastIndex(preview, "\n"); lastNewline > 0 {
			preview = preview[:lastNewline]
		}
		fmt.Printf("📄 Preview:---\n%s...\n---\n", preview)
	}
}

// promptForContentType interactively asks user to select content-type
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

	// Default to image (option 1)
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
- E-book conversion using Calibre (auto-detected)
- HTML/MHTML text extraction with built-in parser
- Text file direct reading
- Cross-platform tool detection

OCR Tools:
- llm-caller: Use LLM Caller with specified template (requires --llm_template)
- surya_ocr: Use Surya OCR (local OCR tool)
- (not specified): Prompt user to select OCR tool interactively

Tool Detection:
- Tools are automatically detected when needed
- No configuration file required
- Clear error messages if tools are missing

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
