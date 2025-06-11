package ocr

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/config"
	"github.com/nodewee/doc-to-text/pkg/interfaces"
	"github.com/nodewee/doc-to-text/pkg/logger"
	"github.com/nodewee/doc-to-text/pkg/ocr/engines"
	"github.com/nodewee/doc-to-text/pkg/types"
)

// DefaultOCRSelector implements OCRSelector interface
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

	// Register all OCR engines (no availability check)
	selector.engines[types.OCRStrategyLLMCaller] = engines.NewLLMCallerEngine(cfg, log)
	selector.engines[types.OCRStrategySuryaOCR] = engines.NewSuryaOCREngine(cfg, log)

	return selector
}

// SelectOCRStrategy selects an OCR tool, either from config or interactively
func (s *DefaultOCRSelector) SelectOCRStrategy(strategy types.OCRStrategy) (interfaces.OCREngine, error) {
	s.logger.Debug("Selecting OCR strategy: %s", strategy)

	// Handle interactive strategy
	if strategy == types.OCRStrategyInteractive {
		selectedStrategy, err := s.PromptUserSelection()
		if err != nil {
			return nil, fmt.Errorf("failed to select OCR strategy interactively: %w", err)
		}
		strategy = selectedStrategy

		// If LLM Caller is selected interactively, prompt for template
		if strategy == types.OCRStrategyLLMCaller && s.config.LLMTemplate == "" {
			reader := bufio.NewReader(os.Stdin)
			template, err := s.promptForLLMTemplate(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to get LLM template: %w", err)
			}
			s.config.LLMTemplate = template
		}
	}

	// Get the engine for the selected strategy
	engine, exists := s.engines[strategy]
	if !exists {
		return nil, fmt.Errorf("unsupported OCR strategy: %s", strategy)
	}

	s.logger.Info("Selected OCR engine: %s (%s)", engine.Name(), engine.GetDescription())
	return engine, nil
}

// GetAvailableStrategies returns all OCR strategies (no availability check)
func (s *DefaultOCRSelector) GetAvailableStrategies() []types.OCRStrategy {
	strategies := make([]types.OCRStrategy, 0, len(s.engines))
	for strategy := range s.engines {
		strategies = append(strategies, strategy)
	}
	return strategies
}

// GetAllStrategies returns all possible OCR strategies
func (s *DefaultOCRSelector) GetAllStrategies() []types.OCRStrategy {
	return []types.OCRStrategy{
		types.OCRStrategyLLMCaller,
		types.OCRStrategySuryaOCR,
	}
}

// PromptUserSelection prompts user to select an OCR tool interactively
func (s *DefaultOCRSelector) PromptUserSelection() (types.OCRStrategy, error) {
	fmt.Println("\n🔧 OCR Tool Selection")
	fmt.Println("=====================")
	fmt.Println("Please select an OCR tool:")
	fmt.Println("  1. llm-caller   - AI-powered OCR with configurable models")
	fmt.Println("  2. surya_ocr    - Local OCR tool with multilingual support")
	fmt.Printf("\nSelect OCR tool (1-2) [default: 2 (surya_ocr)]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Default to surya_ocr (option 2)
	if input == "" || input == "2" {
		fmt.Println("✅ Selected: surya_ocr")
		return types.OCRStrategySuryaOCR, nil
	} else if input == "1" {
		fmt.Println("✅ Selected: llm-caller")
		return types.OCRStrategyLLMCaller, nil
	} else {
		fmt.Println("❌ Invalid choice, using default: surya_ocr")
		return types.OCRStrategySuryaOCR, nil
	}
}

// promptForLLMTemplate prompts user for LLM template
func (s *DefaultOCRSelector) promptForLLMTemplate(reader *bufio.Reader) (string, error) {
	fmt.Printf("\n📝 LLM Template Selection\n")
	fmt.Printf("========================\n")
	fmt.Printf("Please enter LLM template name for llm-caller: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read template input: %w", err)
	}

	template := strings.TrimSpace(input)
	if template == "" {
		return "", fmt.Errorf("LLM template name is required")
	}

	fmt.Printf("✅ Using LLM template: %s\n", template)
	return template, nil
}
