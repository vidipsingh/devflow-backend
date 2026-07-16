package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type RepoStats struct {
	Size       int `bson:"size"       json:"size"`
	Stars      int `bson:"stars"      json:"stars"`
	Forks      int `bson:"forks"      json:"forks"`
	Watchers   int `bson:"watchers"   json:"watchers"`
	OpenIssues int `bson:"openIssues" json:"openIssues"`
	OpenPRs    int `bson:"openPRs"    json:"openPRs"`
}

type RepoSettings struct {
	HasWiki             bool `bson:"hasWiki"             json:"hasWiki"`
	HasIssues           bool `bson:"hasIssues"           json:"hasIssues"`
	HasProjects         bool `bson:"hasProjects"         json:"hasProjects"`
	RequiresPRReview    bool `bson:"requiresPRReview"    json:"requiresPRReview"`
	RequiresCI          bool `bson:"requiresCI"          json:"requiresCI"`
	AllowMergeCommit    bool `bson:"allowMergeCommit"    json:"allowMergeCommit"`
	AllowSquashMerge    bool `bson:"allowSquashMerge"    json:"allowSquashMerge"`
	AllowRebaseMerge    bool `bson:"allowRebaseMerge"    json:"allowRebaseMerge"`
	DeleteBranchOnMerge bool `bson:"deleteBranchOnMerge" json:"deleteBranchOnMerge"`
}

type Collaborator struct {
	UserID bson.ObjectID `bson:"userId" json:"userId"`
	Role   string        `bson:"role"   json:"role"` // "read" | "write" | "admin"
}

type Repository struct {
	ID            bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name          string          `bson:"name"          json:"name"`
	Slug          string          `bson:"slug"          json:"slug"`
	FullName      string          `bson:"fullName"      json:"fullName"`
	Description   string          `bson:"description"   json:"description"`
	OwnerID       bson.ObjectID   `bson:"ownerId"       json:"ownerId"`
	OwnerType     string          `bson:"ownerType"     json:"ownerType"` // "user" | "org"
	Visibility    string          `bson:"visibility"    json:"visibility"` // "public" | "private"
	IsArchived    bool            `bson:"isArchived"    json:"isArchived"`
	IsFork        bool            `bson:"isFork"        json:"isFork"`
	IsPinned      bool            `bson:"isPinned"      json:"isPinned"`
	ForkedFromID  *bson.ObjectID  `bson:"forkedFromId"  json:"forkedFromId"`
	Stats         RepoStats       `bson:"stats"         json:"stats"`
	StarredBy     []bson.ObjectID `bson:"starredBy"     json:"starredBy"` // user IDs who starred
	DefaultBranch string          `bson:"defaultBranch" json:"defaultBranch"`
	Branches      []string        `bson:"branches"      json:"branches"`
	Tags          []string        `bson:"tags"          json:"tags"`
	Language      string          `bson:"language"      json:"language"`
	Topics        []string        `bson:"topics"        json:"topics"`
	Collaborators []Collaborator  `bson:"collaborators" json:"collaborators"`
	Settings      RepoSettings    `bson:"settings"      json:"settings"`
	LastPushAt    *time.Time      `bson:"lastPushAt"    json:"lastPushAt"`
	CreatedAt     time.Time       `bson:"createdAt"     json:"createdAt"`
	UpdatedAt     time.Time       `bson:"updatedAt"     json:"updatedAt"`
}

type CreateRepoRequest struct {
	Name          string `json:"name"          binding:"required,min=1,max=100"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"    binding:"required,oneof=public private"`
	DefaultBranch string `json:"defaultBranch"`
	AutoInit      bool   `json:"autoInit"`
}

type UpdateRepoRequest struct {
	Description *string       `json:"description"`
	Visibility  *string       `json:"visibility"  binding:"omitempty,oneof=public private"`
	Topics      []string      `json:"topics"`
	Settings    *RepoSettings `json:"settings"`
}
