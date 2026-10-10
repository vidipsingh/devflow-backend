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

func discussionCol() *mongo.Collection {
	return database.Collection("team_discussions")
}

func discussionReplyCol() *mongo.Collection {
	return database.Collection("team_discussion_replies")
}

// Discussions

func InsertDiscussion(ctx context.Context, d *models.TeamDiscussion) error {
	d.ID = bson.NewObjectID()
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	_, err := discussionCol().InsertOne(ctx, d)
	return err
}

func FindDiscussionsByTeam(ctx context.Context, teamID bson.ObjectID, limit, skip int64) ([]models.TeamDiscussion, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	opts := options.Find().
		SetSort(bson.D{{Key: "pinned", Value: -1}, {Key: "createdAt", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)
	cursor, err := discussionCol().Find(ctx, bson.M{"teamId": teamID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []models.TeamDiscussion
	return out, cursor.All(ctx, &out)
}

func FindDiscussionByID(ctx context.Context, id bson.ObjectID) (*models.TeamDiscussion, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var d models.TeamDiscussion
	err := discussionCol().FindOne(ctx, bson.M{"_id": id}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &d, err
}

func UpdateDiscussion(ctx context.Context, id bson.ObjectID, update bson.M) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	update["updatedAt"] = time.Now()
	_, err := discussionCol().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func DeleteDiscussion(ctx context.Context, id bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := discussionCol().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func IncrDiscussionReplyCount(ctx context.Context, discussionID bson.ObjectID, delta int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := discussionCol().UpdateOne(
		ctx,
		bson.M{"_id": discussionID},
		bson.M{"$inc": bson.M{"replyCount": delta}, "$set": bson.M{"updatedAt": time.Now()}},
	)
	return err
}

// Replies

func InsertDiscussionReply(ctx context.Context, r *models.TeamDiscussionReply) error {
	r.ID = bson.NewObjectID()
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	_, err := discussionReplyCol().InsertOne(ctx, r)
	return err
}

func FindRepliesByDiscussion(ctx context.Context, discussionID bson.ObjectID, limit, skip int64) ([]models.TeamDiscussionReply, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(limit).
		SetSkip(skip)
	cursor, err := discussionReplyCol().Find(ctx, bson.M{"discussionId": discussionID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []models.TeamDiscussionReply
	return out, cursor.All(ctx, &out)
}

func FindReplyByID(ctx context.Context, id bson.ObjectID) (*models.TeamDiscussionReply, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var r models.TeamDiscussionReply
	err := discussionReplyCol().FindOne(ctx, bson.M{"_id": id}).Decode(&r)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &r, err
}

func DeleteReply(ctx context.Context, id bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := discussionReplyCol().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func DeleteRepliesByDiscussion(ctx context.Context, discussionID bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := discussionReplyCol().DeleteMany(ctx, bson.M{"discussionId": discussionID})
	return err
}
