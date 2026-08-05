package server

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a single connected WebSocket peer.
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	playerID string
}

// Hub tracks the WebSocket clients connected to each room.
type Hub struct {
	mu    sync.Mutex
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Join(roomID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]struct{})
	}
	h.rooms[roomID][c] = struct{}{}
}

func (h *Hub) Leave(roomID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[roomID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

// Clients returns a snapshot of the clients currently in a room.
func (h *Hub) Clients(roomID string) []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := make([]*Client, 0, len(h.rooms[roomID]))
	for c := range h.rooms[roomID] {
		clients = append(clients, c)
	}
	return clients
}

// Find returns the client for a player in a room, or nil if not connected.
func (h *Hub) Find(roomID, playerID string) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.rooms[roomID] {
		if c.playerID == playerID {
			return c
		}
	}
	return nil
}

// sendBestEffort queues a message without blocking the caller.
func (c *Client) sendBestEffort(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) readPump(roomID string, handle func(*Client, clientMessage) error, onDisconnect func()) {
	defer func() {
		onDisconnect()
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		var msg clientMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		if err := handle(c, msg); err != nil {
			c.sendBestEffort(mustJSON(errMessage(err.Error())))
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("ws write: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
