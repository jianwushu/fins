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

	client, err := fins.NewClient(config, true)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	reconnectPolicy := &fins.ReconnectPolicy{
		EnableAutoReconnect:  true,
		MaxReconnectAttempts: 0,
		InitialDelay:         1 * time.Second,
		MaxDelay:             30 * time.Second,
		BackoffFactor:        2.0,
		ReconnectOnError:     true,
		HealthCheckInterval:  10 * time.Second,
	}

	reconnectClient := fins.NewReconnectableClient(client, reconnectPolicy)
	reconnectClient.SetOnReconnect(func() { fmt.Println("✅ 重连成功!") })
	reconnectClient.SetOnDisconnect(func() { fmt.Println("❌ 连接断开!") })

	if err := reconnectClient.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	for i := 0; i < 3; i++ {
		writeValue := uint16(i % 1000)
		fmt.Printf("写入 D100 = %d\n", writeValue)
		if err := reconnectClient.WriteWord("D100", writeValue); err != nil {
			log.Printf("写入失败: %v", err)
		}

		fmt.Printf("读取 D100...\n")
		value, err := reconnectClient.ReadWord("D100")
		if err != nil {
			log.Printf("读取失败: %v", err)
		} else {
			fmt.Printf("✅ 读取成功: D100 = %d\n", value)
		}

		data := []byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3)}
		fmt.Printf("写入字节数组到 D200: % X\n", data)
		if err := reconnectClient.WriteBytes("D200", data); err != nil {
			log.Printf("写入字节数组失败: %v", err)
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Printf("重连次数: %d\n", reconnectClient.GetReconnectCount())
}
