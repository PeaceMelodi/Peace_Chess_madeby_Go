ALTER TABLE games ADD COLUMN creator_color TEXT NOT NULL DEFAULT 'white' CHECK (creator_color IN ('white', 'black'));
