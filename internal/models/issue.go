package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type IssueLabel struct {
	Name  string `bson:"name"  json:"name"`
	Color string `bson:"color" json:"color"`
}

type IssueReactions struct {
	ThumbsUp   int `bson:"thumbsUp"   json:"thumbsUp"`
	ThumbsDown int `bson:"thumbsDown" json:"thumbsDown"`
	Laugh      int `bson:"laugh"      json:"laugh"`
	Hooray     int `bson:"hooray"     json:"hooray"`
	Confused   int `bson:"confused"   json:"confused"`
	Heart      int `bson:"heart"      json:"heart"`
	Rocket     int `bson:"rocket"     json:"rocket"`
	Eyes       int `bson:"eyes"       json:"eyes"`
}

type IssueComment struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID   bson.ObjectID `bson:"authorId"      json:"authorId"`
	AuthorName string        `bson:"authorName"    json:"authorName"`
	Body       string        `bson:"body"          json:"body"`
	IsEdited   bool          `bson:"isEdited"      json:"isEdited"`
	CreatedAt  time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

type Issue struct {
	ID           bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Number       int             `bson:"number"        json:"number"`
	RepoID       bson.ObjectID   `bson:"repoId"        json:"repoId"`
	RepoSlug     string          `bson:"repoSlug"      json:"repoSlug"`
	Title        string          `bson:"title"         json:"title"`
	Body         string          `bson:"body"          json:"body"`
	State        string          `bson:"state"         json:"state"`
	AuthorID     bson.ObjectID   `bson:"authorId"      json:"authorId"`
	AuthorName   string          `bson:"authorName"    json:"authorName"`
	Assignees    []bson.ObjectID `bson:"assignees"     json:"assignees"`
	Labels       []IssueLabel    `bson:"labels"        json:"labels"`
	Milestone    string          `bson:"milestone"     json:"milestone"`
	Comments     []IssueComment  `bson:"comments"      json:"comments"`
	CommentCount int             `bson:"commentCount"  json:"commentCount"`
	Reactions    IssueReactions  `bson:"reactions"     json:"reactions"`
	IsPinned     bool            `bson:"isPinned"      json:"isPinned"`
	IsLocked     bool            `bson:"isLocked"      json:"isLocked"`
	ClosedAt     *time.Time      `bson:"closedAt"      json:"closedAt"`
	ClosedBy     *bson.ObjectID  `bson:"closedBy"      json:"closedBy"`
	CreatedAt    time.Time       `bson:"createdAt"     json:"createdAt"`
	UpdatedAt    time.Time       `bson:"updatedAt"     json:"updatedAt"`
}

type CreateIssueRequest struct {
	Title     string       `json:"title"     binding:"required,min=1,max=256"`
	Body      string       `json:"body"`
	Labels    []IssueLabel `json:"labels"`
	Assignees []string     `json:"assignees"`
	Milestone string       `json:"milestone"`
}

type UpdateIssueRequest struct {
	Title     *string      `json:"title"     binding:"omitempty,min=1,max=256"`
	Body      *string      `json:"body"`
	State     *string      `json:"state"     binding:"omitempty,oneof=open closed"`
	Labels    []IssueLabel `json:"labels"`
	Assignees []string     `json:"assignees"`
	Milestone *string      `json:"milestone"`
	IsPinned  *bool        `json:"isPinned"`
	IsLocked  *bool        `json:"isLocked"`
}

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1"`
}

type UpdateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1"`
}

type ReactRequest struct {
	Reaction string `json:"reaction" binding:"required,oneof=thumbsUp thumbsDown laugh hooray confused heart rocket eyes"`
	Add      bool   `json:"add"`
}
