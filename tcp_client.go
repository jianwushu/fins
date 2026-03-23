package fins

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// TCPReconnectPolicy TCP 专属轻量重连策略。
//
// 第一版仅用于请求路径的连接类错误恢复：
//   - 不做后台健康检查
//   - 不做统一包装器
//   - 不引入新的读写 API
//   - 仅在请求发送/等待响应过程中遇到连接类错误时，尝试内部重连一次
//
// MaxReconnectAttempts 表示单次重连流程内的最大建连尝试次数：
//   - 0 表示无限尝试
//   - 1 表示只尝试一次重新 Connect
//   - 大于 1 表示按退避策略多次尝试
//
// ReconnectOnRequestError 控制 [`SendRequest()`](tcp_client.go:234) 是否在连接类错误时触发内部重连。
type TCPReconnectPolicy struct {
	EnableAutoReconnect     bool
	MaxReconnectAttempts    int
	InitialDelay            time.Duration
	MaxDelay                time.Duration
	BackoffFactor           float64
	ReconnectOnRequestError bool
}

// DefaultTCPReconnectPolicy 返回默认 TCP 重连策略。
func DefaultTCPReconnectPolicy() *TCPReconnectPolicy {
	return &TCPReconnectPolicy{
		EnableAutoReconnect:     false,
		MaxReconnectAttempts:    1,
		InitialDelay:            200 * time.Millisecond,
		MaxDelay:                3 * time.Second,
		BackoffFactor:           2.0,
		ReconnectOnRequestError: true,
	}
}

func cloneTCPReconnectPolicy(policy *TCPReconnectPolicy) *TCPReconnectPolicy {
	if policy == nil {
		policy = DefaultTCPReconnectPolicy()
	}

	cloned := *policy
	if cloned.InitialDelay <= 0 {
		cloned.InitialDelay = 200 * time.Millisecond
	}
	if cloned.MaxDelay <= 0 {
		cloned.MaxDelay = 3 * time.Second
	}
	if cloned.BackoffFactor < 1 {
		cloned.BackoffFactor = 2.0
	}
	if cloned.MaxDelay < cloned.InitialDelay {
		cloned.MaxDelay = cloned.InitialDelay
	}
	return &cloned
}

// FinsTCPClient FINS TCP客户端
//
// 注意：TCP 模式下使用官方 FINS/TCP 外层封装，内层仍是标准 FINS 报文（10B 头 + 2B 命令 + 参数）。
// 请求-响应匹配使用 SID + pending map（与 UDP 一致），以支持并发请求且避免错配。
type FinsTCPClient struct {
	config *FinsClientConfig

	conn      net.Conn
	mutex     sync.RWMutex
	closed    bool
	status    ConnectionStatus
	closeChan chan struct{}

	// 节点号（握手后得到；若握手返回 0 则回退到 config）
	localNode  byte
	serverNode byte

	// SID 生成与 pending 映射（复用 UDP 的思路）
	sequenceNo  uint16
	pendingReqs map[byte]*PendingRequest

	// TCP 内建后台重连状态
	reconnectPolicy  *TCPReconnectPolicy
	reconnecting     bool
	reconnectStopCh  chan struct{}
	lastReconnectErr error
}

// NewTCPClient 创建 TCP 客户端。
func NewTCPClient(config *FinsClientConfig) (*FinsTCPClient, error) {
	return NewTCPClientWithReconnect(config, nil)
}

// NewTCPClientWithReconnect 创建带 TCP 内建重连策略的客户端。
func NewTCPClientWithReconnect(config *FinsClientConfig, policy *TCPReconnectPolicy) (*FinsTCPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	client := &FinsTCPClient{
		config:           config,
		status:           ConnectionStatusDisconnected,
		closeChan:        make(chan struct{}),
		sequenceNo:       uint16(config.StartSID),
		pendingReqs:      make(map[byte]*PendingRequest),
		reconnectPolicy:  cloneTCPReconnectPolicy(policy),
		lastReconnectErr: nil,
	}

	return client, nil
}

// SetReconnectPolicy 设置 TCP 内建重连策略。
func (c *FinsTCPClient) SetReconnectPolicy(policy *TCPReconnectPolicy) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.reconnectPolicy = cloneTCPReconnectPolicy(policy)
}

func (c *FinsTCPClient) setStatusLocked(status ConnectionStatus) {
	c.status = status
	if status == ConnectionStatusClosed {
		c.closed = true
	}
}

// GetConnectionStatus 获取连接状态。
func (c *FinsTCPClient) GetConnectionStatus() ConnectionStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
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
	if c.closed {
		c.mutex.Unlock()
		return ErrConnectionClosed
	}
	if c.conn != nil {
		c.mutex.Unlock()
		return fmt.Errorf("已经连接")
	}
	if c.reconnecting {
		c.setStatusLocked(ConnectionStatusReconnecting)
	} else {
		c.setStatusLocked(ConnectionStatusConnecting)
	}

	addr := fmt.Sprintf("%v:%d", c.config.IP, c.config.Port)
	conn, err := net.DialTimeout("tcp", addr, c.config.Timeout)
	if err != nil {
		c.setStatusLocked(ConnectionStatusDisconnected)
		c.mutex.Unlock()
		return fmt.Errorf("连接失败: %w", err)
	}

	c.conn = conn
	c.mutex.Unlock()

	// 2) 握手（同步完成，避免 receiveLoop 抢读）
	localNode, serverNode, err := c.handshake(conn)
	if err != nil {
		c.handleConnectionFailure(conn)
		return err
	}

	// 3) 保存节点号（按约定的回填/兜底规则）
	c.mutex.Lock()
	c.localNode = resolveLocalNode(c.config.LocalNode, localNode, conn)
	c.serverNode = resolveServerNode(c.config.ServerNode, serverNode)
	c.lastReconnectErr = nil
	c.setStatusLocked(ConnectionStatusConnected)
	currentCloseChan := c.closeChan
	c.mutex.Unlock()

	// 4) 启动接收协程
	go c.receiveLoop(conn, currentCloseChan)

	return nil
}

// Close 关闭连接
func (c *FinsTCPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil
	}

	c.setStatusLocked(ConnectionStatusClosed)
	if c.reconnectStopCh != nil {
		close(c.reconnectStopCh)
		c.reconnectStopCh = nil
	}
	c.signalConnectionEventLocked()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	c.clearPendingRequestsLocked()
	c.lastReconnectErr = ErrConnectionClosed

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

func (c *FinsTCPClient) clearPendingRequestsLocked() {
	for sid := range c.pendingReqs {
		delete(c.pendingReqs, sid)
	}
}

func (c *FinsTCPClient) signalConnectionEventLocked() {
	select {
	case <-c.closeChan:
	default:
		close(c.closeChan)
	}
	c.closeChan = make(chan struct{})
}

func (c *FinsTCPClient) handleConnectionFailure(failedConn net.Conn) {
	shouldStartReconnect := false

	c.mutex.Lock()
	if failedConn != nil && c.conn != nil && c.conn != failedConn {
		c.mutex.Unlock()
		return
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	if !c.closed {
		c.signalConnectionEventLocked()
		if c.reconnectPolicy != nil && c.reconnectPolicy.EnableAutoReconnect {
			c.setStatusLocked(ConnectionStatusReconnecting)
			shouldStartReconnect = true
		} else {
			c.setStatusLocked(ConnectionStatusDisconnected)
		}
	}
	c.clearPendingRequestsLocked()
	c.mutex.Unlock()

	if shouldStartReconnect {
		c.ensureReconnectLoop()
	}
}

func (c *FinsTCPClient) getReconnectPolicy() *TCPReconnectPolicy {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return cloneTCPReconnectPolicy(c.reconnectPolicy)
}

func (c *FinsTCPClient) shouldReconnectOnError(err error) bool {
	policy := c.getReconnectPolicy()
	if policy == nil {
		return false
	}
	if !policy.EnableAutoReconnect {
		return false
	}
	return isTCPConnectionError(err)
}

func (c *FinsTCPClient) ensureReconnectLoop() {
	policy := c.getReconnectPolicy()
	if policy == nil || !policy.EnableAutoReconnect {
		return
	}

	c.mutex.Lock()
	if c.closed || c.reconnecting || c.conn != nil {
		c.mutex.Unlock()
		return
	}
	c.reconnecting = true
	c.lastReconnectErr = nil
	stopCh := make(chan struct{})
	c.reconnectStopCh = stopCh
	c.setStatusLocked(ConnectionStatusReconnecting)
	c.mutex.Unlock()

	go c.reconnectLoop(stopCh)
}

func (c *FinsTCPClient) reconnectLoop(stopCh chan struct{}) {
	policy := c.getReconnectPolicy()
	if policy == nil || !policy.EnableAutoReconnect {
		c.mutex.Lock()
		c.reconnecting = false
		if c.conn == nil && !c.closed {
			c.setStatusLocked(ConnectionStatusDisconnected)
		}
		c.mutex.Unlock()
		return
	}

	delay := policy.InitialDelay
	attempt := 0
	var reconnectErr error

	for {
		attempt++
		reconnectErr = c.Connect()
		if reconnectErr == nil {
			c.mutex.Lock()
			c.reconnecting = false
			c.reconnectStopCh = nil
			c.lastReconnectErr = nil
			c.mutex.Unlock()
			return
		}

		c.mutex.Lock()
		c.lastReconnectErr = reconnectErr
		if c.closed {
			c.reconnecting = false
			c.reconnectStopCh = nil
			c.mutex.Unlock()
			return
		}
		if policy.MaxReconnectAttempts > 0 && attempt >= policy.MaxReconnectAttempts {
			attempt = 0
			delay = policy.MaxDelay
		}
		c.mutex.Unlock()

		select {
		case <-stopCh:
			c.mutex.Lock()
			c.reconnecting = false
			c.reconnectStopCh = nil
			if c.conn == nil && !c.closed {
				c.setStatusLocked(ConnectionStatusDisconnected)
			}
			c.mutex.Unlock()
			return
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
}

func isTCPConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrConnectionClosed) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && !netErr.Timeout() {
		return true
	}

	errText := strings.ToLower(err.Error())
	connectionMarkers := []string{
		"eof",
		"broken pipe",
		"connection reset",
		"connection refused",
		"closed network connection",
		"use of closed network connection",
		"连接已关闭",
		"发送失败",
		"读取失败",
	}
	for _, marker := range connectionMarkers {
		if strings.Contains(errText, marker) {
			return true
		}
	}
	return false
}

func (c *FinsTCPClient) sendRequestOnce(command uint16, data []byte) (*FinsResponse, error) {
	conn, req, closeChan, err := c.prepareTCPRequest(command, data)
	if err != nil {
		return nil, err
	}

	if _, err = conn.Write(req.Request); err != nil {
		c.handleTCPWriteFailure(conn, req.SID)
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	return c.waitTCPResponse(req, closeChan)
}

func (c *FinsTCPClient) prepareTCPRequest(command uint16, data []byte) (net.Conn, *PendingRequest, chan struct{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil, nil, nil, ErrConnectionClosed
	}

	conn := c.conn
	if c.status != ConnectionStatusConnected || conn == nil {
		return nil, nil, nil, ErrNotConnected
	}

	sid := c.getNextSID()
	inner := NewUDPRequestFrame(c.localNode, c.serverNode, sid, command, data)
	innerBytes, err := BuildUDPFrame(inner)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("构建内层FINS帧失败: %w", err)
	}

	outer := NewTCPRequestFrame(TCPCommandFinsFrame, innerBytes)
	outerBytes, err := BuildTCPFrame(outer)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("构建外层TCP帧失败: %w", err)
	}

	req := &PendingRequest{
		SID:       sid,
		Request:   outerBytes,
		CreatedAt: time.Now(),
		Response:  make(chan *FinsResponse, 1),
	}
	c.pendingReqs[sid] = req
	return conn, req, c.closeChan, nil
}

func (c *FinsTCPClient) handleTCPWriteFailure(conn net.Conn, sid byte) {
	c.removePendingRequest(sid)
	c.handleConnectionFailure(conn)
}

func (c *FinsTCPClient) waitTCPResponse(req *PendingRequest, closeChan chan struct{}) (*FinsResponse, error) {
	timer := time.NewTimer(c.config.Timeout)
	defer timer.Stop()

	select {
	case resp := <-req.Response:
		if resp == nil {
			return nil, ErrConnectionClosed
		}
		return resp, nil
	case <-timer.C:
		c.removePendingRequest(req.SID)
		return nil, ErrTimeout
	case <-closeChan:
		c.removePendingRequest(req.SID)
		return nil, ErrConnectionClosed
	}
}

func (c *FinsTCPClient) removePendingRequest(sid byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.pendingReqs, sid)
}

// SendRequest 发送请求。
//
// 当启用自动重连时：
//   - 连接异常由后台协程自动恢复
//   - 重连期间新请求直接返回未连接错误
//   - 不阻塞等待重连完成，也不自动重发当前请求
func (c *FinsTCPClient) SendRequest(command uint16, data []byte) (*FinsResponse, error) {
	resp, err := c.sendRequestOnce(command, data)
	if err == nil {
		return resp, nil
	}
	if c.shouldReconnectOnError(err) {
		c.ensureReconnectLoop()
		return nil, ErrNotConnected
	}
	return nil, err
}

// receiveLoop 接收循环
func (c *FinsTCPClient) receiveLoop(conn net.Conn, closeChan chan struct{}) {
	for {
		select {
		case <-closeChan:
			return
		default:
		}

		if !c.isReceiveConnActive(conn) {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		frameData, err := ReadTCPFrameFromConn(func(buf []byte) (int, error) {
			return io.ReadFull(conn, buf)
		})
		if err != nil {
			if c.handleTCPReceiveError(conn, err) {
				continue
			}
			return
		}

		resp, err := c.parseTCPResponseFrame(frameData)
		if err != nil {
			if err != ErrInvalidResponse {
				fmt.Printf("解析TCP响应失败: %v\n", err)
			}
			continue
		}

		c.deliverPendingResponse(resp)
	}
}

func (c *FinsTCPClient) isReceiveConnActive(conn net.Conn) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return !c.closed && c.conn != nil && c.conn == conn
}

func (c *FinsTCPClient) handleTCPReceiveError(conn net.Conn, err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF || isTCPConnectionError(err) {
		c.handleConnectionFailure(conn)
		return false
	}

	c.mutex.RLock()
	closed := c.closed
	c.mutex.RUnlock()
	if !closed {
		fmt.Printf("读取帧失败: %v\n", err)
	}
	return false
}

func (c *FinsTCPClient) parseTCPResponseFrame(frameData []byte) (*FinsResponse, error) {
	outer, err := ParseTCPFrame(frameData)
	if err != nil {
		return nil, err
	}
	if outer.Command != TCPCommandFinsFrame {
		return nil, ErrInvalidResponse
	}
	return ParseUDPResponse(outer.Data)
}

func (c *FinsTCPClient) deliverPendingResponse(resp *FinsResponse) {
	c.mutex.Lock()
	req, exists := c.pendingReqs[resp.SID]
	if exists {
		delete(c.pendingReqs, resp.SID)
	}
	c.mutex.Unlock()
	if !exists {
		return
	}

	select {
	case req.Response <- resp:
	default:
	}
}

// IsConnected 检查是否已连接
func (c *FinsTCPClient) IsConnected() bool {
	return c.GetConnectionStatus() == ConnectionStatusConnected
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
