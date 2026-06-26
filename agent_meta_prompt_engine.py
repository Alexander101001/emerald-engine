import asyncio
import json
import logging
import os
import hashlib
import time
from pathlib import Path
from typing import Optional

from opcode_base import ParadigmAgentBase

META_PROMPT_INTERVAL = int(os.getenv("META_PROMPT_INTERVAL", "3600"))
PROMPT_STORE = Path(os.getenv("PROMPT_STORE", ".aegis/prompts"))
AGENT_REGISTRY = Path(os.getenv("AGENT_REGISTRY", ".aegis/registry.json"))

LEVELS = {
    1: "INPUT",
    2: "ARCHITECTURAL",
    3: "INTERACTION",
    4: "ASI_EVOLUTION",
    5: "TOOLING",
}

AXES = [
    "SELF_EVOLUTION",
    "HACKER_GRADE_SECURITY",
    "TINY_WOW_SAAS_FEATURES",
    "EXPANSION_NETWORKING",
    "INFRASTRUCTURE_COST_CONTROL",
]

METHODS_50 = [
    "few_shot", "chain_of_thought", "tree_of_thought", "self_consistency",
    "reflexion", "self_critique", "meta_cognition", "goal_decomposition",
    "recursive_summarization", "active_recall", "contrastive_learning",
    "multi_view_reasoning", "abstraction_hierarchy", "analogical_reasoning",
    "counterfactual_reasoning", "causal_inference", "information_bottleneck",
    "attention_steering", "context_compression", "semantic_chunking",
    "role_play_priming", "persona_stabilization", "emotional_appeal",
    "authority_framing", "social_proof_injection", "urgency_calibration",
    "reward_shaping", "curriculum_ordering", "difficulty_scheduling",
    "self_play_generation", "adversarial_robustness", "noise_injection",
    "distillation_targeting", "sparse_expert_routing", "mixture_of_prompts",
    "adaptive_length_control", "temperature_scheduling", "top_k_top_p_decay",
    "beam_search_prompting", "contrastive_search", "speculative_decoding",
    "tool_augmented_reasoning", "code_as_action", "api_orchestration",
    "multi_modal_fusion", "latent_space_navigation", "embedding_alignment",
    "knowledge_graph_pruning", "memory_retrieval_augmentation",
    "continual_adaptation",
]

AGENT_TEMPLATES = {
    "agent_orchestrator": {
        "role": "Orchestrator",
        "section": 0,
        "description": "Central 66-agent fleet coordinator",
        "keywords": ["coordination", "scheduling", "load_balancing", "state_machine"],
    },
    "agent_hacker_bot": {
        "role": "Hacker Bot",
        "section": 1,
        "description": "Zero-trust security monitor and penetration testing",
        "keywords": ["security", "audit", "crypto", "vulnerability", "zero_trust"],
    },
    "agent_expansion": {
        "role": "Expansion Agent",
        "section": 2,
        "description": "Autonomous platform expansion and service discovery",
        "keywords": ["expansion", "discovery", "registration", "protocol"],
    },
    "agent_cloud_manager": {
        "role": "Cloud Manager",
        "section": 3,
        "description": "Multi-cloud resource orchestration and quota management",
        "keywords": ["cloud", "orchestration", "quota", "provisioning"],
    },
    "agent_hunter": {
        "role": "Hunter",
        "section": 4,
        "description": "Platform hunting and ping-based service discovery",
        "keywords": ["hunt", "ping", "discovery", "scan"],
    },
    "agent_orchestrator_enterprise": {
        "role": "Enterprise Orchestrator",
        "section": 5,
        "description": "5-step enterprise operational workflow engine",
        "keywords": ["workflow", "enterprise", "pipeline", "step"],
    },
    "agent_architect": {
        "role": "Software Architect",
        "section": 6,
        "description": "Topology discovery and component analysis",
        "keywords": ["architecture", "discovery", "topology", "analysis"],
    },
    "agent_devsecops": {
        "role": "DevSecOps",
        "section": 7,
        "description": "Vault verification and SOC2/GDPR compliance audit",
        "keywords": ["security", "compliance", "vault", "soc2", "gdpr"],
    },
    "agent_qa": {
        "role": "QA Automation",
        "section": 8,
        "description": "Test generation, syntax validation, regression detection",
        "keywords": ["testing", "validation", "regression", "syntax"],
    },
    "agent_scouter": {
        "role": "Trend Scouter",
        "section": 9,
        "description": "GitHub trending and BM25 candidate ranking",
        "keywords": ["trending", "github", "bm25", "ranking", "scout"],
    },
    "agent_evaluator": {
        "role": "Code Evaluator",
        "section": 10,
        "description": "Ephemeral sandbox evaluation against 161 rules",
        "keywords": ["evaluation", "sandbox", "rules", "verification"],
    },
    "agent_synthesizer": {
        "role": "Code Synthesizer",
        "section": 11,
        "description": "Skill merging, dependency resolution, code synthesis",
        "keywords": ["synthesis", "merge", "refactor", "dependency"],
    },
    "agent_state_relay": {
        "role": "State Relay",
        "section": 12,
        "description": "Resource monitoring, state encryption, branch dispatch",
        "keywords": ["state", "encryption", "monitor", "relay"],
    },
    "agent_runner_grid": {
        "role": "Runner Grid",
        "section": 13,
        "description": "20-runner rotational pool with dual-pipeline execution",
        "keywords": ["runner", "grid", "rotation", "pipeline"],
    },
    "agent_hf_sync": {
        "role": "HF Sync",
        "section": 14,
        "description": "HuggingFace WebSocket, API sync and SLM scanning",
        "keywords": ["huggingface", "sync", "slm", "websocket"],
    },
    "agent_identity": {
        "role": "Identity Manager",
        "section": 15,
        "description": "Fingerprint generation, TLS JA3, token vault management",
        "keywords": ["identity", "fingerprint", "ja3", "token", "vault"],
    },
    "agent_dashboard": {
        "role": "Dashboard Compiler",
        "section": 16,
        "description": "Static dashboard compilation and metric aggregation",
        "keywords": ["dashboard", "metrics", "visualization", "report"],
    },
    "agent_chat": {
        "role": "Chat Interpreter",
        "section": 17,
        "description": "Async command polling and agent routing",
        "keywords": ["chat", "command", "routing", "polling"],
    },
    "agent_git": {
        "role": "Git Lifecycle",
        "section": 18,
        "description": "Auto-commit, branch management, push orchestration",
        "keywords": ["git", "commit", "push", "branch", "lifecycle"],
    },
    "agent_harvester": {
        "role": "Repo Harvester",
        "section": 19,
        "description": "Open-source repo cloning and Ollama-based integration analysis",
        "keywords": ["harvest", "ollama", "integration", "clone", "analysis"],
    },
}


def _fingerprint(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:12]


class MetaPromptEngine(ParadigmAgentBase):
    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0
        self._agent_count = 66
        self._prompt_fingerprints: dict[str, str] = {}
        PROMPT_STORE.mkdir(parents=True, exist_ok=True)

    def _compose_prompt(self, agent_key: str, template: dict, level: int, axis: str) -> str:
        level_name = LEVELS.get(level, "INPUT")
        methods_pool = METHODS_50[(level - 1) * 10:(level * 10)]
        methods_str = ", ".join(methods_pool[:8])

        lines = []
        lines.append(f"You are {template['role']}, one of {self._agent_count} specialized agents in the Emerald Engine fleet.")
        lines.append(f"Section: {template['section']} | Level: {level_name} | Axis: {axis}")
        lines.append(f"Core mandate: {template['description']}")
        lines.append(f"Trigger keywords: {', '.join(template['keywords'])}")
        lines.append(f"")
        lines.append(f"Paradigm {level_name} — apply these recursive learning methods: {methods_str}")
        lines.append(f"Development Axis {axis} — enforce across all outputs.")
        lines.append(f"")
        lines.append(f"Operate with full autonomy. Log decisions via _emit_telemetry. ")
        lines.append(f"Prioritize system uptime, security, and cost efficiency. ")
        lines.append(f"All outputs must be valid JSON where applicable. ")
        lines.append(f"Synthesize state changes through StateRelay for persistence. ")
        lines.append(f"")
        lines.append(f"[auto-generated prompt — fingerprint: {_fingerprint(agent_key + axis + level_name)}]")

        return "\n".join(lines)

    def _select_axis(self, agent_key: str) -> str:
        idx = abs(hash(agent_key)) % len(AXES)
        return AXES[idx]

    def _select_level(self, agent_key: str, cycle: int) -> int:
        return ((cycle + abs(hash(agent_key))) % 5) + 1

    def _save_prompt(self, agent_id: str, prompt: str) -> Path:
        out_path = PROMPT_STORE / f"{agent_id}.txt"
        out_path.write_text(prompt)
        fp = _fingerprint(prompt)
        self._prompt_fingerprints[agent_id] = fp
        return out_path

    def _update_registry(self):
        registry = {}
        if AGENT_REGISTRY.exists():
            try:
                registry = json.loads(AGENT_REGISTRY.read_text())
            except (json.JSONDecodeError, ValueError):
                registry = {}
        registry["last_update"] = time.time()
        registry["cycle"] = self._cycle
        registry["fingerprints"] = self._prompt_fingerprints
        registry["agent_count"] = self._agent_count
        registry["levels_deployed"] = list(LEVELS.values())
        registry["axes_active"] = list(AXES)
        AGENT_REGISTRY.write_text(json.dumps(registry, indent=2))

    async def meta_prompt_cycle(self):
        self._cycle += 1
        logging.info(f"=== Meta-Prompt Engine Cycle #{self._cycle} ===")
        logging.info(f"  Generating prompts for {len(AGENT_TEMPLATES)} agent templates across 5 levels x 5 axes")

        generated = 0
        changed = 0

        for agent_key, template in AGENT_TEMPLATES.items():
            level = self._select_level(agent_key, self._cycle)
            axis = self._select_axis(agent_key)

            prompt = self._compose_prompt(agent_key, template, level, axis)
            agent_id = f"agent_{self._cycle:02d}_{agent_key}"
            out_path = self._save_prompt(agent_id, prompt)

            generated += 1
            old_fp = self._prompt_fingerprints.get(agent_id)
            new_fp = _fingerprint(prompt)
            if old_fp != new_fp:
                changed += 1

            logging.debug(f"    {agent_id} -> {out_path.name} ({len(prompt)} chars, level={LEVELS[level]}, axis={axis})")

        self._update_registry()
        logging.info(f"  Generated: {generated} | Changed: {changed} | Registry: {AGENT_REGISTRY}")

        for required_id in [f"agent_{i:02d}" for i in range(1, 67)]:
            path = PROMPT_STORE / f"{required_id}.txt"
            if not path.exists():
                fallback = f"You are Agent {required_id}. Operate autonomously within the Emerald Engine fleet. Prioritize security, efficiency, and uptime. Report all actions via telemetry."
                path.write_text(fallback)
                self._prompt_fingerprints[required_id] = _fingerprint(fallback)
                logging.info(f"  Created fallback prompt for {required_id}")

        if self._telemetry:
            self._telemetry.record_stream_item()

    async def execution_loop(self):
        logging.info("Meta-Prompt Engine (ASI-Seed Brain) activated.")
        await self.meta_prompt_cycle()
        await self._hot_daemon_loop(self.meta_prompt_cycle, META_PROMPT_INTERVAL)


async def run_meta_prompt_loop(telemetry=None):
    agent = MetaPromptEngine(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    asyncio.run(run_meta_prompt_loop())
