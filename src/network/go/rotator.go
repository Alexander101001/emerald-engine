package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProxyEntry struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Latency int    `json:"latency_ms"`
}

type Rotator struct {
	proxies []ProxyEntry
	client  *http.Client
}

func NewRotator() *Rotator {
	r := &Rotator{
		client: &http.Client{Timeout: 10 * time.Second},
	}
	r.loadProxies()
	return r
}

func (r *Rotator) loadProxies() {
	data, err := os.ReadFile("proxies.json")
	if err != nil {
		r.proxies = r.defaultProxies()
		return
	}
	json.Unmarshal(data, &r.proxies)
}

func (r *Rotator) defaultProxies() []ProxyEntry {
	return []ProxyEntry{
		{Address: "http://proxy1:8080", Type: "datacenter", Latency: 50},
		{Address: "http://proxy2:8080", Type: "residential", Latency: 120},
		{Address: "http://proxy3:8080", Type: "mobile", Latency: 200},
	}
}

func (r *Rotator) Next() ProxyEntry {
	return r.proxies[rand.Intn(len(r.proxies))]
}

func (r *Rotator) Fetch(url string) (string, error) {
	proxy := r.Next()
	fmt.Fprintf(os.Stderr, "go: routing %s via %s (%s)\n", url, proxy.Address, proxy.Type)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "EmeraldEngine/1.0")
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func main() {
	rotator := NewRotator()
	input, _ := io.ReadAll(os.Stdin)
	urls := strings.Fields(string(input))
	results := make(map[string]string)
	for _, url := range urls {
		body, err := rotator.Fetch(url)
		if err != nil {
			results[url] = fmt.Sprintf("error: %s", err)
		} else {
			results[url] = fmt.Sprintf("%d bytes", len(body))
		}
	}
	out, _ := json.Marshal(results)
	fmt.Println(string(out))
}
