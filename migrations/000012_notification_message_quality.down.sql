-- Migration: 000012_notification_message_quality.down.sql
-- Description: Revert per-destination locale override

ALTER TABLE notification_destinations
    DROP CONSTRAINT IF EXISTS chk_notif_dest_locale;

ALTER TABLE notification_destinations
    DROP COLUMN IF EXISTS locale;


REVOKE SELECT ON public.clients FROM clientpulse_worker;
REVOKE SELECT ON public.organizations FROM clientpulse_worker;
