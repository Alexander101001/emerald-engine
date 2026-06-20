package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CFWorkerKV struct {
	AccountID string
	Token     string
}

type CFWorkerAI struct {
	AccountID string
	Token     string
}

type CFTunnel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	OriginURL  string `json:"origin_url"`
	Subdomain  string `json:"subdomain"`
}

type CFAgent struct {
	mu          sync.Mutex
	Token       string
	AccountID   string
	Email       string
	KV          *CFWorkerKV
	AI          *CFWorkerAI
	Tunnels     []CFTunnel
	httpClient  *http.Client
	BaseURL     string
	initialized bool
}

var cfAgent *CFAgent

func initCFAgent() *CFAgent {
	token := vault["CLOUDFLARE_API_TOKEN"]
	if token == "" {
		fmt.Println("[CF_AGENT] No Cloudflare token, agent disabled")
		return nil
	}

	agent := &CFAgent{
		Token:      token,
		Email:      vault["CLOUDFLARE_EMAIL"],
		AccountID:  vault["CLOUDFLARE_ACCOUNT_ID"],
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    "https://api.cloudflare.com/client/v4",
	}

	if agent.AccountID != "" {
		agent.KV = &CFWorkerKV{
			AccountID: agent.AccountID,
			Token:     token,
		}
		agent.AI = &CFWorkerAI{
			AccountID: agent.AccountID,
			Token:     token,
		}
		agent.initialized = true
		fmt.Println("[CF_AGENT] Initialized with Workers KV + AI capabilities")
	} else {
		fmt.Println("[CF_AGENT] No Account ID, limited to DNS-only operations")
	}

	cfAgent = agent
	return agent
}

func (c *CFAgent) KVGet(namespace, key string) (string, error) {
	if c.KV == nil {
		return "", fmt.Errorf("KV not configured")
	}
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		c.BaseURL, c.AccountID, namespace, key)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (c *CFAgent) KVPut(namespace, key, value string) error {
	if c.KV == nil {
		return fmt.Errorf("KV not configured")
	}
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		c.BaseURL, c.AccountID, namespace, key)
	req, _ := http.NewRequest("PUT", url, bytes.NewBufferString(value))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *CFAgent) AIRun(model, prompt string) (string, error) {
	if c.AI == nil {
		return "", fmt.Errorf("Workers AI not configured")
	}
	url := fmt.Sprintf("%s/accounts/%s/ai/run/%s",
		c.BaseURL, c.AccountID, model)
	payload := map[string]string{"prompt": prompt}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}
	if data, ok := result["result"]; ok {
		if d, ok := data.(map[string]interface{}); ok {
			if response, ok := d["response"].(string); ok {
				return response, nil
			}
		}
	}
	return string(respBody), nil
}

func (c *CFAgent) CreateTunnel(name, originURL, subdomain string) (*CFTunnel, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("Account ID required for tunnels")
	}
	url := fmt.Sprintf("%s/accounts/%s/cfd_tunnel", c.BaseURL, c.AccountID)
	payload := map[string]string{
		"name":       name,
		"origin_url": originURL,
		"subdomain":  subdomain,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Result CFTunnel `json:"result"`
		Success bool    `json:"success"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("tunnel create: %s", string(respBody))
	}
	if !result.Success {
		return nil, fmt.Errorf("tunnel create failed: %s", string(respBody))
	}

	c.mu.Lock()
	c.Tunnels = append(c.Tunnels, result.Result)
	c.mu.Unlock()

	return &result.Result, nil
}

func (c *CFAgent) FetchRadarTrends(query string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/radar/trends?query=%s", c.BaseURL, strings.ReplaceAll(query, " ", "%20"))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result []map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("radar trends: %s", string(body))
	}
	return result.Result, nil
}

func (c *CFAgent) Stats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return map[string]interface{}{
		"initialized": c.initialized,
		"has_kv":      c.KV != nil,
		"has_ai":      c.AI != nil,
		"tunnels":     len(c.Tunnels),
		"account_id":  c.AccountID != "",
	}
}
