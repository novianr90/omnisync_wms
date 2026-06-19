-- Seed realistic unit weight/volume for existing products
UPDATE products SET
    unit_weight = 0.9,       -- 0.9 kg per keyboard
    unit_volume = 0.001155   -- ~30×11×3.5 cm box
WHERE id = 'prod-001'; -- Keychron K2 Keyboard

UPDATE products SET
    unit_weight = 0.14,      -- 0.14 kg per mouse
    unit_volume = 0.000545   -- ~12.7×8.4×5.1 cm box
WHERE id = 'prod-002'; -- Logitech MX Master 3S

UPDATE products SET
    unit_weight = 6.7,       -- 6.7 kg per monitor (with packaging)
    unit_volume = 0.0063     -- ~70×50×18 cm box
WHERE id = 'prod-003'; -- Dell 27" Monitor

UPDATE products SET
    unit_weight = 1.0,       -- 1 kg per unit (sold by weight)
    unit_volume = 0.001      -- 1 L per kg of sugar
WHERE id = 'prod-004'; -- Refined White Sugar

-- Seed capacity limits on all locators (standard shelf: 200 kg / 0.5 m³)
UPDATE locators SET max_weight = 200, max_volume = 0.5 WHERE deleted_at IS NULL;
