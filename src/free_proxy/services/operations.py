from __future__ import annotations

import asyncio
from collections.abc import AsyncIterator, Awaitable, Callable
from contextlib import asynccontextmanager
from contextvars import ContextVar
from dataclasses import dataclass
from typing import TypeVar
from uuid import uuid4

from free_proxy.domain.exceptions import FreeProxyError

T = TypeVar("T")


class OperationConflictError(FreeProxyError):
    """Raised when a mutually exclusive network operation is already running."""


@dataclass(frozen=True, slots=True)
class OperationSnapshot:
    operation: str | None
    token: str | None
    started_at: float | None
    waiting: int


class NetworkOperationCoordinator:
    """Serialize network-changing work while allowing trusted nested calls.

    A context-local token lets maintenance call probe and auto-switch without
    deadlocking itself, while unrelated HTTP jobs receive an explicit conflict.
    """

    def __init__(self) -> None:
        self._lock = asyncio.Lock()
        self._current: tuple[str, str, float] | None = None
        self._waiting = 0
        self._token: ContextVar[str | None] = ContextVar("network_operation_token", default=None)

    @property
    def snapshot(self) -> OperationSnapshot:
        current = self._current
        return OperationSnapshot(
            current[0] if current else None,
            current[1] if current else None,
            current[2] if current else None,
            self._waiting,
        )

    async def run(
        self,
        operation: str,
        callback: Callable[[], Awaitable[T]],
        *,
        wait: bool = False,
    ) -> T:
        async with self.acquire(operation, wait=wait):
            return await callback()

    @asynccontextmanager
    async def acquire(
        self,
        operation: str,
        *,
        token: str | None = None,
        wait: bool = False,
    ) -> AsyncIterator[str]:
        inherited = self._token.get()
        if inherited is not None:
            yield inherited
            return
        token = token or uuid4().hex
        if self._lock.locked() and not wait:
            raise OperationConflictError(
                f"Network operation '{self._current[0] if self._current else 'unknown'}' is running"
            )
        self._waiting += 1
        try:
            await self._lock.acquire()
        finally:
            self._waiting -= 1
        self._current = (operation, token, asyncio.get_running_loop().time())
        reset = self._token.set(token)
        try:
            yield token
        finally:
            self._token.reset(reset)
            self._current = None
            self._lock.release()
