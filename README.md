# FINS Protocol Library for Go

欧姆龙PLC FINS协议的Go语言实现库，支持TCP和UDP两种传输方式。

## 特性

- ✅ 支持FINS TCP协议
- ✅ 支持FINS UDP协议
- ✅ 完整的内存区域读写操作
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
    
    // 读取D100寄存器
    value, err := client.ReadDWord(100)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("D100 = %d\n", value)
    
    // 写入D100寄存器
    if err := client.WriteDWord(100, 1234); err != nil {
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
// 读取单个D区寄存器
value, err := client.ReadDWord(100)

// 批量读取D区寄存器
values, err := client.ReadDWords(100, 10)

// 读取CIO区位
bit, err := client.ReadCIOBit(0, 0)

// 读取通用内存区域
data, err := client.ReadMemoryArea(fins.MemAreaD, 100, 5)
```

### 写入操作

```go
// 写入单个D区寄存器
err := client.WriteDWord(100, 1234)

// 批量写入D区寄存器
values := []uint16{100, 200, 300}
err := client.WriteDWords(100, values)

// 写入CIO区位
err := client.WriteCIOBit(0, 0, true)

// 写入通用内存区域
err := client.WriteMemoryArea(fins.MemAreaD, 100, values)
```

### 字节数组操作

```go
// 读取字节数组
data, err := client.ReadDBytes(100, 10)  // 读取D100开始的10个字节
fmt.Printf("读取数据: % X\n", data)

// 写入字节数组
writeData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
err := client.WriteDBytes(200, writeData)

// 不同内存区域的字节操作
cioData, err := client.ReadCIOBytes(0, 8)   // 读取CIO区
err = client.WriteHRBytes(0, []byte{0xAA, 0xBB})  // 写入HR区
err = client.WriteWRBytes(0, []byte{0x11, 0x22})  // 写入WR区

// 字节数组与整数转换
value := uint32(0x12345678)
buf := make([]byte, 4)
binary.BigEndian.PutUint32(buf, value)
err = client.WriteDBytes(300, buf)

// 字符串操作
text := "FINS Protocol"
err = client.WriteDBytes(400, []byte(text))
readData, _ := client.ReadDBytes(400, uint16(len(text)))
readText := string(readData)
```

### 重试机制

```go
// 创建自定义重试策略
retryPolicy := &fins.RetryPolicy{
    MaxRetries:    5,
    InitialDelay:  200 * time.Millisecond,
    MaxDelay:      3 * time.Second,
    BackoffFactor: 2.0,
}

// 创建支持重试的客户端
retryClient := fins.NewRetryableClient(client, retryPolicy)

// 使用重试客户端
data, err := retryClient.ReadMemoryArea(fins.MemAreaD, 100, 1)
```

### 自动重连机制 (新增)

```go
// 创建TCP客户端
client, _ := fins.NewClient(config, true)

// 创建重连策略
reconnectPolicy := &fins.ReconnectPolicy{
    EnableAutoReconnect:  true,              // 启用自动重连
    MaxReconnectAttempts: 0,                 // 0 = 无限重试
    InitialDelay:         1 * time.Second,   // 初始延迟
    MaxDelay:             30 * time.Second,  // 最大延迟
    BackoffFactor:        2.0,               // 退避因子
    ReconnectOnError:     true,              // 读写错误时重连
    HealthCheckInterval:  10 * time.Second,  // 健康检查间隔
}

// 创建支持自动重连的客户端
reconnectClient := fins.NewReconnectableClient(client, reconnectPolicy)

// 设置回调
reconnectClient.SetOnReconnect(func() {
    fmt.Println("✅ 重连成功!")
})

reconnectClient.SetOnDisconnect(func() {
    fmt.Println("❌ 连接断开!")
})

// 连接
reconnectClient.Connect()

// 正常使用,连接断开时会自动重连
value, _ := reconnectClient.ReadDWord(100)

// 查看重连统计
count := reconnectClient.GetReconnectCount()
fmt.Printf("重连次数: %d\n", count)
```

## 内存区域代码

| 代码 | 常量 | 说明 |
|-----|------|------|
| 0x30 | MemAreaCIO | CIO区 - 输入输出继电器 |
| 0x31 | MemAreaWR | WR区 - 工作继电器 |
| 0x32 | MemAreaHR | HR区 - 保持继电器 |
| 0x82 | MemAreaD | D区 - 数据寄存器 |
| 0x89 | MemAreaT | T区 - 定时器当前值 |
| 0x8C | MemAreaC | C区 - 计数器当前值 |

## 配置选项

```go
config := &fins.FinsClientConfig{
    IP:             "192.168.1.10",
    Port:           9600,
    LocalNode:      0x01,
    ServerNode:     0x64,
    Timeout:        5 * time.Second,
    RetryCount:     3,
    SIDMode:        fins.SIDFixed,
    FixedSID:       0x00,
}
```

## 示例

查看 `examples/` 目录获取更多示例：

- `udp_example.go` - UDP模式完整示例
- `tcp_example.go` - TCP模式完整示例
- `retry_example.go` - 重试机制示例
- `bytes_example.go` - 字节数组操作示例
- `reconnect_example.go` - 自动重连示例 (新增)

## 文档

- [快速入门](QUICKSTART.md) - 快速上手指南
- [API参考](API_REFERENCE.md) - 完整API文档
- [自动重连指南](docs/RECONNECT_GUIDE.md) - 重连机制详细说明 (新增)
- [协议规范](docs/FINS_PROTOCOL_SPEC.md) - FINS协议详细规范
- [项目结构](PROJECT_STRUCTURE.md) - 项目文件组织说明
- [更新日志](CHANGELOG.md) - 功能更新记录

## 许可证

MIT License

