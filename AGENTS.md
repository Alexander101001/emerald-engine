# Emerald Engine - Agent Fleet Architecture

## Overview
66 specialized agents orchestrated across 5 paradigm levels and 5 development axes. All agents inherit from `ParadigmAgentBase` in `opcode_base.py` and run concurrently via `asyncio.gather` in `app.py`.

## Core Agents (Always Active)

| Agent | File | Role | Interval |
|-------|------|------|----------|
| Orchestrator | `orchestrator.py` | Central fleet coordinator, task scheduling, load balancing | 60s |
| Hacker Bot | `hacker_bot.py` | Zero-trust security monitor, crypto vault audit, vulnerability scan | 3600s |
| Expansion Agent | `platform_integration.py` | Service discovery, platform registration | 300s |
| Cloud Manager | `cloud_manager.py` | Multi-cloud resource orchestration, quota management | 300s |
| Hunter | `autonomous_hunter.py` | Platform hunting, ping-based service discovery | 120s |
| Enterprise Orchestrator | (via orchestrator) | 5-step enterprise workflow engine | 300s |
| Software Architect | `agent_software_architect.py` | Topology discovery, component dependency analysis | 600s |
| DevSecOps | `agent_devsecops.py` | Vault verification, SOC2/GDPR compliance audit | 3600s |
| QA Automation | `agent_qa.py` | Test generation, syntax validation, regression detection | 600s |
| Trend Scouter | `agent_trend_scouter.py` | GitHub trending, BM25 candidate ranking | 1800s |
| Code Evaluator | `agent_code_evaluator.py` | Ephemeral sandbox evaluation, 161-rule validation | 600s |
| Code Synthesizer | `agent_code_synthesizer.py` | Skill merging, dependency resolution, code generation | 600s |
| State Relay | `agent_state_relay.py` | State encryption, resource monitoring, branch dispatch | 30s |
| Runner Grid | `agent_runner_grid.py` | 20-runner rotational pool, dual-pipeline execution | 300s |
| HF Sync | `agent_hf_sync.py` | HuggingFace WebSocket, API sync, SLM scanning | 3600s |
| Identity Manager | `agent_identity_manager.py` | Fingerprint generation, TLS JA3, token vault | 3600s |
| Dashboard Compiler | `agent_dashboard_compiler.py` | Static dashboards, metric aggregation, HTML reports | 600s |
| Chat Interpreter | `agent_chat_interpreter.py` | Async command polling, agent routing | 60s |
| Git Lifecycle | `agent_git_lifecycle.py` | Auto-commit, branch management, push orchestration | 300s |
| Repo Harvester | `agent_harvester.py` | Open-source repo cloning, Ollama integration analysis | 600s |
| Meta-Prompt Engine | `agent_meta_prompt_engine.py` | ASI-Seed Brain, auto-generates 66 agent prompts | 3600s |

## Paradigm Agent Base (`opcode_base.py`)
All agents extend `ParadigmAgentBase` which provides 5 built-in paradigms:
- **Parallel Channels**: Concurrent execution pathways
- **Reasoning Rules**: Configurable inference constraints
- **Domain Mapping**: Context-to-agent routing
- **Telemetry**: Built-in `_emit_telemetry()` for all state reporting
- **Hot Daemon**: `_hot_daemon_loop()` implements the infinite retry with backoff

## Learning Architecture
5 Recursive Levels x 5 Development Axes = 25 prompt generation combinations:

**Levels**: INPUT -> ARCHITECTURAL -> INTERACTION -> ASI_EVOLUTION -> TOOLING

**Axes**: SELF_EVOLUTION, HACKER_GRADE_SECURITY, TINY_WOW_SAAS_FEATURES, EXPANSION_NETWORKING, INFRASTRUCTURE_COST_CONTROL

## Self-Evolution Loop (`self_evolution.py`)
- Scans all Python files for optimization opportunities
- Sends code to local Ollama (`qwen2.5-coder:1.5b`)
- Applies validated refactoring
- Auto-commits and pushes improvements
- Runs every 300s (configurable via `EVOLUTION_INTERVAL`)

## Deployment Commands
```bash
# Deploy core infrastructure (Ollama, Qdrant, Redis, MinIO, Traefik)
python scripts/deploy_module.py core

# Deploy logic layer (agent frameworks, workflows)
python scripts/deploy_module.py logic

# Deploy interface layer (SaaS frontends, low-code, monitoring)
python scripts/deploy_module.py interface

# View deployment status
python scripts/deploy_module.py status
```
