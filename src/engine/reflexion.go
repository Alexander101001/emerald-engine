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

type VerbalMemory struct {
	GlobalCritiques []string `json:"global_critiques"`
	LearnedLessons  []string `json:"learned_lessons"`
}

type ReflexionLayer struct {
	mu              sync.Mutex
	memoryPath      string
	verbalMemory    VerbalMemory
	verifications   int
	verifiedCount   int
	textGradCount   int
	lastRefinement  time.Time
}

var reflexionLayer *ReflexionLayer

func initReflexionLayer() *ReflexionLayer {
	rl := &ReflexionLayer{
		memoryPath: "config/reflexion_memory.json",
	}
	rl.loadVerbalMemory()
	reflexionLayer = rl
	return rl
}

func (rl *ReflexionLayer) loadVerbalMemory() {
	data, err := os.ReadFile(rl.memoryPath)
	if err != nil {
		rl.verbalMemory = VerbalMemory{
			GlobalCritiques: []string{},
			LearnedLessons:  []string{},
		}
		return
	}
	var vm VerbalMemory
	if json.Unmarshal(data, &vm) == nil {
		rl.verbalMemory = vm
	} else {
		rl.verbalMemory = VerbalMemory{
			GlobalCritiques: []string{},
			LearnedLessons:  []string{},
		}
	}
}

func (rl *ReflexionLayer) saveVerbalMemory() {
	os.MkdirAll(filepath.Dir(rl.memoryPath), 0755)
	data, _ := json.MarshalIndent(rl.verbalMemory, "", "  ")
	os.WriteFile(rl.memoryPath, data, 0644)
}

func (rl *ReflexionLayer) ApplyTextGradientRefinement(corePrompt, failureFeedback string) string {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	critique := fmt.Sprintf("TextGrad Delta Correction: Injected remedy for: %s", failureFeedback)
	rl.verbalMemory.GlobalCritiques = append(rl.verbalMemory.GlobalCritiques, critique)
	rl.textGradCount++
	rl.lastRefinement = time.Now()

	rl.saveVerbalMemory()

	var b strings.Builder
	b.WriteString(corePrompt)
	b.WriteString("\n\n")
	b.WriteString("# [Reflexion-Layer] Systemic Verbal Memory and Countermeasures:\n")
	b.WriteString(fmt.Sprintf("- Avoid: %s\n", failureFeedback))
	b.WriteString("- Constraint: Ensure execution follows unified TextGrad structural boundaries.\n")

	for _, lesson := range rl.verbalMemory.LearnedLessons {
		b.WriteString(fmt.Sprintf("- Reinforce: %s\n", lesson))
	}

	return b.String()
}

func (rl *ReflexionLayer) RecordLearnedLesson(lesson string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.verbalMemory.LearnedLessons = append(rl.verbalMemory.LearnedLessons, lesson)
	if len(rl.verbalMemory.LearnedLessons) > 50 {
		rl.verbalMemory.LearnedLessons = rl.verbalMemory.LearnedLessons[len(rl.verbalMemory.LearnedLessons)-50:]
	}
	rl.saveVerbalMemory()
}

func (rl *ReflexionLayer) RunCriticExternalVerification(targetURL string) map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	rl.mu.Lock()
	rl.verifications++

	result := map[string]interface{}{
		"target":    targetURL,
		"timestamp": time.Now().Format(time.RFC3339),
		"latency_ms": latency,
	}

	if err != nil {
		result["verified"] = false
		result["payload_status"] = fmt.Sprintf("connection_error: %v", err)
		rl.learnedLesson(fmt.Sprintf("CRITIC verification failed for %s: %v", targetURL, err))
		rl.mu.Unlock()
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 && len(body) > 100 {
		result["verified"] = true
		result["payload_status"] = "active"
		result["content_length"] = len(body)
		rl.verifiedCount++
	} else {
		result["verified"] = false
		result["payload_status"] = fmt.Sprintf("invalid_status_%d", resp.StatusCode)
		rl.learnedLesson(fmt.Sprintf("CRITIC verification non-200 for %s: %d", targetURL, resp.StatusCode))
	}

	rl.mu.Unlock()
	return result
}

func (rl *ReflexionLayer) learnedLesson(lesson string) {
	rl.verbalMemory.LearnedLessons = append(rl.verbalMemory.LearnedLessons, lesson)
	if len(rl.verbalMemory.LearnedLessons) > 50 {
		rl.verbalMemory.LearnedLessons = rl.verbalMemory.LearnedLessons[len(rl.verbalMemory.LearnedLessons)-50:]
	}
	rl.saveVerbalMemory()
}

func (rl *ReflexionLayer) GetVerbalMemory() VerbalMemory {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.verbalMemory
}

func (rl *ReflexionLayer) Stats() map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return map[string]interface{}{
		"critiques":       len(rl.verbalMemory.GlobalCritiques),
		"lessons":         len(rl.verbalMemory.LearnedLessons),
		"verifications":   rl.verifications,
		"verified":        rl.verifiedCount,
		"text_grad_applies": rl.textGradCount,
		"last_refinement": rl.lastRefinement.Format(time.RFC3339),
	}
}
