package handlers

import (
	"net/http"

	"devflow-backend/internal/database"
	"devflow-backend/internal/repository"
	"devflow-backend/internal/service"
	ws "devflow-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func pairService() *service.PairService {
	repo := repository.NewPairRepo(database.GetDB())
	return service.NewPairService(repo)
}

// POST /api/v1/pair-sessions
func CreatePairSession(c *gin.Context) {
	ownerID := c.GetString("userID")
	username := c.GetString("username")
	_ = username

	var body struct {
		RepoID   string `json:"repoId" binding:"required"`
		FileID   string `json:"fileId"`
		FilePath string `json:"filePath"`
		Document string `json:"document"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := pairService().CreateSession(c.Request.Context(), ownerID, body.RepoID, body.FileID, body.FilePath, body.Document)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure hub has session
	ws.GlobalHub.GetOrCreateSession(sess.ID.Hex(), sess.Document)

	c.JSON(http.StatusCreated, gin.H{"data": sess})
}

// GET /api/v1/pair-sessions/:sessionId
func GetPairSession(c *gin.Context) {
	sess, err := pairService().GetSession(c.Request.Context(), c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sess == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sess})
}

// POST /api/v1/pair-sessions/:sessionId/join
func JoinPairSession(c *gin.Context) {
	userID := c.GetString("userID")
	username := c.GetString("username")

	sess, err := pairService().JoinSession(c.Request.Context(), c.Param("sessionId"), userID, username)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "session not found" {
			status = http.StatusNotFound
		} else if err.Error() == "session has ended" {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Ensure hub has session
	ws.GlobalHub.GetOrCreateSession(sess.ID.Hex(), sess.Document)

	c.JSON(http.StatusOK, gin.H{"data": sess})
}

// POST /api/v1/pair-sessions/:sessionId/end
func EndPairSession(c *gin.Context) {
	userID := c.GetString("userID")

	if err := pairService().EndSession(c.Request.Context(), c.Param("sessionId"), userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "session not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the session owner can end the session" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session ended"})
}

// GET /api/v1/pair-sessions
func ListPairSessions(c *gin.Context) {
	ownerID := c.GetString("userID")
	sessions, err := pairService().ListSession(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

// GET /ws/pair/:sessionId?token=<jwt>
func PairSessionWS(c *gin.Context) {
	// Token is passed as query param for browser WS compatibility
	userID := c.GetString("userID")
	username := c.GetString("username")
	sessionID := c.Param("sessionId")

	// Verify session exists
	svc := pairService()
	sess, err := svc.GetSession(c.Request.Context(), sessionID)
	if err != nil || sess == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(ws.GlobalHub, sessionID, userID, username, conn)
	ws.GlobalHub.RegisterClient(client)

	go client.WritePump()
	client.ReadPump()
}
