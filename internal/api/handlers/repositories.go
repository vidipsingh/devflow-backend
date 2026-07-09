package handlers

import (
	"errors"
	"net/http"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func mustOwnerID(c *gin.Context) (bson.ObjectID, bool) {
	raw := c.GetString("userID")
	id, err := bson.ObjectIDFromHex(raw)
	if err != nil {
		response.Unauthorized(c)
		return bson.ObjectID{}, false
	}
	return id, true
}

// GET /api/v1/repositories?visibility=public|private
func ListRepositories(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	repos, err := service.ListRepositories(c.Request.Context(), ownerID, c.Query("visibility"))
	if err != nil {
		response.InternalError(c, "failed to fetch repositories")
		return
	}
	response.OK(c, gin.H{"repositories": repos, "total": len(repos)})
}

// GET /api/v1/repositories/:name
func GetRepository(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	repo, err := service.GetRepository(c.Request.Context(), ownerID, c.Param("name"))
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch repository")
		return
	}
	response.OK(c, repo)
}

// POST /api/v1/repositories
func CreateRepository(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var req models.CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	repo, err := service.CreateRepository(c.Request.Context(), ownerID, c.GetString("username"), req)
	if errors.Is(err, service.ErrRepoDuplicate) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": "repository name already taken"})
		return
	}
	if err != nil {
		response.InternalError(c, "failed to create repository")
		return
	}
	response.Created(c, repo)
}

// PATCH /api/v1/repositories/:name
func UpdateRepository(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var req models.UpdateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	repo, err := service.UpdateRepository(c.Request.Context(), ownerID, c.Param("name"), req)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrRepoForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to update repository")
		return
	}
	response.OK(c, repo)
}

// DELETE /api/v1/repositories/:name
func DeleteRepository(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	err := service.DeleteRepository(c.Request.Context(), ownerID, c.Param("name"))
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrRepoForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to delete repository")
		return
	}
	response.OK(c, gin.H{"message": "repository deleted"})
}

// PATCH /api/v1/repositories/:name/pin
// Body: { "pinned": true|false }
func PinRepository(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var body struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	repo, err := service.PinRepository(c.Request.Context(), ownerID, c.Param("name"), body.Pinned)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrRepoForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to update pin state")
		return
	}
	response.OK(c, repo)
}
