package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/jianwushu/fins"
)

func main() {
	// 创建UDP客户端配置
	config := fins.DefaultConfig("192.168.1.10")
	config.LocalNode = 0x01
	config.ServerNode = 0x64

	// 创建UDP客户端
	client, err := fins.NewClient(config, false)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功连接到PLC")

	// ========== 示例1: 读取字节数组 ==========
	fmt.Println("\n=== 示例1: 读取D100开始的10个字节 ===")
	data, err := client.ReadDBytes(100, 10)
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("读取到 %d 字节: %v\n", len(data), data)
		fmt.Printf("十六进制: % X\n", data)
	}

	// ========== 示例2: 写入字节数组 ==========
	fmt.Println("\n=== 示例2: 写入字节数组到D200 ===")
	writeData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := client.WriteDBytes(200, writeData); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Printf("成功写入 %d 字节: % X\n", len(writeData), writeData)

		// 验证写入
		readBack, err := client.ReadDBytes(200, uint16(len(writeData)))
		if err == nil {
			fmt.Printf("验证读取: % X\n", readBack)
		}
	}

	// ========== 示例3: 处理奇数字节 ==========
	fmt.Println("\n=== 示例3: 写入奇数字节(7字节) ===")
	oddData := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11}
	if err := client.WriteDBytes(300, oddData); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Printf("成功写入 %d 字节: % X\n", len(oddData), oddData)

		// 读取回来验证
		readBack, err := client.ReadDBytes(300, uint16(len(oddData)))
		if err == nil {
			fmt.Printf("验证读取: % X\n", readBack)
		}
	}

	// ========== 示例4: 字节数组与整数转换 ==========
	fmt.Println("\n=== 示例4: 字节数组与整数转换 ===")

	// 将整数转换为字节数组并写入
	value := uint32(0x12345678)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, value)
	fmt.Printf("写入整数 0x%08X 为字节: % X\n", value, buf)

	if err := client.WriteDBytes(400, buf); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		// 读取并转换回整数
		readData, err := client.ReadDBytes(400, 4)
		if err == nil {
			readValue := binary.BigEndian.Uint32(readData)
			fmt.Printf("读取字节: % X\n", readData)
			fmt.Printf("转换为整数: 0x%08X (%d)\n", readValue, readValue)
		}
	}

	// ========== 示例5: 字符串操作 ==========
	fmt.Println("\n=== 示例5: 写入和读取字符串 ===")
	text := "FINS Protocol"
	textBytes := []byte(text)
	fmt.Printf("写入字符串: \"%s\" (%d字节)\n", text, len(textBytes))

	if err := client.WriteDBytes(500, textBytes); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		// 读取字符串
		readData, err := client.ReadDBytes(500, uint16(len(textBytes)))
		if err == nil {
			readText := string(readData)
			fmt.Printf("读取字符串: \"%s\"\n", readText)
		}
	}

	// ========== 示例6: 不同内存区域的字节操作 ==========
	fmt.Println("\n=== 示例6: 操作不同内存区域 ===")

	// CIO区
	cioData := []byte{0x01, 0x02, 0x03, 0x04}
	if err := client.WriteCIOBytes(0, cioData); err != nil {
		log.Printf("写入CIO区失败: %v", err)
	} else {
		fmt.Printf("成功写入CIO区: % X\n", cioData)
	}

	// HR区
	hrData := []byte{0x11, 0x22, 0x33, 0x44}
	if err := client.WriteHRBytes(0, hrData); err != nil {
		log.Printf("写入HR区失败: %v", err)
	} else {
		fmt.Printf("成功写入HR区: % X\n", hrData)
	}

	// WR区
	wrData := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	if err := client.WriteWRBytes(0, wrData); err != nil {
		log.Printf("写入WR区失败: %v", err)
	} else {
		fmt.Printf("成功写入WR区: % X\n", wrData)
	}

	// ========== 示例7: 批量数据传输 ==========
	fmt.Println("\n=== 示例7: 批量数据传输(100字节) ===")
	largeData := make([]byte, 100)
	for i := range largeData {
		largeData[i] = byte(i)
	}

	if err := client.WriteDBytes(600, largeData); err != nil {
		log.Printf("批量写入失败: %v", err)
	} else {
		fmt.Printf("成功写入 %d 字节\n", len(largeData))

		// 读取验证
		readData, err := client.ReadDBytes(600, uint16(len(largeData)))
		if err == nil {
			fmt.Printf("成功读取 %d 字节\n", len(readData))
			// 验证数据一致性
			match := true
			for i := range largeData {
				if readData[i] != largeData[i] {
					match = false
					break
				}
			}
			if match {
				fmt.Println("✓ 数据验证成功,读写一致!")
			} else {
				fmt.Println("✗ 数据验证失败,读写不一致!")
			}
		}
	}

	// 显示统计信息
	fmt.Println("\n=== 连接统计信息 ===")
	stats := client.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功次数: %d\n", stats.SuccessCount)
	fmt.Printf("错误次数: %d\n", stats.ErrorCount)

	fmt.Println("\n字节数组操作示例执行完成")
}
