package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type RAGPipeline struct {
	mu     sync.Mutex
	memory *LongTermMemory
	stm    *ShortTermMemory
}

func NewRAGPipeline(memory *LongTermMemory, stm *ShortTermMemory) *RAGPipeline {
	return &RAGPipeline{
		memory: memory,
		stm:    stm,
	}
}

func (r *RAGPipeline) Ingest(content, source string, metadata map[string]string) string {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["source"] = source
	metadata["ingested_at"] = time.Now().UTC().Format(time.RFC3339)
	return r.memory.Store(content, metadata)
}

func (r *RAGPipeline) Retrieve(query string, n int) []MemoryEntry {
	return r.memory.Search(query, n)
}

func (r *RAGPipeline) IngestScrapedSkills(skills []DiscoveredSkill) {
	for _, s := range skills {
		content := fmt.Sprintf(
			"Skill: %s\nDescription: %s\nLanguage: %s\nCategory: %s\nStars: %d",
			s.Name, s.Description, s.Language, s.Category, s.Stars,
		)
		meta := map[string]string{
			"type":     "skill",
			"name":     s.Name,
			"category": string(s.Category),
			"language": s.Language,
			"stars":    fmt.Sprintf("%d", s.Stars),
			"url":      s.URL,
		}
		r.Ingest(content, s.Source, meta)
	}
}

func (r *RAGPipeline) IngestNicheContent(niche Niche, pages []string) {
	content := fmt.Sprintf("Niche: %s (%s)\n", niche.Name, niche.Keyword)
	for i, p := range pages {
		if i < 5 {
			content += fmt.Sprintf("Page %d: %s\n", i+1, truncate(p, 500))
		}
	}
	meta := map[string]string{
		"type":    "niche",
		"name":    niche.Name,
		"keyword": niche.Keyword,
	}
	r.Ingest(content, "content_generator", meta)
}

func (r *RAGPipeline) AugmentPrompt(basePrompt, query string, nContexts int) string {
	contexts := r.Retrieve(query, nContexts)
	if len(contexts) == 0 {
		return basePrompt
	}

	var sb strings.Builder
	sb.WriteString("Context from knowledge base:\n")
	for i, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("--- Context %d ---\n%s\n", i+1, ctx.Content))
	}
	sb.WriteString("\n--- End of context ---\n\n")
	sb.WriteString(basePrompt)

	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
