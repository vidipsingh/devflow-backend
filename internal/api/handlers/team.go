package handlers

import (
	"strconv"

	"devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func callerTeamFields(c *gin.Context) (bson.ObjectID, string, string, bool) {
	userIDStr := c.GetString("userID")
	username := c.GetString("username")
	oid, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return bson.NilObjectID, "", "", false
	}
	// Avatar is not in the JWT; fetch from context if set, else ""
	return oid, username, "", true
}

// POST /teams
func CreateTeam(c *gin.Context) {
	callerID, username, avatar, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req models.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	team, err := service.CreateTeam(c.Request.Context(), callerID, username, avatar, req)
	if err != nil {
		if err == service.ErrTeamDuplicate {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, team)
}

// GET /teams (my teams)
func ListMyTeams(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	teams, err := service.ListMyTeams(c.Request.Context(), callerID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, teams)
}

// GET /teams/discover
func ListPublicTeams(c *gin.Context) {
	search := c.Query("q")
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)

	teams, err := service.ListPublicTeams(c.Request.Context(), search, limit, skip)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, teams)
}

// GET /teams/:slug
func GetTeam(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	team, err := service.GetTeam(c.Request.Context(), callerID, c.Param("slug"))
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, team)
}

// PATCH /teams/:slug
func UpdateTeam(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req models.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	team, err := service.UpdateTeam(c.Request.Context(), callerID, c.Param("slug"), req)
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, team)
}

// DELETE /teams/:slug
func DeleteTeam(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := service.DeleteTeam(c.Request.Context(), callerID, c.Param("slug")); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// GET /teams/:slug/sub-teams
func ListSubTeams(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	teams, err := service.ListSubTeams(c.Request.Context(), callerID, c.Param("slug"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, teams)
}

// Members

// GET /teams/:slug/members
func ListTeamMembers(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	role := c.Query("role")
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 64)
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)

	members, err := service.ListTeamMembers(c.Request.Context(), callerID, c.Param("slug"), role, limit, skip)
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, members)
}

// PATCH /teams/:slug/members/:username/role
func UpdateMemberRole(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req models.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := service.UpdateMemberRole(c.Request.Context(), callerID, c.Param("slug"), c.Param("username"), req.Role); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		if err == service.ErrNotMember {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"updated": true})
}

// DELETE /teams/:slug/members/:username
func RemoveMember(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := service.RemoveMember(c.Request.Context(), callerID, c.Param("slug"), c.Param("username")); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"removed": true})
}

// PATCH /teams/:slug/members/:username/ban
func BanMember(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var body struct {
		Ban bool `json:"ban"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := service.BanMember(c.Request.Context(), callerID, c.Param("slug"), c.Param("username"), body.Ban); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"banned": body.Ban})
}

// POST /teams/:slug/leave
func LeaveTeam(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := service.LeaveTeam(c.Request.Context(), callerID, c.Param("slug")); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"left": true})
}

// POST /teams/:slug/transfer
func TransferOwnership(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var body struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := service.TransferOwnership(c.Request.Context(), callerID, c.Param("slug"), body.Username); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"transferred": true})
}

// Invites

// POST /teams/:slug/invites
func InviteMember(c *gin.Context) {
	callerID, callerUsername, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req models.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	inv, err := service.InviteMember(c.Request.Context(), callerID, callerUsername, c.Param("slug"), req)
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		if err == service.ErrAlreadyMember {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, inv)
}

// GET /teams/:slug/invites
func ListPendingInvites(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	invs, err := service.ListPendingInvites(c.Request.Context(), callerID, c.Param("slug"))
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, invs)
}

// DELETE /teams/:slug/invites/:inviteId
func RevokeInvite(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := service.RevokeInvite(c.Request.Context(), callerID, c.Param("slug"), c.Param("inviteId")); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"revoked": true})
}

// GET /teams/:slug/join-requests/me
func GetMyJoinRequest(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	req, err := service.GetMyJoinRequest(c.Request.Context(), callerID, c.Param("slug"))
	if err == service.ErrTeamNotFound {
		response.NotFound(c, err.Error())
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, req) // nil = no pending request
}

// GET /teams/:slug/members/me
func GetMyMembership(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	m, err := service.GetMyMembership(c.Request.Context(), callerID, c.Param("slug"))
	if err == service.ErrTeamNotFound {
		response.NotFound(c, err.Error())
		return
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// nil membership = caller is not a member; return explicit null body
	response.OK(c, m)
}

// GET /teams/invites/me  (all my pending invites)
func ListMyInvites(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	email := c.GetString("email")

	invs, err := service.ListMyInvites(c.Request.Context(), callerID, email)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, invs)
}

// POST /teams/invites/:token/accept
func AcceptInvite(c *gin.Context) {
	callerID, username, avatar, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	team, err := service.AcceptInvite(c.Request.Context(), callerID, username, avatar, c.Param("token"))
	if err != nil {
		if err == service.ErrInviteNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrInviteExpired {
			response.BadRequest(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, team)
}

// POST /teams/invites/:token/decline
func DeclineInvite(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := service.DeclineInvite(c.Request.Context(), callerID, c.Param("token")); err != nil {
		if err == service.ErrInviteNotFound {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"declined": true})
}

// Join Requests

// POST /teams/:slug/join-requests
func RequestToJoin(c *gin.Context) {
	callerID, username, avatar, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var body models.RequestToJoinRequest
	_ = c.ShouldBindJSON(&body)

	jr, err := service.RequestToJoin(c.Request.Context(), callerID, username, avatar, c.Param("slug"), body.Message)
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamInviteOnly {
			response.Forbidden(c, err.Error())
			return
		}
		if err == service.ErrAlreadyMember {
			response.BadRequest(c, err.Error())
			return
		}
		if err == service.ErrJoinRequestExists {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	if jr == nil {
		// open team: joined directly
		response.OK(c, gin.H{"joined": true})
		return
	}
	response.Created(c, jr)
}

// GET /teams/:slug/join-requests
func ListJoinRequests(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	reqs, err := service.ListJoinRequests(c.Request.Context(), callerID, c.Param("slug"))
	if err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, reqs)
}

// PATCH /teams/:slug/join-requests/:requestId
func ReviewJoinRequest(c *gin.Context) {
	callerID, _, _, ok := callerTeamFields(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req models.ReviewJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := service.ReviewJoinRequest(c.Request.Context(), callerID, c.Param("slug"), c.Param("requestId"), req.Action); err != nil {
		if err == service.ErrTeamNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrTeamForbidden {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"reviewed": true})
}
