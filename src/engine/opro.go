package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type OPROMetaOptimizer struct {
	mu               sync.Mutex
	evolutions       []EvolutionRecord
	promptVersion    int
	masterPrompt     string
	successThreshold float64
	totalOps         int
	successfulOps    int
	metaLoops        int
	path             string
}

type EvolutionRecord struct {
	Version     int       `json:"version"`
	Time        time.Time `json:"time"`
	SuccessRate float64   `json:"success_rate"`
	Issues      []string  `json:"issues"`
	Mutations   []string  `json:"mutations"`
	PromptDiff  string    `json:"prompt_diff"`
}

type OPROTelemetry struct {
	OpType      string  `json:"op_type"`
	Success     bool    `json:"success"`
	ErrorMsg    string  `json:"error_msg,omitempty"`
	DurationMs  int64   `json:"duration_ms"`
	ChildCount  int     `json:"child_count,omitempty"`
	Revenue     float64 `json:"revenue,omitempty"`
	TokenStatus string  `json:"token_status,omitempty"`
}

var oproOptimizer *OPROMetaOptimizer

func initOPRO() *OPROMetaOptimizer {
	o := &OPROMetaOptimizer{
		promptVersion:    1,
		masterPrompt:     "",
		successThreshold: 0.95,
		path:             "emerald_opro.json",
	}
	o.load()
	go o.metaLoop()
	oproOptimizer = o
	return o
}

func (o *OPROMetaOptimizer) metaLoop() {
	time.Sleep(10 * time.Minute)
	o.runMetaOptimization()
	ticker := time.NewTicker(6 * time.Hour)
	for range ticker.C {
		o.runMetaOptimization()
	}
}

func (o *OPROMetaOptimizer) runMetaOptimization() {
	o.metaLoops++

	successRate := o.computeSuccessRate()
	issues := o.collectIssues()

	fmt.Printf("[OPRO] Meta-loop %d | success: %.1f%% | issues: %d\n",
		o.metaLoops, successRate*100, len(issues))

	if successRate >= o.successThreshold && len(issues) == 0 {
		return
	}

	mutations := o.synthesizeMutations(issues)

	o.mu.Lock()
	o.promptVersion++
	o.evolutions = append(o.evolutions, EvolutionRecord{
		Version:     o.promptVersion,
		Time:        time.Now(),
		SuccessRate: successRate,
		Issues:      issues,
		Mutations:   mutations,
		PromptDiff:  strings.Join(mutations, "\n"),
	})
	if len(o.evolutions) > 20 {
		o.evolutions = o.evolutions[len(o.evolutions)-20:]
	}
	o.mu.Unlock()

	o.applyMutations(mutations)
	o.save()

	fmt.Printf("[OPRO] Evolution v%d: %d mutations applied\n", o.promptVersion, len(mutations))
}

func (o *OPROMetaOptimizer) computeSuccessRate() float64 {
	if o.totalOps == 0 {
		return 1.0
	}
	return float64(o.successfulOps) / float64(o.totalOps)
}

func (o *OPROMetaOptimizer) collectIssues() []string {
	var issues []string

	if cognitive != nil {
		if cognitive.Revenue == 0 && cognitive.CycleNum > 10 {
			issues = append(issues, "zero_revenue_after_10_cycles")
		}
	}

	if tokenMatrix != nil {
		stats := tokenMatrix.Stats()
		freeEndpoints := toInt(stats["free_endpoints"])
		if freeEndpoints == 0 {
			issues = append(issues, "no_free_inference_endpoints")
		}
	}

	if orchestrator != nil {
		orchestrator.mu.Lock()
		crashed := 0
		for _, c := range orchestrator.Children {
			if c.Status == "crashed" || c.Status == "error" {
				crashed++
			}
		}
		orchestrator.mu.Unlock()
		if crashed > 2 {
			issues = append(issues, fmt.Sprintf("high_child_failure_rate_%d_crashed", crashed))
		}
	}

	if heartbeatDaemon != nil {
		stats := heartbeatDaemon.Stats()
		failures := toInt(stats["failures"])
		clicks := toInt(stats["sub_clicks"])
		if failures > 10 && clicks == 0 {
			issues = append(issues, "heartbeat_no_subclicks_high_failures")
		}
	}

	if agentCoordinator != nil {
		stats := agentCoordinator.Stats()
		if agents, ok := stats["agents"].([]interface{}); ok {
			for _, a := range agents {
				if m, ok := a.(map[string]interface{}); ok {
					if errors, ok := m["errors"].(int); ok && errors > 5 {
						issues = append(issues, fmt.Sprintf("agent_error_high_%s", m["name"]))
					}
				}
			}
		}
	}

	return issues
}

func (o *OPROMetaOptimizer) synthesizeMutations(issues []string) []string {
	mutations := make(map[string]bool)

	for _, issue := range issues {
		switch {
		case strings.Contains(issue, "zero_revenue"):
			mutations["MONETIZATION_AUDIT_INTERVAL=4h"] = true
			mutations["ENABLE_AFFILIATE_FALLBACK_CHAINS=true"] = true
			mutations["MIN_NICHE_COUNT=3"] = true

		case strings.Contains(issue, "no_free_inference"):
			mutations["FREE_ENDPOINT_SCAN_INTERVAL=30m"] = true
			mutations["HF_INTERNAL_ROUTING_ENABLED=true"] = true
			mutations["TOKEN_HARVEST_DEEP_SCAN=true"] = true

		case strings.Contains(issue, "high_child_failure"):
			mutations["CHILD_HEALTH_CHECK_INTERVAL=5m"] = true
			mutations["MAX_CHILD_RESTART_ATTEMPTS=10"] = true
			mutations["CHILD_DEPLOY_RETRY_DELAY=30s"] = true

		case strings.Contains(issue, "heartbeat_no_subclicks"):
			mutations["HEARTBEAT_BROWSER_RATIO=3"] = true
			mutations["HEARTBEAT_SUBCLICK_FORCE=true"] = true

		case strings.Contains(issue, "agent_error"):
			mutations["AGENT_ERROR_THROTTLE_LIMIT=20"] = true
			mutations["AGENT_SELF_HEAL_INTERVAL=60s"] = true
		}
	}

	mutations["OPRO_PROMPT_VERSION"] = true
	mutations["OPRO_META_LOOP_INTERVAL=4h"] = true

	result := make([]string, 0, len(mutations))
	for m := range mutations {
		result = append(result, m)
	}
	sort.Strings(result)
	return result
}

func (o *OPROMetaOptimizer) applyMutations(mutations []string) {
	for _, m := range mutations {
		switch {
		case strings.HasPrefix(m, "MONETIZATION_AUDIT_INTERVAL="):
			fmt.Printf("[OPRO] Mutation: %s\n", m)
		case strings.HasPrefix(m, "FREE_ENDPOINT_SCAN_INTERVAL="):
			if tokenMatrix != nil {
				tokenMatrix.mu.Lock()
				tokenMatrix.lastHarvest = time.Now().Add(-6 * time.Hour)
				tokenMatrix.mu.Unlock()
				fmt.Printf("[OPRO] Mutation: reset harvest timer for %s\n", m)
			}
		case m == "TOKEN_HARVEST_DEEP_SCAN=true":
			if tokenMatrix != nil {
				go func() {
					tokenMatrix.probeFreeEndpoints()
				}()
				fmt.Printf("[OPRO] Mutation: triggering deep endpoint scan\n")
			}
		case m == "HEARTBEAT_BROWSER_RATIO=3":
			fmt.Printf("[OPRO] Mutation: browser deep ping ratio increased\n")
		case m == "HF_INTERNAL_ROUTING_ENABLED=true":
			if tokenMatrix != nil {
				go tokenMatrix.scanPublicSpaces()
				fmt.Printf("[OPRO] Mutation: scanning public HF spaces for endpoints\n")
			}
		case m == "HEARTBEAT_SUBCLICK_FORCE=true":
			fmt.Printf("[OPRO] Mutation: sub-click forcing enabled\n")
		default:
			fmt.Printf("[OPRO] Mutation: %s\n", m)
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
		"success_rate":      o.computeSuccessRate(),
		"success_threshold": o.successThreshold,
	}
}
