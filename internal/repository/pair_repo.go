package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PairRepo struct {
	col *mongo.Collection
}

func NewPairRepo(db *mongo.Database) *PairRepo {
	return &PairRepo{col: db.Collection("pair_sessions")}
}

// Mongo CRUD

func (r *PairRepo) Create(ctx context.Context, s *models.PairSession) error {
	s.ID = bson.NewObjectID()
	s.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, s)
	return err
}

func (r *PairRepo) GetByID(ctx context.Context, id string) (*models.PairSession, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	var s models.PairSession
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *PairRepo) UpdateDocument(ctx context.Context, id string, doc string, version int) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid session id")
	} 
	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"document": doc, "version": version}},
		options.UpdateOne(),
	)
	return err
}

func (r *PairRepo) AddParticipants(ctx context.Context, id string, p models.ParticipantInfo) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{
			"$set":      bson.M{"status": models.SessionStatusActive},
			"$addToSet": bson.M{"participants": p},
		},
		options.UpdateOne(),
	)
	return err
}

func (r *PairRepo) RemoveParticipant(ctx context.Context, id, userID string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$pull": bson.M{"participants": bson.M{"userId": userID}}},
		options.UpdateOne(),
	)
	return err
}

func (r *PairRepo) EndSession(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	now := time.Now()
	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"status": models.SessionStatusEnded, "endedAt": now}},
		options.UpdateOne(),
	)
	return err
}

func (r *PairRepo) ListByOwner(ctx context.Context, ownerID string) ([]models.PairSession, error) {
	cursor, err := r.col.Find(ctx, bson.M{"ownerId": ownerID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(20))
	if err != nil {
		return nil, err
	}
	var sessions []models.PairSession
	return sessions, cursor.All(ctx, &sessions)
}

// Redis Live State

const redisPairTTL = 24 * time.Hour

func pairKey(id string) string { return "pair:" + id }

func (r *PairRepo) CacheSession(ctx context.Context, s *models.PairSession) {
	b, err := json.Marshal(s)
	if err != nil {
		return
	}
	database.RedisSet(ctx, pairKey(s.ID.Hex()), string(b), redisPairTTL)
}

func (r *PairRepo) GetCached(ctx context.Context, id string) (*models.PairSession, bool) {
	val, ok := database.RedisGet(ctx, pairKey(id))
	if !ok {
		return nil, false
	}
	var s models.PairSession
	if err := json.Unmarshal([]byte(val), &s); err != nil {
		return nil, false
	}
	return &s, true
}

func (r *PairRepo) InvalidateCache(ctx context.Context, id string) {
	database.RedisDel(ctx, pairKey(id))
}
