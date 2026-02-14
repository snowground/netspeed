package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"netspeed/protocol"
	"netspeed/transfer"
)

var total_read int64 = 0
var total_write int64 = 0

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
func bytes2human(n int64, base int64) (str string) {

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
	defer c.Close()
	for {
		time.Sleep(time.Second * 60)
	}

exit:
	wg.Done()
}
func HandleRead(serverAddr string, localAddr string, transferType string, blocksize uint32, wg *sync.WaitGroup) {
	var rwbuf = make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_READ
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)
	var n int

	c, err := connectServer(serverAddr, localAddr, transferType)
	if err != nil {
		log.Println("dial error:", err)
		goto exit
	}
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))
	n, err = c.Write(buf)
	if err != nil || n < 0 {
		log.Println("conn Write header error:", err)
		goto exit
	}
	log.Printf("handle_read to conn:%s %s blocksize:%d", c.RemoteAddr(), transferType, blocksize)

	for {
		n, err = c.Read(rwbuf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			break
		}
		atomic.AddInt64(&total_read, int64(n))
	}
exit:
	wg.Done()
}
func HandleWrite(serverAddr string, localAddr string, transferType string, blocksize uint32, wg *sync.WaitGroup) {
	var rwbuf = make([]byte, blocksize)
	var header protocol.Header
	header.Sig = protocol.HEADER_SIG
	header.Func = protocol.HEADER_FUNC_WRITE
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)
	var n int
	c, err := connectServer(serverAddr, localAddr, transferType)
	if err != nil {
		log.Println("dial error:", err)
		goto exit
	}
	defer c.Close()
	c.SetBuffer(int(blocksize), int(blocksize))

	n, err = c.Write(buf)
	if err != nil || n < 0 {
		log.Println("conn Write header error:", err)
		goto exit
	}
	log.Printf("handle_write to conn:%s %s blocksize:%d", c.RemoteAddr(), transferType, blocksize)

	for {
		n, err = c.Write(rwbuf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			break
		}
		atomic.AddInt64(&total_write, int64(n))
	}
exit:
	wg.Done()
}

func DispalySpeed() {
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
		now_up := atomic.LoadInt64(&total_write)
		now_down := atomic.LoadInt64(&total_read)
		log.Printf("down:%s/s     up:%s/s ...", bytes2human(now_down-last_down, 1000), bytes2human(now_up-last_up, 1000))
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
	header.Func = protocol.HEADER_FUNC_READ
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		log.Println("RunDownloadTest dial error:", err)
		return 0
	}
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
	header.Func = protocol.HEADER_FUNC_WRITE
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		log.Println("RunUploadTest dial error:", err)
		return 0
	}
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
	header.Func = protocol.HEADER_FUNC_READ
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		return 0
	}
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
	header.Func = protocol.HEADER_FUNC_WRITE
	header.DataLen = blocksize
	buf := protocol.Header2Data(&header)

	c, err := connectServer(serverAddr, "", transferType)
	if err != nil {
		return 0
	}
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
