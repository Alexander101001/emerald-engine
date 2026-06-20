package main

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sync"
	"time"
)

type DeploymentToken struct {
	Token     string    `json:"-"`
	ID        string    `json:"id"`
	ChildName string    `json:"child_name"`
	Scope     string    `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

type Shield struct {
	mu              sync.Mutex
	deploymentTokens []DeploymentToken
	cleanupInterval time.Duration
}

var shield *Shield

func initShield() *Shield {
	s := &Shield{
		cleanupInterval: 1 * time.Hour,
	}
	go s.cleanupLoop()
	shield = s
	return s
}

func (s *Shield) GenerateDeploymentToken(childName string) *DeploymentToken {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := "sgl_" + hex.EncodeToString(tokenBytes)

	id := make([]byte, 8)
	rand.Read(id)

	dt := &DeploymentToken{
		Token:     token,
		ID:        hex.EncodeToString(id),
		ChildName: childName,
		Scope:     "deploy:" + childName,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Used:      false,
	}

	s.mu.Lock()
	s.deploymentTokens = append(s.deploymentTokens, *dt)
	s.mu.Unlock()

	return dt
}

func (s *Shield) ValidateToken(token string) *DeploymentToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, dt := range s.deploymentTokens {
		if dt.Token == token && !dt.Used && time.Now().Before(dt.ExpiresAt) {
			s.deploymentTokens[i].Used = true
			return &dt
		}
	}
	return nil
}

func (s *Shield) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	for range ticker.C {
		s.mu.Lock()
		var active []DeploymentToken
		for _, dt := range s.deploymentTokens {
			if time.Now().Before(dt.ExpiresAt) {
				active = append(active, dt)
			}
		}
		s.deploymentTokens = active
		s.mu.Unlock()
	}
}

func (s *Shield) Stats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := len(s.deploymentTokens)
	used := 0
	expired := 0
	active := 0
	for _, dt := range s.deploymentTokens {
		if dt.Used {
			used++
		} else if time.Now().After(dt.ExpiresAt) {
			expired++
		} else {
			active++
		}
	}
	return map[string]interface{}{
		"total":   total,
		"active":  active,
		"used":    used,
		"expired": expired,
		"scope":   "deploy:*",
		"ttl":     "24h",
	}
}

func jitterSleep(baseMin, baseMax int) {
	if baseMax <= baseMin {
		baseMax = baseMin + 10
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(baseMax-baseMin)))
	jitter := time.Duration(baseMin+int(n.Int64())) * time.Millisecond
	time.Sleep(jitter)
}

func jitterDuration() time.Duration {
	n, _ := rand.Int(rand.Reader, big.NewInt(170))
	return time.Duration(10+int(n.Int64())) * time.Millisecond
}

func throttledRequest(domain string) {
	jitterSleep(10, 180)
	// Domain-specific rate limiting placeholder
	_ = domain
}
