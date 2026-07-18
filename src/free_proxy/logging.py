from __future__ import annotations

import json
import logging
import threading
from datetime import UTC, datetime, timedelta
from typing import Any

from free_proxy.config import Settings


class JsonLogStore(logging.Handler):
    def __init__(self, settings: Settings) -> None:
        super().__init__()
        self._logs_dir = settings.data_dir / "logs"
        self._logs_dir.mkdir(parents=True, exist_ok=True)
        self._lock = threading.RLock()
        self._last_cleanup: datetime | None = None

    def emit(self, record: logging.LogRecord) -> None:
        timestamp = datetime.now(UTC)
        entry = {
            "timestamp": timestamp.astimezone().strftime("%Y-%m-%d %H:%M:%S"),
            "level": record.levelname,
            "module": record.name.removeprefix("free_proxy.") or "free_proxy",
            "message": record.getMessage(),
        }
        path = self._logs_dir / f"{timestamp.astimezone():%Y-%m-%d}.json"
        try:
            with self._lock, path.open("a", encoding="utf-8") as handle:
                handle.write(json.dumps(entry, ensure_ascii=False) + "\n")
            self.cleanup()
        except OSError:
            self.handleError(record)

    def read(
        self,
        *,
        date: str | None = None,
        level: str | None = None,
        module: str | None = None,
        limit: int = 1000,
    ) -> list[dict[str, Any]]:
        selected_date = date or f"{datetime.now().astimezone():%Y-%m-%d}"
        try:
            datetime.strptime(selected_date, "%Y-%m-%d")
        except ValueError:
            return []
        path = self._logs_dir / f"{selected_date}.json"
        if not path.exists():
            return []
        entries: list[dict[str, Any]] = []
        with self._lock, path.open(encoding="utf-8") as handle:
            for line in handle:
                try:
                    value = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if isinstance(value, dict):
                    if level and str(value.get("level", "")).upper() != level.upper():
                        continue
                    if module and module.lower() not in str(value.get("module", "")).lower():
                        continue
                    entries.append(value)
        return entries[-limit:]

    def read_today(self) -> list[dict[str, Any]]:
        return self.read()

    def cleanup(self) -> None:
        now = datetime.now(UTC)
        if self._last_cleanup is not None and now - self._last_cleanup < timedelta(hours=1):
            return
        self._last_cleanup = now
        cutoff = now.astimezone().date() - timedelta(days=3)
        with self._lock:
            for path in self._logs_dir.glob("*.json"):
                try:
                    file_date = datetime.strptime(path.stem, "%Y-%m-%d").date()
                except ValueError:
                    continue
                if file_date <= cutoff:
                    path.unlink(missing_ok=True)


def configure_logging(settings: Settings) -> JsonLogStore:
    store = JsonLogStore(settings)
    logger = logging.getLogger("free_proxy")
    logger.setLevel(logging.INFO)
    for handler in list(logger.handlers):
        if isinstance(handler, JsonLogStore):
            logger.removeHandler(handler)
            handler.close()
    logger.addHandler(store)
    return store
