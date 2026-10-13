
package handlers

import (
	"errors"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GET /repositories/:name/labels
func ListRepoLabels(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	labels, err := service.ListRepoLabels(c.Request.Context(), callerID, c.Param("name"))
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch labels")
		return
	}
	response.OK(c, gin.H{"labels": labels})
}

// POST /repositories/:name/labels
func CreateRepoLabel(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var req models.CreateRepoLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	label, err := service.CreateRepoLabel(c.Request.Context(), callerID, c.Param("name"), req)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrLabelForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to create label")
		return
	}
	response.Created(c, label)
}

// PATCH /repositories/:name/labels/:labelId
func UpdateRepoLabel(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var req models.UpdateRepoLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	label, err := service.UpdateRepoLabel(c.Request.Context(), callerID, c.Param("name"), c.Param("labelId"), req)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrLabelNotFound) {
		response.NotFound(c, "label not found")
		return
	}
	if errors.Is(err, service.ErrLabelForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to update label")
		return
	}
	response.OK(c, label)
}

// DELETE /repositories/:name/labels/:labelId
func DeleteRepoLabel(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	err := service.DeleteRepoLabel(c.Request.Context(), callerID, c.Param("name"), c.Param("labelId"))
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if errors.Is(err, service.ErrLabelNotFound) {
		response.NotFound(c, "label not found")
		return
	}
	if errors.Is(err, service.ErrLabelForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to delete label")
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
