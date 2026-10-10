package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TeamDiscussion is a top-level thread in a team.
type TeamDiscussion struct {
	ID         bson.ObjectID `bson:"_id,omitempty"    json:"id"`
	TeamID     bson.ObjectID `bson:"teamId"           json:"teamId"`
	AuthorID   bson.ObjectID `bson:"authorId"         json:"authorId"`
	AuthorName string        `bson:"authorName"       json:"authorName"`
	AuthorAvatar string      `bson:"authorAvatar"     json:"authorAvatar"`
	Title      string        `bson:"title"            json:"title"`
	Body       string        `bson:"body"             json:"body"`
	Pinned     bool          `bson:"pinned"           json:"pinned"`
	Resolved   bool          `bson:"resolved"         json:"resolved"`
	ReplyCount int           `bson:"replyCount"       json:"replyCount"`
	CreatedAt  time.Time     `bson:"createdAt"        json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt"        json:"updatedAt"`
}

// TeamDiscussionReply is a reply to a TeamDiscussion thread.
type TeamDiscussionReply struct {
	ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	DiscussionID bson.ObjectID `bson:"discussionId"   json:"discussionId"`
	TeamID       bson.ObjectID `bson:"teamId"         json:"teamId"`
	AuthorID     bson.ObjectID `bson:"authorId"       json:"authorId"`
	AuthorName   string        `bson:"authorName"     json:"authorName"`
	AuthorAvatar string        `bson:"authorAvatar"   json:"authorAvatar"`
	Body         string        `bson:"body"           json:"body"`
	CreatedAt    time.Time     `bson:"createdAt"      json:"createdAt"`
	UpdatedAt    time.Time     `bson:"updatedAt"      json:"updatedAt"`
}
