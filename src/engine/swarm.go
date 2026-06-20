package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MemoryEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
	Event     string    `json:"event"`
	Task      string    `json:"task,omitempty"`
	Approved  bool      `json:"approved,omitempty"`
}

type SwarmOrchestrator struct {
	mu           sync.Mutex
	memoryStream []MemoryEvent
	streamPath   string
	discussions  int
	approved     int
	rejected     int
}

var swarmOrchestrator *SwarmOrchestrator

func initSwarmOrchestrator() *SwarmOrchestrator {
	so := &SwarmOrchestrator{
		streamPath: "config/swarm_memory_stream.json",
	}
	so.load()
	swarmOrchestrator = so
	return so
}

func (so *SwarmOrchestrator) load() {
	data, err := os.ReadFile(so.streamPath)
	if err != nil {
		return
	}
	var events []MemoryEvent
	if json.Unmarshal(data, &events) == nil {
		so.memoryStream = events
	}
}

func (so *SwarmOrchestrator) save() {
	os.MkdirAll(filepath.Dir(so.streamPath), 0755)
	data, _ := json.MarshalIndent(so.memoryStream, "", "  ")
	os.WriteFile(so.streamPath, data, 0644)
}

func (so *SwarmOrchestrator) LogEvent(agent, event, task string) {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.memoryStream = append(so.memoryStream, MemoryEvent{
		Timestamp: time.Now(),
		Agent:     agent,
		Event:     event,
		Task:      task,
	})
	if len(so.memoryStream) > 500 {
		so.memoryStream = so.memoryStream[len(so.memoryStream)-500:]
	}
	so.save()
}

func (so *SwarmOrchestrator) MultiAgentDiscussion(task, taskType string, params map[string]interface{}) bool {
	so.mu.Lock()
	so.discussions++
	so.mu.Unlock()

	agents := []string{"Coder", "Strategist", "Monetization", "SecurityOfficer"}
	results := make(map[string]bool)

	for _, agentName := range agents {
		approved := so.evaluateTaskFromRole(agentName, task, taskType, params)
		results[agentName] = approved
		event := fmt.Sprintf("Evaluated task: %s", task)
		if !approved {
			event = fmt.Sprintf("REJECTED task: %s", task)
		}
		so.LogEvent(agentName, event, task)

		// Log into agent coordinator memory for cross-agent awareness
		if agentCoordinator != nil {
			agentCoordinator.Broadcast(fmt.Sprintf("[%s] %s → %v", agentName, task, approved))
		}

		if !approved {
			fmt.Printf("[SWARM] %s vetoed task: %s\n", agentName, task)
		}
	}

	// All agents must approve (strict consensus)
	allApproved := true
	for _, approved := range results {
		if !approved {
			allApproved = false
			break
		}
	}

	so.mu.Lock()
	if allApproved {
		so.approved++
	} else {
		so.rejected++
	}
	so.mu.Unlock()

	if allApproved {
		so.LogEvent("SwarmOrchestrator", fmt.Sprintf("All agents approved: %s", task), task)
	} else {
		so.LogEvent("SwarmOrchestrator", fmt.Sprintf("Swarm REJECTED: %s", task), task)
	}

	return allApproved
}

func (so *SwarmOrchestrator) evaluateTaskFromRole(agent, task, taskType string, params map[string]interface{}) bool {
	switch agent {
	case "Coder":
		// Check code quality: verify required params exist, code isn't empty
		if taskType == "write_code" || taskType == "deploy" {
			if code, ok := params["code"].(string); ok && len(code) < 10 {
				return false
			}
		}
		return true

	case "Strategist":
		// ROI check: reject if no revenue potential or too many children already
		if taskType == "deploy" {
			if orchestrator != nil {
				orchestrator.mu.Lock()
				count := len(orchestrator.Children)
				orchestrator.mu.Unlock()
				if count >= 10 {
					return false
				}
			}
		}
		return true

	case "Monetization":
		// Monetization viability: check if adsense is configured
		if taskType == "deploy" || taskType == "publish" {
			if vault != nil {
				adSense := vaultGet("ADSENSE_CLIENT_ID", "")
				if adSense == "" {
					return false
				}
			}
		}
		return true

	case "SecurityOfficer":
		// Security audit: check for secret leaks in params
		if code, ok := params["code"].(string); ok {
			secrets := []string{"ghp_", "hf_", "sk-", "api_key", "secret", "password", "token", "bearer", "auth"}
			codeLower := strings.ToLower(code)
			for _, s := range secrets {
				if strings.Contains(codeLower, s) && strings.Contains(codeLower, "=") {
					return false
				}
			}
		}
		// CRITIC external verification for deploy tasks
		if taskType == "deploy" && reflexionLayer != nil {
			if url, ok := params["url"].(string); ok && url != "" {
				result := reflexionLayer.RunCriticExternalVerification(url)
				if verified, ok := result["verified"].(bool); ok && !verified {
					return false
				}
			}
		}
		return true
	}
	return true
}

func (so *SwarmOrchestrator) GetMemoryStream() []MemoryEvent {
	so.mu.Lock()
	defer so.mu.Unlock()
	out := make([]MemoryEvent, len(so.memoryStream))
	copy(out, so.memoryStream)
	return out
}

func (so *SwarmOrchestrator) GetRecentEvents(n int) []MemoryEvent {
	so.mu.Lock()
	defer so.mu.Unlock()
	if len(so.memoryStream) <= n {
		out := make([]MemoryEvent, len(so.memoryStream))
		copy(out, so.memoryStream)
		return out
	}
	out := make([]MemoryEvent, n)
	copy(out, so.memoryStream[len(so.memoryStream)-n:])
	return out
}

func (so *SwarmOrchestrator) Stats() map[string]interface{} {
	so.mu.Lock()
	defer so.mu.Unlock()

	return map[string]interface{}{
		"memory_events":  len(so.memoryStream),
		"discussions":    so.discussions,
		"approved":       so.approved,
		"rejected":       so.rejected,
		"agents":         []string{"Coder", "Strategist", "Monetization", "SecurityOfficer"},
	}
}
