package client

import (
	"bufio"
	"context"
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
	log.Println("Discovered servers:")
	for i, a := range addrs {
		log.Printf("  [%d] %s", i+1, a)
	}
	var target string
	if len(addrs) == 1 {
		target = addrs[0]
	} else {
		print("Select server (number or address): ")
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
	log.Println("Confirm server:", target)
	RunAutoTests(target, "tcp", default_block_size)
	log.Println("Done.")
	print("Press Enter to exit... ")
	bufio.NewScanner(os.Stdin).Scan()
}

func RunAutoTests(target, transferType string, blocksize uint32) {
	const conns = 3

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if ms, ok := UDPLatencyProbe(target); ok {
					log.Printf("  [UDP RTT] %s", FormatUDPLatency(ms))
				} else {
					log.Printf("  [UDP RTT] ---")
				}
			}
		}
	}()

	log.Println("Test 1: Download 10s (3 connections)...")
	deadline1 := time.Now().Add(AutoTestDuration)
	var wg sync.WaitGroup
	var d1, d2, d3 int64
	wg.Add(conns)
	go func() { defer wg.Done(); d1 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	go func() { defer wg.Done(); d2 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	go func() { defer wg.Done(); d3 = RunDownloadTestWithDeadline(target, transferType, blocksize, deadline1) }()
	wg.Wait()
	totalDown := d1 + d2 + d3
	log.Printf("  Download: %s in 10s -> %s/s", Bytes2Human(totalDown, 1000), Bytes2Human(totalDown/10, 1000))

	log.Println("Test 2: Upload 10s (3 connections)...")
	deadline2 := time.Now().Add(AutoTestDuration)
	var u1, u2, u3 int64
	wg.Add(conns)
	go func() { defer wg.Done(); u1 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	go func() { defer wg.Done(); u2 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	go func() { defer wg.Done(); u3 = RunUploadTestWithDeadline(target, transferType, blocksize, deadline2) }()
	wg.Wait()
	totalUp := u1 + u2 + u3
	log.Printf("  Upload:   %s in 10s -> %s/s", Bytes2Human(totalUp, 1000), Bytes2Human(totalUp/10, 1000))

	log.Println("Test 3: Download + Upload 10s (3 connections, each conn both down+up)...")
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
	log.Printf("  Download: %s in 10s -> %s/s", Bytes2Human(bothDown, 1000), Bytes2Human(bothDown/10, 1000))
	log.Printf("  Upload:   %s in 10s -> %s/s", Bytes2Human(bothUp, 1000), Bytes2Human(bothUp/10, 1000))
}

func RunAutoModeWithServer(target string) {
	RunAutoTests(target, "tcp", default_block_size)
	log.Println("Done.")
	print("Press Enter to exit... ")
	bufio.NewScanner(os.Stdin).Scan()
}
