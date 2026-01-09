package fins

import (
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
		MaxReconnectAttempts: 0, // 无限重试
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
	onReconnect     func() // 重连成功回调
	onDisconnect    func() // 断开连接回调
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

	// 启动健康检查
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

// reconnect 执行重连
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

	// 调用断开连接回调
	if rc.onDisconnect != nil {
		rc.onDisconnect()
	}

	// 关闭旧连接
	rc.client.Close()

	delay := rc.policy.InitialDelay
	attempts := 0

	for {
		attempts++
		rc.reconnectCount++

		// 检查是否超过最大重试次数
		if rc.policy.MaxReconnectAttempts > 0 && attempts > rc.policy.MaxReconnectAttempts {
			return fmt.Errorf("重连失败: 超过最大重试次数 %d", rc.policy.MaxReconnectAttempts)
		}

		fmt.Printf("[重连] 第 %d 次尝试重连...\n", attempts)

		// 尝试重连
		err := rc.client.Connect()
		if err == nil {
			rc.lastReconnectAt = time.Now()
			fmt.Printf("[重连] 重连成功!\n")

			// 调用重连成功回调
			if rc.onReconnect != nil {
				rc.onReconnect()
			}

			return nil
		}

		fmt.Printf("[重连] 重连失败: %v, %v 后重试\n", err, delay)

		// 等待后重试
		time.Sleep(delay)

		// 计算下次延迟(指数退避)
		delay = time.Duration(float64(delay) * rc.policy.BackoffFactor)
		if delay > rc.policy.MaxDelay {
			delay = rc.policy.MaxDelay
		}
	}
}

// healthCheckLoop 健康检查循环
func (rc *ReconnectableClient) healthCheckLoop() {
	ticker := time.NewTicker(rc.policy.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !rc.client.IsConnected() {
				fmt.Println("[健康检查] 检测到连接断开，尝试重连...")
				if rc.policy.EnableAutoReconnect {
					go rc.reconnect()
				}
			}
		case <-rc.stopHealthCheck:
			return
		}
	}
}

// executeWithReconnect 执行操作，失败时自动重连
func (rc *ReconnectableClient) executeWithReconnect(operation func() error) error {
	err := operation()
	if err != nil && rc.policy.ReconnectOnError && rc.policy.EnableAutoReconnect {
		// 判断是否是连接错误
		if isConnectionError(err) {
			fmt.Printf("[自动重连] 检测到连接错误: %v\n", err)

			// 尝试重连
			if reconnectErr := rc.reconnect(); reconnectErr != nil {
				return fmt.Errorf("重连失败: %w, 原始错误: %v", reconnectErr, err)
			}

			// 重连成功后重试操作
			return operation()
		}
	}
	return err
}

// isConnectionError 判断是否是连接错误
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查常见的连接错误
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

// contains 检查字符串是否包含子串
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

// ReadMemoryArea 读取内存区域(带自动重连)
func (rc *ReconnectableClient) ReadMemoryArea(areaCode byte, address uint16, count uint16) ([]byte, error) {
	var result []byte
	err := rc.executeWithReconnect(func() error {
		data, err := rc.client.ReadMemoryArea(areaCode, address, count)
		result = data
		return err
	})
	return result, err
}

// WriteMemoryArea 写入内存区域(带自动重连)
func (rc *ReconnectableClient) WriteMemoryArea(areaCode byte, address uint16, values []uint16) error {
	return rc.executeWithReconnect(func() error {
		return rc.client.WriteMemoryArea(areaCode, address, values)
	})
}

// ReadDWord 读取D区单个字(带自动重连)
func (rc *ReconnectableClient) ReadDWord(address uint16) (uint16, error) {
	var result uint16
	err := rc.executeWithReconnect(func() error {
		value, err := rc.client.ReadDWord(address)
		result = value
		return err
	})
	return result, err
}

// WriteDWord 写入D区单个字(带自动重连)
func (rc *ReconnectableClient) WriteDWord(address uint16, value uint16) error {
	return rc.executeWithReconnect(func() error {
		return rc.client.WriteDWord(address, value)
	})
}

// ReadDBytes 读取D区字节数组(带自动重连)
func (rc *ReconnectableClient) ReadDBytes(address uint16, byteCount uint16) ([]byte, error) {
	var result []byte
	err := rc.executeWithReconnect(func() error {
		data, err := rc.client.ReadDBytes(address, byteCount)
		result = data
		return err
	})
	return result, err
}

// WriteDBytes 写入D区字节数组(带自动重连)
func (rc *ReconnectableClient) WriteDBytes(address uint16, data []byte) error {
	return rc.executeWithReconnect(func() error {
		return rc.client.WriteDBytes(address, data)
	})
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

// GetStats 获取统计信息
func (rc *ReconnectableClient) GetStats() ConnectionStats {
	return rc.client.GetStats()
}

// IsConnected 检查是否已连接
func (rc *ReconnectableClient) IsConnected() bool {
	return rc.client.IsConnected()
}
