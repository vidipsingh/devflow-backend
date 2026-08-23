
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

type eventType int

const (
	eventOp eventType = iota
	eventCursor
)

type hubEvent struct {
	eventType eventType
	client    *Client
	msg       IncomingMsg
}

// session holds the live in-memory state for one pair programming session.
type session struct {
	mu            sync.Mutex
	clients       map[*Client]bool
	document      []rune
	version       int
	ops           []models.OTOperation // all ops since session start (for OT transform)
	lastSavedVer  int                  // version of the last successfully persisted snapshot
}

func newSession(initialDoc string) *session {
	return &session{
		clients:  make(map[*Client]bool),
		document: []rune(initialDoc),
	}
}

func (s *session) broadcast(msg []byte, except *Client) {
	for c := range s.clients {
		if c == except {
			continue
		}
		select {
		case c.send <- msg:
		default:
			close(c.send)
			delete(s.clients, c)
		}
	}
}

func (s *session) sendTo(c *Client, msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

// Hub manages all active pair sessions.
type Hub struct {
	mu         sync.RWMutex
	sessions   map[string]*session
	register   chan *Client
	unregister chan *Client
	incoming   chan *hubEvent
	pairRepo   *repository.PairRepo
}

var GlobalHub *Hub

func NewHub(pairRepo *repository.PairRepo) *Hub {
	return &Hub{
		sessions:   make(map[string]*session),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		incoming:   make(chan *hubEvent, 512),
		pairRepo:   pairRepo,
	}
}

// GetOrCreateSession returns the in-memory session, loading doc from DB if needed.
func (h *Hub) GetOrCreateSession(sessionID, initialDoc string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[sessionID]; ok {
		return s
	}
	s := newSession(initialDoc)
	h.sessions[sessionID] = s
	return s
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.handleRegister(c)
		case c := <-h.unregister:
			h.handleUnregister(c)
		case evt := <-h.incoming:
			switch evt.eventType {
			case eventOp:
				h.handleOp(evt)
			case eventCursor:
				h.handleCursor(evt)
			}
		}
	}
}

func (h *Hub) handleRegister(c *Client) {
	h.mu.RLock()
	s, ok := h.sessions[c.sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	s.mu.Lock()
	s.clients[c] = true
	doc := string(s.document)
	ver := s.version
	s.mu.Unlock()

	// Send current authoritative state to the joining client
	out, _ := json.Marshal(OutgoingMsg{
		Type:     "state",
		Document: doc,
		Version:  ver,
	})
	s.sendTo(c, out)
}

func (h *Hub) handleUnregister(c *Client) {
	h.mu.RLock()
	s, ok := h.sessions[c.sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	s.mu.Lock()
	if _, exists := s.clients[c]; exists {
		delete(s.clients, c)
		close(c.send)
	}
	// Capture current state before releasing the mutex for persistence
	doc := string(s.document)
	ver := s.version
	s.mu.Unlock()

	// Notify remaining participants
	out, _ := json.Marshal(OutgoingMsg{
		Type:     "leave",
		UserID:   c.userID,
		UserName: c.userName,
	})
	s.mu.Lock()
	s.broadcast(out, nil)
	s.mu.Unlock()

	// Persist document state and remove participant in background
	sessionID := c.sessionID
	userID    := c.userID
	go func() {
		ctx := context.Background()
		if err := h.pairRepo.UpdateDocument(ctx, sessionID, doc, ver); err != nil {
			log.Printf("pair disconnect flush error: %v", err)
		}
		h.pairRepo.RemoveParticipant(ctx, sessionID, userID)
	}()
}

func (h *Hub) handleOp(evt *hubEvent) {
	h.mu.RLock()
	s, ok := h.sessions[evt.client.sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	s.mu.Lock()

	incoming := models.OTOperation{
		ClientSeq:  evt.msg.ClientSeq,
		Index:      evt.msg.Index,
		Insert:     evt.msg.Insert,
		Delete:     evt.msg.Delete,
		AuthorID:   evt.client.userID,
		AuthorName: evt.client.userName,
	}

	// Transform incoming op against all server ops that the client hasn't seen yet
	// (i.e., ops with ServerSeq > client's last known ServerSeq = ClientSeq in our protocol)
	transformed := incoming
	for _, historical := range s.ops {
		if historical.AuthorID == evt.client.userID {
			continue // this client's own prior ops are already reflected in its local doc
		}
		if historical.ServerSeq > evt.msg.BaseVersion {
			transformed = Transform(historical, transformed)
		}
	}

	// Apply to server document
	s.document = ApplyOp(s.document, transformed)
	s.version++
	transformed.ServerSeq = s.version
	s.ops = append(s.ops, transformed)

	// Capture snapshot data while holding the lock
	doc         := string(s.document)
	ver         := s.version
	sessionID   := evt.client.sessionID
	shouldSave  := ver > s.lastSavedVer // always true since ver increments by 1 each time
	if shouldSave {
		s.lastSavedVer = ver
	}

	s.mu.Unlock()

	// ACK to the sender
	ack, _ := json.Marshal(OutgoingMsg{
		Type:      "ack",
		ServerSeq: ver,
		Op:        transformed,
	})
	s.sendTo(evt.client, ack)

	// Broadcast transformed op to all other clients
	bcast, _ := json.Marshal(OutgoingMsg{
		Type:       "op",
		ServerSeq:  ver,
		Op:         transformed,
		AuthorID:   transformed.AuthorID,
		AuthorName: transformed.AuthorName,
	})
	s.mu.Lock()
	s.broadcast(bcast, evt.client)
	s.mu.Unlock()

	// Persist every op to MongoDB in the background.
	// We pass doc+ver by value (captured above under lock) so there's no race.
	if shouldSave {
		go func(d string, v int, sid string) {
			ctx := context.Background()
			if err := h.pairRepo.UpdateDocument(ctx, sid, d, v); err != nil {
				log.Printf("pair snapshot error (v%d): %v", v, err)
			}
		}(doc, ver, sessionID)
	}
}

func (h *Hub) handleCursor(evt *hubEvent) {
	h.mu.RLock()
	s, ok := h.sessions[evt.client.sessionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	out, _ := json.Marshal(OutgoingMsg{
		Type:     "cursor",
		UserID:   evt.client.userID,
		UserName: evt.client.userName,
		Index:    evt.msg.Index,
	})

	s.mu.Lock()
	s.broadcast(out, evt.client)
	s.mu.Unlock()
}

// RegisterClient is called from the HTTP handler to add a client to the hub.
func (h *Hub) RegisterClient(c *Client) {
	h.register <- c
}
