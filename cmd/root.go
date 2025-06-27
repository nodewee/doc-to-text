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
			log.Fatalf("Error: LLM template is required when using llm-caller OCR strategy")
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
// For pure text documents, HTML documents, e-books, and image files, no need to ask for content type
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

	// Check if it's a pure text document, HTML document, e-book, or image file
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
	case "jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "tiff", "tif":
		// Image files - automatically use image content type
		return true
	default:
		// Other document types (like PDF) need prompting
		return false
	}
}

// processFile executes file processing logic
func (h *AppHandler) processFile(inputFile string) (*interfaces.ExtractionResult, error) {
	absPath, _ := filepath.Abs(inputFile)

	// Determine output path with early validation
	outputFilePath, err := h.determineOutputPath(absPath)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeValidation, "error determining output path")
	}

	// Validate the determined output path before processing
	if err := h.validateOutputPath(outputFilePath); err != nil {
		return nil, utils.WrapError(err, utils.ErrorTypeValidation, "output path validation failed")
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
		// Convert to absolute path
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			return "", utils.WrapError(err, utils.ErrorTypeValidation, fmt.Sprintf("failed to resolve output path '%s'", outputPath))
		}

		// Check if the output path points to an existing directory
		if info, err := os.Stat(absOutputPath); err == nil {
			if info.IsDir() {
				// User specified an existing directory - this is not allowed
				return "", utils.NewValidationError(fmt.Sprintf("output path '%s' is a directory, please specify a file path", absOutputPath), nil)
			}
			// If it's an existing file, we'll overwrite it (this is intentional)
		} else if err != nil && !os.IsNotExist(err) {
			// If there's an error other than "not exist", report it
			return "", utils.WrapError(err, utils.ErrorTypeIO, fmt.Sprintf("failed to check output path '%s'", absOutputPath))
		}

		// If the path doesn't exist or is a file, use it as-is
		return absOutputPath, nil
	}

	// Use MD5 hash path
	md5Hash, err := utils.CalculateFileMD5(inputPath)
	if err != nil {
		return "", utils.WrapError(err, utils.ErrorTypeIO, "failed to calculate MD5 hash")
	}

	// Generate output path based on input directory and MD5 hash
	inputDir := filepath.Dir(inputPath)
	return filepath.Join(inputDir, md5Hash, "text.txt"), nil
}

// validateOutputPath validates the output file path
func (h *AppHandler) validateOutputPath(outputPath string) error {
	if outputPath == "" {
		return utils.NewValidationError("output path cannot be empty", nil)
	}

	// Check if the output directory exists or can be created
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.WrapError(err, utils.ErrorTypePermission, fmt.Sprintf("failed to create output directory '%s'", outputDir))
	}

	// Check if we can write to the output directory
	testFile := filepath.Join(outputDir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return utils.WrapError(err, utils.ErrorTypePermission, fmt.Sprintf("no write permission for output directory '%s'", outputDir))
	}
	os.Remove(testFile) // Clean up test file

	// Validate the output filename if it's not a directory
	outputFileName := filepath.Base(outputPath)
	if err := utils.ValidatePath(outputFileName); err != nil {
		return utils.WrapError(err, utils.ErrorTypeValidation, fmt.Sprintf("invalid output filename '%s'", outputFileName))
	}

	return nil
}

// displayResults displays processing results
func (h *AppHandler) displayResults(result *interfaces.ExtractionResult) {
	fmt.Printf("✅ Text extraction completed successfully\n")
	fmt.Printf("📊 Extractor used: %s\n", result.ExtractorUsed)
	fmt.Printf("⏱️  Processing time: %dms\n", result.ProcessTime)

	if result.FallbackUsed {
		fmt.Printf("⚠️  Fallback extractor was used\n")
		fmt.Printf("🔄 Attempted extractors: %v\n", result.AttemptedExtractors)
	}

	if len(result.Text) > 0 {
		fmt.Printf("📝 Text length: %d characters\n", len(result.Text))
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
		fmt.Printf("📄 Preview: %s...\n", preview)
	}
}

// promptForContentType interactively asks user to select content-type
func (h *AppHandler) promptForContentType() (types.ContentType, error) {
	fmt.Printf("\n📄 Content Type Selection\n")
	fmt.Println("==========================")
	fmt.Printf("Please select PDF processing strategy:\n")
	fmt.Printf("  1. image - Direct OCR processing (default for scanned documents)\n")
	fmt.Printf("  2. text  - Calibre text extraction first, OCR fallback (fast for text-based PDFs)\n")
	fmt.Printf("\nSelect option (1-2) [default: 1]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	// Default to image (option 1)
	if input == "" || input == "1" {
		fmt.Printf("✅ Selected: image\n")
		return types.ContentTypeImage, nil
	} else if input == "2" {
		fmt.Printf("✅ Selected: text\n")
		return types.ContentTypeText, nil
	} else {
		fmt.Printf("❌ Invalid choice '%s', using default: image\n", input)
		return types.ContentTypeImage, nil
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	SilenceUsage: true,
	Use:          "doc-to-text [input_file]",
	Short:        "A CLI tool for extracting text from various document formats",
	Long: "A CLI tool for extracting text from various document formats with configurable OCR capabilities.\n\n" +
		"Features:\n" +
		"- Multi-format support: PDF, Word, HTML, E-books, Images, and text files\n" +
		"- Configurable OCR: LLM Caller (AI-powered) or Surya OCR (fast & multilingual)\n" +
		"- Smart content strategy: Choose between text-first or image-first processing for PDFs\n" +
		"- Interactive tool selection: Auto-detects available tools and prompts for selection\n" +
		"- Cross-platform: macOS, Linux, Windows with automatic tool detection\n\n" +
		"Examples:\n" +
		"  doc-to-text document.pdf                                        # Interactive mode with tool selection\n" +
		"  doc-to-text document.pdf --ocr llm-caller --llm-template qwen-vl-ocr  # Use LLM Caller with template\n" +
		"  doc-to-text document.pdf --ocr surya_ocr                       # Use Surya OCR\n" +
		"  doc-to-text document.pdf --content-type text                   # Text-first processing\n" +
		"  doc-to-text document.pdf --content-type image                  # Image-first processing\n" +
		"  doc-to-text ebook.epub                                          # Extract from e-book\n" +
		"  doc-to-text image.png                                           # Extract from image\n" +
		"  doc-to-text document.pdf -o ./output.txt                       # Custom output file\n" +
		"  doc-to-text document.pdf --verbose                             # Enable verbose output\n" +
		"  doc-to-text document.pdf -v --ocr surya_ocr                    # Verbose with Surya OCR",
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

		inputFile := args[0]

		// Early validation: Check if input is a directory
		if fileInfo, err := os.Stat(inputFile); err == nil && fileInfo.IsDir() {
			log.Fatalf("Error (validation): Input path '%s' is a directory, please specify a file", inputFile)
		}

		// Early validation: Check if output path (if specified) points to an existing directory
		if outputPath != "" {
			if absOutputPath, err := filepath.Abs(outputPath); err == nil {
				if fileInfo, err := os.Stat(absOutputPath); err == nil && fileInfo.IsDir() {
					log.Fatalf("Error (validation): Output path '%s' is a directory, please specify a file path", absOutputPath)
				}
			}
		}

		handler := NewAppHandler()
		if err := handler.ProcessFile(inputFile); err != nil {
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
	// Update flag descriptions
	updateFlagDescriptions()
	return rootCmd
}

// updateFlagDescriptions updates flag descriptions
func updateFlagDescriptions() {
	rootCmd.Flags().Lookup("output").Usage = "Output file path"
	rootCmd.Flags().Lookup("ocr").Usage = "OCR strategy (interactive, llm-caller, surya_ocr)"
	rootCmd.Flags().Lookup("llm-template").Usage = "LLM template name (required for llm-caller)"
	rootCmd.Flags().Lookup("content-type").Usage = "Content processing type (text, image)"
	rootCmd.Flags().Lookup("verbose").Usage = "Enable verbose output"
	rootCmd.Flags().Lookup("version").Usage = "Show version information"
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Initialize flags
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	rootCmd.Flags().StringVar(&ocrStrategy, "ocr", "", "OCR strategy")
	rootCmd.Flags().StringVar(&llmTemplate, "llm-template", "", "LLM template")
	rootCmd.Flags().StringVar(&contentType, "content-type", "", "Content type")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Show version")
}
