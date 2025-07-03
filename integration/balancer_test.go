package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	for i := 0; i < 5; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
			continue
		}
		resp.Body.Close()

		lbFrom := resp.Header.Get("lb-from")
		if lbFrom == "" {
			t.Errorf("Request %d: missing lb-from header", i+1)
		} else {
			t.Logf("Request %d served by: %s", i+1, lbFrom)
		}
	}
}

func BenchmarkBalancer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			b.Errorf("Request %d failed: %v", i+1, err)
			continue
		}
		resp.Body.Close()
	}
}
