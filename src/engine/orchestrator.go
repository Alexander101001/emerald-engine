package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type ChildSpace struct {
	Name       string    `json:"name"`
	Niche      string    `json:"niche"`
	Namespace  string    `json:"namespace"`
	Status     string    `json:"status"` // deploying, running, crashed
	URL        string    `json:"url"`
	DeployedAt time.Time `json:"deployed_at"`
	LastHealth time.Time `json:"last_health"`
	ErrorCount int       `json:"error_count"`
}

type Orchestrator struct {
	mu         sync.Mutex
	Token      string
	Username   string
	Children   map[string]*ChildSpace
	DeployLock sync.Mutex
	Stop       chan struct{}
}

var orchestrator *Orchestrator

func initOrchestrator() *Orchestrator {
	o := &Orchestrator{
		Token:    vaultGet("HF_TOKEN", ""),
		Username: vaultGet("HF_USER", "AlexanderGreater90"),
		Children: make(map[string]*ChildSpace),
		Stop:     make(chan struct{}),
	}
	orchestrator = o

	loadExistingChildren(o)

	if o.Token != "" && o.Username != "" {
		fmt.Printf("[ORCH] Factory initialized | User: %s | Token: %d chars\n", o.Username, len(o.Token))
		go o.runDeployLoop()
		go o.runHealthLoop()
	} else {
		fmt.Printf("[ORCH] HF token not configured — factory mode disabled\n")
	}
	return o
}

func loadExistingChildren(o *Orchestrator) {
	data, err := os.ReadFile("emerald_children.json")
	if err != nil {
		return
	}
	var children []ChildSpace
	json.Unmarshal(data, &children)
	for i := range children {
		o.Children[children[i].Niche] = &children[i]
	}
	fmt.Printf("[ORCH] Loaded %d existing children\n", len(children))
}

func (o *Orchestrator) saveChildren() {
	o.mu.Lock()
	defer o.mu.Unlock()
	var list []ChildSpace
	for _, c := range o.Children {
		list = append(list, *c)
	}
	data, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile("emerald_children.json", data, 0644)
}

func (o *Orchestrator) getDeployedNiches() map[string]bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	deployed := make(map[string]bool)
	for _, c := range o.Children {
		deployed[c.Niche] = true
	}
	return deployed
}

func (o *Orchestrator) getHealthyCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	count := 0
	for _, c := range o.Children {
		if c.Status == "running" {
			count++
		}
	}
	return count
}

func (o *Orchestrator) runDeployLoop() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	o.deployNextNiche()

	for {
		select {
		case <-o.Stop:
			return
		case <-ticker.C:
			o.deployNextNiche()
		}
	}
}

func (o *Orchestrator) deployNextNiche() {
	if !o.DeployLock.TryLock() {
		fmt.Printf("[ORCH] Deploy already in progress\n")
		return
	}
	defer o.DeployLock.Unlock()

	deployed := o.getDeployedNiches()
	for _, niche := range niches {
		if deployed[niche.Keyword] {
			continue
		}
		fmt.Printf("[ORCH] Deploying child for niche: %s %s\n", niche.Emoji, niche.Name)
		err := o.deployChild(niche)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ORCH] Deploy failed for %s: %v\n", niche.Name, err)
			time.Sleep(30 * time.Minute)
			return
		}
		fmt.Printf("[ORCH] Deployed %s — next deploy in 6 hours\n", niche.Name)
		return
	}
	fmt.Printf("[ORCH] All %d niches deployed\n", len(niches))
}

func (o *Orchestrator) deployChild(niche Niche) error {
	spaceName := "emerald-" + niche.Keyword
	fullName := o.Username + "/" + spaceName

	exists, err := o.spaceExists(spaceName)
	if err != nil {
		return fmt.Errorf("check exists: %w", err)
	}
	if !exists {
		err = o.createSpace(spaceName, niche)
		if err != nil {
			return fmt.Errorf("create space: %w", err)
		}
		time.Sleep(30 * time.Second)
	}

	err = o.setSpaceSecrets(spaceName)
	if err != nil {
		return fmt.Errorf("set secrets: %w", err)
	}

	err = o.uploadChildCode(spaceName, niche)
	if err != nil {
		return fmt.Errorf("upload code: %w", err)
	}

	err = o.restartSpace(spaceName)
	if err != nil {
		return fmt.Errorf("restart: %w", err)
	}

	child := &ChildSpace{
		Name:       spaceName,
		Niche:      niche.Keyword,
		Namespace:  o.Username,
		Status:     "deploying",
		URL:        fmt.Sprintf("https://%s.hf.space", fullName),
		DeployedAt: time.Now(),
	}

	o.mu.Lock()
	o.Children[niche.Keyword] = child
	o.mu.Unlock()
	o.saveChildren()

	fmt.Printf("[ORCH] Child %s deployed at %s\n", spaceName, child.URL)
	return nil
}

// ─── HF Hub API ───

func (o *Orchestrator) hfRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	url := "https://huggingface.co" + path
	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Authorization", "Bearer "+o.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("hf api %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (o *Orchestrator) spaceExists(name string) (bool, error) {
	_, err := o.hfRequest("GET", "/api/spaces/"+o.Username+"/"+name, nil)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (o *Orchestrator) createSpace(name string, niche Niche) error {
	body := map[string]interface{}{
		"type":      "space",
		"name":      name,
		"namespace": o.Username,
		"private":   false,
		"sdk":       "docker",
	}
	_, err := o.hfRequest("POST", "/api/repos/create", body)
	if err != nil {
		return err
	}

	hw := nicheHardware(niche)
	settings := map[string]interface{}{
		"hardware": hw,
		"secrets":  []map[string]string{},
	}
	_, err = o.hfRequest("PUT", "/api/spaces/"+o.Username+"/"+name, settings)
	return err
}

func nicheHardware(niche Niche) string {
	highCPU := map[string]bool{
		"ai-tools": true, "crypto-web3": true, "saas-marketing": true,
	}
	if highCPU[niche.Keyword] {
		return "cpu-basic"
	}
	return "cpu-basic"
}

func (o *Orchestrator) setSpaceSecrets(name string) error {
	secrets := map[string]string{
		"ADSENSE_CLIENT_ID":     vaultGet("ADSENSE_CLIENT_ID", ""),
		"AMAZON_ASSOCIATES_TAG": vaultGet("AMAZON_ASSOCIATES_TAG", ""),
		"STRIPE_API_KEY":        vaultGet("STRIPE_API_KEY", ""),
		"GA_ID":                 vaultGet("GA_ID", ""),
		"CLOUDFLARE_API_TOKEN":  vaultGet("CLOUDFLARE_API_TOKEN", ""),
	}

	for key, val := range secrets {
		if val == "" {
			continue
		}
		secretBody := map[string]string{"key": key, "value": val}
		_, err := o.hfRequest("POST", "/api/spaces/"+o.Username+"/"+name+"/secrets", secretBody)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ORCH] Failed to set secret %s: %v\n", key, err)
		} else {
			fmt.Printf("[ORCH] Secret set: %s\n", key)
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (o *Orchestrator) restartSpace(name string) error {
	_, err := o.hfRequest("POST", "/api/spaces/"+o.Username+"/"+name+"/restart", nil)
	return err
}

func (o *Orchestrator) uploadChildCode(spaceName string, niche Niche) error {
	tmpDir, err := os.MkdirTemp("", "child-"+niche.Keyword)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	code := generateChildCode(niche)
	docker := generateChildDockerfile(niche)
	readme := generateChildReadme(niche)

	os.WriteFile(filepath.Join(tmpDir, "child.go"), []byte(code), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(docker), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(readme), 0644)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module child\n\ngo 1.26.4\n"), 0644)

	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	initCmd.Run()

	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = tmpDir
	addCmd.Run()

	commitCmd := exec.Command("git", "commit", "-m", "initial deploy: "+niche.Name)
	commitCmd.Dir = tmpDir
	commitCmd.Run()

	remoteURL := fmt.Sprintf("https://%s:%s@huggingface.co/spaces/%s/%s", o.Username, o.Token, o.Username, spaceName)
	remoteCmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	remoteCmd.Dir = tmpDir
	remoteCmd.Run()

	pushCmd := exec.Command("git", "push", "-u", "origin", "main", "--force")
	pushCmd.Dir = tmpDir
	out, err := pushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("push failed: %s: %w", string(out), err)
	}
	fmt.Printf("[ORCH] Code pushed to %s (%d bytes)\n", spaceName, len(code)+len(docker))
	return nil
}

// ─── Health Monitoring ───

func (o *Orchestrator) runHealthLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-o.Stop:
			return
		case <-ticker.C:
			o.checkAllHealth()
		}
	}
}

func (o *Orchestrator) checkAllHealth() {
	o.mu.Lock()
	children := make([]*ChildSpace, 0, len(o.Children))
	for _, c := range o.Children {
		children = append(children, c)
	}
	o.mu.Unlock()

	for _, child := range children {
		healthy := o.pingChild(child)
		o.mu.Lock()
		if healthy {
			child.Status = "running"
			child.LastHealth = time.Now()
			child.ErrorCount = 0
		} else {
			child.ErrorCount++
			if child.ErrorCount > 3 {
				child.Status = "crashed"
				fmt.Fprintf(os.Stderr, "[ORCH] Child %s crashed (%d errors)\n", child.Name, child.ErrorCount)
			} else {
				child.Status = "degraded"
			}
		}
		o.mu.Unlock()
		time.Sleep(5 * time.Second)
	}
	o.saveChildren()

	healthy := 0
	total := len(children)
	o.mu.Lock()
	for _, c := range o.Children {
		if c.Status == "running" {
			healthy++
		}
	}
	o.mu.Unlock()

	fmt.Printf("[ORCH] Health: %d/%d children healthy\n", healthy, total)

	if total > 0 {
		telegramSend(fmt.Sprintf(
			"<b>🏭 Factory Health</b>\nChildren: %d/%d healthy\nDeployed: %d/%d niches",
			healthy, total, len(o.Children), len(niches),
		))
	}
}

func (o *Orchestrator) pingChild(child *ChildSpace) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(child.URL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (o *Orchestrator) getChildURL(nicheKeyword string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if child, ok := o.Children[nicheKeyword]; ok {
		return child.URL
	}
	return ""
}

func orchestratorStats() map[string]interface{} {
	if orchestrator == nil {
		return map[string]interface{}{
			"status":   "disabled",
			"children": 0,
			"healthy":  0,
		}
	}
	deployed := len(orchestrator.Children)
	healthy := 0
	orchestrator.mu.Lock()
	for _, c := range orchestrator.Children {
		if c.Status == "running" {
			healthy++
		}
	}
	orchestrator.mu.Unlock()
	return map[string]interface{}{
		"status":         "active",
		"niche_count":    len(niches),
		"children":       deployed,
		"healthy":        healthy,
		"remaining":      len(niches) - deployed,
		"deploy_cycle":   "6 hours",
		"health_cycle":   "15 minutes",
	}
}

func (o *Orchestrator) hfFileUpload(spaceName, pathInRepo string, content []byte) error {
	url := fmt.Sprintf("https://huggingface.co/api/repos/%s/%s/content/%s", o.Username, spaceName, pathInRepo)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(content))
	req.Header.Set("Authorization", "Bearer "+o.Token)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
