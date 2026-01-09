package fins

import (
	"fmt"
	"time"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	RetryableErrors []error       // 可重试的错误类型
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []error{
			ErrTimeout,
			ErrConnectionClosed,
		},
	}
}

// IsRetryable 判断错误是否可重试
func (p *RetryPolicy) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	for _, retryableErr := range p.RetryableErrors {
		if err == retryableErr {
			return true
		}
	}

	return false
}

// GetDelay 获取重试延迟时间
func (p *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialDelay
	}

	delay := p.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * p.BackoffFactor)
		if delay > p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}

	return delay
}

// RetryableClient 支持重试的客户端包装器
type RetryableClient struct {
	client *FinsClient
	policy *RetryPolicy
}

// NewRetryableClient 创建支持重试的客户端
func NewRetryableClient(client *FinsClient, policy *RetryPolicy) *RetryableClient {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	return &RetryableClient{
		client: client,
		policy: policy,
	}
}

// executeWithRetry 执行带重试的操作
func (r *RetryableClient) executeWithRetry(operation func() (*FinsResponse, error)) (*FinsResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.policy.GetDelay(attempt - 1)
			time.Sleep(delay)
		}

		resp, err := operation()
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// 检查是否可重试
		if !r.policy.IsRetryable(err) {
			return nil, err
		}

		// 如果是最后一次尝试，不再重试
		if attempt == r.policy.MaxRetries {
			break
		}
	}

	return nil, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, lastErr)
}

// ReadMemoryArea 读取内存区域（带重试）
func (r *RetryableClient) ReadMemoryArea(areaCode byte, address uint16, count uint16) ([]byte, error) {
	// 实际实现
	var result []byte
	var execErr error

	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.policy.GetDelay(attempt - 1)
			time.Sleep(delay)
		}

		result, execErr = r.client.ReadMemoryArea(areaCode, address, count)
		if execErr == nil {
			return result, nil
		}

		if !r.policy.IsRetryable(execErr) {
			return nil, execErr
		}

		if attempt == r.policy.MaxRetries {
			break
		}
	}

	return nil, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, execErr)
}

// WriteMemoryArea 写入内存区域（带重试）
func (r *RetryableClient) WriteMemoryArea(areaCode byte, address uint16, values []uint16) error {
	var lastErr error

	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.policy.GetDelay(attempt - 1)
			time.Sleep(delay)
		}

		err := r.client.WriteMemoryArea(areaCode, address, values)
		if err == nil {
			return nil
		}

		lastErr = err

		if !r.policy.IsRetryable(err) {
			return err
		}

		if attempt == r.policy.MaxRetries {
			break
		}
	}

	return fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, lastErr)
}
