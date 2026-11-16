-- Example SQL for first deploy (illustrative).
-- Prefer scripts/init-postgres.ps1 / init-postgres.sh which are idempotent
-- and write DATABASE_URL into .env files.
--
-- Run as superuser (postgres):
--   psql -U postgres -d postgres -f scripts/sql/init_roles.example.sql

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'game') THEN
    CREATE ROLE game LOGIN PASSWORD 'change-me';
  ELSE
    ALTER ROLE game WITH LOGIN PASSWORD 'change-me';
  END IF;
END
$$;

SELECT format('CREATE DATABASE %I OWNER %I', 'game', 'game')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'game')\gexec

GRANT ALL PRIVILEGES ON DATABASE game TO game;
ALTER DATABASE game OWNER TO game;

-- Optional Auth DB (physical split):
-- SELECT format('CREATE DATABASE %I OWNER %I', 'auth', 'game')
-- WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'auth')\gexec
-- GRANT ALL PRIVILEGES ON DATABASE auth TO game;
-- ALTER DATABASE auth OWNER TO game;

\c game
GRANT ALL ON SCHEMA public TO game;
ALTER SCHEMA public OWNER TO game;
