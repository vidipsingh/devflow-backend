package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

// Notification is stored in the "notifications" collection.
// One notification is created per recipient per event.
type Notification struct {
    ID          bson.ObjectID  `bson:"_id,omitempty"  json:"id"`
    RecipientID string         `bson:"recipientId"    json:"recipientId"`
    Type        string         `bson:"type"           json:"type"`
    ActorID     string         `bson:"actorId"        json:"actorId"`
    ActorName   string         `bson:"actorName"      json:"actorName"`
    RepoID      string         `bson:"repoId"         json:"repoId"`
    RepoName    string         `bson:"repoName"       json:"repoName"`
    Meta        map[string]any `bson:"meta,omitempty" json:"meta,omitempty"`
    Read        bool           `bson:"read"           json:"read"`
    CreatedAt   time.Time      `bson:"createdAt"      json:"createdAt"`
}
