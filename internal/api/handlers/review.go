package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
	ws "devflow-backend/internal/websocket"
)

var reviewColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7", "#DDA0DD",
}

func colorForUser(userID string) string {
	if len(userID) == 0 {
		return reviewColors[0]
	}
	return reviewColors[int(userID[len(userID)-1])%len(reviewColors)]
}

type ReviewHandler struct {
	repo *repository.ReviewRepo
	hub  *ws.ReviewHub
}

func NewReviewHandler(repo *repository.ReviewRepo, hub *ws.ReviewHub) *ReviewHandler {
	return &ReviewHandler{repo: repo, hub: hub}
}

// POST /api/v1/repos/:repoId/pulls/:prId/review-sessions
func (h *ReviewHandler) CreateReviewSession(c *gin.Context) {
	prOID, err := bson.ObjectIDFromHex(c.Param("prId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prId"})
		return
	}
	repoOID, err := bson.ObjectIDFromHex(c.Param("repoId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoId"})
		return
	}
	sess := &models.ReviewSession{
		PRID:      prOID,
		RepoID:    repoOID,
		OwnerID:   c.GetString("userID"),
		OwnerName: c.GetString("username"),
		Status:    "waiting",
	}
	if err := h.repo.Create(c, sess); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.hub.GetOrCreateRoom(sess.ID.Hex())
	c.JSON(http.StatusCreated, gin.H{"data": sess})
}

// GET /api/v1/repos/:repoId/pulls/:prId/review-sessions
func (h *ReviewHandler) ListReviewSessions(c *gin.Context) {
	sessions, err := h.repo.ListByPR(c, c.Param("prId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

// GET /api/v1/repos/:repoId/pulls/:prId/review-sessions/:sessionId
func (h *ReviewHandler) GetReviewSession(c *gin.Context) {
	sess, err := h.repo.GetByID(c, c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sess})
}

// POST /api/v1/repos/:repoId/pulls/:prId/review-sessions/:sessionId/end
func (h *ReviewHandler) EndReviewSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	callerID  := c.GetString("userID")

	// Verify the caller is the session owner
	sess, err := h.repo.GetByID(c, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	if sess.OwnerID != callerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the session owner can end the session"})
		return
	}

	// Mark ended in DB
	if err := h.repo.End(c, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast session-ended to all WS clients and tear down the room
	h.hub.EndRoom(sessionID)

	c.JSON(http.StatusOK, gin.H{"message": "ended"})
}

var reviewUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// GET /api/v1/ws/review/:sessionId?token=...
func (h *ReviewHandler) WebSocketSignaling(c *gin.Context) {
	sessionID := c.Param("sessionId")
	userID    := c.GetString("userID")
	userName  := c.GetString("username")

	h.hub.GetOrCreateRoom(sessionID)

	conn, err := reviewUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewReviewClient(h.hub, conn, sessionID, userID, userName, colorForUser(userID))
	h.hub.RegisterReviewClient(client)

	go client.WritePump()
	client.ReadPump()
}
