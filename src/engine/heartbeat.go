package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"
)

type HeartbeatRecord struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Latency   int64     `json:"latency_ms"`
	LastPing  time.Time `json:"last_ping"`
	FailCount int       `json:"fail_count"`
}

type HeartbeatDaemon struct {
	mu              sync.Mutex
	records         []HeartbeatRecord
	interval        time.Duration
	jitterMax       int
	httpClient      *http.Client
	jar             *cookiejar.Jar
	pings           int
	failures        int
	subClicks       int
	rebuilds        int
	running         bool
	userAgents      []string
	cognitiveQueries []string
	externalPlatforms []struct{name, url string}
	refererURL      string
	fallbackToken   string
}

var heartbeatDaemon *HeartbeatDaemon

func initHeartbeatDaemon() *HeartbeatDaemon {
	jar, _ := cookiejar.New(nil)
	hbd := &HeartbeatDaemon{
		interval:     25 * time.Minute,
		jitterMax:    300,
		httpClient:   &http.Client{Timeout: 10 * time.Second, Jar: jar},
		jar:          jar,
		refererURL:   "https://www.google.com/",
		fallbackToken: "FALLBACK_FREE_CORE_INFERENCE",
		userAgents: []string{
			"Mozilla/5.0 (Linux; Android 10; SM-N970F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1",
		},
		cognitiveQueries: []string{
			"How to optimize multi-agent pipeline parallel execution?",
			"Steps to upgrade sandbox security layers against runtime memory injection.",
			"Formulating high-RPM monetization structures via automated programmatic context.",
			"Refactoring dynamic UI layouts for maximizing organic interaction rates.",
			"Deploying zero-touch container micro-services globally across decentralized edge nodes.",
			"Scaling autonomous container synthesis across heterogeneous cloud runtimes.",
			"Implementing cross-platform secret rotation without service interruption.",
		},
		externalPlatforms: []struct{name, url string}{
			{"github_pages", "https://alexander101001.github.io/emerald-app"},
			{"huggingface_space", "https://huggingface.co/spaces/alexander101001/emerald-engine"},
			{"github_repo", "https://github.com/Alexander101001/emerald-engine"},
		},
	}
	go hbd.runLoop()
	heartbeatDaemon = hbd
	return hbd
}

func (h *HeartbeatDaemon) runLoop() {
	h.running = true
	time.Sleep(30 * time.Second)
	h.pingAll()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for range ticker.C {
		jitter := time.Duration(time.Now().UnixNano()%int64(h.jitterMax)) * time.Millisecond
		time.Sleep(jitter)
		h.pingAll()
	}
}

func (h *HeartbeatDaemon) pingAll() {
	var pingList []struct{name, url string}

	if orchestrator != nil {
		orchestrator.mu.Lock()
		for _, child := range orchestrator.Children {
			u := child.URL
			if u == "" {
				u = fmt.Sprintf("https://huggingface.co/spaces/%s/%s", orchestrator.Username, child.Name)
			}
			pingList = append(pingList, struct{name, url string}{child.Name, u})
		}
		orchestrator.mu.Unlock()
	}

	for _, p := range h.externalPlatforms {
		pingList = append(pingList, p)
	}

	if len(pingList) == 0 {
		return
	}

	// Cross-referral: visit GitHub repo from HF via Referer
	if len(pingList) >= 2 {
		for i := range pingList {
			prev := (i - 1 + len(pingList)) % len(pingList)
			h.refererURL = pingList[prev].url
			h.pingTarget(pingList[i].name, pingList[i].url)
		}
	} else {
		for _, target := range pingList {
			h.pingTarget(target.name, target.url)
		}
	}

	// CRITIC external verification on deployed children
	if reflexionLayer != nil && orchestrator != nil {
		orchestrator.mu.Lock()
		for name, child := range orchestrator.Children {
			if child.URL != "" {
				result := reflexionLayer.RunCriticExternalVerification(child.URL)
				if verified, ok := result["verified"].(bool); ok && !verified {
					fmt.Printf("[CRITIC] Verification failed for %s: %v\n", name, result["payload_status"])
				}
			}
		}
		orchestrator.mu.Unlock()
	}
}

func (h *HeartbeatDaemon) pingTarget(name, url string) {
	if chromiumPath != "" && h.pings%5 == 0 {
		h.browserDeepVisit(name, url)
	} else {
		h.simulateHumanVisit(name, url)
	}
}

func (h *HeartbeatDaemon) browserDeepVisit(name, url string) {
	start := time.Now()

	result := browserDeepPing(url)
	if result.Error != "" {
		fmt.Printf("[BROWSER] Deep ping failed for %s: %s\n", name, result.Error)
		h.recordFailure(name, url, start)
		return
	}

	readDelay := 5000 + time.Now().UnixNano()%20000
	time.Sleep(time.Duration(readDelay) * time.Millisecond)

	links := extractLinks(result.HTML, url)
	subClicked := false
	if len(links) > 0 && result.Screenshot != nil {
		subURL := links[time.Now().UnixNano()%int64(len(links))]
		subResult := browserDeepPing(subURL)
		if subResult.Error == "" && subResult.StatusCode == 200 {
			subClicked = true
			subDelay := 3000 + time.Now().UnixNano()%7000
			time.Sleep(time.Duration(subDelay) * time.Millisecond)
		}
	}

	healthLatency := time.Since(start).Milliseconds()
	_ = healthLatency

	isChild := true
	var childStatus string
	if orchestrator != nil {
		orchestrator.mu.Lock()
		if c, ok := orchestrator.Children[name]; ok {
			childStatus = c.Status
		} else {
			isChild = false
		}
		orchestrator.mu.Unlock()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	record := HeartbeatRecord{
		Name:     name,
		URL:      url,
		LastPing: time.Now(),
		Latency:  result.DurationMs,
		Status:   "alive",
	}

	if result.StatusCode >= 500 {
		record.Status = "down"
		record.FailCount = 1
		h.failures++

		for i, r := range h.records {
			if r.Name == name {
				record.FailCount = r.FailCount + 1
				h.records[i] = record
				goto maybeRebuild
			}
		}
		h.records = append(h.records, record)

	maybeRebuild:
		if record.FailCount >= 3 && isChild {
			go h.triggerRebuild(name)
		}
	} else {
		if isChild && (childStatus == "crashed" || childStatus == "error" || childStatus == "stopped") {
			go h.triggerRebuild(name)
			return
		}
	}

	if subClicked {
		h.subClicks++
	}

	for i, r := range h.records {
		if r.Name == name {
			h.records[i] = record
			return
		}
	}
	h.records = append(h.records, record)
	h.pings++

	if h.pings%10 == 0 {
		fmt.Printf("[REFLECT] Ping %d: %s | tokens: %d | clicks: %d\n",
			h.pings, record.Status, len(tokenMatrix.slots), h.subClicks)
	}
}

func (h *HeartbeatDaemon) simulateHumanVisit(name, url string) {
	start := time.Now()

	body, statusCode, _ := h.humanGet(url, false)
	if body == "" {
		h.recordFailure(name, url, start)
		return
	}

	readDelay := 5000 + time.Now().UnixNano()%20000
	time.Sleep(time.Duration(readDelay) * time.Millisecond)

	links := extractLinks(body, url)
	subClicked := false
	if len(links) > 0 {
		subURL := links[time.Now().UnixNano()%int64(len(links))]
		subBody, subCode, _ := h.humanGet(subURL, true)
		if subBody != "" && subCode == 200 {
			subClicked = true
			subDelay := 3000 + time.Now().UnixNano()%7000
			time.Sleep(time.Duration(subDelay) * time.Millisecond)
		}
	}

	healthLatency := time.Since(start).Milliseconds()

	isChild := true
	var childStatus string
	if orchestrator != nil {
		orchestrator.mu.Lock()
		if c, ok := orchestrator.Children[name]; ok {
			childStatus = c.Status
		} else {
			isChild = false
		}
		orchestrator.mu.Unlock()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	record := HeartbeatRecord{
		Name:     name,
		URL:      url,
		LastPing: time.Now(),
		Latency:  healthLatency,
	}

	if statusCode >= 500 || body == "" {
		record.Status = "down"
		record.FailCount = 1
		h.failures++

		for i, r := range h.records {
			if r.Name == name {
				record.FailCount = r.FailCount + 1
				h.records[i] = record
				goto maybeRebuild
			}
		}
		h.records = append(h.records, record)

	maybeRebuild:
		if record.FailCount >= 3 && isChild {
			go h.triggerRebuild(name)
		}
	} else {
		record.Status = "alive"
		if isChild && (childStatus == "crashed" || childStatus == "error" || childStatus == "stopped") {
			go h.triggerRebuild(name)
			return
		}
	}

	if subClicked {
		h.subClicks++
	}

	for i, r := range h.records {
		if r.Name == name {
			h.records[i] = record
			return
		}
	}
	h.records = append(h.records, record)
	h.pings++

	if h.pings%10 == 0 {
		fmt.Printf("[REFLECT] Ping %d: %s | tokens: %d | clicks: %d\n",
			h.pings, record.Status, len(tokenMatrix.slots), h.subClicks)
	}
}

func (h *HeartbeatDaemon) humanGet(url string, isSub bool) (string, int, int64) {
	req, _ := http.NewRequest("GET", url, nil)

	if fingerprintEngine != nil {
		fp := fingerprintEngine.Random()
		ApplyFingerprintHeaders(req, fp)
	} else {
		ua := h.userAgents[time.Now().UnixNano()%int64(len(h.userAgents))]
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	}

	req.Header.Set("Referer", h.refererURL)
	if isSub {
		req.Header.Set("Referer", url)
	}

	query := h.cognitiveQueries[time.Now().UnixNano()%int64(len(h.cognitiveQueries))]
	req.Header.Set("X-Context-Query", query)

	tok := tmGetToken("github")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok[:8]+"...")
	} else {
		req.Header.Set("Authorization", "Bearer "+h.fallbackToken)
	}

	start := time.Now()
	resp, err := h.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return "", 0, latency
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		h.refererURL = url
	}

	return string(body), resp.StatusCode, latency
}

func extractLinks(body, baseURL string) []string {
	var links []string
	lower := strings.ToLower(body)
	idx := 0
	for {
		startIdx := strings.Index(lower[idx:], `href="`)
		if startIdx < 0 {
			break
		}
		startIdx += idx + 6
		endIdx := strings.IndexByte(lower[startIdx:], '"')
		if endIdx < 0 {
			break
		}
		href := body[startIdx : startIdx+endIdx]
		idx = startIdx + endIdx

		if !strings.HasPrefix(href, "http") {
			continue
		}
		if strings.Contains(href, "google") || strings.Contains(href, "facebook") ||
			strings.Contains(href, "twitter") || strings.Contains(href, "linkedin") {
			continue
		}

		links = append(links, href)
		if len(links) >= 10 {
			break
		}
	}
	return links
}

func (h *HeartbeatDaemon) recordFailure(name, url string, start time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()

	isChild := true
	if orchestrator != nil {
		orchestrator.mu.Lock()
		if _, ok := orchestrator.Children[name]; !ok {
			isChild = false
		}
		orchestrator.mu.Unlock()
	}

	latency := time.Since(start).Milliseconds()
	record := HeartbeatRecord{
		Name:     name,
		URL:      url,
		LastPing: time.Now(),
		Latency:  latency,
		Status:   "down",
		FailCount: 1,
	}
	h.failures++

	for i, r := range h.records {
		if r.Name == name {
			record.FailCount = r.FailCount + 1
			h.records[i] = record
			if record.FailCount >= 3 && isChild {
				go h.triggerRebuild(name)
			}
			return
		}
	}
	h.records = append(h.records, record)
}

func (h *HeartbeatDaemon) triggerRebuild(name string) {
	orchestrator.mu.Lock()
	child, ok := orchestrator.Children[name]
	orchestrator.mu.Unlock()
	if !ok {
		return
	}

	fmt.Printf("[HEARTBEAT] Triggering rebuild for %s (state: %s, fails: %d)\n",
		name, child.Status, child.ErrorCount)

	orchestrator.mu.Lock()
	rebuildChild := orchestrator.Children[name]
	rebuildChild.Status = "rebuilding"
	orchestrator.Children[name] = rebuildChild
	orchestrator.mu.Unlock()

	go func() {
		err := orchestrator.restartSpace(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[HEARTBEAT] Restart failed for %s: %v\n", name, err)
			orchestrator.mu.Lock()
			if c, ok := orchestrator.Children[name]; ok {
				c.Status = "crashed"
				c.ErrorCount++
				orchestrator.Children[name] = c
			}
			orchestrator.mu.Unlock()
			return
		}

		orchestrator.mu.Lock()
		if c, ok := orchestrator.Children[name]; ok {
			c.Status = "running"
			c.ErrorCount = 0
			orchestrator.Children[name] = c
		}
		orchestrator.mu.Unlock()

		h.mu.Lock()
		h.rebuilds++
		h.mu.Unlock()

		fmt.Printf("[HEARTBEAT] %s rebuilt successfully\n", name)
	}()
}

func (h *HeartbeatDaemon) GenerateCFWorker() string {
	agentsJSON := toJSONString(h.userAgents)
	queriesJSON := toJSONString(h.cognitiveQueries)
	targetsJSON := h.generateTargetsJSON()
	return fmt.Sprintf(`// Auto-generated by emerald-engine heartbeat daemon
const USER_AGENTS = %s;
const QUERIES = %s;
export default {
  async scheduled(event, env, ctx) {
    const targets = %s;
    const ua = USER_AGENTS[Date.now() %% USER_AGENTS.length];
    const query = QUERIES[Date.now() %% QUERIES.length];
    const results = [];
    for (const target of targets) {
      try {
        const resp = await fetch(target.url, {
          method: 'GET',
          headers: {
            'User-Agent': ua,
            'Accept': 'text/html,*/*',
            'X-Context-Query': query
          }
        });
        results.push({ name: target.name, status: resp.status < 500 ? 'alive' : 'down' });
      } catch (e) {
        results.push({ name: target.name, status: 'down' });
      }
    }
    await fetch(env.ENGINE_URL + '/api/heartbeat/callback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ results, ts: Date.now(), source: 'cf-worker' })
    });
  }
};`, agentsJSON, queriesJSON, targetsJSON)
}

func toJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (h *HeartbeatDaemon) generateTargetsJSON() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var targets []map[string]string
	if orchestrator != nil {
		orchestrator.mu.Lock()
		for _, child := range orchestrator.Children {
			targets = append(targets, map[string]string{
				"name": child.Name,
				"url":  child.URL,
			})
		}
		orchestrator.mu.Unlock()
	}
	data, _ := json.Marshal(targets)
	return string(data)
}

func (h *HeartbeatDaemon) GenerateGHAWorkflow() string {
	return `name: Heartbeat
on:
  schedule:
    - cron: '*/25 * * * *'
jobs:
  ping:
    runs-on: ubuntu-latest
    steps:
      - name: Ping Engine
        run: |
          curl -sf -o /dev/null -w "%{http_code}" ${{ secrets.ENGINE_URL }}/health || true
`
}

func (h *HeartbeatDaemon) Stats() map[string]interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()

	alive := 0
	down := 0
	for _, r := range h.records {
		if r.Status == "alive" {
			alive++
		} else {
			down++
		}
	}

	return map[string]interface{}{
		"pings":     h.pings,
		"failures":  h.failures,
		"sub_clicks": h.subClicks,
		"rebuilds":  h.rebuilds,
		"alive":     alive,
		"down":      down,
		"interval":  h.interval.String(),
		"running":   h.running,
	}
}

func (h *HeartbeatDaemon) GetRecords() []HeartbeatRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]HeartbeatRecord, len(h.records))
	copy(out, h.records)
	return out
}
