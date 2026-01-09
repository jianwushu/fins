# 更新日志

## [新增功能] 字节数组操作支持

### 新增API

#### 通用字节操作
- `ReadBytes(areaCode, address, byteCount)` - 读取任意内存区域的字节数组
- `WriteBytes(areaCode, address, data)` - 写入字节数组到任意内存区域

#### D区字节操作
- `ReadDBytes(address, byteCount)` - 读取D区字节数组
- `WriteDBytes(address, data)` - 写入D区字节数组

#### CIO区字节操作
- `ReadCIOBytes(address, byteCount)` - 读取CIO区字节数组
- `WriteCIOBytes(address, data)` - 写入CIO区字节数组

#### HR区字节操作
- `ReadHRBytes(address, byteCount)` - 读取HR区字节数组
- `WriteHRBytes(address, data)` - 写入HR区字节数组

#### WR区字节操作
- `ReadWRBytes(address, byteCount)` - 读取WR区字节数组
- `WriteWRBytes(address, data)` - 写入WR区字节数组

### 功能特性

#### 1. 自动字节对齐
PLC内存按字(16位)存储,字节操作会自动处理对齐:
- **读取**: 自动计算需要读取的字数,支持奇数字节读取
- **写入**: 奇数字节自动补0对齐到偶数字节

```go
// 读取7个字节(自动读取4个字,返回7字节)
data, _ := client.ReadDBytes(100, 7)

// 写入5个字节(自动补0到6字节,写入3个字)
client.WriteDBytes(200, []byte{0x01, 0x02, 0x03, 0x04, 0x05})
```

#### 2. 灵活的数据类型转换
支持字节数组与各种数据类型的转换:

**整数转换**
```go
// 32位整数
value := uint32(0x12345678)
buf := make([]byte, 4)
binary.BigEndian.PutUint32(buf, value)
client.WriteDBytes(100, buf)

// 读取并转换回整数
data, _ := client.ReadDBytes(100, 4)
readValue := binary.BigEndian.Uint32(data)
```

**字符串操作**
```go
// 写入字符串
text := "FINS Protocol"
client.WriteDBytes(200, []byte(text))

// 读取字符串
data, _ := client.ReadDBytes(200, uint16(len(text)))
readText := string(data)
```

#### 3. 多内存区域支持
所有内存区域都支持字节操作:
- D区 (数据寄存器)
- CIO区 (输入输出继电器)
- HR区 (保持继电器)
- WR区 (工作继电器)

#### 4. 批量数据传输
支持大量字节的高效传输:
```go
// 传输100字节
largeData := make([]byte, 100)
client.WriteDBytes(600, largeData)
```

### 新增测试

#### TestBuildWriteMemoryRequest
测试写入内存请求的构建:
- 验证请求数据长度
- 验证内存区域代码
- 验证地址编码
- 验证写入数据

#### TestByteAlignment
测试字节对齐计算:
- 1字节 → 1字
- 2字节 → 1字
- 3字节 → 2字
- 5字节 → 3字
- 11字节 → 6字

### 新增示例

#### examples/bytes_example.go
完整的字节数组操作示例,包含:
1. 读取字节数组
2. 写入字节数组
3. 处理奇数字节
4. 字节数组与整数转换
5. 字符串操作
6. 不同内存区域操作
7. 批量数据传输(100字节)

### 文档更新

#### README.md
- 添加字节数组操作章节
- 添加数据类型转换示例
- 更新示例列表

#### QUICKSTART.md
- 添加字节数组读写示例
- 添加整数转换示例
- 添加字符串操作示例

#### API_REFERENCE.md (新增)
完整的API参考文档,包含:
- 所有函数签名
- 参数说明
- 返回值说明
- 使用示例

### 测试结果

```
=== 测试统计 ===
测试用例数: 8 (+2)
通过: 8
失败: 0
代码覆盖率: 14.2%

✅ TestBuildUDPFrame
✅ TestParseUDPFrame
✅ TestBuildTCPFrame
✅ TestParseTCPFrame
✅ TestBuildReadMemoryRequest
✅ TestGetErrorMessage
✅ TestBuildWriteMemoryRequest (新增)
✅ TestByteAlignment (新增)
```

### 使用场景

字节数组操作适用于以下场景:

1. **原始数据传输**: 直接传输二进制数据
2. **自定义数据格式**: 实现自定义的数据编码/解码
3. **字符串通信**: 在PLC和上位机之间传输文本
4. **结构化数据**: 传输复杂的数据结构
5. **文件传输**: 分块传输文件数据
6. **协议桥接**: 与其他协议进行数据转换

### 性能优化

- 批量操作减少网络往返次数
- 自动对齐避免额外的读写操作
- 零拷贝设计提高效率

### 向后兼容

所有新增功能完全向后兼容,不影响现有代码。

