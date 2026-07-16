package database

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func ConnectRedis() {
    url := os.Getenv("REDIS_URL")
    if url == "" {
        log.Println("REDIS_URL not set - caching disabled")
        return
    }

    opt, err := redis.ParseURL(url)
    if err != nil {
        log.Printf("Redis Url parse error: %v - caching disabled", err)
        return
    }

    redisClient = redis.NewClient(opt)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := redisClient.Ping(ctx).Err(); err != nil {
        log.Printf("Redis ping failed: %v - caching disabled", err)
        redisClient = nil
        return
    }
    log.Println("✅ Connected to Redis!")
}

// RedisGet returns the cached value for key, or ("", false) if not found or Redis is down
func RedisGet(ctx context.Context, key string) (string, bool) {
    if redisClient == nil {
        return "", false
    }
    val, err := redisClient.Get(ctx, key).Result()
    if err == redis.Nil || err != nil {
        return "", false
    }
    return val, true
}

// RedisSet stores a string value with a TTL
func RedisSet(ctx context.Context, key, value string, ttl time.Duration) {
    if redisClient == nil {
        return
    }
    redisClient.Set(ctx, key, value, ttl)
}

// RedisDel deletes one or more cache keys
func RedisDel(ctx context.Context, keys ...string) {
    if redisClient == nil {
        return
    }
    redisClient.Del(ctx, keys...)
}

// RedisDelPattern deletes all keys matching a glob pattern (e.g. "tree:repoId:*").
func RedisDelPattern(ctx context.Context, pattern string) {
    if redisClient == nil {
        return
    }
    var cursor uint64
    for  {
        keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            break
        }
        if len(keys) > 0 {
            redisClient.Del(ctx, keys...)
        }
        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }
}