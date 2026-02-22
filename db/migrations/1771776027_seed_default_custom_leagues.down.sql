BEGIN;

DELETE FROM custom_leagues
WHERE public_id LIKE 'default-league-%'
   OR public_id LIKE 'default-country-%';

COMMIT;
