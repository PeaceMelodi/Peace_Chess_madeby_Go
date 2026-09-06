package handler

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

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
	service    *service.GameService
	hub        *ws.Hub
	mu         sync.Mutex
	drawTimers map[string]*time.Timer
}

func NewWebSocketHandler(service *service.GameService, hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		service:    service,
		hub:        hub,
		drawTimers: make(map[string]*time.Timer),
	}
}

type incomingMessage struct {
	Type string `json:"type"`
	Move string `json:"move,omitempty"`
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

	// Send initial game state
	client.Send(map[string]interface{}{
		"type":         "game_state",
		"board_state":  game.BoardState,
		"current_turn": game.CurrentTurn,
		"status":       game.Status,
		"your_color":   seat,
	})

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type": "opponent_reconnected",
		"seat": seat,
	})

	// Set a read deadline to detect dead connections.
	// We'll reset it on every successful read.
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		// Also reset on pong (from browser's protocol-level pong)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	defer func() {
		h.hub.Unregister(gameID, client)
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type": "opponent_disconnected",
			"seat": seat,
		})
	}()

	for {
		// Reset read deadline before each read (so any message keeps it alive)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg incomingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// Check if it's a timeout (connection dead) or a normal close
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("connection timeout for %s in game %s", seat, gameID)
			} else {
				log.Printf("player disconnected from game %s (%s): %v", gameID, seat, err)
			}
			return
		}

		switch msg.Type {
		case "move":
			h.handleMove(r.Context(), gameID, seat, msg.Move)
		case "draw_offer":
			h.handleDrawOffer(r.Context(), gameID, seat)
		case "draw_accept":
			h.handleDrawAccept(r.Context(), gameID, seat)
		case "draw_decline":
			h.handleDrawDecline(r.Context(), gameID, seat)
		case "close_game":
			h.handleCloseGame(r.Context(), gameID, token, seat)
		case "ping":
			// Respond to client ping with pong
			client.Send(map[string]interface{}{
				"type": "pong",
			})
		default:
			client.Send(map[string]interface{}{
				"type":    "error",
				"message": "unknown message type",
			})
		}
	}
}

func (h *WebSocketHandler) handleMove(ctx context.Context, gameID string, seat string, move string) {
	result, err := h.service.MakeMove(ctx, gameID, seat, move)
	if err != nil {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
			"seat":    seat,
		})
		return
	}

	if result.GameOver {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":         "game_over",
			"board_state":  result.Game.BoardState,
			"current_turn": result.Game.CurrentTurn,
			"status":       result.Game.Status,
			"outcome":      result.Outcome,
			"method":       result.Method,
		})
		return
	}

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type":         "move_made",
		"board_state":  result.Game.BoardState,
		"current_turn": result.Game.CurrentTurn,
		"status":       result.Game.Status,
	})
}

func (h *WebSocketHandler) handleDrawOffer(ctx context.Context, gameID string, seat string) {
	err := h.service.OfferDraw(ctx, gameID, seat)
	if err != nil {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
			"seat":    seat,
		})
		return
	}

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type":       "draw_offered",
		"offered_by": seat,
	})

	h.mu.Lock()
	if timer, exists := h.drawTimers[gameID]; exists {
		timer.Stop()
	}
	h.drawTimers[gameID] = time.AfterFunc(5*time.Second, func() {
		expired, err := h.service.CheckDrawExpired(ctx, gameID)
		if err == nil && expired {
			h.hub.Broadcast(gameID, map[string]interface{}{
				"type": "draw_expired",
			})
		}
		h.mu.Lock()
		delete(h.drawTimers, gameID)
		h.mu.Unlock()
	})
	h.mu.Unlock()
}

func (h *WebSocketHandler) handleDrawAccept(ctx context.Context, gameID string, seat string) {
	h.mu.Lock()
	if timer, exists := h.drawTimers[gameID]; exists {
		timer.Stop()
		delete(h.drawTimers, gameID)
	}
	h.mu.Unlock()

	result, err := h.service.AcceptDraw(ctx, gameID, seat)
	if err != nil {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
			"seat":    seat,
		})
		return
	}

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type":         "game_draw",
		"board_state":  result.Game.BoardState,
		"current_turn": result.Game.CurrentTurn,
		"status":       "draw",
		"outcome":      "draw",
		"method":       "agreement",
	})
}

func (h *WebSocketHandler) handleDrawDecline(ctx context.Context, gameID string, seat string) {
	h.mu.Lock()
	if timer, exists := h.drawTimers[gameID]; exists {
		timer.Stop()
		delete(h.drawTimers, gameID)
	}
	h.mu.Unlock()

	err := h.service.DeclineDraw(ctx, gameID, seat)
	if err != nil {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
			"seat":    seat,
		})
		return
	}

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type":        "draw_declined",
		"declined_by": seat,
	})
}

func (h *WebSocketHandler) handleCloseGame(ctx context.Context, gameID string, token string, seat string) {
	h.mu.Lock()
	if timer, exists := h.drawTimers[gameID]; exists {
		timer.Stop()
		delete(h.drawTimers, gameID)
	}
	h.mu.Unlock()

	err := h.service.CloseGame(ctx, gameID, token)
	if err != nil {
		h.hub.Broadcast(gameID, map[string]interface{}{
			"type":    "error",
			"message": err.Error(),
			"seat":    seat,
		})
		return
	}

	h.hub.Broadcast(gameID, map[string]interface{}{
		"type":    "game_closed",
		"message": "Game has been closed by the creator",
	})
}