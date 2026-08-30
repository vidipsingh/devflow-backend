package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

// ReviewSession ties a PR to a live pair-review session
type ReviewSession struct {
    ID           bson.ObjectID  `bson:"_id,omitempty"    json:"id"`
    PRID         bson.ObjectID  `bson:"prId"             json:"prId"`
    RepoID       bson.ObjectID  `bson:"repoId"           json:"repoId"`
    OwnerID      string              `bson:"ownerId"          json:"ownerId"`
    OwnerName    string              `bson:"ownerName"        json:"ownerName"`
    Status       string              `bson:"status"           json:"status"` // waiting | active | ended
    Participants []ReviewParticipant `bson:"participants"     json:"participants"`
    CreatedAt    time.Time           `bson:"createdAt"        json:"createdAt"`
    EndedAt      *time.Time          `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
}

type ReviewParticipant struct {
    UserID   string    `bson:"userId"   json:"userId"`
    UserName string    `bson:"userName" json:"userName"`
    Color    string    `bson:"color"    json:"color"`
    JoinedAt time.Time `bson:"joinedAt" json:"joinedAt"`
}
