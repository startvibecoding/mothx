package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// 检查hostname
	hostname, err := os.Hostname()
	if err == nil && hostname == "sandbox" {
		fmt.Println("✅ 检测到 sandbox (hostname: sandbox)")
		return
	}

	// 检查环境变量
	for _, env := range os.Environ() {
		if strings.Contains(strings.ToLower(env), "sandbox") {
			fmt.Printf("✅ 检测到 sandbox (环境变量: %s)\n", env)
			return
		}
	}

	// 检查 /proc 文件系统
	if _, err := os.Stat("/proc/1/cgroup"); err == nil {
		data, err := os.ReadFile("/proc/1/cgroup")
		if err == nil {
			content := string(data)
			if strings.Contains(content, "sandbox") || strings.Contains(content, "bwrap") {
				fmt.Println("✅ 检测到 sandbox (cgroup)")
				return
			}
		}
	}

	fmt.Println("❌ 未检测到 sandbox")
}
