---
title: Emerald Engine
emoji: 
colorFrom: green
colorTo: blue
sdk: docker
app_port: 7860
app_file: Dockerfile
pinned: true
license: mit
short_description: 66-agent autonomous orchestrator with SaaS pipeline
---

# Emerald Engine

Autonomous 66-agent orchestrator running on Hugging Face Spaces. Deployed via Docker container with supervisor-managed multi-process architecture.

## Architecture

```
                  Cloudflare Tunnel (optional)
                           |
                    Traefik :80/:443
                           |
                    emerald_network (bridge)
         ┌───────────┬──────┼──────┬───────────┐
    Ollama:11434  Qdrant:6333  Redis:6379  MinIO:9000
         │                           
    emerald-engine:7860
    ├── app.py (main server + 21 agent loops)
    ├── infinite_research_stream.py (arXiv/Anthropic/OpenAI ingestion)
    └── self_evolution.py (Ollama-driven auto-refactoring)
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| Engine | 7860 | Main aiohttp server with /health, /api/telemetry, /api/prompts |
| Research Stream | - | Infinite async pipeline fetching AI papers from arXiv, Anthropic, OpenAI |
| Self-Evolution | - | Auto-refactoring loop using Ollama qwen2.5-coder:1.5b |
| Cloudflare Tunnel | - | Optional sidecar for custom domain access |

## Storage

Persistent storage mounted at `/data`:
- `/data/qdrant` - Vector database storage
- `/data/research_stream` - Research paper cache
- `/data/logs` - Supervisor and service logs
- `/data/huggingface` - HF cache

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API endpoint |
| `EVOLUTION_INTERVAL` | `300` | Self-evolution loop interval (seconds) |
| `EVOLUTION_AUTO_PUSH` | `false` | Auto-commit and push optimizations |
| `CLOUDFLARE_TUNNEL_TOKEN` | - | Cloudflare tunnel token (optional) |
| `HF_TOKEN` | - | Hugging Face API token |
| `QDRANT_URL` | `http://qdrant:6333` | Qdrant vector DB endpoint |

## API Endpoints

```
GET  /health          → {"status": "ok", "version": "5.0-opcode", "uptime": "..."}
GET  /api/telemetry   → System telemetry snapshot
GET  /api/stream      → Stream pipeline status
GET  /api/prompts     → Registered agent prompts
```

## Deployment

Push to GitHub main branch triggers automatic deployment to Hugging Face Spaces via GitHub Actions. The CI/CD pipeline builds the Docker image, pushes to HF Container Registry, and restarts the Space.

```bash
# Manual trigger from GitHub Actions UI
# Or use the CLI:
git push origin main
```

## Cloudflare Tunnel (Optional)

Set `CLOUDFLARE_TUNNEL_TOKEN` in the Space Secrets to enable a Cloudflare tunnel for custom domain access. The tunnel runs as a supervisor-managed sidecar process.
