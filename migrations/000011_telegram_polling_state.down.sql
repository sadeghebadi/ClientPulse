-- Rollback Migration 000011: Telegram Long Polling State
DROP TABLE IF EXISTS telegram_poller_state CASCADE;

