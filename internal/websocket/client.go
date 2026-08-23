package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 32 * 1024 // 32KB
)

type IncomingMsg struct {
	Type      string `json:"type"`
	ClientSeq int    `json:"clientSeq"`
	BaseVersion int    `json:"baseVersion"`
	Index     int    `json:"index"`
	Insert    string `json:"insert"`
	Delete    int    `json:"delete"`
}

// OutgoingMsg is the JSON structure the server sends to clients
type OutgoingMsg struct {
	Type        string      `json:"type"`
	ServerSeq   int         `json:"serverSeq,omitempty"`
	Op          interface{} `json:"op,omitempty"`
	AuthorID    string      `json:"authorId,omitempty"`
	AuthorName  string      `json:"authorName,omitempty"`
	UserID      string      `json:"userId,omitempty"`
	UserName    string      `json:"userName,omitempty"`
	Index       int         `json:"index,omitempty"`
	Document    string      `json:"document,omitempty"`
	Version     int         `json:"version,omitempty"`
	Participants interface{} `json:"participants,omitempty"`
	Message     string      `json:"message,omitempty"`
}

// Client represents a single WebSocket connection in a pair session
type Client struct {
	hub       *Hub
	sessionID string
	userID    string
	userName  string
	conn      *websocket.Conn
	send      chan []byte
}

func NewClient(hub *Hub, sessionID, userID, userName string, conn *websocket.Conn) *Client {
	return &Client{
		hub:       hub,
		sessionID: sessionID,
		userID:    userID,
		userName:  userName,
		conn:      conn,
		send:      make(chan []byte, 256),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func ()  {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		var msg IncomingMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "op":
			c.hub.incoming <- &hubEvent{
				eventType: eventOp,
				client:    c,
				msg:       msg,
			}
		case "cursor":
			c.hub.incoming <- &hubEvent{
				eventType: eventCursor,
				client:    c,
				msg:       msg,
			}
		case "ping":
			out, _ := json.Marshal(OutgoingMsg{Type: "pong"})
			c.send <- out
		}
	}
}

// WritePump pumps messages from the send channel to the WebSocket connection.
func (c *Client) WritePump() {
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
