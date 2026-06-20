package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Stats holds all metrics
type Stats struct {
	TotalRequests      uint64
	SuccessfulRequests uint64
	FailedRequests     uint64
	TotalBytes         uint64
	TotalLatency       int64
	MinLatency         int64
	MaxLatency         int64
	StatusCodes        map[int]uint64

	mu sync.Mutex
}

func newStats() *Stats {
	return &Stats{
		MinLatency:  1<<63 - 1,
		MaxLatency:  0,
		StatusCodes: make(map[int]uint64),
	}
}

func (s *Stats) recordSuccess(latency time.Duration, bytes int, statusCode int) {
	atomic.AddUint64(&s.TotalRequests, 1)
	atomic.AddUint64(&s.SuccessfulRequests, 1)
	atomic.AddUint64(&s.TotalBytes, uint64(bytes))
	atomic.AddInt64(&s.TotalLatency, latency.Nanoseconds())

	s.mu.Lock()
	s.StatusCodes[statusCode]++
	if latency.Nanoseconds() < s.MinLatency {
		s.MinLatency = latency.Nanoseconds()
	}
	if latency.Nanoseconds() > s.MaxLatency {
		s.MaxLatency = latency.Nanoseconds()
	}
	s.mu.Unlock()
}

func (s *Stats) recordFailure() {
	atomic.AddUint64(&s.TotalRequests, 1)
	atomic.AddUint64(&s.FailedRequests, 1)

	s.mu.Lock()
	s.StatusCodes[0]++
	s.mu.Unlock()
}

func (s *Stats) avgLatency() time.Duration {
	total := atomic.LoadUint64(&s.TotalRequests)
	if total == 0 {
		return 0
	}
	totalLat := atomic.LoadInt64(&s.TotalLatency)
	return time.Duration(totalLat / int64(total))
}

func (s *Stats) print(title string, duration time.Duration) {
	total := atomic.LoadUint64(&s.TotalRequests)
	success := atomic.LoadUint64(&s.SuccessfulRequests)
	failed := atomic.LoadUint64(&s.FailedRequests)
	bytes := atomic.LoadUint64(&s.TotalBytes)

	minLat := atomic.LoadInt64(&s.MinLatency)
	if minLat == 1<<63-1 {
		minLat = 0
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("  📊 %s\n", title)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  ⏱️  Duration:           %v\n", duration.Round(time.Millisecond))
	fmt.Printf("  📨 Total Requests:     %d\n", total)
	if total > 0 {
		fmt.Printf("  ✅ Successful:         %d (%.2f%%)\n", success, float64(success)/float64(total)*100)
		fmt.Printf("  ❌ Failed:             %d (%.2f%%)\n", failed, float64(failed)/float64(total)*100)
	}
	fmt.Printf("  🚀 Requests/Second:    %.2f\n", float64(total)/duration.Seconds())
	fmt.Printf("  📦 Total Data:         %.2f KB\n", float64(bytes)/1024)
	fmt.Printf("  📥 Throughput:         %.2f KB/s\n", float64(bytes)/1024/duration.Seconds())
	fmt.Printf("  ⏳ Avg Latency:        %v\n", s.avgLatency())
	fmt.Printf("  ⚡ Min Latency:        %v\n", time.Duration(minLat))
	fmt.Printf("  🐢 Max Latency:        %v\n", time.Duration(atomic.LoadInt64(&s.MaxLatency)))
	fmt.Println("\n  📋 Status Code Distribution:")
	s.mu.Lock()
	for code, count := range s.StatusCodes {
		label := http.StatusText(code)
		if code == 0 {
			label = "Connection Error"
		}
		fmt.Printf("     %d (%s): %d\n", code, label, count)
	}
	s.mu.Unlock()
	fmt.Println(strings.Repeat("=", 60))
}

func stressTest(targetURL string, concurrency int, totalRequests int, timeout time.Duration) (*Stats, time.Duration) {
	stats := newStats()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			reqStart := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
			if err != nil {
				stats.recordFailure()
				return
			}

			resp, err := client.Do(req)
			latency := time.Since(reqStart)

			if err != nil {
				stats.recordFailure()
				return
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				stats.recordFailure()
				return
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				stats.recordSuccess(latency, len(body), resp.StatusCode)
			} else {
				stats.recordFailure()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	return stats, duration
}

func main() {
	var (
		target      = flag.String("target", "http://localhost:8080", "Target URL to stress test")
		concurrency = flag.Int("c", 100, "Number of concurrent connections")
		requests    = flag.Int("n", 10000, "Total number of requests")
		timeout     = flag.Duration("t", 10*time.Second, "Request timeout")
		compare     = flag.Bool("compare", false, "Run comparison: single server vs load balancer")
		singleURL   = flag.String("single", "http://localhost:9001", "Single server URL for comparison")
		lbURL       = flag.String("lb", "http://localhost:8080", "Load balancer URL for comparison")
	)
	flag.Parse()

	log.Println("🔥 Go Stress Test Tool")
	log.Printf("Config: %d concurrent, %d total requests, %v timeout", *concurrency, *requests, *timeout)

	if *compare {
		log.Println("🖥️  Testing SINGLE SERVER...")
		time.Sleep(2 * time.Second)
		stats1, dur1 := stressTest(*singleURL, *concurrency, *requests, *timeout)
		stats1.print("SINGLE SERVER RESULTS", dur1)

		log.Println("\n🔄 Testing LOAD BALANCER...")
		time.Sleep(2 * time.Second)
		stats2, dur2 := stressTest(*lbURL, *concurrency, *requests, *timeout)
		stats2.print("LOAD BALANCER RESULTS", dur2)

		log.Println("\n" + strings.Repeat("=", 60))
		log.Println("  📈 COMPARISON: Load Balancer vs Single Server")
		log.Println(strings.Repeat("=", 60))

		rps1 := float64(stats1.TotalRequests) / dur1.Seconds()
		rps2 := float64(stats2.TotalRequests) / dur2.Seconds()
		avg1 := stats1.avgLatency()
		avg2 := stats2.avgLatency()

		fmt.Printf("  %-20s | %-13s | %-13s | %-10s\n", "Metric", "Single Server", "Load Balancer", "Change")
		fmt.Printf("  %-20s-+-%13s-+-%13s-+-%10s\n", strings.Repeat("-", 20), strings.Repeat("-", 13), strings.Repeat("-", 13), strings.Repeat("-", 10))
		fmt.Printf("  %-20s | %13.2f | %13.2f | %+.2f%%\n", "Req/Second", rps1, rps2, (rps2-rps1)/rps1*100)
		fmt.Printf("  %-20s | %13v | %13v | %+.2f%%\n", "Avg Latency", avg1, avg2, (float64(avg1)-float64(avg2))/float64(avg1)*100)
		if stats1.TotalRequests > 0 && stats2.TotalRequests > 0 {
			fmt.Printf("  %-20s | %12.2f%% | %12.2f%% |\n", "Success Rate", float64(stats1.SuccessfulRequests)/float64(stats1.TotalRequests)*100, float64(stats2.SuccessfulRequests)/float64(stats2.TotalRequests)*100)
		}
		fmt.Printf("  %-20s | %13d | %13d |\n", "Total Failed", stats1.FailedRequests, stats2.FailedRequests)
		fmt.Println(strings.Repeat("=", 60))

		if rps2 > rps1 {
			log.Println("  ✅ Load Balancer is FASTER")
		} else if rps1 > rps2 {
			log.Println("  ⚠️  Single Server was faster (LB overhead or backends down?)")
		} else {
			log.Println("  ⚖️  Roughly equal performance")
		}
	} else {
		stats, duration := stressTest(*target, *concurrency, *requests, *timeout)
		stats.print("STRESS TEST RESULTS", duration)
	}
}
