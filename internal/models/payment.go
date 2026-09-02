package models

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Payment records every Razorpay transaction attempt and its outcome.
type Payment struct {
	ID              bson.ObjectID `bson:"_id,omitempty"     json:"id"`
	UserID          string        `bson:"userId"            json:"userId"`
	PlanKey         string        `bson:"planKey"           json:"planKey"`      // "pro" | "team"
	RazorpayOrderID string        `bson:"razorpayOrderId"   json:"razorpayOrderId"`
	RazorpayPayID   string        `bson:"razorpayPaymentId" json:"razorpayPaymentId"`
	AmountPaise     int64         `bson:"amountPaise"       json:"amountPaise"`  // smallest unit (paise)
	Currency        string        `bson:"currency"          json:"currency"`     // "INR"
	Status          string        `bson:"status"            json:"status"`       // "created" | "paid" | "failed"
	CreatedAt       time.Time     `bson:"createdAt"         json:"createdAt"`
	PaidAt          *time.Time    `bson:"paidAt,omitempty"  json:"paidAt,omitempty"`
}