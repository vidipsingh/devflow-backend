package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ─── Role constants ───────────────────────────────────────────────────────────

const (
	TeamRoleOwner      = "owner"
	TeamRoleAdmin      = "admin"
	TeamRoleMaintainer = "maintainer"
	TeamRoleDeveloper  = "developer"
	TeamRoleViewer     = "viewer"
	TeamRoleGuest      = "guest"
)

// ─── Join policy ─────────────────────────────────────────────────────────────

const (
	TeamJoinOpen        = "open"        // anyone can join
	TeamJoinInviteOnly  = "invite_only" // must be invited
	TeamJoinRequestOnly = "request"     // must request + admin approves
)

// ─── Visibility ──────────────────────────────────────────────────────────────

const (
	TeamVisibilityPublic  = "public"
	TeamVisibilityPrivate = "private"
	TeamVisibilitySecret  = "secret" // not listed anywhere
)

// ─── Invite status ───────────────────────────────────────────────────────────

const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"
	InviteStatusExpired  = "expired"
)

// ─── Team document ───────────────────────────────────────────────────────────

type Team struct {
	ID          bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name        string         `bson:"name"          json:"name"`
	Slug        string         `bson:"slug"          json:"slug"` // URL-safe unique
	Description string         `bson:"description"   json:"description"`
	Bio         string         `bson:"bio"           json:"bio"` // markdown README
	Avatar      string         `bson:"avatar"        json:"avatar"`
	Banner      string         `bson:"banner"        json:"banner"`
	Tags        []string       `bson:"tags"          json:"tags"`
	Type        string         `bson:"type"          json:"type"`                    // "organization"|"project"|"department"
	Visibility  string         `bson:"visibility"    json:"visibility"`              // public|private|secret
	JoinPolicy  string         `bson:"joinPolicy"    json:"joinPolicy"`              // open|invite_only|request
	ParentID    *bson.ObjectID `bson:"parentId,omitempty" json:"parentId,omitempty"` // nested sub-team

	// Denormalised counts for fast reads
	MemberCount int `bson:"memberCount" json:"memberCount"`
	RepoCount   int `bson:"repoCount"   json:"repoCount"`

	CreatedBy bson.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// ─── Team member ─────────────────────────────────────────────────────────────

type TeamMember struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	TeamID    bson.ObjectID `bson:"teamId"        json:"teamId"`
	UserID    bson.ObjectID `bson:"userId"        json:"userId"`
	Username  string        `bson:"username"      json:"username"`
	Avatar    string        `bson:"avatar"        json:"avatar"`
	Role      string        `bson:"role"          json:"role"` // owner|admin|maintainer|developer|viewer|guest
	IsBanned  bool          `bson:"isBanned"      json:"isBanned"`
	JoinedAt  time.Time     `bson:"joinedAt"      json:"joinedAt"`
	UpdatedAt time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

// ─── Team invite ─────────────────────────────────────────────────────────────

type TeamInvite struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	TeamID      bson.ObjectID `bson:"teamId"        json:"teamId"`
	TeamName    string        `bson:"teamName"      json:"teamName"`
	TeamSlug    string        `bson:"teamSlug"      json:"teamSlug"`
	InvitedBy   bson.ObjectID `bson:"invitedBy"     json:"invitedBy"`
	InviterName string        `bson:"inviterName"   json:"inviterName"`
	// Invite is either to a specific user OR via email
	InviteeID    *bson.ObjectID `bson:"inviteeId,omitempty"   json:"inviteeId,omitempty"`
	InviteeEmail string         `bson:"inviteeEmail,omitempty" json:"inviteeEmail,omitempty"`
	Token        string         `bson:"token"         json:"token,omitempty"` // returned only to the invitee
	Role         string         `bson:"role"          json:"role"`            // role to assign on accept
	Status       string         `bson:"status"        json:"status"`          // pending|accepted|declined|expired
	ExpiresAt    time.Time      `bson:"expiresAt"     json:"expiresAt"`
	CreatedAt    time.Time      `bson:"createdAt"     json:"createdAt"`
	UpdatedAt    time.Time      `bson:"updatedAt"     json:"updatedAt"`
}

// ─── Join request ────────────────────────────────────────────────────────────

type TeamJoinRequest struct {
	ID         bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	TeamID     bson.ObjectID  `bson:"teamId"        json:"teamId"`
	UserID     bson.ObjectID  `bson:"userId"        json:"userId"`
	Username   string         `bson:"username"      json:"username"`
	Avatar     string         `bson:"avatar"        json:"avatar"`
	Message    string         `bson:"message"       json:"message"` // optional note
	Status     string         `bson:"status"        json:"status"`  // pending|approved|rejected
	ReviewedBy *bson.ObjectID `bson:"reviewedBy,omitempty" json:"reviewedBy,omitempty"`
	ReviewedAt *time.Time     `bson:"reviewedAt,omitempty" json:"reviewedAt,omitempty"`
	CreatedAt  time.Time      `bson:"createdAt"     json:"createdAt"`
}

// ─── Request / Response DTOs ─────────────────────────────────────────────────

type CreateTeamRequest struct {
	Name        string   `json:"name"        binding:"required,min=2,max=64"`
	Description string   `json:"description"`
	Bio         string   `json:"bio"`
	Type        string   `json:"type"`       // default "organization"
	Visibility  string   `json:"visibility"` // default "public"
	JoinPolicy  string   `json:"joinPolicy"` // default "invite_only"
	Tags        []string `json:"tags"`
	ParentID    string   `json:"parentId"` // optional
}

type UpdateTeamRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Bio         *string  `json:"bio"`
	Avatar      *string  `json:"avatar"`
	Banner      *string  `json:"banner"`
	Visibility  *string  `json:"visibility"`
	JoinPolicy  *string  `json:"joinPolicy"`
	Tags        []string `json:"tags"`
}

type InviteMemberRequest struct {
	Username string `json:"username"`                     // invite by username
	Email    string `json:"email"`                        // OR by email
	Role     string `json:"role"      binding:"required"` // role to assign
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type ReviewJoinRequest struct {
	Action string `json:"action" binding:"required"` // "approve" | "reject"
}

type RequestToJoinRequest struct {
	Message string `json:"message"` // optional
}

// TeamRepo links a repository to a team.
type TeamRepo struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	TeamID       bson.ObjectID `bson:"teamId"        json:"teamId"`
	RepoID       bson.ObjectID `bson:"repoId"        json:"repoId"`
	RepoName     string        `bson:"repoName"      json:"repoName"`
	RepoSlug     string        `bson:"repoSlug"      json:"repoSlug"`
	RepoFullName string        `bson:"repoFullName"  json:"repoFullName"` // e.g. "alice/my-repo"
	// Visibility is cached from the repo at add-time for fast list filtering
	Visibility string        `bson:"visibility" json:"visibility"` // "public"|"private"
	AddedBy    bson.ObjectID `bson:"addedBy"    json:"addedBy"`
	AddedAt    time.Time     `bson:"addedAt"    json:"addedAt"`
}

// TeamActivity is a single event in a team's activity feed.
type TeamActivity struct {
	ID          bson.ObjectID  `bson:"_id,omitempty"     json:"id"`
	TeamID      bson.ObjectID  `bson:"teamId"            json:"teamId"`
	ActorID     bson.ObjectID  `bson:"actorId"           json:"actorId"`
	ActorName   string         `bson:"actorName"         json:"actorName"`
	ActorAvatar string         `bson:"actorAvatar"       json:"actorAvatar"`
	Action      string         `bson:"action"            json:"action"`     // see TeamAction* constants
	TargetType  string         `bson:"targetType"        json:"targetType"` // "member"|"repo"|"team"|"invite"|"join_request"
	TargetName  string         `bson:"targetName"        json:"targetName"`
	Meta        map[string]any `bson:"meta,omitempty"    json:"meta,omitempty"`
	CreatedAt   time.Time      `bson:"createdAt"         json:"createdAt"`
}

// TeamAuditLog records every privileged action for compliance/trust.
type TeamAuditLog struct {
	ID             bson.ObjectID  `bson:"_id,omitempty"       json:"id"`
	TeamID         bson.ObjectID  `bson:"teamId"              json:"teamId"`
	ActorID        bson.ObjectID  `bson:"actorId"             json:"actorId"`
	ActorUsername  string         `bson:"actorUsername"       json:"actorUsername"`
	Action         string         `bson:"action"              json:"action"`
	TargetUsername string         `bson:"targetUsername,omitempty" json:"targetUsername,omitempty"`
	TargetRole     string         `bson:"targetRole,omitempty"     json:"targetRole,omitempty"`
	Meta           map[string]any `bson:"meta,omitempty"      json:"meta,omitempty"`
	CreatedAt      time.Time      `bson:"createdAt"           json:"createdAt"`
}

// Action constants
const (
	TeamActionMemberJoined         = "member_joined"
	TeamActionMemberLeft           = "member_left"
	TeamActionMemberRemoved        = "member_removed"
	TeamActionMemberBanned         = "member_banned"
	TeamActionMemberUnbanned       = "member_unbanned"
	TeamActionRoleChanged          = "role_changed"
	TeamActionRepoAdded            = "repo_added"
	TeamActionRepoRemoved          = "repo_removed"
	TeamActionJoinRequestApproved  = "join_request_approved"
	TeamActionJoinRequestRejected  = "join_request_rejected"
	TeamActionInviteSent           = "invite_sent"
	TeamActionInviteRevoked        = "invite_revoked"
	TeamActionOwnershipTransferred = "ownership_transferred"
	TeamActionSettingsUpdated      = "settings_updated"
	TeamActionSubTeamCreated       = "sub_team_created"
)

// Permission constants — computed from role
const (
	PermRead           = "read"
	PermWrite          = "write"
	PermMergePR        = "merge_pr"
	PermManageIssues   = "manage_issues"
	PermManageMembers  = "manage_members"
	PermManageRepos    = "manage_repos"
	PermManageSettings = "manage_settings"
	PermDeleteTeam     = "delete_team"
)

// RolePermissions maps each role to its permission set.
var RolePermissions = map[string][]string{
	TeamRoleOwner:      {PermRead, PermWrite, PermMergePR, PermManageIssues, PermManageMembers, PermManageRepos, PermManageSettings, PermDeleteTeam},
	TeamRoleAdmin:      {PermRead, PermWrite, PermMergePR, PermManageIssues, PermManageMembers, PermManageRepos, PermManageSettings},
	TeamRoleMaintainer: {PermRead, PermWrite, PermMergePR, PermManageIssues},
	TeamRoleDeveloper:  {PermRead, PermWrite},
	TeamRoleViewer:     {PermRead},
	TeamRoleGuest:      {PermRead},
}

// HasPermission returns true if role grants the given permission.
func HasPermission(role, perm string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// DTOs
type AddTeamRepoRequest struct {
	RepoName string `json:"repoName" binding:"required"`
}

type CreateSubTeamRequest struct {
	Name        string   `json:"name"       binding:"required,min=2,max=64"`
	Description string   `json:"description"`
	Visibility  string   `json:"visibility"`
	JoinPolicy  string   `json:"joinPolicy"`
	Tags        []string `json:"tags"`
}
