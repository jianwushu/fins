package fins

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

// ReconnectPolicy 重连策略
type ReconnectPolicy struct {
	EnableAutoReconnect  bool          // 是否启用自动重连
	MaxReconnectAttempts int           // 最大重连尝试次数 (0表示无限重试)
	InitialDelay         time.Duration // 初始重连延迟
	MaxDelay             time.Duration // 最大重连延迟
	BackoffFactor        float64       // 退避因子
	ReconnectOnError     bool          // 读写错误时是否重连
	HealthCheckInterval  time.Duration // 健康检查间隔 (0表示禁用)
}

// DefaultReconnectPolicy 返回默认重连策略
func DefaultReconnectPolicy() *ReconnectPolicy {
	return &ReconnectPolicy{
		EnableAutoReconnect:  true,
		MaxReconnectAttempts: 0,
		InitialDelay:         1 * time.Second,
		MaxDelay:             30 * time.Second,
		BackoffFactor:        2.0,
		ReconnectOnError:     true,
		HealthCheckInterval:  10 * time.Second,
	}
}

// ReconnectableClient 支持自动重连的客户端包装器
type ReconnectableClient struct {
	client          *FinsClient
	policy          *ReconnectPolicy
	mutex           sync.RWMutex
	reconnecting    bool
	reconnectCount  int
	lastReconnectAt time.Time
	stopHealthCheck chan struct{}
	onReconnect     func()
	onDisconnect    func()
}

// NewReconnectableClient 创建支持自动重连的客户端
func NewReconnectableClient(client *FinsClient, policy *ReconnectPolicy) *ReconnectableClient {
	if policy == nil {
		policy = DefaultReconnectPolicy()
	}

	rc := &ReconnectableClient{
		client:          client,
		policy:          policy,
		stopHealthCheck: make(chan struct{}),
	}

	if policy.HealthCheckInterval > 0 {
		go rc.healthCheckLoop()
	}

	return rc
}

// SetOnReconnect 设置重连成功回调
func (rc *ReconnectableClient) SetOnReconnect(callback func()) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	rc.onReconnect = callback
}

// SetOnDisconnect 设置断开连接回调
func (rc *ReconnectableClient) SetOnDisconnect(callback func()) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	rc.onDisconnect = callback
}

// Connect 连接到PLC
func (rc *ReconnectableClient) Connect() error {
	return rc.client.Connect()
}

// Close 关闭连接
func (rc *ReconnectableClient) Close() error {
	close(rc.stopHealthCheck)
	return rc.client.Close()
}

// IsConnected 检查是否已连接
func (rc *ReconnectableClient) IsConnected() bool {
	return rc.client.IsConnected()
}

// GetStats 获取统计信息
func (rc *ReconnectableClient) GetStats() ConnectionStats {
	return rc.client.GetStats()
}

// GetReconnectCount 获取重连次数
func (rc *ReconnectableClient) GetReconnectCount() int {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	return rc.reconnectCount
}

// GetLastReconnectTime 获取最后重连时间
func (rc *ReconnectableClient) GetLastReconnectTime() time.Time {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	return rc.lastReconnectAt
}

// IsReconnecting 是否正在重连
func (rc *ReconnectableClient) IsReconnecting() bool {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	return rc.reconnecting
}

// ========== 对外统一 API（字符串地址，带自动重连） ==========

func (rc *ReconnectableClient) ReadWord(address string) (uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return 0, err
	}
	if pa.IsBit {
		return 0, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var data []byte
	err = rc.executeWithReconnect(func() error {
		v, execErr := rc.client.readMemoryArea(pa.AreaCode, pa.Address, 1)
		data = v
		return execErr
	})
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, ErrInvalidResponse
	}
	return binary.BigEndian.Uint16(data[0:2]), nil
}

func (rc *ReconnectableClient) ReadWords(address string, count uint16) ([]uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var data []byte
	err = rc.executeWithReconnect(func() error {
		v, execErr := rc.client.readMemoryArea(pa.AreaCode, pa.Address, count)
		data = v
		return execErr
	})
	if err != nil {
		return nil, err
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

func (rc *ReconnectableClient) WriteWord(address string, value uint16) error {
	return rc.WriteWords(address, []uint16{value})
}

func (rc *ReconnectableClient) WriteWords(address string, values []uint16) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return rc.executeWithReconnect(func() error {
		return rc.client.writeMemoryArea(pa.AreaCode, pa.Address, values)
	})
}

func (rc *ReconnectableClient) ReadBytes(address string, byteCount uint16) ([]byte, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	var result []byte
	err = rc.executeWithReconnect(func() error {
		v, execErr := rc.client.readBytes(pa.AreaCode, pa.Address, byteCount)
		result = v
		return execErr
	})
	return result, err
}

func (rc *ReconnectableClient) WriteBytes(address string, data []byte) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return rc.executeWithReconnect(func() error {
		return rc.client.writeBytes(pa.AreaCode, pa.Address, data)
	})
}

func (rc *ReconnectableClient) ReadBit(address string) (bool, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return false, err
	}
	if !pa.IsBit {
		return false, fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}

	var result bool
	err = rc.executeWithReconnect(func() error {
		v, execErr := rc.client.readBit(pa.AreaCode, pa.Address, pa.BitNo)
		result = v
		return execErr
	})
	return result, err
}

func (rc *ReconnectableClient) WriteBit(address string, value bool) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if !pa.IsBit {
		return fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}
	return rc.executeWithReconnect(func() error {
		return rc.client.writeBit(pa.AreaCode, pa.Address, pa.BitNo, value)
	})
}

// ========== 自动重连内部实现 ==========

func (rc *ReconnectableClient) reconnect() error {
	rc.mutex.Lock()
	if rc.reconnecting {
		rc.mutex.Unlock()
		return fmt.Errorf("正在重连中")
	}
	rc.reconnecting = true
	rc.mutex.Unlock()

	defer func() {
		rc.mutex.Lock()
		rc.reconnecting = false
		rc.mutex.Unlock()
	}()

	if rc.onDisconnect != nil {
		rc.onDisconnect()
	}

	_ = rc.client.Close()

	delay := rc.policy.InitialDelay
	attempts := 0

	for {
		attempts++
		rc.reconnectCount++

		if rc.policy.MaxReconnectAttempts > 0 && attempts > rc.policy.MaxReconnectAttempts {
			return fmt.Errorf("重连失败: 超过最大重试次数 %d", rc.policy.MaxReconnectAttempts)
		}

		fmt.Printf("[重连] 第 %d 次尝试重连...\n", attempts)

		err := rc.client.Connect()
		if err == nil {
			rc.lastReconnectAt = time.Now()
			fmt.Printf("[重连] 重连成功!\n")
			if rc.onReconnect != nil {
				rc.onReconnect()
			}
			return nil
		}

		fmt.Printf("[重连] 重连失败: %v, %v 后重试\n", err, delay)
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * rc.policy.BackoffFactor)
		if delay > rc.policy.MaxDelay {
			delay = rc.policy.MaxDelay
		}
	}
}

func (rc *ReconnectableClient) healthCheckLoop() {
	ticker := time.NewTicker(rc.policy.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !rc.client.IsConnected() {
				fmt.Println("[健康检查] 检测到连接断开，尝试重连...")
				if rc.policy.EnableAutoReconnect {
					go func() { _ = rc.reconnect() }()
				}
			}
		case <-rc.stopHealthCheck:
			return
		}
	}
}

func (rc *ReconnectableClient) executeWithReconnect(operation func() error) error {
	err := operation()
	if err != nil && rc.policy.ReconnectOnError && rc.policy.EnableAutoReconnect {
		if isConnectionError(err) {
			fmt.Printf("[自动重连] 检测到连接错误: %v\n", err)
			if reconnectErr := rc.reconnect(); reconnectErr != nil {
				return fmt.Errorf("重连失败: %w, 原始错误: %v", reconnectErr, err)
			}
			return operation()
		}
	}
	return err
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"EOF",
		"连接已关闭",
		"发送失败",
		"读取失败",
	}

	for _, connErr := range connectionErrors {
		if contains(errStr, connErr) {
			return true
		}
	}

	return err == ErrConnectionClosed || err == ErrTimeout
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
