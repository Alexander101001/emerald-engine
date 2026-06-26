import asyncio
import os
import time
import json
import hashlib
import random
import logging
from urllib.parse import quote_plus

import aiohttp
import aiofiles

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger("InfiniteResearchStream")

ARXIV_QUERY = os.getenv("ARXIV_QUERY", "cat:cs.AI+OR+cat:cs.LG+OR+cat:cs.CL")
ANTHROPIC_FEED = os.getenv("ANTHROPIC_FEED", "https://www.anthropic.com/research")
OPENAI_BLOG = os.getenv("OPENAI_BLOG", "https://openai.com/index/")
QDRANT_URL = os.getenv("QDRANT_URL", "http://qdrant:6333")
QDRANT_COLLECTION = os.getenv("QDRANT_COLLECTION", "research_vectors")
MAX_WORKERS = int(os.getenv("STREAM_WORKERS", "4"))
BATCH_SIZE = int(os.getenv("STREAM_BATCH_SIZE", "10"))
PAGE_SIZE = int(os.getenv("ARXIV_PAGE_SIZE", "50"))
OUTPUT_DIR = os.getenv("STREAM_OUTPUT_DIR", "/tmp/research_stream")

os.makedirs(OUTPUT_DIR, exist_ok=True)

ARXIV_BASE = "http://export.arxiv.org/api/query"


def _jitter_backoff(attempt: int, base: float = 1.0, cap: float = 60.0) -> float:
    delay = min(cap, base * (2 ** attempt))
    return delay + random.uniform(0, delay * 0.5)


def _micros() -> str:
    return str(time.time_ns() // 1000)


def _chunk_text(text: str, size: int = 512) -> list[str]:
    words = text.split()
    for i in range(0, len(words), size):
        yield " ".join(words[i:i + size])


async def _fetch_arxiv(session: aiohttp.ClientSession, start: int) -> list[dict]:
    url = (
        f"{ARXIV_BASE}?search_query={quote_plus(ARXIV_QUERY)}"
        f"&start={start}&max_results={PAGE_SIZE}&sortBy=submittedDate&sortOrder=descending"
    )
    headers = {"User-Agent": "EmeraldEngine/5.0 (research_stream)"}
    async with session.get(url, headers=headers, timeout=aiohttp.ClientTimeout(total=30)) as resp:
        if resp.status == 429:
            retry_after = int(resp.headers.get("Retry-After", "30"))
            raise aiohttp.ClientError(f"rate_limited:{retry_after}")
        resp.raise_for_status()
        text = await resp.text()
    papers = []
    import xml.etree.ElementTree as ET
    ns = {"atom": "http://www.w3.org/2005/Atom", "arxiv": "http://arxiv.org/schemas/atom"}
    root = ET.fromstring(text)
    for entry in root.findall("atom:entry", ns):
        paper_id = entry.find("atom:id", ns)
        title = entry.find("atom:title", ns)
        summary = entry.find("atom:summary", ns)
        published = entry.find("atom:published", ns)
        authors = [a.find("atom:name", ns).text for a in entry.findall("atom:author", ns) if a.find("atom:name", ns) is not None]
        link_el = entry.find("atom:link[@title='pdf']", ns)
        pdf_link = link_el.attrib["href"] if link_el is not None else None
        papers.append({
            "id": paper_id.text.strip() if paper_id is not None else "",
            "title": title.text.strip() if title is not None else "",
            "summary": summary.text.strip() if summary is not None else "",
            "published": published.text.strip() if published is not None else "",
            "authors": authors,
            "pdf_url": pdf_link,
            "source": "arxiv",
        })
    return papers


async def _fetch_anthropic(session: aiohttp.ClientSession) -> list[dict]:
    papers = []
    try:
        async with session.get(ANTHROPIC_FEED, timeout=aiohttp.ClientTimeout(total=20)) as resp:
            if resp.status == 429:
                raise aiohttp.ClientError(f"rate_limited:{resp.headers.get('Retry-After', '30')}")
            resp.raise_for_status()
            html = await resp.text()
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(html, "html.parser")
        articles = soup.select("article, .post, [class*=research], [class*=post]")
        for art in articles[:10]:
            heading = art.find(["h1", "h2", "h3", "h4"])
            link = art.find("a")
            title = heading.get_text(strip=True) if heading else ""
            url = link.get("href") if link else ""
            if not title:
                continue
            papers.append({
                "id": hashlib.md5(url.encode()).hexdigest() if url else hashlib.md5(title.encode()).hexdigest(),
                "title": title,
                "summary": art.get_text(strip=True)[:1000],
                "published": "",
                "authors": [],
                "pdf_url": url,
                "source": "anthropic",
            })
    except Exception as e:
        logger.warning(f"Anthropic fetch error: {e}")
    return papers


async def _fetch_openai(session: aiohttp.ClientSession) -> list[dict]:
    papers = []
    try:
        async with session.get(OPENAI_BLOG, timeout=aiohttp.ClientTimeout(total=20)) as resp:
            if resp.status == 429:
                raise aiohttp.ClientError(f"rate_limited:{resp.headers.get('Retry-After', '30')}")
            resp.raise_for_status()
            html = await resp.text()
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(html, "html.parser")
        articles = soup.select("article, .post, [class*=post], [class*=card], [class*=blog]")
        for art in articles[:10]:
            heading = art.find(["h1", "h2", "h3", "h4"])
            link = art.find("a")
            title = heading.get_text(strip=True) if heading else ""
            url = link.get("href") if link else ""
            if not title:
                continue
            papers.append({
                "id": hashlib.md5(url.encode()).hexdigest() if url else hashlib.md5(title.encode()).hexdigest(),
                "title": title,
                "summary": art.get_text(strip=True)[:1000],
                "published": "",
                "authors": [],
                "pdf_url": url,
                "source": "openai",
            })
    except Exception as e:
        logger.warning(f"OpenAI blog fetch error: {e}")
    return papers


async def _upsert_qdrant(vectors: list[dict]):
    if not vectors:
        return
    try:
        url = f"{QDRANT_URL}/collections/{QDRANT_COLLECTION}/points"
        payload = {
            "points": [
                {
                    "id": abs(hash(v["id"])) % (2 ** 63),
                    "vector": v.get("vector", [0.0] * 384),
                    "payload": v.get("payload", {}),
                }
                for v in vectors
            ]
        }
        async with aiohttp.ClientSession() as session:
            async with session.put(url, json=payload, timeout=aiohttp.ClientTimeout(total=10)) as resp:
                if resp.status == 200:
                    logger.info(f"Upserted {len(vectors)} vectors to Qdrant")
                else:
                    logger.warning(f"Qdrant upsert status {resp.status}")
    except Exception as e:
        logger.warning(f"Qdrant upsert error: {e}")


async def _store_local(paper: dict):
    fname = f"{paper['source']}_{hashlib.md5(paper['id'].encode()).hexdigest()[:12]}.json"
    path = os.path.join(OUTPUT_DIR, fname)
    async with aiofiles.open(path, "w") as f:
        await f.write(json.dumps(paper, indent=2))


async def _process_paper(paper: dict, queue: asyncio.Queue):
    micro_ts = _micros()
    chunks = list(_chunk_text(f"{paper['title']} {paper['summary']}", size=384))
    vectors = []
    for i, chunk in enumerate(chunks):
        vectors.append({
            "id": f"{paper['source']}_{paper['id']}_{i}",
            "vector": [float(hash(chunk) % 1000) / 1000.0 for _ in range(384)],
            "payload": {
                "source": paper["source"],
                "title": paper["title"],
                "published": paper["published"],
                "authors": paper["authors"][:3],
                "pdf_url": paper.get("pdf_url", ""),
                "chunk_index": i,
                "chunk_total": len(chunks),
                "stream_origin": "live_evolution",
                "sync_microsecond": micro_ts,
                "ingested_at": int(time.time()),
            },
        })
    await _upsert_qdrant(vectors)
    await _store_local(paper)
    logger.info(f"Processed {paper['source']}: {paper['title'][:60]}... ({len(chunks)} chunks)")


async def _fetch_worker(fetch_queue: asyncio.Queue, process_queue: asyncio.Queue):
    connector = aiohttp.TCPConnector(limit=10, force_close=True)
    async with aiohttp.ClientSession(connector=connector) as session:
        while True:
            task = await fetch_queue.get()
            source = task.get("source", "arxiv")
            attempt = 0
            while True:
                try:
                    if source == "arxiv":
                        papers = await _fetch_arxiv(session, task.get("start", 0))
                    elif source == "anthropic":
                        papers = await _fetch_anthropic(session)
                    elif source == "openai":
                        papers = await _fetch_openai(session)
                    else:
                        papers = []
                    for p in papers:
                        await process_queue.put(p)
                    break
                except aiohttp.ClientError as e:
                    err_str = str(e)
                    if "rate_limited" in err_str:
                        retry_after = int(err_str.split(":")[1])
                        logger.warning(f"Rate limited on {source}, backing off {retry_after}s")
                        await asyncio.sleep(retry_after)
                        attempt += 1
                    else:
                        logger.warning(f"Fetch error {source}: {e}")
                        await asyncio.sleep(5)
                        attempt += 1
                except Exception as e:
                    logger.error(f"Unexpected fetch error {source}: {e}")
                    await asyncio.sleep(10)
                    attempt += 1
            fetch_queue.task_done()


async def _process_worker(process_queue: asyncio.Queue):
    while True:
        paper = await process_queue.get()
        try:
            await _process_paper(paper, process_queue)
        except Exception as e:
            logger.error(f"Process error for {paper.get('id', 'unknown')}: {e}")
        process_queue.task_done()


async def _infinite_generator():
    start = 0
    while True:
        yield {"source": "arxiv", "start": start}
        start += PAGE_SIZE
        yield {"source": "anthropic", "start": 0}
        yield {"source": "openai", "start": 0}
        if start > 1000:
            start = 0


async def _ensure_qdrant_collection():
    try:
        url = f"{QDRANT_URL}/collections/{QDRANT_COLLECTION}"
        payload = {
            "name": QDRANT_COLLECTION,
            "vectors": {"size": 384, "distance": "Cosine"},
        }
        async with aiohttp.ClientSession() as session:
            async with session.put(url, json=payload, timeout=aiohttp.ClientTimeout(total=10)) as resp:
                if resp.status in (200, 201):
                    logger.info(f"Qdrant collection {QDRANT_COLLECTION} ready")
                elif resp.status == 409:
                    logger.info(f"Qdrant collection {QDRANT_COLLECTION} already exists")
                else:
                    logger.warning(f"Qdrant collection creation status {resp.status}")
    except Exception as e:
        logger.warning(f"Qdrant collection check skipped: {e}")


async def main():
    logger.info("Infinite Research Stream Pipeline starting")
    await _ensure_qdrant_collection()

    fetch_queue = asyncio.Queue(maxsize=MAX_WORKERS * 2)
    process_queue = asyncio.Queue(maxsize=MAX_WORKERS * 4)

    fetch_workers = [asyncio.create_task(_fetch_worker(fetch_queue, process_queue)) for _ in range(MAX_WORKERS)]
    process_workers = [asyncio.create_task(_process_worker(process_queue)) for _ in range(MAX_WORKERS)]

    logger.info(f"Launched {MAX_WORKERS} fetch workers, {MAX_WORKERS} process workers")

    attempt = 0
    async for task in _infinite_generator():
        await fetch_queue.put(task)
        attempt += 1
        if attempt % 10 == 0:
            logger.info(f"Queue status: fetch={fetch_queue.qsize()}, process={process_queue.qsize()}")

    await asyncio.gather(*fetch_workers, *process_workers)


if __name__ == "__main__":
    asyncio.run(main())
