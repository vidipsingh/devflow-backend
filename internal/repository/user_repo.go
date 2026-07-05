package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
    "devflow-backend/internal/models"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
)

// FindUserByEmail looks up a user by email address
// Returns nil, nil if not found (not an error)
func FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	col := database.Collection("users")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User
	err := col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil // user not found — not an error
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByUsername looks up a user by username
func FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	col := database.Collection("users")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User
	err := col.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil // user not found — not an error
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByOAuth finds a user by provider + provider's user ID
func FindUserByOAuth(ctx context.Context, provider, oauthID string) (*models.User, error) {
	col := database.Collection("users")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user models.User
	err := col.FindOne(ctx, bson.M{
		"oauthProvider": provider,
        "oauthId":       oauthID,
    }).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdateUser(ctx context.Context, user *models.User) error {
	col := database.Collection("users")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user.UpdatedAt = time.Now()
	_, err := col.ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	return err
}

// CreateUser inserts a new user document
func CreateUser(ctx context.Context, user *models.User) error {
	col := database.Collection("users")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user.ID = bson.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Plan == "" {
		user.Plan = "free"
	}
	
	_, err := col.InsertOne(ctx, user)
	return err
}