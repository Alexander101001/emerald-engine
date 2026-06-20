package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const vectorDim = 256

type MemoryEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	AccessCount int     `json:"access_count"`
}

type ShortTermMemory struct {
	mu     sync.RWMutex
	data   map[string]string
	expiry map[string]time.Time
	ttl    time.Duration
}

func NewShortTermMemory(ttl time.Duration) *ShortTermMemory {
	stm := &ShortTermMemory{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
		ttl:    ttl,
	}
	go stm.sweepLoop()
	return stm
}

func (s *ShortTermMemory) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	s.expiry[key] = time.Now().Add(s.ttl)
}

func (s *ShortTermMemory) SetTTL(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	s.expiry[key] = time.Now().Add(ttl)
}

func (s *ShortTermMemory) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.expiry[key]
	if !ok || time.Now().After(exp) {
		return "", false
	}
	val, ok := s.data[key]
	return val, ok
}

func (s *ShortTermMemory) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	delete(s.expiry, key)
}

func (s *ShortTermMemory) sweepLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, exp := range s.expiry {
			if now.After(exp) {
				delete(s.data, k)
				delete(s.expiry, k)
			}
		}
		s.mu.Unlock()
	}
}

type LongTermMemory struct {
	mu       sync.RWMutex
	entries  []MemoryEntry
	path     string
	dim      int
}

func NewLongTermMemory(path string) *LongTermMemory {
	ltm := &LongTermMemory{
		entries: []MemoryEntry{},
		path:    path,
		dim:     vectorDim,
	}
	ltm.load()
	go ltm.persistLoop()
	return ltm
}

func (l *LongTermMemory) Store(content string, metadata map[string]string) string {
	embedding := generateEmbedding(content)
	hash := sha256.Sum256([]byte(content + time.Now().String()))
	id := fmt.Sprintf("%x", hash[:8])

	l.mu.Lock()
	l.entries = append(l.entries, MemoryEntry{
		ID:        id,
		Content:   content,
		Embedding: embedding,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	})
	l.mu.Unlock()
	l.persist()
	return id
}

func (l *LongTermMemory) Search(query string, n int) []MemoryEntry {
	qVec := generateEmbedding(query)
	l.mu.RLock()
	defer l.mu.RUnlock()

	type scored struct {
		entry MemoryEntry
		score float64
	}
	var results []scored
	for _, e := range l.entries {
		score := cosineSimilarity(qVec, e.Embedding)
		if score > 0.1 {
			results = append(results, scored{e, score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if n > len(results) {
		n = len(results)
	}
	out := make([]MemoryEntry, n)
	for i := 0; i < n; i++ {
		out[i] = results[i].entry
		out[i].AccessCount++
	}
	return out
}

func (l *LongTermMemory) GetRecent(n int) []MemoryEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if n > len(l.entries) {
		n = len(l.entries)
	}
	return l.entries[len(l.entries)-n:]
}

func (l *LongTermMemory) load() {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return
	}
	var entries []MemoryEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		l.entries = entries
	}
}

func (l *LongTermMemory) persist() {
	l.mu.RLock()
	data, err := json.Marshal(l.entries)
	l.mu.RUnlock()
	if err != nil {
		return
	}
	os.WriteFile(l.path, data, 0644)
}

func (l *LongTermMemory) persistLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		l.persist()
	}
}

func (l *LongTermMemory) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func generateEmbedding(text string) []float64 {
	vec := make([]float64, vectorDim)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec
	}

	for _, token := range tokens {
		h := sha256.Sum256([]byte(token))
		for i := 0; i < 4; i++ {
			idx := int(binary.BigEndian.Uint32(h[i*4:])) % vectorDim
			vec[idx] += 1.0
		}
	}

	magnitude := 0.0
	for _, v := range vec {
		magnitude += v * v
	}
	magnitude = math.Sqrt(magnitude)
	if magnitude > 0 {
		for i := range vec {
			vec[i] /= magnitude
		}
	}
	return vec
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	dot, normA, normB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	text = strings.NewReplacer(
		".", " ", ",", " ", ";", " ", ":", " ",
		"!", " ", "?", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "{", " ", "}", " ",
		"\"", " ", "'", " ", "-", " ", "_", " ",
		"/", " ", "\\", " ", "\n", " ", "\t", " ",
		"\r", " ", "0", "", "1", "", "2", "", "3", "",
		"4", "", "5", "", "6", "", "7", "", "8", "",
		"9", "",
	).Replace(text)

	parts := strings.Fields(text)
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "shall": true, "can": true, "need": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "out": true, "off": true,
		"over": true, "under": true, "again": true, "further": true, "then": true,
		"once": true, "here": true, "there": true, "when": true, "where": true,
		"why": true, "how": true, "all": true, "each": true, "every": true,
		"both": true, "few": true, "more": true, "most": true, "other": true,
		"some": true, "such": true, "no": true, "nor": true, "not": true,
		"only": true, "own": true, "same": true, "so": true, "than": true,
		"too": true, "very": true, "just": true, "because": true,
		"until": true, "while": true, "about": true, "without": true,
		"and": true, "but": true, "or": true, "if": true,
		"that": true, "this": true, "these": true, "those": true,
		"its": true, "they": true, "them": true, "their": true, "what": true,
		"which": true, "who": true, "whom": true, "whose": true, "i": true,
		"me": true, "my": true, "myself": true, "we": true, "our": true,
		"you": true, "your": true, "he": true, "him": true, "his": true,
		"she": true, "her": true, "hers": true, "it": true,
	}

	var out []string
	for _, p := range parts {
		if len(p) > 1 && !stopwords[p] {
			out = append(out, p)
		}
	}
	return out
}
