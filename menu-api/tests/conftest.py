from __future__ import annotations

from typing import Any

import fakeredis
import pytest

from app.cache import MenuCache
from app.config import Settings
from app.service import MenuService

SAMPLE_MENU = [
    {
        "id": 1,
        "name": "Starters",
        "slug": "starters",
        "items": [
            {
                "id": 10,
                "name": "Samosa Chaat",
                "description": "crispy",
                "priceCents": 35000,
                "isAvailable": True,
                "isVegetarian": True,
                "spiceLevel": 1,
            },
        ],
    }
]


class FakeRepository:
    def __init__(self) -> None:
        self.menu_calls = 0
        self.item_calls = 0
        self.items: dict[int, dict[str, Any]] = {
            10: {
                **SAMPLE_MENU[0]["items"][0],
                "category": {"id": 1, "name": "Starters", "slug": "starters"},
            },
        }
        self.fail_ping = False

    def fetch_menu(self) -> list[dict[str, Any]]:
        self.menu_calls += 1
        return SAMPLE_MENU

    def fetch_item(self, item_id: int) -> dict[str, Any] | None:
        self.item_calls += 1
        return self.items.get(item_id)

    def ping(self) -> None:
        if self.fail_ping:
            raise ConnectionError("db down")


@pytest.fixture
def redis_client() -> fakeredis.FakeRedis:
    return fakeredis.FakeRedis()


@pytest.fixture
def repo() -> FakeRepository:
    return FakeRepository()


@pytest.fixture
def cache(redis_client: fakeredis.FakeRedis) -> MenuCache:
    return MenuCache(redis_client, ttl_seconds=60)


@pytest.fixture
def service(repo: FakeRepository, cache: MenuCache) -> MenuService:
    return MenuService(repo, cache)


@pytest.fixture
def settings() -> Settings:
    return Settings(app_version="test")
