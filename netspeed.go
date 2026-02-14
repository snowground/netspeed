package main

import (
	"flag"
	"runtime"
	"sync"

	"netspeed/client"
	"netspeed/server"
)

var wg sync.WaitGroup
var default_address string = "127.0.0.1:8888"
var default_block_size uint32 = 64 * 1024

const autoServicePort = "1234"

func main() {

	caddress := flag.String("c", "", "connect address(client)")
	baddress := flag.String("B", "", "bind address(client)")
	saddress := flag.String("s", "", "listen address(server)")
	blocksize := flag.Uint64("b", uint64(default_block_size), "block_size")
	count := flag.Int("P", 1, "count for connect")
	read := flag.Bool("r", true, "connect read")
	write := flag.Bool("w", true, "connect write")
	transferType := flag.String("t", "tcp", "transfer type (tcp,kcp)")
	onlyConnect := flag.Bool("O", false, "connect only")
	flag.Parse()

	noClient := *caddress == ""
	noServer := *saddress == ""
	if noClient && noServer {
		if runtime.GOOS == "linux" {
			wg.Add(1)
			server.ServerMain(":"+autoServicePort, *transferType, &wg)
			wg.Wait()
			return
		}
		if runtime.GOOS == "windows" {
			client.RunAutoMode()
			return
		}
	}
	if (noClient && noServer) || (!noClient && !noServer) {
		flag.PrintDefaults()
		return
	}
	onlyC := *caddress != "" && *saddress == "" && *baddress == "" &&
		*count == 1 && *read && *write && *transferType == "tcp" && !*onlyConnect
	if onlyC {
		client.RunAutoModeWithServer(*caddress)
		return
	}
	if *caddress != "" {
		for i := 0; i < *count; i++ {
			if *onlyConnect == true {
				wg.Add(1)
				go client.HandleOnlyConnect(*caddress, *baddress, *transferType, uint32(*blocksize), &wg)
			} else {
				if *read == true {
					wg.Add(1)
					go client.HandleRead(*caddress, *baddress, *transferType, uint32(*blocksize), &wg)
				}
				if *write == true {
					wg.Add(1)
					go client.HandleWrite(*caddress, *baddress, *transferType, uint32(*blocksize), &wg)
				}

			}

		}
		if *onlyConnect != true {
			go client.DispalySpeed()
		}

	} else if *saddress != "" {
		wg.Add(1)
		server.ServerMain(*saddress, *transferType, &wg)
	}

	wg.Wait()
}
