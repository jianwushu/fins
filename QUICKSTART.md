# FINS协议库快速入门指南

## 1. 安装

```bash
go get github.com/yourusername/fins
```

## 2. 最简单的例子 (UDP模式)

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/fins"
)

func main() {
    // 1. 创建配置
    config := fins.DefaultConfig("192.168.1.10")
    config.ServerNode = 0x64  // PLC的FINS节点地址
    
    // 2. 创建客户端 (false = UDP模式)
    client, err := fins.NewClient(config, false)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 3. 连接
    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }
    
    // 4. 读取D100寄存器
    value, err := client.ReadDWord(100)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("D100 = %d\n", value)
    
    // 5. 写入D100寄存器
    if err := client.WriteDWord(100, 1234); err != nil {
        log.Fatal(err)
    }
    fmt.Println("写入成功!")
}
```

## 3. TCP模式

只需将第二个参数改为 `true`:

```go
client, err := fins.NewClient(config, true)  // true = TCP模式
```

## 4. 常用操作

### 读取单个寄存器
```go
value, err := client.ReadDWord(100)  // 读取D100
```

### 批量读取
```go
values, err := client.ReadDWords(100, 10)  // 读取D100-D109
for i, v := range values {
    fmt.Printf("D%d = %d\n", 100+i, v)
}
```

### 写入单个寄存器
```go
err := client.WriteDWord(100, 1234)  // 写入D100
```

### 批量写入
```go
values := []uint16{100, 200, 300, 400, 500}
err := client.WriteDWords(200, values)  // 写入D200-D204
```

### 读取位
```go
bit, err := client.ReadCIOBit(0, 0)  // 读取CIO0.00
fmt.Printf("CIO0.00 = %v\n", bit)
```

### 写入位
```go
err := client.WriteCIOBit(0, 0, true)  // 写入CIO0.00 = ON
```

### 读取字节数组
```go
data, err := client.ReadDBytes(100, 10)  // 读取D100开始的10个字节
fmt.Printf("读取数据: % X\n", data)
```

### 写入字节数组
```go
writeData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
err := client.WriteDBytes(200, writeData)  // 写入到D200
```

### 字节数组与整数转换
```go
// 写入32位整数
value := uint32(0x12345678)
buf := make([]byte, 4)
binary.BigEndian.PutUint32(buf, value)
err := client.WriteDBytes(300, buf)

// 读取并转换回整数
readData, _ := client.ReadDBytes(300, 4)
readValue := binary.BigEndian.Uint32(readData)
```

### 字符串操作
```go
// 写入字符串
text := "FINS Protocol"
err := client.WriteDBytes(400, []byte(text))

// 读取字符串
readData, _ := client.ReadDBytes(400, uint16(len(text)))
readText := string(readData)
```

## 5. 配置选项

```go
config := &fins.FinsClientConfig{
    IP:         "192.168.1.10",      // PLC IP地址
    Port:       9600,                 // 端口(默认9600)
    LocalNode:  0x01,                 // 本地节点地址
    ServerNode: 0x64,                 // PLC节点地址
    Timeout:    5 * time.Second,      // 超时时间
    SIDMode:    fins.SIDFixed,        // SID模式
    FixedSID:   0x00,                 // 固定SID值
}
```

## 6. 使用重试机制

```go
// 创建重试策略
retryPolicy := &fins.RetryPolicy{
    MaxRetries:    3,                      // 最多重试3次
    InitialDelay:  200 * time.Millisecond, // 初始延迟
    MaxDelay:      3 * time.Second,        // 最大延迟
    BackoffFactor: 2.0,                    // 退避因子
}

// 创建支持重试的客户端
retryClient := fins.NewRetryableClient(client, retryPolicy)

// 使用重试客户端
data, err := retryClient.ReadMemoryArea(fins.MemAreaD, 100, 1)
```

## 7. 内存区域代码

| 常量 | 值 | 说明 |
|-----|---|------|
| `fins.MemAreaCIO` | 0x30 | CIO区 - 输入输出继电器 |
| `fins.MemAreaWR` | 0x31 | WR区 - 工作继电器 |
| `fins.MemAreaHR` | 0x32 | HR区 - 保持继电器 |
| `fins.MemAreaD` | 0x82 | D区 - 数据寄存器 |
| `fins.MemAreaT` | 0x89 | T区 - 定时器 |
| `fins.MemAreaC` | 0x8C | C区 - 计数器 |

## 8. 错误处理

```go
value, err := client.ReadDWord(100)
if err != nil {
    if err == fins.ErrTimeout {
        fmt.Println("操作超时")
    } else if err == fins.ErrConnectionClosed {
        fmt.Println("连接已关闭")
    } else {
        fmt.Printf("错误: %v\n", err)
    }
    return
}
```

## 9. 查看统计信息

```go
stats := client.GetStats()
fmt.Printf("总请求数: %d\n", stats.TotalRequests)
fmt.Printf("成功次数: %d\n", stats.SuccessCount)
fmt.Printf("错误次数: %d\n", stats.ErrorCount)
fmt.Printf("超时次数: %d\n", stats.TimeoutCount)
```

## 10. 完整示例

查看 `examples/` 目录:
- `udp_example.go` - UDP完整示例
- `tcp_example.go` - TCP完整示例
- `retry_example.go` - 重试机制示例
- `bytes_example.go` - 字节数组操作示例

## 11. 运行测试

```bash
# 运行单元测试
go test -v

# 运行示例
go run examples/udp_example.go
```

## 12. 常见问题

### Q: 如何获取PLC的节点地址?
A: 在PLC的网络设置中查看FINS节点号,通常在1-254之间。

### Q: TCP和UDP应该选择哪个?
A: 
- TCP: 可靠性高,适合重要数据传输
- UDP: 延迟低,适合实时性要求高的场景

### Q: 超时时间应该设置多少?
A: 建议3-5秒,根据网络状况和PLC响应速度调整。

### Q: 可以并发请求吗?
A: 可以,但建议使用SID递增模式:
```go
config.SIDMode = fins.SIDIncrement
```

## 13. 更多信息

- 详细协议文档: `docs/FINS_PROTOCOL_SPEC.md`
- 项目结构说明: `PROJECT_STRUCTURE.md`
- 完整API文档: `README.md`

