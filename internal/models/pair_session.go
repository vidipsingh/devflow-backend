package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type SessionStatus string

const (
	SessionStatusWaiting SessionStatus = "waiting"
	SessionStatusActive  SessionStatus = "active"
	SessionStatusEnded   SessionStatus = "ended"
)

type ParticipantInfo struct {
	UserID   string `bson:"userId"   json:"userId"`
	Username string `bson:"username" json:"username"`
	Color    string `bson:"color"    json:"color"` // for cursor highlighting
}

// OTOperation represents a single Operational Transform edit
type OTOperation struct {
	ClientSeq  int    `bson:"clientSeq"  json:"clientSeq"`
	ServerSeq  int    `bson:"serverSeq"  json:"serverSeq"`
	Index      int    `bson:"index"      json:"index"`
	Insert     string `bson:"insert"     json:"insert"`
	Delete     int    `bson:"delete"     json:"delete"` // number of chars to delete at Index
	AuthorID   string `bson:"authorId"   json:"authorId"`
	AuthorName string `bson:"authorName" json:"authorName"`
}

type PairSession struct {
	ID           bson.ObjectID     `bson:"_id,omitempty"   json:"id"`
	OwnerID      string            `bson:"ownerId"         json:"ownerId"`
	RepoID       string            `bson:"repoId"          json:"repoId"`
	FileID       string            `bson:"fileId"          json:"fileId"`
	FilePath     string            `bson:"filePath"        json:"filePath"`
	Document     string            `bson:"document"        json:"document"`
	Version      int               `bson:"version"         json:"version"`
	Status       SessionStatus     `bson:"status"          json:"status"`
	Participants []ParticipantInfo `bson:"participants"    json:"participants"`
	CreatedAt    time.Time         `bson:"createdAt"       json:"createdAt"`
	EndedAt      *time.Time        `bson:"endedAt"         json:"endedAt"`
}
