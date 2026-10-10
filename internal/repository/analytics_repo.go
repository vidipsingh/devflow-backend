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

// ─── Time-series with granularity ─────────────────────────────────────────────

// TimeSeriesPoint is a generic date+count point used for daily/weekly/monthly series.
type TimeSeriesPoint struct {
	Date  string `bson:"date"  json:"date"`
	Count int    `bson:"count" json:"count"`
}

// granularityFormat maps "daily" → "%Y-%m-%d", "weekly" → "%Y-%U", "monthly" → "%Y-%m".
func granularityFormat(granularity string) string {
	switch granularity {
	case "weekly":
		return "%Y-%U"
	case "monthly":
		return "%Y-%m"
	default: // "daily"
		return "%Y-%m-%d"
	}
}

// GetActivityTimeSeries returns commit/activity counts bucketed by the given granularity.
func GetActivityTimeSeries(ctx context.Context, ownerID string, since time.Time, granularity string) ([]TimeSeriesPoint, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"actorId":   ownerID,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": granularityFormat(granularity), "date": "$timestamp"}},
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
	var out []TimeSeriesPoint
	return out, cur.All(ctx, &out)
}

// ─── Per-repository analytics ─────────────────────────────────────────────────

// RepoAnalytics holds analytics for a single repository.
type RepoAnalytics struct {
	RepoID      string            `json:"repoId"`
	RepoName    string            `json:"repoName"`
	Commits     int               `json:"commits"`
	Stars       int               `json:"stars"`
	Forks       int               `json:"forks"`
	OpenIssues  int               `json:"openIssues"`
	OpenPRs     int               `json:"openPRs"`
	Language    string            `json:"language"`
	TimeSeries  []TimeSeriesPoint `json:"timeSeries"`
}

// RepoSummary is a lightweight struct to read name+stats from the repositories collection.
type RepoSummary struct {
	ID       bson.ObjectID `bson:"_id"`
	Name     string        `bson:"name"`
	Language string        `bson:"language"`
	Stats    struct {
		Stars      int `bson:"stars"`
		Forks      int `bson:"forks"`
		OpenIssues int `bson:"openIssues"`
		OpenPRs    int `bson:"openPRs"`
	} `bson:"stats"`
}

// GetRepoAnalytics returns analytics for a specific repository owned by ownerID.
func GetRepoAnalytics(ctx context.Context, ownerID, repoName string, since time.Time, granularity string) (*RepoAnalytics, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}

	// 1. Load repo summary
	var repo RepoSummary
	err = database.Collection("repositories").FindOne(ctx, bson.M{
		"ownerId": oid,
		"name":    repoName,
	}).Decode(&repo)
	if err != nil {
		return nil, err
	}

	repoIDStr := repo.ID.Hex()

	// 2. Count commits (activity_feed entries) for this repo in the window
	commitPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"actorId":   ownerID,
			"repoId":    repoIDStr,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$count", Value: "total"}},
	}
	commitCur, err := database.Collection("activity_feed").Aggregate(ctx, commitPipeline)
	commits := 0
	if err == nil {
		defer commitCur.Close(ctx)
		var cr []struct{ Total int `bson:"total"` }
		if e := commitCur.All(ctx, &cr); e == nil && len(cr) > 0 {
			commits = cr[0].Total
		}
	}

	// 3. Time series for this specific repo
	tsPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"actorId":   ownerID,
			"repoId":    repoIDStr,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": granularityFormat(granularity), "date": "$timestamp"}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "date": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: 1}}}},
	}
	tsCur, err := database.Collection("activity_feed").Aggregate(ctx, tsPipeline)
	var ts []TimeSeriesPoint
	if err == nil {
		defer tsCur.Close(ctx)
		_ = tsCur.All(ctx, &ts)
	}
	if ts == nil {
		ts = []TimeSeriesPoint{}
	}

	return &RepoAnalytics{
		RepoID:     repoIDStr,
		RepoName:   repo.Name,
		Commits:    commits,
		Stars:      repo.Stats.Stars,
		Forks:      repo.Stats.Forks,
		OpenIssues: repo.Stats.OpenIssues,
		OpenPRs:    repo.Stats.OpenPRs,
		Language:   repo.Language,
		TimeSeries: ts,
	}, nil
}

// GetAllReposAnalytics returns per-repo activity summary for all repos owned by ownerID.
type RepoActivitySummary struct {
	RepoID   string `json:"repoId"`
	RepoName string `json:"repoName"`
	Language string `json:"language"`
	Commits  int    `json:"commits"`
	Stars    int    `json:"stars"`
	Forks    int    `json:"forks"`
}

func GetAllReposAnalytics(ctx context.Context, ownerID string, since time.Time) ([]RepoActivitySummary, error) {
	oid, err := bson.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, err
	}

	// Fetch all repos for this owner
	cur, err := database.Collection("repositories").Find(ctx, bson.M{"ownerId": oid})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var repos []RepoSummary
	if err := cur.All(ctx, &repos); err != nil {
		return nil, err
	}

	// Build activity count per repoId from activity_feed
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"actorId":   ownerID,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$repoId",
			"count": bson.M{"$sum": 1},
		}}},
	}
	actCur, err := database.Collection("activity_feed").Aggregate(ctx, pipeline)
	activityMap := map[string]int{}
	if err == nil {
		defer actCur.Close(ctx)
		var rows []struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if e := actCur.All(ctx, &rows); e == nil {
			for _, r := range rows {
				activityMap[r.ID] = r.Count
			}
		}
	}

	out := make([]RepoActivitySummary, 0, len(repos))
	for _, r := range repos {
		out = append(out, RepoActivitySummary{
			RepoID:   r.ID.Hex(),
			RepoName: r.Name,
			Language: r.Language,
			Commits:  activityMap[r.ID.Hex()],
			Stars:    r.Stats.Stars,
			Forks:    r.Stats.Forks,
		})
	}
	return out, nil
}

// ─── Team analytics ───────────────────────────────────────────────────────────

// TeamAnalytics holds analytics for a team.
type TeamAnalytics struct {
	TotalMembers   int               `json:"totalMembers"`
	TotalRepos     int               `json:"totalRepos"`
	TotalCommits   int               `json:"totalCommits"`
	TotalDiscussions int             `json:"totalDiscussions"`
	MemberActivity []MemberActivity  `json:"memberActivity"`
	TimeSeries     []TimeSeriesPoint `json:"timeSeries"`
}

// MemberActivity holds activity count for a single team member.
type MemberActivity struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Commits  int    `json:"commits"`
}

// GetTeamAnalytics returns analytics for a team identified by its slug.
func GetTeamAnalytics(ctx context.Context, teamSlug string, since time.Time, granularity string) (*TeamAnalytics, error) {
	// 1. Load team to get its ID
	var team struct {
		ID          bson.ObjectID `bson:"_id"`
		MemberCount int           `bson:"memberCount"`
		RepoCount   int           `bson:"repoCount"`
	}
	err := database.Collection("teams").FindOne(ctx, bson.M{"slug": teamSlug}).Decode(&team)
	if err != nil {
		return nil, err
	}
	teamIDStr := team.ID.Hex()

	// 2. Count discussions for this team
	discussionCount, _ := database.Collection("discussions").CountDocuments(ctx, bson.M{"teamId": team.ID})

	// 3. Aggregate activity from activity_feed scoped to this team's members
	// activity_feed has teamId field on team-scoped events; fall back to counting by teamId
	tsPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"teamId":    teamIDStr,
			"timestamp": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": granularityFormat(granularity), "date": "$timestamp"}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{"_id": 0, "date": "$_id", "count": 1}}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: 1}}}},
	}
	tsCur, _ := database.Collection("activity_feed").Aggregate(ctx, tsPipeline)
	var ts []TimeSeriesPoint
	if tsCur != nil {
		defer tsCur.Close(ctx)
		_ = tsCur.All(ctx, &ts)
	}
	if ts == nil {
		ts = []TimeSeriesPoint{}
	}

	// 4. Total commits for this team
	totalCommits := 0
	for _, p := range ts {
		totalCommits += p.Count
	}

	// 5. Per-member activity — join team_members with activity_feed
	memberPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"teamId": team.ID}}},
		{{Key: "$lookup", Value: bson.M{
			"from": "activity_feed",
			"let":  bson.M{"uid": bson.M{"$toString": "$userId"}},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$actorId", "$$uid"}},
					bson.M{"$gte": bson.A{"$timestamp", since}},
				}}}}},
				{{Key: "$count", Value: "c"}},
			},
			"as": "activity",
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      0,
			"userId":   bson.M{"$toString": "$userId"},
			"username": 1,
			"commits":  bson.M{"$ifNull": bson.A{bson.M{"$arrayElemAt": bson.A{"$activity.c", 0}}, 0}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "commits", Value: -1}}}},
		{{Key: "$limit", Value: 10}},
	}
	memberCur, err := database.Collection("team_members").Aggregate(ctx, memberPipeline)
	var memberActivity []MemberActivity
	if err == nil {
		defer memberCur.Close(ctx)
		_ = memberCur.All(ctx, &memberActivity)
	}
	if memberActivity == nil {
		memberActivity = []MemberActivity{}
	}

	return &TeamAnalytics{
		TotalMembers:     team.MemberCount,
		TotalRepos:       team.RepoCount,
		TotalCommits:     totalCommits,
		TotalDiscussions: int(discussionCount),
		MemberActivity:   memberActivity,
		TimeSeries:       ts,
	}, nil
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
