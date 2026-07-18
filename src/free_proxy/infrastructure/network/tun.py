from __future__ import annotations

import asyncio
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager


class TunAllocator:
    def __init__(self, start: int = 2, end: int = 99) -> None:
        if start > end:
            raise ValueError("TUN allocation start must not exceed end")
        self._start = start
        self._end = end
        self._allocated: set[int] = set()
        self._lock = asyncio.Lock()

    @asynccontextmanager
    async def allocate(self) -> AsyncIterator[str]:
        index = await self._acquire()
        try:
            yield f"tun{index}"
        finally:
            async with self._lock:
                self._allocated.discard(index)

    async def _acquire(self) -> int:
        async with self._lock:
            for index in range(self._start, self._end + 1):
                if index not in self._allocated:
                    self._allocated.add(index)
                    return index
        raise RuntimeError("No test TUN devices are available")
