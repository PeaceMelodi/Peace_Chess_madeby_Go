package service

import (
	"context"
	"errors"
	"time"

	chess "github.com/corentings/chess/v2"
	"github.com/google/uuid"
	"peacechess/internal/model"
	"peacechess/internal/repository"
)

type GameService struct {
	repo *repository.GameRepository
}

func NewGameService(repo *repository.GameRepository) *GameService {
	return &GameService{repo: repo}
}

func (s *GameService) CreateGame(ctx context.Context, chosenColor string) (*model.Game, string, error) {
	if chosenColor != "white" && chosenColor != "black" {
		return nil, "", errors.New("color must be 'white' or 'black'")
	}

	newGame := chess.NewGame()
	startFEN := newGame.Position().String()

	game := &model.Game{
		BoardState:   startFEN,
		CurrentTurn:  "white",
		Status:       "waiting",
		WhiteToken:   uuid.NewString(),
		BlackToken:   uuid.NewString(),
		CreatorColor: chosenColor,
	}

	if err := s.repo.CreateGame(ctx, game); err != nil {
		return nil, "", err
	}

	playerToken := game.WhiteToken
	if chosenColor == "black" {
		playerToken = game.BlackToken
	}

	return game, playerToken, nil
}

func (s *GameService) JoinGame(ctx context.Context, gameID string) (*model.Game, string, string, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, "", "", err
	}
	if game == nil {
		return nil, "", "", errors.New("game not found")
	}
	if game.Status != "waiting" {
		return nil, "", "", errors.New("game is not open for joining")
	}

	if err := s.repo.UpdateStatus(ctx, game.ID, "ongoing"); err != nil {
		return nil, "", "", err
	}
	game.Status = "ongoing"

	var joinerColor, joinerToken string
	if game.CreatorColor == "white" {
		joinerColor = "black"
		joinerToken = game.BlackToken
	} else {
		joinerColor = "white"
		joinerToken = game.WhiteToken
	}

	return game, joinerColor, joinerToken, nil
}

func (s *GameService) GetGameForConnection(ctx context.Context, gameID string) (*model.Game, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errors.New("game not found")
	}
	return game, nil
}

type MoveResult struct {
	Game      *model.Game
	GameOver  bool
	Outcome   string
	Method    string
}

func (s *GameService) MakeMove(ctx context.Context, gameID string, seat string, moveUCI string) (*MoveResult, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errors.New("game not found")
	}
	if game.Status != "ongoing" {
		return nil, errors.New("game is not in progress")
	}

	if game.CurrentTurn != seat {
		return nil, errors.New("not your turn")
	}

	fen, err := chess.FEN(game.BoardState)
	if err != nil {
		return nil, err
	}
	chessGame := chess.NewGame(fen)

	if err := chessGame.PushNotationMove(moveUCI, chess.UCINotation{}, nil); err != nil {
		return nil, errors.New("illegal move")
	}

	newTurn := "black"
	if seat == "black" {
		newTurn = "white"
	}
	game.BoardState = chessGame.Position().String()
	game.CurrentTurn = newTurn

	outcome := chessGame.Outcome()
	result := &MoveResult{Game: game}

	if outcome != chess.NoOutcome {
		game.Status = "finished"
		if err := s.repo.FinishGame(ctx, game.ID, game.BoardState, game.CurrentTurn); err != nil {
			return nil, err
		}
		result.GameOver = true
		result.Outcome = string(outcome)
		result.Method = chessGame.Method().String()
	} else {
		if err := s.repo.UpdateBoard(ctx, game.ID, game.BoardState, game.CurrentTurn); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *GameService) OfferDraw(ctx context.Context, gameID string, seat string) error {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil {
		return errors.New("game not found")
	}
	if game.Status != "ongoing" {
		return errors.New("game is not in progress")
	}
	if game.DrawOfferedBy != nil && *game.DrawOfferedBy == seat {
		return errors.New("draw already offered")
	}
	if game.DrawOfferedBy != nil && *game.DrawOfferedBy != seat && game.DrawExpiresAt != nil && time.Now().Before(*game.DrawExpiresAt) {
		return errors.New("opponent has a pending draw offer")
	}

	expiresAt := time.Now().Add(5 * time.Second)
	if err := s.repo.SetDrawOffer(ctx, gameID, seat, expiresAt); err != nil {
		return err
	}
	return nil
}

func (s *GameService) AcceptDraw(ctx context.Context, gameID string, seat string) (*MoveResult, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errors.New("game not found")
	}
	if game.Status != "ongoing" {
		return nil, errors.New("game is not in progress")
	}
	if game.DrawOfferedBy == nil {
		return nil, errors.New("no draw offer to accept")
	}
	if *game.DrawOfferedBy == seat {
		return nil, errors.New("cannot accept your own draw offer")
	}
	if game.DrawExpiresAt == nil || time.Now().After(*game.DrawExpiresAt) {
		// Draw offer expired
		if err := s.repo.ClearDrawOffer(ctx, gameID); err != nil {
			return nil, err
		}
		return nil, errors.New("draw offer has expired")
	}

	if err := s.repo.FinishDraw(ctx, gameID, game.BoardState, game.CurrentTurn); err != nil {
		return nil, err
	}

	game.Status = "draw"
	game.DrawOfferedBy = nil
	game.DrawExpiresAt = nil

	result := &MoveResult{
		Game:     game,
		GameOver: true,
		Outcome:  "draw",
		Method:   "agreement",
	}
	return result, nil
}

func (s *GameService) DeclineDraw(ctx context.Context, gameID string, seat string) error {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil {
		return errors.New("game not found")
	}
	if game.Status != "ongoing" {
		return errors.New("game is not in progress")
	}
	if game.DrawOfferedBy == nil {
		return errors.New("no draw offer to decline")
	}
	if *game.DrawOfferedBy == seat {
		return errors.New("cannot decline your own draw offer")
	}

	if err := s.repo.ClearDrawOffer(ctx, gameID); err != nil {
		return err
	}
	return nil
}

func (s *GameService) CheckDrawExpired(ctx context.Context, gameID string) (bool, error) {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return false, err
	}
	if game == nil {
		return false, errors.New("game not found")
	}
	if game.DrawOfferedBy == nil {
		return false, nil
	}
	if game.DrawExpiresAt != nil && time.Now().After(*game.DrawExpiresAt) {
		if err := s.repo.ClearDrawOffer(ctx, gameID); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *GameService) CloseGame(ctx context.Context, gameID string, token string) error {
	game, err := s.repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil {
		return errors.New("game not found")
	}

	// Verify creator token
	var creatorToken string
	if game.CreatorColor == "white" {
		creatorToken = game.WhiteToken
	} else {
		creatorToken = game.BlackToken
	}

	if token != creatorToken {
		return errors.New("only the game creator can close the game")
	}

	if err := s.repo.CloseGame(ctx, gameID); err != nil {
		return err
	}
	return nil
}