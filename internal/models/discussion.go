package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// DiscussionReactions holds emoji reaction counts for discussions and replies.
type DiscussionReactions struct {
	ThumbsUp int `bson:"thumbsUp" json:"thumbsUp"`
	Heart    int `bson:"heart"    json:"heart"`
	Hooray   int `bson:"hooray"   json:"hooray"`
	Rocket   int `bson:"rocket"   json:"rocket"`
	Eyes     int `bson:"eyes"     json:"eyes"`
	Laugh    int `bson:"laugh"    json:"laugh"`
}

// DiscussionReactRequest is the request body for POST …/react
type DiscussionReactRequest struct {
	Reaction string `json:"reaction" binding:"required,oneof=thumbsUp heart hooray rocket eyes laugh"`
	Add      bool   `json:"add"`
}

// TeamDiscussion is a top-level thread in a team.
type TeamDiscussion struct {
	ID           bson.ObjectID       `bson:"_id,omitempty"    json:"id"`
	TeamID       bson.ObjectID       `bson:"teamId"           json:"teamId"`
	AuthorID     bson.ObjectID       `bson:"authorId"         json:"authorId"`
	AuthorName   string              `bson:"authorName"       json:"authorName"`
	AuthorAvatar string              `bson:"authorAvatar"     json:"authorAvatar"`
	Title        string              `bson:"title"            json:"title"`
	Body         string              `bson:"body"             json:"body"`
	Pinned       bool                `bson:"pinned"           json:"pinned"`
	Resolved     bool                `bson:"resolved"         json:"resolved"`
	ReplyCount   int                 `bson:"replyCount"       json:"replyCount"`
	Reactions    DiscussionReactions `bson:"reactions"        json:"reactions"`
	CreatedAt    time.Time           `bson:"createdAt"        json:"createdAt"`
	UpdatedAt    time.Time           `bson:"updatedAt"        json:"updatedAt"`
}

// TeamDiscussionReply is a reply to a TeamDiscussion thread.
type TeamDiscussionReply struct {
	ID           bson.ObjectID       `bson:"_id,omitempty"  json:"id"`
	DiscussionID bson.ObjectID       `bson:"discussionId"   json:"discussionId"`
	TeamID       bson.ObjectID       `bson:"teamId"         json:"teamId"`
	AuthorID     bson.ObjectID       `bson:"authorId"       json:"authorId"`
	AuthorName   string              `bson:"authorName"     json:"authorName"`
	AuthorAvatar string              `bson:"authorAvatar"   json:"authorAvatar"`
	Body         string              `bson:"body"           json:"body"`
	Reactions    DiscussionReactions `bson:"reactions"      json:"reactions"`
	CreatedAt    time.Time           `bson:"createdAt"      json:"createdAt"`
	UpdatedAt    time.Time           `bson:"updatedAt"      json:"updatedAt"`
}
