import os
import sys
import time
import json
import threading
import requests
import subprocess
from fastapi import FastAPI

app = FastAPI(title="Emerald Omni-Engine Core")

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://ollama_ai:11434")
OLLAMA_URL = f"{OLLAMA_HOST}/api/generate"
OLLAMA_PULL = f"{OLLAMA_HOST}/api/pull"
OLLAMA_TAGS = f"{OLLAMA_HOST}/api/tags"
MODEL_NAME = "qwen2.5-coder:1.5b"

TARGET_REPOS = [
    "https://github.com/crewAI/crewAI",
    "https://github.com/langchain-ai/langgraph",
    "https://github.com/microsoft/autogen",
    "https://github.com/unclecode/crawl4ai",
    "https://github.com/browser-use/browser-use",
]


def init_workspace_guards():
    os.makedirs("backend", exist_ok=True)
    os.makedirs("harvested_repos", exist_ok=True)
    os.makedirs("integrations", exist_ok=True)


@app.get("/health")
def health_check():
    return {"status": "healthy", "engine": "66-agents active"}


def wait_for_ollama():
    print("[Infra] Checking Ollama service availability...")
    while True:
        try:
            response = requests.get(OLLAMA_TAGS, timeout=5)
            if response.status_code == 200:
                print("[Infra] Ollama network service is responsive and online.")
                break
        except requests.exceptions.ConnectionError:
            print("[Warning] Ollama unreachable. Retrying connection in 10 seconds...")
            time.sleep(10)


def autonomous_harvester_loop():
    init_workspace_guards()
    wait_for_ollama()

    print(f"[Ollama] Verifying presence of local model: {MODEL_NAME}")
    try:
        requests.post(OLLAMA_PULL, json={"name": MODEL_NAME}, timeout=600)
    except Exception as e:
        print(f"[Ollama Error] Could not complete model pull sequence: {e}")

    while True:
        print("[Harvester] Initiating structured data harvesting cycle...")
        for repo in TARGET_REPOS:
            repo_name = repo.split("/")[-1]
            target_path = f"harvested_repos/{repo_name}"

            try:
                if not os.path.exists(target_path):
                    subprocess.run(["git", "clone", "--depth", "1", repo, target_path],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                else:
                    subprocess.run(["git", "-C", target_path, "pull"],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            except Exception as e:
                print(f"[Git Error] Skipping repository fetch for {repo_name}: {e}")
                continue

            readme_file = f"{target_path}/README.md"
            context = ""
            if os.path.exists(readme_file):
                try:
                    with open(readme_file, 'r', encoding='utf-8', errors='ignore') as f:
                        context = f.read()[:1000]
                except Exception as e:
                    print(f"[File Error] Read failure on {readme_file}: {e}")

            if not context:
                print(f"[Analysis] Empty target profile for {repo_name}. Moving forward.")
                continue

            prompt = (
                f"Analyze the structural layout for {repo_name}. Data snippet: {context}. "
                f"Provide an automated integration mapping blueprint for this system."
            )

            payload = {"model": MODEL_NAME, "prompt": prompt, "stream": False, "format": "json"}

            try:
                res = requests.post(OLLAMA_URL, json=payload, timeout=90)
                if res.status_code == 200 and res.json():
                    ai_data = res.json().get("response", "{}")
                    with open(f"integrations/{repo_name}_schema.json", "w", encoding='utf-8') as out:
                        out.write(ai_data)
                    print(f"[Success] Mapped telemetry integration configs for: {repo_name}")
                else:
                    print(f"[Warning] Received empty execution context from backend container for {repo_name}")
            except Exception as e:
                print(f"[Pipeline Blocked] AI connection cycle skipped to prevent engine crash: {e}")

            print("[Cycle Complete] Entering regular operation cooldown for 10 minutes.")
        time.sleep(600)


threading.Thread(target=autonomous_harvester_loop, daemon=True).start()
