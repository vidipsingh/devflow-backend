package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PRComment struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID   bson.ObjectID `bson:"authorId"      json:"authorId"`
	AuthorName string        `bson:"authorName"    json:"authorName"`
	Body       string        `bson:"body"          json:"body"`
	FilePath   string        `bson:"filePath"      json:"filePath"`
	LineNumber int           `bson:"lineNumber"    json:"lineNumber"`
	IsEdited   bool          `bson:"isEdited"      json:"isEdited"`
	CreatedAt  time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

type PullRequest struct {
	ID           bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Number       int             `bson:"number"        json:"number"`
	RepoID       bson.ObjectID   `bson:"repoId"        json:"repoId"`
	RepoSlug     string          `bson:"repoSlug"      json:"repoSlug"`
	Title        string          `bson:"title"         json:"title"`
	Body         string          `bson:"body"          json:"body"`
	State        string          `bson:"state"         json:"state"` // "open"|"closed"|"merged"
	HeadBranch   string          `bson:"headBranch"    json:"headBranch"`
	BaseBranch   string          `bson:"baseBranch"    json:"baseBranch"`
	AuthorID     bson.ObjectID   `bson:"authorId"      json:"authorId"`
	AuthorName   string          `bson:"authorName"    json:"authorName"`
	ReviewerIDs  []bson.ObjectID `bson:"reviewerIds"   json:"reviewerIds"`
	Labels       []IssueLabel    `bson:"labels"        json:"labels"`
	Comments     []PRComment     `bson:"comments"      json:"comments"`
	CommentCount int             `bson:"commentCount"  json:"commentCount"`
	Additions    int             `bson:"additions"     json:"additions"`
	Deletions    int             `bson:"deletions"     json:"deletions"`
	ChangedFiles []string        `bson:"changedFiles"  json:"changedFiles"`
	IsDraft      bool            `bson:"isDraft"       json:"isDraft"`
	IsMergeable  bool            `bson:"isMergeable"   json:"isMergeable"`
	MergedAt     *time.Time      `bson:"mergedAt"      json:"mergedAt"`
	MergedBy     *bson.ObjectID  `bson:"mergedBy"      json:"mergedBy"`
	ClosedAt     *time.Time      `bson:"closedAt"      json:"closedAt"`
	CreatedAt    time.Time       `bson:"createdAt"     json:"createdAt"`
	UpdatedAt    time.Time       `bson:"updatedAt"     json:"updatedAt"`
}

type CreatePRRequest struct {
	Title      string       `json:"title"      binding:"required,min=1,max=256"`
	Body       string       `json:"body"`
	HeadBranch string       `json:"headBranch" binding:"required"`
	BaseBranch string       `json:"baseBranch" binding:"required"`
	IsDraft    bool         `json:"isDraft"`
	Labels     []IssueLabel `json:"labels"`
}

type UpdatePRRequest struct {
	Title   *string      `json:"title"   binding:"omitempty,min=1,max=256"`
	Body    *string      `json:"body"`
	State   *string      `json:"state"   binding:"omitempty,oneof=open closed"`
	IsDraft *bool        `json:"isDraft"`
	Labels  []IssueLabel `json:"labels"`
}

type MergePRRequest struct {
	Method string `json:"mergeMethod" binding:"omitempty,oneof=merge squash rebase"`
}

type CreatePRCommentRequest struct {
	Body       string `json:"body"       binding:"required,min=1"`
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
}

type UpdatePRCommentRequest struct {
	Body string `json:"body" binding:"required,min=1"`
}
