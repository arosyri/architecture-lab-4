package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var port = flag.Int("port", 8080, "server port")

const (
	confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
	confHealthFailure    = "CONF_HEALTH_FAILURE"
)

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if os.Getenv(confHealthFailure) == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	mux.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {

		respDelayString := os.Getenv(confResponseDelaySec)
		if delaySec, err := strconv.Atoi(respDelayString); err == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		rw.Header().Set("X-Server-Name", hostname)
		rw.Header().Set("content-type", "application/json")

		var payload []string
		switch {
		case strings.Contains(hostname, "server1"):
			payload = []string{strings.Repeat("A", 100)} // ~100 bytes
		case strings.Contains(hostname, "server2"):
			payload = []string{strings.Repeat("B", 200)} // ~200 bytes
		case strings.Contains(hostname, "server3"):
			payload = []string{strings.Repeat("C", 300)} // ~300 bytes
		default:
			payload = []string{strings.Repeat("Z", 150)} // fallback
		}

		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(payload)
	})

	mux.HandleFunc("/report", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("report endpoint"))
	})

	server := httptools.CreateServer(*port, mux)
	server.Start()
	signal.WaitForTerminationSignal()
}
