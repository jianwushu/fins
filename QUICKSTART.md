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

    // 4. 读取 D100
    value, err := client.ReadWord("D100")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("D100 = %d\n", value)

    // 5. 写入 D100
    if err := client.WriteWord("D100", 1234); err != nil {
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

## 4. 常用操作（统一字符串地址）

### 读取单个寄存器（word）
```go
value, err := client.ReadWord("D100")
```

### 批量读取（word）
```go
values, err := client.ReadWords("D100", 10)  // D100-D109
for i, v := range values {
    fmt.Printf("D%d = %d\n", 100+i, v)
}
```

### 写入单个寄存器（word）
```go
err := client.WriteWord("D100", 1234)
```

### 批量写入（word）
```go
values := []uint16{100, 200, 300, 400, 500}
err := client.WriteWords("D200", values)
```

### 读取位（bit）
```go
bit, err := client.ReadBit("CIO0.00")
fmt.Printf("CIO0.00 = %v\n", bit)
```

### 写入位（bit）
```go
err := client.WriteBit("CIO0.00", true)
```

### 读取字节数组
```go
data, err := client.ReadBytes("D100", 10)  // 从D100开始读10个字节
fmt.Printf("读取数据: % X\n", data)
```

### 写入字节数组
```go
writeData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
err := client.WriteBytes("D200", writeData)
```

## 5. 地址格式

- Word 地址：`D100`、`CIO0`、`WR200`、`HR300`、`A0`、`T0`、`C0`
- Bit 地址：`CIO0.00`、`WR10.15`、`HR200.01`、`A5.0`

约束：
- bit 仅支持 `CIO/WR/HR/A`
- bit 位号范围 `0~15`

## 6. 使用重试机制

```go
retryPolicy := &fins.RetryPolicy{
    MaxRetries:    3,
    InitialDelay:  200 * time.Millisecond,
    MaxDelay:      3 * time.Second,
    BackoffFactor: 2.0,
}

retryClient := fins.NewRetryableClient(client, retryPolicy)
value, err := retryClient.ReadWord("D100")
```

## 7. 运行测试

```bash
# 运行单元测试
go test -v

# 运行示例
go run examples/udp/udp_example.go
```
