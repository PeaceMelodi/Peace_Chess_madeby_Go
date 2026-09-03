package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"peacechess/internal/model"
)

type GameRepository struct {
	db *sqlx.DB
}

func NewGameRepository(db *sqlx.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) CreateGame(ctx context.Context, game *model.Game) error {
	query := `
		INSERT INTO games (board_state, current_turn, status, white_token, black_token, creator_color)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	row := r.db.QueryRowxContext(ctx, query,
		game.BoardState,
		game.CurrentTurn,
		game.Status,
		game.WhiteToken,
		game.BlackToken,
		game.CreatorColor,
	)

	return row.Scan(&game.ID, &game.CreatedAt, &game.UpdatedAt)
}

func (r *GameRepository) GetGameByID(ctx context.Context, id string) (*model.Game, error) {
	var game model.Game
	query := `SELECT * FROM games WHERE id = $1`

	err := r.db.GetContext(ctx, &game, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &game, nil
}

func (r *GameRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE games SET status = $1, updated_at = now() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *GameRepository) UpdateBoard(ctx context.Context, id string, boardState string, currentTurn string) error {
	query := `UPDATE games SET board_state = $1, current_turn = $2, updated_at = now() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, boardState, currentTurn, id)
	return err
}

func (r *GameRepository) FinishGame(ctx context.Context, id string, boardState string, currentTurn string) error {
	query := `UPDATE games SET board_state = $1, current_turn = $2, status = 'finished', updated_at = now() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, boardState, currentTurn, id)
	return err
}