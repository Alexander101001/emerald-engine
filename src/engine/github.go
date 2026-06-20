package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

type GHRepo struct {
	FullName    string `json:"full_name"`
	CloneURL    string `json:"clone_url"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
	Language    string `json:"language"`
	Topics      []string `json:"topics"`
}

type ghSearchResponse struct {
	Items []struct {
		FullName    string   `json:"full_name"`
		CloneURL    string   `json:"clone_url"`
		Description string   `json:"description"`
		Stars       int      `json:"stargazers_count"`
		Language    string   `json:"language"`
		Topics      []string `json:"topics"`
	} `json:"items"`
}

func searchGitHubSkills(keyword string, limit int) []GHRepo {
	token := vaultGet("GITHUB_TOKEN", "")
	if token == "" {
		fmt.Printf("[GH] No GITHUB_TOKEN in vault — skill search disabled\n")
		return nil
	}

	query := fmt.Sprintf("%s+topic:automation", keyword)
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=%d", query, limit)
	if limit <= 0 || limit > 50 {
		limit = 3
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[GH] search failed for %s: %v\n", keyword, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "[GH] API %d for %s: %s\n", resp.StatusCode, keyword, string(body))
		return nil
	}

	var result ghSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "[GH] decode failed: %v\n", err)
		return nil
	}

	repos := make([]GHRepo, 0, len(result.Items))
	for _, item := range result.Items {
		repos = append(repos, GHRepo{
			FullName:    item.FullName,
			CloneURL:    item.CloneURL,
			Description: item.Description,
			Stars:       item.Stars,
			Language:    item.Language,
			Topics:      item.Topics,
		})
	}

	sort.Slice(repos, func(i, j int) bool { return repos[i].Stars > repos[j].Stars })

	if len(repos) > 0 {
		fmt.Printf("[GH] Found %d automation repos for %s (top: %s ⭐%d)\n",
			len(repos), keyword, repos[0].FullName, repos[0].Stars)
	}

	return repos
}

func findSkillReposAcrossNiches() map[string][]GHRepo {
	results := make(map[string][]GHRepo)
	for _, niche := range niches {
		repos := searchGitHubSkills(niche.Keyword, 3)
		if len(repos) > 0 {
			results[niche.Keyword] = repos
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("[GH] Mapped skills for %d/%d niches\n", len(results), len(niches))
	return results
}

func compileCompositeDockerfile(skills []GHRepo, baseImage string) string {
	if baseImage == "" {
		baseImage = "golang:1.26"
	}

	docker := fmt.Sprintf("FROM %s\n", baseImage)
	docker += "RUN apt-get update && apt-get install -y git ca-certificates supervisor && rm -rf /var/lib/apt/lists/*\n"

	for i, skill := range skills {
		docker += fmt.Sprintf("RUN git clone --depth 1 %s /opt/skill_%d 2>/dev/null || true\n", skill.CloneURL, i)
	}

	docker += `COPY . /app
WORKDIR /app

RUN mkdir -p /var/log/supervisor

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 7860
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
`
	return docker
}

func generateCompositeSupervisord() string {
	return `[supervisord]
nodaemon=true
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid

[program:child]
command=/app/child
autorestart=true
stderr_logfile=/var/log/supervisor/child.err.log
stdout_logfile=/var/log/supervisor/child.out.log
`
}
