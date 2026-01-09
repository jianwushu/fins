# 自动重连机制使用指南

## 概述

FINS协议库提供了强大的自动重连机制,可以在以下情况下自动恢复连接:
- 🔌 **网络故障**: 网络中断、路由问题
- 🔄 **PLC重启**: 服务端重启或断电
- ⚠️ **连接超时**: 长时间无响应
- 💥 **读写失败**: 操作过程中连接断开

## 为什么需要重连机制?

### 问题场景

1. **生产环境稳定性**
   - PLC可能因维护而重启
   - 网络设备可能临时故障
   - 长时间运行可能出现连接异常

2. **无人值守系统**
   - 需要7x24小时运行
   - 无法及时人工干预
   - 必须自动恢复

3. **数据采集连续性**
   - 不能因短暂断开而丢失数据
   - 需要持续监控PLC状态
   - 保证业务连续性

## 核心功能

### 1. 自动重连
- ✅ 检测到连接断开时自动重连
- ✅ 读写操作失败时自动重连
- ✅ 指数退避算法避免网络拥塞
- ✅ 可配置最大重试次数

### 2. 健康检查
- ✅ 定期检查连接状态
- ✅ 主动发现连接问题
- ✅ 可配置检查间隔

### 3. 事件回调
- ✅ 重连成功回调
- ✅ 断开连接回调
- ✅ 便于监控和日志记录

### 4. 统计信息
- ✅ 重连次数统计
- ✅ 最后重连时间
- ✅ 重连状态查询

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/yourusername/fins"
)

func main() {
    // 1. 创建普通客户端
    config := fins.DefaultConfig("192.168.1.10")
    config.ServerNode = 0x64
    client, _ := fins.NewClient(config, true) // TCP模式
    
    // 2. 创建重连策略(使用默认配置)
    policy := fins.DefaultReconnectPolicy()
    
    // 3. 创建支持重连的客户端
    reconnectClient := fins.NewReconnectableClient(client, policy)
    
    // 4. 连接
    reconnectClient.Connect()
    defer reconnectClient.Close()
    
    // 5. 正常使用,自动处理重连
    value, _ := reconnectClient.ReadDWord(100)
    fmt.Println(value)
}
```

## 重连策略配置

### ReconnectPolicy 结构

```go
type ReconnectPolicy struct {
    EnableAutoReconnect  bool          // 是否启用自动重连
    MaxReconnectAttempts int           // 最大重连次数(0=无限)
    InitialDelay         time.Duration // 初始延迟
    MaxDelay             time.Duration // 最大延迟
    BackoffFactor        float64       // 退避因子
    ReconnectOnError     bool          // 读写错误时是否重连
    HealthCheckInterval  time.Duration // 健康检查间隔(0=禁用)
}
```

### 默认策略

```go
policy := fins.DefaultReconnectPolicy()
// EnableAutoReconnect:  true
// MaxReconnectAttempts: 0 (无限重试)
// InitialDelay:         1秒
// MaxDelay:             30秒
// BackoffFactor:        2.0
// ReconnectOnError:     true
// HealthCheckInterval:  10秒
```

### 自定义策略

```go
// 保守策略 - 适合稳定网络
conservativePolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  true,
    MaxReconnectAttempts: 5,              // 最多重试5次
    InitialDelay:         2 * time.Second,
    MaxDelay:             10 * time.Second,
    BackoffFactor:        1.5,
    ReconnectOnError:     true,
    HealthCheckInterval:  30 * time.Second, // 30秒检查一次
}

// 激进策略 - 适合不稳定网络
aggressivePolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  true,
    MaxReconnectAttempts: 0,              // 无限重试
    InitialDelay:         500 * time.Millisecond,
    MaxDelay:             60 * time.Second,
    BackoffFactor:        2.5,
    ReconnectOnError:     true,
    HealthCheckInterval:  5 * time.Second, // 5秒检查一次
}

// 禁用自动重连
noReconnectPolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  false,
    ReconnectOnError:     false,
    HealthCheckInterval:  0, // 禁用健康检查
}
```

## 高级用法

### 1. 设置回调函数

```go
reconnectClient := fins.NewReconnectableClient(client, policy)

// 重连成功回调
reconnectClient.SetOnReconnect(func() {
    fmt.Println("✅ 重连成功!")
    // 可以在这里:
    // - 记录日志
    // - 发送通知
    // - 重新初始化状态
})

// 断开连接回调
reconnectClient.SetOnDisconnect(func() {
    fmt.Println("❌ 连接断开!")
    // 可以在这里:
    // - 记录日志
    // - 发送告警
    // - 保存当前状态
})
```

### 2. 监控连接状态

```go
// 检查是否已连接
if reconnectClient.IsConnected() {
    fmt.Println("已连接")
}

// 检查是否正在重连
if reconnectClient.IsReconnecting() {
    fmt.Println("正在重连中...")
}

// 获取重连统计
count := reconnectClient.GetReconnectCount()
lastTime := reconnectClient.GetLastReconnectTime()
fmt.Printf("重连次数: %d, 最后重连: %s\n", count, lastTime)
```

### 3. 完整监控示例

```go
func monitorConnection(client *fins.ReconnectableClient) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        connected := client.IsConnected()
        reconnecting := client.IsReconnecting()
        
        if reconnecting {
            fmt.Println("🟡 状态: 重连中...")
        } else if connected {
            fmt.Println("🟢 状态: 已连接")
        } else {
            fmt.Println("🔴 状态: 未连接")
        }
        
        stats := client.GetStats()
        fmt.Printf("   请求: %d, 成功: %d, 错误: %d\n",
            stats.TotalRequests, stats.SuccessCount, stats.ErrorCount)
    }
}
```

## 工作原理

### 重连触发条件

1. **健康检查触发**
   ```
   定期检查 → 发现断开 → 启动重连
   ```

2. **读写错误触发**
   ```
   执行操作 → 检测错误 → 判断是否连接错误 → 启动重连 → 重试操作
   ```

### 重连流程

```
1. 检测到连接断开
   ↓
2. 调用 onDisconnect 回调
   ↓
3. 关闭旧连接
   ↓
4. 等待初始延迟
   ↓
5. 尝试重连
   ├─ 成功 → 调用 onReconnect 回调 → 完成
   └─ 失败 → 增加延迟(指数退避) → 返回步骤4
```

### 指数退避算法

```
第1次: 1秒
第2次: 2秒  (1 × 2.0)
第3次: 4秒  (2 × 2.0)
第4次: 8秒  (4 × 2.0)
第5次: 16秒 (8 × 2.0)
第6次: 30秒 (达到最大延迟)
第7次: 30秒
...
```

## 最佳实践

### 1. 选择合适的策略

```go
// 生产环境 - 平衡策略
productionPolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  true,
    MaxReconnectAttempts: 0,              // 无限重试
    InitialDelay:         1 * time.Second,
    MaxDelay:             30 * time.Second,
    BackoffFactor:        2.0,
    ReconnectOnError:     true,
    HealthCheckInterval:  10 * time.Second,
}
```

### 2. 添加日志记录

```go
reconnectClient.SetOnReconnect(func() {
    log.Printf("[%s] 重连成功", time.Now().Format("2006-01-02 15:04:05"))
})

reconnectClient.SetOnDisconnect(func() {
    log.Printf("[%s] 连接断开", time.Now().Format("2006-01-02 15:04:05"))
})
```

### 3. 监控重连次数

```go
// 定期检查重连次数,如果过高可能需要人工介入
if reconnectClient.GetReconnectCount() > 100 {
    log.Println("警告: 重连次数过多,请检查网络或PLC状态")
}
```

## 注意事项

1. **无限重试风险**
   - `MaxReconnectAttempts = 0` 会无限重试
   - 适合需要高可用的场景
   - 但可能掩盖严重的配置错误

2. **健康检查开销**
   - 检查间隔太短会增加网络负担
   - 建议设置为 5-30 秒

3. **回调函数性能**
   - 回调函数应快速执行
   - 避免阻塞重连流程

4. **并发安全**
   - 重连机制是线程安全的
   - 可以在多个goroutine中使用

## 完整示例

参考 `examples/reconnect_example.go` 查看完整的使用示例。

## 总结

✅ **自动重连机制是生产环境必备功能**  
✅ **提高系统稳定性和可用性**  
✅ **减少人工干预,降低运维成本**  
✅ **灵活配置,适应不同场景**

