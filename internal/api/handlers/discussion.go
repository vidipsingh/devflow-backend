package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"devflow-backend/internal/service"
	ws "devflow-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// serviceErrStatus maps well-known service errors to HTTP status codes.
func serviceErrStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrTeamNotFound),
		errors.Is(err, service.ErrRepoNotFound),
		errors.Is(err, service.ErrDiscussionNotFound),
		errors.Is(err, service.ErrReplyNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrTeamForbidden),
		errors.Is(err, service.ErrInsufficientRole):
		return http.StatusForbidden
	case errors.Is(err, service.ErrNotMember):
		return http.StatusForbidden
	case errors.Is(err, service.ErrTeamDuplicate),
		errors.Is(err, service.ErrRepoAlreadyInTeam):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// POST /api/v1/teams/:slug/discussions
func CreateDiscussion(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		var body struct {
			Title string `json:"title" binding:"required"`
			Body  string `json:"body"  binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		d, err := service.CreateDiscussion(c.Request.Context(), callerID, slug, body.Title, body.Body)
		if err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": d})
		BroadcastDiscussionEvent(hub, slug, "new_discussion", d)
	}
}

// GET /api/v1/teams/:slug/discussions
func ListDiscussions(c *gin.Context) {
	callerID := mustCallerID(c)
	slug      := c.Param("slug")
	limit, _  := strconv.ParseInt(c.DefaultQuery("limit", "30"), 10, 64)
	skip, _   := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	list, err := service.ListDiscussions(c.Request.Context(), callerID, slug, limit, skip)
	if err != nil {
		c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GET /api/v1/teams/:slug/discussions/:discussionId
func GetDiscussion(c *gin.Context) {
	callerID := mustCallerID(c)
	slug := c.Param("slug")
	dID := c.Param("discussionId")
	d, replies, err := service.GetDiscussion(c.Request.Context(), callerID, slug, dID)
	if err != nil {
		c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"discussion": d, "replies": replies}})
}

// PATCH /api/v1/teams/:slug/discussions/:discussionId
func UpdateDiscussion(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		var body struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		d, err := service.UpdateDiscussion(c.Request.Context(), callerID, slug, dID, body.Title, body.Body)
		if err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": d})
		BroadcastDiscussionEvent(hub, slug, "update_discussion", d)
	}
}

// PATCH /api/v1/teams/:slug/discussions/:discussionId/pin
func PinDiscussion(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		var body struct{ Pinned bool `json:"pinned"` }
		_ = c.ShouldBindJSON(&body)
		if err := service.PinDiscussion(c.Request.Context(), callerID, slug, dID, body.Pinned); err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		BroadcastDiscussionEvent(hub, slug, "pin_discussion", gin.H{"discussionId": dID, "pinned": body.Pinned})
	}
}

// PATCH /api/v1/teams/:slug/discussions/:discussionId/resolve
func ResolveDiscussion(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		var body struct{ Resolved bool `json:"resolved"` }
		_ = c.ShouldBindJSON(&body)
		if err := service.ResolveDiscussion(c.Request.Context(), callerID, slug, dID, body.Resolved); err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		BroadcastDiscussionEvent(hub, slug, "resolve_discussion", gin.H{"discussionId": dID, "resolved": body.Resolved})
	}
}

// DELETE /api/v1/teams/:slug/discussions/:discussionId
func DeleteDiscussion(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		if err := service.DeleteDiscussion(c.Request.Context(), callerID, slug, dID); err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		BroadcastDiscussionEvent(hub, slug, "delete_discussion", gin.H{"discussionId": dID})
	}
}

// POST /api/v1/teams/:slug/discussions/:discussionId/replies
func AddReply(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		var body struct {
			Body string `json:"body" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		r, err := service.AddReply(c.Request.Context(), callerID, slug, dID, body.Body)
		if err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": r})
		BroadcastDiscussionEvent(hub, slug, "new_reply", r)
	}
}

// DELETE /api/v1/teams/:slug/discussions/:discussionId/replies/:replyId
func DeleteReply(hub *ws.DiscussionHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := mustCallerID(c)
		slug := c.Param("slug")
		dID  := c.Param("discussionId")
		rID  := c.Param("replyId")
		if err := service.DeleteReply(c.Request.Context(), callerID, slug, dID, rID); err != nil {
			c.JSON(serviceErrStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		BroadcastDiscussionEvent(hub, slug, "delete_reply", gin.H{"replyId": rID, "discussionId": dID})
	}
}

// mustCallerID extracts the caller's ObjectID from the gin context (set by RequireAuth middleware).
func mustCallerID(c *gin.Context) bson.ObjectID {
	userIDStr := c.GetString("userID")
	oid, _ := bson.ObjectIDFromHex(userIDStr)
	return oid
}