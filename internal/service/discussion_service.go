package service

import (
	"context"
	"errors"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrDiscussionNotFound = errors.New("discussion not found")
	ErrReplyNotFound      = errors.New("reply not found")
)

// CreateDiscussion creates a new thread in a team. Any member can post.
func CreateDiscussion(ctx context.Context, callerID bson.ObjectID, slug, title, body string) (*models.TeamDiscussion, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	member, err := repository.FindTeamMember(ctx, team.ID, callerID)
	if err != nil || member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	d := &models.TeamDiscussion{
		TeamID:       team.ID,
		AuthorID:     callerID,
		AuthorName:   member.Username,
		AuthorAvatar: member.Avatar,
		Title:        title,
		Body:         body,
	}
	if err := repository.InsertDiscussion(ctx, d); err != nil {
		return nil, err
	}
	go func() {
		bgCtx := context.Background()
		_ = repository.InsertTeamActivity(bgCtx, &models.TeamActivity{
			TeamID: team.ID, ActorID: callerID, ActorName: member.Username,
			Action: "discussion_created", TargetType: "discussion", TargetName: title,
		})
	}()
	return d, nil
}

// ListDiscussions returns paginated threads for a team (any member).
func ListDiscussions(ctx context.Context, callerID bson.ObjectID, slug string, limit, skip int64) ([]models.TeamDiscussion, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	return repository.FindDiscussionsByTeam(ctx, team.ID, limit, skip)
}

// GetDiscussion fetches a single discussion + its replies.
func GetDiscussion(ctx context.Context, callerID bson.ObjectID, slug, discussionID string) (*models.TeamDiscussion, []models.TeamDiscussionReply, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, nil, ErrNotMember
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return nil, nil, ErrDiscussionNotFound
	}
	d, err := repository.FindDiscussionByID(ctx, oid)
	if err != nil || d == nil {
		return nil, nil, ErrDiscussionNotFound
	}
	replies, err := repository.FindRepliesByDiscussion(ctx, oid, 200, 0)
	if err != nil {
		return nil, nil, err
	}
	return d, replies, nil
}

// UpdateDiscussion edits title/body — only the author or admin can edit.
func UpdateDiscussion(ctx context.Context,  callerID bson.ObjectID, slug, discussionID, title, body string) (*models.TeamDiscussion, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return nil, ErrDiscussionNotFound
	}
	d, err := repository.FindDiscussionByID(ctx, oid)
	if err != nil || d == nil {
		return nil, ErrDiscussionNotFound
	}
	if d.AuthorID != callerID && !isTeamAdmin(member.Role) {
		return nil, ErrTeamForbidden
	}
	update := bson.M{"title": title, "body": body}
	if err := repository.UpdateDiscussion(ctx, oid, update); err != nil {
		return nil, err
	}
	d.Title = title
	d.Body = body
	return d, nil
}

// PinDiscussion toggles pin status — admin only.
func PinDiscussion(ctx context.Context, callerID bson.ObjectID, slug, discussionID string, pin bool) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || !isTeamAdmin(member.Role) {
		return ErrTeamForbidden
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return ErrDiscussionNotFound
	}
	return repository.UpdateDiscussion(ctx, oid, bson.M{"pinned": pin})
}

// ResolveDiscussion marks thread resolved — author or admin.
func ResolveDiscussion(ctx context.Context, callerID bson.ObjectID, slug, discussionID string, resolved bool) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return ErrNotMember
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return ErrDiscussionNotFound
	}
	d, err := repository.FindDiscussionByID(ctx, oid)
	if err != nil || d == nil {
		return ErrDiscussionNotFound
	}
	if d.AuthorID != callerID && !isTeamAdmin(member.Role) {
		return ErrTeamForbidden
	}
	return repository.UpdateDiscussion(ctx, oid, bson.M{"resolved": resolved})
}

// DeleteDiscussion removes thread + all replies — author or admin.
func DeleteDiscussion(ctx context.Context, callerID bson.ObjectID, slug, discussionID string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return ErrNotMember
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return ErrDiscussionNotFound
	}
	d, err := repository.FindDiscussionByID(ctx, oid)
	if err != nil || d == nil {
		return ErrDiscussionNotFound
	}
	if d.AuthorID != callerID && !isTeamAdmin(member.Role) {
		return ErrTeamForbidden
	}
	_ = repository.DeleteRepliesByDiscussion(ctx, oid)
	return repository.DeleteDiscussion(ctx, oid)
}

// Replies

// AddReply posts a reply to a discussion. Any non-banned member can reply.
func AddReply(ctx context.Context, callerID bson.ObjectID, slug, discussionID, body string) (*models.TeamDiscussionReply, error) {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return nil, ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return nil, ErrNotMember
	}
	oid, err := bson.ObjectIDFromHex(discussionID)
	if err != nil {
		return nil, ErrDiscussionNotFound
	}
	d, err := repository.FindDiscussionByID(ctx, oid)
	if err != nil || d == nil {
		return nil, ErrDiscussionNotFound
	}
	r := &models.TeamDiscussionReply{
		DiscussionID: oid,
		TeamID:       team.ID,
		AuthorID:     callerID,
		AuthorName:   member.Username,
		AuthorAvatar: member.Avatar,
		Body:         body,
	}
	if err := repository.InsertDiscussionReply(ctx, r); err != nil {
		return nil, err
	}
	_ = repository.IncrDiscussionReplyCount(ctx, oid, 1)
	return r, nil
}

// DeleteReply removes a single reply — author or admin.
func DeleteReply(ctx context.Context, callerID bson.ObjectID, slug, discussionID, replyID string) error {
	team, err := repository.FindTeamBySlug(ctx, slug)
	if err != nil || team == nil {
		return ErrTeamNotFound
	}
	member, _ := repository.FindTeamMember(ctx, team.ID, callerID)
	if member == nil || member.IsBanned {
		return ErrNotMember
	}
	rid, err := bson.ObjectIDFromHex(replyID)
	if err != nil {
		return ErrReplyNotFound
	}
	reply, err := repository.FindReplyByID(ctx, rid)
	if err != nil || reply == nil {
		return ErrReplyNotFound
	}
	if reply.AuthorID != callerID && !isTeamAdmin(member.Role) {
		return ErrTeamForbidden
	}
	if err := repository.DeleteReply(ctx, rid); err != nil {
		return err
	}
	did, _ := bson.ObjectIDFromHex(discussionID)
	_ = repository.IncrDiscussionReplyCount(ctx, did, -1)
	return nil
}
