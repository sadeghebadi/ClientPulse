-- Migration 000011: Telegram Long Polling State
CREATE TABLE IF NOT EXISTS telegram_poller_state (
    id INT PRIMARY KEY DEFAULT 1,
    last_update_id BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_telegram_poller_single_row CHECK (id = 1)
);

INSERT INTO telegram_poller_state (id, last_update_id, updated_at)
VALUES (1, 0, NOW())
ON CONFLICT (id) DO NOTHING;

GRANT SELECT, INSERT, UPDATE ON telegram_poller_state TO clientpulse_app, clientpulse_worker;

