package models

import "time"

type PlatformStats struct {
	TotalUsers       int64 `json:"totalUsers"`
	TotalRepositories int64 `json:"totalRepositories"`
	TotalSnippets    int64 `json:"totalSnippets"`
	TotalAIReviews   int64 `json:"totalAIReviews"`
}

type DailyAnalytics struct {
	UserID   interface{} `bson:"userId"   json:"userId"`
	Date     time.Time   `bson:"date"     json:"date"`
	Activity Activity    `bson:"activity" json:"activity"`
	Code     CodeStats   `bson:"code"     json:"code"`
}

type Activity struct {
	Commits               int `bson:"commits"               json:"commits"`
	PullRequestsOpened    int `bson:"pullRequestsOpened"    json:"pullRequestsOpened"`
	PullRequestsMerged    int `bson:"pullRequestsMerged"    json:"pullRequestsMerged"`
	IssuesOpened          int `bson:"issuesOpened"          json:"issuesOpened"`
	IssuesClosed          int `bson:"issuesClosed"          json:"issuesClosed"`
	CodeReviewsGiven      int `bson:"codeReviewsGiven"      json:"codeReviewsGiven"`
}

type CodeStats struct {
	LinesAdded         int `bson:"linesAdded"         json:"linesAdded"`
	LinesDeleted       int `bson:"linesDeleted"       json:"linesDeleted"`
	FilesChanged       int `bson:"filesChanged"       json:"filesChanged"`
	RepositoriesActive int `bson:"repositoriesActive" json:"repositoriesActive"`
}
