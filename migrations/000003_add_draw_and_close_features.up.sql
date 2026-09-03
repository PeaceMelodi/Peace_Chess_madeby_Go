-- Add draw offer tracking columns
ALTER TABLE games 
ADD COLUMN draw_offered_by TEXT,
ADD COLUMN draw_expires_at TIMESTAMPTZ;

-- Update status constraint to include 'closed' and 'draw'
ALTER TABLE games DROP CONSTRAINT IF EXISTS games_status_check;
ALTER TABLE games ADD CONSTRAINT games_status_check CHECK (status IN ('waiting', 'ongoing', 'finished', 'closed', 'draw'));