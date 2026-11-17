"""Runtime configuration, read entirely from environment variables."""

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=None, extra="ignore")

    app_version: str = "1.0.0"
    log_level: str = "INFO"

    postgres_host: str = "localhost"
    postgres_port: int = 5432
    postgres_db: str = "dhaba"
    postgres_user: str = "dhaba"
    postgres_password: str = ""
    db_pool_min: int = 1
    db_pool_max: int = 10

    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_password: str = ""
    redis_db: int = 0

    menu_cache_ttl_seconds: int = 300

    @property
    def postgres_conninfo(self) -> str:
        parts = {
            "host": self.postgres_host,
            "port": str(self.postgres_port),
            "dbname": self.postgres_db,
            "user": self.postgres_user,
            "password": self.postgres_password,
            "application_name": "menu-api",
            "connect_timeout": "5",
        }
        return " ".join(f"{k}={_quote(v)}" for k, v in parts.items() if v != "")


def _quote(value: str) -> str:
    escaped = value.replace("\\", "\\\\").replace("'", "\\'")
    return f"'{escaped}'"


@lru_cache
def get_settings() -> Settings:
    return Settings()
