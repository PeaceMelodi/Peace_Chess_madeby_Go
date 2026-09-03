package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"peacechess/internal/service"
	"peacechess/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true 
	},
}

type WebSocketHandler struct {
	service *service.GameService
	hub     *ws.Hub
}

func NewWebSocketHandler(service *service.GameService, hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{service: service, hub: hub}
}

type incomingMove struct {
	Move string `json:"move"` 
}

func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("game_id")
	token := r.URL.Query().Get("token")

	if gameID == "" || token == "" {
		http.Error(w, "game_id and token are required", http.StatusBadRequest)
		return
	}

	game, err := h.service.GetGameForConnection(r.Context(), gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var seat string
	switch token {
	case game.WhiteToken:
		seat = "white"
	case game.BlackToken:
		seat = "black"
	default:
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	client := ws.NewClient(conn, seat)
	h.hub.Register(gameID, client)

	log.Printf("player connected to game %s as %s", gameID, seat)

	client.Send(map[string]string{
		"board_state":  game.BoardState,
		"current_turn": game.CurrentTurn,
		"status":       game.Status,
		"your_color":   seat,
	})

	h.hub.Broadcast(gameID, map[string]string{
		"event": "opponent_reconnected",
		"seat":  seat,
	})

	defer func() {
		h.hub.Unregister(gameID, client)
		h.hub.Broadcast(gameID, map[string]string{
			"event": "opponent_disconnected",
			"seat":  seat,
		})
	}()

	for {
		var msg incomingMove
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("player disconnected from game %s (%s): %v", gameID, seat, err)
			return
		}

		result, err := h.service.MakeMove(r.Context(), gameID, seat, msg.Move)
		if err != nil {
			client.Send(map[string]string{"error": err.Error()})
			continue
		}

		if result.GameOver {
			h.hub.Broadcast(gameID, map[string]string{
				"board_state":  result.Game.BoardState,
				"current_turn": result.Game.CurrentTurn,
				"status":       "finished",
				"outcome":      result.Outcome,
				"method":       result.Method,
			})
			return
		}

		h.hub.Broadcast(gameID, map[string]string{
			"board_state":  result.Game.BoardState,
			"current_turn": result.Game.CurrentTurn,
		})
	}
}