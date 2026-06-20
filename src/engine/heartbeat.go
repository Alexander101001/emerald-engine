package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	pings           int
	failures        int
	rebuilds        int
	running         bool
	userAgents      []string
	cognitiveQueries []string
	externalPlatforms []struct{name, url string}
}

var heartbeatDaemon *HeartbeatDaemon

func initHeartbeatDaemon() *HeartbeatDaemon {
	hbd := &HeartbeatDaemon{
		interval:  25 * time.Minute,
		jitterMax: 300,
		httpClient: &http.Client{Timeout: 10 * time.Second},
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

	for _, target := range pingList {
		h.pingTarget(target.name, target.url)
	}
}

func (h *HeartbeatDaemon) pingTarget(name, url string) {
	start := time.Now()

	req, _ := http.NewRequest("GET", url, nil)
	ua := h.userAgents[time.Now().UnixNano()%int64(len(h.userAgents))]
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	query := h.cognitiveQueries[time.Now().UnixNano()%int64(len(h.cognitiveQueries))]
	req.Header.Set("X-Context-Query", query)

	tok := tmGetToken("github")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok[:8]+"...")
	}

	resp, err := h.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

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
		Latency:  latency,
	}

	if err != nil || (resp != nil && resp.StatusCode >= 500) {
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

	for i, r := range h.records {
		if r.Name == name {
			h.records[i] = record
			return
		}
	}
	h.records = append(h.records, record)
	h.pings++
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
		"pings":    h.pings,
		"failures": h.failures,
		"rebuilds": h.rebuilds,
		"alive":    alive,
		"down":     down,
		"interval": h.interval.String(),
		"running":  h.running,
	}
}

func (h *HeartbeatDaemon) GetRecords() []HeartbeatRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]HeartbeatRecord, len(h.records))
	copy(out, h.records)
	return out
}
