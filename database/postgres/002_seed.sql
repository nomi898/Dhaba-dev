-- Sample dhaba menu. Prices are in paisa (PKR cents). Idempotent.

INSERT INTO menu_categories (name, slug, sort_order) VALUES
    ('Starters',       'starters',  1),
    ('Karahi & Curry', 'karahi',    2),
    ('BBQ & Tikka',    'bbq',       3),
    ('Rice',           'rice',      4),
    ('Breads',         'breads',    5),
    ('Drinks',         'drinks',    6),
    ('Desserts',       'desserts',  7)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO menu_items (category_id, name, description, price_cents, is_available, is_vegetarian, spice_level)
SELECT c.id, v.name, v.description, v.price_cents, v.is_available, v.is_vegetarian, v.spice_level
FROM (VALUES
    ('starters', 'Samosa Chaat',        'Crushed samosas with chickpeas, yogurt, tamarind and mint chutney', 35000,  TRUE,  TRUE,  1),
    ('starters', 'Chicken Pakora',      'Gram-flour battered chicken bites fried golden',                    55000,  TRUE,  FALSE, 2),
    ('starters', 'Aloo Tikki',          'Spiced potato patties with green chutney',                          30000,  TRUE,  TRUE,  1),
    ('karahi',   'Chicken Karahi',      'Tomato-based wok curry with green chillies and ginger (half)',      145000, TRUE,  FALSE, 2),
    ('karahi',   'Mutton Karahi',       'Slow-cooked mutton karahi, black pepper finish (half)',             245000, TRUE,  FALSE, 2),
    ('karahi',   'Daal Mash',           'Buttery white lentils tempered with garlic and cumin',              65000,  TRUE,  TRUE,  1),
    ('karahi',   'Palak Paneer',        'Spinach puree with fresh cottage cheese',                           85000,  TRUE,  TRUE,  1),
    ('karahi',   'Nihari',              'Overnight beef stew with bone marrow, ginger and lemon',            120000, FALSE, FALSE, 3),
    ('bbq',      'Chicken Tikka',       'Charcoal-grilled leg quarter in red masala',                        65000,  TRUE,  FALSE, 2),
    ('bbq',      'Seekh Kabab (4 pcs)', 'Minced beef skewers with herbs',                                    70000,  TRUE,  FALSE, 2),
    ('bbq',      'Malai Boti',          'Creamy, mildly spiced boneless chicken cubes',                      90000,  TRUE,  FALSE, 0),
    ('rice',     'Chicken Biryani',     'Layered basmati rice with spiced chicken and saffron',              60000,  TRUE,  FALSE, 2),
    ('rice',     'Vegetable Pulao',     'Fragrant rice with seasonal vegetables',                            45000,  TRUE,  TRUE,  0),
    ('breads',   'Tandoori Roti',       'Whole-wheat flatbread from the clay oven',                          4000,   TRUE,  TRUE,  0),
    ('breads',   'Garlic Naan',         'Leavened naan brushed with garlic butter',                          12000,  TRUE,  TRUE,  0),
    ('breads',   'Aloo Paratha',        'Layered flatbread stuffed with spiced potato',                      18000,  TRUE,  TRUE,  1),
    ('drinks',   'Doodh Patti Chai',    'Strong milk tea brewed the dhaba way',                              10000,  TRUE,  TRUE,  0),
    ('drinks',   'Sweet Lassi',         'Chilled churned yogurt drink',                                      25000,  TRUE,  TRUE,  0),
    ('drinks',   'Fresh Lime Soda',     'Sweet or salted, with crushed ice',                                 15000,  TRUE,  TRUE,  0),
    ('desserts', 'Gulab Jamun (2 pcs)', 'Milk dumplings soaked in rose syrup',                               20000,  TRUE,  TRUE,  0),
    ('desserts', 'Kheer',               'Slow-cooked rice pudding with cardamom and pistachio',              25000,  TRUE,  TRUE,  0)
) AS v(slug, name, description, price_cents, is_available, is_vegetarian, spice_level)
JOIN menu_categories c ON c.slug = v.slug
ON CONFLICT (category_id, name) DO NOTHING;
