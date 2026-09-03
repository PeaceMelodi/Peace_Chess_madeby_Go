CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_state TEXT NOT NULL DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    current_turn TEXT NOT NULL DEFAULT 'white' CHECK (current_turn IN ('white', 'black')),
    status TEXT NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'ongoing', 'finished', 'abandoned')),
    white_token UUID NOT NULL DEFAULT gen_random_uuid(),
    black_token UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);