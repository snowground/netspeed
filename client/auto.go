package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	AutoDiscoveryPort = "1235"
	AutoServicePort   = "1234"
	AutoTestDuration  = 10 * time.Second
)

func RunAutoMode() {
	addrs := DiscoverByBroadcast(AutoDiscoveryPort, 3*time.Second)
	if len(addrs) == 0 {
		log.Println("no server found (broadcast discovery)")
		return
	}
	fmt.Println("Discovered servers:")
	for i, a := range addrs {
		fmt.Printf("  [%d] %s\n", i+1, a)
	}
	var target string
	if len(addrs) == 1 {
		target = addrs[0]
	} else {
		fmt.Print("Select server (number or address): ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return
		}
		choice := strings.TrimSpace(sc.Text())
		if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(addrs) {
			target = addrs[n-1]
		} else {
			for _, a := range addrs {
				if a == choice {
					target = a
					break
				}
			}
		}
	}
	if target == "" {
		log.Println("invalid selection")
		return
	}
	fmt.Println("Confirm server:", target)

	transferType := "tcp"
	blocksize := default_block_size

	const conns = 3

	fmt.Println("Test 1: Download 10s (3 connections)...")
	deadline1 := time.Now().Add(AutoTestDuration)
	var wg sync.WaitGroup
	var d1, d2, d3 int64
	wg.Add(conns)
	go func() { defer wg.Done(); d1 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	go func() { defer wg.Done(); d2 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	go func() { defer wg.Done(); d3 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	wg.Wait()
	totalDown := d1 + d2 + d3
	fmt.Printf("  Download: %s in 10s -> %s/s\n", bytes2human(totalDown, 1000), bytes2human(totalDown/10, 1000))

	fmt.Println("Test 2: Upload 10s (3 connections)...")
	deadline2 := time.Now().Add(AutoTestDuration)
	var u1, u2, u3 int64
	wg.Add(conns)
	go func() { defer wg.Done(); u1 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	go func() { defer wg.Done(); u2 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	go func() { defer wg.Done(); u3 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	wg.Wait()
	totalUp := u1 + u2 + u3
	fmt.Printf("  Upload:   %s in 10s -> %s/s\n", bytes2human(totalUp, 1000), bytes2human(totalUp/10, 1000))

	fmt.Println("Test 3: Download + Upload 10s (3 connections, each conn both down+up)...")
	deadline3 := time.Now().Add(AutoTestDuration)
	var bd1, bd2, bd3, bu1, bu2, bu3 int64
	wg.Add(conns * 2)
	go func() { defer wg.Done(); bd1 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	go func() { defer wg.Done(); bd2 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	go func() { defer wg.Done(); bd3 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	go func() { defer wg.Done(); bu1 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	go func() { defer wg.Done(); bu2 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	go func() { defer wg.Done(); bu3 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline3) }()
	wg.Wait()
	bothDown := bd1 + bd2 + bd3
	bothUp := bu1 + bu2 + bu3
	fmt.Printf("  Download: %s in 10s -> %s/s\n", bytes2human(bothDown, 1000), bytes2human(bothDown/10, 1000))
	fmt.Printf("  Upload:   %s in 10s -> %s/s\n", bytes2human(bothUp, 1000), bytes2human(bothUp/10, 1000))

	fmt.Println("Done.")

	fmt.Print("Press Enter to exit... ")
	bufio.NewScanner(os.Stdin).Scan()
}
