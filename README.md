# FINS Protocol Library for Go

欧姆龙PLC FINS协议的Go语言实现库，支持TCP和UDP两种传输方式。

## 特性

- ✅ 支持FINS TCP协议
- ✅ 支持FINS UDP协议
- ✅ 完整的内存区域读写操作（**统一字符串地址 API：如 `D100`、`CIO0.00`**）
- ✅ 支持位操作和字操作
- ✅ 字节数组操作支持
- ✅ 灵活的SID模式（固定/递增）
- ✅ 自动重试机制
- ✅ **自动重连机制** (新增)
- ✅ 健康检查和连接监控
- ✅ 连接统计信息
- ✅ 线程安全设计
- ✅ 详细的错误处理

## 安装

```bash
go get github.com/yourusername/fins
```

## 快速开始

### UDP模式

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/fins"
)

func main() {
    // 创建配置
    config := fins.DefaultConfig("192.168.1.10")
    config.LocalNode = 0x01
    config.ServerNode = 0x64

    // 创建UDP客户端
    client, err := fins.NewClient(config, false)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 连接
    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // 读取 D100
    value, err := client.ReadWord("D100")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("D100 = %d\n", value)

    // 写入 D100
    if err := client.WriteWord("D100", 1234); err != nil {
        log.Fatal(err)
    }
}
```

### TCP模式

```go
// 创建TCP客户端（第二个参数为true）
client, err := fins.NewClient(config, true)
```

## 主要功能

### 读取操作

```go
// 读取单个 word
value, err := client.ReadWord("D100")

// 批量读取 word（起始地址 + 数量）
values, err := client.ReadWords("D100", 10)

// 读取 bit
bit, err := client.ReadBit("CIO0.00")

// 读取字节数组（按字对齐，内部自动处理奇数字节）
data, err := client.ReadBytes("D100", 10)
```

### 写入操作

```go
// 写入单个 word
err := client.WriteWord("D100", 1234)

// 批量写入 word
values := []uint16{100, 200, 300}
err := client.WriteWords("D200", values)

// 写入 bit
err := client.WriteBit("CIO0.00", true)

// 写入字节数组
err := client.WriteBytes("D200", []byte{0x01, 0x02, 0x03})
```

### 重试机制

```go
retryPolicy := &fins.RetryPolicy{
    MaxRetries:    5,
    InitialDelay:  200 * time.Millisecond,
    MaxDelay:      3 * time.Second,
    BackoffFactor: 2.0,
}

retryClient := fins.NewRetryableClient(client, retryPolicy)
value, err := retryClient.ReadWord("D100")
```

### 自动重连机制 (新增)

```go
client, _ := fins.NewClient(config, true)

reconnectPolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  true,
    MaxReconnectAttempts: 0,
    InitialDelay:         1 * time.Second,
    MaxDelay:             30 * time.Second,
    BackoffFactor:        2.0,
    ReconnectOnError:     true,
    HealthCheckInterval:  10 * time.Second,
}

reconnectClient := fins.NewReconnectableClient(client, reconnectPolicy)
reconnectClient.SetOnReconnect(func() {
    fmt.Println("✅ 重连成功!")
})
reconnectClient.SetOnDisconnect(func() {
    fmt.Println("❌ 连接断开!")
})

reconnectClient.Connect()
value, _ := reconnectClient.ReadWord("D100")
```

## 地址格式

- Word 地址：`D100`、`CIO0`、`WR200`、`HR300`、`A0`、`T0`、`C0`
- Bit 地址：`CIO0.00`、`WR10.15`、`HR200.01`、`A5.0`

约束：
- bit 仅支持 `CIO/WR/HR/A`
- bit 位号范围 `0~15`

## 示例

查看 `examples/` 目录获取更多示例：

- `udp_example.go` - UDP模式完整示例
- `tcp_example.go` - TCP模式完整示例
- `retry_example.go` - 重试机制示例
- `bytes_example.go` - 字节数组操作示例
- `reconnect_example.go` - 自动重连示例 (新增)

## 文档

- [快速入门](QUICKSTART.md) - 快速上手指南
