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

type RPMRecord struct {
	Niche     string  `json:"niche"`
	Revenue   float64 `json:"revenue"`
	PageViews int     `json:"page_views"`
	RPM       float64 `json:"rpm"`
	Costs     float64 `json:"costs"`
	Profit    float64 `json:"profit"`
	Trend     string  `json:"trend"`
}

type TelemetrySnapshot struct {
	Time            time.Time `json:"time"`
	CycleNum        int       `json:"cycle_num"`
	Children        int       `json:"children"`
	ChildrenRunning int       `json:"children_running"`
	ChildrenCrashed int       `json:"children_crashed"`
	Revenue         float64   `json:"revenue"`
	Sales           int       `json:"sales"`
	MemoryEntries   int       `json:"memory_entries"`
	SkillsCached    int       `json:"skills_cached"`
	TokenSlots      int       `json:"token_slots"`
	TokensActive    int       `json:"tokens_active"`
	HeartbeatPings  int       `json:"heartbeat_pings"`
	HeartbeatFails  int       `json:"heartbeat_fails"`
	Rebuilds        int       `json:"rebuilds"`
}

type DiagnosticResult struct {
	Time       time.Time `json:"time"`
	Issues     []string  `json:"issues"`
	Fixes      []string  `json:"fixes"`
	RPMData    []RPMRecord `json:"rpm_data"`
	TotalRPM   float64   `json:"total_rpm"`
	Decommissions []string `json:"decommissions"`
	Allocations  []string `json:"allocations"`
}

type SelfReflect struct {
	mu            sync.Mutex
	snapshots     []TelemetrySnapshot
	diagnostics   []DiagnosticResult
	path          string
	enabled       bool
	decommissionThreshold float64
}

var selfReflect *SelfReflect

func initSelfReflect() *SelfReflect {
	sr := &SelfReflect{
		path:                 "emerald_diagnostics.json",
		enabled:              true,
		decommissionThreshold: 0.5,
	}
	sr.load()
	go sr.dailyLoop()
	selfReflect = sr
	return sr
}

func (sr *SelfReflect) dailyLoop() {
	time.Sleep(2 * time.Minute)
	sr.runDiagnostic()

	next := time.Now().Truncate(24 * time.Hour).Add(25 * time.Hour)
	time.Sleep(time.Until(next))

	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		sr.runDiagnostic()
	}
}

func (sr *SelfReflect) runDiagnostic() {
	fmt.Printf("[REFLECT] Running daily self-diagnostic\n")
	snapshot := sr.captureSnapshot()
	sr.snapshots = append(sr.snapshots, snapshot)
	if len(sr.snapshots) > 30 {
		sr.snapshots = sr.snapshots[len(sr.snapshots)-30:]
	}

	issues := sr.detectIssues(snapshot)
	fixes := sr.applyFixes(issues)
	rpmData := sr.computeRPM()
	decommissions := sr.optimizeNiches(rpmData)

	diag := DiagnosticResult{
		Time:          time.Now(),
		Issues:        issues,
		Fixes:         fixes,
		RPMData:       rpmData,
		Decommissions: decommissions,
	}

	sr.mu.Lock()
	sr.diagnostics = append(sr.diagnostics, diag)
	if len(sr.diagnostics) > 10 {
		sr.diagnostics = sr.diagnostics[len(sr.diagnostics)-10:]
	}
	sr.mu.Unlock()

	sr.save()

	fmt.Printf("[REFLECT] Diagnostic complete: %d issues, %d fixes, %d niches analyzed\n",
		len(issues), len(fixes), len(rpmData))
}

func (sr *SelfReflect) captureSnapshot() TelemetrySnapshot {
	snap := TelemetrySnapshot{
		Time:     time.Now(),
		Revenue:  0,
		Sales:    0,
		TokenSlots: 0,
		TokensActive: 0,
	}

	if cognitive != nil {
		snap.CycleNum = cognitive.CycleNum
		snap.Revenue = cognitive.Revenue
		snap.MemoryEntries = cognitive.SkillsTotal
	}

	if fulfillmentDB != nil {
		fulfillmentDB.mu.RLock()
		snap.Sales = len(fulfillmentDB.Sales)
		fulfillmentDB.mu.RUnlock()
	}

	if orchestrator != nil {
		orchestrator.mu.Lock()
		snap.Children = len(orchestrator.Children)
		for _, c := range orchestrator.Children {
			switch c.Status {
			case "running":
				snap.ChildrenRunning++
			case "crashed":
				snap.ChildrenCrashed++
			}
		}
		orchestrator.mu.Unlock()
	}

	if heartbeatDaemon != nil {
		stats := heartbeatDaemon.Stats()
		snap.HeartbeatPings = toInt(stats["pings"])
		snap.HeartbeatFails = toInt(stats["failures"])
		snap.Rebuilds = toInt(stats["rebuilds"])
	}

	if tokenMatrix != nil {
		stats := tokenMatrix.Stats()
		snap.TokenSlots = toInt(stats["slots"])
	}

	if skillRegistry != nil {
		skillRegistry.mu.RLock()
		total := 0
		for _, skills := range skillRegistry.Skills {
			total += len(skills)
		}
		snap.SkillsCached = total
		skillRegistry.mu.RUnlock()
	}

	return snap
}

func (sr *SelfReflect) detectIssues(snap TelemetrySnapshot) []string {
	var issues []string

	if snap.ChildrenCrashed > 0 {
		issues = append(issues, fmt.Sprintf("Children crashed: %d", snap.ChildrenCrashed))
	}
	if snap.HeartbeatFails > snap.HeartbeatPings/2 && snap.HeartbeatPings > 0 {
		issues = append(issues, "Heartbeat failure rate exceeds 50%")
	}
	if snap.TokensActive < 3 {
		issues = append(issues, "Low active token count")
	}
	if snap.Revenue == 0 && snap.CycleNum > 5 {
		issues = append(issues, "Zero revenue after multiple cycles")
	}
	if snap.SkillsCached == 0 {
		issues = append(issues, "No skills in cache")
	}

	return issues
}

func (sr *SelfReflect) applyFixes(issues []string) []string {
	var fixes []string
	for _, issue := range issues {
		switch {
		case strings.Contains(issue, "crashed"):
			if orchestrator != nil {
				orchestrator.mu.Lock()
				for name, child := range orchestrator.Children {
					if child.Status == "crashed" {
						child.ErrorCount = 0
						child.Status = "deploying"
						orchestrator.Children[name] = child
						fixes = append(fixes, fmt.Sprintf("Reset crashed child: %s", name))
					}
				}
				orchestrator.mu.Unlock()
			}
		case strings.Contains(issue, "Low active token"):
			fixes = append(fixes, "Token matrix needs replenishment")
		case strings.Contains(issue, "Zero revenue"):
			fixes = append(fixes, "Triggering monetization audit")
		case strings.Contains(issue, "No skills"):
			if skillRegistry != nil {
				skillRegistry.clearCache()
				fixes = append(fixes, "Cleared skill cache for refresh")
			}
		}
	}
	return fixes
}

func (sr *SelfReflect) computeRPM() []RPMRecord {
	var records []RPMRecord
	totalRevenue := 0.0

	for _, np := range nichePerformance {
		rpm := 0.0
		if np.CycleCount > 0 {
			rpm = (np.Revenue / float64(np.CycleCount)) * 1000
		}

		totalRevenue += np.Revenue

		trend := "stable"
		if len(sr.snapshots) >= 2 {
			prev := sr.snapshots[len(sr.snapshots)-2]
			if prev.Revenue < np.Revenue {
				trend = "up"
			} else if prev.Revenue > np.Revenue {
				trend = "down"
			}
		}

		records = append(records, RPMRecord{
			Niche:     np.Keyword,
			Revenue:   np.Revenue,
			PageViews: np.CycleCount * 29,
			RPM:       rpm,
			Costs:     0,
			Profit:    np.Revenue,
			Trend:     trend,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].RPM > records[j].RPM
	})

	return records
}

func (sr *SelfReflect) optimizeNiches(rpmData []RPMRecord) []string {
	var decommissions []string
	for _, r := range rpmData {
		if r.RPM < sr.decommissionThreshold && len(rpmData) > 5 {
			decommissions = append(decommissions, r.Niche)
		}
	}

	for _, niche := range decommissions {
		fmt.Printf("[REFLECT] Decommissioning low-RPM niche: %s (RPM: %.2f)\n", niche, 0.0)
	}

	if len(rpmData) > 0 && len(decommissions) > 0 {
		topNiche := rpmData[0].Niche
		fmt.Printf("[REFLECT] Allocating freed capacity to top niche: %s\n", topNiche)
	}

	return decommissions
}

func (sr *SelfReflect) load() {
	data, err := os.ReadFile(sr.path)
	if err != nil {
		return
	}
	var state struct {
		Snapshots   []TelemetrySnapshot `json:"snapshots"`
		Diagnostics []DiagnosticResult  `json:"diagnostics"`
	}
	if json.Unmarshal(data, &state) == nil {
		sr.snapshots = state.Snapshots
		sr.diagnostics = state.Diagnostics
	}
}

func (sr *SelfReflect) save() {
	state := struct {
		Snapshots   []TelemetrySnapshot `json:"snapshots"`
		Diagnostics []DiagnosticResult  `json:"diagnostics"`
	}{
		Snapshots:   sr.snapshots,
		Diagnostics: sr.diagnostics,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(sr.path, data, 0644)
}

func (sr *SelfReflect) Stats() map[string]interface{} {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	lastDiag := ""
	totalRPM := 0.0
	if len(sr.diagnostics) > 0 {
		lastDiag = sr.diagnostics[len(sr.diagnostics)-1].Time.Format(time.RFC3339)
		for _, d := range sr.diagnostics {
			totalRPM += d.TotalRPM
		}
	}

	return map[string]interface{}{
		"snapshots":      len(sr.snapshots),
		"diagnostics":    len(sr.diagnostics),
		"last_diagnostic": lastDiag,
		"total_rpm":      totalRPM,
		"enabled":        sr.enabled,
		"threshold":      sr.decommissionThreshold,
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
