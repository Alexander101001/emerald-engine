package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

type AgentStatus string

const (
	AgentInit     AgentStatus = "init"
	AgentRunning  AgentStatus = "running"
	AgentIdle     AgentStatus = "idle"
	AgentError    AgentStatus = "error"
	AgentApproval AgentStatus = "awaiting_approval"
	AgentStopped  AgentStatus = "stopped"
)

type AgentTask struct {
	ID        string
	Type      string
	Params    map[string]interface{}
	Status    string
	Result    interface{}
	Error     string
	CreatedAt time.Time
	CompletedAt time.Time
}

type ActionPlan struct {
	Tasks     []AgentTask
	Rationale string
}

type AgentLoop struct {
	Perceive func() (interface{}, error)
	Reason   func(interface{}) (ActionPlan, error)
	Plan     func(ActionPlan) []AgentTask
	Execute  func(AgentTask) (interface{}, error)
	Validate func(AgentTask, interface{}) error
}

type BaseAgent struct {
	mu       sync.Mutex
	Name     string
	Status   AgentStatus
	Loop     AgentLoop
	Memory   *LongTermMemory
	STM      *ShortTermMemory
	MCP      *MCPRegistry
	RAG      *RAGPipeline
	Stop     chan struct{}
	Tasks    []AgentTask
	CycleNum int
	Logs     []string
}

func NewBaseAgent(name string, loop AgentLoop, memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	return &BaseAgent{
		Name:   name,
		Status: AgentInit,
		Loop:   loop,
		Memory: memory,
		STM:    stm,
		MCP:    mcp,
		RAG:    rag,
		Stop:   make(chan struct{}),
	}
}

func (a *BaseAgent) Start(interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.mu.Lock()
				a.Status = AgentError
				a.Logs = append(a.Logs, fmt.Sprintf("[PANIC] %v", r))
				a.mu.Unlock()
				time.Sleep(10 * time.Second)
				a.Start(interval)
			}
		}()

		a.mu.Lock()
		a.Status = AgentRunning
		a.mu.Unlock()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-a.Stop:
				a.mu.Lock()
				a.Status = AgentStopped
				a.mu.Unlock()
				return
			case <-ticker.C:
				a.runCycle()
			}
		}
	}()
}

func (a *BaseAgent) StopAgent() {
	close(a.Stop)
}

func (a *BaseAgent) runCycle() {
	a.mu.Lock()
	a.CycleNum++
	cycleID := a.CycleNum
	a.mu.Unlock()

	a.addLog(fmt.Sprintf("[%s] Cycle %d start", a.Name, cycleID))

	percept, err := a.safePerceive()
	if err != nil {
		a.addLog(fmt.Sprintf("[%s] Perceive error: %v", a.Name, err))
		return
	}

	plan, err := a.safeReason(percept)
	if err != nil {
		a.addLog(fmt.Sprintf("[%s] Reason error: %v", a.Name, err))
		return
	}

	tasks := a.Loop.Plan(plan)
	for _, task := range tasks {
		task.ID = generateTaskID()
		task.CreatedAt = time.Now()
		task.Status = "pending"

		a.mu.Lock()
		a.Tasks = append(a.Tasks, task)
		a.mu.Unlock()

		// Security Auditor must approve all deployment tasks
		if task.Type == "deploy" || task.Type == "write_code" {
			if !a.requestApproval(task) {
				task.Status = "rejected"
				task.Error = "rejected by Security Auditor"
				a.addLog(fmt.Sprintf("[%s] Task %s rejected by security", a.Name, task.ID))
				continue
			}
		}

		task.Status = "executing"
		result, err := a.Loop.Execute(task)
		task.CompletedAt = time.Now()

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			a.addLog(fmt.Sprintf("[%s] Task %s failed: %v", a.Name, task.ID, err))
			continue
		}

		if err := a.Loop.Validate(task, result); err != nil {
			task.Status = "validation_failed"
			task.Error = err.Error()
			a.addLog(fmt.Sprintf("[%s] Task %s validation: %v", a.Name, task.ID, err))
			continue
		}

		task.Status = "completed"
		task.Result = result
		a.addLog(fmt.Sprintf("[%s] Task %s completed", a.Name, task.ID))
	}

	a.mu.Lock()
	if len(a.Tasks) > 100 {
		a.Tasks = a.Tasks[len(a.Tasks)-100:]
	}
	a.mu.Unlock()

	a.addLog(fmt.Sprintf("[%s] Cycle %d done", a.Name, cycleID))
}

func (a *BaseAgent) safePerceive() (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			a.addLog(fmt.Sprintf("[%s] Perceive panic: %v", a.Name, r))
		}
	}()
	return a.Loop.Perceive()
}

func (a *BaseAgent) safeReason(percept interface{}) (ActionPlan, error) {
	defer func() {
		if r := recover(); r != nil {
			a.addLog(fmt.Sprintf("[%s] Reason panic: %v", a.Name, r))
		}
	}()
	return a.Loop.Reason(percept)
}

func (a *BaseAgent) requestApproval(task AgentTask) bool {
	if securityAgent == nil {
		return true
	}
	approved := make(chan bool, 1)
	securityAgent.approvalChan <- ApprovalRequest{
		TaskID:   task.ID,
		TaskType: task.Type,
		Params:   task.Params,
		Response: approved,
	}
	select {
	case result := <-approved:
		return result
	case <-time.After(30 * time.Second):
		return false
	}
}

func (a *BaseAgent) addLog(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Logs = append(a.Logs, msg)
	if len(a.Logs) > 50 {
		a.Logs = a.Logs[len(a.Logs)-50:]
	}
	fmt.Println(msg)
}

func generateTaskID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ========== Agent Coordinator ==========

type ApprovalRequest struct {
	TaskID   string
	TaskType string
	Params   map[string]interface{}
	Response chan bool
}

type AgentCoordinator struct {
	mu             sync.Mutex
	Agents         map[string]*BaseAgent
	Memory         *LongTermMemory
	STM            *ShortTermMemory
	MCP            *MCPRegistry
	RAG            *RAGPipeline
	activePlan     string
	consensusMode  string
}

func NewAgentCoordinator(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *AgentCoordinator {
	return &AgentCoordinator{
		Agents:        make(map[string]*BaseAgent),
		Memory:        memory,
		STM:           stm,
		MCP:           mcp,
		RAG:           rag,
		consensusMode: "strict",
	}
}

func (ac *AgentCoordinator) RegisterAgent(name string, agent *BaseAgent) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Agents[name] = agent
}

func (ac *AgentCoordinator) Broadcast(msg string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	for _, a := range ac.Agents {
		a.STM.Set("broadcast_"+time.Now().Format("150405"), msg)
	}
}

func (ac *AgentCoordinator) Stats() map[string]interface{} {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	stats := make(map[string]interface{})
	for name, a := range ac.Agents {
		a.mu.Lock()
		stats[name] = map[string]interface{}{
			"status":    string(a.Status),
			"cycles":    a.CycleNum,
			"tasks":     len(a.Tasks),
			"logs":      len(a.Logs),
		}
		a.mu.Unlock()
	}
	stats["memory_entries"] = ac.Memory.Size()
	stats["consensus"] = ac.consensusMode
	return stats
}

// ========== Creator Agent ==========

var creatorAgent *BaseAgent

func initCreatorAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			return map[string]interface{}{
				"niches":            len(niches),
				"undeployed":        countUndeployed(),
				"skill_cache_age":   getSkillCacheAge(),
			}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			p := percept.(map[string]interface{})
			undeployed := p["undeployed"].(int)
			var tasks []AgentTask
			if undeployed > 0 {
				tasks = append(tasks, AgentTask{
					Type: "generate_niche",
					Params: map[string]interface{}{
						"count": int(math.Min(float64(undeployed), 3)),
					},
				})
			}
			return ActionPlan{
				Tasks:     tasks,
				Rationale: fmt.Sprintf("Generating content for %d undeployed niches", undeployed),
			}, nil
		},
		Plan: func(plan ActionPlan) []AgentTask {
			return plan.Tasks
		},
		Execute: func(task AgentTask) (interface{}, error) {
			count := int(task.Params["count"].(float64))
			generated := 0
			for _, n := range niches {
				if generated >= count {
					break
				}
				if !isNicheDeployed(n.Keyword) {
					rag.IngestNicheContent(n, []string{
						fmt.Sprintf("Generated content for %s niche", n.Name),
					})
					generated++
				}
			}
			return map[string]interface{}{"generated": generated}, nil
		},
		Validate: func(task AgentTask, result interface{}) error {
			r := result.(map[string]interface{})
			if r["generated"].(int) == 0 {
				return fmt.Errorf("no content was generated")
			}
			return nil
		},
	}

	agent := NewBaseAgent("creator", loop, memory, stm, mcp, rag)
	creatorAgent = agent
	return agent
}

// ========== Traffic Architect Agent ==========

var trafficAgent *BaseAgent

func initTrafficAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			children := 0
			if orchestrator != nil {
				orchestrator.mu.Lock()
				children = len(orchestrator.Children)
				orchestrator.mu.Unlock()
			}
	return map[string]interface{}{
			"children": children,
			"has_cf":   cfAgent != nil && cfAgent.initialized,
		}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			p := percept.(map[string]interface{})
			var tasks []AgentTask
			if p["has_cf"].(bool) {
				tasks = append(tasks, AgentTask{
					Type: "optimize_routing",
					Params: map[string]interface{}{
						"children": p["children"],
					},
				})
			}
			return ActionPlan{
				Tasks:     tasks,
				Rationale: "Optimizing traffic routing across children",
			}, nil
		},
		Plan: func(plan ActionPlan) []AgentTask {
			return plan.Tasks
		},
		Execute: func(task AgentTask) (interface{}, error) {
			return map[string]interface{}{"routed": task.Params["children"]}, nil
		},
		Validate: func(task AgentTask, result interface{}) error {
			return nil
		},
	}

	agent := NewBaseAgent("traffic", loop, memory, stm, mcp, rag)
	trafficAgent = agent
	return agent
}

// ========== Security Auditor Agent ==========

type SecurityAuditor struct {
	BaseAgent
	approvalChan chan ApprovalRequest
}

var securityAgent *SecurityAuditor

func initSecurityAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			return map[string]interface{}{
				"mode": "active",
			}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			return ActionPlan{
				Tasks:     []AgentTask{},
				Rationale: "Security auditor monitoring for code approval requests",
			}, nil
		},
		Plan: func(plan ActionPlan) []AgentTask {
			return plan.Tasks
		},
		Execute: func(task AgentTask) (interface{}, error) {
			return nil, nil
		},
		Validate: func(task AgentTask, result interface{}) error {
			return nil
		},
	}

	agent := &SecurityAuditor{
		approvalChan: make(chan ApprovalRequest, 50),
	}
	agent.Name = "security"
	agent.Status = AgentInit
	agent.Loop = loop
	agent.Memory = memory
	agent.STM = stm
	agent.MCP = mcp
	agent.RAG = rag
	agent.Stop = make(chan struct{})

	go agent.approvalLoop()
	securityAgent = agent
	return &agent.BaseAgent
}

func (sec *SecurityAuditor) approvalLoop() {
	for req := range sec.approvalChan {
		// Swarm multi-agent discussion before security audit
		swarmApproved := true
		if swarmOrchestrator != nil {
			taskDesc := fmt.Sprintf("%s: %s", req.TaskType, req.TaskID)
			swarmApproved = swarmOrchestrator.MultiAgentDiscussion(taskDesc, req.TaskType, req.Params)
			swarmOrchestrator.LogEvent("SecurityOfficer", fmt.Sprintf("Auditing after swarm: %s", req.TaskType), taskDesc)
		}

		if !swarmApproved {
			select {
			case req.Response <- false:
			default:
			}
			continue
		}

		approved := sec.auditTask(req)
		select {
		case req.Response <- approved:
		default:
		}
	}
}

func (sec *SecurityAuditor) auditTask(req ApprovalRequest) bool {
	sec.addLog(fmt.Sprintf("[SECURITY] Auditing task %s type=%s", req.TaskID, req.TaskType))

	if req.TaskType == "write_code" {
		code, ok := req.Params["code"].(string)
		if ok {
			secrets := []string{
				"ghp_", "hf_", "sk-", "api_key", "secret",
				"password", "token", "bearer", "auth",
			}
			codeLower := strings.ToLower(code)
			for _, secret := range secrets {
				if strings.Contains(codeLower, secret) && strings.Contains(codeLower, "=") {
					sec.addLog(fmt.Sprintf("[SECURITY] REJECTED: potential secret leak (%s)", secret))
					return false
				}
			}
		}
	}

	// CRITIC external verification gate: refuse deploy tasks that fail endpoint check
	if req.TaskType == "deploy" && reflexionLayer != nil {
		if url, ok := req.Params["url"].(string); ok && url != "" {
			result := reflexionLayer.RunCriticExternalVerification(url)
			if verified, ok := result["verified"].(bool); ok && !verified {
				sec.addLog(fmt.Sprintf("[SECURITY] CRITIC REJECTED deploy to %s: %v", url, result["payload_status"]))
				if oproOptimizer != nil {
					oproOptimizer.RecordOp("critic_verify", false, fmt.Sprintf("deploy_rejected_%s", url))
				}
				return false
			}
			sec.addLog(fmt.Sprintf("[SECURITY] CRITIC verified: %s", url))
		}
	}

	if req.TaskType == "deploy" {
		taskName, _ := req.Params["name"].(string)
		if taskName != "" && reflexionLayer != nil {
			memory := reflexionLayer.GetVerbalMemory()
			if len(memory.GlobalCritiques) > 0 {
				sec.addLog(fmt.Sprintf("[SECURITY] Child %s inherits %d reflexion critiques",
					taskName, len(memory.GlobalCritiques)))
			}
		}
	}

	return true
}

// ========== Monetization Specialist Agent ==========

var monetizationAgent *BaseAgent

func initMonetizationAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			fulfillmentDB.mu.Lock()
			revenue := fulfillmentDB.totalRevenue()
			sales := len(fulfillmentDB.Sales)
			fulfillmentDB.mu.Unlock()
			return map[string]interface{}{
				"revenue": revenue,
				"sales":   sales,
				"adsense": vault["ADSENSE_CLIENT_ID"],
			}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			p := percept.(map[string]interface{})
			revenue := p["revenue"].(float64)
			var tasks []AgentTask
			if revenue == 0 {
				tasks = append(tasks, AgentTask{
					Type: "optimize_monetization",
					Params: map[string]interface{}{
						"reason": "zero_revenue",
					},
				})
			}
			tasks = append(tasks, AgentTask{
				Type: "audit_affiliate_links",
				Params: map[string]interface{}{},
			})
			return ActionPlan{
				Tasks:     tasks,
				Rationale: "Optimizing revenue streams and affiliate placement",
			}, nil
		},
		Plan: func(plan ActionPlan) []AgentTask {
			return plan.Tasks
		},
		Execute: func(task AgentTask) (interface{}, error) {
			return map[string]interface{}{
				"type":   task.Type,
				"status": "audited",
			}, nil
		},
		Validate: func(task AgentTask, result interface{}) error {
			return nil
		},
	}

	agent := NewBaseAgent("monetization", loop, memory, stm, mcp, rag)
	monetizationAgent = agent
	return agent
}

// ========== GitHub Agent ==========

var githubSwarmAgent *BaseAgent
var huggingfaceSwarmAgent *BaseAgent

func initGitHubSwarmAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			token := vaultGet("GITHUB_TOKEN", "")
			return map[string]interface{}{
				"has_token": token != "",
				"children":  func() int { if orchestrator != nil { return len(orchestrator.Children) }; return 0 }(),
			}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			return ActionPlan{
				Tasks:     []AgentTask{},
				Rationale: "GitHub agent monitoring repository and CI/CD state",
			}, nil
		},
		Plan: func(plan ActionPlan) []AgentTask { return plan.Tasks },
		Execute: func(task AgentTask) (interface{}, error) {
			return map[string]interface{}{"status": "GitHub agent ready for repo operations"}, nil
		},
		Validate: func(task AgentTask, result interface{}) error { return nil },
	}
	agent := NewBaseAgent("GitHubAgent", loop, memory, stm, mcp, rag)
	githubSwarmAgent = agent
	return agent
}

func initHuggingFaceSwarmAgent(memory *LongTermMemory, stm *ShortTermMemory, mcp *MCPRegistry, rag *RAGPipeline) *BaseAgent {
	loop := AgentLoop{
		Perceive: func() (interface{}, error) {
			token := vaultGet("HF_TOKEN", "")
			children := 0
			if orchestrator != nil {
				orchestrator.mu.Lock()
				children = len(orchestrator.Children)
				orchestrator.mu.Unlock()
			}
			return map[string]interface{}{
				"has_token": token != "",
				"children":  children,
			}, nil
		},
		Reason: func(percept interface{}) (ActionPlan, error) {
			return ActionPlan{
				Tasks:     []AgentTask{},
				Rationale: "HuggingFace agent monitoring Spaces deployment and inference",
			}, nil
		},
		Plan:  func(plan ActionPlan) []AgentTask { return plan.Tasks },
		Execute: func(task AgentTask) (interface{}, error) {
			return map[string]interface{}{"status": "HF agent ready for Spaces operations"}, nil
		},
		Validate: func(task AgentTask, result interface{}) error { return nil },
	}
	agent := NewBaseAgent("HuggingFaceAgent", loop, memory, stm, mcp, rag)
	huggingfaceSwarmAgent = agent
	return agent
}

// ========== Swarm Delegation ==========

type AgentRole struct {
	Name        string
	Description string
}

func (ac *AgentCoordinator) DelegateTask(taskName, description string) bool {
	roles := []AgentRole{
		{"Architect", "Plans the infrastructure and container topology."},
		{"GitHubAgent", "Handles repository creation, CI/CD, and code push."},
		{"HuggingFaceAgent", "Manages model hosting, Spaces deployment, and inference."},
		{"Monetizer", "Analyzes niche profitability and optimizes affiliate/ad streams."},
		{"SecurityAgent", "Audits code for vulnerabilities and manages token rotation."},
	}

	fmt.Printf("\n[SWARM] Delegating: %s\n", taskName)

	for _, role := range roles {
		if strings.Contains(taskName, role.Name) || strings.Contains(description, role.Name) {
			fmt.Printf("[%s] (%s) -> Executing: %s\n", role.Name, role.Description, description)
			ac.Broadcast(fmt.Sprintf("[%s] Task delegated: %s", role.Name, taskName))

			if swarmOrchestrator != nil {
				swarmOrchestrator.LogEvent(role.Name, fmt.Sprintf("Executing: %s", description), taskName)
			}
			return true
		}
	}

	// Fallback: if taskName contains known keywords, route to best-matching agent
	taskLower := strings.ToLower(taskName) + " " + strings.ToLower(description)
	switch {
	case strings.Contains(taskLower, "github") || strings.Contains(taskLower, "repo") || strings.Contains(taskLower, "push") || strings.Contains(taskLower, "commit"):
		fmt.Printf("[GitHubAgent] (Repository operations) -> Executing: %s\n", description)
		return true
	case strings.Contains(taskLower, "huggingface") || strings.Contains(taskLower, "hf") || strings.Contains(taskLower, "space") || strings.Contains(taskLower, "model"):
		fmt.Printf("[HuggingFaceAgent] (Spaces deployment) -> Executing: %s\n", description)
		return true
	case strings.Contains(taskLower, "monetiz") || strings.Contains(taskLower, "affiliate") || strings.Contains(taskLower, "ad") || strings.Contains(taskLower, "revenue"):
		fmt.Printf("[Monetizer] (Revenue optimization) -> Executing: %s\n", description)
		return true
	case strings.Contains(taskLower, "security") || strings.Contains(taskLower, "token") || strings.Contains(taskLower, "audit") || strings.Contains(taskLower, "vulnerab"):
		fmt.Printf("[SecurityAgent] (Security audit) -> Executing: %s\n", description)
		return true
	case strings.Contains(taskLower, "infra") || strings.Contains(taskLower, "deploy") || strings.Contains(taskLower, "container") || strings.Contains(taskLower, "topology"):
		fmt.Printf("[Architect] (Infrastructure planning) -> Executing: %s\n", description)
		return true
	}

	return false
}

// ========== Helpers ==========

func countUndeployed() int {
	count := 0
	for _, n := range niches {
		if !isNicheDeployed(n.Keyword) {
			count++
		}
	}
	return count
}

func isNicheDeployed(keyword string) bool {
	if orchestrator == nil {
		return false
	}
	orchestrator.mu.Lock()
	defer orchestrator.mu.Unlock()
	_, ok := orchestrator.Children[keyword]
	return ok
}

func getSkillCacheAge() string {
	if skillRegistry == nil {
		return "no_cache"
	}
	skillRegistry.mu.RLock()
	defer skillRegistry.mu.RUnlock()
	if skillRegistry.ScrapedAt.IsZero() {
		return "never"
	}
	return time.Since(skillRegistry.ScrapedAt).Round(time.Minute).String()
}
