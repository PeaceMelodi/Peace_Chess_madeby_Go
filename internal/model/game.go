package model

import "time"

type Game struct {
	ID           string    `db:"id" json:"id"`
	BoardState   string    `db:"board_state" json:"board_state"`
	CurrentTurn  string    `db:"current_turn" json:"current_turn"`
	Status       string    `db:"status" json:"status"`
	WhiteToken   string    `db:"white_token" json:"white_token"`
	BlackToken   string    `db:"black_token" json:"black_token"`
	CreatorColor string    `db:"creator_color" json:"creator_color"`
	DrawOfferedBy *string  `db:"draw_offered_by" json:"draw_offered_by"`
	DrawExpiresAt *time.Time `db:"draw_expires_at" json:"draw_expires_at"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}