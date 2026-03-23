package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jianwushu/fins"
)

func main() {
	config := fins.DefaultConfig("127.0.0.1")
	config.LocalNode = 0x01
	config.ServerNode = 0x64

	reconnectPolicy := &fins.TCPReconnectPolicy{
		EnableAutoReconnect:     true,
		MaxReconnectAttempts:    3,
		InitialDelay:            500 * time.Millisecond,
		MaxDelay:                5 * time.Second,
		BackoffFactor:           2.0,
		ReconnectOnRequestError: true,
	}

	tcpClient, err := fins.NewTCPClientWithReconnect(config, reconnectPolicy)
	if err != nil {
		log.Fatalf("创建 TCP 客户端失败: %v", err)
	}
	defer tcpClient.Close()

	if err := tcpClient.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	client, err := fins.NewClientWithTransport(tcpClient, config)
	if err != nil {
		log.Fatalf("包装 FINS 客户端失败: %v", err)
	}

	for i := 0; i < 6; i++ {
		status := client.GetConnectionStatus()
		fmt.Printf("当前连接状态: %s, IsConnected=%v\n", status.String(), client.IsConnected())

		writeValue := uint16(i % 1000)
		fmt.Printf("写入 D100 = %d\n", writeValue)
		if err := client.WriteWord("D100", writeValue); err != nil {
			log.Printf("写入失败: %v, status=%s", err, client.GetConnectionStatus().String())
		}

		fmt.Printf("读取 D100...\n")
		value, err := client.ReadWord("D100")
		if err != nil {
			log.Printf("读取失败: %v, status=%s", err, client.GetConnectionStatus().String())
		} else {
			fmt.Printf("✅ 读取成功: D100 = %d\n", value)
		}

		data := []byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3)}
		fmt.Printf("写入字节数组到 D200: % X\n", data)
		if err := client.WriteBytes("D200", data); err != nil {
			log.Printf("写入字节数组失败: %v, status=%s", err, client.GetConnectionStatus().String())
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Println("测试完成：如中途断开 TCP 连接，客户端会进入后台自动重连；重连期间新请求会直接返回未连接错误")

	for {
		fmt.Printf("后台状态: %s, IsConnected=%v\n", client.GetConnectionStatus().String(), client.IsConnected())
		time.Sleep(2 * time.Second)
	}
}
