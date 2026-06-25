package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ── Task Scheduler (Paradigm 1: 5-Channel Parallel) ──────────────────────

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

type Task struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Channel   int        `json:"channel"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	Result    string     `json:"result,omitempty"`
}

type TaskScheduler struct {
	mu       sync.RWMutex
	tasks    map[string]*Task
	channels [5]chan *Task
	nextID   int
}

func NewTaskScheduler() *TaskScheduler {
	ts := &TaskScheduler{
		tasks:  make(map[string]*Task),
		nextID: 0,
	}
	for i := 0; i < 5; i++ {
		ts.channels[i] = make(chan *Task, 100)
		go ts.workerLoop(i, ts.channels[i])
	}
	return ts
}

func (ts *TaskScheduler) workerLoop(channel int, tasks <-chan *Task) {
	for t := range tasks {
		t.Status = TaskRunning
		time.Sleep(time.Duration(50+channel*10) * time.Millisecond)
		t.Status = TaskCompleted
		t.Result = fmt.Sprintf("processed by channel %d at %s", channel, time.Now().Format(time.RFC3339))
	}
}

func (ts *TaskScheduler) Submit(name string) *Task {
	ts.mu.Lock()
	ts.nextID++
	id := fmt.Sprintf("task-%d", ts.nextID)
	channel := ts.nextID % 5
	t := &Task{
		ID:        id,
		Name:      name,
		Channel:   channel,
		Status:    TaskPending,
		CreatedAt: time.Now(),
	}
	ts.tasks[id] = t
	ts.mu.Unlock()
	ts.channels[channel] <- t
	return t
}

func (ts *TaskScheduler) Get(id string) *Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.tasks[id]
}

func (ts *TaskScheduler) List() []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	out := make([]*Task, 0, len(ts.tasks))
	for _, t := range ts.tasks {
		out = append(out, t)
	}
	return out
}

// ── State Synchronization ────────────────────────────────────────────────

type SyncState struct {
	mu     sync.RWMutex
	states map[string]interface{}
}

var globalState = &SyncState{states: make(map[string]interface{})}

func (s *SyncState) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[key] = val
}

func (s *SyncState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.states[key]
	return v, ok
}

func (s *SyncState) Snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]interface{}, len(s.states))
	for k, v := range s.states {
		cp[k] = v
	}
	return cp
}

// ── Platform Scanner ─────────────────────────────────────────────────────

type TargetPlatform struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

var platforms = []TargetPlatform{
	{ID: "github", URL: "https://api.github.com"},
	{ID: "huggingface", URL: "https://huggingface.co/api/models"},
	{ID: "gitlab", URL: "https://gitlab.com/api/v4/projects"},
	{ID: "bitbucket", URL: "https://api.bitbucket.org/2.0/repositories"},
	{ID: "digitalocean", URL: "https://api.digitalocean.com/v2/account"},
	{ID: "linode", URL: "https://api.linode.com/v4/account"},
	{ID: "vultr", URL: "https://api.vultr.com/v2/account"},
	{ID: "heroku", URL: "https://api.heroku.com/apps"},
	{ID: "railway", URL: "https://backboard.railway.app/graphql"},
	{ID: "render", URL: "https://api.render.com/v1/services"},
	{ID: "netlify", URL: "https://api.netlify.com/api/v1/sites"},
	{ID: "vercel", URL: "https://api.vercel.com/v9/projects"},
	{ID: "cloudflare", URL: "https://api.cloudflare.com/client/v4/zones"},
	{ID: "flyio", URL: "https://api.fly.io/v1/apps"},
	{ID: "koyeb", URL: "https://app.koyeb.com/api/v1/apps"},
	{ID: "deno_deploy", URL: "https://api.deno.com/v1/projects"},
	{ID: "replit", URL: "https://replit.com/api/v1/user"},
	{ID: "cyclic", URL: "https://api.cyclic.sh/v1/apps"},
	{ID: "adaptable", URL: "https://api.adaptable.io/v1/apps"},
	{ID: "pythonanywhere", URL: "https://www.pythonanywhere.com/api/v0/user"},
	{ID: "scaleway", URL: "https://api.scaleway.com/v1/instances"},
	{ID: "civo", URL: "https://api.civo.com/v2/instances"},
	{ID: "hetzner", URL: "https://api.hetzner.cloud/v1/servers"},
	{ID: "upcloud", URL: "https://api.upcloud.com/1.3/account"},
	{ID: "ovhcloud", URL: "https://api.ovh.com/1.0/me"},
	{ID: "clever_cloud", URL: "https://api.clever-cloud.com/v2/products"},
	{ID: "scalingo", URL: "https://api.scalingo.com/v1/apps"},
	{ID: "glitch", URL: "https://api.glitch.com/v1/projects"},
	{ID: "deta", URL: "https://api.deta.sh/v1/projects"},
	{ID: "mogenius", URL: "https://api.mogenius.com/v1/projects"},
	{ID: "alwaysdata", URL: "https://api.alwaysdata.com/v1/account"},
	{ID: "exoscale", URL: "https://api.exoscale.com/v1/compute"},
	{ID: "ionos", URL: "https://api.ionos.com/cloudapi/v5/datacenters"},
	{ID: "pulumi", URL: "https://api.pulumi.com/api/user"},
	{ID: "terraform_cloud", URL: "https://app.terraform.io/api/v2/account/details"},
	{ID: "supabase", URL: "https://api.supabase.com/v1/projects"},
	{ID: "neon", URL: "https://console.neon.tech/api/v2/projects"},
	{ID: "planetscale", URL: "https://api.planetscale.com/v1/organizations"},
	{ID: "mongodb_atlas", URL: "https://cloud.mongodb.com/api/atlas/v1.0/groups"},
	{ID: "redis_cloud", URL: "https://api.redislabs.com/v1/subscriptions"},
	{ID: "cloudamqp", URL: "https://customer.cloudamqp.com/api/instances"},
	{ID: "confluent_cloud", URL: "https://api.confluent.cloud/org/v2/organizations"},
	{ID: "ably", URL: "https://api.ably.io/v1/apps"},
	{ID: "sentry", URL: "https://sentry.io/api/0/projects"},
	{ID: "datadog", URL: "https://api.datadoghq.com/api/v1/validate"},
	{ID: "grafana_cloud", URL: "https://grafana.com/api/instances"},
	{ID: "betterstack", URL: "https://uptime.betterstack.com/api/v2/monitors"},
	{ID: "checkly", URL: "https://api.checklyhq.com/v1/checks"},
	{ID: "algolia", URL: "https://api.algolia.com/1/indexes"},
	{ID: "logz_io", URL: "https://api.logz.io/v1/account"},
}

var scheduler = NewTaskScheduler()

// ── HTTP Handlers ────────────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleScan(w http.ResponseWriter, r *http.Request) {
	maxConcurrent := 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	results := make(chan string, len(platforms))
	client := &http.Client{Timeout: 6 * time.Second}

	for _, p := range platforms {
		wg.Add(1)
		sem <- struct{}{}
		go func(pl TargetPlatform) {
			defer func() { <-sem }()
			defer wg.Done()
			req, err := http.NewRequest("GET", pl.URL, nil)
			if err != nil {
				results <- fmt.Sprintf("%s:Error:%v", pl.ID, err)
				return
			}
			req.Header.Set("User-Agent", "EmeraldEngine/BackendAgent/1.0")
			resp, err := client.Do(req)
			if err != nil {
				results <- fmt.Sprintf("%s:Offline", pl.ID)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 201 {
				results <- fmt.Sprintf("%s:Active", pl.ID)
			} else {
				results <- fmt.Sprintf("%s:Unreachable:%d", pl.ID, resp.StatusCode)
			}
		}(p)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var lines []string
	for r := range results {
		lines = append(lines, r)
	}
	jsonResponse(w, map[string]interface{}{"scanned": len(platforms), "results": lines})
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResponse(w, scheduler.List())
	case "POST":
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		t := scheduler.Submit(body.Name)
		jsonResponse(w, t)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResponse(w, globalState.Snapshot())
	case "POST":
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		for k, v := range body {
			globalState.Set(k, v)
		}
		jsonResponse(w, map[string]string{"status": "synced"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"status":      "operational",
		"tasks":       len(scheduler.List()),
		"platforms":   len(platforms),
		"role":        "backend_performance",
		"opcode_spec": []string{"5_channel_parallel", "task_scheduler", "state_sync"},
	})
}

func main() {
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/scan", handleScan)
	mux.HandleFunc("/api/tasks", handleTasks)
	mux.HandleFunc("/api/state", handleState)

	log.Printf("[backend_performance] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
