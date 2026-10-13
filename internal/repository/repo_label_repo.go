
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

func repoLabelCol() *mongo.Collection {
	return database.Collection("repo_labels")
}

// FindRepoLabels returns all labels defined for a repository, sorted alphabetically.
func FindRepoLabels(ctx context.Context, repoID bson.ObjectID) ([]models.RepoLabel, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cur, err := repoLabelCol().Find(timeout, bson.M{"repoId": repoID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(timeout)

	var labels []models.RepoLabel
	if err := cur.All(timeout, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		labels = []models.RepoLabel{}
	}
	return labels, nil
}

// FindRepoLabelByID returns a single label by its _id.
func FindRepoLabelByID(ctx context.Context, id bson.ObjectID) (*models.RepoLabel, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var label models.RepoLabel
	err := repoLabelCol().FindOne(timeout, bson.M{"_id": id}).Decode(&label)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &label, err
}

// InsertRepoLabel inserts a new label and back-fills the generated ID.
func InsertRepoLabel(ctx context.Context, label *models.RepoLabel) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now()
	label.CreatedAt = now
	label.UpdatedAt = now

	res, err := repoLabelCol().InsertOne(timeout, label)
	if err != nil {
		return err
	}
	label.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// UpdateRepoLabel applies a partial update to a label document.
func UpdateRepoLabel(ctx context.Context, id bson.ObjectID, update bson.M) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update["updatedAt"] = time.Now()
	_, err := repoLabelCol().UpdateOne(timeout, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

// DeleteRepoLabel removes a label by its _id.
func DeleteRepoLabel(ctx context.Context, id bson.ObjectID) error {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := repoLabelCol().DeleteOne(timeout, bson.M{"_id": id})
	return err
}
