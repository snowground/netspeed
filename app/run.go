package app

import (
	"sync"

	"netspeed/client"
	"netspeed/server"
)

func RunClient(cfg Config, done func()) {
	client.SetSocks5Proxy(cfg.Socks5Addr)
	defer client.SetSocks5Proxy("")

	var wg sync.WaitGroup

	for i := 0; i < cfg.Count; i++ {
		if cfg.OnlyConnect {
			wg.Add(1)
			go client.HandleOnlyConnect(cfg.ConnectAddr, cfg.BindAddr, cfg.TransferType, cfg.BlockSize, &wg)
		} else {
			if cfg.Download {
				wg.Add(1)
				go client.HandleDownload(cfg.ConnectAddr, cfg.BindAddr, cfg.TransferType, cfg.BlockSize, &wg)
			}
			if cfg.Upload {
				wg.Add(1)
				go client.HandleUpload(cfg.ConnectAddr, cfg.BindAddr, cfg.TransferType, cfg.BlockSize, &wg)
			}
		}
	}

	if !cfg.OnlyConnect && cfg.DisplaySpeed {
		go client.DispalySpeed(cfg.ConnectAddr)
	}

	wg.Wait()
	if done != nil {
		done()
	}
}

func RunServer(cfg Config, done func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	server.ServerMain(cfg.ListenAddr, cfg.TransferType, &wg)
	wg.Wait()
	if done != nil {
		done()
	}
}

func RunAuto(target, transferType string, blocksize uint32, socks5Addr string, done func()) {
	client.SetSocks5Proxy(socks5Addr)
	defer client.SetSocks5Proxy("")

	client.RunAutoTests(target, transferType, blocksize)
	if done != nil {
		done()
	}
}

func StopClient() {
	client.StopAll()
}

func StopServer() {
	server.Stop()
}

func StopAuto() {
	client.StopAll()
}
