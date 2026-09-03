package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Seat string
	mu   sync.Mutex
}

func NewClient(conn *websocket.Conn, seat string) *Client {
	return &Client{Conn: conn, Seat: seat}
}

func (c *Client) Send(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(v)
}