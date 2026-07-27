package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)


// mustIssueNumber parses :number param as a positive integer
func mustIssueNumber(c *gin.Context) (int, bool) {
	n, err := strconv.Atoi(c.Param("number"))
	if err != nil || n < 1 {
		response.BadRequest(c, "invalid issue number")
		return 0, false
	}
	return n, true
}

// GET /api/v1/repositories/:name/issues?state=open|closed&page=1&limit=20
func ListIssues(c *gin.Context) {
	state := c.DefaultQuery("state", "")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	if page < 1 { 
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	issues, total, err :=  service.ListIssues(c.Request.Context(), c.Param("name"), state, page, limit)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch issues")
	}
	response.OK(c, gin.H{
		"issues": issues,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GET /api/v1/repositories/:name/issues/:number
func GetIssue(c *gin.Context) {
	number, ok := mustIssueNumber(c)
	if !ok{
		return
	}
	issue, err := service.GetIssue(c.Request.Context(), c.Param("name"), number)
	if errors.Is(err, service.ErrRepoNotFound) || errors.Is(err, service.ErrIssueNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch issue")
		return
	}
	response.OK(c, issue)
}

// POST /api/v1/repositories/:name/issues
func CreateIssue(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	var req models.CreateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	issue, err := service.CreateIssue(c.Request.Context(), callerID, c.GetString("username"), c.Param("name"), req)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to create issue")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": issue})
}

// PATCH /api/v1/repositories/:name/issues/:number
func UpdateIssue(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	var req models.UpdateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	issue, err := service.UpdateIssue(c.Request.Context(), callerID, c.Param("name"), number, req)
	if errors.Is(err, service.ErrIssueNotFound) || errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if errors.Is(err, service.ErrIssueForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to update issue")
		return
	}
	response.OK(c, issue)
}

// DELETE /api/v1/repositories/:name/issues/:number
func DeleteIssue(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	err := service.DeleteIssue(c.Request.Context(), callerID, c.Param("name"), number)
	if errors.Is(err, service.ErrIssueNotFound) || errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if errors.Is(err, service.ErrIssueForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, "failed to delete issue")
		return
	}
	response.OK(c, gin.H{"message": "issue deleted"})
}

// POST /api/v1/repositories/:name/issues/:number/comments
func AddComment(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	issue, err := service.AddComment(c.Request.Context(), callerID, c.GetString("username"), c.Param("name"), number, req)
	if errors.Is(err, service.ErrIssueNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, issue)
}

// PATCH /api/v1/repositories/:name/issues/:number/comments/:commentId
func UpdateComment(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	var req models.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	issue, err := service.UpdateComment(c.Request.Context(), callerID, c.Param("name"), number,  c.Param("commentId"), req)
	if errors.Is(err, service.ErrIssueNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if errors.Is(err, service.ErrIssueForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, issue)
}

// DELETE /api/v1/repositories/:name/issues/:number/comments/:commentId
func DeleteComment(c *gin.Context) {
	callerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	issue, err := service.DeleteComment(c.Request.Context(), callerID, c.Param("name"), number, c.Param("commentId"))
	if errors.Is(err, service.ErrIssueNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if errors.Is(err, service.ErrIssueForbidden) {
		response.Unauthorized(c)
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, issue)
}

func ReactToIssue(c *gin.Context) {
	number, ok := mustIssueNumber(c)
	if !ok {
		return
	}
	var req models.ReactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	issue, err := service.ReactToIssue(c.Request.Context(), c.Param("name"), number, req)
	if errors.Is(err, service.ErrIssueNotFound) {
		response.NotFound(c, "issue not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to update reaction")
		return
	}
	response.OK(c, issue)
}
