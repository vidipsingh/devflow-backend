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

// Helpers
func snippetsCol() *mongo.Collection  { return database.Collection("snippets") }
func purchasesCol() *mongo.Collection { return database.Collection("snippet_purchases") }
func reviewsCol() *mongo.Collection   { return database.Collection("snippet_reviews") }

// Snippets

// FindPublishedSnippets returns published snippets sorted by the requested field.
func FindPublishedSnippets(ctx context.Context, limit int, sortBy string) ([]models.Snippet, error) {
	sortField := map[string]string{
		"downloads": "stats.downloads",
		"rating":    "stats.rating",
		"recent":    "publishedAt",
		"purchases": "stats.purchases",
	}[sortBy]
	if sortField == "" {
		sortField = "stats.downloads"
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: -1}}).
		SetLimit(int64(limit))

	cur, err := snippetsCol().Find(ctx, bson.M{"status": "published"}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Snippet
	return out, cur.All(ctx, &out)
}

// FindSnippetByID returns a single snippet by its ObjectID hex string.
func FindSnippetByID(ctx context.Context, id string) (*models.Snippet, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid snippet id")
	}
	var s models.Snippet
	err = snippetsCol().FindOne(ctx, bson.M{"_id": oid}).Decode(&s)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &s, err
}

// FindSnippetsByCreator returns all snippets (any status) owned by a user.
func FindSnippetsByCreator(ctx context.Context, creatorID string) ([]models.Snippet, error) {
	oid, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, errors.New("invalid creator id")
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := snippetsCol().Find(ctx, bson.M{"creatorId": oid}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Snippet
	return out, cur.All(ctx, &out)
}

// InsertSnippet creates a new snippet document.
func InsertSnippet(ctx context.Context, s *models.Snippet) error {
	s.ID = bson.NewObjectID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	_, err := snippetsCol().InsertOne(ctx, s)
	return err
}

// UpdateSnippet replaces the mutable fields of a snippet.
func UpdateSnippet(ctx context.Context, id, ownerID string, update bson.M) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid snippet id")
	}
	ooid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return errors.New("invalid owner id")
	}
	update["updatedAt"] = time.Now()
	res, err := snippetsCol().UpdateOne(ctx,
		bson.M{"_id": oid, "creatorId": ooid},
		bson.M{"$set": update},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("snippet not found or not yours")
	}
	return nil
}

// PublishSnippet sets status=published and records publishedAt.
func PublishSnippet(ctx context.Context, id, ownerID string) error {
	now := time.Now()
	return UpdateSnippet(ctx, id, ownerID, bson.M{
		"status":      "published",
		"publishedAt": now,
	})
}

// UnpublishSnippet sets status=draft.
func UnpublishSnippet(ctx context.Context, id, ownerID string) error {
	return UpdateSnippet(ctx, id, ownerID, bson.M{"status": "draft"})
}

// DeleteSnippet hard-deletes a snippet owned by the caller.
func DeleteSnippet(ctx context.Context, id, ownerID string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid snippet id")
	}
	ooid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return errors.New("invalid owner id")
	}
	res, err := snippetsCol().DeleteOne(ctx, bson.M{"_id": oid, "creatorId": ooid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("snippet not found or not yours")
	}
	return nil
}

// IncrementSnippetViews atomically increments the view counter.
func IncrementSnippetViews(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = snippetsCol().UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$inc": bson.M{"stats.views": 1}},
	)
	return err
}

// IncrementSnippetDownloads atomically increments the download counter.
func IncrementSnippetDownloads(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = snippetsCol().UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$inc": bson.M{"stats.downloads": 1}},
	)
	return err
}

// RecalcSnippetRating re-computes the denormalised average rating stored on the snippet.
func RecalcSnippetRating(ctx context.Context, snippetID string) error {
	oid, err := bson.ObjectIDFromHex(snippetID)
	if err != nil {
		return err
	}

	// Aggregate: avg + count from snippet_reviews
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"snippetId": oid}}},
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"avg":   bson.M{"$avg": "$rating"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err := reviewsCol().Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var result []struct {
		Avg   float64 `bson:"avg"`
		Count int     `bson:"count"`
	}
	if err := cur.All(ctx, &result); err != nil || len(result) == 0 {
		return err
	}

	_, err = snippetsCol().UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"stats.rating":      result[0].Avg,
			"stats.ratingCount": result[0].Count,
		}},
	)
	return err
}

// Purchases

// InsertPurchase creates a new purchase record (status = "created").
func InsertPurchase(ctx context.Context, p *models.SnippetPurchase) error {
	p.ID = bson.NewObjectID()
	p.CreatedAt = time.Now()
	_, err := purchasesCol().InsertOne(ctx, p)
	return err
}

// FindPurchaseByOrderID looks up a purchase by its Razorpay order ID.
func FindPurchaseByOrderID(ctx context.Context, orderID string) (*models.SnippetPurchase, error) {
	var p models.SnippetPurchase
	err := purchasesCol().FindOne(ctx, bson.M{"razorpayOrderId": orderID}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &p, err
}

// MarkPurchasePaid updates a purchase record to paid.
func MarkPurchasePaid(ctx context.Context, orderID, paymentID string) error {
	now := time.Now()
	_, err := purchasesCol().UpdateOne(ctx,
		bson.M{"razorpayOrderId": orderID},
		bson.M{"$set": bson.M{
			"razorpayPaymentId": paymentID,
			"status":            "paid",
			"paidAt":            now,
		}},
	)
	return err
}

// HasPurchased returns true when the user has a paid purchase for the snippet.
func HasPurchased(ctx context.Context, snippetID string, buyerID string) (bool, error) {
	soid, err := bson.ObjectIDFromHex(snippetID)
	if err != nil {
		return false, err
	}
	boid, err := bson.ObjectIDFromHex(buyerID)
	if err != nil {
		return false, err
	}
	count, err := purchasesCol().CountDocuments(ctx, bson.M{
		"snippetId": soid,
		"buyerId":   boid,
		"status":    "paid",
	})
	return count > 0, err
}

// ListPurchasesByBuyer returns all paid purchases for a user.
func ListPurchasesByBuyer(ctx context.Context, buyerID string) ([]models.SnippetPurchase, error) {
	boid, err := bson.ObjectIDFromHex(buyerID)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "paidAt", Value: -1}})
	cur, err := purchasesCol().Find(ctx, bson.M{"buyerId": boid, "status": "paid"}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.SnippetPurchase
	return out, cur.All(ctx, &out)
}

// IncrementSnippetPurchaseCount atomically increments the purchase counter.
func IncrementSnippetPurchaseCount(ctx context.Context, id string) (int64, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return 0, err
	}
	res, err := snippetsCol().UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$inc": bson.M{"stats.purchases": 1}},
	)
	return res.ModifiedCount, err
}

// Reviews

// FindReviewsBySnippet returns all reviews for a snippet newest-first.
func FindReviewsBySnippet(ctx context.Context, snippetID string) ([]models.SnippetReview, error) {
	oid, err := bson.ObjectIDFromHex(snippetID)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := reviewsCol().Find(ctx, bson.M{"snippetId": oid}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.SnippetReview
	return out, cur.All(ctx, &out)
}

// HasReviewed returns true if the user already left a review for this snippet.
func HasReviewed(ctx context.Context, snippetID, authorID string) (bool, error) {
	soid, err := bson.ObjectIDFromHex(snippetID)
	if err != nil {
		return false, err
	}
	aoid, err := bson.ObjectIDFromHex(authorID)
	if err != nil {
		return false, err
	}
	count, err := reviewsCol().CountDocuments(ctx, bson.M{
		"snippetId": soid,
		"authorId":  aoid,
	})
	return count > 0, err
}

// InsertReview saves a review document.
func InsertReview(ctx context.Context, r *models.SnippetReview) error {
	r.ID = bson.NewObjectID()
	r.CreatedAt = time.Now()
	_, err := reviewsCol().InsertOne(ctx, r)
	return err
}
