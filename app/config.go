package app

type Config struct {
	ConnectAddr  string
	BindAddr     string
	ListenAddr   string
	BlockSize    uint32
	Count        int
	Download     bool
	Upload       bool
	TransferType string
	OnlyConnect  bool
	DisplaySpeed bool
	Socks5Addr   string
}

var DefaultAddress string = "127.0.0.1:8888"
var DefaultBlockSize uint32 = 64 * 1024
