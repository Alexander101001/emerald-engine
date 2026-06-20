package main

import (
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

type ScrapedScript struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Language    string   `json:"language"`
	Stars       int      `json:"stars"`
	Description string   `json:"description"`
	Content     string   `json:"-"`
	Sanitized   bool     `json:"sanitized"`
	Bundled     bool     `json:"bundled"`
	Categories  []string `json:"categories"`
}

type ScraperAgent struct {
	mu           sync.Mutex
	scripts      []ScrapedScript
	httpClient   *http.Client
	totalScraped int
	totalBundled int
	totalSkipped int
	scrapeDir    string
}

var scraperAgent *ScraperAgent

func initScraperAgent() *ScraperAgent {
	sa := &ScraperAgent{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		scrapeDir:  ".scraped_scripts",
	}
	os.MkdirAll(sa.scrapeDir, 0755)
	go sa.scrapeLoop()
	scraperAgent = sa
	return sa
}

func (sa *ScraperAgent) scrapeLoop() {
	time.Sleep(2 * time.Minute)
	sa.scrapeRound()
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		sa.scrapeRound()
	}
}

func (sa *ScraperAgent) scrapeRound() {
	queries := []struct {
		query string
		cat   string
	}{
		{"utility+script+stars:>100", "utility"},
		{"dockerfile+automation+stars:>50", "docker"},
		{"bash+tool+stars:>50", "bash"},
		{"python+microservice+stars:>50", "microservice"},
		{"javascript+webhook+stars:>50", "webhook"},
		{"go+cli+tool+stars:>50", "cli"},
		{"monitoring+dashboard+stars:>50", "monitoring"},
		{"deployment+script+stars:>50", "deployment"},
	}

	for _, q := range queries {
		sa.searchAndProcess(q.query, q.cat)
		time.Sleep(2 * time.Second)
	}
}

func (sa *ScraperAgent) searchAndProcess(query, category string) {
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&per_page=5", query)
	req, _ := http.NewRequest("GET", url, nil)
	tok := tmGetToken("github")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := sa.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Items []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Language    string `json:"language"`
			Stargazers  int    `json:"stargazers_count"`
			CloneURL    string `json:"clone_url"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}

	for _, item := range result.Items {
		sa.processItem(item.Name, item.FullName, item.Description, item.Language, item.Stargazers, category)
	}
}

func (sa *ScraperAgent) processItem(name, fullName, desc, lang string, stars int, category string) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	for _, s := range sa.scripts {
		if s.URL == fullName {
			return
		}
	}

	if stars < 30 {
		sa.totalSkipped++
		return
	}

	script := ScrapedScript{
		Name:        name,
		URL:         fullName,
		Language:    lang,
		Stars:       stars,
		Description: desc,
		Categories:  []string{category},
		Sanitized:   false,
		Bundled:     false,
	}

	if sa.sanitize(&script) {
		script.Sanitized = true
		sa.scripts = append(sa.scripts, script)
		sa.totalScraped++
		sa.saveScript(script)
	} else {
		sa.totalSkipped++
	}
}

func (sa *ScraperAgent) sanitize(s *ScrapedScript) bool {
	rejectPatterns := []string{
		"secret", "password", "token", "api_key", "credential",
		".env", "config.yml", "config.json",
		"npm install", "pip install", "bundle install",
		"gem install", "cargo install",
	}

	nameLower := strings.ToLower(s.Name)
	for _, p := range rejectPatterns {
		if strings.Contains(nameLower, p) {
			return false
		}
	}

	descLower := strings.ToLower(s.Description)
	for _, p := range rejectPatterns[:4] {
		if strings.Contains(descLower, p) {
			return false
		}
	}

	return true
}

func (sa *ScraperAgent) saveScript(s ScrapedScript) {
	path := filepath.Join(sa.scrapeDir, s.Name+".json")
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (sa *ScraperAgent) BundleForDocker(skills []ScrapedScript) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Bundled scraped scripts\n")
	sb.WriteString("RUN mkdir -p /opt/scripts\n\n")

	for _, s := range skills {
		// Add appropriate run command based on language
		switch strings.ToLower(s.Language) {
		case "python":
			sb.WriteString(fmt.Sprintf("COPY --from=scripts /opt/%s /opt/scripts/%s\n", s.Name, s.Name))
			sb.WriteString(fmt.Sprintf("RUN chmod +x /opt/scripts/%s/main.py 2>/dev/null || true\n", s.Name))
		case "go":
			sb.WriteString(fmt.Sprintf("COPY --from=scripts /opt/%s /opt/scripts/%s\n", s.Name, s.Name))
			sb.WriteString(fmt.Sprintf("RUN chmod +x /opt/scripts/%s 2>/dev/null || true\n", s.Name))
		case "javascript", "typescript":
			sb.WriteString(fmt.Sprintf("COPY --from=scripts /opt/%s /opt/scripts/%s\n", s.Name, s.Name))
		case "bash", "shell":
			sb.WriteString(fmt.Sprintf("COPY --from=scripts /opt/%s /opt/scripts/%s.sh\n", s.Name, s.Name))
			sb.WriteString(fmt.Sprintf("RUN chmod +x /opt/scripts/%s.sh\n", s.Name))
		default:
			sb.WriteString(fmt.Sprintf("COPY --from=scripts /opt/%s /opt/scripts/%s\n", s.Name, s.Name))
		}
		sa.totalBundled++
	}

	return sb.String()
}

func (sa *ScraperAgent) Stats() map[string]interface{} {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return map[string]interface{}{
		"scraped": sa.totalScraped,
		"bundled": sa.totalBundled,
		"skipped": sa.totalSkipped,
		"cached":  len(sa.scripts),
	}
}
