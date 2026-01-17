package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
)

func CheckNetwork() {
	fmt.Println("\n--- Network Status ---")
	// Cek IP Public
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		fmt.Println("❌ Internet: Disconnected")
		return
	}
	defer resp.Body.Close()
	ip, _ := ioutil.ReadAll(resp.Body)
	
	fmt.Printf("🌍 Public IP  : %s\n", string(ip))
	fmt.Println("✅ Internet   : Connected")
	fmt.Println("----------------------")
}

