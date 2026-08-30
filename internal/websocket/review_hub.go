package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"github.com/gorilla/websocket"
)

// ─── Message types ────────────────────────────────────────────────────────────

type ReviewIncoming struct {
	Type    string          `json:"type"`
	To      string          `json:"to"`
	Payload json.RawMessage `json:"payload"`
	// mute-state broadcast fields
	AudioMuted *bool `json:"audioMuted,omitempty"`
	VideoMuted *bool `json:"videoMuted,omitempty"`
}

type ReviewOutgoing struct {
	Type         string                     `json:"type"`
	From         string                     `json:"from"`
	FromName     string                     `json:"fromName"`
	To           string                     `json:"to,omitempty"`
	Payload      json.RawMessage            `json:"payload,omitempty"`
	Participants []models.ReviewParticipant `json:"participants,omitempty"`
	// mute-state broadcast fields — forwarded as-is
	AudioMuted *bool `json:"audioMuted,omitempty"`
	VideoMuted *bool `json:"videoMuted,omitempty"`
}

// ─── ReviewClient ─────────────────────────────────────────────────────────────

type ReviewClient struct {
	hub       *ReviewHub
	sessionID string
	userID    string
	userName  string
	color     string
	conn      *websocket.Conn
	send      chan []byte
}

// NewReviewClient is the public constructor — all fields remain unexported.
func NewReviewClient(
	hub *ReviewHub,
	conn *websocket.Conn,
	sessionID, userID, userName, color string,
) *ReviewClient {
	return &ReviewClient{
		hub:       hub,
		conn:      conn,
		sessionID: sessionID,
		userID:    userID,
		userName:  userName,
		color:     color,
		send:      make(chan []byte, 256),
	}
}

const (
	reviewWriteWait  = 10 * time.Second
	reviewPongWait   = 60 * time.Second
	reviewPingPeriod = (reviewPongWait * 9) / 10
	reviewMaxMsgSize = 64 * 1024
)

func (c *ReviewClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(reviewMaxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(reviewPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(reviewPongWait))
		return nil
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("review ws read error [%s]: %v", c.sessionID, err)
			}
			break
		}
		var msg ReviewIncoming
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("review ws json parse error [%s]: %v", c.sessionID, err)
			continue
		}
		log.Printf("review ws incoming [%s] from %s: type=%s", c.sessionID, c.userID, msg.Type)
		c.hub.incoming <- &reviewEvent{client: c, msg: msg}
	}
}

func (c *ReviewClient) WritePump() {
	ticker := time.NewTicker(reviewPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(reviewWriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("review ws write error [%s]: %v", c.sessionID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(reviewWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("review ws ping error [%s]: %v", c.sessionID, err)
				return
			}
		}
	}
}

// ─── ReviewRoom ───────────────────────────────────────────────────────────────

type ReviewRoom struct {
	mu      sync.Mutex
	clients map[string]*ReviewClient
}

func newReviewRoom() *ReviewRoom {
	return &ReviewRoom{clients: make(map[string]*ReviewClient)}
}

// ─── ReviewHub ────────────────────────────────────────────────────────────────

type reviewEvent struct {
	client *ReviewClient
	msg    ReviewIncoming
}

type ReviewHub struct {
	mu         sync.RWMutex
	rooms      map[string]*ReviewRoom
	register   chan *ReviewClient
	unregister chan *ReviewClient
	incoming   chan *reviewEvent
	repo       *repository.ReviewRepo
}

var GlobalReviewHub *ReviewHub

func NewReviewHub(repo *repository.ReviewRepo) *ReviewHub {
	return &ReviewHub{
		rooms:      make(map[string]*ReviewRoom),
		register:   make(chan *ReviewClient, 64),
		unregister: make(chan *ReviewClient, 64),
		incoming:   make(chan *reviewEvent, 256),
		repo:       repo,
	}
}

// ⚠️ CRITICAL: Start the hub's event loop
// Call this in your main() or initialization function ONCE, in a goroutine:
//
//	hub := NewReviewHub(repo)
//	go hub.Run()  // <-- THIS LINE WAS MISSING!
func (h *ReviewHub) Run() {
	log.Println("review hub started")
	for {
		select {
		case c := <-h.register:
			log.Printf("review hub: client registered [%s] %s (%s)", c.sessionID, c.userID, c.userName)
			h.handleJoin(c)

		case c := <-h.unregister:
			log.Printf("review hub: client unregistered [%s] %s", c.sessionID, c.userID)
			h.handleLeave(c)

		case evt := <-h.incoming:
			log.Printf("review hub: processing message [%s] type=%s from=%s", evt.client.sessionID, evt.msg.Type, evt.client.userID)
			h.handleMessage(evt)
		}
	}
}

func (h *ReviewHub) GetOrCreateRoom(sessionID string) *ReviewRoom {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[sessionID]; ok {
		log.Printf("review hub: reusing room [%s], clients=%d", sessionID, len(r.clients))
		return r
	}
	r := newReviewRoom()
	h.rooms[sessionID] = r
	log.Printf("review hub: created room [%s]", sessionID)
	return r
}

func (h *ReviewHub) RegisterReviewClient(c *ReviewClient) {
	log.Printf("review hub: queueing register for [%s] %s", c.sessionID, c.userID)
	h.register <- c
}

// EndRoom broadcasts a "session-ended" message to every client in the room then
// closes their connections, effectively kicking everyone out.
func (h *ReviewHub) EndRoom(sessionID string) {
	h.mu.RLock()
	room, ok := h.rooms[sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	ended, _ := json.Marshal(ReviewOutgoing{
		Type: "session-ended",
		From: "server",
	})

	room.mu.Lock()
	for _, c := range room.clients {
		// Send the message non-blocking, then close the connection
		select {
		case c.send <- ended:
		default:
		}
	}
	room.mu.Unlock()

	// Give pumps a moment to drain then forcefully close
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.mu.Lock()
		room, ok := h.rooms[sessionID]
		if ok {
			room.mu.Lock()
			for _, c := range room.clients {
				c.conn.Close()
			}
			room.mu.Unlock()
			delete(h.rooms, sessionID)
		}
		h.mu.Unlock()
		log.Printf("review hub: room [%s] torn down after session-ended", sessionID)
	}()
}

func (h *ReviewHub) handleJoin(c *ReviewClient) {
	h.mu.RLock()
	room, ok := h.rooms[c.sessionID]
	h.mu.RUnlock()
	if !ok {
		log.Printf("review hub: room not found [%s], cannot add client %s", c.sessionID, c.userID)
		return
	}

	room.mu.Lock()
	var existing []models.ReviewParticipant
	for _, peer := range room.clients {
		existing = append(existing, models.ReviewParticipant{
			UserID:   peer.userID,
			UserName: peer.userName,
			Color:    peer.color,
			JoinedAt: time.Now(), // Note: this should come from the database if tracking is needed
		})
	}
	room.clients[c.userID] = c
	room.mu.Unlock()

	log.Printf("review hub: client joined [%s] %s, existing participants=%d", c.sessionID, c.userID, len(existing))

	// Send join acknowledgement with current participant list
	joinAck, _ := json.Marshal(ReviewOutgoing{
		Type:         "joined",
		From:         c.userID,
		FromName:     c.userName,
		Participants: existing,
	})
	select {
	case c.send <- joinAck:
		log.Printf("review hub: sent joined ack to %s", c.userID)
	default:
		log.Printf("review hub: failed to send joined ack to %s (buffer full)", c.userID)
	}

	// Notify OTHER peers that a new peer joined
	newPeerMsg, _ := json.Marshal(ReviewOutgoing{
		Type:     "peer-joined",
		From:     c.userID,
		FromName: c.userName,
	})
	room.mu.Lock()
	for uid, peer := range room.clients {
		if uid != c.userID {
			select {
			case peer.send <- newPeerMsg:
				log.Printf("review hub: sent peer-joined to %s about %s", uid, c.userID)
			default:
				log.Printf("review hub: failed to send peer-joined to %s (buffer full)", uid)
			}
		}
	}
	room.mu.Unlock()

	// Persist participant in database (async)
	go func() {
		if err := h.repo.AddParticipant(context.Background(), c.sessionID, models.ReviewParticipant{
			UserID:   c.userID,
			UserName: c.userName,
			Color:    c.color,
			JoinedAt: time.Now(),
		}); err != nil {
			log.Printf("review hub: failed to add participant to DB [%s] %s: %v", c.sessionID, c.userID, err)
		}
	}()
}

func (h *ReviewHub) handleLeave(c *ReviewClient) {
	h.mu.RLock()
	room, ok := h.rooms[c.sessionID]
	h.mu.RUnlock()
	if !ok {
		log.Printf("review hub: room not found on leave [%s]", c.sessionID)
		return
	}

	room.mu.Lock()
	if _, exists := room.clients[c.userID]; exists {
		delete(room.clients, c.userID)
		close(c.send)
		log.Printf("review hub: client removed [%s] %s, remaining=%d", c.sessionID, c.userID, len(room.clients))
	}

	// Notify remaining peers that someone left
	leaveMsg, _ := json.Marshal(ReviewOutgoing{
		Type:     "peer-left",
		From:     c.userID,
		FromName: c.userName,
	})
	for uid, peer := range room.clients {
		select {
		case peer.send <- leaveMsg:
			log.Printf("review hub: sent peer-left to %s about %s", uid, c.userID)
		default:
			log.Printf("review hub: failed to send peer-left to %s (buffer full)", uid)
		}
	}
	room.mu.Unlock()

	// Remove participant from database (async)
	go func() {
		if err := h.repo.RemoveParticipant(context.Background(), c.sessionID, c.userID); err != nil {
			log.Printf("review hub: failed to remove participant from DB [%s] %s: %v", c.sessionID, c.userID, err)
		}
	}()
}

func (h *ReviewHub) handleMessage(evt *reviewEvent) {
	h.mu.RLock()
	room, ok := h.rooms[evt.client.sessionID]
	h.mu.RUnlock()
	if !ok {
		log.Printf("review hub: room not found for message [%s]", evt.client.sessionID)
		return
	}

	out, _ := json.Marshal(ReviewOutgoing{
		Type:       evt.msg.Type,
		From:       evt.client.userID,
		FromName:   evt.client.userName,
		To:         evt.msg.To,
		Payload:    evt.msg.Payload,
		AudioMuted: evt.msg.AudioMuted,
		VideoMuted: evt.msg.VideoMuted,
	})

	room.mu.Lock()
	defer room.mu.Unlock()

	if evt.msg.To != "" {
		// Peer-to-peer message
		if peer, ok := room.clients[evt.msg.To]; ok {
			select {
			case peer.send <- out:
				log.Printf("review hub: relayed %s from %s to %s [%s]", evt.msg.Type, evt.client.userID, evt.msg.To, evt.client.sessionID)
			default:
				log.Printf("review hub: send buffer full for %s (type=%s)", evt.msg.To, evt.msg.Type)
			}
		} else {
			log.Printf("review hub: target peer not found %s (type=%s)", evt.msg.To, evt.msg.Type)
		}
	} else {
		// Broadcast message (mute-state, etc.)
		count := 0
		for uid, peer := range room.clients {
			if uid != evt.client.userID {
				select {
				case peer.send <- out:
					count++
				default:
					log.Printf("review hub: send buffer full for %s (broadcast)", uid)
				}
			}
		}
		log.Printf("review hub: broadcast %s from %s to %d peers [%s]", evt.msg.Type, evt.client.userID, count, evt.client.sessionID)
	}
}
