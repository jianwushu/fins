package main

import (
	"fmt"
	"log"

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

	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功通过TCP连接到PLC")

	fmt.Println("\n=== 示例1: 读取 D100 ===")
	value, err := client.ReadWord("D100")
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	fmt.Println("\n=== 示例2: 批量读取 D100-D109 ===")
	values, err := client.ReadWords("D100", 10)
	if err != nil {
		log.Printf("批量读取失败: %v", err)
	} else {
		for i, v := range values {
			fmt.Printf("D%d = %d\n", 100+i, v)
		}
	}

	fmt.Println("\n=== 示例3: 写入 D100 ===")
	if err := client.WriteWord("D100", 9999); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("成功写入 D100 = 9999")
		if value, err := client.ReadWord("D100"); err == nil {
			fmt.Printf("验证: D100 = %d\n", value)
		}
	}

	fmt.Println("\n=== 连接统计信息 ===")
	stats := client.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功次数: %d\n", stats.SuccessCount)
	fmt.Printf("错误次数: %d\n", stats.ErrorCount)
	fmt.Printf("超时次数: %d\n", stats.TimeoutCount)

	fmt.Println("\nTCP示例执行完成")
}
