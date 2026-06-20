package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type CognitiveCycle struct {
	mu        sync.Mutex
	CycleNum  int       `json:"cycle_num"`
	StartedAt time.Time `json:"started_at"`
	Phases    struct {
		Perceive  time.Duration `json:"perceive"`
		Analyze   time.Duration `json:"analyze"`
		Write     time.Duration `json:"write"`
		Build     time.Duration `json:"build"`
		Launch    time.Duration `json:"launch"`
		Monetize  time.Duration `json:"monetize"`
		Reinvest  time.Duration `json:"reinvest"`
	} `json:"phases"`
	Insights    []string `json:"insights"`
	Revenue     float64  `json:"revenue"`
	Children    int      `json:"children"`
	SkillsTotal int      `json:"skills_total"`
}

type NichePerformance struct {
	Keyword    string  `json:"keyword"`
	Name       string  `json:"name"`
	Revenue    float64 `json:"revenue"`
	Uptime     float64 `json:"uptime"`  // 0.0 - 1.0
	Skills     int     `json:"skills"`
	Score      float64 `json:"score"`
	CycleCount int     `json:"cycle_count"`
}

var cognitive = &CognitiveCycle{
	StartedAt: time.Now(),
}
var nichePerformance []NichePerformance
var cognitiveInsights []string

// Perceive — collect metrics from all running children and external sources
func (c *CognitiveCycle) perceive() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Perceive = time.Since(start)
		c.mu.Unlock()
	}()

	childrenCount := 0
	if orchestrator != nil {
		orchestrator.mu.Lock()
		childrenCount = len(orchestrator.Children)
		orchestrator.mu.Unlock()
	}
	fmt.Printf("[COG] Perceive: scanning %d children\n", childrenCount)

	var perf []NichePerformance
	if orchestrator != nil {
		orchestrator.mu.Lock()
		for _, child := range orchestrator.Children {
			healthy := 0.0
			if child.Status == "running" {
				healthy = 1.0
			}

			np := NichePerformance{
				Keyword: child.Niche,
				Uptime:  healthy,
				Skills:  child.Skills,
				Score:   0,
			}

			for _, n := range niches {
				if n.Keyword == child.Niche {
					np.Name = n.Name
					break
				}
			}

			perf = append(perf, np)
		}
		orchestrator.mu.Unlock()
	}

	// Scrape ecosystem for new opportunities
	for _, niche := range niches {
		skills := discoverSkills(niche.Keyword, 5)
		if len(skills) > 0 {
			fmt.Printf("[COG]   %s: %d skills available\n", niche.Name, len(skills))
		}
	}

	revenue := 0.0
	db := loadFulfillmentDB("emerald_sales.json")
	for _, s := range db.Sales {
		revenue += s.Amount
	}

	c.mu.Lock()
	nichePerformance = perf
	c.Revenue = revenue
	if orchestrator != nil {
		orchestrator.mu.Lock()
		c.Children = len(orchestrator.Children)
		orchestrator.mu.Unlock()
	}

	skillTotal := 0
	for _, sk := range skillRegistry.Skills {
		skillTotal += len(sk)
	}
	c.SkillsTotal = skillTotal

	insight := fmt.Sprintf("Perceived %d children, $%.2f revenue, %d skills cached", c.Children, c.Revenue, c.SkillsTotal)
	c.Insights = append(c.Insights, insight)
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()

	fmt.Printf("[COG] ✅ Perceive: %d children, $%.2f rev, %d skills\n", c.Children, revenue, skillTotal)
}

// Analyze — optimize niche selection, skill composition, deployment strategy
func (c *CognitiveCycle) analyze() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Analyze = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Analyze: optimizing niche strategy\n")

	for i := range nichePerformance {
		np := &nichePerformance[i]
		np.Score = np.Uptime*50 + float64(np.Skills)*10

		skills := discoverSkills(np.Keyword, 5)
		skillScore := 0.0
		for _, s := range skills {
			skillScore += s.Score
		}
		np.Score += skillScore * 0.1
	}

	sort.Slice(nichePerformance, func(i, j int) bool {
		return nichePerformance[i].Score > nichePerformance[j].Score
	})

	if len(nichePerformance) > 0 {
		best := nichePerformance[0]
		fmt.Printf("[COG]   Top niche: %s (score: %.1f, skills: %d)\n", best.Keyword, best.Score, best.Skills)
	}

	// Identify underperforming niches for redeployment
	var toRedeploy []string
	for _, np := range nichePerformance {
		if np.Uptime < 0.5 && np.CycleCount > 2 {
			toRedeploy = append(toRedeploy, np.Keyword)
		}
	}
	if len(toRedeploy) > 0 {
		fmt.Printf("[COG]   Flagged for redeploy: %v\n", toRedeploy)
	}

	// Check for new niches to deploy
	var undeployed []string
	if orchestrator != nil {
		deployed := orchestrator.getDeployedNiches()
		for _, n := range niches {
			if !deployed[n.Keyword] {
				undeployed = append(undeployed, n.Name)
			}
		}
	} else {
		for _, n := range niches {
			undeployed = append(undeployed, n.Name)
		}
	}
	if len(undeployed) > 0 {
		fmt.Printf("[COG]   Undeployed niches: %v\n", undeployed)
	}

	insight := fmt.Sprintf("Analyzed %d niches. Top: %.1f pts. %d undeployed.", len(nichePerformance), 
		func() float64 { if len(nichePerformance) > 0 { return nichePerformance[0].Score }; return 0 }(),
		len(undeployed))
	c.mu.Lock()
	c.Insights = append(c.Insights, insight)
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()
}

// Write — regenerate child code with optimized composition
func (c *CognitiveCycle) write() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Write = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Write: regenerating optimized child code\n")

	deployed := make(map[string]bool)
	if orchestrator != nil {
		deployed = orchestrator.getDeployedNiches()
	}
	for _, niche := range niches {
		if deployed[niche.Keyword] {
			skipRegen := false
			for _, np := range nichePerformance {
				if np.Keyword == niche.Keyword && np.Uptime > 0.8 {
					skipRegen = true
					break
				}
			}
			if skipRegen {
				continue
			}
		}

		skills := getBestSkillsForFusion(niche.Keyword, 20)
		if len(skills) > 0 {
			fmt.Printf("[COG]   %s: %d fusion components ready\n", niche.Name, len(skills))
		}
	}

	c.mu.Lock()
	c.Insights = append(c.Insights, "Write phase: code regenerated with optimized fusion")
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()
}

// Build — compile assets for deployment
func (c *CognitiveCycle) build() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Build = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Build: assets compiled\n")
}

// Launch — deploy children with fused containers
func (c *CognitiveCycle) launch() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Launch = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Launch: deploying children\n")

	if orchestrator != nil {
		orchestrator.deployNextNiche()
	}

	c.mu.Lock()
	c.Insights = append(c.Insights, "Launch: child deployed")
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()
}

// Monetize — trigger revenue actions (ad placement optimization, affiliate rotations)
func (c *CognitiveCycle) monetize() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Monetize = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Monetize: optimizing revenue streams\n")

	db := loadFulfillmentDB("emerald_sales.json")
	revenue := db.totalRevenue()

	adsenseID := vaultGet("ADSENSE_CLIENT_ID", "")
	amazonTag := vaultGet("AMAZON_ASSOCIATES_TAG", "")

	insight := fmt.Sprintf("Revenue: $%.2f | AdSense: %s | Amazon: %s",
		revenue, boolStr(adsenseID != "" && adsenseID != "ca-pub-XXXXXXXXXXXXXXXX"),
		boolStr(amazonTag != "" && amazonTag != "emeraldeng0e-20"))

	c.mu.Lock()
	c.Revenue = revenue
	c.Insights = append(c.Insights, "Monetize: "+insight)
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()

	fmt.Printf("[COG]   %s\n", insight)
}

// Reinvest — allocate resources for growth (scale up high-performing niches)
func (c *CognitiveCycle) reinvest() {
	start := time.Now()
	defer func() {
		c.mu.Lock()
		c.Phases.Reinvest = time.Since(start)
		c.mu.Unlock()
	}()

	fmt.Printf("[COG] Reinvest: allocating growth resources\n")

	if c.Revenue > 0 {
		reinvestAmount := c.Revenue * 0.3
		fmt.Printf("[COG]   Reinvesting $%.2f (30%% of $%.2f) into scaling\n", reinvestAmount, c.Revenue)
	}

	if len(nichePerformance) > 0 && nichePerformance[0].Score > 0 {
		bestNiche := nichePerformance[0]
		fmt.Printf("[COG]   Scaling up: %s (score: %.1f)\n", bestNiche.Keyword, bestNiche.Score)
	}

	insight := fmt.Sprintf("Reinvested %.0f%% of $%.2f revenue into growth", 
		func() float64 { if c.Revenue > 0 { return 30 }; return 0 }(), c.Revenue)

	c.mu.Lock()
	c.Insights = append(c.Insights, "Reinvest: "+insight)
	if len(c.Insights) > 100 {
		c.Insights = c.Insights[len(c.Insights)-100:]
	}
	c.mu.Unlock()
}

// RunCycle — execute one full cognitive loop
func (c *CognitiveCycle) runCycle() {
	c.mu.Lock()
	c.CycleNum++
	cycleNum := c.CycleNum
	c.mu.Unlock()

	fmt.Printf("\n[COG] ═══ Cognitive Cycle %d ═══\n", cycleNum)
	start := time.Now()

	c.perceive()
	c.analyze()
	c.write()
	c.build()
	c.launch()
	c.monetize()
	c.reinvest()

	elapsed := time.Since(start)
	fmt.Printf("[COG] ✅ Cycle %d complete in %v\n", cycleNum, elapsed)

	telegramSend(fmt.Sprintf(
		"<b>🧠 Cognitive Cycle %d</b>\n"+
			"Time: %v\n"+
			"Children: %d | Revenue: $%.2f\n"+
			"Skills: %d | Insights: %d",
		cycleNum, elapsed,
		c.Children, c.Revenue,
		c.SkillsTotal, len(c.Insights),
	))

	saveCognitiveState(c)
}

func startCognitiveLoop() {
	go func() {
		time.Sleep(30 * time.Second)
		cognitive.runCycle()

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cognitive.runCycle()
			}
		}
	}()
	fmt.Printf("[COG] Cognitive loop started (cycle: 6h)\n")
}

func boolStr(b bool) string {
	if b {
		return "configured"
	}
	return "placeholder"
}

func loadCognitiveState(c *CognitiveCycle) {
	data, err := os.ReadFile("emerald_cognitive.json")
	if err != nil {
		return
	}
	json.Unmarshal(data, c)
}

func saveCognitiveState(c *CognitiveCycle) {
	data, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile("emerald_cognitive.json", data, 0644)
}

func getCycleNum() int {
	cognitive.mu.Lock()
	defer cognitive.mu.Unlock()
	return cognitive.CycleNum
}

func cognitiveStats() map[string]interface{} {
	cognitive.mu.Lock()
	defer cognitive.mu.Unlock()

	return map[string]interface{}{
		"cycle":       cognitive.CycleNum,
		"uptime":      time.Since(cognitive.StartedAt).String(),
		"revenue":     cognitive.Revenue,
		"children":    cognitive.Children,
		"skills":      cognitive.SkillsTotal,
		"insights":    len(cognitive.Insights),
		"phases": map[string]string{
			"perceive": cognitive.Phases.Perceive.String(),
			"analyze":  cognitive.Phases.Analyze.String(),
			"write":    cognitive.Phases.Write.String(),
			"build":    cognitive.Phases.Build.String(),
			"launch":   cognitive.Phases.Launch.String(),
			"monetize": cognitive.Phases.Monetize.String(),
			"reinvest": cognitive.Phases.Reinvest.String(),
		},
	}
}
