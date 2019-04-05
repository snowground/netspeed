package main

import (
	"flag"
	"netspeed/client"
	"netspeed/server"
	"sync"
)

var wg sync.WaitGroup
var default_address string = "127.0.0.1:8888"
var default_block_size uint32 = 64 * 1024

func main() {

	caddress := flag.String("c", "", "connect address(client)")
	saddress := flag.String("s", "", "listen address(server)")
	blocksize := flag.Uint64("b", uint64(default_block_size), "block_size")
	count := flag.Int("P", 1, "count for connect")
	read := flag.Bool("r", true, "connect read")
	write := flag.Bool("w", true, "connect write")

	flag.Parse()

	if (*caddress == "" && *saddress == "") ||
		(*caddress != "" && *saddress != "") {
		flag.PrintDefaults()
	}
	if *caddress != "" {
		for i := 0; i < *count; i++ {
			if *read == true {
				wg.Add(1)
				go client.HandleRead(*caddress, uint32(*blocksize), &wg)
			}
			if *write == true {
				wg.Add(1)
				go client.HandleWrite(*caddress, uint32(*blocksize), &wg)
			}
		}
		go client.DispalySpeed()
	} else if *saddress != "" {
		wg.Add(1)
		server.ServerMain(*saddress, &wg)
	}

	wg.Wait()
}
