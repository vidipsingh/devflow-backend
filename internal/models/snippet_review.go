package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SnippetReview is a rating+comment left on a marketplace snippet.
type SnippetReview struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	SnippetID bson.ObjectID `bson:"snippetId"     json:"snippetId"`
	AuthorID  bson.ObjectID `bson:"authorId"      json:"authorId"`
	Rating    int           `bson:"rating"        json:"rating"` // 1–5
	Comment   string        `bson:"comment"       json:"comment"`
	CreatedAt time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"     json:"updatedAt"`

	// Populated on read
	AuthorUsername string `bson:"authorUsername,omitempty" json:"authorUsername,omitempty"`
}
