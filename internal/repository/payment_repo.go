package repository

import (
	"context"
	"time"

	"devflow-backend/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PaymentRepo struct {
	col *mongo.Collection
}

func NewPaymentRepo(db *mongo.Database) *PaymentRepo {
	return &PaymentRepo{col: db.Collection("payments")}
}

func (r *PaymentRepo) Create(ctx context.Context, p *models.Payment) error {
	p.ID = bson.NewObjectID()
	p.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, p)
	return err
}

func (r *PaymentRepo) FindByOrderID(ctx context.Context, orderID string) (*models.Payment, error) {
	var p models.Payment
	err := r.col.FindOne(ctx, bson.M{"razorpayOrderId": orderID}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepo) MarkPaid(ctx context.Context, orderID, paymentID string) error {
	now := time.Now()
	_, err := r.col.UpdateOne(ctx,
		bson.M{"razorpayOrderId": orderID},
		bson.M{"$set": bson.M{
			"razorpayPaymentId": paymentID,
			"status":            "paid",
			"paidAt":            now,
		}},
	)
	return err
}

func (r *PaymentRepo) ListByUser(ctx context.Context, userID string) ([]models.Payment, error) {
	opts := options.Find().SetSort(bson.M{"createdAt": -1}).SetLimit(50)
	cur, err := r.col.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, err
	}
	var payments []models.Payment
	return payments, cur.All(ctx, &payments)
}
