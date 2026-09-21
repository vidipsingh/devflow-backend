package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

// ActivityEvent is written to MongoDB by the activity Kafka consumer
// and served to the dashboard activity feed API endpoint.
type ActivityEvent struct {
    ID        bson.ObjectID  `bson:"_id,omitempty" json:"id"`
    Type      string         `bson:"type"          json:"type"`
    ActorID   string         `bson:"actorId"       json:"actorId"`
    ActorName string         `bson:"actorName"     json:"actorName"`
    RepoID    string         `bson:"repoId"        json:"repoId"`
    RepoName  string         `bson:"repoName"      json:"repoName"`
    Meta      map[string]any `bson:"meta"          json:"meta,omitempty"`
    Timestamp time.Time      `bson:"timestamp"     json:"timestamp"`
}
