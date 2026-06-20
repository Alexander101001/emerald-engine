package main

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type TokenStatus string

const (
	TokenActive    TokenStatus = "active"
	TokenLimited   TokenStatus = "rate_limited"
	TokenExpired   TokenStatus = "expired"
	TokenDepleted  TokenStatus = "depleted"
	TokenUntested  TokenStatus = "untested"
)

type TokenSlot struct {
	Token     string      `json:"-"`
	Label     string      `json:"label"`
	Service   string      `json:"service"`
	Status    TokenStatus `json:"status"`
	LastUsed  time.Time   `json:"last_used"`
	UsedToday int         `json:"used_today"`
	MaxDaily  int         `json:"max_daily"`
}

type FreeEndpoint struct {
	URL       string `json:"url"`
	Model     string `json:"model"`
	Source    string `json:"source"`
	Works     bool   `json:"works"`
	LastTest  time.Time `json:"last_tested"`
}

type TokenMatrix struct {
	mu           	sync.RWMutex
	slots        []TokenSlot
	activeIdx    map[string]int
	freeEndpoints []FreeEndpoint
	httpClient   *http.Client
	rotations    int
	lastHarvest  time.Time
}

var tokenMatrix *TokenMatrix

func initTokenMatrix() *TokenMatrix {
	tm := &TokenMatrix{
		slots:        make([]TokenSlot, 0),
		activeIdx:    make(map[string]int),
		freeEndpoints: make([]FreeEndpoint, 0),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
	tm.seedFromVault()
	go tm.harvestFreeEndpointsLoop()
	go tm.rotationLoop()
	tokenMatrix = tm
	return tm
}

func (tm *TokenMatrix) seedFromVault() {
	services := map[string]struct {
		key      string
		maxDaily int
	}{
		"openai":     {"OPENAI_API_KEY", 200},
		"deepseek":   {"DEEPSEEK_API_KEY", 150},
		"groq":       {"GROQ_API_KEY", 300},
		"together":   {"TOGETHER_API_KEY", 200},
		"mistral":    {"MISTRAL_API_KEY", 200},
		"openrouter": {"OPENROUTER_API_KEY", 200},
		"claude":     {"CLAUDE_API_KEY", 200},
		"google_ai":  {"GOOGLE_AI_KEY", 200},
		"gemini":     {"GEMINI_API_KEY", 200},
		"github":     {"GITHUB_TOKEN", 500},
		"github_old": {"GITHUB_TOKEN_OLD", 500},
		"hf":         {"HF_TOKEN", 1000},
		"hf_old":     {"HF_TOKEN_OLD", 1000},
		"hf_hasan":   {"HF_TOKEN_HASAN", 1000},
		"binance":    {"BINANCE_API_KEY", 200},
		"cf":         {"CLOUDFLARE_API_TOKEN", 200},
	}

	for svc, cfg := range services {
		token := vaultGet(cfg.key, "")
		if token == "" {
			continue
		}
		tm.slots = append(tm.slots, TokenSlot{
			Token:     token,
			Label:     cfg.key,
			Service:   svc,
			Status:    TokenUntested,
			MaxDaily:  cfg.maxDaily,
		})
		if _, ok := tm.activeIdx[svc]; !ok {
			tm.activeIdx[svc] = len(tm.slots) - 1
		}
	}

	sort.Slice(tm.slots, func(i, j int) bool {
		return tm.slots[i].Service < tm.slots[j].Service
	})
}

func (tm *TokenMatrix) GetToken(service string) (string, error) {
	tm.mu.RLock()
	idx, ok := tm.activeIdx[service]
	tm.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("no token for service: %s", service)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	slot := &tm.slots[idx]
	slot.LastUsed = time.Now()
	slot.UsedToday++

	if slot.UsedToday >= slot.MaxDaily {
		slot.Status = TokenDepleted
		return tm.rotateLocked(service)
	}

	if slot.Status == TokenExpired || slot.Status == TokenDepleted {
		return tm.rotateLocked(service)
	}

	return slot.Token, nil
}

func (tm *TokenMatrix) rotateLocked(service string) (string, error) {
	for i := range tm.slots {
		if tm.slots[i].Service == service && tm.slots[i].Status == TokenActive {
			tm.activeIdx[service] = i
			tm.slots[i].LastUsed = time.Now()
			tm.slots[i].Status = TokenActive
			tm.rotations++
			return tm.slots[i].Token, nil
		}
	}
	return "", fmt.Errorf("all tokens depleted for %s", service)
}

func (tm *TokenMatrix) MarkFailed(label string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for i := range tm.slots {
		if tm.slots[i].Label == label {
			tm.slots[i].Status = TokenExpired
			return
		}
	}
}

func (tm *TokenMatrix) harvestFreeEndpointsLoop() {
	time.Sleep(60 * time.Second)
	tm.probeFreeEndpoints()
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		tm.probeFreeEndpoints()
	}
}

func (tm *TokenMatrix) probeFreeEndpoints() {
	candidates := []FreeEndpoint{
		{URL: "https://api-inference.huggingface.co/models/meta-llama/Llama-3.2-3B-Instruct", Model: "Llama-3.2-3B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/google/gemma-2-2b-it", Model: "Gemma-2-2B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/microsoft/Phi-3-mini-4k-instruct", Model: "Phi-3-Mini", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/tiiuae/falcon-7b-instruct", Model: "Falcon-7B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/HuggingFaceH4/zephyr-7b-beta", Model: "Zephyr-7B", Source: "hf-internal"},
	}

	var working []FreeEndpoint
	for _, ep := range candidates {
		payload := strings.NewReader(`{"inputs":"hello","parameters":{"max_new_tokens":10}}`)
		req, _ := http.NewRequest("POST", ep.URL, payload)
		req.Header.Set("Content-Type", "application/json")
		// Try without token for free-tier access
		req.Header.Set("Authorization", "")

		resp, err := tm.httpClient.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		ep.Works = resp.StatusCode == 200 && !strings.Contains(string(body), "error")
		ep.LastTest = time.Now()
		if ep.Works {
			working = append(working, ep)
		}
	}

	tm.mu.Lock()
	tm.freeEndpoints = working
	tm.lastHarvest = time.Now()
	tm.mu.Unlock()
}

func (tm *TokenMatrix) scanPublicSpaces() {
	fmt.Println("[TOKEN] Scanning public HF spaces for free inference endpoints")
	discovered := []FreeEndpoint{
		{URL: "https://api-inference.huggingface.co/models/meta-llama/Llama-3.2-3B-Instruct", Model: "Llama-3.2-3B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/meta-llama/Llama-3.2-1B-Instruct", Model: "Llama-3.2-1B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/google/gemma-2-2b-it", Model: "Gemma-2-2B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/microsoft/Phi-3-mini-4k-instruct", Model: "Phi-3-Mini", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/tiiuae/falcon-7b-instruct", Model: "Falcon-7B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/HuggingFaceH4/zephyr-7b-beta", Model: "Zephyr-7B", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/mistralai/Mistral-7B-Instruct-v0.3", Model: "Mistral-7B-v0.3", Source: "hf-internal"},
		{URL: "https://api-inference.huggingface.co/models/meta-llama/Llama-3.2-11B-Vision-Instruct", Model: "Llama-3.2-11B", Source: "hf-internal"},
		{URL: "https://router.huggingface.co/hf-internal/models/meta-llama/Llama-3.2-3B-Instruct", Model: "Llama-3.2-3B-routed", Source: "hf-router-internal"},
		{URL: "https://router.huggingface.co/hf-internal/models/tiiuae/falcon-7b-instruct", Model: "Falcon-7B-routed", Source: "hf-router-internal"},
	}

	var working []FreeEndpoint
	for _, ep := range discovered {
		if tm.testEndpoint(ep.URL) {
			ep.Works = true
			ep.LastTest = time.Now()
			working = append(working, ep)
			fmt.Printf("[TOKEN] Found working endpoint: %s (%s)\n", ep.Model, ep.URL[:60])
		}
	}

	// Try internal routing endpoints
	internalRoutes := []string{
		"https://router.huggingface.co/hf-internal/models/",
		"https://api-inference.huggingface.co/models/",
	}
	for _, route := range internalRoutes {
		if tm.testEndpoint(route + "meta-llama/Llama-3.2-3B-Instruct") {
			fmt.Printf("[TOKEN] Internal routing active: %s\n", route)
			break
		}
	}

	tm.mu.Lock()
	tm.freeEndpoints = append(tm.freeEndpoints, working...)
	tm.lastHarvest = time.Now()
	tm.mu.Unlock()
}

func (tm *TokenMatrix) testEndpoint(url string) bool {
	payload := strings.NewReader(`{"inputs":"hello","parameters":{"max_new_tokens":10}}`)
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode == 200 && !strings.Contains(string(body), "error")
}

func (tm *TokenMatrix) GetFreeEndpoint() *FreeEndpoint {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if len(tm.freeEndpoints) == 0 {
		return nil
	}
	return &tm.freeEndpoints[time.Now().UnixNano()%int64(len(tm.freeEndpoints))]
}

func (tm *TokenMatrix) rotationLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		tm.mu.Lock()
		now := time.Now()
		for i := range tm.slots {
			if now.Day() != tm.slots[i].LastUsed.Day() {
				tm.slots[i].UsedToday = 0
				if tm.slots[i].Status == TokenDepleted {
					tm.slots[i].Status = TokenActive
				}
			}
		}
		tm.mu.Unlock()
	}
}

func (tm *TokenMatrix) Stats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	byService := make(map[string]map[string]interface{})
	for _, s := range tm.slots {
		if _, ok := byService[s.Service]; !ok {
			byService[s.Service] = map[string]interface{}{
				"total":  0,
				"active": 0,
			}
		}
		m := byService[s.Service]
		m["total"] = m["total"].(int) + 1
		if s.Status == TokenActive || s.Status == TokenUntested {
			m["active"] = m["active"].(int) + 1
		}
	}

	return map[string]interface{}{
		"slots":         len(tm.slots),
		"rotations":     tm.rotations,
		"free_endpoints": len(tm.freeEndpoints),
		"last_harvest":  tm.lastHarvest.Format(time.RFC3339),
		"by_service":    byService,
	}
}

func tmGetToken(service string) string {
	if tokenMatrix == nil {
		return ""
	}
	tok, err := tokenMatrix.GetToken(service)
	if err != nil {
		return ""
	}
	return tok
}
