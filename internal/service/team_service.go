package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrTeamNotFound      = errors.New("team not found")
	ErrTeamForbidden     = errors.New("access denied")
	ErrTeamDuplicate     = errors.New("team slug already taken")
	ErrAlreadyMember     = errors.New("already a team member")
	ErrNotMember         = errors.New("not a member of this team")
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrJoinRequestExists = errors.New("join request already pending")
	ErrCannotRemoveOwner = errors.New("cannot remove the team owner")
	ErrTeamInviteOnly    = errors.New("this team is invite-only")
)

var teamSlugRe = regexp.MustCompile(`[^a-z0-9\-]`)

func teamSlugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	return teamSlugRe.ReplaceAllString(s, "")
}

// isTeamAdmin returns true if the member's role is owner or admin.
func isTeamAdmin(role string) bool {
	return role == models.TeamRoleOwner || role == models.TeamRoleAdmin
}

func CreateTeam(ctx context.Context, callerID bson.ObjectID, callerUsername, callerAvatar string, req models.CreateTeamRequest) (*models.Team, error) {
	slug := teamSlugify(req.Name)
	if slug == "" {
		return nil, errors.New("invalid team name")
	}

	existing, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTeamDuplicate
	}

	teamType := req.Type
	if teamType == "" {
		teamType = "organization"
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = models.TeamVisibilityPublic
	}
	joinPolicy := req.JoinPolicy
	if joinPolicy == "" {
		joinPolicy = models.TeamJoinInviteOnly
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	var parentID *bson.ObjectID
	if req.ParentID != "" {
		id, err := bson.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return nil, errors.New("invalid parentId")
		}
		parentID = &id
	}

	now := time.Now()
	team := &models.Team{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Bio:         req.Bio,
		Type:        teamType,
		Visibility:  visibility,
		JoinPolicy:  joinPolicy,
		Tags:        tags,
		ParentID:    parentID,
		MemberCount: 1,
		CreatedBy:   callerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repository.CreateTeam(ctx, team); err != nil {
		return nil, err
	}

	// Add creator as owner
	member := &models.TeamMember{
		TeamID:    team.ID,
		UserID:    callerID,
		Username:  callerUsername,
		Avatar:    callerAvatar,
		Role:      models.TeamRoleOwner,
		IsBanned:  false,
		JoinedAt:  now,
		UpdatedAt: now,
	}
	if err := repository.InsertTeamMember(ctx, member); err != nil {
		return nil, err
	}

	return team, nil
}

func GetTeam(ctx context.Context, callerID bson.ObjectID, slug string) (*models.Team, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}
	// Secret teams: only visible to members
	if team.Visibility == models.TeamVisibilitySecret {
		m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
		if m == nil {
			return nil, ErrTeamForbidden
		}
	}
	return team, nil
}

func ListMyTeams(ctx context.Context, userID bson.ObjectID) ([]models.Team, error) {
	return repository.FindTeamsByMember(ctx, userID, 100)
}

func ListPublicTeams(ctx context.Context, search string, limit, skip int64) ([]models.Team, error) {
	return repository.FindPublicTeams(ctx, search, limit, skip)
}

func ListSubTeams(ctx context.Context, callerID bson.ObjectID, parentSlug string) ([]models.Team, error) {
	parent, err := repository.FindTeamBySlug(ctx, parentSlug)
	if err != nil || parent == nil {
		return nil, ErrTeamNotFound
	}
	return repository.FindSubTeams(ctx, parent.ID)
}

func UpdateTeam(ctx context.Context, callerID bson.ObjectID, slug string, req models.UpdateTeamRequest) (*models.Team, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if m == nil || !isTeamAdmin(m.Role) {
		return nil, ErrTeamForbidden
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Bio != nil {
		update["bio"] = *req.Bio
	}
	if req.Avatar != nil {
		update["avatar"] = *req.Avatar
	}
	if req.Banner != nil {
		update["banner"] = *req.Banner
	}
	if req.Visibility != nil {
		update["visibility"] = *req.Visibility
	}
	if req.JoinPolicy != nil {
		update["joinPolicy"] = *req.JoinPolicy
	}
	if req.Tags != nil {
		update["tags"] = req.Tags
	}

	if err := repository.UpdateTeam(ctx, team.ID, update); err != nil {
		return nil, err
	}
	return repository.FindTeamBySlug(ctx, slug)
}

func DeleteTeam(ctx context.Context, callerID bson.ObjectID, slug string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil {
		return ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if m == nil || m.Role != models.TeamRoleOwner {
		return ErrTeamForbidden
	}
	return repository.DeleteTeam(ctx, team.ID)
}

// Members

func ListTeamMembers(ctx context.Context, callerID bson.ObjectID, slug, role string, limit, skip int64) ([]models.TeamMember, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	if team.Visibility == models.TeamVisibilitySecret {
		m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
		if m == nil {
			return nil, ErrTeamForbidden
		}
	}
	return repository.FindTeamMembers(ctx, team.ID, role, limit, skip)
}

func UpdateMemberRole(ctx context.Context, callerID bson.ObjectID, slug, targetUsername, newRole string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return ErrTeamForbidden
	}

	targetUser, err := repository.FindUserByUsername(ctx, targetUsername)
	if err != nil || targetUser == nil {
		return errors.New("user not found")
	}
	target, _ := repository.FindTeamMember(ctx, team.ID, targetUser.ID)
	if target == nil {
		return ErrNotMember
	}
	if target.Role == models.TeamRoleOwner {
		return errors.New("cannot change owner role directly; transfer ownership instead")
	}
	return repository.UpdateTeamMemberRole(ctx, team.ID, targetUser.ID, newRole)
}

func RemoveMember(ctx context.Context, callerID bson.ObjectID, slug, targetUsername string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return ErrTeamForbidden
	}

	targetUser, err := repository.FindUserByUsername(ctx, targetUsername)
	if err != nil || targetUser == nil {
		return errors.New("user not found")
	}
	target, _ := repository.FindTeamMember(ctx, team.ID, targetUser.ID)
	if target == nil {
		return ErrNotMember
	}
	if target.Role == models.TeamRoleOwner {
		return ErrCannotRemoveOwner
	}

	if err := repository.RemoveTeamMember(ctx, team.ID, targetUser.ID); err != nil {
		return err
	}
	_ = repository.IncrTeamStat(ctx, team.ID, "memberCount", -1)
	return nil
}

func BanMember(ctx context.Context, callerID bson.ObjectID, slug, targetUsername string, ban bool) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return ErrTeamForbidden
	}

	targetUser, err := repository.FindUserByUsername(ctx, targetUsername)
	if err != nil || targetUser == nil {
		return errors.New("user not found")
	}
	return repository.SetTeamMemberBanned(ctx, team.ID, targetUser.ID, ban)
}

func LeaveTeam(ctx context.Context, callerID bson.ObjectID, slug string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if m == nil {
		return ErrNotMember
	}
	if m.Role == models.TeamRoleOwner {
		return errors.New("owner cannot leave; transfer ownership or delete the team")
	}
	if err := repository.RemoveTeamMember(ctx, team.ID, callerID); err != nil {
		return err
	}
	_ = repository.IncrTeamStat(ctx, team.ID, "memberCount", -1)
	return nil
}

// TransferOwnership gives owner role to another member, demotes caller to admin.
func TransferOwnership(ctx context.Context, callerID bson.ObjectID, slug, newOwnerUsername string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || caller.Role != models.TeamRoleOwner {
		return ErrTeamForbidden
	}

	newOwner, err := repository.FindUserByUsername(ctx, newOwnerUsername)
	if err != nil || newOwner == nil {
		return errors.New("user not found")
	}
	target, _ := repository.FindTeamMember(ctx, team.ID, newOwner.ID)
	if target == nil {
		return ErrNotMember
	}

	if err := repository.UpdateTeamMemberRole(ctx, team.ID, newOwner.ID, models.TeamRoleOwner); err != nil {
		return err
	}
	return repository.UpdateTeamMemberRole(ctx, team.ID, callerID, models.TeamRoleAdmin)
}

// Invites

func InviteMember(ctx context.Context, callerID bson.ObjectID, callerUsername, slug string, req models.InviteMemberRequest) (*models.TeamInvite, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return nil, ErrTeamForbidden
	}

	var inviteeID *bson.ObjectID
	var inviteeEmail string

	if req.Username != "" {
		u, err := repository.FindUserByUsername(ctx, req.Username)
		if err != nil || u == nil {
			return nil, errors.New("user not found")
		}
		id := u.ID
		inviteeID = &id
		// Already a member?
		existing, _ := repository.FindTeamMember(ctx, team.ID, u.ID)
		if existing != nil && !existing.IsBanned {
			return nil, ErrAlreadyMember
		}
	} else if req.Email != "" {
		inviteeEmail = req.Email
	} else {
		return nil, errors.New("provide username or email")
	}

	// Duplicate pending invite check
	existing, _ := repository.FindInviteByTeamAndInvitee(ctx, team.ID, inviteeID, inviteeEmail)
	if existing != nil {
		return nil, errors.New("invite already pending")
	}

	token, err := repository.GenerateInviteToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	inv := &models.TeamInvite{
		TeamID:       team.ID,
		TeamName:     team.Name,
		TeamSlug:     team.Slug,
		InvitedBy:    callerID,
		InviterName:  callerUsername,
		InviteeID:    inviteeID,
		InviteeEmail: inviteeEmail,
		Token:        token,
		Role:         req.Role,
		Status:       models.InviteStatusPending,
		ExpiresAt:    now.Add(7 * 24 * time.Hour), // 7-day expiry
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repository.InsertTeamInvite(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func AcceptInvite(ctx context.Context, callerID bson.ObjectID, callerUsername, callerAvatar, token string) (*models.Team, error) {
	_ = repository.ExpireOldInvites(ctx) // lazy expiry

	inv, err := repository.FindInviteByToken(ctx, token)
	if err != nil || inv == nil {
		return nil, ErrInviteNotFound
	}
	if inv.Status != models.InviteStatusPending || inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrInviteExpired
	}

	// Ensure it's for this caller (if user-targeted)
	if inv.InviteeID != nil && *inv.InviteeID != callerID {
		return nil, ErrTeamForbidden
	}

	// Check not already a member
	existing, _ := repository.FindTeamMember(ctx, inv.TeamID, callerID)
	if existing != nil && !existing.IsBanned {
		_ = repository.UpdateInviteStatus(ctx, inv.ID, models.InviteStatusAccepted)
		return repository.FindTeamByID(ctx, inv.TeamID)
	}

	now := time.Now()
	member := &models.TeamMember{
		TeamID:    inv.TeamID,
		UserID:    callerID,
		Username:  callerUsername,
		Avatar:    callerAvatar,
		Role:      inv.Role,
		IsBanned:  false,
		JoinedAt:  now,
		UpdatedAt: now,
	}
	if err := repository.InsertTeamMember(ctx, member); err != nil {
		return nil, err
	}
	_ = repository.IncrTeamStat(ctx, inv.TeamID, "memberCount", 1)
	_ = repository.UpdateInviteStatus(ctx, inv.ID, models.InviteStatusAccepted)

	return repository.FindTeamByID(ctx, inv.TeamID)
}

func DeclineInvite(ctx context.Context, callerID bson.ObjectID, token string) error {
	inv, err := repository.FindInviteByToken(ctx, token)
	if err != nil || inv == nil {
		return ErrInviteNotFound
	}
	if inv.InviteeID != nil && *inv.InviteeID != callerID {
		return ErrTeamForbidden
	}
	return repository.UpdateInviteStatus(ctx, inv.ID, models.InviteStatusDeclined)
}

func ListPendingInvites(ctx context.Context, callerID bson.ObjectID, slug string) ([]models.TeamInvite, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if m == nil || !isTeamAdmin(m.Role) {
		return nil, ErrTeamForbidden
	}
	return repository.FindPendingInvitesByTeam(ctx, team.ID)
}

func ListMyInvites(ctx context.Context, callerID bson.ObjectID, callerEmail string) ([]models.TeamInvite, error) {
	return repository.FindPendingInvitesByUser(ctx, callerID, callerEmail)
}

func RevokeInvite(ctx context.Context, callerID bson.ObjectID, slug, inviteID string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if m == nil || !isTeamAdmin(m.Role) {
		return ErrTeamForbidden
	}
	id, err := bson.ObjectIDFromHex(inviteID)
	if err != nil {
		return errors.New("invalid inviteId")
	}
	return repository.UpdateInviteStatus(ctx, id, "revoked")
}

// GetMyJoinRequest returns the caller's own pending join request for this team, or nil if none.
func GetMyJoinRequest(ctx context.Context, callerID bson.ObjectID, slug string) (*models.TeamJoinRequest, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	req, _ := repository.FindJoinRequest(ctx, team.ID, callerID)
	return req, nil
}

// GetMyMembership returns the caller's own TeamMember record, or nil if not a member.
func GetMyMembership(ctx context.Context, callerID bson.ObjectID, slug string) (*models.TeamMember, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	m, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	return m, nil // nil = not a member, no error
}

// Join Requests
func RequestToJoin(ctx context.Context, callerID bson.ObjectID, callerUsername, callerAvatar, slug, message string) (*models.TeamJoinRequest, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	if team.JoinPolicy == models.TeamJoinInviteOnly {
		return nil, ErrTeamInviteOnly
	}

	existing, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if existing != nil && !existing.IsBanned {
		return nil, ErrAlreadyMember
	}

	// If open policy, just add directly
	if team.JoinPolicy == models.TeamJoinOpen {
		now := time.Now()
		m := &models.TeamMember{
			TeamID:    team.ID,
			UserID:    callerID,
			Username:  callerUsername,
			Avatar:    callerAvatar,
			Role:      models.TeamRoleDeveloper,
			JoinedAt:  now,
			UpdatedAt: now,
		}
		_ = repository.InsertTeamMember(ctx, m)
		_ = repository.IncrTeamStat(ctx, team.ID, "memberCount", 1)
		return nil, nil // nil = already joined directly
	}

	// request policy: check for existing pending request
	existingReq, _ := repository.FindJoinRequest(ctx, team.ID, callerID)
	if existingReq != nil {
		return nil, ErrJoinRequestExists
	}

	jr := &models.TeamJoinRequest{
		TeamID:    team.ID,
		UserID:    callerID,
		Username:  callerUsername,
		Avatar:    callerAvatar,
		Message:   message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := repository.InsertJoinRequest(ctx, jr); err != nil {
		return nil, err
	}
	return jr, nil
}

func ReviewJoinRequest(ctx context.Context, callerID bson.ObjectID, slug, requestID, action string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return ErrTeamForbidden
	}

	rid, err := bson.ObjectIDFromHex(requestID)
	if err != nil {
		return errors.New("invalid requestId")
	}
	jr, err := repository.FindJoinRequestByID(ctx, rid)
	if err != nil || jr == nil {
		return errors.New("join request not found")
	}

	status := "rejected"
	if action == "approve" {
		status = "approved"
		// Fetch user details
		u, _ := repository.FindUserByIDRaw(ctx, jr.UserID.Hex())
		avatar := ""
		if u != nil {
			avatar = u.Avatar
		}
		now := time.Now()
		m := &models.TeamMember{
			TeamID:    team.ID,
			UserID:    jr.UserID,
			Username:  jr.Username,
			Avatar:    avatar,
			Role:      models.TeamRoleDeveloper,
			JoinedAt:  now,
			UpdatedAt: now,
		}
		_ = repository.InsertTeamMember(ctx, m)
		_ = repository.IncrTeamStat(ctx, team.ID, "memberCount", 1)
	}

	return repository.UpdateJoinRequestStatus(ctx, rid, status, callerID)
}

func ListJoinRequests(ctx context.Context, callerID bson.ObjectID, slug string) ([]models.TeamJoinRequest, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return nil, ErrTeamForbidden
	}
	return repository.FindPendingJoinRequests(ctx, team.ID)
}
