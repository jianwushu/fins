package fins

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestBuildUDPFrame(t *testing.T) {
	frame := &FinsUDPFrame{
		ICF:     ICFRequest,
		RSV:     0x00,
		GCT:     0x02,
		DNA:     0x00,
		DA1:     0x64,
		DA2:     0x00,
		SNA:     0x00,
		SA1:     0x01,
		SA2:     0x00,
		SID:     0x00,
		Command: CmdMemoryRead,
		Data:    []byte{0x82, 0x00, 0x64, 0x00, 0x00, 0x01},
	}

	data, err := BuildUDPFrame(frame)
	if err != nil {
		t.Fatalf("构建UDP帧失败: %v", err)
	}

	// 验证帧头长度
	if len(data) < UDPHeaderLength+2 {
		t.Errorf("帧长度不正确: got %d, want >= %d", len(data), UDPHeaderLength+2)
	}

	// 验证ICF
	if data[0] != ICFRequest {
		t.Errorf("ICF不正确: got 0x%02X, want 0x%02X", data[0], ICFRequest)
	}

	// 验证命令码
	cmd := binary.BigEndian.Uint16(data[10:12])
	if cmd != CmdMemoryRead {
		t.Errorf("命令码不正确: got 0x%04X, want 0x%04X", cmd, CmdMemoryRead)
	}
}

func TestParseUDPFrame(t *testing.T) {
	// 构建测试数据
	data := []byte{
		0x02,       // ICF
		0x00,       // RSV
		0x02,       // GCT
		0x00,       // DNA
		0x64,       // DA1
		0x00,       // DA2
		0x00,       // SNA
		0x01,       // SA1
		0x00,       // SA2
		0x00,       // SID
		0x01, 0x01, // Command
		0x00, 0x00, // Status
	}

	frame, err := ParseUDPFrame(data)
	if err != nil {
		t.Fatalf("解析UDP帧失败: %v", err)
	}

	if frame.ICF != 0x02 {
		t.Errorf("ICF不正确: got 0x%02X, want 0x02", frame.ICF)
	}

	if frame.Command != CmdMemoryRead {
		t.Errorf("命令码不正确: got 0x%04X, want 0x%04X", frame.Command, CmdMemoryRead)
	}
}

func TestBuildTCPFrame(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	frame := NewTCPRequestFrame(TCPCommandFinsFrame, payload)

	data, err := BuildTCPFrame(frame)
	if err != nil {
		t.Fatalf("构建TCP帧失败: %v", err)
	}

	// 验证魔数
	if string(data[0:4]) != TCPMagic {
		t.Errorf("魔数不正确: got %s, want %s", string(data[0:4]), TCPMagic)
	}

	// 验证长度：Length = Command(4)+Error(4)+Data(N)
	length := binary.BigEndian.Uint32(data[4:8])
	wantLen := uint32(8 + len(payload))
	if length != wantLen {
		t.Errorf("长度不正确: got %d, want %d", length, wantLen)
	}

	// 验证总长度：8 + Length
	if len(data) != int(8+length) {
		t.Errorf("总长度不正确: got %d, want %d", len(data), int(8+length))
	}

	// 验证命令
	cmd := binary.BigEndian.Uint32(data[8:12])
	if cmd != TCPCommandFinsFrame {
		t.Errorf("命令不正确: got 0x%08X, want 0x%08X", cmd, TCPCommandFinsFrame)
	}
}

func TestParseTCPFrame(t *testing.T) {
	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	// 构建测试数据
	magic := []byte("FINS")
	length := uint32(8 + len(payload))

	data := make([]byte, 8+length)
	copy(data[0:4], magic)
	binary.BigEndian.PutUint32(data[4:8], length)
	binary.BigEndian.PutUint32(data[8:12], TCPCommandFinsFrame)
	binary.BigEndian.PutUint32(data[12:16], 0)
	copy(data[16:], payload)

	frame, err := ParseTCPFrame(data)
	if err != nil {
		t.Fatalf("解析TCP帧失败: %v", err)
	}

	if string(frame.Magic[:]) != TCPMagic {
		t.Errorf("魔数不正确: got %s, want %s", string(frame.Magic[:]), TCPMagic)
	}

	if frame.Command != TCPCommandFinsFrame {
		t.Errorf("命令不正确: got 0x%08X, want 0x%08X", frame.Command, TCPCommandFinsFrame)
	}

	if !bytes.Equal(frame.Data, payload) {
		t.Errorf("payload不正确: got %v, want %v", frame.Data, payload)
	}
}

func TestBuildReadMemoryRequest(t *testing.T) {
	req := &ReadRequest{
		AreaCode: MemAreaD,
		Address:  100,
		BitNo:    0,
		Count:    1,
	}

	data := BuildReadMemoryRequest(req)

	if len(data) != 6 {
		t.Errorf("请求数据长度不正确: got %d, want 6", len(data))
	}

	if data[0] != MemAreaD {
		t.Errorf("内存区域代码不正确: got 0x%02X, want 0x%02X", data[0], MemAreaD)
	}

	addr := binary.BigEndian.Uint16(data[1:3])
	if addr != 100 {
		t.Errorf("地址不正确: got %d, want 100", addr)
	}
}

func TestGetErrorMessage(t *testing.T) {
	msg := GetErrorMessage(ErrCodeSuccess)
	if msg != "成功" {
		t.Errorf("错误消息不正确: got %s, want 成功", msg)
	}

	msg = GetErrorMessage(ErrCodeAddressOutOfRange)
	if msg != "地址越界" {
		t.Errorf("错误消息不正确: got %s, want 地址越界", msg)
	}

	msg = GetErrorMessage(0xFFFF)
	if msg != "未知错误" {
		t.Errorf("未知错误消息不正确: got %s, want 未知错误", msg)
	}
}

func TestBuildWriteMemoryRequest(t *testing.T) {
	writeData := []byte{0x01, 0x02, 0x03, 0x04}
	req := &WriteRequest{
		AreaCode: MemAreaD,
		Address:  200,
		BitNo:    0,
		Count:    2,
		Data:     writeData,
	}

	data := BuildWriteMemoryRequest(req)

	// 验证数据长度: 6字节头部 + 4字节数据
	if len(data) != 10 {
		t.Errorf("请求数据长度不正确: got %d, want 10", len(data))
	}

	// 验证内存区域代码
	if data[0] != MemAreaD {
		t.Errorf("内存区域代码不正确: got 0x%02X, want 0x%02X", data[0], MemAreaD)
	}

	// 验证地址
	addr := binary.BigEndian.Uint16(data[1:3])
	if addr != 200 {
		t.Errorf("地址不正确: got %d, want 200", addr)
	}

	// 验证写入数据
	if !bytes.Equal(data[6:], writeData) {
		t.Errorf("写入数据不正确: got %v, want %v", data[6:], writeData)
	}
}

func TestByteAlignment(t *testing.T) {
	// 测试奇数字节数的处理
	tests := []struct {
		name      string
		byteCount uint16
		wantWords uint16
	}{
		{"1字节", 1, 1},
		{"2字节", 2, 1},
		{"3字节", 3, 2},
		{"4字节", 4, 2},
		{"5字节", 5, 3},
		{"10字节", 10, 5},
		{"11字节", 11, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wordCount := (tt.byteCount + 1) / 2
			if wordCount != tt.wantWords {
				t.Errorf("字数计算错误: byteCount=%d, got %d words, want %d words",
					tt.byteCount, wordCount, tt.wantWords)
			}
		})
	}
}

// ---- helpers for tcp_client.go tests ----

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

type stubConn struct {
	local net.Addr
}

func (c *stubConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (c *stubConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (c *stubConn) Close() error                       { return nil }
func (c *stubConn) LocalAddr() net.Addr                { return c.local }
func (c *stubConn) RemoteAddr() net.Addr               { return stubAddr("192.168.0.10:9600") }
func (c *stubConn) SetDeadline(t time.Time) error      { return nil }
func (c *stubConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *stubConn) SetWriteDeadline(t time.Time) error { return nil }

func TestLocalIPv4BytesFromConn(t *testing.T) {
	conn := &stubConn{local: stubAddr("127.0.0.1:12345")}
	b, ok := localIPv4BytesFromConn(conn)
	if !ok {
		t.Fatalf("expected ok")
	}
	if !bytes.Equal(b, []byte{127, 0, 0, 1}) {
		t.Fatalf("unexpected bytes: %v", b)
	}
}

func TestDeriveNodeFromIPv4(t *testing.T) {
	if n, ok := deriveNodeFromIPv4([]byte{127, 0, 0, 1}); !ok || n != 1 {
		t.Fatalf("expected node=1 ok=true, got node=%d ok=%v", n, ok)
	}
	if _, ok := deriveNodeFromIPv4([]byte{192, 168, 1, 0}); ok {
		t.Fatalf("expected ok=false for last octet 0")
	}
	if _, ok := deriveNodeFromIPv4([]byte{1, 2, 3}); ok {
		t.Fatalf("expected ok=false for len!=4")
	}
}

func TestResolveLocalNode_ConfigZero_HandshakeNonZero(t *testing.T) {
	conn := &stubConn{local: stubAddr("127.0.0.1:12345")}
	got := resolveLocalNode(0, 77, conn)
	if got != 77 {
		t.Fatalf("expected 77, got %d", got)
	}
}

func TestResolveLocalNode_ConfigZero_HandshakeZero_DeriveFromIP(t *testing.T) {
	conn := &stubConn{local: stubAddr("10.0.0.5:12345")}
	got := resolveLocalNode(0, 0, conn)
	if got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}

func TestResolveLocalNode_ConfigZero_AllFail_FallbackTo1(t *testing.T) {
	conn := &stubConn{local: stubAddr("[::1]:12345")}
	got := resolveLocalNode(0, 0, conn)
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestResolveLocalNode_ConfigNonZero_ForceDeriveFromIP(t *testing.T) {
	conn := &stubConn{local: stubAddr("192.168.1.200:12345")}
	got := resolveLocalNode(9, 77, conn)
	if got != 200 {
		t.Fatalf("expected 200, got %d", got)
	}
}

func TestResolveLocalNode_ConfigNonZero_DeriveFail_FallbackToConfig(t *testing.T) {
	conn := &stubConn{local: stubAddr("[::1]:12345")}
	got := resolveLocalNode(9, 77, conn)
	if got != 9 {
		t.Fatalf("expected 9, got %d", got)
	}
}

func TestResolveServerNode(t *testing.T) {
	if got := resolveServerNode(10, 0); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	if got := resolveServerNode(10, 20); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}
