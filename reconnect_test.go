package fins

import (
	"testing"
	"time"
)

func TestDefaultReconnectPolicy(t *testing.T) {
	policy := DefaultReconnectPolicy()

	if !policy.EnableAutoReconnect {
		t.Error("默认策略应启用自动重连")
	}

	if policy.MaxReconnectAttempts != 0 {
		t.Errorf("默认最大重连次数应为0(无限), got %d", policy.MaxReconnectAttempts)
	}

	if policy.InitialDelay != 1*time.Second {
		t.Errorf("默认初始延迟应为1秒, got %v", policy.InitialDelay)
	}

	if policy.MaxDelay != 30*time.Second {
		t.Errorf("默认最大延迟应为30秒, got %v", policy.MaxDelay)
	}

	if policy.BackoffFactor != 2.0 {
		t.Errorf("默认退避因子应为2.0, got %f", policy.BackoffFactor)
	}

	if !policy.ReconnectOnError {
		t.Error("默认应在错误时重连")
	}

	if policy.HealthCheckInterval != 10*time.Second {
		t.Errorf("默认健康检查间隔应为10秒, got %v", policy.HealthCheckInterval)
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil错误", nil, false},
		{"连接已关闭", ErrConnectionClosed, true},
		{"超时错误", ErrTimeout, true},
		{"无效帧", ErrInvalidFrame, false},
		{"无效地址", ErrInvalidAddress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("isConnectionError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"包含子串", "connection refused", "refused", true},
		{"不包含子串", "connection refused", "accepted", false},
		{"完全匹配", "EOF", "EOF", true},
		{"空字符串", "", "test", false},
		{"子串为空", "test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestReconnectableClientCreation(t *testing.T) {
	config := DefaultConfig("192.168.1.10")
	client, err := NewClient(config, false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 使用默认策略
	rc1 := NewReconnectableClient(client, nil)
	if rc1 == nil {
		t.Error("创建重连客户端失败")
	}

	if rc1.policy == nil {
		t.Error("应使用默认策略")
	}

	// 使用自定义策略
	customPolicy := &ReconnectPolicy{
		EnableAutoReconnect:  false,
		MaxReconnectAttempts: 5,
		InitialDelay:         2 * time.Second,
		MaxDelay:             60 * time.Second,
		BackoffFactor:        1.5,
		ReconnectOnError:     false,
		HealthCheckInterval:  0,
	}

	rc2 := NewReconnectableClient(client, customPolicy)
	if rc2 == nil {
		t.Error("创建重连客户端失败")
	}

	if rc2.policy.MaxReconnectAttempts != 5 {
		t.Errorf("自定义策略未生效, got %d", rc2.policy.MaxReconnectAttempts)
	}
}

func TestReconnectableClientCallbacks(t *testing.T) {
	config := DefaultConfig("192.168.1.10")
	client, _ := NewClient(config, false)
	rc := NewReconnectableClient(client, nil)

	rc.SetOnReconnect(func() {
		// 重连回调
	})

	rc.SetOnDisconnect(func() {
		// 断开回调
	})

	// 验证回调已设置
	if rc.onReconnect == nil {
		t.Error("重连回调未设置")
	}

	if rc.onDisconnect == nil {
		t.Error("断开回调未设置")
	}
}

func TestReconnectableClientStats(t *testing.T) {
	config := DefaultConfig("192.168.1.10")
	client, _ := NewClient(config, false)
	rc := NewReconnectableClient(client, nil)

	// 初始状态
	if rc.GetReconnectCount() != 0 {
		t.Errorf("初始重连次数应为0, got %d", rc.GetReconnectCount())
	}

	if rc.IsReconnecting() {
		t.Error("初始状态不应处于重连中")
	}

	lastReconnect := rc.GetLastReconnectTime()
	if !lastReconnect.IsZero() {
		t.Error("初始最后重连时间应为零值")
	}
}
