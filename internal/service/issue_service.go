package service

import (
	"context"
	"errors"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrIssueNotFound  = errors.New("issue not found")
	ErrIssueForbidden = errors.New("not authorized to modify this issue")
)

// ListIssues returns paginated issues for a repo (looked up by slug)
func ListIssues(ctx context.Context, repoSlug, state string, page, limit int64) ([]models.Issue, int64, error) {
	repo, err := repository.FindRepoBySlug(ctx, repoSlug)
	if err != nil {
		return nil, 0, err
	}
	if repo == nil {
		return nil, 0, ErrRepoNotFound
	}
	return repository.FindIssueByRepo(ctx, repo.ID, state, page, limit)
}

// GetIssue returns a single issue by its sequential number within a repo
func GetIssue(ctx context.Context, repoSlug string, number int) (*models.Issue, error) {
	repo, err := repository.FindRepoBySlug(ctx, repoSlug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	issue, err := repository.FindIssueByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrIssueNotFound
	}
	return issue, nil
}

// CreateIssue creates a new issue in the repo
func CreateIssue(ctx context.Context, callerID bson.ObjectID, callerUsername, repoSlug string, req models.CreateIssueRequest) (*models.Issue, error) {
	repo, err := repository.FindRepoBySlug(ctx, repoSlug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	number, err := repository.NextIssueNumber(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	var assignees []bson.ObjectID
	for _, hex := range req.Assignees {
		if id, err := bson.ObjectIDFromHex(hex); err == nil {
			assignees = append(assignees, id)
		}
	}
	if assignees == nil {
		assignees = []bson.ObjectID{}
	}

	labels := req.Labels
	if labels == nil {
		labels = []models.IssueLabel{}
	}

	now := time.Now()
	issue := &models.Issue{
		Number:       number,
		RepoID:       repo.ID,
		RepoSlug:     repoSlug,
		Title:        req.Title,
		Body:         req.Body,
		State:        "open",
		AuthorID:     callerID,
		AuthorName:   callerUsername,
		Assignees:    assignees,
		Labels:       labels,
		Milestone:    req.Milestone,
		Comments:     []models.IssueComment{},
		CommentCount: 0,
		Reactions:    models.IssueReactions{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repository.CreateIssue(ctx, issue); err != nil {
		return nil, err
	}

	// Increment repo open issues counter
	_ = repository.UpdateRepoRaw(ctx, repo.ID, bson.M{
		"$inc": bson.M{"stats.openIssues": 1},
		"$set": bson.M{"updatedAt": now},
	})

	return issue, nil
}

// UpdateIssue edits any mutable field — only author or repo owner may do so
func UpdateIssue(ctx context.Context, callerID bson.ObjectID, repoSlug string, number int, req models.UpdateIssueRequest) (*models.Issue, error) {
	issue, err := GetIssue(ctx, repoSlug, number)
	if err != nil {
		return nil, err
	}

	repo, err := repository.FindRepoBySlug(ctx, repoSlug)
	if err != nil {
		return nil, err
	}

	if issue.AuthorID != callerID && (repo == nil || repo.OwnerID != callerID) {
		return nil, ErrIssueForbidden
	}

	set := bson.M{"updatedAt": time.Now()}

	// Track state transition to sync the repo openIssues counter
	var stateDelta int
	stateChanged := false
	if req.State != nil && *req.State != issue.State {
		set["state"] = *req.State
		stateChanged = true
		if *req.State == "closed" {
			now := time.Now()
			set["closedAt"] = now
			set["closedBy"] = callerID
			stateDelta = -1
		} else {
			set["closedAt"] = nil
			set["closedBy"] = nil
			stateDelta = 1
		}
	}
	if req.Title != nil {
		set["title"] = *req.Title
	}
	if req.Body != nil {
		set["body"] = *req.Body
	}
	if req.Labels != nil {
		set["labels"] = req.Labels
	}
	if req.Assignees != nil {
		var ids []bson.ObjectID
		for _, hex := range req.Assignees {
			if id, err := bson.ObjectIDFromHex(hex); err == nil {
				ids = append(ids, id)
			}
		}
		set["assignes"] = ids
	}
	if req.Milestone != nil {
		set["milestone"] = req.Milestone
	}
	if req.IsPinned != nil {
		set["isPinned"] = *req.IsPinned
	}
	if req.IsLocked != nil {
		set["isLocked"] = *req.IsLocked
	}

	if err := repository.UpdateIssueRaw(ctx, issue.ID, bson.M{"$set": set}); err != nil {
		return nil, err
	}

	if stateChanged && repo != nil {
		_ = repository.UpdateRepoRaw(ctx, repo.ID, bson.M{
			"$inc": bson.M{"stats.OpenIssues": stateDelta},
		})
	}

	return repository.FindIssueByNumber(ctx, repo.ID, number)
}

// DeleteIssue hard-deletes — only author or repo owner can delete it
func DeleteIssue(ctx context.Context, callerID bson.ObjectID, repoSlug string, number int) error {
	issue, err := GetIssue(ctx, repoSlug, number)
	if err != nil {
		return err
	}
	repo, err := repository.FindRepoBySlug(ctx, repoSlug)
	if err != nil {
		return err
	}
	if issue.AuthorID != callerID && (repo == nil || repo.OwnerID != callerID) {
		return ErrIssueForbidden
	}
	if err := repository.DeleteIssue(ctx, issue.ID); err != nil {
		return err
	}
	if issue.State == "open" && repo != nil {
		_ = repository.UpdateRepoRaw(ctx, repo.ID, bson.M{
			"$inc": bson.M{"stats.openIssues": -1},
		})
	}
	return nil
}

// AddComment appends a new comment
func AddComment(ctx context.Context, callerID bson.ObjectID, callerUsername, repoSlug string, number int, req models.CreateCommentRequest) (*models.Issue, error) {
	issue, err := GetIssue(ctx , repoSlug, number)
	if err != nil {
		return nil, err
	}
	if issue.IsLocked {
		return nil, errors.New("issue is locked - comments are disabled")
	}

	now := time.Now()
	comment := models.IssueComment{
		ID:         bson.NewObjectID(),
		AuthorID:   callerID,
		AuthorName: callerUsername,
		Body:       req.Body,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	update := bson.M{
		"$push": bson.M{"comments": comment},
		"$inc":  bson.M{"commentCount": 1},
		"$set":  bson.M{"updatedAt": now},
	}
	if err := repository.UpdateIssueRaw(ctx, issue.ID, update); err != nil {
		return nil, err
	}
	return repository.FindIssueByID(ctx, issue.ID)
}

// UpdateComment edits the body of a specific embedded comment
func UpdateComment(ctx context.Context, callerID bson.ObjectID, repoSlug string, number int, commentIDHex string, req models.UpdateCommentRequest) (*models.Issue, error) {
	issue, err := GetIssue(ctx, repoSlug, number)
	if err != nil {
		return nil, err
	}

	commentID, err := bson.ObjectIDFromHex(commentIDHex)
	if err != nil {
		return nil, errors.New("invalid comment id")
	}

	found := false
	for _, c := range issue.Comments {
		if c.ID == commentID {
			if c.AuthorID != callerID {
				return nil, ErrIssueForbidden
			}
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("comment not found")
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"comments.$[elem].body":      req.Body,
			"comments.$[elem].updatedAt": now,
			"comments.$[elem].isEdited":  true,
			"updatedAt":                  now,
		},
	}
	arrayFilters := []interface{}{bson.M{"elem._id": commentID}}

	if err := repository.UpdateIssueRawWithFilter(ctx, issue.ID, update, arrayFilters); err != nil {
		return nil, err
	}
	return repository.FindIssueByID(ctx, issue.ID)
}

// DeleteComment removes a comment from the embedded array
func DeleteComment(ctx context.Context, callerID bson.ObjectID, repoSlug string, number int, commentIDHex string) (*models.Issue, error) {
	issue, err := GetIssue(ctx, repoSlug, number)
	if err != nil {
		return nil, err
	}

	commentID, err := bson.ObjectIDFromHex(commentIDHex)
	if err != nil {
		return nil, errors.New("invlid comment id")
	}

	found := false
	for _, c := range issue.Comments {
		if c.ID == commentID {
			if c.AuthorID != callerID {
				return nil, ErrIssueForbidden
			}
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("comment not found")
	}

	now := time.Now()
	update := bson.M{
		"$pull": bson.M{"comments": bson.M{"_id": commentID}},
		"$inc":  bson.M{"commentCount": -1},
		"$set":  bson.M{"updatedAt": now},
	}
	if err := repository.UpdateIssueRaw(ctx, issue.ID, update); err != nil {
		return nil, err
	}
	return repository.FindIssueByID(ctx, issue.ID)
}

// ReactToIssue increments or decrements a named reaction counter
func ReactToIssue(ctx context.Context, repoSlug string, number int, req models.ReactRequest) (*models.Issue, error) {
	issue, err := GetIssue(ctx, repoSlug, number)
	if err != nil {
		return nil, err
	}

	delta := 1
	if !req.Add {
		delta = -1
	}

	update := bson.M{
		"$inc": bson.M{"reactions." + req.Reaction: delta},
		"$set": bson.M{"updatedAt": time.Now()},
	}
	if err := repository.UpdateIssueRaw(ctx, issue.ID, update); err != nil {
		return nil, err
	}
	return repository.FindIssueByID(ctx, issue.ID)
}
