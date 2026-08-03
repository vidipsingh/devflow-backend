package handlers

import (
	"errors"
	"strconv"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// GET /repositories/:name/pulls?state=open
func ListPRs(c *gin.Context) {
	ownerID,  ok := mustOwnerID(c)
	if !ok { return }
	prs, err := service.ListPRs(c.Request.Context(), ownerID, c.Param("name"), c.Query("state"))
	if err != nil {
		response.NotFound(c, "repository not found"); return
	}
	response.OK(c, gin.H{"pullRequests": prs, "total": len(prs)})
}

// POST /repositories/:name/pulls
func CreatePR(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	var req models.CreatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error()); return
	}
	pr, err := service.CreatePR(c.Request.Context(), ownerID, c.GetString("username"), c.Param("name"), req)
	if errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "repository not found"); 
	}
	if err != nil {
		response.InternalError(c, err.Error()); return
	}
	response.Created(c, pr)
}

// GET /repositories/:name/pulls/:number
func GetPR(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	num, _ := strconv.Atoi(c.Param("number"))
	pr, err := service.GetPR(c.Request.Context(), ownerID, c.Param("name"), num)
	if errors.Is(err, service.ErrPRNotFound) || errors.Is(err, service.ErrRepoNotFound) {
		response.NotFound(c, "pull request not found"); return
	}
	if err != nil {
		response.InternalError(c, err.Error()); return
	}
	response.OK(c, pr)
}

// PATCH /repositories/:name/pulls/:number
func UpdatePR(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	var req models.UpdatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error()); return
	}
	num, _ := strconv.Atoi(c.Param("number"))
	pr, err := service.UpdatePR(c.Request.Context(), ownerID, c.Param("name"), num, req)
	if errors.Is(err, service.ErrPRNotFound) { response.NotFound(c, "pull request not found"); return }
	if errors.Is(err, service.ErrPRForbidden) { response.Unauthorized(c); return }
	if err != nil { response.InternalError(c, err.Error()); return }
	response.OK(c, pr)
}

// PATCH /repositories/:name/pulls/:number
func DeletePR(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	num, _ := strconv.Atoi(c.Param("number"))
	err := service.DeletePR(c.Request.Context(), ownerID, c.Param("name"), num)
	if errors.Is(err, service.ErrPRNotFound) { response.NotFound(c, "pull request not found"); return }
	if errors.Is(err, service.ErrPRForbidden) { response.Unauthorized(c); return }
	if err != nil { response.InternalError(c, err.Error()); return }
	response.OK(c, gin.H{"message": "pull request deleted"})
}

// POST /repositories/:name/pulls/:number/merge
func MergePR(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	var req models.MergePRRequest
	_ = c.ShouldBindJSON(&req)
	if req.Method == "" { req.Method = "merge" }
	num, _ := strconv.Atoi(c.Param("number"))
	pr, err := service.MergePR(c.Request.Context(), ownerID, c.Param("name"), num, req.Method)
	if errors.Is(err, service.ErrPRNotFound) { response.NotFound(c, "pull request not found"); return }
	if errors.Is(err, service.ErrPRForbidden) { response.Unauthorized(c); return }
	if err != nil { response.InternalError(c, err.Error()); return }
	response.OK(c, pr)
}

// POST /repositories/:name/pulls/:number/comments
func AddPRComment(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	var req models.CreatePRCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error()); return
	}
	num, _ := strconv.Atoi(c.Param("number"))
	repo, _ := service.GetPR(c.Request.Context(), ownerID, c.Param("name"), num)
	if repo == nil { response.NotFound(c, "pull request not found") }

	comment := models.PRComment{
		AuthorID:   ownerID,
		AuthorName: c.GetString("username"),
		Body:       req.Body,
		FilePath:   req.FilePath,
		LineNumber: req.LineNumber,
	}
	if err := service.AddPRCommentDirect(c.Request.Context(), repo.ID, comment); err != nil {
		response.InternalError(c, err.Error()); return
	}
	response.Created(c, gin.H{"message": "comment added"})
}

// PATCH /repositories/:name/pulls/:number/comments/:commentId
func UpdatePRComment(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	var req models.UpdatePRCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error()); return
	}
	num, _ := strconv.Atoi(c.Param("number"))
	pr, _ := service.GetPR(c.Request.Context(), ownerID, c.Param("name"), num)
	if pr == nil { response.NotFound(c, "pull request not found"); return }
	commentID, err := bson.ObjectIDFromHex(c.Param("commentId"))
	if err != nil { response.BadRequest(c, "invalid comment id"); return }
	if err := service.UpdatePRCommentDirect(c.Request.Context(), pr.ID, commentID, req.Body); err != nil {
		response.InternalError(c, err.Error()); return
	}
	response.OK(c, gin.H{"message": "comment added"})
}

// DELETE /repositories/:name/pulls/:number/comments/:commentId
func DeletePRComment(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok { return }
	num, _ := strconv.Atoi(c.Param("number"))
	pr, _ := service.GetPR(c.Request.Context(), ownerID, c.Param("name"), num)
	if pr == nil { response.NotFound(c, "pull request not found"); return }
	commentID, err := bson.ObjectIDFromHex(c.Param("commentId"))
	if err != nil { response.BadRequest(c, "invalid comment id"); return }
	if err := service.DeletePRCommentDirect(c.Request.Context(), pr.ID, commentID); err != nil {
		response.InternalError(c, err.Error()); return
	}
	response.OK(c, gin.H{"message": "comment deleted"})
}
