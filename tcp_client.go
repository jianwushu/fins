package fins

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// FinsTCPClient FINS TCP客户端
//
// 注意：TCP 模式下使用官方 FINS/TCP 外层封装，内层仍是标准 FINS 报文（10B 头 + 2B 命令 + 参数）。
// 请求-响应匹配使用 SID + pending map（与 UDP 一致），以支持并发请求且避免错配。
type FinsTCPClient struct {
	config *FinsClientConfig

	conn      net.Conn
	mutex     sync.RWMutex
	stats     ConnectionStats
	closed    bool
	closeChan chan struct{}

	// 节点号（握手后得到；若握手返回 0 则回退到 config）
	localNode  byte
	serverNode byte

	// SID 生成与 pending 映射（复用 UDP 的思路）
	sequenceNo  uint16
	currentSID  byte
	pendingReqs map[byte]*PendingRequest
}

// NewTCPClient 创建TCP客户端
func NewTCPClient(config *FinsClientConfig) (*FinsTCPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	client := &FinsTCPClient{
		config:      config,
		closeChan:   make(chan struct{}),
		sequenceNo:  uint16(config.StartSID),
		currentSID:  config.FixedSID,
		pendingReqs: make(map[byte]*PendingRequest),
	}

	return client, nil
}

// Connect 连接到服务器
//
// 策略：建连后始终执行握手请求/响应（0x00000000/0x00000001）。
//
// 节点号回填规则（按约定）：
//   - 当 config.LocalNode == 0：表示“由 PLC 分配/协商本机节点号”。优先使用握手返回的 localNode；若返回 0，则从本机 IP 最后一段推导；仍失败则回退 1。
//   - 当 config.LocalNode != 0：表示“本机节点号由本机 IP 最后一段决定”。忽略握手返回的 localNode，强制从本机 IP 最后一段推导；若无法推导则回退 config.LocalNode。
//   - serverNode：优先使用握手返回的 serverNode；若返回 0 则回退 config.ServerNode。
func (c *FinsTCPClient) Connect() error {
	// 1) 建立 TCP 连接
	c.mutex.Lock()
	if c.conn != nil {
		c.mutex.Unlock()
		return fmt.Errorf("已经连接")
	}

	addr := fmt.Sprintf("%v:%d", c.config.IP, c.config.Port)
	conn, err := net.DialTimeout("tcp", addr, c.config.Timeout)
	if err != nil {
		c.mutex.Unlock()
		return fmt.Errorf("连接失败: %w", err)
	}

	c.conn = conn
	c.closed = false
	c.mutex.Unlock()

	// 2) 握手（同步完成，避免 receiveLoop 抢读）
	localNode, serverNode, err := c.handshake(conn)
	if err != nil {
		_ = c.Close()
		return err
	}

	// 3) 保存节点号（按约定的回填/兜底规则）
	c.mutex.Lock()
	c.localNode = resolveLocalNode(c.config.LocalNode, localNode, conn)
	c.serverNode = resolveServerNode(c.config.ServerNode, serverNode)
	c.mutex.Unlock()

	// 4) 启动接收协程
	go c.receiveLoop()

	return nil
}

// Close 关闭连接
func (c *FinsTCPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	// 清理待处理请求
	for sid, req := range c.pendingReqs {
		close(req.Response)
		delete(c.pendingReqs, sid)
	}

	// 创建新的 closeChan，允许重连
	c.closeChan = make(chan struct{})

	return nil
}

// getNextSID 获取下一个SID
func (c *FinsTCPClient) getNextSID() byte {
	switch c.config.SIDMode {
	case SIDFixed:
		return c.config.FixedSID
	case SIDIncrement:
		c.sequenceNo++
		if c.sequenceNo > uint16(c.config.MaxSID) {
			c.sequenceNo = uint16(c.config.StartSID)
		}
		return byte(c.sequenceNo)
	default:
		return c.config.FixedSID
	}
}

// SendRequest 发送请求
func (c *FinsTCPClient) SendRequest(command uint16, data []byte) (*FinsResponse, error) {
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return nil, ErrConnectionClosed
	}
	conn := c.conn
	if conn == nil {
		c.mutex.Unlock()
		return nil, ErrConnectionClosed
	}

	sid := c.getNextSID()

	// 构建“内层”FINS 请求（复用 UDP 的帧格式）
	inner := NewUDPRequestFrame(c.localNode, c.serverNode, sid, command, data)
	innerBytes, err := BuildUDPFrame(inner)
	if err != nil {
		c.mutex.Unlock()
		return nil, fmt.Errorf("构建内层FINS帧失败: %w", err)
	}

	// 构建“外层”FINS/TCP 帧（0x00000002；请求/响应命令码相同）
	outer := NewTCPRequestFrame(TCPCommandFinsFrame, innerBytes)
	outerBytes, err := BuildTCPFrame(outer)
	if err != nil {
		c.mutex.Unlock()
		return nil, fmt.Errorf("构建外层TCP帧失败: %w", err)
	}

	// 创建待处理请求
	req := &PendingRequest{
		SID:       sid,
		Request:   outerBytes,
		CreatedAt: time.Now(),
		Response:  make(chan *FinsResponse, 1),
	}
	c.pendingReqs[sid] = req
	c.mutex.Unlock()

	// 发送数据
	_, err = conn.Write(outerBytes)
	if err != nil {
		c.mutex.Lock()
		delete(c.pendingReqs, sid)
		c.mutex.Unlock()
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	c.stats.TotalRequests++
	c.stats.LastRequestAt = time.Now()

	// 等待响应或超时
	select {
	case resp := <-req.Response:
		c.stats.LastResponseAt = time.Now()
		if resp.IsSuccess() {
			c.stats.SuccessCount++
		} else {
			c.stats.ErrorCount++
		}
		return resp, nil
	case <-time.After(c.config.Timeout):
		c.mutex.Lock()
		delete(c.pendingReqs, sid)
		c.mutex.Unlock()
		c.stats.TimeoutCount++
		return nil, ErrTimeout
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	}
}

// receiveLoop 接收循环
func (c *FinsTCPClient) receiveLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		c.mutex.RLock()
		conn := c.conn
		closed := c.closed
		c.mutex.RUnlock()
		if closed || conn == nil {
			return
		}

		// 设置读取超时，便于响应 closeChan
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		frameData, err := ReadTCPFrameFromConn(func(buf []byte) (int, error) {
			return io.ReadFull(conn, buf)
		})
		if err != nil {
			// 读取超时：继续循环检查 closeChan
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			if !closed {
				fmt.Printf("读取帧失败: %v\n", err)
			}
			return
		}

		outer, err := ParseTCPFrame(frameData)
		if err != nil {
			fmt.Printf("解析TCP帧失败: %v\n", err)
			continue
		}

		// 只处理正常读写帧（0x00000002；请求/响应命令码相同）
		if outer.Command != TCPCommandFinsFrame {
			continue
		}

		resp, err := ParseUDPResponse(outer.Data)
		if err != nil {
			fmt.Printf("解析内层FINS响应失败: %v\n", err)
			continue
		}

		// 命中 pending 并投递
		c.mutex.Lock()
		req, exists := c.pendingReqs[resp.SID]
		if exists {
			delete(c.pendingReqs, resp.SID)
		}
		c.mutex.Unlock()

		if exists {
			select {
			case req.Response <- resp:
			default:
			}
		}
	}
}

// GetStats 获取统计信息
func (c *FinsTCPClient) GetStats() ConnectionStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.stats
}

// IsConnected 检查是否已连接
func (c *FinsTCPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.conn != nil && !c.closed
}

// handshake 执行握手请求/响应（0x00000000/0x00000001），返回 (localNode, serverNode)
func (c *FinsTCPClient) handshake(conn net.Conn) (byte, byte, error) {
	// 握手请求的 Data（4 字节）常见为“客户端 IPv4”。
	//
	// 按约定：
	//   - config.LocalNode == 0：发送 0.0.0.0，让 PLC 自动分配/协商节点号；
	//   - config.LocalNode != 0：发送本机连接使用的 IPv4（4 字节），便于 PLC 按 IP 规则确认节点号。
	var ipBytes []byte
	if c.config.LocalNode == 0 {
		ipBytes = []byte{0, 0, 0, 0}
	} else {
		if b, ok := localIPv4BytesFromConn(conn); ok {
			ipBytes = b
		} else {
			// 解析失败时退回 0.0.0.0，让 PLC 自行决定。
			ipBytes = []byte{0, 0, 0, 0}
		}
	}

	reqFrame := NewTCPRequestFrame(TCPCommandHandshakeRequest, ipBytes)
	bytesToSend, err := BuildTCPFrame(reqFrame)
	if err != nil {
		return 0, 0, fmt.Errorf("构建握手请求失败: %w", err)
	}

	if _, err := conn.Write(bytesToSend); err != nil {
		return 0, 0, fmt.Errorf("发送握手请求失败: %w", err)
	}

	// 读取响应
	_ = conn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	respBytes, err := ReadTCPFrameFromConn(func(buf []byte) (int, error) {
		return io.ReadFull(conn, buf)
	})
	if err != nil {
		return 0, 0, fmt.Errorf("读取握手响应失败: %w", err)
	}

	respFrame, err := ParseTCPFrame(respBytes)
	if err != nil {
		return 0, 0, fmt.Errorf("解析握手响应失败: %w", err)
	}
	if respFrame.Command != TCPCommandHandshakeResponse {
		return 0, 0, fmt.Errorf("握手响应命令不匹配: 0x%08X", respFrame.Command)
	}
	if respFrame.ErrorCode != 0 {
		return 0, 0, fmt.Errorf("握手响应错误码: 0x%08X", respFrame.ErrorCode)
	}
	if len(respFrame.Data) < 8 {
		return 0, 0, ErrInvalidResponse
	}

	// 常见实现：返回 4B client + 4B server。
	//
	// 多数场景下“节点号”与 IPv4 最后一段对应，因此通常取每个 4B 的最后 1 字节：
	//   - localNode = Data[3]
	//   - serverNode = Data[7]
	localNode := respFrame.Data[3]
	serverNode := respFrame.Data[7]

	return localNode, serverNode, nil
}

func localIPv4BytesFromConn(conn net.Conn) ([]byte, bool) {
	if conn == nil || conn.LocalAddr() == nil {
		return nil, false
	}
	// LocalAddr() 典型格式："192.168.1.10:54321" 或 "[::1]:54321"
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return nil, false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, false
	}
	b := []byte{ip4[0], ip4[1], ip4[2], ip4[3]}
	return b, true
}

func deriveNodeFromIPv4(ipBytes []byte) (byte, bool) {
	if len(ipBytes) != 4 {
		return 0, false
	}
	// 常见约定：节点号 = IPv4 最后一段。节点号 0 通常视为无效/未分配。
	if ipBytes[3] == 0 {
		return 0, false
	}
	return ipBytes[3], true
}

func resolveLocalNode(configLocalNode byte, handshakeLocalNode byte, conn net.Conn) byte {
	// configLocalNode == 0：期望由 PLC 分配；PLC 返回 0 时再从本机 IP 推导。
	if configLocalNode == 0 {
		if handshakeLocalNode != 0 {
			return handshakeLocalNode
		}
		if ipBytes, ok := localIPv4BytesFromConn(conn); ok {
			if n, ok := deriveNodeFromIPv4(ipBytes); ok {
				return n
			}
		}
		// 最后兜底：1（避免 0 节点号）
		return 0x01
	}

	// configLocalNode != 0：强制使用本机 IP 最后一段。
	if ipBytes, ok := localIPv4BytesFromConn(conn); ok {
		if n, ok := deriveNodeFromIPv4(ipBytes); ok {
			return n
		}
	}
	return configLocalNode
}

func resolveServerNode(configServerNode byte, handshakeServerNode byte) byte {
	if handshakeServerNode != 0 {
		return handshakeServerNode
	}
	return configServerNode
}
