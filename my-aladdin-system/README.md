---
title: Aladdin System
emoji: 
colorFrom: gold
colorTo: green
sdk: docker
app_port: 7860
pinned: false
license: mit
short_description: Autonomous financial agent with pulse system, evolutionary memory, and micro-earning
---

# Aladdin Autonomous Financial Agent

Self-evolving 24/7 financial agent that accumulates capital through Binance micro-earning strategies and evolves its trading logic via local Ollama reasoning.

## Architecture

```
supervisord (4 processes)
├── aladdin-core (main.py)
│   ├── Pulse System (every 300s)
│   ├── Evolutionary Loop (every 1800s)
│   └── Self-Healing Process Monitor
├── micro-earning (micro_earning.py)
│   ├── Dust Conversion to BNB
│   ├── Simple Earn Subscription
│   └── Rewards Hub Monitoring
├── ollama serve
│   └── qwen2.5-coder:1.5b (local reasoning)
└── cloudflared (optional tunnel)
```

## Lifecycle

1. **Accumulation Mode** (balance < $20): Micro-earning converts dust, subscribes idle balances, monitors rewards
2. **Trading Mode** (balance >= $20): Evolution loop scales trade size by 1% per profitable cycle
3. **Failure Learning**: Any loss > $0.001 is recorded in `evolutionary_memory.json` as a forbidden pattern; Ollama proposes safer alternatives

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BINANCE_API_KEY` | - | Binance API key |
| `BINANCE_API_SECRET` | - | Binance API secret |
| `CF_TOKEN` | - | Cloudflare tunnel token |
| `HF_TOKEN` | - | Hugging Face token |
| `OLLAMA_MODEL` | `qwen2.5-coder:1.5b` | Local reasoning model |
| `ACCUMULATION_THRESHOLD` | `20.0` | USD threshold to switch modes |
| `SCALING_FACTOR` | `1.01` | Trade size multiplier per profitable cycle |

## Shared Library

The `shared_lib/` directory links to core modules from `emerald-engine`:
- `ParadigmAgentBase` - Base agent class
- `SystemTelemetry` - Telemetry and monitoring
- `EmeraldCryptoVault` - AES-CTR encrypted state storage

## Deployment

Push to GitHub main triggers auto-deploy to Hugging Face Spaces.
