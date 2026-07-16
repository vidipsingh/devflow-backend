package handlers

import (
	"net/http"
	"strconv"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// POST /api/v1/repositories/:name/files
func UploadFile(c * gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}

	var req models.UploadFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	commit, err := service.UploadFile(
		c.Request.Context(),
		ownerID,
		c.GetString("username"),
		c.Param("name"),
		req,
	)
	if err == service.ErrRepoNotFound {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    commit,
	})
}

// GET /api/v1/repositories/:name/tree?ref=main&path=/
func GetTree(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}

	entries, err := service.GetTree(
		c.Request.Context(),
		ownerID,
		c.Param("name"),
		c.DefaultQuery("ref", ""),
		c.DefaultQuery("path", "."),
	)
	if err == service.ErrRepoNotFound {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"tree": entries, "total": len(entries)})
}

// GET /api/v1/repositories/:name/blob?ref=main&path=README.md
func GetBlob(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}

	content, meta, err := service.GetBlob(
		c.Request.Context(),
		ownerID,
		c.Param("name"),
		c.DefaultQuery("ref", ""),
		c.Query("path"),
	)
	if err == service.ErrFileNotFound || err == service.ErrRepoNotFound {
		response.NotFound(c, "file not found")
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// If client accepts raw bytes, serve binary
	accept := c.GetHeader("Accept")
	if accept == "application/octet-stream" {
		c.Data(http.StatusOK, meta.MimeType, content)
		return
	}

	// Default: return JSON with content + metadata
	response.OK(c, gin.H{
		"path":     meta.Path,
        "name":     meta.Name,
        "size":     meta.Size,
        "mimeType": meta.MimeType,
        "encoding": meta.Encoding,
        "sha":      meta.SHA,
        "content":  string(content),
	})
}

func GetCommits(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.ParseInt(limitStr, 10, 64)

	commits, err := service.GetCommits(
		c.Request.Context(),
		ownerID,
		c.Param("name"),
		c.DefaultQuery("ref", ""),
		limit,
	)
	if err == service.ErrRepoNotFound{
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{"commits": commits, "total": len(commits)})
}