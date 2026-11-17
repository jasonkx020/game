BEGIN;

DROP TABLE IF EXISTS agent_withdrawals;
DROP TABLE IF EXISTS agent_deposits;
DROP TABLE IF EXISTS risk_alerts;
DROP TABLE IF EXISTS player_playtime_daily;
DROP TABLE IF EXISTS player_devices;

ALTER TABLE players DROP COLUMN IF EXISTS risk_status;
ALTER TABLE players DROP COLUMN IF EXISTS is_minor;

COMMIT;
