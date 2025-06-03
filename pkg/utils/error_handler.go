package utils

import (
	"context"
	"time"
)

// SimpleErrorHandler 简化的错误处理器
type SimpleErrorHandler struct {
	maxRetries int
	baseDelay  time.Duration
}

// NewSimpleErrorHandler 创建简化的错误处理器
func NewSimpleErrorHandler(maxRetries int) *SimpleErrorHandler {
	return &SimpleErrorHandler{
		maxRetries: maxRetries,
		baseDelay:  time.Second,
	}
}

// WithRetrySimple 简化的重试逻辑
func (h *SimpleErrorHandler) WithRetrySimple(fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err

			// 如果不是最后一次尝试，等待后重试
			if attempt < h.maxRetries {
				delay := h.baseDelay * time.Duration(attempt+1)
				time.Sleep(delay)
				continue
			}
		} else {
			return nil // 成功
		}
	}

	return lastErr
}

// WithRetryContext 带上下文的重试逻辑
func (h *SimpleErrorHandler) WithRetryContext(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			lastErr = err

			// 如果不是最后一次尝试，等待后重试
			if attempt < h.maxRetries {
				delay := h.baseDelay * time.Duration(attempt+1)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		} else {
			return nil // 成功
		}
	}

	return lastErr
}

// IsRetryable 判断错误是否可重试
func (h *SimpleErrorHandler) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errorType := GetErrorType(err)
	switch errorType {
	case ErrorTypeTimeout, ErrorTypeNetwork, ErrorTypeIO:
		return true
	case ErrorTypeValidation, ErrorTypePermission, ErrorTypeNotFound:
		return false
	default:
		return false
	}
}
