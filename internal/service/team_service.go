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

var (
	ErrRepoAlreadyInTeam = errors.New("repository is already in this team")
	ErrRepoNotOwned      = errors.New("you do not own this repository")
	ErrInsufficientRole  = errors.New("you do not have sufficient role")
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
	go func() {
		repository.NotifyTeamAdmins(context.Background(), team.ID,
			"team_join_request", callerID.Hex(), callerUsername,
			map[string]any{"teamSlug": team.Slug, "teamName": team.Name, "requestId": jr.ID.Hex()},
		)
	}()
	return jr, nil
}

func ReviewJoinRequest(ctx context.Context, callerID bson.ObjectID, slug, requestID, action string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	caller, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if caller == nil || !isTeamAdmin(caller.Role) {
		return ErrInsufficientRole
	}
	reviewerUsername := caller.Username
	// shadow the error return below — restore the if block
	if false {
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
		go func() {
			bgCtx := context.Background()
			// Notify the requester
			_ = repository.InsertNotification(bgCtx, &models.Notification{
				RecipientID: jr.UserID.Hex(),
				Type:        "team_join_approved",
				ActorID:     callerID.Hex(),
				ActorName:   reviewerUsername,
				Meta:        map[string]any{"teamSlug": team.Slug, "teamName": team.Name},
			})
			// Activity + Audit
			_ = repository.InsertTeamActivity(bgCtx, &models.TeamActivity{
				TeamID: team.ID, ActorID: callerID, ActorName: reviewerUsername,
				Action: models.TeamActionJoinRequestApproved, TargetType: "member", TargetName: jr.Username,
			})
			_ = repository.InsertTeamAuditLog(bgCtx, &models.TeamAuditLog{
				TeamID: team.ID, ActorID: callerID, ActorUsername: reviewerUsername,
				Action: models.TeamActionJoinRequestApproved, TargetUsername: jr.Username,
			})
		}()
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

// AddTeamRepo links a caller-owned repository to a team (admin+ required).
func AddTeamRepo(ctx context.Context, callerID bson.ObjectID, slug, repoName string) (*models.TeamRepo, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}

	// Caller must be admin+
	member, err := repository.FindTeamMember(ctx, team.ID, callerID)
	if err != nil || member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	if !models.HasPermission(member.Role, models.PermManageRepos) {
		return nil, ErrInsufficientRole
	}

	// Look up the repo — caller must own it
	repo, err := repository.FindRepoByOwnerAndName(ctx, callerID, repoName)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}

	// Duplicate Check
	existing, err := repository.FindTeamRepo(ctx, team.ID, repo.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRepoAlreadyInTeam
	}

	tr := &models.TeamRepo{
		TeamID:       team.ID,
		RepoID:       repo.ID,
		RepoName:     repo.Name,
		RepoSlug:     repo.Slug,
		RepoFullName: repo.FullName,
		Visibility:   repo.Visibility,
		AddedBy:      callerID,
	}
	if err := repository.InsertTeamRepo(ctx, tr); err != nil {
		return nil, err
	}

	// Increment repoCount
	_ = repository.IncrTeamStat(ctx, team.ID, "repoCount", 1)

	// Activity + Audit
	go func() {
		bgCtx := context.Background()
		_ = repository.InsertTeamActivity(bgCtx, &models.TeamActivity{
			TeamID: team.ID, ActorID: callerID, ActorName: member.Username,
			Action: models.TeamActionRepoAdded, TargetType: "repo", TargetName: repo.Name,
		})
		_ = repository.InsertTeamAuditLog(bgCtx, &models.TeamAuditLog{
			TeamID: team.ID, ActorID: callerID, ActorUsername: member.Username,
			Action: models.TeamActionRepoAdded, Meta: map[string]any{"repoName": repo.Name},
		})
	}()
	return tr, nil
}

// ListTeamRepos returns repos linked to a team.
// Members see all; guests/viewers see only public repos.
func ListTeamRepos(ctx context.Context, callerID bson.ObjectID, slug string, limit, skip int64) ([]models.TeamRepo, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}

	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	visFilter := ""
	if member == nil || member.IsBanned || member.Role == models.TeamRoleGuest {
		visFilter = "public"
	}

	repos, err := repository.FindTeamRepos(ctx, team.ID, visFilter, limit, skip)
	if err != nil {
		return nil, err
	}

	// Backfill RepoFullName for legacy documents that were inserted before the field existed.
	for i := range repos {
		if repos[i].RepoFullName == "" && !repos[i].RepoID.IsZero() {
			if repo, err2 := repository.FindRepoByID(ctx, repos[i].RepoID); err2 == nil && repo != nil {
				repos[i].RepoFullName = repo.FullName
			}
		}
	}

	return repos, nil
}

// RemoveTeamRepo unlinks a repo from a team (admin+ required).
func RemoveTeamRepo(ctx context.Context, callerID bson.ObjectID, slug, repoSlug string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	member, err := repository.FindTeamMember(ctx, team.ID, callerID)
	if err != nil || member == nil || member.IsBanned {
		return ErrTeamNotFound
	}
	if !models.HasPermission(member.Role, models.PermManageRepos) {
		return ErrInsufficientRole
	}

	if err := repository.DeleteTeamRepo(ctx, team.ID, repoSlug); err != nil {
		return err
	}
	_ = repository.IncrTeamStat(ctx, team.ID, "repoCount", -1)

	go func() {
		bgCtx := context.Background()
		_ = repository.InsertTeamActivity(bgCtx, &models.TeamActivity{
			TeamID: team.ID, ActorID: callerID, ActorName: member.Username,
			Action: models.TeamActionRepoRemoved, TargetType: "repo", TargetName: repoSlug,
		})
		_ = repository.InsertTeamAuditLog(bgCtx, &models.TeamAuditLog{
			TeamID: team.ID, ActorID: callerID, ActorUsername: member.Username,
			Action: models.TeamActionRepoRemoved, Meta: map[string]any{"repoSlug": repoSlug},
		})
	}()
	return nil
}

// Activity Feed
func GetTeamActivity(ctx context.Context, callerID bson.ObjectID, slug string, limit, skip int64) ([]models.TeamActivity, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}

	// Must be a member to see activity
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	return repository.FindTeamActivity(ctx, team.ID, limit, skip)
}

// Audit Logs
func GetTeamAuditLog(ctx context.Context, callerID bson.ObjectID, slug string, limit, skip int64) ([]models.TeamAuditLog, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	if !models.HasPermission(member.Role, models.PermManageSettings) {
		return nil, ErrInsufficientRole
	}
	return repository.FindTeamAuditLog(ctx, team.ID, limit, skip)
}

// Sub-team creation via parent slug
func CreateSubTeam(ctx context.Context, callerID bson.ObjectID, callerUsername, parentSlug string, req models.CreateSubTeamRequest) (*models.Team, error) {
	parent, err := repository.FindTeamBySlug(ctx, parentSlug)
	if err != nil || parent == nil {
		return nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, parent.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	if !models.HasPermission(member.Role, models.PermManageSettings) {
		return nil, ErrInsufficientRole
	}

	// Build slug from parent slug + child name
	childSlug := parentSlug + "-" + slugify(req.Name)

	// Ensure slug uniqueness
	existing, _ := repository.FindTeamBySlug(ctx, childSlug)
	if existing != nil {
		return nil, errors.New("a team with that slug already exists")
	}

	if req.Visibility == "" {
		req.Visibility = parent.Visibility
	}
	if req.JoinPolicy == "" {
		req.JoinPolicy = models.TeamJoinInviteOnly
	}

	now := time.Now()
	sub := &models.Team{
		Name:        req.Name,
		Slug:        childSlug,
		Description: req.Description,
		Type:        parent.Type,
		Visibility:  req.Visibility,
		JoinPolicy:  req.JoinPolicy,
		Tags:        req.Tags,
		ParentID:    &parent.ID,
		CreatedBy:   callerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repository.CreateTeam(ctx, sub); err != nil {
		return nil, err
	}

	// Add creator as owner
	_ = repository.InsertTeamMember(ctx, &models.TeamMember{
		TeamID:    sub.ID,
		UserID:    callerID,
		Username:  callerUsername,
		Role:      models.TeamRoleOwner,
		JoinedAt:  now,
		UpdatedAt: now,
	})
	_ = repository.IncrTeamStat(ctx, sub.ID, "memberCount", 1)

	go func() {
		bgCtx := context.Background()
		_ = repository.InsertTeamActivity(bgCtx, &models.TeamActivity{
			TeamID: parent.ID, ActorID: callerID, ActorName: callerUsername,
			Action: models.TeamActionSubTeamCreated, TargetType: "team", TargetName: sub.Name,
		})
		_ = repository.InsertTeamAuditLog(bgCtx, &models.TeamAuditLog{
			TeamID: parent.ID, ActorID: callerID, ActorUsername: callerUsername,
			Action: models.TeamActionSubTeamCreated, Meta: map[string]any{"subTeamSlug": childSlug},
		})
	}()

	return sub, nil
}

// GetMyPermissions
func GetMyPermissions(ctx context.Context, callerID bson.ObjectID, slug string) ([]string, string, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, "", ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, "", nil
	}
	perms := models.RolePermissions[member.Role]
	if perms == nil {
		perms = []string{}
	}
	return perms, member.Role, err
}
