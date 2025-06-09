package config

import (
	"fmt"
	"strings"

	"github.com/nodewee/doc-to-text/pkg/types"
	"github.com/nodewee/doc-to-text/pkg/utils"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate 验证配置
func (v *ConfigValidator) Validate(c *Config) error {
	var errors []string

	// 验证OCR策略
	if err := v.validateOCRStrategy(c.OCRStrategy); err != nil {
		errors = append(errors, err.Error())
	}

	// 验证内容类型
	if err := v.validateContentType(c.ContentType); err != nil {
		errors = append(errors, err.Error())
	}

	// 验证数值参数
	if err := v.validateNumericValues(c); err != nil {
		errors = append(errors, err.Error())
	}

	// 验证日志级别
	if err := v.validateLogLevel(c.LogLevel); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return utils.NewValidationError("configuration validation failed",
			fmt.Errorf("validation errors: %s", strings.Join(errors, "; ")))
	}

	return nil
}

// validateOCRStrategy 验证OCR策略
func (v *ConfigValidator) validateOCRStrategy(strategy types.OCRStrategy) error {
	validStrategies := []types.OCRStrategy{
		types.OCRStrategyInteractive,
		types.OCRStrategyLLMCaller,
		types.OCRStrategySuryaOCR,
	}

	for _, valid := range validStrategies {
		if strategy == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid OCR strategy: %s", strategy)
}

// validateContentType 验证内容类型
func (v *ConfigValidator) validateContentType(contentType types.ContentType) error {
	validTypes := []types.ContentType{
		types.ContentTypeText,
		types.ContentTypeImage,
	}

	for _, valid := range validTypes {
		if contentType == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid content type: %s", contentType)
}

// validateNumericValues 验证数值参数
func (v *ConfigValidator) validateNumericValues(c *Config) error {
	if c.MaxConcurrency < 1 {
		return fmt.Errorf("max concurrency must be at least 1")
	}
	if c.MaxConcurrency > 20 {
		return fmt.Errorf("max concurrency should not exceed 20")
	}
	if c.MinTextThreshold < 0 {
		return fmt.Errorf("min text threshold must be non-negative")
	}
	if c.TimeoutMinutes < 1 {
		return fmt.Errorf("timeout must be at least 1 minute")
	}

	return nil
}

// validateLogLevel 验证日志级别
func (v *ConfigValidator) validateLogLevel(level string) error {
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, valid := range validLevels {
		if strings.ToLower(level) == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid log level: %s", level)
}
