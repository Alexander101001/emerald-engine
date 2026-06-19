---
title: Emerald Engine
emoji: 🤖
colorFrom: green
colorTo: gray
sdk: docker
app_port: 8080
pinned: false
---

# Emerald Engine 🤖

## Autonomous Monetization Engine

Generates 29 monetized pages every 5 minutes with:
- Google AdSense + Amazon Associates + Stripe
- 8 interactive micro tools (calculators)
- 7 trading platform pages (Binance)
- Auto-deploy to GitHub Pages
- Telegram notifications
- LLM content from 5 providers (Groq, Together, OpenAI, Claude, OpenRouter)

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `MASTER_KEY` | ✅ Yes | Decrypts the encrypted vault containing all API keys |

Set `MASTER_KEY` as a **Space Secret** in HF Space settings.

## How It Works

1. Engine decrypts vault at startup using MASTER_KEY
2. Every 5 minutes: picks a random niche, generates 29 pages
3. Each page includes ads, affiliate links, email capture, crypto donations
4. Pages are served via the built-in web server on port 8080
5. Trading platform runs alongside with Binance API integration

## License

Proprietary — All rights reserved
