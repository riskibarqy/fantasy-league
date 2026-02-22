BEGIN;

DELETE FROM custom_leagues
WHERE is_default = TRUE
  AND country_code IS NOT NULL
  AND country_code <> 'ID'
  AND deleted_at IS NULL;

WITH league_defaults AS (
    SELECT
        l.public_id AS league_public_id,
        'default-league-' || l.public_id AS public_id,
        'Global - ' || l.name AS group_name,
        'DL' || UPPER(SUBSTRING(md5('default-league:' || l.public_id) FROM 1 FOR 10)) AS invite_code
    FROM leagues l
    WHERE l.deleted_at IS NULL
)
UPDATE custom_leagues cl
SET owner_user_id = 'system',
    name = ld.group_name,
    invite_code = ld.invite_code,
    is_default = TRUE,
    country_code = NULL,
    deleted_at = NULL
FROM league_defaults ld
WHERE cl.league_public_id = ld.league_public_id
  AND cl.is_default = TRUE
  AND cl.country_code IS NULL
  AND cl.deleted_at IS NULL;

WITH league_defaults AS (
    SELECT
        l.public_id AS league_public_id,
        'default-league-' || l.public_id AS public_id,
        'Global - ' || l.name AS group_name,
        'DL' || UPPER(SUBSTRING(md5('default-league:' || l.public_id) FROM 1 FOR 10)) AS invite_code
    FROM leagues l
    WHERE l.deleted_at IS NULL
)
INSERT INTO custom_leagues (
    public_id,
    league_public_id,
    country_code,
    owner_user_id,
    name,
    invite_code,
    is_default
)
SELECT
    ld.public_id,
    ld.league_public_id,
    NULL,
    'system',
    ld.group_name,
    ld.invite_code,
    TRUE
FROM league_defaults ld
LEFT JOIN custom_leagues cl
       ON cl.league_public_id = ld.league_public_id
      AND cl.is_default = TRUE
      AND cl.country_code IS NULL
      AND cl.deleted_at IS NULL
WHERE cl.id IS NULL
ON CONFLICT (public_id) DO UPDATE
SET league_public_id = EXCLUDED.league_public_id,
    country_code = EXCLUDED.country_code,
    owner_user_id = EXCLUDED.owner_user_id,
    name = EXCLUDED.name,
    invite_code = EXCLUDED.invite_code,
    is_default = EXCLUDED.is_default,
    deleted_at = NULL;

WITH league_country_defaults AS (
    SELECT
        l.public_id AS league_public_id,
        'ID'::TEXT AS country_code,
        'Indonesia Fans - ' || l.name AS group_name,
        'default-country-' || l.public_id || '-id' AS public_id,
        'DC' || UPPER(SUBSTRING(md5('default-country:' || l.public_id || ':ID') FROM 1 FOR 10)) AS invite_code
    FROM leagues l
    WHERE l.deleted_at IS NULL
)
UPDATE custom_leagues cl
SET owner_user_id = 'system',
    name = lcd.group_name,
    invite_code = lcd.invite_code,
    is_default = TRUE,
    country_code = lcd.country_code,
    deleted_at = NULL
FROM league_country_defaults lcd
WHERE cl.league_public_id = lcd.league_public_id
  AND cl.country_code = lcd.country_code
  AND cl.is_default = TRUE
  AND cl.deleted_at IS NULL;

WITH league_country_defaults AS (
    SELECT
        l.public_id AS league_public_id,
        'ID'::TEXT AS country_code,
        'Indonesia Fans - ' || l.name AS group_name,
        'default-country-' || l.public_id || '-id' AS public_id,
        'DC' || UPPER(SUBSTRING(md5('default-country:' || l.public_id || ':ID') FROM 1 FOR 10)) AS invite_code
    FROM leagues l
    WHERE l.deleted_at IS NULL
)
INSERT INTO custom_leagues (
    public_id,
    league_public_id,
    country_code,
    owner_user_id,
    name,
    invite_code,
    is_default
)
SELECT
    lcd.public_id,
    lcd.league_public_id,
    lcd.country_code,
    'system',
    lcd.group_name,
    lcd.invite_code,
    TRUE
FROM league_country_defaults lcd
LEFT JOIN custom_leagues cl
       ON cl.league_public_id = lcd.league_public_id
      AND cl.country_code = lcd.country_code
      AND cl.is_default = TRUE
      AND cl.deleted_at IS NULL
WHERE cl.id IS NULL
ON CONFLICT (public_id) DO UPDATE
SET league_public_id = EXCLUDED.league_public_id,
    country_code = EXCLUDED.country_code,
    owner_user_id = EXCLUDED.owner_user_id,
    name = EXCLUDED.name,
    invite_code = EXCLUDED.invite_code,
    is_default = EXCLUDED.is_default,
    deleted_at = NULL;

COMMIT;
