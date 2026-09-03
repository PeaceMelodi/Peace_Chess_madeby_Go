package handler

import (
	"encoding/json"
	"net/http"

	"peacechess/internal/service"
)

type GameHandler struct {
	service *service.GameService
}

func NewGameHandler(service *service.GameService) *GameHandler {
	return &GameHandler{service: service}
}

type createGameRequest struct {
	Color string `json:"color"`
}

type createGameResponse struct {
	GameID      string `json:"game_id"`
	BoardState  string `json:"board_state"`
	CurrentTurn string `json:"current_turn"`
	Status      string `json:"status"`
	YourColor   string `json:"your_color"`
	YourToken   string `json:"your_token"`
}

func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req createGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	game, token, err := h.service.CreateGame(r.Context(), req.Color)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := createGameResponse{
		GameID:      game.ID,
		BoardState:  game.BoardState,
		CurrentTurn: game.CurrentTurn,
		Status:      game.Status,
		YourColor:   req.Color,
		YourToken:   token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

type joinGameResponse struct {
	GameID      string `json:"game_id"`
	BoardState  string `json:"board_state"`
	CurrentTurn string `json:"current_turn"`
	Status      string `json:"status"`
	YourColor   string `json:"your_color"`
	YourToken   string `json:"your_token"`
}

func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")

	game, color, token, err := h.service.JoinGame(r.Context(), gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := joinGameResponse{
		GameID:      game.ID,
		BoardState:  game.BoardState,
		CurrentTurn: game.CurrentTurn,
		Status:      game.Status,
		YourColor:   color,
		YourToken:   token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type getGameResponse struct {
	GameID      string `json:"game_id"`
	BoardState  string `json:"board_state"`
	CurrentTurn string `json:"current_turn"`
	Status      string `json:"status"`
	WhiteToken  string `json:"white_token"`
	BlackToken  string `json:"black_token"`
	CreatorColor string `json:"creator_color"`
}

func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")

	game, err := h.service.GetGameForConnection(r.Context(), gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := getGameResponse{
		GameID:       game.ID,
		BoardState:   game.BoardState,
		CurrentTurn:  game.CurrentTurn,
		Status:       game.Status,
		WhiteToken:   game.WhiteToken,
		BlackToken:   game.BlackToken,
		CreatorColor: game.CreatorColor,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type closeGameRequest struct {
	Token string `json:"token"`
}

func (h *GameHandler) CloseGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")

	var req closeGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.CloseGame(r.Context(), gameID, req.Token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "game closed successfully",
		"status":  "closed",
	})
}