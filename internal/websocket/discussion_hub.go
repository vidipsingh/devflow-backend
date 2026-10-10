
package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ─── Wire types ───────────────────────────────────────────────────────────────

// DiscussionEvent is the JSON envelope broadcast to every client in a team room.
type DiscussionEvent struct {
	Type    string      `json:"type"`    // new_discussion | update_discussion | delete_discussion | pin_discussion | resolve_discussion | new_reply | delete_reply
	Payload interface{} `json:"payload"` // the affected object (TeamDiscussion or TeamDiscussionReply) or minimal {id}
}

// ─── DiscussionClient ─────────────────────────────────────────────────────────

type DiscussionClient struct {
	hub      *DiscussionHub
	roomKey  string // team slug
	userID   string
	userName string
	conn     *websocket.Conn
	send     chan []byte
}

func NewDiscussionClient(
	hub *DiscussionHub,
	conn *websocket.Conn,
	roomKey, userID, userName string,
) *DiscussionClient {
	return &DiscussionClient{
		hub:      hub,
		conn:     conn,
		roomKey:  roomKey,
		userID:   userID,
		userName: userName,
		send:     make(chan []byte, 256),
	}
}

const (
	discWriteWait  = 10 * time.Second
	discPongWait   = 60 * time.Second
	discPingPeriod = (discPongWait * 9) / 10
	discMaxMsgSize = 4 * 1024 // 4 KB — clients only send pings
)

func (c *DiscussionClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(discMaxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(discPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(discPongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("discussion ws read error [%s] %s: %v", c.roomKey, c.userID, err)
			}
			break
		}
		// Clients are read-only (events flow server → client).
		// We still drain the read loop so pings work.
	}
}

func (c *DiscussionClient) WritePump() {
	ticker := time.NewTicker(discPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(discWriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("discussion ws write error [%s] %s: %v", c.roomKey, c.userID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(discWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ─── DiscussionRoom ───────────────────────────────────────────────────────────

type discussionRoom struct {
	mu      sync.Mutex
	clients map[string]*DiscussionClient // keyed by userID
}

func newDiscussionRoom() *discussionRoom {
	return &discussionRoom{clients: make(map[string]*DiscussionClient)}
}

func (r *discussionRoom) broadcast(msg []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.clients {
		select {
		case c.send <- msg:
		default:
			log.Printf("discussion room: send buffer full for %s", c.userID)
		}
	}
}

// ─── DiscussionHub ────────────────────────────────────────────────────────────

type discClientEvent struct {
	client *DiscussionClient
}

// DiscussionHub manages one room per team slug.
type DiscussionHub struct {
	mu         sync.RWMutex
	rooms      map[string]*discussionRoom
	register   chan *DiscussionClient
	unregister chan *DiscussionClient
}

// GlobalDiscussionHub is the singleton started in main / router init.
var GlobalDiscussionHub *DiscussionHub

func NewDiscussionHub() *DiscussionHub {
	return &DiscussionHub{
		rooms:      make(map[string]*discussionRoom),
		register:   make(chan *DiscussionClient, 64),
		unregister: make(chan *DiscussionClient, 64),
	}
}

func (h *DiscussionHub) Run() {
	log.Println("discussion hub started")
	for {
		select {
		case c := <-h.register:
			h.handleJoin(c)
		case c := <-h.unregister:
			h.handleLeave(c)
		}
	}
}

// GetOrCreateRoom lazily creates a room for the given team slug.
func (h *DiscussionHub) GetOrCreateRoom(slug string) *discussionRoom {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[slug]; ok {
		return r
	}
	r := newDiscussionRoom()
	h.rooms[slug] = r
	log.Printf("discussion hub: created room [%s]", slug)
	return r
}

// RegisterClient queues a client for joining.
func (h *DiscussionHub) RegisterClient(c *DiscussionClient) {
	h.register <- c
}

// Broadcast serialises evt and sends it to every connected client in the team room.
func (h *DiscussionHub) Broadcast(slug string, evt DiscussionEvent) {
	h.mu.RLock()
	room, ok := h.rooms[slug]
	h.mu.RUnlock()
	if !ok {
		return
	}
	msg, err := json.Marshal(evt)
	if err != nil {
		log.Printf("discussion hub: marshal error: %v", err)
		return
	}
	room.broadcast(msg)
	log.Printf("discussion hub: broadcast type=%s slug=%s", evt.Type, slug)
}

func (h *DiscussionHub) handleJoin(c *DiscussionClient) {
	h.mu.Lock()
	room, ok := h.rooms[c.roomKey]
	if !ok {
		room = newDiscussionRoom()
		h.rooms[c.roomKey] = room
		log.Printf("discussion hub: created room on join [%s]", c.roomKey)
	}
	h.mu.Unlock()

	room.mu.Lock()
	room.clients[c.userID] = c
	count := len(room.clients)
	room.mu.Unlock()

	log.Printf("discussion hub: client joined [%s] %s (%s), total=%d", c.roomKey, c.userID, c.userName, count)
}

func (h *DiscussionHub) handleLeave(c *DiscussionClient) {
	h.mu.RLock()
	room, ok := h.rooms[c.roomKey]
	h.mu.RUnlock()
	if !ok {
		return
	}

	room.mu.Lock()
	// Only remove if the entry in the map is still THIS client (pointer equality).
	// A rapid reconnect (e.g. React StrictMode double-invoke) can replace the map
	// entry before the old ReadPump's deferred unregister fires.
	if existing, exists := room.clients[c.userID]; exists && existing == c {
		delete(room.clients, c.userID)
		close(c.send)
		log.Printf("discussion hub: client left [%s] %s, remaining=%d", c.roomKey, c.userID, len(room.clients))
	} else if exists {
		log.Printf("discussion hub: stale unregister ignored [%s] %s (already replaced by new conn)", c.roomKey, c.userID)
	}
	room.mu.Unlock()
}
