package ocr

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"doc-to-text/pkg/config"
	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/logger"
	"doc-to-text/pkg/ocr/engines"
	"doc-to-text/pkg/types"
)

// DefaultOCRSelector implements OCR tool selection
type DefaultOCRSelector struct {
	config  *config.Config
	logger  *logger.Logger
	engines map[types.OCRStrategy]interfaces.OCREngine
}

// NewOCRSelector creates a new OCR selector
func NewOCRSelector(cfg *config.Config, log *logger.Logger) interfaces.OCRSelector {
	selector := &DefaultOCRSelector{
		config:  cfg,
		logger:  log,
		engines: make(map[types.OCRStrategy]interfaces.OCREngine),
	}

	// Register available OCR engines
	selector.engines[types.OCRStrategyLLMCaller] = engines.NewLLMCallerEngine(cfg, log)
	selector.engines[types.OCRStrategySuryaOCR] = engines.NewSuryaOCREngine(cfg, log)

	return selector
}

// SelectOCRStrategy selects an OCR tool, either from config or interactively
func (s *DefaultOCRSelector) SelectOCRStrategy(strategy types.OCRStrategy) (interfaces.OCREngine, error) {
	// If interactive mode is requested, check if we should auto-select for text content type
	if strategy == types.OCRStrategyInteractive {
		// If content type is text, auto-select the first available OCR tool instead of prompting
		if s.config.ContentType == types.ContentTypeText {
			s.logger.Info("Content type is 'text', auto-selecting OCR tool for fallback processing")
			available := s.GetAvailableStrategies()
			if len(available) == 0 {
				return nil, fmt.Errorf("no OCR engines are available on this system")
			}

			// Prefer Surya OCR for text content fallback (faster and more suitable)
			for _, availableStrategy := range available {
				if availableStrategy == types.OCRStrategySuryaOCR {
					strategy = availableStrategy
					tool := s.engines[strategy]
					s.logger.Info("Auto-selected preferred OCR tool for text content fallback: %s", tool.GetDescription())
					break
				}
			}

			// If Surya OCR is not available, use the first available tool
			if strategy == types.OCRStrategyInteractive {
				strategy = available[0]
				tool := s.engines[strategy]
				s.logger.Info("Auto-selected OCR tool for text content fallback: %s", tool.GetDescription())
			}
		} else {
			// For image content type or other cases, use interactive selection
			selectedStrategy, err := s.PromptUserSelection()
			if err != nil {
				return nil, fmt.Errorf("failed to select OCR tool: %w", err)
			}
			strategy = selectedStrategy
		}
	}

	// Get the tool for the selected tool
	tool, exists := s.engines[strategy]
	if !exists {
		return nil, fmt.Errorf("unknown OCR tool: %s", strategy)
	}

	// Check if the tool is available
	if !tool.IsAvailable() {
		return nil, fmt.Errorf("OCR tool '%s' is not available on this system", tool.Name())
	}

	s.logger.Info("Selected OCR tool: %s", tool.GetDescription())
	return tool, nil
}

// GetAvailableStrategies returns all available OCR strategies
func (s *DefaultOCRSelector) GetAvailableStrategies() []types.OCRStrategy {
	var available []types.OCRStrategy

	for strategy, tool := range s.engines {
		if tool.IsAvailable() {
			available = append(available, strategy)
		}
	}

	return available
}

// PromptUserSelection prompts user to select an OCR tool interactively
func (s *DefaultOCRSelector) PromptUserSelection() (types.OCRStrategy, error) {
	fmt.Println("\n🔍 OCR Tool Selection")
	fmt.Println("========================")

	// Get available strategies
	available := s.GetAvailableStrategies()

	if len(available) == 0 {
		return "", fmt.Errorf("no OCR engines are available on this system")
	}

	// Display options
	fmt.Println("Available OCR engines:")
	for i, strategy := range available {
		tool := s.engines[strategy]
		fmt.Printf("  %d. %s\n", i+1, tool.GetDescription())
	}

	// Get user input
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nSelect OCR tool (1-%d): ", len(available))
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(available) {
			fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", len(available))
			continue
		}

		selectedStrategy := available[choice-1]
		tool := s.engines[selectedStrategy]
		fmt.Printf("✅ Selected: %s\n", tool.GetDescription())

		// If LLM Caller is selected, prompt for template
		if selectedStrategy == types.OCRStrategyLLMCaller {
			template, err := s.promptForLLMTemplate(reader)
			if err != nil {
				return "", fmt.Errorf("failed to get LLM template: %w", err)
			}
			// Update the configuration with the provided template
			s.config.LLMTemplate = template
			fmt.Printf("✅ LLM Template set: %s\n", template)
		}

		fmt.Println()
		return selectedStrategy, nil
	}
}

// promptForLLMTemplate prompts user to input LLM template for llm-caller
func (s *DefaultOCRSelector) promptForLLMTemplate(reader *bufio.Reader) (string, error) {
	fmt.Println("\n📝 LLM Template Configuration")
	fmt.Println("=============================")
	fmt.Println("LLM Caller requires a template to specify which AI model to use.")
	fmt.Println("Examples: qwen-vl-ocr, gpt-4-vision, claude-vision, etc.")

	for {
		fmt.Print("\nEnter LLM template name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("error reading template input: %w", err)
		}

		template := strings.TrimSpace(input)
		if template == "" {
			fmt.Println("❌ Template name cannot be empty. Please enter a valid template name.")
			continue
		}

		// Basic validation - template should not contain spaces or special characters
		if strings.ContainsAny(template, " \t\n\r") {
			fmt.Println("❌ Template name should not contain spaces. Please enter a valid template name.")
			continue
		}

		return template, nil
	}
}
