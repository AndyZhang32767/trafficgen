package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Stats 保存累计流量与按月流量，会持久化到磁盘，重启后自动续上。
type Stats struct {
	mu      sync.Mutex
	Total   uint64            `json:"total"`   // 累计发送字节
	Monthly map[string]uint64 `json:"monthly"` // "2006-01" -> 字节
}

var (
	stats        = &Stats{Monthly: make(map[string]uint64)}
	sentInWindow uint64 // 原子计数：本采样窗口内发送的字节
	currentSpeed uint64 // 原子计数：最近一秒的字节/秒
	statsPath    string
)

func currentMonth() string { return time.Now().Format("2006-01") }

func (s *Stats) add(n uint64) {
	s.mu.Lock()
	s.Total += n
	s.Monthly[currentMonth()] += n
	s.mu.Unlock()
	atomic.AddUint64(&sentInWindow, n)
}

func (s *Stats) load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // 首次运行没有文件，忽略
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := json.Unmarshal(data, s); err != nil {
		log.Printf("统计文件解析失败，从零开始: %v", err)
		return
	}
	if s.Monthly == nil {
		s.Monthly = make(map[string]uint64)
	}
}

func (s *Stats) save(path string) {
	s.mu.Lock()
	data, _ := json.MarshalIndent(s, "", "  ")
	s.mu.Unlock()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Printf("写入统计文件失败: %v", err)
		return
	}
	os.Rename(tmp, path)
}

func (s *Stats) snapshot() (total, monthly uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Total, s.Monthly[currentMonth()]
}

// streamHandler 持续向客户端发送数据，产生服务端上行流量。
func streamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	buf := make([]byte, 256*1024) // 全零缓冲区即可，客户端会直接丢弃
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return // 客户端断开
		default:
		}
		n, err := w.Write(buf)
		if n > 0 {
			stats.add(uint64(n))
		}
		if err != nil {
			return
		}
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	total, monthly := stats.snapshot()
	speed := atomic.LoadUint64(&currentSpeed)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"speed":   speed, // 字节/秒
		"monthly": monthly,
		"total":   total,
		"month":   currentMonth(),
	})
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

func main() {
	addr := flag.String("addr", ":8080", "监听地址，例如 :8080")
	flag.StringVar(&statsPath, "stats", "stats.json", "统计数据持久化文件路径")
	flag.Parse()

	stats.load(statsPath)

	// 采样当前速率：每秒把窗口计数搬到 currentSpeed
	go func() {
		for range time.Tick(time.Second) {
			atomic.StoreUint64(&currentSpeed, atomic.SwapUint64(&sentInWindow, 0))
		}
	}()

	// 定期持久化，保证累计/当月流量不因重启丢失
	go func() {
		for range time.Tick(5 * time.Second) {
			stats.save(statsPath)
		}
	}()

	// 收到退出信号时保存后再退出
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		stats.save(statsPath)
		log.Println("已保存统计，退出")
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", dashboardHandler)
	mux.HandleFunc("/stats", statsHandler)
	mux.HandleFunc("/stream", streamHandler)

	srv := &http.Server{Addr: *addr, Handler: mux}
	log.Printf("流量服务端启动，监听 %s，面板 http://<你的IP>%s/ ，刷流量入口 /stream", *addr, *addr)
	log.Fatal(srv.ListenAndServe())
}
