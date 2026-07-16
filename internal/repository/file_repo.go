package repository

import (
    "context"
    "time"

    "devflow-backend/internal/database"
    "devflow-backend/internal/models"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

const filesCol   = "repo_files"
const commitsCol = "repo_commits"

// UpsertFile inserts or replaces the metadata for a file at a given path+branch.
func UpsertFile(ctx context.Context, f *models.RepoFile) error {
	coll := database.Collection(filesCol)
	filter := bson.M{
		"repoId": f.RepoID,
		"branch": f.Branch,
		"path":	  f.Path,
	}
	f.UpdatedAt = time.Now()
	update := bson.M{"$set": f}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := coll.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindFilesByDir returns all file metadata for a given repo/branch/dir
func FindFilesByDir(ctx context.Context, repoID bson.ObjectID, branch, dir string) ([]models.RepoFile, error) {
	coll := database.Collection(filesCol)
	filter := bson.M{
		"repoId": repoID,
		"branch": branch,
		"dir":	  dir,
	}
	cursor, err := coll.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "isDirectory", Value: -1}, // dirs first
		{Key: "name", Value: 1},
	}))
	if err != nil {
		return nil, err
	}
	var files []models.RepoFile
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// FindFileByPath returns the metadata for a single file at a specific path
func FindFileByPath(ctx context.Context, repoID bson.ObjectID, branch, path string) (*models.RepoFile, error) {
	coll := database.Collection(filesCol)
	var f models.RepoFile
	err := coll.FindOne(ctx , bson.M{
		"repoId": repoID,
        "branch": branch,
        "path":   path,
	}).Decode(&f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// InsertCommit saves a commit record
func InsertCommit(ctx context.Context, c *models.RepoCommit) error {
	c.CreatedAt = time.Now()
	_, err := database.Collection(commitsCol).InsertOne(ctx, c)
	return err
}

// FindCommitsByRepo returns recent commits for a repo/branch
func FindCommitsByRepo(ctx context.Context, repoID bson.ObjectID, branch string, limit int64) ([]models.RepoCommit, error) {
	coll := database.Collection(commitsCol)
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit)
	cursor, err := coll.Find(ctx, bson.M{"repoId": repoID, "branch": branch}, opts)
	if err != nil {
		return nil, err
	}
	var commits []models.RepoCommit
	if err := cursor.All(ctx, &commits); err != nil {
		return nil, err
	}
	return commits, nil
}
