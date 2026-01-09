package main

import (
	"fmt"
	"log"

	"github.com/jianwushu/fins"
)

func main() {
	// 创建TCP客户端配置
	config := fins.DefaultConfig("192.168.1.10")
	config.LocalNode = 0x01
	config.ServerNode = 0x64

	// 创建TCP客户端
	client, err := fins.NewClient(config, true) // true表示使用TCP
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功通过TCP连接到PLC")

	// 示例1: 读取D100寄存器
	fmt.Println("\n=== 示例1: 读取D100寄存器 ===")
	value, err := client.ReadDWord(100)
	if err != nil {
		log.Printf("读取D100失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	// 示例2: 批量读取
	fmt.Println("\n=== 示例2: 批量读取D100-D109 ===")
	values, err := client.ReadDWords(100, 10)
	if err != nil {
		log.Printf("批量读取失败: %v", err)
	} else {
		for i, v := range values {
			fmt.Printf("D%d = %d\n", 100+i, v)
		}
	}

	// 示例3: 写入数据
	fmt.Println("\n=== 示例3: 写入D100 ===")
	if err := client.WriteDWord(100, 9999); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("成功写入D100 = 9999")

		// 验证写入
		if value, err := client.ReadDWord(100); err == nil {
			fmt.Printf("验证: D100 = %d\n", value)
		}
	}

	// 显示统计信息
	fmt.Println("\n=== 连接统计信息 ===")
	stats := client.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功次数: %d\n", stats.SuccessCount)
	fmt.Printf("错误次数: %d\n", stats.ErrorCount)
	fmt.Printf("超时次数: %d\n", stats.TimeoutCount)

	fmt.Println("\nTCP示例执行完成")
}
