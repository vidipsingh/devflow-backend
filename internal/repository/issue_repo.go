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

func issueCol() *mongo.Collection {
	return database.Collection("issues")
}

// NextIssueNumber returns max(number)+1 for the given repo, starting at 1
func NextIssueNumber(ctx context.Context, repoID bson.ObjectID) (int, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	opts := options.FindOne().SetSort(bson.D{{Key: "number", Value: -1}})
	var last models.Issue
	err := issueCol().FindOne(timeout, bson.M{"repoId": repoID}, opts).Decode(&last)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return last.Number + 1, nil
}

// CreateIssue inserts a new issue and back-fills the generated ID
func CreateIssue(ctx context.Context, issue *models.Issue) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := issueCol().InsertOne(timeout, issue)
	if err != nil {
		return err
	}
	issue.ID = result.InsertedID.(bson.ObjectID)
	return nil
}

// FindIssuesByRepo returns paginated issues for a repo
// state: "" = all, "open", "closed"
func FindIssueByRepo(ctx context.Context, repoID bson.ObjectID, state string, page, limit int64) ([]models.Issue, int64, error) {
	timeout, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	filter := bson.M{"repoId": repoID}
	if state == "open" || state == "closed" {
		filter["state"] = state
	}

	total, err := issueCol().CountDocuments(timeout, filter)
	if err != nil{
		return nil, 0, err
	}

	skip := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)
	
	cursor, err := issueCol().Find(timeout, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(timeout)

	var issues []models.Issue
	if err := cursor.All(timeout, &issues); err != nil {
		return nil, 0, err
	}
	return issues, total, nil
}

// FindIssueByNumber finds one issue by repo + sequential number
func FindIssueByNumber(ctx context.Context, repoID bson.ObjectID, number int) (*models.Issue, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var issue models.Issue
	err := issueCol().FindOne(timeout, bson.M{"repoId": repoID, "number": number}).Decode(&issue)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &issue, err
}

// FindIssueByID finds an issue by its MongoDB _id
func FindIssueByID(ctx context.Context, id bson.ObjectID) (*models.Issue, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var issue models.Issue
	err := issueCol().FindOne(timeout, bson.M{"_id": id}).Decode(&issue)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &issue, err
}

// UpdateIssueRaw applies an arbitrary MongoDB update document
func UpdateIssueRaw(ctx context.Context, id bson.ObjectID, update bson.M) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := issueCol().UpdateOne(timeout, bson.M{"_id": id}, update)
	return err
}

// UpdateIssueRawWithFilter applies an update with array filters
func UpdateIssueRawWithFilter(ctx context.Context, id bson.ObjectID, update bson.M, arrayFilters []interface{}) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := options.UpdateOne().SetArrayFilters(arrayFilters)
	_, err := issueCol().UpdateOne(timeout, bson.M{"_id": id}, update, opts)
	return err
}

func DeleteIssue(ctx context.Context, id bson.ObjectID) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := issueCol().DeleteOne(timeout, bson.M{"_id": id})
	return err
}
