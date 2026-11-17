"""Redis cache for menu responses. Cache failures never break reads."""

from __future__ import annotations

import json
import logging
from typing import Any

import redis

log = logging.getLogger(__name__)

MENU_KEY = "menu:all"
ITEM_KEY_PREFIX = "menu:item:"
ITEM_INDEX_KEY = "menu:item-keys"


class MenuCache:
    def __init__(self, client: redis.Redis, ttl_seconds: int) -> None:
        if ttl_seconds <= 0:
            raise ValueError("ttl_seconds must be positive")
        self._redis = client
        self._ttl = ttl_seconds

    @property
    def ttl(self) -> int:
        return self._ttl

    def _get(self, key: str) -> Any | None:
        try:
            raw = self._redis.get(key)
        except redis.RedisError as exc:
            log.warning("cache read failed for %s: %s", key, exc)
            return None
        if raw is None:
            return None
        try:
            return json.loads(raw)
        except (TypeError, ValueError):
            log.warning("discarding corrupt cache entry %s", key)
            self._safe_delete(key)
            return None

    def _set(self, key: str, value: Any) -> None:
        try:
            pipe = self._redis.pipeline()
            pipe.set(key, json.dumps(value), ex=self._ttl)
            if key.startswith(ITEM_KEY_PREFIX):
                pipe.sadd(ITEM_INDEX_KEY, key)
                pipe.expire(ITEM_INDEX_KEY, self._ttl * 2)
            pipe.execute()
        except redis.RedisError as exc:
            log.warning("cache write failed for %s: %s", key, exc)

    def _safe_delete(self, *keys: str) -> None:
        try:
            if keys:
                self._redis.delete(*keys)
        except redis.RedisError as exc:
            log.warning("cache delete failed: %s", exc)

    def get_menu(self) -> list[dict[str, Any]] | None:
        return self._get(MENU_KEY)

    def set_menu(self, menu: list[dict[str, Any]]) -> None:
        self._set(MENU_KEY, menu)

    def get_item(self, item_id: int) -> dict[str, Any] | None:
        return self._get(f"{ITEM_KEY_PREFIX}{item_id}")

    def set_item(self, item_id: int, item: dict[str, Any]) -> None:
        self._set(f"{ITEM_KEY_PREFIX}{item_id}", item)

    def invalidate(self) -> None:
        """Remove every cached menu entry."""
        try:
            item_keys = [
                k.decode() if isinstance(k, bytes) else k
                for k in self._redis.smembers(ITEM_INDEX_KEY)
            ]
            self._redis.delete(MENU_KEY, ITEM_INDEX_KEY, *item_keys)
            log.info("menu cache invalidated (%d item keys)", len(item_keys))
        except redis.RedisError as exc:
            log.error("menu cache invalidation failed: %s", exc)

    def ping(self) -> None:
        self._redis.ping()
