import asyncio
import os
import json
import time
import hmac
import hashlib as hl
import logging
from pathlib import Path
from datetime import datetime, timezone
from typing import Optional

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger("MicroEarning")

BASE_DIR = Path(__file__).parent
NOTIF_PATH = BASE_DIR / "notifications.json"
STATE_PATH = BASE_DIR / "accumulation_state.json"

DUST_CHECK_INTERVAL = 3600
EARN_CHECK_INTERVAL = 7200
REWARDS_CHECK_INTERVAL = 86400

BINANCE_API = "https://api.binance.com"
API_KEY = os.getenv("BINANCE_API_KEY", "")
API_SECRET = os.getenv("BINANCE_API_SECRET", "")


def _load_json(path: Path) -> dict:
    if path.exists() and path.stat().st_size > 0:
        try:
            return json.loads(path.read_text())
        except (json.JSONDecodeError, ValueError):
            return {}
    return {}


def _save_json(path: Path, data):
    path.write_text(json.dumps(data, indent=2, default=str))


def _signed_headers(query: str) -> dict:
    sig = hmac.new(API_SECRET.encode(), query.encode(), hl.sha256).hexdigest()
    return {
        "X-MBX-APIKEY": API_KEY,
        "Content-Type": "application/json",
    }, f"{query}&signature={sig}"


def _notify(category: str, message: str):
    notif = _load_json(NOTIF_PATH)
    if "notifications" not in notif:
        notif["notifications"] = []
    notif["notifications"].append({
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "category": category,
        "message": message,
    })
    _save_json(NOTIF_PATH, notif)
    logger.info(f"[NOTIFICATION] {category}: {message}")


async def _binance_post(session, endpoint: str, params: dict = None) -> Optional[dict]:
    from human_mimicry import mimicked_request
    ts = int(time.time() * 1000)
    query = f"timestamp={ts}"
    if params:
        query += "&" + "&".join(f"{k}={v}" for k, v in params.items())
    headers, signed_query = _signed_headers(query)
    url = f"{BINANCE_API}{endpoint}?{signed_query}"
    return await mimicked_request(session, "POST", url, headers=headers)


async def _binance_get(session, endpoint: str, params: dict = None) -> Optional[dict]:
    from human_mimicry import mimicked_request
    ts = int(time.time() * 1000)
    query = f"timestamp={ts}"
    if params:
        query += "&" + "&".join(f"{k}={v}" for k, v in params.items())
    headers, signed_query = _signed_headers(query)
    url = f"{BINANCE_API}{endpoint}?{signed_query}"
    return await mimicked_request(session, "GET", url, headers=headers)


async def check_dust(session) -> float:
    try:
        result = await _binance_post(session, "/sapi/v1/asset/dust-btc")
        if result and "totalTransferBtc" in result:
            total_btc = float(result["totalTransferBtc"])
            logger.info(f"Dust conversion: {total_btc:.8f} BTC worth of dust available")
            return total_btc
        return 0.0
    except Exception as e:
        logger.warning(f"Dust check error: {e}")
        return 0.0


async def convert_dust(session) -> bool:
    try:
        result = await _binance_post(session, "/sapi/v1/asset/dust-btc")
        if result and result.get("totalServiceCharge") is not None:
            logger.info(f"Dust converted to BNB. Service charge: {result['totalServiceCharge']}")
            _notify("dust_conversion", f"Dust converted successfully. Charge: {result['totalServiceCharge']}")
            return True
        logger.info("No dust to convert or conversion unavailable")
        return False
    except Exception as e:
        logger.warning(f"Dust conversion error: {e}")
        return False


async def check_simple_earn(session) -> dict:
    result = {"flexible": 0.0, "locked": 0.0}
    try:
        flex = await _binance_get(session, "/sapi/v1/simple-earn/flexible/position")
        if flex and "rows" in flex:
            for row in flex["rows"]:
                result["flexible"] += float(row.get("totalAmount", 0))
        locked = await _binance_get(session, "/sapi/v1/simple-earn/locked/position")
        if locked and "rows" in locked:
            for row in locked["rows"]:
                result["locked"] += float(row.get("totalAmount", 0))
        logger.info(f"Simple Earn: flex=${result['flexible']:.4f} locked=${result['locked']:.4f}")
    except Exception as e:
        logger.warning(f"Simple Earn check error: {e}")
    return result


async def subscribe_idle_balances(session, free_balances: dict):
    try:
        for asset, amount in free_balances.items():
            if float(amount) > 0.001:
                params = {"product": "FLEXIBLE", "asset": asset, "amount": str(amount)}
                result = await _binance_post(session, "/sapi/v1/simple-earn/flexible/subscribe", params)
                if result and result.get("success"):
                    logger.info(f"Subscribed {amount} {asset} to Simple Earn")
    except Exception as e:
        logger.warning(f"Simple Earn subscription error: {e}")


async def check_rewards_hub(session) -> list:
    rewards = []
    try:
        result = await _binance_get(session, "/sapi/v1/asset/wallet/balance")
        if result:
            logger.info("Rewards Hub check completed")
    except Exception as e:
        logger.warning(f"Rewards Hub check error: {e}")
    return rewards


async def accumulation_cycle():
    if not API_KEY or not API_SECRET:
        logger.warning("Binance API credentials not set. Running in simulation mode.")
        return

    import aiohttp
    connector = aiohttp.TCPConnector(limit=2, force_close=True)
    async with aiohttp.ClientSession(connector=connector) as session:
        dust_btc = await check_dust(session)
        if dust_btc > 0:
            await convert_dust(session)
        earn = await check_simple_earn(session)
        rewards = await check_rewards_hub(session)
        state = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "dust_btc": dust_btc,
            "earn_flexible": earn["flexible"],
            "earn_locked": earn["locked"],
            "total_saved": earn["flexible"] + earn["locked"],
        }
        _save_json(STATE_PATH, state)
        logger.info(f"Accumulation state saved: ${state['total_saved']:.4f} total")


async def main():
    logger.info("Micro-Earning Engine starting")
    logger.info(f"Dust check: every {DUST_CHECK_INTERVAL}s | Earn: every {EARN_CHECK_INTERVAL}s")
    last_dust = 0
    last_earn = 0
    last_rewards = 0
    while True:
        now = time.time()
        if now - last_dust >= DUST_CHECK_INTERVAL:
            await accumulation_cycle()
            last_dust = now
        if now - last_earn >= EARN_CHECK_INTERVAL:
            last_earn = now
        if now - last_rewards >= REWARDS_CHECK_INTERVAL:
            last_rewards = now
        await asyncio.sleep(60)


if __name__ == "__main__":
    asyncio.run(main())
