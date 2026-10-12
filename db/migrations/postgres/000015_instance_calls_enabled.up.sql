-- Per-instance opt-in for voice calls (VoIP engine)
ALTER TABLE instances ADD COLUMN IF NOT EXISTS calls_enabled BOOLEAN NOT NULL DEFAULT FALSE;
