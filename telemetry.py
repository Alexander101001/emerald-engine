import asyncio
import json
import os
import platform
import time
from pathlib import Path

TELEMETRY_FILE = Path("/tmp/telemetry.json")

class SystemTelemetry:
    def __init__(self):
        self.metrics = {
            "cpu_percent": 0.0,
            "memory_used_mb": 0,
            "memory_total_mb": 0,
            "memory_percent": 0.0,
            "disk_used_gb": 0,
            "disk_total_gb": 0,
            "disk_percent": 0.0,
            "uptime_seconds": 0,
            "app_requests": 0,
            "app_errors": 0,
            "avg_response_ms": 0.0,
            "last_cycle_ms": 0.0,
            "stream_items_processed": 0,
            "agent_events_count": 0,
            "timestamp": 0.0,
        }
        self._start = time.time()
        self._request_times = []
        self._errors = 0
        self._requests = 0
        self._stream_count = 0
        self._agent_events = []

    def record_request(self, duration_ms: float):
        self._requests += 1
        self._request_times.append(duration_ms)
        if len(self._request_times) > 1000:
            self._request_times.pop(0)

    def record_error(self):
        self._errors += 1

    def record_stream_item(self):
        self._stream_count += 1

    def record_agent_event(self, agent: str, event: str, **kwargs):
        self._agent_events.append({
            "agent": agent, "event": event, "time": time.time(), **kwargs,
        })
        if len(self._agent_events) > 500:
            self._agent_events = self._agent_events[-500:]

    def record_cycle(self, duration_ms: float):
        self.metrics["last_cycle_ms"] = duration_ms

    def snapshot(self) -> dict:
        try:
            cpu = self._read_cpu()
            mem = self._read_memory()
            disk = self._read_disk()
        except Exception:
            cpu, mem, disk = 0.0, {}, {}

        avg_rt = sum(self._request_times) / len(self._request_times) if self._request_times else 0.0

        self.metrics.update({
            "cpu_percent": cpu,
            "memory_used_mb": mem.get("used_mb", 0),
            "memory_total_mb": mem.get("total_mb", 0),
            "memory_percent": mem.get("percent", 0.0),
            "disk_used_gb": disk.get("used_gb", 0),
            "disk_total_gb": disk.get("total_gb", 0),
            "disk_percent": disk.get("percent", 0.0),
            "uptime_seconds": time.time() - self._start,
            "app_requests": self._requests,
            "app_errors": self._errors,
            "avg_response_ms": round(avg_rt, 2),
            "stream_items_processed": self._stream_count,
            "agent_events_count": len(self._agent_events),
            "agent_events": self._agent_events[-50:],
            "timestamp": time.time(),
        })
        return dict(self.metrics)

    def _read_cpu(self) -> float:
        try:
            with open("/proc/stat") as f:
                line = f.readline()
            parts = line.strip().split()
            if len(parts) < 5:
                return 0.0
            user = int(parts[1])
            nice = int(parts[2])
            system = int(parts[3])
            idle = int(parts[4])
            total = user + nice + system + idle
            prev = getattr(self, "_prev_cpu", None)
            setattr(self, "_prev_cpu", (total, idle))
            if prev:
                d_total = total - prev[0]
                d_idle = idle - prev[1]
                if d_total:
                    return round(100.0 * (1 - d_idle / d_total), 1)
            return 0.0
        except Exception:
            return 0.0

    def _read_memory(self) -> dict:
        try:
            with open("/proc/meminfo") as f:
                lines = f.readlines()
            total = int([l for l in lines if "MemTotal" in l][0].split()[1]) // 1024
            avail = int([l for l in lines if "MemAvailable" in l][0].split()[1]) // 1024
            used = total - avail
            return {
                "total_mb": total,
                "used_mb": used,
                "percent": round(100.0 * used / total, 1) if total else 0,
            }
        except Exception:
            return {}

    def _read_disk(self) -> dict:
        try:
            s = os.statvfs("/")
            total = s.f_frsize * s.f_blocks
            free = s.f_frsize * s.f_bfree
            used = total - free
            return {
                "total_gb": round(total / (1024**3), 1),
                "used_gb": round(used / (1024**3), 1),
                "percent": round(100.0 * used / total, 1) if total else 0,
            }
        except Exception:
            return {}

    def dump(self):
        snap = self.snapshot()
        TELEMETRY_FILE.write_text(json.dumps(snap, indent=2))
        return snap

    def load(self) -> dict:
        try:
            return json.loads(TELEMETRY_FILE.read_text())
        except Exception:
            return self.snapshot()


async def telemetry_loop(telemetry: SystemTelemetry, interval: int = 60):
    while True:
        telemetry.dump()
        await asyncio.sleep(interval)
