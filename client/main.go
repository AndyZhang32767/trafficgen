package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// 客户端：持续从服务端 /stream 拉取数据并直接丢弃，
// 断线后自动重连，从而让服务端持续产生上行流量。

var received uint64 // 原子计数：本窗口收到的字节

// discardCounter 边丢弃数据边计数
type discardCounter struct{}

func (discardCounter) Write(p []byte) (int, error) {
	atomic.AddUint64(&received, uint64(len(p)))
	return len(p), nil
}

func loop(id int, url string) {
	buf := make([]byte, 256*1024)
	client := &http.Client{Timeout: 0} // 不设超时，长连接持续下载
	for {
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[连接%d] 连接失败: %v，2秒后重连", id, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("[连接%d] 状态码 %d，2秒后重连", id, resp.StatusCode)
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		_, err = io.CopyBuffer(discardCounter{}, resp.Body, buf)
		resp.Body.Close()
		log.Printf("[连接%d] 断开: %v，1秒后重连", id, err)
		time.Sleep(time.Second)
	}
}

func fmtSpeed(bps uint64) string {
	bits := float64(bps) * 8
	units := []string{"bps", "Kbps", "Mbps", "Gbps", "Tbps"}
	i := 0
	for bits >= 1000 && i < len(units)-1 {
		bits /= 1000
		i++
	}
	return fmt.Sprintf("%.2f %s", bits, units[i])
}

func main() {
	url := flag.String("url", "", "服务端流地址，例如 http://1.2.3.4:8080/stream")
	conns := flag.Int("c", 4, "并发连接数（越多刷得越快，注意别打满自己带宽）")
	flag.Parse()

	if *url == "" {
		log.Fatal("必须指定 -url，例如 -url http://1.2.3.4:8080/stream")
	}

	for i := 0; i < *conns; i++ {
		go loop(i+1, *url)
	}

	// 每秒打印一次接收速率
	for range time.Tick(time.Second) {
		n := atomic.SwapUint64(&received, 0)
		log.Printf("下载中(全部丢弃) 速率: %s", fmtSpeed(n))
	}
}
