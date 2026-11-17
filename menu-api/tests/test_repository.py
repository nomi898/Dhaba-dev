from app.repository import group_menu_rows


def row(cat_id, cat_name, item_id=None, name=None):
    return {
        "category_id": cat_id,
        "category_name": cat_name,
        "category_slug": cat_name.lower(),
        "sort_order": cat_id,
        "id": item_id,
        "name": name,
        "description": "d",
        "price_cents": 100,
        "is_available": True,
        "is_vegetarian": False,
        "spice_level": 0,
    }


def test_groups_items_by_category_preserving_order():
    rows = [row(1, "Starters", 1, "A"), row(1, "Starters", 2, "B"), row(2, "Rice", 3, "C")]
    menu = group_menu_rows(rows)
    assert [c["name"] for c in menu] == ["Starters", "Rice"]
    assert [i["name"] for i in menu[0]["items"]] == ["A", "B"]
    assert menu[1]["items"][0]["priceCents"] == 100


def test_empty_category_has_no_items():
    menu = group_menu_rows([row(5, "Desserts")])
    assert menu == [{"id": 5, "name": "Desserts", "slug": "desserts", "items": []}]
