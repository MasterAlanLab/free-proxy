from __future__ import annotations

import asyncio
import re
import sys


async def measure_node_latency(host: str, port: int, fallback_ping_ms: float = 0) -> int:
    interface = await get_physical_interface()
    if sys.platform.startswith("linux") and interface:
        latency = await run_ping(host, interface=interface)
        if latency > 0:
            return latency
    latency = await run_ping(host)
    if latency > 0:
        return latency
    latency = await measure_tcp_latency(host, port, timeout_seconds=5)
    if latency > 0:
        return latency
    return max(0, int(fallback_ping_ms))


async def get_physical_interface() -> str | None:
    if not sys.platform.startswith("linux"):
        return None
    try:
        process = await asyncio.create_subprocess_exec(
            "ip",
            "route",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.DEVNULL,
        )
        stdout, _ = await asyncio.wait_for(process.communicate(), timeout=2)
    except (FileNotFoundError, OSError, TimeoutError):
        return None
    routes: list[tuple[int, str]] = []
    for line in stdout.decode("utf-8", errors="replace").splitlines():
        if not line.startswith("default"):
            continue
        parts = line.split()
        if "dev" not in parts:
            continue
        try:
            device = parts[parts.index("dev") + 1]
            metric = int(parts[parts.index("metric") + 1]) if "metric" in parts else 0
        except (IndexError, ValueError):
            continue
        if not device.startswith(("tun", "tap", "wg", "ppp")):
            routes.append((metric, device))
    routes.sort()
    return routes[0][1] if routes else None


async def run_ping(host: str, *, interface: str | None = None) -> int:
    command = ["ping", "-c", "1", "-W", "2"]
    if interface:
        command.extend(["-I", interface])
    command.append(host)
    try:
        process = await asyncio.create_subprocess_exec(
            *command,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.DEVNULL,
        )
        stdout, _ = await asyncio.wait_for(process.communicate(), timeout=3)
    except (FileNotFoundError, OSError, TimeoutError):
        return 0
    if process.returncode != 0:
        return 0
    match = re.search(rb"time=([\d.]+)\s*ms", stdout)
    return max(1, int(float(match.group(1)))) if match else 0


async def measure_tcp_latency(host: str, port: int, timeout_seconds: float = 5) -> int:
    if not host or port <= 0:
        return 0
    loop = asyncio.get_running_loop()
    started = loop.time()
    writer: asyncio.StreamWriter | None = None
    try:
        _, writer = await asyncio.wait_for(
            asyncio.open_connection(host, port), timeout=timeout_seconds
        )
        return max(1, int((loop.time() - started) * 1000))
    except (OSError, TimeoutError):
        return 0
    finally:
        if writer is not None:
            writer.close()
            await writer.wait_closed()
