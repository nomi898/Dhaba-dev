"""menu-api: serves the restaurant menu from PostgreSQL with a Redis cache."""

from __future__ import annotations

import logging
from collections.abc import Callable
from contextlib import asynccontextmanager

import redis
from fastapi import FastAPI, Path, Request, Response
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

from .config import Settings, get_settings
from .service import MenuService

log = logging.getLogger("menu-api")

HealthCheck = Callable[[], None]


def error_response(status: int, code: str, message: str) -> JSONResponse:
    return JSONResponse(status_code=status, content={"error": {"code": code, "message": message}})


def create_app(
    service: MenuService | None = None,
    health_checks: dict[str, HealthCheck] | None = None,
    settings: Settings | None = None,
) -> FastAPI:
    settings = settings or get_settings()
    state: dict[str, object] = {"service": service, "checks": health_checks or {}}

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        resources = []
        if state["service"] is None:
            resources = _build_runtime(settings, state)
        yield
        for close in reversed(resources):
            try:
                close()
            except Exception:  # pragma: no cover - best effort shutdown
                log.exception("error during shutdown")

    app = FastAPI(title="Dhaba Orders menu-api", version=settings.app_version, lifespan=lifespan)

    def svc() -> MenuService:
        return state["service"]  # type: ignore[return-value]

    @app.exception_handler(StarletteHTTPException)
    async def http_error(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        code = "not_found" if exc.status_code == 404 else "http_error"
        return error_response(exc.status_code, code, str(exc.detail))

    @app.exception_handler(RequestValidationError)
    async def validation_error(_: Request, exc: RequestValidationError) -> JSONResponse:
        first = exc.errors()[0] if exc.errors() else {}
        loc = ".".join(str(p) for p in first.get("loc", []))
        return error_response(422, "validation_error", f"{loc}: {first.get('msg', 'invalid')}")

    @app.exception_handler(Exception)
    async def unhandled(_: Request, exc: Exception) -> JSONResponse:
        log.exception("unhandled error: %s", exc)
        return error_response(500, "internal_error", "internal server error")

    @app.get("/api/menu")
    def list_menu(response: Response) -> dict:
        menu, hit = svc().get_menu()
        response.headers["X-Cache"] = "HIT" if hit else "MISS"
        return {"data": menu}

    @app.get("/api/menu/{item_id}")
    def get_item(response: Response, item_id: int = Path(ge=1, le=2_147_483_647)) -> dict:
        item, hit = svc().get_item(item_id)
        if item is None:
            return error_response(404, "not_found", f"menu item {item_id} not found")
        response.headers["X-Cache"] = "HIT" if hit else "MISS"
        return {"data": item}

    @app.get("/health")
    def health() -> JSONResponse:
        checks: dict[str, str] = {}
        healthy = True
        for name, check in state["checks"].items():  # type: ignore[union-attr]
            try:
                check()
                checks[name] = "ok"
            except Exception as exc:
                healthy = False
                checks[name] = f"error: {exc.__class__.__name__}"
        body = {
            "status": "ok" if healthy else "degraded",
            "service": "menu-api",
            "version": settings.app_version,
            "checks": checks,
        }
        return JSONResponse(status_code=200 if healthy else 503, content=body)

    return app


def _build_runtime(settings: Settings, state: dict[str, object]) -> list[Callable[[], None]]:
    from psycopg_pool import ConnectionPool

    from .cache import MenuCache
    from .listener import MenuChangeListener
    from .repository import PostgresMenuRepository

    pool = ConnectionPool(
        settings.postgres_conninfo,
        min_size=settings.db_pool_min,
        max_size=settings.db_pool_max,
        open=True,
        timeout=10,
    )
    redis_client = redis.Redis(
        host=settings.redis_host,
        port=settings.redis_port,
        password=settings.redis_password or None,
        db=settings.redis_db,
        socket_timeout=2,
        socket_connect_timeout=2,
        health_check_interval=30,
    )
    repo = PostgresMenuRepository(pool)
    cache = MenuCache(redis_client, settings.menu_cache_ttl_seconds)
    service = MenuService(repo, cache)
    listener = MenuChangeListener(settings.postgres_conninfo, service.invalidate)
    listener.start()

    state["service"] = service
    state["checks"] = {"postgres": repo.ping, "redis": cache.ping}
    log.info(
        "menu-api %s started (cache ttl %ss)", settings.app_version, settings.menu_cache_ttl_seconds
    )
    return [pool.close, redis_client.close, listener.stop]


logging.basicConfig(
    level=get_settings().log_level.upper(),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)

app = create_app()
