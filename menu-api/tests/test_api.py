import pytest
from fastapi.testclient import TestClient

from app.main import create_app


@pytest.fixture
def client(service, repo, cache, settings):
    app = create_app(
        service=service,
        health_checks={"postgres": repo.ping, "redis": cache.ping},
        settings=settings,
    )
    with TestClient(app) as c:
        yield c


def test_get_menu(client):
    r = client.get("/api/menu")
    assert r.status_code == 200
    assert r.headers["X-Cache"] == "MISS"
    assert r.json()["data"][0]["items"][0]["priceCents"] == 35000
    assert client.get("/api/menu").headers["X-Cache"] == "HIT"


def test_get_item(client):
    r = client.get("/api/menu/10")
    assert r.status_code == 200
    assert r.json()["data"]["category"]["slug"] == "starters"


def test_get_item_not_found(client):
    r = client.get("/api/menu/12345")
    assert r.status_code == 404
    assert r.json() == {"error": {"code": "not_found", "message": "menu item 12345 not found"}}


@pytest.mark.parametrize("bad", ["abc", "0", "-1"])
def test_get_item_invalid_id(client, bad):
    r = client.get(f"/api/menu/{bad}")
    assert r.status_code == 422
    assert r.json()["error"]["code"] == "validation_error"


def test_unknown_route_uses_error_envelope(client):
    r = client.get("/api/nope")
    assert r.status_code == 404
    assert r.json()["error"]["code"] == "not_found"


def test_health_ok(client):
    r = client.get("/health")
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert body["checks"] == {"postgres": "ok", "redis": "ok"}


def test_health_degraded_when_db_down(client, repo):
    repo.fail_ping = True
    r = client.get("/health")
    assert r.status_code == 503
    assert r.json()["checks"]["postgres"].startswith("error")
