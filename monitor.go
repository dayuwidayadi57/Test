package main

import (
	"fmt"
	"runtime"
)

func GetSystemStatus() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("\n--- System Health Check ---")
	fmt.Printf("📦 Allocated Memory : %v MB\n", m.Alloc/1024/1024)
	fmt.Printf("🧵 Total Goroutines : %v\n", runtime.NumGoroutine())
	fmt.Printf("⚙️  CPU Cores        : %v\n", runtime.NumCPU())
	fmt.Printf("💻 OS / Arch        : %s / %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("---------------------------")
}

