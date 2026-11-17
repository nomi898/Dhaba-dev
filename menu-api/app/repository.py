"""PostgreSQL access for menu data."""

from __future__ import annotations

from typing import Any, Protocol

from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool

MENU_SQL = """
SELECT c.id   AS category_id,
       c.name AS category_name,
       c.slug AS category_slug,
       c.sort_order,
       i.id, i.name, i.description, i.price_cents,
       i.is_available, i.is_vegetarian, i.spice_level
FROM menu_categories c
LEFT JOIN menu_items i ON i.category_id = c.id
ORDER BY c.sort_order, c.id, i.name
"""

ITEM_SQL = """
SELECT i.id, i.name, i.description, i.price_cents, i.is_available,
       i.is_vegetarian, i.spice_level,
       c.id AS category_id, c.name AS category_name, c.slug AS category_slug
FROM menu_items i
JOIN menu_categories c ON c.id = i.category_id
WHERE i.id = %s
"""


class MenuRepository(Protocol):
    def fetch_menu(self) -> list[dict[str, Any]]: ...

    def fetch_item(self, item_id: int) -> dict[str, Any] | None: ...

    def ping(self) -> None: ...


def _item_from_row(row: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": row["id"],
        "name": row["name"],
        "description": row["description"],
        "priceCents": row["price_cents"],
        "isAvailable": row["is_available"],
        "isVegetarian": row["is_vegetarian"],
        "spiceLevel": row["spice_level"],
    }


def group_menu_rows(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Turn flat category/item join rows into a list of categories with nested items."""
    categories: dict[int, dict[str, Any]] = {}
    for row in rows:
        cat = categories.get(row["category_id"])
        if cat is None:
            cat = {
                "id": row["category_id"],
                "name": row["category_name"],
                "slug": row["category_slug"],
                "items": [],
            }
            categories[row["category_id"]] = cat
        if row.get("id") is not None:
            cat["items"].append(_item_from_row(row))
    return list(categories.values())


class PostgresMenuRepository:
    def __init__(self, pool: ConnectionPool) -> None:
        self._pool = pool

    def fetch_menu(self) -> list[dict[str, Any]]:
        with self._pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(MENU_SQL)
            return group_menu_rows(cur.fetchall())

    def fetch_item(self, item_id: int) -> dict[str, Any] | None:
        with self._pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(ITEM_SQL, (item_id,))
            row = cur.fetchone()
        if row is None:
            return None
        item = _item_from_row(row)
        item["category"] = {
            "id": row["category_id"],
            "name": row["category_name"],
            "slug": row["category_slug"],
        }
        return item

    def ping(self) -> None:
        with self._pool.connection() as conn:
            conn.execute("SELECT 1")
