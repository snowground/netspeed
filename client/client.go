package client

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"netspeed/protocol"
	"netspeed/transfer"
)

var total_download int64 = 0
var total_upload int64 = 0

var (
	activeConns   []transfer.Conn
	activeConnsMu sync.Mutex
)

func registerConn(c transfer.Conn) transfer.Conn {
	activeConnsMu.Lock()
	activeConns = append(activeConns, c)
	activeConnsMu.Unlock()
	return c
}

func unregisterConn(c transfer.Conn) {
	activeConnsMu.Lock()
	for i, ac := range activeConns {
		if ac == c {
			activeConns = append(activeConns[:i], activeConns[i+1:]...)
			break
		}
	}
	activeConnsMu.Unlock()
}

func StopAll() {
	activeConnsMu.Lock()
	for _, c := range activeConns {
		c.Close()
	}
	activeConns = nil
	activeConnsMu.Unlock()
}

func GetTotalDownload() int64 {
	return atomic.LoadInt64(&total_download)
}

func GetTotalUpload() int64 {
	return atomic.LoadInt64(&total_upload)
}

func ResetCounters() {
	atomic.StoreInt64(&total_download, 0)
	atomic.StoreInt64(&total_upload, 0)
}

var wg sync.WaitGroup
var default_address string = "127.0.0.1:8888"
var default_block_size uint32 = 64 * 1024

func factorial(n int) uint64 {
	var facVal uint64 = 1
	if n < 0 {
		fmt.Print("Factorial of negative number doesn't exist.")
	} else {
		for i := 1; i <= n; i++ {
			facVal *= uint64(i)
		}
	}
	return facVal

}
func Bytes2Human(n int64, base int64) (str string) {

	symbols := []string{"K", "M", "G", "T", "P", "E"}
	prefix := make(map[string]int64)
	for i, s := range symbols {
		if i == 0 {
			prefix[s] = base
		} else {
			prefix[s] = prefix[symbols[i-1]] * base
		}
	}

	for i := len(symbols) - 1; i >= 0; i-- {
		s := symbols[i]
		if n >= prefix[s] {
			value := float64(n) / float64(prefix[s])
			return fmt.Sprintf("%8.2f %s", value, s)
		}
	}
	return fmt.Sprintf("%8.2f B", float64(n))
}

func UDPLatencyProbe(serverAddr string) (ms float64, ok bool) {
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return 0, false
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return 0, false
	}
	echoAddr := net.JoinHostPort(host, strconv.Itoa(portNum+1))
	conn, err := net.DialTimeout("udp", echoAddr, 5*time.Second)
	if err != nil {
		return 0, false
	}
	defer conn.Close()
	payload := []byte("ping")
	start := time.Now()
	if _, err := conn.Write(payload); err != nil {
		return 0, false
	}
	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Read(buf); err != nil {
		return 0, false
	}
	return float64(time.Since(start).Nanoseconds()) / 1e6, true
}

func FormatUDPLatency(ms float64) string {
	if ms < 0.001 {
		return "< 0.001 ms"
	}
	return fmt.Sprintf("%.3f ms", ms)
}

func connectServer(serverAddr string, localAddr string, transferType string) (transfer.Conn, error) {
	var l transfer.Conn
	var err error

	switch transferType {
	case "tcp":
		l, err = transfer.TcpConnect(serverAddr, localAddr)
	case "kcp":
		l, err = transfer.KcpConnect(serverAddr, localAddr)
	default:
		wg.Done()
		return nil, errors.New("transferType error")
	}

	return l, err
}
func HandleOnlyConnect(serverAddr string, localAddr string, transferType string, blocksize uint32, wg *sync.WaitGroup) {
	c, err := connectServer(serverAddr, localAddr, transferType)
	if err != nil {
		log.Println("dial error:", err)
		goto exit
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	for {
		time.Sleep(time.Second * 60)
	}

exit:
	wg.Done()
}
func HandleDownload(serverAddr string, localAddr string, transferType string, blocksize uint32, wg *sync.WaitGroup) {
	var rwbuf = make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_DOWNLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)
	var n int

	c, err := connectServer(serverAddr, localAddr, transferType)
	if err != nil {
		log.Println("dial error:", err)
		goto exit
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	n, err = c.Write(buf)
	if err != nil || n < 0 {
		log.Println("conn Write header error:", err)
		goto exit
	}
	log.Printf("handle_download to conn:%s %s blocksize:%d", c.RemoteAddr(), transferType, blocksize)

	for {
		n, err = c.Read(rwbuf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			break
		}
		atomic.AddInt64(&total_download, int64(n))
	}
exit:
	wg.Done()
}
func HandleUpload(serverAddr string, localAddr string, transferType string, blocksize uint32, wg *sync.WaitGroup) {
	var rwbuf = make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_UPLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)
	var n int
	c, err := connectServer(serverAddr, localAddr, transferType)
	if err != nil {
		log.Println("dial error:", err)
		goto exit
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))

	n, err = c.Write(buf)
	if err != nil || n < 0 {
		log.Println("conn Write header error:", err)
		goto exit
	}
	log.Printf("handle_upload to conn:%s %s blocksize:%d", c.RemoteAddr(), transferType, blocksize)

	for {
		n, err = c.Write(rwbuf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			break
		}
		atomic.AddInt64(&total_upload, int64(n))
	}
exit:
	wg.Done()
}

func DispalySpeed(serverAddr string) {
	var last_up int64 = 0
	var last_down int64 = 0

	g_can_down := true
	g_can_up := false
	swap_time := 0
	limiter := time.Tick(time.Second * 1)

	test_time := time.Now()
	for {
		<-limiter

		now_time := time.Now()

		if now_time.Sub(test_time).Seconds() > float64(swap_time) {
			g_can_down, g_can_up = g_can_up, g_can_down
			test_time = now_time
		}
	now_up := atomic.LoadInt64(&total_upload)
	now_down := atomic.LoadInt64(&total_download)
		latencyStr := "---"
		if serverAddr != "" {
			if ms, ok := UDPLatencyProbe(serverAddr); ok {
				latencyStr = FormatUDPLatency(ms)
			}
		}
		log.Printf("down:%s/s     up:%s/s     udp_rtt:%s", Bytes2Human(now_down-last_down, 1000), Bytes2Human(now_up-last_up, 1000), latencyStr)
		last_up = now_up
		last_down = now_down
	}
}

func start_timer(myTimer func(), sec uint32) {
	timer1 := time.NewTicker(time.Duration(sec) * time.Second)
	for {
		select {
		case <-timer1.C:
			myTimer()
		}
	}
}

func RunDownloadTest(serverAddr string, transferType string, blocksize uint32, duration time.Duration) int64 {
	var total int64
	rwbuf := make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_DOWNLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		log.Println("RunDownloadTest dial error:", err)
		return 0
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	deadline := time.Now().Add(duration)
	c.SetDeadline(deadline, deadline)

	if n, err := c.Write(buf); err != nil || n < 0 {
		log.Println("RunDownloadTest write header error:", err)
		return 0
	}
	for {
		n, err := c.Read(rwbuf)
		if err != nil || n <= 0 {
			break
		}
		total += int64(n)
	}
	return total
}

func RunUploadTest(serverAddr string, transferType string, blocksize uint32, duration time.Duration) int64 {
	var total int64
	rwbuf := make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_UPLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		log.Println("RunUploadTest dial error:", err)
		return 0
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	deadline := time.Now().Add(duration)
	c.SetDeadline(deadline, deadline)

	if n, err := c.Write(buf); err != nil || n < 0 {
		log.Println("RunUploadTest write header error:", err)
		return 0
	}
	for {
		n, err := c.Write(rwbuf)
		if err != nil || n <= 0 {
			break
		}
		total += int64(n)
	}
	return total
}

func RunBidirectionalTest(serverAddr string, transferType string, blocksize uint32, duration time.Duration) (down int64, up int64) {
	deadline := time.Now().Add(duration)
	var downResult, upResult int64
	var bw sync.WaitGroup
	bw.Add(2)
	go func() {
		defer bw.Done()
		downResult = RunDownloadTestWithDeadline(serverAddr, transferType, blocksize, deadline)
	}()
	go func() {
		defer bw.Done()
		upResult = RunUploadTestWithDeadline(serverAddr, transferType, blocksize, deadline)
	}()
	bw.Wait()
	return downResult, upResult
}

func RunDownloadTestWithDeadline(serverAddr string, transferType string, blocksize uint32, deadline time.Time) int64 {
	var total int64
	rwbuf := make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_DOWNLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		return 0
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	c.SetDeadline(deadline, deadline)
	if _, err := c.Write(buf); err != nil {
		return 0
	}
	for {
		n, err := c.Read(rwbuf)
		if err != nil || n <= 0 {
			break
		}
		total += int64(n)
	}
	return total
}

func RunUploadTestWithDeadline(serverAddr string, transferType string, blocksize uint32, deadline time.Time) int64 {
	var total int64
	rwbuf := make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_UPLOAD
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		return 0
	}
	registerConn(c)
	defer unregisterConn(c)
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	c.SetDeadline(deadline, deadline)
	if _, err := c.Write(buf); err != nil {
		return 0
	}
	for {
		n, err := c.Write(rwbuf)
		if err != nil || n <= 0 {
			break
		}
		total += int64(n)
	}
	return total
}
