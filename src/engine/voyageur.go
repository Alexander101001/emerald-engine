package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type VoyageurEngine struct {
	mu              sync.Mutex
	skillsPath      string
	totPath         string
	compiledSkills  map[string]SkillRecord
	verifiedRoutines []string
	totGraph        []ThoughtPath
	skillsArchived  int
	totEvaluations  int
}

type SkillRecord struct {
	Code      string `json:"code"`
	Timestamp string `json:"timestamp"`
	Category  string `json:"category"`
	Successes int    `json:"successes"`
}

type ThoughtPath struct {
	Label           string  `json:"label"`
	StabilityRating float64 `json:"stability_rating"`
	TokenEfficiency float64 `json:"token_efficiency"`
	SurvivalScore   float64 `json:"survival_score"`
	Selected        bool    `json:"selected"`
	EvaluatedAt     string  `json:"evaluated_at"`
}

var voyageurEngine *VoyageurEngine

func initVoyageurEngine() *VoyageurEngine {
	ve := &VoyageurEngine{
		skillsPath:     "config/voyageur_skills.json",
		totPath:        "config/tot_exploration_graph.json",
		compiledSkills: make(map[string]SkillRecord),
	}
	ve.load()
	voyageurEngine = ve
	return ve
}

func (ve *VoyageurEngine) load() {
	if data, err := os.ReadFile(ve.skillsPath); err == nil {
		var lib struct {
			CompiledSkills   map[string]SkillRecord `json:"compiled_skills"`
			VerifiedRoutines []string               `json:"verified_routines"`
		}
		if json.Unmarshal(data, &lib) == nil {
			ve.compiledSkills = lib.CompiledSkills
			ve.verifiedRoutines = lib.VerifiedRoutines
		}
	}
	if data, err := os.ReadFile(ve.totPath); err == nil {
		var graph struct {
			Paths []ThoughtPath `json:"exploration_paths"`
		}
		if json.Unmarshal(data, &graph) == nil {
			ve.totGraph = graph.Paths
		}
	}
}

func (ve *VoyageurEngine) save() {
	os.MkdirAll("config", 0755)

	lib := struct {
		CompiledSkills   map[string]SkillRecord `json:"compiled_skills"`
		VerifiedRoutines []string               `json:"verified_routines"`
	}{
		CompiledSkills:   ve.compiledSkills,
		VerifiedRoutines: ve.verifiedRoutines,
	}
	data, _ := json.MarshalIndent(lib, "", "  ")
	os.WriteFile(ve.skillsPath, data, 0644)

	graph := struct {
		Paths []ThoughtPath `json:"exploration_paths"`
	}{
		Paths: ve.totGraph,
	}
	data, _ = json.MarshalIndent(graph, "", "  ")
	os.WriteFile(ve.totPath, data, 0644)
}

func (ve *VoyageurEngine) RegisterSuccessfulSkill(skillName, code, category string) {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	existing, ok := ve.compiledSkills[skillName]
	if ok {
		existing.Successes++
		existing.Code = code
		existing.Timestamp = time.Now().Format(time.RFC3339)
		ve.compiledSkills[skillName] = existing
	} else {
		ve.compiledSkills[skillName] = SkillRecord{
			Code:      code,
			Timestamp: time.Now().Format(time.RFC3339),
			Category:  category,
			Successes: 1,
		}
	}
	ve.skillsArchived++
	ve.save()
	fmt.Printf("[Voyageur] Dynamically integrated new execution skill: %s (%s)\n", skillName, category)
}

func (ve *VoyageurEngine) EvaluateTreeOfThoughts(prospectivePaths []ThoughtPath) *ThoughtPath {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	ve.totEvaluations++
	highestScore := -1.0
	var selected *ThoughtPath

	for i := range prospectivePaths {
		p := &prospectivePaths[i]
		p.SurvivalScore = p.StabilityRating * p.TokenEfficiency
		p.EvaluatedAt = time.Now().Format(time.RFC3339)

		if p.SurvivalScore > highestScore {
			highestScore = p.SurvivalScore
			selected = p
		}
	}

	if selected != nil {
		selected.Selected = true
	}

	ve.totGraph = append(ve.totGraph, prospectivePaths...)
	if len(ve.totGraph) > 100 {
		ve.totGraph = ve.totGraph[len(ve.totGraph)-100:]
	}
	ve.save()

	if selected != nil {
		fmt.Printf("[ToT] Selected optimal thought branch '%s' with stability score: %.2f\n",
			selected.Label, highestScore)
	}
	return selected
}

func (ve *VoyageurEngine) GetSkill(name string) *SkillRecord {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	if s, ok := ve.compiledSkills[name]; ok {
		return &s
	}
	return nil
}

func (ve *VoyageurEngine) GetAllSkills() []string {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	names := make([]string, 0, len(ve.compiledSkills))
	for n := range ve.compiledSkills {
		names = append(names, n)
	}
	return names
}

func (ve *VoyageurEngine) Stats() map[string]interface{} {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	return map[string]interface{}{
		"skills_archived":  ve.skillsArchived,
		"compiled_skills":  len(ve.compiledSkills),
		"verified_routines": len(ve.verifiedRoutines),
		"tot_evaluations":  ve.totEvaluations,
		"tot_paths":        len(ve.totGraph),
	}
}
