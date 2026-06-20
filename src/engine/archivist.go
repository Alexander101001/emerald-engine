package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ArchivistMemory struct {
	ID        int       `json:"id"`
	Timestamp string    `json:"timestamp"`
	AgentRole string    `json:"agent_role"`
	Event     string    `json:"event"`
	Outcome   string    `json:"outcome"`
}

type ArchivistAgent struct {
	mu       sync.Mutex
	dbPath   string
	memories []ArchivistMemory
	nextID   int
	archives int
}

var archivistAgent *ArchivistAgent

func initArchivistAgent() *ArchivistAgent {
	aa := &ArchivistAgent{
		dbPath: "config/memory_stream.db",
	}
	aa.load()
	archivistAgent = aa
	return aa
}

func (aa *ArchivistAgent) load() {
	data, err := os.ReadFile(aa.dbPath)
	if err != nil {
		aa.memories = []ArchivistMemory{}
		aa.nextID = 1
		return
	}
	var records struct {
		Memories []ArchivistMemory `json:"memories"`
		NextID   int               `json:"next_id"`
	}
	if json.Unmarshal(data, &records) == nil {
		aa.memories = records.Memories
		aa.nextID = records.NextID
	} else {
		aa.memories = []ArchivistMemory{}
		aa.nextID = 1
	}
}

func (aa *ArchivistAgent) save() {
	os.MkdirAll(filepath.Dir(aa.dbPath), 0755)
	records := struct {
		Memories []ArchivistMemory `json:"memories"`
		NextID   int               `json:"next_id"`
	}{
		Memories: aa.memories,
		NextID:   aa.nextID,
	}
	data, _ := json.MarshalIndent(records, "", "  ")
	os.WriteFile(aa.dbPath, data, 0644)
}

func (aa *ArchivistAgent) ArchiveExperience(role, event, outcome string) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	entry := ArchivistMemory{
		ID:        aa.nextID,
		Timestamp: time.Now().Format(time.RFC3339),
		AgentRole: role,
		Event:     event,
		Outcome:   outcome,
	}
	aa.nextID++
	aa.memories = append(aa.memories, entry)
	aa.archives++

	if len(aa.memories) > 10000 {
		aa.memories = aa.memories[len(aa.memories)-5000:]
	}

	aa.save()
	fmt.Printf("[Archivist] Logic and outcome successfully embedded in long-term memory.\n")
}

func (aa *ArchivistAgent) QueryPastWisdom(problem string) []ArchivistMemory {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	var results []ArchivistMemory
	problemLower := strings.ToLower(problem)

	for _, m := range aa.memories {
		if strings.Contains(strings.ToLower(m.Event), problemLower) ||
			strings.Contains(strings.ToLower(m.Outcome), problemLower) ||
			strings.Contains(strings.ToLower(m.AgentRole), problemLower) {
			results = append(results, m)
		}
	}

	if len(results) > 100 {
		results = results[len(results)-100:]
	}

	return results
}

func (aa *ArchivistAgent) GetRecentMemories(n int) []ArchivistMemory {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	if len(aa.memories) <= n {
		out := make([]ArchivistMemory, len(aa.memories))
		copy(out, aa.memories)
		return out
	}
	out := make([]ArchivistMemory, n)
	copy(out, aa.memories[len(aa.memories)-n:])
	return out
}

func (aa *ArchivistAgent) Stats() map[string]interface{} {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	return map[string]interface{}{
		"total_memories": len(aa.memories),
		"archives":       aa.archives,
		"next_id":        aa.nextID,
		"db_path":        aa.dbPath,
	}
}
