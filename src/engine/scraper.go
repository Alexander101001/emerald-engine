package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type SkillCategory string

const (
	CatAutomation   SkillCategory = "automation"
	CatTraffic      SkillCategory = "traffic"
	CatAnalytics    SkillCategory = "analytics"
	CatMonetization SkillCategory = "monetization"
	CatAPI          SkillCategory = "api"
	CatBot          SkillCategory = "bot"
	CatScraper      SkillCategory = "scraper"
	CatSEO          SkillCategory = "seo"
	CatAds          SkillCategory = "ads"
	CatAffiliate    SkillCategory = "affiliate"
	CatCrypto       SkillCategory = "crypto"
	CatTrading      SkillCategory = "trading"
	CatDashboard    SkillCategory = "dashboard"
	CatTemplate     SkillCategory = "template"
	CatMicroservice SkillCategory = "microservice"
)

type DiscoveredSkill struct {
	Name        string         `json:"name"`
	FullName    string         `json:"full_name"`
	URL         string         `json:"url"`
	CloneURL    string         `json:"clone_url"`
	Description string         `json:"description"`
	Stars       int            `json:"stars"`
	Language    string         `json:"language"`
	Source      string         `json:"source"` // github or huggingface
	Category    SkillCategory  `json:"category"`
	Topics      []string       `json:"topics"`
	Score       float64        `json:"score"`
	HasDocker   bool           `json:"has_docker"`
	HasAPI      bool           `json:"has_api"`
}

type SkillRegistry struct {
	mu     sync.RWMutex
	Skills map[string][]DiscoveredSkill
	ScrapedAt time.Time
}

func (sr *SkillRegistry) clearCache() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.Skills = make(map[string][]DiscoveredSkill)
	sr.ScrapedAt = time.Time{}
}

var skillRegistry = &SkillRegistry{
	Skills: make(map[string][]DiscoveredSkill),
}

var nicheTopics = map[string][]string{
	"saas-marketing":    {"saas", "marketing-automation", "crm", "email-marketing", "analytics", "dashboard"},
	"health-wellness":   {"health", "fitness", "wellness", "tracker", "nutrition", "workout"},
	"personal-finance":  {"finance", "budgeting", "investment", "accounting", "banking", "expense"},
	"digital-products":  {"digital-products", "ecommerce", "download", "membership", "delivery"},
	"online-education":  {"education", "elearning", "course", "lms", "quiz", "certification"},
	"ecommerce":         {"ecommerce", "shop", "cart", "inventory", "payment", "store"},
	"ai-tools":          {"artificial-intelligence", "machine-learning", "llm", "nlp", "vision", "chatbot"},
	"crypto-web3":       {"cryptocurrency", "blockchain", "web3", "defi", "nft", "wallet"},
	"real-estate":       {"real-estate", "property", "mortgage", "rental", "housing", "market"},
	"travel":            {"travel", "hotel", "flight", "booking", "trip", "vacation"},
	"fitness-nutrition": {"fitness", "nutrition", "exercise", "diet", "calorie", "training"},
	"remote-work":       {"remote-work", "productivity", "collaboration", "time-tracking", "virtual"},
	"content-creation":  {"content", "writing", "video", "audio", "editing", "publishing"},
	"cybersecurity":     {"cybersecurity", "security", "privacy", "encryption", "vpn", "scanner"},
	"self-improvement":  {"self-improvement", "habits", "meditation", "journal", "goal-tracking"},
}

var searchTopics = []string{
	"automation", "api", "bot", "scraper", "microservice",
	"dashboard", "template", "tool", "cli", "sdk",
}

func classifySkill(name, description string, topics []string) SkillCategory {
	combined := strings.ToLower(name + " " + description + " " + strings.Join(topics, " "))

	cats := []struct {
		cat  SkillCategory
		keywords []string
	}{
		{CatAutomation, []string{"automation", "workflow", "pipeline", "ci/cd", "deploy"}},
		{CatTraffic, []string{"traffic", "proxy", "load-balancer", "reverse-proxy", "cdn", "gateway"}},
		{CatAnalytics, []string{"analytics", "metrics", "monitoring", "dashboard", "statistics", "tracking"}},
		{CatMonetization, []string{"monetization", "ad", "adsense", "revenue", "payment", "billing", "stripe"}},
		{CatAPI, []string{"api", "rest", "graphql", "grpc", "endpoint", "backend"}},
		{CatBot, []string{"bot", "telegram", "discord", "slack", "chatbot", "messaging"}},
		{CatScraper, []string{"scraper", "crawler", "spider", "extract", "harvest"}},
		{CatSEO, []string{"seo", "search-engine", "ranking", "keyword", "sitemap"}},
		{CatAds, []string{"ad", "advertisement", "adserver", "adnetwork"}},
		{CatAffiliate, []string{"affiliate", "referral", "commission", "partner"}},
		{CatCrypto, []string{"crypto", "blockchain", "bitcoin", "ethereum", "wallet", "defi"}},
		{CatTrading, []string{"trading", "exchange", "market", "stock", "signal", "binance"}},
		{CatDashboard, []string{"dashboard", "ui", "visualization", "chart", "panel"}},
		{CatTemplate, []string{"template", "boilerplate", "starter", "scaffold"}},
		{CatMicroservice, []string{"microservice", "service", "container", "docker", "kubernetes"}},
	}

	bestCat := CatAutomation
	bestScore := 0
	for _, c := range cats {
		score := 0
		for _, kw := range c.keywords {
			if strings.Contains(combined, kw) {
				score += 10
			}
		}
		if score > bestScore {
			bestScore = score
			bestCat = c.cat
		}
	}
	return bestCat
}

func scoreSkill(s DiscoveredSkill) float64 {
	score := float64(s.Stars) * 0.5
	if s.HasDocker {
		score += 15
	}
	if s.HasAPI {
		score += 10
	}
	if s.Language == "Go" || s.Language == "Python" || s.Language == "TypeScript" {
		score += 5
	}
	if s.Description != "" {
		score += 3
	}
	return score
}

func searchGitHubMulti(nicheKeyword string) []DiscoveredSkill {
	token := vaultGet("GITHUB_TOKEN", "")
	if token == "" {
		return nil
	}

	nicheTopicsList := nicheTopics[nicheKeyword]
	if nicheTopicsList == nil {
		nicheTopicsList = []string{nicheKeyword}
	}

	seen := make(map[string]bool)
	var allSkills []DiscoveredSkill

	for _, topic := range nicheTopicsList {
		for _, searchTopic := range searchTopics {
			query := fmt.Sprintf("%s+topic:%s", topic, searchTopic)
			url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=5", query)

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "token "+token)
			req.Header.Set("Accept", "application/vnd.github.v3+json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			var result struct {
				Items []struct {
					FullName    string   `json:"full_name"`
					CloneURL    string   `json:"clone_url"`
					Description string   `json:"description"`
					Stars       int      `json:"stargazers_count"`
					Language    string   `json:"language"`
					Topics      []string `json:"topics"`
					HTMLURL     string   `json:"html_url"`
				} `json:"items"`
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err := json.Unmarshal(body, &result); err != nil {
				continue
			}

			for _, item := range result.Items {
				if seen[item.FullName] {
					continue
				}
				seen[item.FullName] = true

				hasDocker := false
				for _, t := range item.Topics {
					if t == "docker" || t == "container" {
						hasDocker = true
						break
					}
				}

				skill := DiscoveredSkill{
					Name:        item.FullName[strings.Index(item.FullName, "/")+1:],
					FullName:    item.FullName,
					URL:         item.HTMLURL,
					CloneURL:    item.CloneURL,
					Description: item.Description,
					Stars:       item.Stars,
					Language:    item.Language,
					Source:      "github",
					Topics:      item.Topics,
					HasDocker:   hasDocker,
					HasAPI:      strings.Contains(strings.ToLower(item.Description), "api"),
				}
				skill.Category = classifySkill(skill.Name, item.Description, item.Topics)
				skill.Score = scoreSkill(skill)
				allSkills = append(allSkills, skill)
			}

			time.Sleep(200 * time.Millisecond)
		}
	}

	sort.Slice(allSkills, func(i, j int) bool {
		return allSkills[i].Score > allSkills[j].Score
	})

	if len(allSkills) > 0 {
		fmt.Printf("[SCRAPER] GitHub: %d unique skills for %s (top: %s ⭐%d %.1fpt)\n",
			len(allSkills), nicheKeyword, allSkills[0].FullName, allSkills[0].Stars, allSkills[0].Score)
	}

	return allSkills
}

func searchHFSpaces(nicheKeyword string) []DiscoveredSkill {
	token := vaultGet("HF_TOKEN", "")
	if token == "" {
		return nil
	}

	query := nicheKeyword
	url := fmt.Sprintf("https://huggingface.co/api/spaces?search=%s&sort=likes&direction=-1&limit=10", query)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SCRAPER] HF Spaces search failed: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	var spaces []struct {
		Name        string   `json:"name"`
		Namespace   string   `json:"namespace"`
		Sdk         string   `json:"sdk"`
		Likes       int      `json:"likes"`
		Description string   `json:"description"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &spaces); err != nil {
		fmt.Fprintf(os.Stderr, "[SCRAPER] HF Spaces decode failed: %v\n", err)
		return nil
	}

	var skills []DiscoveredSkill
	seen := make(map[string]bool)

	for _, sp := range spaces {
		fullName := sp.Namespace + "/" + sp.Name
		if seen[fullName] {
			continue
		}
		seen[fullName] = true

		skill := DiscoveredSkill{
			Name:        sp.Name,
			FullName:    fullName,
			URL:         fmt.Sprintf("https://huggingface.co/spaces/%s", fullName),
			CloneURL:    fmt.Sprintf("https://huggingface.co/spaces/%s", fullName),
			Description: sp.Description,
			Stars:       sp.Likes,
			Language:    sp.Sdk,
			Source:      "huggingface",
			HasDocker:   sp.Sdk == "docker",
			HasAPI:      strings.Contains(strings.ToLower(sp.Description), "api"),
		}
		skill.Category = classifySkill(sp.Name, sp.Description, nil)
		skill.Score = float64(sp.Likes)*0.3 + 10
		skills = append(skills, skill)
	}

	if len(skills) > 0 {
		fmt.Printf("[SCRAPER] HF: %d spaces for %s\n", len(skills), nicheKeyword)
	}

	return skills
}

func discoverSkills(nicheKeyword string, maxPerSource int) []DiscoveredSkill {
	skillRegistry.mu.Lock()
	defer skillRegistry.mu.Unlock()

	if cached, ok := skillRegistry.Skills[nicheKeyword]; ok && time.Since(skillRegistry.ScrapedAt) < 1*time.Hour {
		return cached
	}

	ghSkills := searchGitHubMulti(nicheKeyword)
	hfSkills := searchHFSpaces(nicheKeyword)

	all := append(ghSkills, hfSkills...)

	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })

	if len(all) > maxPerSource && maxPerSource > 0 {
		all = all[:maxPerSource]
	}

	skillRegistry.Skills[nicheKeyword] = all
	skillRegistry.ScrapedAt = time.Now()

	fmt.Printf("[SCRAPER] Discovered %d total skills for %s\n", len(all), nicheKeyword)
	return all
}

func getTopSkillsByCategory(skills []DiscoveredSkill) map[SkillCategory][]DiscoveredSkill {
	grouped := make(map[SkillCategory][]DiscoveredSkill)
	for _, s := range skills {
		if len(grouped[s.Category]) < 3 {
			grouped[s.Category] = append(grouped[s.Category], s)
		}
	}
	return grouped
}

func getBestSkillsForFusion(nicheKeyword string, maxComponents int) []DiscoveredSkill {
	skills := discoverSkills(nicheKeyword, 30)
	if len(skills) == 0 {
		return nil
	}

	byCategory := getTopSkillsByCategory(skills)

	categories := []SkillCategory{
		CatAutomation, CatAPI, CatBot, CatScraper,
		CatAnalytics, CatMonetization, CatTraffic,
		CatDashboard, CatSEO, CatAds, CatAffiliate,
	}

	var fused []DiscoveredSkill
	seen := make(map[string]bool)

	for _, cat := range categories {
		catSkills := byCategory[cat]
		for _, s := range catSkills {
			if len(fused) >= maxComponents {
				break
			}
			if seen[s.FullName] {
				continue
			}
			seen[s.FullName] = true
			fused = append(fused, s)
		}
		if len(fused) >= maxComponents {
			break
		}
	}

	if len(fused) < maxComponents {
		for _, s := range skills {
			if len(fused) >= maxComponents {
				break
			}
			if seen[s.FullName] {
				continue
			}
			seen[s.FullName] = true
			fused = append(fused, s)
		}
	}

	fmt.Printf("[SCRAPER] Fusion selected %d/%d components for %s\n", len(fused), maxComponents, nicheKeyword)
	return fused
}

func scraperStats() map[string]interface{} {
	skillRegistry.mu.RLock()
	defer skillRegistry.mu.RUnlock()

	total := 0
	nicheCount := len(skillRegistry.Skills)
	for _, skills := range skillRegistry.Skills {
		total += len(skills)
	}

	return map[string]interface{}{
		"cached_niches": nicheCount,
		"total_skills":  total,
		"scraped_at":    skillRegistry.ScrapedAt.Format(time.RFC3339),
		"age_minutes":   int(time.Since(skillRegistry.ScrapedAt).Minutes()),
	}
}
