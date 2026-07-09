package main

import (
	"flag"
	"runtime"

	"netspeed/app"
	"netspeed/client"
)

const autoServicePort = "1234"

func main() {
	caddress := flag.String("c", "", "connect address(client)")
	baddress := flag.String("B", "", "bind address(client)")
	saddress := flag.String("s", "", "listen address(server)")
	blocksize := flag.Uint64("b", uint64(app.DefaultBlockSize), "block_size")
	count := flag.Int("P", 1, "count for connect")
	download := flag.Bool("r", true, "connect download")
	upload := flag.Bool("w", true, "connect upload")
	transferType := flag.String("t", "tcp", "transfer type (tcp,kcp)")
	onlyConnect := flag.Bool("O", false, "connect only")
	socks5Addr := flag.String("S", "", "socks5 proxy address (client), e.g. 127.0.0.1:1080 or user:pass@127.0.0.1:1080")
	flag.Parse()

	noClient := *caddress == ""
	noServer := *saddress == ""
	if noClient && noServer {
		if runtime.GOOS == "linux" {
			app.RunServer(app.Config{
				ListenAddr:   ":" + autoServicePort,
				TransferType: *transferType,
			}, nil)
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

	cfg := app.Config{
		ConnectAddr:  *caddress,
		BindAddr:     *baddress,
		ListenAddr:   *saddress,
		BlockSize:    uint32(*blocksize),
		Count:        *count,
		Download:     *download,
		Upload:        *upload,
		TransferType: *transferType,
		OnlyConnect:  *onlyConnect,
		DisplaySpeed: true,
		Socks5Addr:   *socks5Addr,
	}

	if *caddress != "" {
		app.RunClient(cfg, nil)
	} else if *saddress != "" {
		app.RunServer(cfg, nil)
	}
}
