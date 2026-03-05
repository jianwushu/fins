package fins

import (
	"encoding/binary"
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
	return &RetryableClient{client: client, policy: policy}
}

// ========== 对外统一 API（字符串地址，带重试） ==========

func (r *RetryableClient) ReadWord(address string) (uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return 0, err
	}
	if pa.IsBit {
		return 0, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var data []byte
	var execErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		data, execErr = r.client.readMemoryArea(pa.AreaCode, pa.Address, 1)
		if execErr == nil {
			break
		}
		if !r.policy.IsRetryable(execErr) {
			return 0, execErr
		}
	}
	if execErr != nil {
		return 0, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, execErr)
	}
	if len(data) < 2 {
		return 0, ErrInvalidResponse
	}
	return binary.BigEndian.Uint16(data[0:2]), nil
}

func (r *RetryableClient) ReadWords(address string, count uint16) ([]uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var data []byte
	var execErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		data, execErr = r.client.readMemoryArea(pa.AreaCode, pa.Address, count)
		if execErr == nil {
			break
		}
		if !r.policy.IsRetryable(execErr) {
			return nil, execErr
		}
	}
	if execErr != nil {
		return nil, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, execErr)
	}
	if len(data) < int(count)*2 {
		return nil, ErrInvalidResponse
	}

	result := make([]uint16, count)
	for i := uint16(0); i < count; i++ {
		result[i] = binary.BigEndian.Uint16(data[i*2 : (i+1)*2])
	}
	return result, nil
}

func (r *RetryableClient) WriteWord(address string, value uint16) error {
	return r.WriteWords(address, []uint16{value})
}

func (r *RetryableClient) WriteWords(address string, values []uint16) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var lastErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		err := r.client.writeMemoryArea(pa.AreaCode, pa.Address, values)
		if err == nil {
			return nil
		}
		lastErr = err
		if !r.policy.IsRetryable(err) {
			return err
		}
	}
	return fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, lastErr)
}

func (r *RetryableClient) ReadBytes(address string, byteCount uint16) ([]byte, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var result []byte
	var execErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		result, execErr = r.client.readBytes(pa.AreaCode, pa.Address, byteCount)
		if execErr == nil {
			return result, nil
		}
		if !r.policy.IsRetryable(execErr) {
			return nil, execErr
		}
	}
	return nil, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, execErr)
}

func (r *RetryableClient) WriteBytes(address string, data []byte) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var lastErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		err := r.client.writeBytes(pa.AreaCode, pa.Address, data)
		if err == nil {
			return nil
		}
		lastErr = err
		if !r.policy.IsRetryable(err) {
			return err
		}
	}
	return fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, lastErr)
}

func (r *RetryableClient) ReadBit(address string) (bool, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return false, err
	}
	if !pa.IsBit {
		return false, fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}

	var result bool
	var execErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		result, execErr = r.client.readBit(pa.AreaCode, pa.Address, pa.BitNo)
		if execErr == nil {
			return result, nil
		}
		if !r.policy.IsRetryable(execErr) {
			return false, execErr
		}
	}
	return false, fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, execErr)
}

func (r *RetryableClient) WriteBit(address string, value bool) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if !pa.IsBit {
		return fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}

	var lastErr error
	for attempt := 0; attempt <= r.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(r.policy.GetDelay(attempt - 1))
		}
		err := r.client.writeBit(pa.AreaCode, pa.Address, pa.BitNo, value)
		if err == nil {
			return nil
		}
		lastErr = err
		if !r.policy.IsRetryable(err) {
			return err
		}
	}
	return fmt.Errorf("重试%d次后失败: %w", r.policy.MaxRetries, lastErr)
}
