package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub { return &Hub{rooms: make(map[string]*Room)} }

func (h *Hub) EnsureRoom(id string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[id]; ok {
		return r
	}
	r := &Room{
		id:      id,
		clients: make(map[*Client]struct{}),
	}
	h.rooms[id] = r
	return r
}

func (h *Hub) CloseRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[id]; ok {
		for c := range r.clients {
			c.Close()
		}
		delete(h.rooms, id)
	}
}

func (h *Hub) Broadcast(roomID string, payload []byte) {
	h.mu.RLock()
	r := h.rooms[roomID]
	h.mu.RUnlock()
	if r != nil {
		r.Broadcast(payload)
	}
}

type Room struct {
	id      string
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func (r *Room) Add(c *Client) {
	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()
}

func (r *Room) Remove(c *Client) {
	r.mu.Lock()
	delete(r.clients, c)
	r.mu.Unlock()
}

func (r *Room) Broadcast(msg []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for c := range r.clients {
		select {
		case c.Send <- msg:
		default:
			go c.Close()
		}
	}
}

type Client struct {
	conn *websocket.Conn
	room *Room
	uid  string
	Send chan []byte
}

func NewClient(conn *websocket.Conn, room *Room, userID string) *Client {
	return &Client{
		conn: conn,
		room: room,
		uid:  userID,
		Send: make(chan []byte, 64),
	}
}

func (c *Client) Close() {
	_ = c.conn.Close()
	c.room.Remove(c)
	close(c.Send)
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(onIncoming func(fromUserID string, message []byte)) {
	defer c.Close()
	c.conn.SetReadLimit(64 << 10) // 64KB
	c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})
	for {
		mt, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if mt == websocket.TextMessage && onIncoming != nil {
			onIncoming(c.uid, msg)
		}
	}
}

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
