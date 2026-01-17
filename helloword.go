package main

import (
	"fmt"
	"time"
)
func Logger(msg string) {
    now := time.Now().Format("15:04:05")
    fmt.Printf("[%s] LOG: %s\n", now, msg)
}
