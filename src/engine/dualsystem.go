package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type ThinkingMode string

const (
	System1Fast     ThinkingMode = "system_1_fast"
	System2Slow     ThinkingMode = "system_2_slow"
)

type DualSystemScheduler struct {
	mu            sync.Mutex
	system1Loops  int
	system2Loops  int
	lastSystem1   time.Time
	lastSystem2   time.Time
	system1Errors int
	system2Errors int
	enabled       bool
}

var dualSystem *DualSystemScheduler

func initDualSystem() *DualSystemScheduler {
	ds := &DualSystemScheduler{
		enabled: true,
	}
	dualSystem = ds
	fmt.Println("[DUAL] System 1 (Fast): heartbeat, health, token rotation, anti-adblock, fingerprint")
	fmt.Println("[DUAL] System 2 (Slow): cognitive analysis, OPRO evolution, swarm discussion, ToT, Voyageur")
	go ds.system1Loop()
	go ds.system2Loop()
	return ds
}

func (ds *DualSystemScheduler) system1Loop() {
	time.Sleep(5 * time.Second)
	ds.logSystem1("System 1 online — fast reflex loop active")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ds.mu.Lock()
		ds.system1Loops++
		ds.lastSystem1 = time.Now()
		ds.mu.Unlock()

		if heartbeatDaemon != nil && heartbeatDaemon.running {
			ds.logSystem1(fmt.Sprintf("Heartbeat active | pings: %d", heartbeatDaemon.pings))
		}

		if tokenMatrix != nil {
			ds.logSystem1(fmt.Sprintf("Token slots: %d | free endpoints: %d",
				len(tokenMatrix.slots), len(tokenMatrix.freeEndpoints)))
		}

		if shield != nil {
			shield.mu.Lock()
			tokens := len(shield.deploymentTokens)
			shield.mu.Unlock()
			ds.logSystem1(fmt.Sprintf("Shield tokens active: %d", tokens))
		}
	}
}

func (ds *DualSystemScheduler) system2Loop() {
	time.Sleep(30 * time.Second)
	ds.logSystem2("System 2 online — analytical reasoning loop active")

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ds.mu.Lock()
		ds.system2Loops++
		ds.lastSystem2 = time.Now()
		ds.mu.Unlock()

		children := 0
		if orchestrator != nil {
			orchestrator.mu.Lock()
			children = len(orchestrator.Children)
			orchestrator.mu.Unlock()
		}
		ds.logSystem2(fmt.Sprintf("Children: %d | Cognitive: %t | Swarm: %t",
			children, cognitive != nil, swarmOrchestrator != nil))

		if voyageurEngine != nil {
			skills := voyageurEngine.GetAllSkills()
			ds.logSystem2(fmt.Sprintf("Skill library: %d modules archived", len(skills)))
		}

		if reflexionLayer != nil {
			vm := reflexionLayer.GetVerbalMemory()
			ds.logSystem2(fmt.Sprintf("Verbal memory: %d critiques, %d lessons",
				len(vm.GlobalCritiques), len(vm.LearnedLessons)))
		}

		if oproOptimizer != nil {
			oproOptimizer.mu.Lock()
			version := oproOptimizer.promptVersion
			oproOptimizer.mu.Unlock()
			ds.logSystem2(fmt.Sprintf("OPRO prompt version: %d", version))
		}

		ds.selfCheck()
	}
}

func (ds *DualSystemScheduler) logSystem1(msg string) {
	if !ds.enabled {
		return
	}
	fmt.Printf("[SYS1] %s\n", msg)
	if swarmOrchestrator != nil {
		swarmOrchestrator.LogEvent("System1_Fast", msg, "system_1")
	}
}

func (ds *DualSystemScheduler) logSystem2(msg string) {
	if !ds.enabled {
		return
	}
	fmt.Printf("[SYS2] %s\n", msg)
	if swarmOrchestrator != nil {
		swarmOrchestrator.LogEvent("System2_Slow", msg, "system_2")
	}
}

func (ds *DualSystemScheduler) selfCheck() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.enabled {
		return
	}

	if time.Since(ds.lastSystem1) > 10*time.Minute {
		ds.system1Errors++
		fmt.Fprintf(os.Stderr, "[DUAL] WARNING: System 1 silent for >10min\n")
	}

	if time.Since(ds.lastSystem2) > 60*time.Minute {
		ds.system2Errors++
		fmt.Fprintf(os.Stderr, "[DUAL] WARNING: System 2 silent for >60min\n")
	}

	if ds.system1Errors > 3 {
		fmt.Fprintf(os.Stderr, "[DUAL] CRITICAL: System 1 has %d errors — restarting fast loop\n", ds.system1Errors)
		go ds.system1Loop()
		ds.system1Errors = 0
	}

	if ds.system2Errors > 3 {
		fmt.Fprintf(os.Stderr, "[DUAL] CRITICAL: System 2 has %d errors — restarting slow loop\n", ds.system2Errors)
		go ds.system2Loop()
		ds.system2Errors = 0
	}
}

func (ds *DualSystemScheduler) Stats() map[string]interface{} {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return map[string]interface{}{
		"system_1_loops":    ds.system1Loops,
		"system_2_loops":    ds.system2Loops,
		"system_1_errors":   ds.system1Errors,
		"system_2_errors":   ds.system2Errors,
		"system_1_last":     ds.lastSystem1.Format(time.RFC3339),
		"system_2_last":     ds.lastSystem2.Format(time.RFC3339),
		"enabled":           ds.enabled,
		"thinking_modes":    []string{string(System1Fast), string(System2Slow)},
	}
}
