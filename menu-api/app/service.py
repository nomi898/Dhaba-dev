"""Menu read logic: cache-aside over the repository."""

from __future__ import annotations

from typing import Any

from .cache import MenuCache
from .repository import MenuRepository


class MenuService:
    def __init__(self, repository: MenuRepository, cache: MenuCache) -> None:
        self._repo = repository
        self._cache = cache

    def get_menu(self) -> tuple[list[dict[str, Any]], bool]:
        """Return (menu, served_from_cache)."""
        cached = self._cache.get_menu()
        if cached is not None:
            return cached, True
        menu = self._repo.fetch_menu()
        self._cache.set_menu(menu)
        return menu, False

    def get_item(self, item_id: int) -> tuple[dict[str, Any] | None, bool]:
        cached = self._cache.get_item(item_id)
        if cached is not None:
            return cached, True
        item = self._repo.fetch_item(item_id)
        if item is not None:
            self._cache.set_item(item_id, item)
        return item, False

    def invalidate(self) -> None:
        self._cache.invalidate()
