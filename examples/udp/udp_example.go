package main

import (
	"fmt"
	"log"

	"github.com/jianwushu/fins"
)

func main() {
	// 创建UDP客户端配置
	config := fins.DefaultConfig("127.0.0.1")
	config.LocalNode = 0x01
	config.ServerNode = 0x64 // PLC的FINS节点地址
	config.SIDMode = fins.SIDFixed
	config.FixedSID = 0x00

	// 创建UDP客户端
	client, err := fins.NewClient(config, false) // false表示使用UDP
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功连接到PLC")

	// 示例1: 读取D100寄存器
	fmt.Println("\n=== 示例1: 读取D100寄存器 ===")
	value, err := client.ReadDWord(100)
	if err != nil {
		log.Printf("读取D100失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	// 示例2: 读取D200-D204（5个寄存器）
	fmt.Println("\n=== 示例2: 读取D200-D204 ===")
	values, err := client.ReadDWords(200, 5)
	if err != nil {
		log.Printf("读取D200-D204失败: %v", err)
	} else {
		for i, v := range values {
			fmt.Printf("D%d = %d (0x%04X)\n", 200+i, v, v)
		}
	}

	// 示例3: 写入D100
	fmt.Println("\n=== 示例3: 写入D100 ===")
	if err := client.WriteDWord(100, 1234); err != nil {
		log.Printf("写入D100失败: %v", err)
	} else {
		fmt.Println("成功写入D100 = 1234")
	}

	// 示例4: 批量写入D200-D204
	fmt.Println("\n=== 示例4: 批量写入D200-D204 ===")
	writeValues := []uint16{100, 200, 300, 400, 500}
	if err := client.WriteDWords(200, writeValues); err != nil {
		log.Printf("批量写入失败: %v", err)
	} else {
		fmt.Println("成功批量写入D200-D204")
	}

	// 示例5: 读取CIO区位
	fmt.Println("\n=== 示例5: 读取CIO0.00位 ===")
	bitValue, err := client.ReadCIOBit(0, 0)
	if err != nil {
		log.Printf("读取CIO0.00失败: %v", err)
	} else {
		fmt.Printf("CIO0.00 = %v\n", bitValue)
	}

	// 示例6: 写入CIO区位
	fmt.Println("\n=== 示例6: 写入CIO0.00位 ===")
	if err := client.WriteCIOBit(0, 0, true); err != nil {
		log.Printf("写入CIO0.00失败: %v", err)
	} else {
		fmt.Println("成功写入CIO0.00 = true")
	}

	// 示例7: 读取通用内存区域
	fmt.Println("\n=== 示例7: 读取HR区 ===")
	hrData, err := client.ReadMemoryArea(fins.MemAreaHR, 0, 10)
	if err != nil {
		log.Printf("读取HR区失败: %v", err)
	} else {
		fmt.Printf("读取到%d字节数据\n", len(hrData))
	}

	// 显示统计信息
	fmt.Println("\n=== 连接统计信息 ===")
	stats := client.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功次数: %d\n", stats.SuccessCount)
	fmt.Printf("错误次数: %d\n", stats.ErrorCount)
	fmt.Printf("超时次数: %d\n", stats.TimeoutCount)
	fmt.Printf("最后请求时间: %v\n", stats.LastRequestAt)
	fmt.Printf("最后响应时间: %v\n", stats.LastResponseAt)

	fmt.Println("\n示例执行完成")
}
