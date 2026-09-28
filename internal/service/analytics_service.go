package service

import (
    "context"
    "encoding/json"
    "time"

    "devflow-backend/internal/database"
    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

const statsCacheTTL = 5 * time.Minute
const statsCacheKey  = "platform:stats"

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
