-- Per-instance opt-in for voice calls (VoIP engine)
ALTER TABLE instances ADD COLUMN calls_enabled INTEGER NOT NULL DEFAULT 0;
