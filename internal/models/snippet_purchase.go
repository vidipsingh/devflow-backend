package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SnippetPurchase records a verified purchase of a paid snippet.
type SnippetPurchase struct {
	ID              bson.ObjectID `bson:"_id,omitempty"      json:"id"`
	SnippetID       bson.ObjectID `bson:"snippetId"          json:"snippetId"`
	BuyerID         bson.ObjectID `bson:"buyerId"            json:"buyerId"`
	SellerID        bson.ObjectID `bson:"sellerId"           json:"sellerId"`
	AmountPaise     int64         `bson:"amountPaise"        json:"amountPaise"`
	Currency        string        `bson:"currency"           json:"currency"`
	RazorpayOrderID string        `bson:"razorpayOrderId"    json:"razorpayOrderId"`
	RazorpayPayID   string        `bson:"razorpayPaymentId"  json:"razorpayPaymentId"`
	Status          string        `bson:"status"             json:"status"` // created | paid | failed
	CreatedAt       time.Time     `bson:"createdAt"          json:"createdAt"`
	PaidAt          *time.Time    `bson:"paidAt,omitempty"   json:"paidAt,omitempty"`
}
