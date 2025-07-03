package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var (
	port         = flag.Int("port", 8090, "load balancer port")
	timeoutSec   = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https        = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     time.Duration
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

type Balancer struct {
	mu      sync.RWMutex
	traffic map[string]int64
	healthy map[string]bool
}

func NewBalancer() *Balancer {
	b := &Balancer{
		traffic: make(map[string]int64),
		healthy: make(map[string]bool),
	}

	go func() {
		for {
			for _, s := range serversPool {
				ok := health(s)
				b.mu.Lock()
				b.healthy[s] = ok
				b.mu.Unlock()
			}
			time.Sleep(5 * time.Second)
		}
	}()

	return b
}

func (b *Balancer) chooseServer() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var selected string
	var minBytes int64 = 1<<63 - 1

	for _, s := range serversPool {
		if !b.healthy[s] {
			continue
		}
		if b.traffic[s] < minBytes {
			minBytes = b.traffic[s]
			selected = s
		}
	}
	return selected
}

func (b *Balancer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	target := b.chooseServer()
	if target == "" {
		http.Error(rw, "No healthy servers", http.StatusServiceUnavailable)
		return
	}

	ctx, _ := context.WithTimeout(r.Context(), timeout)
	req := r.Clone(ctx)
	req.RequestURI = ""
	req.URL.Host = target
	req.URL.Scheme = scheme()
	req.Host = target

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Forward error to %s: %v", target, err)
		http.Error(rw, "Bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, values := range resp.Header {
		for _, v := range values {
			rw.Header().Add(k, v)
		}
	}
	if *traceEnabled {
		rw.Header().Set("lb-from", target)
	}

	rw.WriteHeader(resp.StatusCode)
	n, err := io.Copy(rw, resp.Body)
	if err != nil {
		log.Printf("copy error: %v", err)
	}

	b.mu.Lock()
	b.traffic[target] += n
	b.mu.Unlock()
}

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	return resp.StatusCode == http.StatusOK
}

func main() {
	flag.Parse()
	timeout = time.Duration(*timeoutSec) * time.Second

	balancer := NewBalancer()
	server := httptools.CreateServer(*port, balancer)

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)

	server.Start()
	signal.WaitForTerminationSignal()
}
