package fins

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// BuildTCPFrame 构建 FINS/TCP 外层帧
func BuildTCPFrame(frame *FinsTCPFrame) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 写入魔数 "FINS"
	buf.Write(frame.Magic[:])

	// Length = Command(4) + ErrorCode(4) + Data(N)
	length := frame.Length
	if length == 0 {
		length = uint32(8 + len(frame.Data))
	}
	if length < 8 {
		return nil, fmt.Errorf("无效的TCP长度: %d", length)
	}
	if int(length) != 8+len(frame.Data) {
		return nil, fmt.Errorf("TCP长度与数据不匹配: length=%d data=%d", length, len(frame.Data))
	}

	// 写入长度（大端序）
	if err := binary.Write(buf, binary.BigEndian, length); err != nil {
		return nil, fmt.Errorf("写入长度失败: %w", err)
	}

	// 写入外层命令（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.Command); err != nil {
		return nil, fmt.Errorf("写入命令失败: %w", err)
	}

	// 写入外层错误码（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.ErrorCode); err != nil {
		return nil, fmt.Errorf("写入错误码失败: %w", err)
	}

	// 写入数据
	if len(frame.Data) > 0 {
		buf.Write(frame.Data)
	}

	return buf.Bytes(), nil
}

// ParseTCPFrame 解析 FINS/TCP 外层帧
func ParseTCPFrame(data []byte) (*FinsTCPFrame, error) {
	if len(data) < TCPHeaderLength {
		return nil, ErrInvalidFrame
	}

	frame := &FinsTCPFrame{}

	// 读取魔数
	copy(frame.Magic[:], data[0:4])
	if string(frame.Magic[:]) != TCPMagic {
		return nil, ErrInvalidMagic
	}

	// 读取长度（Command+ErrorCode+Data）
	frame.Length = binary.BigEndian.Uint32(data[4:8])
	if frame.Length < 8 {
		return nil, ErrInvalidFrame
	}

	expectedTotal := 8 + int(frame.Length)
	if len(data) < expectedTotal {
		return nil, ErrInvalidFrame
	}

	// 读取命令与错误码
	frame.Command = binary.BigEndian.Uint32(data[8:12])
	frame.ErrorCode = binary.BigEndian.Uint32(data[12:16])

	// 读取数据部分
	dataLen := int(frame.Length) - 8
	if dataLen > 0 {
		frame.Data = make([]byte, dataLen)
		copy(frame.Data, data[16:16+dataLen])
	}

	return frame, nil
}

// NewTCPRequestFrame 创建 FINS/TCP 外层请求帧
//
// command: 外层命令码（例如 `TCPCommandFinsFrame`）
// data: 外层数据（例如内层 FINS 报文）
func NewTCPRequestFrame(command uint32, data []byte) *FinsTCPFrame {
	magic := [4]byte{'F', 'I', 'N', 'S'}

	// Length = Command(4) + ErrorCode(4) + Data(N)
	length := uint32(8 + len(data))

	return &FinsTCPFrame{
		Magic:     magic,
		Length:    length,
		Command:   command,
		ErrorCode: 0,
		Data:      data,
	}
}

// ParseTCPResponse 解析 FINS/TCP 的正常读写帧（外层 0x00000002；请求/响应命令码相同）
//
// 返回值为“内层”FINS 响应（SID/StatusCode/Data）。
func ParseTCPResponse(data []byte) (*FinsResponse, error) {
	frame, err := ParseTCPFrame(data)
	if err != nil {
		return nil, err
	}

	if frame.Command != TCPCommandFinsFrame {
		return nil, ErrInvalidResponse
	}

	return ParseUDPResponse(frame.Data)
}

// ReadTCPFrameFromConn 从连接中读取完整的 FINS/TCP 外层帧
//
// FINS/TCP 的 Length 表示后续字节数（Command+ErrorCode+Data），因此总帧长为 8 + Length。
func ReadTCPFrameFromConn(reader func([]byte) (int, error)) ([]byte, error) {
	// 1. 读取魔数（4字节）
	magic := make([]byte, 4)
	n, err := reader(magic)
	if err != nil {
		return nil, err
	}
	if n != 4 || string(magic) != TCPMagic {
		return nil, ErrInvalidMagic
	}

	// 2. 读取长度字段（4字节）
	lengthBuf := make([]byte, 4)
	n, err = reader(lengthBuf)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, ErrInvalidFrame
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length < 8 {
		return nil, ErrInvalidFrame
	}

	// 3. 读取剩余数据（Length 字段指定的后续字节）
	remaining := int(length)
	remainingData := make([]byte, remaining)
	n, err = reader(remainingData)
	if err != nil {
		return nil, err
	}
	if n != remaining {
		return nil, ErrInvalidFrame
	}

	// 组合完整帧：Magic(4)+Length(4)+remaining(length)
	fullFrame := make([]byte, 8+length)
	copy(fullFrame[0:4], magic)
	copy(fullFrame[4:8], lengthBuf)
	copy(fullFrame[8:], remainingData)

	return fullFrame, nil
}
