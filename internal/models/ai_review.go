package models

import "time"

// models/ai_review.go
type AISuggestion struct {
    FilePath   string `bson:"filePath"   json:"filePath"`
    LineStart  int    `bson:"lineStart"  json:"lineStart"`
    LineEnd    int    `bson:"lineEnd"    json:"lineEnd"`
    Severity   string `bson:"severity"   json:"severity"`   // "info" | "warning" | "critical"
    Category   string `bson:"category"   json:"category"`   // "modularity" | "robustness" | "style" | "security" | "performance"
    Message    string `bson:"message"    json:"message"`
    Suggestion string `bson:"suggestion" json:"suggestion"`
}

type AIReview struct {
    Status      string         `bson:"status"      json:"status"`       // "pending" | "done" | "error" | "skipped"
    Summary     string         `bson:"summary"     json:"summary"`
    Suggestions []AISuggestion `bson:"suggestions" json:"suggestions"`
    Model       string         `bson:"model"       json:"model"`
    ReviewedAt  *time.Time     `bson:"reviewedAt"  json:"reviewedAt"`
    ErrorMsg    string         `bson:"errorMsg"    json:"errorMsg"`
}
