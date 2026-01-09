package main

import (
	"fmt"
	"log"
	"time"

	"github.com/yourusername/fins"
)

func main() {
	// 创建客户端配置
	config := fins.DefaultConfig("192.168.1.10")
	config.LocalNode = 0x01
	config.ServerNode = 0x64
	config.Timeout = 3 * time.Second

	// 创建TCP客户端
	client, err := fins.NewClient(config, true) // true = TCP模式
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 创建重连策略
	reconnectPolicy := &fins.ReconnectPolicy{
		EnableAutoReconnect:  true,               // 启用自动重连
		MaxReconnectAttempts: 0,                  // 0 = 无限重试
		InitialDelay:         1 * time.Second,    // 初始延迟1秒
		MaxDelay:             30 * time.Second,   // 最大延迟30秒
		BackoffFactor:        2.0,                // 指数退避因子
		ReconnectOnError:     true,               // 读写错误时自动重连
		HealthCheckInterval:  10 * time.Second,   // 每10秒健康检查
	}

	// 创建支持自动重连的客户端
	reconnectClient := fins.NewReconnectableClient(client, reconnectPolicy)

	// 设置重连回调
	reconnectClient.SetOnReconnect(func() {
		fmt.Println("✅ [回调] 重连成功!")
	})

	reconnectClient.SetOnDisconnect(func() {
		fmt.Println("❌ [回调] 连接断开!")
	})

	// 初始连接
	fmt.Println("正在连接到PLC...")
	if err := reconnectClient.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	fmt.Println("✅ 成功连接到PLC\n")

	// 启动监控协程
	go monitorConnection(reconnectClient)

	// 模拟持续的读写操作
	fmt.Println("=== 开始持续读写操作 ===")
	fmt.Println("提示: 可以在运行过程中断开PLC连接来测试自动重连功能\n")

	for i := 1; ; i++ {
		fmt.Printf("\n--- 第 %d 次操作 ---\n", i)

		// 写入数据
		writeValue := uint16(i % 1000)
		fmt.Printf("写入 D100 = %d\n", writeValue)
		err := reconnectClient.WriteDWord(100, writeValue)
		if err != nil {
			fmt.Printf("❌ 写入失败: %v\n", err)
		} else {
			fmt.Printf("✅ 写入成功\n")
		}

		// 读取数据
		fmt.Printf("读取 D100...\n")
		value, err := reconnectClient.ReadDWord(100)
		if err != nil {
			fmt.Printf("❌ 读取失败: %v\n", err)
		} else {
			fmt.Printf("✅ 读取成功: D100 = %d\n", value)
		}

		// 字节数组操作
		data := []byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3)}
		fmt.Printf("写入字节数组到 D200: % X\n", data)
		err = reconnectClient.WriteDBytes(200, data)
		if err != nil {
			fmt.Printf("❌ 写入字节失败: %v\n", err)
		} else {
			fmt.Printf("✅ 写入字节成功\n")
		}

		// 显示统计信息
		stats := reconnectClient.GetStats()
		reconnectCount := reconnectClient.GetReconnectCount()
		fmt.Printf("\n📊 统计信息:\n")
		fmt.Printf("  总请求数: %d\n", stats.TotalRequests)
		fmt.Printf("  成功次数: %d\n", stats.SuccessCount)
		fmt.Printf("  错误次数: %d\n", stats.ErrorCount)
		fmt.Printf("  超时次数: %d\n", stats.TimeoutCount)
		fmt.Printf("  重连次数: %d\n", reconnectCount)

		if reconnectCount > 0 {
			lastReconnect := reconnectClient.GetLastReconnectTime()
			fmt.Printf("  最后重连: %s\n", lastReconnect.Format("2006-01-02 15:04:05"))
		}

		// 等待5秒后继续
		time.Sleep(5 * time.Second)
	}
}

// monitorConnection 监控连接状态
func monitorConnection(client *fins.ReconnectableClient) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		connected := client.IsConnected()
		reconnecting := client.IsReconnecting()

		status := "🟢 已连接"
		if reconnecting {
			status = "🟡 重连中..."
		} else if !connected {
			status = "🔴 未连接"
		}

		fmt.Printf("\r[监控] 连接状态: %s", status)
	}
}

