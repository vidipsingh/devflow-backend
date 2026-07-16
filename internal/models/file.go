package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

type RepoFile struct {
    ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
    RepoID      bson.ObjectID `bson:"repoId"        json:"repoId"`
    Path        string        `bson:"path"          json:"path"`        
    Name        string        `bson:"name"          json:"name"`        // "main.go"
    Dir         string        `bson:"dir"           json:"dir"`         // "src"
    Size        int64         `bson:"size"          json:"size"`        // bytes
    MimeType    string        `bson:"mimeType"      json:"mimeType"`    // "text/plain"
    Encoding    string        `bson:"encoding"      json:"encoding"`    // "utf-8" | "binary"
    SHA         string        `bson:"sha"           json:"sha"`         // SHA256 of content
    GridFSID    bson.ObjectID `bson:"gridfsId"      json:"gridfsId"`    // pointer to fs.files doc
    Branch      string        `bson:"branch"        json:"branch"`      // "main"
    CommitID    bson.ObjectID `bson:"commitId"      json:"commitId"`
    IsDirectory bool          `bson:"isDirectory"   json:"isDirectory"`
    CreatedAt   time.Time     `bson:"createdAt"     json:"createdAt"`
    UpdatedAt   time.Time     `bson:"updatedAt"     json:"updatedAt"`
}

type RepoCommit struct {
    ID          bson.ObjectID   `bson:"_id,omitempty" json:"id"`
    RepoID      bson.ObjectID   `bson:"repoId"        json:"repoId"`
    Branch      string          `bson:"branch"        json:"branch"`
    Message     string          `bson:"message"       json:"message"`
    AuthorID    bson.ObjectID   `bson:"authorId"      json:"authorId"`
    AuthorName  string          `bson:"authorName"    json:"authorName"`
    ShortHash   string          `bson:"shortHash"     json:"shortHash"` 
    FilePaths   []string        `bson:"filePaths"     json:"filePaths"`
    Additions   int             `bson:"additions"     json:"additions"`
    Deletions   int             `bson:"deletions"     json:"deletions"`
    CreatedAt   time.Time       `bson:"createdAt"     json:"createdAt"`
}

type FileTreeEntry struct {
    Name        string        `json:"name"`
    Path        string        `json:"path"`
    Type        string        `json:"type"`     // "file" | "dir"
    Size        int64         `json:"size"`
    MimeType    string        `json:"mimeType"`
    SHA         string        `json:"sha"`
    LastCommit  *CommitSummary `json:"lastCommit,omitempty"`
}

type CommitSummary struct {
    Hash    string    `json:"hash"`
    Message string    `json:"message"`
    Author  string    `json:"author"`
    Date    time.Time `json:"date"`
}

type UploadFileRequest struct {
    Path    string `json:"path"    binding:"required"` // "src/main.go"
    Content string `json:"content" binding:"required"` // base64-encoded file content
    Message string `json:"message"`                   // commit message
    Branch  string `json:"branch"`                    // defaults to repo.DefaultBranch
}