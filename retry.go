package fins

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultRetryInitialDelay = 100 * time.Millisecond
	defaultRetryMaxDelay     = 5 * time.Second
	defaultRetryBackoff      = 2.0
)

// RetryPolicy 重试策略。
type RetryPolicy struct {
	MaxRetries      int           // 最大重试次数（不含首次执行）
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	RetryableErrors []error       // 可重试错误集合
}

// DefaultRetryPolicy 默认重试策略。
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  defaultRetryInitialDelay,
		MaxDelay:      defaultRetryMaxDelay,
		BackoffFactor: defaultRetryBackoff,
		RetryableErrors: []error{
			ErrTimeout,
			ErrConnectionClosed,
		},
	}
}

func cloneRetryPolicy(policy *RetryPolicy) *RetryPolicy {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	cloned := *policy
	if cloned.InitialDelay <= 0 {
		cloned.InitialDelay = defaultRetryInitialDelay
	}
	if cloned.MaxDelay <= 0 {
		cloned.MaxDelay = defaultRetryMaxDelay
	}
	if cloned.MaxDelay < cloned.InitialDelay {
		cloned.MaxDelay = cloned.InitialDelay
	}
	if cloned.BackoffFactor < 1 {
		cloned.BackoffFactor = defaultRetryBackoff
	}
	if cloned.MaxRetries < 0 {
		cloned.MaxRetries = 0
	}
	return &cloned
}

// IsRetryable 判断错误是否可重试。
//
// 使用 [`errors.Is()`](retry.go:55) 兼容包装错误。
func (p *RetryPolicy) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	policy := cloneRetryPolicy(p)
	for _, retryableErr := range policy.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}
	return false
}

// GetDelay 获取指定重试序号的延迟时间。
//
// attempt 从 0 开始，表示“第一次重试前”的等待时间。
func (p *RetryPolicy) GetDelay(attempt int) time.Duration {
	policy := cloneRetryPolicy(p)
	if attempt <= 0 {
		return policy.InitialDelay
	}

	delay := policy.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}
	return delay
}

// DoWithRetry 按策略执行操作。
//
// 约定：
//   - 总执行次数 = 1 + [`RetryPolicy.MaxRetries`](retry.go:11)
//   - 首次执行立即进行
//   - 仅当 [`RetryPolicy.IsRetryable()`](retry.go:54) 返回 true 时才重试
func DoWithRetry(policy *RetryPolicy, operation func() error) error {
	if operation == nil {
		return fmt.Errorf("重试操作不能为空")
	}

	effectivePolicy := cloneRetryPolicy(policy)

	for attempt := 0; attempt <= effectivePolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(effectivePolicy.GetDelay(attempt - 1))
		}

		err := operation()
		if err == nil {
			return nil
		}
		if !effectivePolicy.IsRetryable(err) {
			return err
		}
		if attempt == effectivePolicy.MaxRetries {
			return fmt.Errorf("重试%d次后失败: %w", effectivePolicy.MaxRetries, err)
		}
	}

	return nil
}
