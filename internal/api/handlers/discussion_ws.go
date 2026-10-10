package handlers

import (
	"net/http"

	"devflow-backend/internal/service"
	ws "devflow-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var discUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WsDiscussions handles GET /api/v1/teams/:slug/discussions/ws
// The client connects here to receive real-time discussion events.
func WsDiscussions(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		callerID := mustCallerID(c)
		userName := c.GetString("username")
		if userName == "" {
			userName = callerID.Hex()
		}

		// Only team members may connect.
		if _, err := service.GetMyMembership(c.Request.Context(), callerID, slug); err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}

		conn, err := discUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		// Ensure room exists before handing off to pumps.
		hub.GetOrCreateRoom(slug)

		client := ws.NewDiscussionClient(hub, conn, slug, callerID.Hex(), userName)
		hub.RegisterClient(client)

		go client.WritePump()
		client.ReadPump() // blocks until disconnected
	}
}

func BroadcastDiscussionEvent(hub *ws.DiscussionHub, slug string, eventType string, payload interface{}) {
	if hub == nil {
		return
	}
	hub.Broadcast(slug, ws.DiscussionEvent{
		Type:    eventType,
		Payload: payload,
	})
}
