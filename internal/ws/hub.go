package ws

import "sync"

type Hub struct {
	mu    sync.Mutex
	games map[string][]*Client 
}

func NewHub() *Hub {
	return &Hub{
		games: make(map[string][]*Client),
	}
}

func (h *Hub) Register(gameID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.games[gameID] = append(h.games[gameID], client)
}

func (h *Hub) Unregister(gameID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.games[gameID]
	for i, c := range clients {
		if c == client {
			h.games[gameID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

func (h *Hub) Broadcast(gameID string, v interface{}) {
	h.mu.Lock()
	clients := h.games[gameID]
	h.mu.Unlock()

	for _, c := range clients {
		c.Send(v)
	}
}