-- Add lock state column to managed_locks
ALTER TABLE managed_locks
ADD COLUMN state TEXT NOT NULL DEFAULT 'unknown'
    CHECK (state IN ('locked', 'unlocked', 'jammed', 'unknown'));

