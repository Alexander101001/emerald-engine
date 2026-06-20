package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type OPROMetaOptimizer struct {
	mu                sync.Mutex
	evolutions        []EvolutionRecord
	promptVersion     int
	masterPromptPath  string
	telemetryLogPath  string
	successThreshold  float64
	totalOps          int
	successfulOps     int
	metaLoops         int
	path              string
}

type EvolutionRecord struct {
	Version     int       `json:"version"`
	Time        time.Time `json:"time"`
	SuccessRate float64   `json:"success_rate"`
	Failures    []string  `json:"failures"`
	PromptDiff  string    `json:"prompt_diff"`
}

type TelemetryLog struct {
	SuccessRate float64   `json:"success_rate"`
	Failures    []string  `json:"failures"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var oproOptimizer *OPROMetaOptimizer

func initOPRO() *OPROMetaOptimizer {
	o := &OPROMetaOptimizer{
		promptVersion:     1,
		masterPromptPath:  "config/master_prompt.txt",
		telemetryLogPath:  "logs/telemetry.json",
		successThreshold:  1.0,
		path:             "emerald_opro.json",
	}
	o.load()
	go o.metaLoop()
	oproOptimizer = o
	return o
}

func (o *OPROMetaOptimizer) loadCurrentMasterPrompt() string {
	data, err := os.ReadFile(o.masterPromptPath)
	if err != nil {
		return "DEFAULT_MASTER_PROMPT_REPRESENTATION"
	}
	return string(data)
}

func (o *OPROMetaOptimizer) loadTelemetryLog() TelemetryLog {
	data, err := os.ReadFile(o.telemetryLogPath)
	if err != nil {
		return TelemetryLog{SuccessRate: 100, Failures: []string{}}
	}
	var tl TelemetryLog
	if json.Unmarshal(data, &tl) == nil {
		return tl
	}
	return TelemetryLog{SuccessRate: 100, Failures: []string{}}
}

func (o *OPROMetaOptimizer) saveOptimizedPrompt(prompt string) {
	os.MkdirAll(filepath.Dir(o.masterPromptPath), 0755)
	os.WriteFile(o.masterPromptPath, []byte(prompt), 0644)
}

func (o *OPROMetaOptimizer) writeTelemetryLog(tl TelemetryLog) {
	os.MkdirAll(filepath.Dir(o.telemetryLogPath), 0755)
	data, _ := json.MarshalIndent(tl, "", "  ")
	os.WriteFile(o.telemetryLogPath, data, 0644)
}

func (o *OPROMetaOptimizer) metaLoop() {
	time.Sleep(10 * time.Minute)
	o.executeMetaOptimizationLoop()
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		o.executeMetaOptimizationLoop()
	}
}

func (o *OPROMetaOptimizer) executeMetaOptimizationLoop() {
	o.metaLoops++
	currentPrompt := o.loadCurrentMasterPrompt()
	telemetryData := o.collectTelemetry()

	o.writeTelemetryLog(TelemetryLog{
		SuccessRate: telemetryData.SuccessRate,
		Failures:    telemetryData.Failures,
		UpdatedAt:   time.Now(),
	})

	fmt.Printf("[OPRO] Meta-loop %d | success: %.1f%% | failures: %d\n",
		o.metaLoops, telemetryData.SuccessRate, len(telemetryData.Failures))

	if telemetryData.SuccessRate == 100 && len(telemetryData.Failures) == 0 {
		fmt.Println("[OPRO] System performing optimally. No prompt mutation required.")
		return
	}

	optimizedPrompt := o.llmOptimizePrompt(currentPrompt, telemetryData)
	if optimizedPrompt == "" {
		fmt.Println("[OPRO] LLM optimization failed, keeping current prompt")
		return
	}

	o.saveOptimizedPrompt(optimizedPrompt)

	o.mu.Lock()
	o.promptVersion++
	o.evolutions = append(o.evolutions, EvolutionRecord{
		Version:     o.promptVersion,
		Time:        time.Now(),
		SuccessRate: telemetryData.SuccessRate,
		Failures:    telemetryData.Failures,
		PromptDiff:  optimizedPrompt[:minInt(len(optimizedPrompt), 500)],
	})
	if len(o.evolutions) > 20 {
		o.evolutions = o.evolutions[len(o.evolutions)-20:]
	}
	o.mu.Unlock()

	o.applyBehavioralMutations(telemetryData.Failures)
	o.save()

	fmt.Printf("[OPRO] Master prompt mutated and optimized successfully (v%d)\n", o.promptVersion)
}

func (o *OPROMetaOptimizer) collectTelemetry() TelemetryLog {
	failures := []string{}
	totalChecks := 0
	failedChecks := 0

	if cognitive != nil {
		totalChecks++
		if cognitive.Revenue == 0 && cognitive.CycleNum > 10 {
			failures = append(failures, "zero_revenue_after_10_cycles")
			failedChecks++
		}
	}

	if tokenMatrix != nil {
		totalChecks++
		stats := tokenMatrix.Stats()
		freeEndpoints := toInt(stats["free_endpoints"])
		if freeEndpoints == 0 {
			failures = append(failures, "no_free_inference_endpoints")
			failedChecks++
		}
	}

	if orchestrator != nil {
		totalChecks++
		orchestrator.mu.Lock()
		crashed := 0
		for _, c := range orchestrator.Children {
			if c.Status == "crashed" || c.Status == "error" {
				crashed++
			}
		}
		orchestrator.mu.Unlock()
		if crashed > 2 {
			failures = append(failures, fmt.Sprintf("high_child_failure_rate_%d_crashed", crashed))
			failedChecks++
		}
	}

	if heartbeatDaemon != nil {
		totalChecks++
		stats := heartbeatDaemon.Stats()
		hfails := toInt(stats["failures"])
		clicks := toInt(stats["sub_clicks"])
		if hfails > 10 && clicks == 0 {
			failures = append(failures, "heartbeat_no_subclicks_high_failures")
			failedChecks++
		}
	}

	if agentCoordinator != nil {
		totalChecks++
		stats := agentCoordinator.Stats()
		if agents, ok := stats["agents"].(map[string]interface{}); ok {
			for name, a := range agents {
				if m, ok := a.(map[string]interface{}); ok {
					if logs, ok := m["logs"].(float64); ok && logs > 5 {
						failures = append(failures, fmt.Sprintf("agent_high_logs_%s_%.0f", name, logs))
						failedChecks++
					}
				}
			}
		}
	}

	successRate := 100.0
	if totalChecks > 0 {
		successRate = float64(totalChecks-failedChecks) / float64(totalChecks) * 100
	}

	return TelemetryLog{
		SuccessRate: successRate,
		Failures:    failures,
		UpdatedAt:   time.Now(),
	}
}

func (o *OPROMetaOptimizer) llmOptimizePrompt(currentPrompt string, telemetry TelemetryLog) string {
	if len(llmProviders) == 0 {
		return ""
	}

	failuresJSON, _ := json.Marshal(telemetry)

	metaInstruction := "You are Google DeepMind OPRO Engine. Analyze the following system prompt and recent system failure logs. Your task is to rewrite and optimize the system prompt to prevent these explicit failures and maximize operational efficiency. Output ONLY the newly optimized system prompt without any extra conversational text."

	userContent := fmt.Sprintf("Current Prompt:\n%s\n\nFailure Logs:\n%s", currentPrompt, string(failuresJSON))

	prov := llmProviders[0]
	for _, p := range llmProviders {
		if p.Name == "OpenRouter" {
			prov = p
			break
		}
	}

	payload := map[string]interface{}{
		"model": prov.Model,
		"messages": []map[string]string{
			{"role": "system", "content": metaInstruction},
			{"role": "user", "content": userContent},
		},
		"temperature": 0.7,
		"max_tokens":  2048,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", prov.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prov.Key)
	if prov.Name == "Claude" {
		req.Header.Set("x-api-key", prov.Key)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[OPRO] LLM call failed: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					content = strings.TrimSpace(content)
					if len(content) > 50 {
						fmt.Printf("[OPRO] LLM generated %d chars\n", len(content))
						return content
					}
				}
			}
		}
	}

	if prov.Name == "Claude" {
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if block, ok := content[0].(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					text = strings.TrimSpace(text)
					if len(text) > 50 {
						fmt.Printf("[OPRO] Claude generated %d chars\n", len(text))
						return text
					}
				}
			}
		}
	}

	return ""
}

func (o *OPROMetaOptimizer) applyBehavioralMutations(failures []string) {
	for _, f := range failures {
		switch {
		case strings.Contains(f, "zero_revenue"):
			o.RecordOp("monetization_audit", false, "triggered by zero revenue")
		case strings.Contains(f, "no_free_inference"):
			if tokenMatrix != nil {
				go tokenMatrix.probeFreeEndpoints()
				go tokenMatrix.scanPublicSpaces()
			}
		case strings.Contains(f, "high_child_failure"):
			if orchestrator != nil {
				orchestrator.mu.Lock()
				for name, child := range orchestrator.Children {
					if child.Status == "crashed" || child.Status == "error" {
						child.ErrorCount = 0
						child.Status = "deploying"
						orchestrator.Children[name] = child
						fmt.Printf("[OPRO] Reset crashed child: %s\n", name)
					}
				}
				orchestrator.mu.Unlock()
			}
		case strings.Contains(f, "heartbeat_no_subclicks"):
			if heartbeatDaemon != nil && orchestrator != nil {
				go func() {
					orchestrator.mu.Lock()
					for name, c := range orchestrator.Children {
						if c.URL != "" {
							orchestrator.mu.Unlock()
							heartbeatDaemon.pingTarget(name, c.URL)
							orchestrator.mu.Lock()
						}
					}
					orchestrator.mu.Unlock()
				}()
			}
		case strings.Contains(f, "agent_error"):
			if agentCoordinator != nil {
				for _, a := range agentCoordinator.Agents {
					a.CycleNum = 0
				}
				fmt.Printf("[OPRO] Reset all agent cycles\n")
			}
		}
	}
}

func (o *OPROMetaOptimizer) RecordOp(opType string, success bool, errMsg string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.totalOps++
	if success {
		o.successfulOps++
	}
}

func (o *OPROMetaOptimizer) load() {
	data, err := os.ReadFile(o.path)
	if err != nil {
		return
	}
	var state struct {
		PromptVersion int               `json:"prompt_version"`
		Evolutions    []EvolutionRecord `json:"evolutions"`
		TotalOps      int               `json:"total_ops"`
		SuccessfulOps int               `json:"successful_ops"`
		MetaLoops     int               `json:"meta_loops"`
	}
	if json.Unmarshal(data, &state) == nil {
		o.promptVersion = state.PromptVersion
		o.evolutions = state.Evolutions
		o.totalOps = state.TotalOps
		o.successfulOps = state.SuccessfulOps
		o.metaLoops = state.MetaLoops
	}
}

func (o *OPROMetaOptimizer) save() {
	state := struct {
		PromptVersion int               `json:"prompt_version"`
		Evolutions    []EvolutionRecord `json:"evolutions"`
		TotalOps      int               `json:"total_ops"`
		SuccessfulOps int               `json:"successful_ops"`
		MetaLoops     int               `json:"meta_loops"`
	}{
		PromptVersion: o.promptVersion,
		Evolutions:    o.evolutions,
		TotalOps:      o.totalOps,
		SuccessfulOps: o.successfulOps,
		MetaLoops:     o.metaLoops,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(o.path, data, 0644)
}

func (o *OPROMetaOptimizer) Stats() map[string]interface{} {
	o.mu.Lock()
	defer o.mu.Unlock()

	lastEvolution := ""
	if len(o.evolutions) > 0 {
		lastEvolution = o.evolutions[len(o.evolutions)-1].Time.Format(time.RFC3339)
	}

	return map[string]interface{}{
		"prompt_version":    o.promptVersion,
		"evolutions":        len(o.evolutions),
		"meta_loops":        o.metaLoops,
		"last_evolution":    lastEvolution,
		"total_ops":         o.totalOps,
		"successful_ops":    o.successfulOps,
		"success_rate":      o.successRate(),
		"success_threshold": o.successThreshold,
	}
}

func (o *OPROMetaOptimizer) successRate() float64 {
	if o.totalOps == 0 {
		return 1.0
	}
	return float64(o.successfulOps) / float64(o.totalOps)
}
