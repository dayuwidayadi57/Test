test1
test2
test3
test4
func Logger(msg string) {
    now := time.Now().Format("15:04:05")
    fmt.Printf("[%s] LOG: %s\n", now, msg)
}
