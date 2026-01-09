# FINS协议库项目结构

## 项目概述

这是一个完整的欧姆龙PLC FINS协议Go语言实现库,支持TCP和UDP两种传输方式。

## 文件结构

```
fins/
├── constants.go          # 协议常量定义(命令码、内存区域、错误码等)
├── types.go             # 核心数据类型定义
├── udp_frame.go         # UDP帧构建和解析
├── udp_client.go        # UDP客户端实现
├── tcp_frame.go         # TCP帧构建和解析
├── tcp_client.go        # TCP客户端实现
├── client.go            # 统一客户端接口和便捷方法
├── retry.go             # 重试机制实现
├── frame_test.go        # 单元测试
├── go.mod               # Go模块定义
├── README.md            # 使用文档
├── .gitignore           # Git忽略文件
├── docs/
│   └── FINS_PROTOCOL_SPEC.md  # FINS协议详细技术文档
└── examples/
    ├── udp_example.go   # UDP使用示例
    ├── tcp_example.go   # TCP使用示例
    └── retry_example.go # 重试机制示例
```

## 核心模块说明

### 1. 常量和类型定义

- **constants.go**: 定义所有协议常量
  - 命令码 (0x0101读取, 0x0102写入等)
  - 内存区域代码 (D区0x82, CIO区0x30等)
  - 错误码及错误消息映射
  - SID模式枚举

- **types.go**: 定义核心数据结构
  - `FinsClientConfig`: 客户端配置
  - `FinsUDPFrame`: UDP帧结构
  - `FinsTCPFrame`: TCP帧结构
  - `FinsResponse`: 响应结构
  - `ReadRequest/WriteRequest`: 读写请求参数

### 2. UDP协议实现

- **udp_frame.go**: UDP帧处理
  - `BuildUDPFrame()`: 构建UDP帧
  - `ParseUDPFrame()`: 解析UDP帧
  - `BuildReadMemoryRequest()`: 构建读取请求
  - `BuildWriteMemoryRequest()`: 构建写入请求

- **udp_client.go**: UDP客户端
  - 连接管理
  - 异步接收循环
  - SID管理(固定/递增模式)
  - 请求-响应匹配

### 3. TCP协议实现

- **tcp_frame.go**: TCP帧处理
  - `BuildTCPFrame()`: 构建TCP帧
  - `ParseTCPFrame()`: 解析TCP帧
  - `ReadTCPFrameFromConn()`: 从TCP流中读取完整帧

- **tcp_client.go**: TCP客户端
  - TCP连接管理
  - 帧边界识别
  - 异步接收循环

### 4. 统一客户端接口

- **client.go**: 高层API
  - `NewClient()`: 创建客户端(自动选择TCP/UDP)
  - `ReadDWord()/WriteDWord()`: D区操作
  - `ReadCIOBit()/WriteCIOBit()`: CIO位操作
  - `ReadMemoryArea()/WriteMemoryArea()`: 通用内存操作

### 5. 错误处理和重试

- **retry.go**: 重试机制
  - `RetryPolicy`: 重试策略配置
  - `RetryableClient`: 支持重试的客户端包装器
  - 指数退避算法
  - 可配置的重试条件

## 主要特性

### 1. 协议支持
- ✅ FINS TCP协议(魔数"FINS" + 20字节头部)
- ✅ FINS UDP协议(10字节头部)
- ✅ 内存读取命令(0x0101)
- ✅ 内存写入命令(0x0102)
- ✅ 位写入命令(0x0103)

### 2. 内存区域
- ✅ CIO区(0x30) - 输入输出继电器
- ✅ WR区(0x31) - 工作继电器
- ✅ HR区(0x32) - 保持继电器
- ✅ D区(0x82) - 数据寄存器
- ✅ T区(0x89) - 定时器
- ✅ C区(0x8C) - 计数器

### 3. SID模式
- ✅ 固定模式: 所有请求使用相同SID
- ✅ 递增模式: SID自动递增,支持并发请求

### 4. 错误处理
- ✅ 完整的错误码映射
- ✅ 超时检测
- ✅ 自动重试机制
- ✅ 指数退避算法

### 5. 线程安全
- ✅ 使用互斥锁保护共享状态
- ✅ 支持并发请求
- ✅ 异步接收处理

### 6. 统计信息
- ✅ 请求计数
- ✅ 成功/失败统计
- ✅ 超时统计
- ✅ 时间戳记录

## 使用流程

1. **创建配置**
   ```go
   config := fins.DefaultConfig("192.168.1.10")
   config.ServerNode = 0x64
   ```

2. **创建客户端**
   ```go
   client, err := fins.NewClient(config, false) // UDP
   // 或
   client, err := fins.NewClient(config, true)  // TCP
   ```

3. **连接PLC**
   ```go
   err := client.Connect()
   ```

4. **执行操作**
   ```go
   value, err := client.ReadDWord(100)
   err = client.WriteDWord(100, 1234)
   ```

5. **关闭连接**
   ```go
   client.Close()
   ```

## 测试

运行单元测试:
```bash
go test -v
```

运行示例:
```bash
go run examples/udp_example.go
go run examples/tcp_example.go
go run examples/retry_example.go
```

## 依赖

- Go 1.16+
- 无第三方依赖,仅使用标准库

## 下一步扩展

可能的扩展方向:
- [ ] 添加更多命令支持(参数读写、控制器操作等)
- [ ] 实现连接池
- [ ] 添加日志系统
- [ ] 性能优化
- [ ] 更多单元测试
- [ ] 集成测试
- [ ] 基准测试

