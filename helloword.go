package main

import (
	"fmt"
	"time"
)

func Logger(msg string) {
    now := time.Now().Format("15:04:05")
    fmt.Printf("[%s] LOG: %s\n", now, msg)
}

func main() {
	Logger("Hello World dari GoSmartPush v17.2!")
	Logger("Sistem @dev_dayuwidayadi siap tempur!")
}