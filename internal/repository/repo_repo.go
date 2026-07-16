package repository

import (
	"context"
	"errors"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func repoCol() *mongo.Collection {
	return database.Collection("repositories")
}

// FindReposByOwner returns all repos for ownerID sorted by updatedAt desc
func FindReposByOwner(ctx context.Context, ownerId bson.ObjectID, visibility string) ([]models.Repository, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"ownerId": ownerId}
	if visibility == "public" || visibility == "private" {
		filter["visibility"] = visibility
	}

	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	cursor, err := repoCol().Find(timeout, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(timeout)

	var repos []models.Repository
	if err := cursor.All(timeout, &repos); err != nil{
		return nil, err
	}
	return repos, nil
}

// FindRepoByOwnerAndSlug finds a single repo by ownerId + slug. Returns nil, nil if not found
func FindRepoByOwnerAndSlug(ctx context.Context, ownerID bson.ObjectID, slug string) (*models.Repository, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var repo models.Repository
	err := repoCol().FindOne(timeout, bson.M{"ownerId": ownerID, "slug": slug}).Decode(&repo)
	if errors.Is(err, mongo.ErrNoDocuments){
		return nil, nil
	}
	return &repo, err
}

// CreateRepo inserts a new repository document
func CreateRepo(ctx context.Context, repo *models.Repository) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := repoCol().InsertOne(timeout, repo)
	if err != nil {
		return err
	}
	repo.ID = result.InsertedID.(bson.ObjectID)
	return nil
}

// UpdateRepo performs a $set update on a repo by ID
func UpdateRepo(ctx context.Context, id bson.ObjectID, update bson.M) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := repoCol().UpdateOne(timeout, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

// DeleteRepo deletes a repository by ID
func DeleteRepo(ctx context.Context, id bson.ObjectID) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := repoCol().DeleteOne(timeout, bson.M{"_id": id})
	return err
}

// FindRepoBySlug finds a repo by slug only (any owner) — used for starring
func FindRepoBySlug(ctx context.Context, slug string) (*models.Repository, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var repo models.Repository
	err := repoCol().FindOne(timeout, bson.M{"slug": slug}).Decode(&repo)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &repo, err
}

// UpdateRepoRaw applies an arbitrary MongoDB update document (supports $inc, $set, etc.)
func UpdateRepoRaw(ctx context.Context, id bson.ObjectID, update bson.M) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := repoCol().UpdateOne(timeout, bson.M{"_id": id}, update)
	return err
}