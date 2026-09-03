-- Remove draw offer tracking columns
ALTER TABLE games 
DROP COLUMN IF EXISTS draw_offered_by,
DROP COLUMN IF EXISTS draw_expires_at;

-- Restore original status constraint
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_status_check;
ALTER TABLE games ADD CONSTRAINT games_status_check CHECK (status IN ('waiting', 'ongoing', 'finished', 'abandoned'));