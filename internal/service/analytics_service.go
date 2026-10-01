package service

import (
    "context"
    "encoding/json"
    "time"

    "devflow-backend/internal/database"
    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

type AnalyticsOverview struct {
	Activity         []repository.DailyActivity       `json:"activity"`
	Languages        []repository.LanguageStat        `json:"languages"`
	Repos            *repository.RepoOverview         `json:"repos"`
	PRs              *repository.PRStats              `json:"prs"`
	Issues           *repository.IssueStats           `json:"issues"`
	Marketplace      *repository.MarketplaceStats     `json:"marketplace"`
	SnippetPerf      []repository.SnippetPerformance  `json:"snippetPerformance"`
	PairSessions     *repository.PairStats            `json:"pairSessions"`
	MonthlyCommits   []repository.MonthlySeries       `json:"monthlyCommits"`
	MonthlyRevenue   []repository.MonthlySeries       `json:"monthlyRevenue"`
}

const statsCacheTTL = 5 * time.Minute
const statsCacheKey  = "platform:stats"

func GetAnalyticsOverview(ctx context.Context, userID string, days int) (*AnalyticsOverview, error) {
    since := time.Now().AddDate(0, 0, -days)

    activity, _ := repository.GetDailyCommitActivity(ctx, userID, since)
	languages, _ := repository.GetLanguageBreakdown(ctx, userID)
	repos, _ := repository.GetRepoOverview(ctx, userID)
	prs, _ := repository.GetPRStats(ctx, userID, since)
	issues, _ := repository.GetIssueStats(ctx, userID, since)
	marketplace, _ := repository.GetMarketplaceStats(ctx, userID)
	snippetPerf, _ := repository.GetSnippetPerformance(ctx, userID)
	pairStats, _ := repository.GetPairStats(ctx, userID, since)
	monthlyCommits, _ := repository.GetMonthlyCommits(ctx, userID, 12)
	monthlyRevenue, _ := repository.GetMonthlyRevenue(ctx, userID, 12)

    // Normalise nils to empty slices
	if activity == nil { activity = []repository.DailyActivity{} }
	if languages == nil { languages = []repository.LanguageStat{} }
	if snippetPerf == nil { snippetPerf = []repository.SnippetPerformance{} }
	if monthlyCommits == nil { monthlyCommits = []repository.MonthlySeries{} }
	if monthlyRevenue == nil { monthlyRevenue = []repository.MonthlySeries{} }
	if repos == nil { repos = &repository.RepoOverview{} }
	if prs == nil { prs = &repository.PRStats{} }
	if issues == nil { issues = &repository.IssueStats{} }
	if marketplace == nil { marketplace = &repository.MarketplaceStats{} }
	if pairStats == nil { pairStats = &repository.PairStats{} }

    return &AnalyticsOverview{
		Activity:       activity,
		Languages:      languages,
		Repos:          repos,
		PRs:            prs,
		Issues:         issues,
		Marketplace:    marketplace,
		SnippetPerf:    snippetPerf,
		PairSessions:   pairStats,
		MonthlyCommits: monthlyCommits,
		MonthlyRevenue: monthlyRevenue,
	}, nil
}

func GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
    if cached, ok := database.RedisGet(ctx, statsCacheKey); ok {
        var stats models.PlatformStats
        if err := json.Unmarshal([]byte(cached), &stats); err == nil {
            return &stats, nil
        }
    }
    stats, err := repository.GetPlatformStats(ctx)
    if err != nil {
        return nil, err
    }
    if data, err := json.Marshal(stats); err == nil {
        database.RedisSet(ctx, statsCacheKey, string(data), statsCacheTTL)
    }
    return stats, nil
}
