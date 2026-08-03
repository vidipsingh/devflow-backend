package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func prCol() *mongo.Collection {
	return database.Collection("pull_requests")
}

func NextPRNumber(ctx context.Context, repoID bson.ObjectID) (int, error) {
	count, err := prCol().CountDocuments(ctx, bson.M{"repoId": repoID})
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}

func InsertPR(ctx context.Context, pr *models.PullRequest) error {
	pr.CreatedAt = time.Now()
	pr.UpdatedAt = time.Now()
	res, err := prCol().InsertOne(ctx, pr)
	if err != nil {
		return err
	}
	pr.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

func FindPRsRepo(ctx context.Context, repoID bson.ObjectID, state string) ([]models.PullRequest, error) {
	filter := bson.M{"repoId": repoID}
	if state != "" {
		filter["state"] = state
	}
	opts := options.Find().SetSort(bson.D{{Key: "number", Value: -1}})
	cur, err := prCol().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var prs []models.PullRequest
	return prs, cur.All(ctx, &prs)
}

func FindPRByNumber(ctx context.Context, repoID bson.ObjectID, number int) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := prCol().FindOne(ctx, bson.M{"repoId": repoID, "number": number}).Decode(&pr)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func UpdatePR(ctx context.Context, id bson.ObjectID, update bson.M) error {
	update["updatedAt"] = time.Now()
	_, err := prCol().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func DeletePR(ctx context.Context, id bson.ObjectID) error {
	_, err := prCol().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func AddPRComment(ctx context.Context, prID bson.ObjectID, comment models.PRComment) error {
	comment.ID = bson.NewObjectID()
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	_, err := prCol().UpdateOne(ctx, bson.M{"_id": prID}, bson.M{
		"$push": bson.M{"comments": comment},
		"$inc":  bson.M{"commentCount": 1},
		"$set":  bson.M{"updatedAt": time.Now()},
	})
	return err
}

func UpdatePRComment(ctx context.Context, prID bson.ObjectID, commentID bson.ObjectID, body string) error {
	_, err := prCol().UpdateOne(ctx,
		bson.M{"_id": prID, "comments._id": commentID},
		bson.M{"$set": bson.M{
			"comments.$.body":      body,
			"comments.$.isEdited": true,
			"comments.$.updatedAt": time.Now(),
			"updatedAt":            time.Now(),
		}},
	)
	return err
}

func DeletePRComment(ctx context.Context, prID bson.ObjectID, commentID bson.ObjectID) error {
	_, err := prCol().UpdateOne(ctx, bson.M{"_id": prID}, bson.M{
		"$pull":  bson.M{"comments": bson.M{"_id": commentID}},
		"$inc":   bson.M{"commentCount": -1},
		"$set":   bson.M{"updatedAt": time.Now()},
	})
	return err
}
