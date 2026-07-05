package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID           bson.ObjectID      `bson:"_id,omitempty"    json:"id"`
	Name         string        		`bson:"name"             json:"name"`
	Username     string             `bson:"username"         json:"username"`
	Email        string             `bson:"email"            json:"email"`
	PasswordHash string             `bson:"passwordHash"      json:"-"`
	Plan         string        		`bson:"plan"			 json:"plan"`
	OAuthProvider string 			`bson:"oauthProvider" 	 json:"oauthProvider"`
	OAuthID       string 			`bson:"oauthId"       	 json:"-"`             
	Avatar       string             `bson:"avatar"           json:"avatar"`
	Bio          string             `bson:"bio"              json:"bio"`
	IsVerified   bool               `bson:"isVerified"       json:"isVerified"`
	IsActive     bool               `bson:"isActive"         json:"isActive"`
	Subscription UserSubscription   `bson:"subscription"     json:"subscription"`
	AIUsage      AIUsage            `bson:"aiUsage"          json:"aiUsage"`
	Wallet       UserWallet         `bson:"wallet"           json:"wallet"`
	Stats        UserStats          `bson:"stats"            json:"stats"`
	Preferences  UserPreferences    `bson:"preferences"      json:"preferences"`
	LastLoginAt  *time.Time         `bson:"lastLoginAt"      json:"lastLoginAt"`
	CreatedAt    time.Time          `bson:"createdAt"        json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"        json:"updatedAt"`
}

type UserSubscription struct {
	Plan        string     `bson:"plan"        json:"plan"`
	Status      string     `bson:"status"      json:"status"`
	StartDate   *time.Time `bson:"startDate"   json:"startDate"`
	RenewalDate *time.Time `bson:"renewalDate" json:"renewalDate"`
}

type AIUsage struct {
	ReviewsUsed  int        `bson:"reviewsUsed"  json:"reviewsUsed"`
	ReviewsLimit int        `bson:"reviewsLimit" json:"reviewsLimit"`
	ResetDate    *time.Time `bson:"resetDate"    json:"resetDate"`
}

type UserWallet struct {
	Balance      float64 `bson:"balance"      json:"balance"`
	TotalEarned  float64 `bson:"totalEarned"  json:"totalEarned"`
	Currency     string  `bson:"currency"     json:"currency"`
}

type UserStats struct {
	PublicRepos  int `bson:"publicRepos"  json:"publicRepos"`
	Followers    int `bson:"followers"    json:"followers"`
	Following    int `bson:"following"    json:"following"`
	TotalStars   int `bson:"totalStars"   json:"totalStars"`
}

type UserPreferences struct {
	Theme                string `bson:"theme"                json:"theme"`
	EmailNotifications   bool   `bson:"emailNotifications"   json:"emailNotifications"`
	DefaultVisibility    string `bson:"defaultVisibility"    json:"defaultVisibility"`
}
