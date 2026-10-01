package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Activity heatmap — sourced from activity_feed (actorId is string, timestamp field)
type DailyActivity struct {
	Date  string `bson:"date"  json:"date"`
	Count int    `bson:"count" json:"count"`
}

func GetDailyCommitActivity(ctx context.Context, ownerID string, since time.Time) ([]DailyActivity, error) {
	// activity_feed stores actorId as a hex string, not ObjectID
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"actorId":   ownerID,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$timestamp"}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "date": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: 1}}}},
	}
	cur, err := database.Collection("activity_feed").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []DailyActivity
	return out, cur.All(ctx, &out)
}

// Language breakdown — Repository.Language is a single string field, not an array
type LanguageStat struct {
	Language string `bson:"language" json:"language"`
	Count    int    `bson:"count"    json:"count"`
}

func GetLanguageBreakdown(ctx context.Context, ownerID string) ([]LanguageStat, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"ownerId": oid, "language": bson.M{"$ne": ""}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$language",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "language": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		{{Key: "$limit", Value: 10}},
	}
	cur, err := database.Collection("repositories").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []LanguageStat
	return out, cur.All(ctx, &out)
}

// Repository overview — stars/forks are nested under stats.stars / stats.forks
type RepoOverview struct {
	TotalRepos   int `bson:"totalRepos"   json:"totalRepos"`
	TotalStars   int `bson:"totalStars"   json:"totalStars"`
	TotalForks   int `bson:"totalForks"   json:"totalForks"`
	PublicRepos  int `bson:"publicRepos"  json:"publicRepos"`
	PrivateRepos int `bson:"privateRepos" json:"privateRepos"`
}

func GetRepoOverview(ctx context.Context, ownerID string) (*RepoOverview, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"ownerId": oid}}},
		{{Key: "$group", Value: bson.M{
			"_id":          nil,
			"totalRepos":   bson.M{"$sum": 1},
			"totalStars":   bson.M{"$sum": "$stats.stars"},
			"totalForks":   bson.M{"$sum": "$stats.forks"},
			"publicRepos":  bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$visibility", "public"}}, 1, 0}}},
			"privateRepos": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$visibility", "private"}}, 1, 0}}},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0}}},
	}
	cur, err := database.Collection("repositories").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []RepoOverview
	if err := cur.All(ctx, &results); err != nil || len(results) == 0 {
		return &RepoOverview{}, err
	}
	return &results[0], nil
}

// PR analytics
type PRStats struct {
	TotalOpened int     `bson:"totalOpened" json:"totalOpened"`
	TotalMerged int     `bson:"totalMerged" json:"totalMerged"`
	TotalClosed int     `bson:"totalClosed" json:"totalClosed"`
	AvgMergeHrs float64 `bson:"avgMergeHrs" json:"avgMergeHrs"`
}

// PR model uses state field: "open" | "closed" | "merged"
func GetPRStats(ctx context.Context, ownerID string, since time.Time) (*PRStats, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"authorId":  oid,
			"createdAt": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"totalOpened": bson.M{"$sum": 1},
			"totalMerged": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$state", "merged"}}, 1, 0}}},
			"totalClosed": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$state", "closed"}}, 1, 0}}},
			"avgMergeMs": bson.M{"$avg": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$state", "merged"}},
				bson.M{"$subtract": bson.A{"$mergedAt", "$createdAt"}},
				nil,
			}}},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":         0,
			"totalOpened": 1,
			"totalMerged": 1,
			"totalClosed": 1,
			"avgMergeHrs": bson.M{"$divide": bson.A{"$avgMergeMs", 3600000}},
		}}},
	}
	cur, err := database.Collection("pull_requests").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []PRStats
	if err := cur.All(ctx, &results); err != nil || len(results) == 0 {
		return &PRStats{}, err
	}
	return &results[0], nil
}

// Issue analytics
type IssueStats struct {
	TotalOpened int     `bson:"totalOpened" json:"totalOpened"`
	TotalClosed int     `bson:"totalClosed" json:"totalClosed"`
	AvgCloseHrs float64 `bson:"avgCloseHrs" json:"avgCloseHrs"`
}

// Issue model uses state field: "open" | "closed"
func GetIssueStats(ctx context.Context, ownerID string, since time.Time) (*IssueStats, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"authorId":  oid,
			"createdAt": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"totalOpened": bson.M{"$sum": 1},
			"totalClosed": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$state", "closed"}}, 1, 0}}},
			"avgCloseMs": bson.M{"$avg": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$state", "closed"}},
				bson.M{"$subtract": bson.A{"$closedAt", "$createdAt"}},
				nil,
			}}},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":         0,
			"totalOpened": 1,
			"totalClosed": 1,
			"avgCloseHrs": bson.M{"$divide": bson.A{"$avgCloseMs", 3600000}},
		}}},
	}
	cur, err := database.Collection("issues").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []IssueStats
	if err := cur.All(ctx, &results); err != nil || len(results) == 0 {
		return &IssueStats{}, err
	}
	return &results[0], nil
}

// Marketplace analytics
type MarketplaceStats struct {
	TotalSnippets     int     `bson:"totalSnippets"     json:"totalSnippets"`
	PublishedSnippets int     `bson:"publishedSnippets" json:"publishedSnippets"`
	TotalDownloads    int     `bson:"totalDownloads"    json:"totalDownloads"`
	TotalPurchases    int     `bson:"totalPurchases"    json:"totalPurchases"`
	TotalRevenue      float64 `bson:"totalRevenue"      json:"totalRevenue"`
	CreatorEarnings   float64 `bson:"creatorEarnings"   json:"creatorEarnings"`
	AvgRating         float64 `bson:"avgRating"         json:"avgRating"`
}

func GetMarketplaceStats(ctx context.Context, creatorID string) (*MarketplaceStats, error) {
	oid, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"creatorId": oid}}},
		{{Key: "$group", Value: bson.M{
			"_id":               nil,
			"totalSnippets":     bson.M{"$sum": 1},
			"publishedSnippets": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "published"}}, 1, 0}}},
			"totalDownloads":    bson.M{"$sum": "$stats.downloads"},
			"totalPurchases":    bson.M{"$sum": "$stats.purchases"},
			"totalRevenue":      bson.M{"$sum": "$earnings.totalRevenue"},
			"creatorEarnings":   bson.M{"$sum": "$earnings.creatorEarnings"},
			"avgRating":         bson.M{"$avg": "$stats.rating"},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0}}},
	}
	cur, err := database.Collection("snippets").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []MarketplaceStats
	if err := cur.All(ctx, &results); err != nil || len(results) == 0 {
		return &MarketplaceStats{}, err
	}
	return &results[0], nil
}

type SnippetPerformance struct {
	ID        string  `bson:"id"        json:"id"`
	Title     string  `bson:"title"     json:"title"`
	Downloads int     `bson:"downloads" json:"downloads"`
	Purchases int     `bson:"purchases" json:"purchases"`
	Revenue   float64 `bson:"revenue"   json:"revenue"`
	Rating    float64 `bson:"rating"    json:"rating"`
	Status    string  `bson:"status"    json:"status"`
}

func GetSnippetPerformance(ctx context.Context, creatorID string) ([]SnippetPerformance, error) {
	oid, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"creatorId": oid}}},
		{{Key: "$project", Value: bson.M{
			"id":        bson.M{"$toString": "$_id"},
			"title":     1,
			"downloads": "$stats.downloads",
			"purchases": "$stats.purchases",
			"revenue":   "$earnings.totalRevenue",
			"rating":    "$stats.rating",
			"status":    1,
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "downloads", Value: -1}}}},
	}
	cur, err := database.Collection("snippets").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []SnippetPerformance
	return out, cur.All(ctx, &out)
}

// Pair session analytics — ownerId is a string (not ObjectID), participants is []ParticipantInfo
type PairStats struct {
	TotalSessions int     `bson:"totalSessions" json:"totalSessions"`
	TotalMinutes  float64 `bson:"totalMinutes"  json:"totalMinutes"`
	AvgMinutes    float64 `bson:"avgMinutes"    json:"avgMinutes"`
}

func GetPairStats(ctx context.Context, userID string, since time.Time) (*PairStats, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"$or":       bson.A{bson.M{"ownerId": userID}, bson.M{"participants.userId": userID}},
			"createdAt": bson.M{"$gte": since},
			"status":    "ended",
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"totalSessions": bson.M{"$sum": 1},
			"totalMs":       bson.M{"$sum": bson.M{"$subtract": bson.A{"$endedAt", "$createdAt"}}},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":           0,
			"totalSessions": 1,
			"totalMinutes":  bson.M{"$divide": bson.A{"$totalMs", 60000}},
			"avgMinutes":    bson.M{"$divide": bson.A{bson.M{"$divide": bson.A{"$totalMs", 60000}}, "$totalSessions"}},
		}}},
	}
	cur, err := database.Collection("pair_sessions").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []PairStats
	if err := cur.All(ctx, &results); err != nil || len(results) == 0 {
		return &PairStats{}, err
	}
	return &results[0], nil
}

// Monthly series
type MonthlySeries struct {
	Month string `bson:"month" json:"month"`
	Count int    `bson:"count" json:"count"`
}

// GetMonthlyCommits counts activity_feed events per month for the given actor
func GetMonthlyCommits(ctx context.Context, ownerID string, months int) ([]MonthlySeries, error) {
	since := time.Now().AddDate(0, -months, 0)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"actorId": ownerID, "timestamp": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m", "date": "$timestamp"}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "month": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "month", Value: 1}}}},
	}
	cur, err := database.Collection("activity_feed").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []MonthlySeries
	return out, cur.All(ctx, &out)
}

func GetMonthlyRevenue(ctx context.Context, creatorID string, months int) ([]MonthlySeries, error) {
	oid, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}
	since := time.Now().AddDate(0, -months, 0)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"sellerId": oid, "status": "paid", "paidAt": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m", "date": "$paidAt"}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "month": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "month", Value: 1}}}},
	}
	cur, err := database.Collection("snippet_purchases").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []MonthlySeries
	return out, cur.All(ctx, &out)
}

func GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	db := database.GetDB()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	users, err := db.Collection("users").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	repos, err := db.Collection("repositories").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	snippets, err := db.Collection("snippets").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	aiReviews, err := db.Collection("aiReviews").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	return &models.PlatformStats{
		TotalUsers: users,
		TotalRepositories: repos,
		TotalSnippets: snippets,
		TotalAIReviews: aiReviews,
	}, nil
}
