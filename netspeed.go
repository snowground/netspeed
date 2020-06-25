package main

import (
	"flag"
	"sync"

	"github.com/snowground/netspeed/client"
	"github.com/snowground/netspeed/server"
)

var wg sync.WaitGroup
var default_address string = "127.0.0.1:8888"
var default_block_size uint32 = 64 * 1024

func main() {

	caddress := flag.String("c", "", "connect address(client)")
	baddress := flag.String("B", "", "bind address(client)")
	saddress := flag.String("s", "", "listen address(server)")
	blocksize := flag.Uint64("b", uint64(default_block_size), "block_size")
	count := flag.Int("P", 1, "count for connect")
	read := flag.Bool("r", true, "connect read")
	write := flag.Bool("w", true, "connect write")
	transferType := flag.String("t", "tcp", "transfer type (tcp,kcp)")

	flag.Parse()

	if (*caddress == "" && *saddress == "") ||
		(*caddress != "" && *saddress != "") {
		flag.PrintDefaults()
	}
	if *caddress != "" {
		for i := 0; i < *count; i++ {
			if *read == true {
				wg.Add(1)
				go client.HandleRead(*caddress, *baddress, *transferType, uint32(*blocksize), &wg)
			}
			if *write == true {
				wg.Add(1)
				go client.HandleWrite(*caddress, *baddress, *transferType, uint32(*blocksize), &wg)
			}
		}
		go client.DispalySpeed()
	} else if *saddress != "" {
		wg.Add(1)
		server.ServerMain(*saddress, *transferType, &wg)
	}

	wg.Wait()
}
