package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type MCPToolFunc func(params map[string]interface{}) (interface{}, error)

type MCPTool struct {
	Name        string
	Description string
	Handler     MCPToolFunc
}

type MCPDataSource struct {
	Name        string
	Description string
	Query       func(query string) (interface{}, error)
}

type MCPRegistry struct {
	mu          sync.RWMutex
	tools       map[string]MCPTool
	dataSources map[string]MCPDataSource
	execLog     []MCPExecRecord
	maxLog      int
}

type MCPExecRecord struct {
	Tool      string
	Params    json.RawMessage
	Result    string
	Error     string
	Duration  time.Duration
	Timestamp time.Time
}

func NewMCPRegistry() *MCPRegistry {
	return &MCPRegistry{
		tools:       make(map[string]MCPTool),
		dataSources: make(map[string]MCPDataSource),
		execLog:     make([]MCPExecRecord, 0, 100),
		maxLog:      100,
	}
}

func (m *MCPRegistry) RegisterTool(tool MCPTool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name] = tool
}

func (m *MCPRegistry) RegisterDataSource(ds MCPDataSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataSources[ds.Name] = ds
}

func (m *MCPRegistry) ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
	m.mu.RLock()
	tool, ok := m.tools[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mcp: unknown tool %q", name)
	}

	start := time.Now()
	result, err := tool.Handler(params)
	duration := time.Since(start)

	paramsRaw, _ := json.Marshal(params)
	resultStr := ""
	if result != nil {
		resultStr = fmt.Sprintf("%v", result)
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	record := MCPExecRecord{
		Tool:      name,
		Params:    paramsRaw,
		Result:    resultStr,
		Error:     errStr,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	m.mu.Lock()
	m.execLog = append(m.execLog, record)
	if len(m.execLog) > m.maxLog {
		m.execLog = m.execLog[len(m.execLog)-m.maxLog:]
	}
	m.mu.Unlock()

	return result, err
}

func (m *MCPRegistry) QueryDataSource(name, query string) (interface{}, error) {
	m.mu.RLock()
	ds, ok := m.dataSources[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mcp: unknown data source %q", name)
	}
	return ds.Query(query)
}

func (m *MCPRegistry) ListTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for n := range m.tools {
		names = append(names, n)
	}
	return names
}

func (m *MCPRegistry) ListDataSources() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for n := range m.dataSources {
		names = append(names, n)
	}
	return names
}

func (m *MCPRegistry) GetExecLog() []MCPExecRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MCPExecRecord, len(m.execLog))
	copy(out, m.execLog)
	return out
}

func (m *MCPRegistry) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	toolCount := len(m.tools)
	dsCount := len(m.dataSources)
	logCount := len(m.execLog)
	return map[string]interface{}{
		"tools":     toolCount,
		"sources":   dsCount,
		"exec_log":  logCount,
	}
}
