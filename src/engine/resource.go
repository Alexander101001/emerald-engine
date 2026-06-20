package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

type ComputeBudget struct {
	mu             sync.Mutex
	TotalSpaces    int            `json:"total_spaces"`
	ActiveChildren int            `json:"active_children"`
	IdleChildren   int            `json:"idle_children"`
	CPULimit       int            `json:"cpu_limit"`
	MemoryLimitMB  int64          `json:"memory_limit_mb"`
	FreeTier       bool           `json:"free_tier"`
	HarvestedCPU   float64        `json:"harvested_cpu"`
	HarvestedMemMB int64          `json:"harvested_mem_mb"`
	LastOptimized  time.Time      `json:"last_optimized"`
	SpaceTiers     map[string]int `json:"space_tiers"` // space name -> tier (0=free, 1=basic, 2=pro)
}

type ScalingDecision struct {
	Action       string `json:"action"` // deploy, idle, terminate, scale_up
	Reason       string `json:"reason"`
	TargetNiche  string `json:"target_niche"`
	Priority     int    `json:"priority"`
}

var resourceManager = &ComputeBudget{
	CPULimit:      2,
	MemoryLimitMB: 16384,
	FreeTier:      true,
	SpaceTiers:    make(map[string]int),
	LastOptimized: time.Now(),
}

// ─── Resource Discovery ───

func detectPlatformLimits() {
	resourceManager.mu.Lock()
	defer resourceManager.mu.Unlock()

	cpus := runtime.NumCPU()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("[RES] Platform: %d CPUs, %d MB allocated\n", cpus, memStats.Alloc/1024/1024)

	resourceManager.CPULimit = cpus
	resourceManager.MemoryLimitMB = int64(memStats.TotalAlloc / 1024 / 1024 * 2)
	if resourceManager.MemoryLimitMB < 1024 {
		resourceManager.MemoryLimitMB = 1024
	}

	platform := os.Getenv("SPACE_ID")
	if platform != "" {
		resourceManager.FreeTier = true
		fmt.Printf("[RES] Running on HuggingFace Space (free tier optimizations enabled)\n")
	} else {
		resourceManager.FreeTier = true
	}
}

func (rb *ComputeBudget) recordDeployment(spaceName string, tier int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.TotalSpaces++
	rb.ActiveChildren++
	rb.SpaceTiers[spaceName] = tier
}

func (rb *ComputeBudget) recordIdle(spaceName string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.ActiveChildren--
	rb.IdleChildren++
}

func (rb *ComputeBudget) recordTermination(spaceName string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.TotalSpaces--
	if rb.SpaceTiers[spaceName] == 1 {
		rb.ActiveChildren--
	} else {
		rb.IdleChildren--
	}
	delete(rb.SpaceTiers, spaceName)
}

// ─── Idle Management ───

func (rb *ComputeBudget) optimizeIdleChildren() []ScalingDecision {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var decisions []ScalingDecision

	if orchestrator == nil {
		return decisions
	}

	// Find crashed children to idle
	orchestrator.mu.Lock()
	for _, child := range orchestrator.Children {
		if child.Status == "crashed" && child.ErrorCount > 5 {
			decisions = append(decisions, ScalingDecision{
				Action:      "idle",
				Reason:      fmt.Sprintf("crashed %d times", child.ErrorCount),
				TargetNiche: child.Niche,
				Priority:    1,
			})
		}
	}

	// Scale down idle children if over budget
	if rb.ActiveChildren > rb.CPULimit*3 {
		excess := rb.ActiveChildren - rb.CPULimit*3
		for _, child := range orchestrator.Children {
			if excess <= 0 {
				break
			}
			if child.Status != "running" {
				decisions = append(decisions, ScalingDecision{
					Action:      "terminate",
					Reason:      "resource budget exceeded",
					TargetNiche: child.Niche,
					Priority:    2,
				})
				excess--
			}
		}
	}
	orchestrator.mu.Unlock()

	rb.LastOptimized = time.Now()
	return decisions
}

// ─── Scaling Decisions ───

func (rb *ComputeBudget) shouldDeploy() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	maxFree := rb.CPULimit * 4
	if rb.FreeTier && rb.ActiveChildren >= maxFree {
		fmt.Printf("[RES] Free tier limit reached (%d active). Waiting for capacity.\n", rb.ActiveChildren)
		return false
	}

	// HF Spaces free tier: max ~10 spaces
	if rb.FreeTier && rb.TotalSpaces >= 10 {
		fmt.Printf("[RES] Max 10 HF Spaces reached. Optimize existing before deploying.\n")
		return false
	}

	return true
}

func (rb *ComputeBudget) recommendTierForNiche(nicheKeyword string) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.FreeTier {
		return 0
	}

	highResource := map[string]bool{
		"ai-tools": true, "crypto-web3": true,
		"saas-marketing": true, "ecommerce": true,
	}

	if highResource[nicheKeyword] {
		return 1
	}
	return 0
}

// ─── Free Tier Strategy ───

func (rb *ComputeBudget) freeTierStrategy() map[string]interface{} {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	recommendations := map[string]interface{}{
		"max_spaces":              10,
		"current_spaces":          rb.TotalSpaces,
		"active_children":         rb.ActiveChildren,
		"idle_children":           rb.IdleChildren,
		"cpu_budget":             rb.CPULimit,
		"memory_budget_mb":       rb.MemoryLimitMB,
		"deploy_interval":        "6 hours (anti-throttle)",
		"health_interval":        "15 minutes",
		"harvesting_active":      true,
		"container_reuse":        true,
		"idle_reduction":         "Terminating crashed containers",
		"recommendations": []string{
			"Deploy max 1 child per 6 hours",
			"Use cpu-basic hardware for all spaces",
			"Terminate crashed containers within 3 cycles",
			"Stagger deployments across 15 niches",
			"Reuse idle containers for new niches",
		},
	}

	return recommendations
}

func (rb *ComputeBudget) harvestStats() map[string]interface{} {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return map[string]interface{}{
		"total_spaces":      rb.TotalSpaces,
		"active_children":  rb.ActiveChildren,
		"idle_children":    rb.IdleChildren,
		"cpu_limit":        rb.CPULimit,
		"memory_mb":        rb.MemoryLimitMB,
		"free_tier":        rb.FreeTier,
		"harvested_cpu":    rb.HarvestedCPU,
		"harvested_mem_mb": rb.HarvestedMemMB,
		"space_tiers":      rb.SpaceTiers,
	}
}

func initResourceManager() {
	detectPlatformLimits()

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				decisions := resourceManager.optimizeIdleChildren()
				if len(decisions) > 0 {
					fmt.Printf("[RES] %d optimization decisions\n", len(decisions))
					for _, d := range decisions {
						fmt.Printf("[RES]   %s: %s (%s)\n", d.Action, d.TargetNiche, d.Reason)
					}
				}
			}
		}
	}()

	fmt.Printf("[RES] Resource manager initialized (free tier: %v, %d CPUs)\n",
		resourceManager.FreeTier, resourceManager.CPULimit)
}

func saveResourceState() {
	data, _ := json.MarshalIndent(resourceManager, "", "  ")
	os.WriteFile("emerald_resource.json", data, 0644)
}

func loadResourceState() {
	data, err := os.ReadFile("emerald_resource.json")
	if err != nil {
		return
	}
	json.Unmarshal(data, resourceManager)
}
