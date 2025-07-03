package main

import (
	"sync"
	"testing"
)

func TestBalancer_ChooseServer(t *testing.T) {
	b := &Balancer{
		mu: sync.RWMutex{},
		traffic: map[string]int64{
			"server1:8080": 1000,
			"server2:8080": 500,
			"server3:8080": 700,
		},
		healthy: map[string]bool{
			"server1:8080": true,
			"server2:8080": true,
			"server3:8080": true,
		},
	}

	selected := b.chooseServer()
	if selected != "server2:8080" {
		t.Errorf("Expected server2:8080, got %s", selected)
	}

	b.mu.Lock()
	b.healthy["server2:8080"] = false
	b.mu.Unlock()

	selected = b.chooseServer()
	if selected != "server3:8080" {
		t.Errorf("Expected server3:8080, got %s", selected)
	}

	b.mu.Lock()
	b.healthy["server1:8080"] = false
	b.healthy["server3:8080"] = false
	b.mu.Unlock()

	selected = b.chooseServer()
	if selected != "" {
		t.Errorf("Expected empty string when no healthy servers, got %s", selected)
	}
}

func TestBalancer_TrafficUpdate(t *testing.T) {
	b := &Balancer{
		mu:      sync.RWMutex{},
		traffic: map[string]int64{"server1:8080": 0},
		healthy: map[string]bool{"server1:8080": true},
	}

	b.mu.Lock()
	b.traffic["server1:8080"] += 150
	b.mu.Unlock()

	if got := b.traffic["server1:8080"]; got != 150 {
		t.Errorf("Expected traffic 150, got %d", got)
	}
}
