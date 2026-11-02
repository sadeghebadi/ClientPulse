-- Migration: 000012_notification_message_quality.up.sql
-- Description: Add optional per-destination locale override for localized notifications (en, fa)

ALTER TABLE notification_destinations
    ADD COLUMN IF NOT EXISTS locale VARCHAR(16) NULL;

ALTER TABLE notification_destinations
    DROP CONSTRAINT IF EXISTS chk_notif_dest_locale;

ALTER TABLE notification_destinations
    ADD CONSTRAINT chk_notif_dest_locale
    CHECK (locale IS NULL OR locale IN ('en', 'fa'));


-- Allow worker service to enrich notification payloads with client and organization metadata
GRANT SELECT ON public.clients TO clientpulse_worker;
GRANT SELECT ON public.organizations TO clientpulse_worker;
