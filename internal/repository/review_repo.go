package repository

import (
    "context"
    "time"

    "devflow-backend/internal/models"
    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ReviewRepo struct {
	col *mongo.Collection
}

func NewReviewRepo(db *mongo.Database) *ReviewRepo {
	return &ReviewRepo{col: db.Collection("review_session")}
}

func (r *ReviewRepo) Create(ctx context.Context,  sess *models.ReviewSession) error {
	sess.ID = bson.NewObjectID()
	sess.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, sess)
	return err
}

func (r *ReviewRepo) GetByID(ctx context.Context, id string) (*models.ReviewSession, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil { return nil, err }
	var s models.ReviewSession
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&s)
	return &s, err
}

func (r *ReviewRepo) ListByPR(ctx context.Context, prID string) ([]models.ReviewSession, error) {
	oid, err := bson.ObjectIDFromHex(prID)
	if err != nil { return nil, err }
	cur, err := r.col.Find(ctx, bson.M{"prId": oid}, options.Find().SetSort(bson.M{"createdAt": -1}))
	if err != nil { return nil, err }
	var out []models.ReviewSession
	return out, cur.All(ctx, &out)
}

func (r *ReviewRepo) AddParticipant(ctx context.Context, id string, p models.ReviewParticipant) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil { return err }
	_, err = r.col.UpdateOne(ctx,
        bson.M{"_id": oid},
        bson.M{
            "$addToSet": bson.M{"participants": p},
            "$set":      bson.M{"status": "active"},
        })
    return err
}

func (r *ReviewRepo) RemoveParticipant(ctx context.Context, id, userID string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil { return err }
	_, err = r.col.UpdateOne(ctx,
        bson.M{"_id": oid},
        bson.M{"$pull": bson.M{"participants": bson.M{"userId": userID}}})
    return err
}

func (r *ReviewRepo) End(ctx context.Context, id string) error {
    oid, err := bson.ObjectIDFromHex(id)
    if err != nil { return err }
    now := time.Now()
    _, err = r.col.UpdateOne(ctx,
        bson.M{"_id": oid},
        bson.M{"$set": bson.M{"status": "ended", "endedAt": now}})
    return err
}
