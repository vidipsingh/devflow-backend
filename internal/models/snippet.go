package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Snippet struct {
	ID          bson.ObjectID 	   `bson:"_id,omitempty"  json:"id"`
	CreatorID   bson.ObjectID 	   `bson:"creatorId"      json:"creatorId"`
	Title       string             `bson:"title"          json:"title"`
	Description string             `bson:"description"    json:"description"`
	Code        string             `bson:"code"           json:"code"`
	Preview     string             `bson:"preview"        json:"preview"`
	Language    string             `bson:"language"       json:"language"`
	Tags        []string           `bson:"tags"           json:"tags"`
	Category    string             `bson:"category"       json:"category"`
	Status      string             `bson:"status"         json:"status"` // draft | published
	Version     string             `bson:"version"        json:"version"`
	Pricing     SnippetPricing     `bson:"pricing"        json:"pricing"`
	Stats       SnippetStats       `bson:"stats"          json:"stats"`
	Earnings    SnippetEarnings    `bson:"earnings"       json:"earnings"`
	CreatedAt   time.Time          `bson:"createdAt"      json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"      json:"updatedAt"`
	PublishedAt *time.Time         `bson:"publishedAt"    json:"publishedAt"`

	CreatorUsername string `bson:"creatorUsername,omitempty" json:"creatorUsername,omitempty"`
}

type SnippetPricing struct {
	Type     string  `bson:"type"     json:"type"` // free | paid
	Price    float64 `bson:"price"    json:"price"`
	Currency string  `bson:"currency" json:"currency"`
}

type SnippetStats struct {
	Downloads   int     `bson:"downloads"   json:"downloads"`
	Views       int     `bson:"views"       json:"views"`
	Rating      float64 `bson:"rating"      json:"rating"`
	RatingCount int     `bson:"ratingCount" json:"ratingCount"`
	Purchases   int     `bson:"purchases"   json:"purchases"`
}

type SnippetEarnings struct {
	TotalRevenue    float64 `bson:"totalRevenue"    json:"totalRevenue"`
	CreatorEarnings float64 `bson:"creatorEarnings" json:"creatorEarnings"`
	PlatformFee     float64 `bson:"platformFee"     json:"platformFee"`
}
