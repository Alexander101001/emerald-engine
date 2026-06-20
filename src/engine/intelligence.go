package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

type IntelligenceLayer struct {
	mu          sync.Mutex
	ollamaHost  string
	model       string
	ready       bool
	pullAttempt bool
}

var intelligence *IntelligenceLayer

func initIntelligence() *IntelligenceLayer {
	il := &IntelligenceLayer{
		ollamaHost: "http://127.0.0.1:11434",
		model:      "qwen2.5:0.5b",
	}
	intelligence = il
	go il.bootstrap()
	return il
}

func (il *IntelligenceLayer) bootstrap() {
	fmt.Printf("[intelligence] Booting lightweight Qwen via init_intelligence.sh\n")
	cmd := exec.Command("bash", "init_intelligence.sh")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Printf("[intelligence] bootstrap script error: %v\n%s\n", err, out.String())
		return
	}
	fmt.Printf("[intelligence] %s\n", strings.TrimSpace(out.String()))
	il.mu.Lock()
	il.ready = true
	il.mu.Unlock()
	fmt.Printf("[intelligence] Qwen2.5:0.5b ready at %s\n", il.ollamaHost)
}

// Lightweight inference — returns generated text from local Ollama
func (il *IntelligenceLayer) Generate(prompt string, maxTokens int) (string, error) {
	il.mu.Lock()
	ready := il.ready
	il.mu.Unlock()
	if !ready {
		return "", fmt.Errorf("intelligence layer not ready yet")
	}

	body := map[string]interface{}{
		"model":    il.model,
		"prompt":   prompt,
		"stream":   false,
		"options":  map[string]interface{}{
			"num_predict": maxTokens,
			"temperature": 0.7,
		},
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(il.ollamaHost+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return "", fmt.Errorf("ollama decode: %s", string(raw))
	}
	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}
	return strings.TrimSpace(result.Response), nil
}

func (il *IntelligenceLayer) IsReady() bool {
	il.mu.Lock()
	defer il.mu.Unlock()
	return il.ready
}

func (il *IntelligenceLayer) Stats() map[string]interface{} {
	il.mu.Lock()
	defer il.mu.Unlock()
	return map[string]interface{}{
		"ready":  il.ready,
		"model":  il.model,
		"host":   il.ollamaHost,
	}
}
