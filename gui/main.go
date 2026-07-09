package main

import (
	"log"
	"strconv"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"netspeed/app"
	"netspeed/client"
)

const (
	modeClient = 0
	modeServer = 1
	modeAuto   = 2
)

type GUI struct {
	mw *walk.MainWindow

	cbMode       *walk.ComboBox
	leServerAddr *walk.LineEdit
	leBindAddr   *walk.LineEdit
	leSocks5Addr *walk.LineEdit
	leBlockSize  *walk.LineEdit
	leCount      *walk.LineEdit
	cbTransfer   *walk.ComboBox
	cbDownload   *walk.CheckBox
	cbUpload     *walk.CheckBox
	cbOnlyConn   *walk.CheckBox

	teOutput     *walk.TextEdit
	lblDownSpeed *walk.Label
	lblUpSpeed   *walk.Label
	lblRTT       *walk.Label
	btnStart     *walk.PushButton
	btnDiscover  *walk.PushButton

	running  bool
	stopCh   chan struct{}
	lastDown int64
	lastUp   int64
	runMode  int
}

func main() {
	g := &GUI{}
	saved := loadConfig()

	log.SetOutput(&logWriter{gui: g})

	defMode := 0
	defServerAddr := app.DefaultAddress
	defBindAddr := ""
	defSocks5Addr := ""
	defBlockSize := strconv.FormatUint(uint64(app.DefaultBlockSize), 10)
	defCount := "1"
	defTransferIndex := 0
	defDownload := true
	defUpload := true
	defOnlyConnect := false

	if saved != nil {
		defMode = saved.Mode
		if saved.ServerAddr != "" {
			defServerAddr = saved.ServerAddr
		}
		defBindAddr = saved.BindAddr
		defSocks5Addr = saved.Socks5Addr
		if saved.BlockSize != "" {
			defBlockSize = saved.BlockSize
		}
		if saved.Count != "" {
			defCount = saved.Count
		}
		defTransferIndex = saved.TransferIndex
		defDownload = saved.Download
		defUpload = saved.Upload
		defOnlyConnect = saved.OnlyConnect
	}

	mw := MainWindow{
		AssignTo: &g.mw,
		Title:    "netspeed GUI",
		Size:     Size{Width: 620, Height: 580},
		MinSize:  Size{Width: 520, Height: 480},
		Layout:   VBox{MarginsZero: false, Spacing: 8},
		Children: []Widget{
			GroupBox{
				Title:  "Mode",
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "Mode:"},
					ComboBox{
						AssignTo:     &g.cbMode,
						Model:        []string{"Client", "Server", "Auto"},
						CurrentIndex: defMode,
					},
				},
			},
			GroupBox{
				Title: "Parameters",
				Layout: Grid{
					Columns: 2,
					Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8},
				},
				Children: []Widget{
					Label{Text: "Server Address:"},
					LineEdit{AssignTo: &g.leServerAddr, Text: defServerAddr},

					Label{Text: "Bind Address:"},
					LineEdit{AssignTo: &g.leBindAddr, Text: defBindAddr},

					Label{Text: "SOCKS5 Proxy:"},
					LineEdit{AssignTo: &g.leSocks5Addr, Text: defSocks5Addr},

					Label{Text: "Block Size:"},
					LineEdit{AssignTo: &g.leBlockSize, Text: defBlockSize},

					Label{Text: "Connections:"},
					LineEdit{AssignTo: &g.leCount, Text: defCount},

					Label{Text: "Transfer Type:"},
					ComboBox{AssignTo: &g.cbTransfer, Model: []string{"tcp", "kcp"}, CurrentIndex: defTransferIndex},

					Composite{
						ColumnSpan: 2,
						Layout:     HBox{MarginsZero: true},
						Children: []Widget{
					CheckBox{AssignTo: &g.cbDownload, Text: "Download", Checked: defDownload},
					CheckBox{AssignTo: &g.cbUpload, Text: "Upload", Checked: defUpload},
						CheckBox{AssignTo: &g.cbOnlyConn, Text: "Connect Only", Checked: defOnlyConnect},
						},
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						Text:     "Start",
						AssignTo: &g.btnStart,
						OnClicked: func() {
							g.onStart()
						},
					},
					PushButton{
						Text:     "Discover Servers",
						AssignTo: &g.btnDiscover,
						OnClicked: func() {
							g.onDiscover()
						},
					},
					PushButton{
						Text: "Clear Output",
						OnClicked: func() {
							g.teOutput.SetText("")
						},
					},
				},
			},
			GroupBox{
				Title:  "Speed",
				Layout: Grid{Columns: 2, MarginsZero: true},
				Children: []Widget{
					Label{Text: "Download:", MinSize: Size{Width: 80, Height: 0}},
					Label{Text: "0.00 B/s", AssignTo: &g.lblDownSpeed, MinSize: Size{Width: 150, Height: 0}},

					Label{Text: "Upload:", MinSize: Size{Width: 80, Height: 0}},
					Label{Text: "0.00 B/s", AssignTo: &g.lblUpSpeed, MinSize: Size{Width: 150, Height: 0}},

					Label{Text: "UDP RTT:", MinSize: Size{Width: 80, Height: 0}},
					Label{Text: "---", AssignTo: &g.lblRTT, MinSize: Size{Width: 150, Height: 0}},
				},
			},
			TextEdit{
				AssignTo:      &g.teOutput,
				ReadOnly:      true,
				StretchFactor: 1,
				MinSize:       Size{Height: 150},
			},
		},
	}

	if _, err := mw.Run(); err != nil {
		log.Fatal(err)
	}
}

func (g *GUI) getMode() int {
	return g.cbMode.CurrentIndex()
}

func (g *GUI) setRunning(running bool) {
	g.running = running
	if running {
		g.btnStart.SetText("Stop")
		g.btnDiscover.SetEnabled(false)
	} else {
		g.btnStart.SetText("Start")
		g.btnDiscover.SetEnabled(true)
	}
}

func (g *GUI) onStart() {
	if g.running {
		if g.stopCh != nil {
			close(g.stopCh)
			g.stopCh = nil
		}
		switch g.runMode {
		case modeClient:
			app.StopClient()
		case modeServer:
			app.StopServer()
		case modeAuto:
			app.StopAuto()
		}
		client.ResetCounters()
		g.lastDown = 0
		g.lastUp = 0
		g.lblDownSpeed.SetText("0.00 B/s")
		g.lblUpSpeed.SetText("0.00 B/s")
		g.setRunning(false)
		return
	}

	mode := g.getMode()
	serverAddr := g.leServerAddr.Text()
	bindAddr := g.leBindAddr.Text()
	socks5Addr := g.leSocks5Addr.Text()
	blockSize, err := strconv.ParseUint(g.leBlockSize.Text(), 10, 32)
	if err != nil {
		walk.MsgBox(g.mw, "Error", "Invalid block size", walk.MsgBoxIconError)
		return
	}
	count, err := strconv.Atoi(g.leCount.Text())
	if err != nil || count < 1 {
		count = 1
	}
	transferType := g.cbTransfer.Text()
	transferIndex := g.cbTransfer.CurrentIndex()
	download := g.cbDownload.Checked()
	upload := g.cbUpload.Checked()
	onlyConnect := g.cbOnlyConn.Checked()

	saveConfig(&guiConfig{
		Mode:          mode,
		ServerAddr:    serverAddr,
		BindAddr:      bindAddr,
		Socks5Addr:    socks5Addr,
		BlockSize:     strconv.FormatUint(blockSize, 10),
		Count:         strconv.Itoa(count),
		TransferIndex: transferIndex,
		Download:      download,
		Upload:        upload,
		OnlyConnect:   onlyConnect,
	})

	client.ResetCounters()
	g.stopCh = make(chan struct{})
	g.lastDown = 0
	g.lastUp = 0
	g.runMode = mode
	g.setRunning(true)

	switch mode {
	case modeClient:
		cfg := app.Config{
			ConnectAddr:  serverAddr,
			BindAddr:     bindAddr,
			BlockSize:    uint32(blockSize),
			Count:        count,
			Download:     download,
			Upload:        upload,
			TransferType: transferType,
			OnlyConnect:  onlyConnect,
			DisplaySpeed: false,
			Socks5Addr:   socks5Addr,
		}
		g.startSpeedMonitor(serverAddr)
		go func() {
			app.RunClient(cfg, func() {
				g.mw.Synchronize(func() {
					g.setRunning(false)
				})
			})
		}()

	case modeServer:
		listenAddr := serverAddr
		if listenAddr == "" {
			listenAddr = app.DefaultAddress
		}
		cfg := app.Config{
			ListenAddr:   listenAddr,
			TransferType: transferType,
		}
		go func() {
			app.RunServer(cfg, func() {
				g.mw.Synchronize(func() {
					g.setRunning(false)
				})
			})
		}()

	case modeAuto:
		if serverAddr == "" {
			log.Println("Auto mode: discovering servers...")
			addrs := client.DiscoverByBroadcast(client.AutoDiscoveryPort, 3*time.Second)
			if len(addrs) == 0 {
				log.Println("no server found")
				g.mw.Synchronize(func() {
					g.setRunning(false)
				})
				return
			}
			serverAddr = addrs[0]
			log.Println("Using server:", serverAddr)
			g.mw.Synchronize(func() {
				g.leServerAddr.SetText(serverAddr)
			})
		}
		target := serverAddr
		g.startSpeedMonitor(target)
		go func() {
			app.RunAuto(target, transferType, uint32(blockSize), socks5Addr, func() {
				g.mw.Synchronize(func() {
					g.setRunning(false)
				})
			})
		}()
	}
}

func (g *GUI) onDiscover() {
	g.btnDiscover.SetEnabled(false)
	go func() {
		addrs := client.DiscoverByBroadcast(client.AutoDiscoveryPort, 3*time.Second)
		g.mw.Synchronize(func() {
			g.btnDiscover.SetEnabled(true)
			if len(addrs) == 0 {
				walk.MsgBox(g.mw, "Discover", "No servers found", walk.MsgBoxIconInformation)
				return
			}
			if len(addrs) == 1 {
				g.leServerAddr.SetText(addrs[0])
				return
			}
			g.showServerSelection(addrs)
		})
	}()
}

func (g *GUI) showServerSelection(addrs []string) {
	var dlg *walk.Dialog
	var lb *walk.ListBox

	items := make([]string, len(addrs))
	for i, a := range addrs {
		items[i] = strconv.Itoa(i+1) + ". " + a
	}

	if err := (Dialog{
		AssignTo: &dlg,
		Title:    "Select Server",
		Size:     Size{Width: 320, Height: 250},
		Layout:   VBox{},
		Children: []Widget{
			ListBox{
				AssignTo: &lb,
				Model:    items,
			},
			PushButton{
				Text: "OK",
				OnClicked: func() {
					idx := lb.CurrentIndex()
					if idx >= 0 && idx < len(addrs) {
						g.leServerAddr.SetText(addrs[idx])
					}
					dlg.Accept()
				},
			},
		},
	}).Create(g.mw); err != nil {
		log.Println("dialog create error:", err)
		return
	}

	dlg.Run()
}

func (g *GUI) startSpeedMonitor(serverAddr string) {
	stopCh := g.stopCh
	go func() {
		lastRTT := "---"
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			nowDown := client.GetTotalDownload()
			nowUp := client.GetTotalUpload()
			downSpeed := nowDown - g.lastDown
			upSpeed := nowUp - g.lastUp
			g.lastDown = nowDown
			g.lastUp = nowUp

			if serverAddr != "" {
				if ms, ok := client.UDPLatencyProbe(serverAddr); ok {
					lastRTT = client.FormatUDPLatency(ms)
				}
			}

			g.mw.Synchronize(func() {
				g.lblDownSpeed.SetText(client.Bytes2Human(downSpeed, 1000) + "/s")
				g.lblUpSpeed.SetText(client.Bytes2Human(upSpeed, 1000) + "/s")
				g.lblRTT.SetText(lastRTT)
			})

			time.Sleep(time.Second)
		}
	}()
}

type logWriter struct {
	gui *GUI
}

func (w *logWriter) Write(p []byte) (int, error) {
	text := string(p)
	if w.gui.mw != nil {
		w.gui.mw.Synchronize(func() {
			current := w.gui.teOutput.Text()
			current += "\r\n"
			w.gui.teOutput.SetText(current + text)
		})
	}
	return len(p), nil
}
