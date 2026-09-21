package kafka

import "time"

// ─── Repo Events ──────────────────────────────────────────────────────────────

type RepoEvent struct {
    Type      string    `json:"type"`      // "repo.created" | "repo.deleted" | "repo.forked" | "repo.starred"
    RepoID    string    `json:"repoId"`
    RepoSlug  string    `json:"repoSlug"`
    FullName  string    `json:"fullName"`
    OwnerID   string    `json:"ownerId"`
    ActorID   string    `json:"actorId"`
    ActorName string    `json:"actorName"`
    Timestamp time.Time `json:"timestamp"`
}

// ─── PR Events ────────────────────────────────────────────────────────────────

type PREvent struct {
    Type        string    `json:"type"`       // "pr.created" | "pr.merged" | "pr.closed" | "pr.deleted"
    PRNumber    int       `json:"prNumber"`
    PRTitle     string    `json:"prTitle"`
    RepoID      string    `json:"repoId"`
    RepoSlug    string    `json:"repoSlug"`
    FullName    string    `json:"fullName"`
    AuthorID    string    `json:"authorId"`
    AuthorName  string    `json:"authorName"`
    ActorID     string    `json:"actorId"`
	ActorName  string     `json:"actorName"`
    Timestamp   time.Time `json:"timestamp"`
}

// ─── Issue Events ─────────────────────────────────────────────────────────────

type IssueEvent struct {
    Type        string    `json:"type"`       // "issue.created" | "issue.closed" | "issue.deleted"
    IssueNumber int       `json:"issueNumber"`
    IssueTitle  string    `json:"issueTitle"`
    RepoID      string    `json:"repoId"`
    RepoSlug    string    `json:"repoSlug"`
    FullName    string    `json:"fullName"`
    AuthorID    string    `json:"authorId"`
    AuthorName  string    `json:"authorName"`
    ActorID     string    `json:"actorId"`
	ActorName  string     `json:"actorName"`
    Timestamp   time.Time `json:"timestamp"`
}

// ─── File Events ──────────────────────────────────────────────────────────────

type FileEvent struct {
    Type      string    `json:"type"`      // "file.uploaded"
    RepoID    string    `json:"repoId"`
    Branch    string    `json:"branch"`
    Path      string    `json:"path"`
    ActorID   string    `json:"actorId"`
    ActorName string    `json:"actorName"`
    Timestamp time.Time `json:"timestamp"`
}
