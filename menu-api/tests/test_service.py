import redis

from app.cache import ITEM_INDEX_KEY, MENU_KEY, MenuCache
from app.service import MenuService


def test_menu_miss_then_hit(service, repo):
    menu, hit = service.get_menu()
    assert hit is False
    assert menu[0]["slug"] == "starters"
    menu2, hit2 = service.get_menu()
    assert hit2 is True
    assert menu2 == menu
    assert repo.menu_calls == 1


def test_menu_cache_has_ttl(service, redis_client):
    service.get_menu()
    ttl = redis_client.ttl(MENU_KEY)
    assert 0 < ttl <= 60


def test_item_cached_and_indexed(service, repo, redis_client):
    item, hit = service.get_item(10)
    assert item["name"] == "Samosa Chaat" and hit is False
    _, hit = service.get_item(10)
    assert hit is True
    assert repo.item_calls == 1
    assert redis_client.sismember(ITEM_INDEX_KEY, "menu:item:10")


def test_missing_item_not_cached(service, repo, redis_client):
    item, _ = service.get_item(999)
    assert item is None
    service.get_item(999)
    assert repo.item_calls == 2
    assert redis_client.get("menu:item:999") is None


def test_invalidate_removes_all_entries(service, repo, redis_client):
    service.get_menu()
    service.get_item(10)
    service.invalidate()
    assert redis_client.get(MENU_KEY) is None
    assert redis_client.get("menu:item:10") is None
    service.get_menu()
    assert repo.menu_calls == 2


def test_corrupt_cache_entry_falls_back_to_db(service, repo, redis_client):
    redis_client.set(MENU_KEY, "{not json")
    menu, hit = service.get_menu()
    assert hit is False and repo.menu_calls == 1 and menu


class BrokenRedis:
    def __getattr__(self, name):
        def fail(*_a, **_k):
            raise redis.ConnectionError("redis down")

        return fail


def test_redis_outage_falls_back_to_db(repo):
    svc = MenuService(repo, MenuCache(BrokenRedis(), ttl_seconds=60))
    menu, hit = svc.get_menu()
    assert menu and hit is False
    svc.invalidate()  # must not raise


def test_invalid_ttl_rejected(redis_client):
    import pytest

    with pytest.raises(ValueError):
        MenuCache(redis_client, ttl_seconds=0)
