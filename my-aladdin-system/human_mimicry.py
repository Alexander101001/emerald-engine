import asyncio
import random
import time
import logging
from typing import Optional

logger = logging.getLogger("HumanMimicry")

MIN_JITTER_MS = 200
MAX_JITTER_MS = 3000
TYPING_DELAY_MS = 50
MOUSE_MOVE_PROBABILITY = 0.3

_jitter_state = {
    "last_call": 0.0,
    "consecutive_calls": 0,
    "session_calls": 0,
}


def _human_jitter_ms() -> int:
    now = time.time()
    elapsed_ms = (now - _jitter_state["last_call"]) * 1000
    if elapsed_ms < 100:
        _jitter_state["consecutive_calls"] += 1
    else:
        _jitter_state["consecutive_calls"] = 0
    _jitter_state["last_call"] = now
    _jitter_state["session_calls"] += 1

    base_jitter = random.randint(MIN_JITTER_MS, MAX_JITTER_MS)
    if _jitter_state["consecutive_calls"] > 3:
        burst_penalty = random.randint(500, 3000)
        base_jitter += burst_penalty
        logger.debug(f"Burst detected: added {burst_penalty}ms penalty")
    if _jitter_state["session_calls"] % 10 == 0:
        extra_delay = random.randint(1000, 5000)
        base_jitter += extra_delay
        logger.debug(f"Session pause: added {extra_delay}ms")
    return base_jitter


async def human_delay():
    delay_ms = _human_jitter_ms()
    await asyncio.sleep(delay_ms / 1000.0)


async def mimicked_request(session, method: str, url: str, headers: dict = None, json_data: dict = None) -> Optional[dict]:
    await human_delay()
    if random.random() < MOUSE_MOVE_PROBABILITY:
        mouse_delay = random.randint(100, 500)
        await asyncio.sleep(mouse_delay / 1000.0)
    user_agent = (
        f"Mozilla/5.0 (Linux; Android 14; SM-N976B) "
        f"AppleWebKit/537.36 (KHTML, like Gecko) "
        f"Chrome/125.0.6422.146 Mobile Safari/537.36"
    )
    safe_headers = {
        "User-Agent": user_agent,
        "Accept": "application/json",
        "Accept-Language": "en-US,en;q=0.9",
        "Origin": "https://www.binance.com",
        "Referer": "https://www.binance.com/en/trade",
    }
    if headers:
        safe_headers.update(headers)
    try:
        async with session.request(
            method, url, headers=safe_headers, json=json_data, timeout=30
        ) as resp:
            if resp.status == 429:
                retry_after = int(resp.headers.get("Retry-After", "10"))
                logger.warning(f"Rate limited. Waiting {retry_after}s")
                await asyncio.sleep(retry_after + random.uniform(1, 5))
                return None
            if resp.status == 418:
                cooldown = random.randint(60, 300)
                logger.warning(f"IP banned (418). Cooling down {cooldown}s")
                await asyncio.sleep(cooldown)
                return None
            if resp.status >= 200 and resp.status < 300:
                return await resp.json()
            logger.warning(f"Request failed: {resp.status} {method} {url[:60]}")
            return None
    except Exception as e:
        logger.warning(f"Request error: {e}")
        return None
