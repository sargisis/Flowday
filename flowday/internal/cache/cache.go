package cache

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"flowday/internal/logger"

	"github.com/redis/go-redis/v9"
)

var (
	cacheMap sync.Map
	mu       sync.RWMutex
	redisClient *redis.Client
	useRedis bool
)

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// Init initializes cache with Redis support (if available) or falls back to in-memory
func Init() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		logger.Log.Info("Redis not configured, using in-memory cache")
		useRedis = false
		return
	}

	// Try to connect to Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Log.WithError(err).Warn("Failed to parse Redis URL, using in-memory cache")
		useRedis = false
		return
	}

	redisClient = redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Log.WithError(err).Warn("Failed to connect to Redis, using in-memory cache")
		useRedis = false
		return
	}

	useRedis = true
	logger.Log.Info("✅ Redis cache initialized successfully")
}

// Get retrieves a value from cache (Redis or in-memory)
func Get(key string) (interface{}, bool) {
	if useRedis && redisClient != nil {
		return getFromRedis(key)
	}
	return getFromMemory(key)
}

// Set stores a value in cache (Redis or in-memory)
func Set(key string, value interface{}, ttl time.Duration) {
	if useRedis && redisClient != nil {
		setInRedis(key, value, ttl)
		return
	}
	setInMemory(key, value, ttl)
}

// Delete removes a key from cache
func Delete(key string) {
	if useRedis && redisClient != nil {
		deleteFromRedis(key)
		return
	}
	deleteFromMemory(key)
}

// Clear removes all keys from cache
func Clear() {
	if useRedis && redisClient != nil {
		clearRedis()
		return
	}
	clearMemory()
}

// In-memory cache functions
func getFromMemory(key string) (interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()
	
	item, ok := cacheMap.Load(key)
	if !ok {
		return nil, false
	}
	
	cacheItem := item.(cacheItem)
	if time.Now().After(cacheItem.expiration) {
		cacheMap.Delete(key)
		return nil, false
	}
	
	return cacheItem.value, true
}

func setInMemory(key string, value interface{}, ttl time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	
	cacheMap.Store(key, cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	})
}

func deleteFromMemory(key string) {
	mu.Lock()
	defer mu.Unlock()
	cacheMap.Delete(key)
}

func clearMemory() {
	mu.Lock()
	defer mu.Unlock()
	cacheMap.Range(func(key, value interface{}) bool {
		cacheMap.Delete(key)
		return true
	})
}

// Redis cache functions
func getFromRedis(key string) (interface{}, bool) {
	ctx := context.Background()
	val, err := redisClient.Get(ctx, "cache:"+key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		logger.Log.WithError(err).Warn("Redis get error, falling back to memory")
		return getFromMemory(key)
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, false
	}
	return result, true
}

func setInRedis(key string, value interface{}, ttl time.Duration) {
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		logger.Log.WithError(err).Warn("Failed to marshal cache value")
		return
	}

	if err := redisClient.Set(ctx, "cache:"+key, data, ttl).Err(); err != nil {
		logger.Log.WithError(err).Warn("Redis set error, falling back to memory")
		setInMemory(key, value, ttl)
	}
}

func deleteFromRedis(key string) {
	ctx := context.Background()
	if err := redisClient.Del(ctx, "cache:"+key).Err(); err != nil {
		logger.Log.WithError(err).Warn("Redis delete error")
	}
}

func clearRedis() {
	ctx := context.Background()
	// Clear all cache keys (keys with prefix "cache:")
	iter := redisClient.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		redisClient.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		logger.Log.WithError(err).Warn("Redis clear error")
	}
}

// Helper functions for common cache patterns

// GetOrSet retrieves a value from cache, or sets it if not found
func GetOrSet(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := Get(key); ok {
		return val, nil
	}

	val, err := fn()
	if err != nil {
		return nil, err
	}

	Set(key, val, ttl)
	return val, nil
}

// InvalidatePattern removes all keys matching a pattern (Redis only)
func InvalidatePattern(pattern string) {
	if !useRedis || redisClient == nil {
		return
	}

	ctx := context.Background()
	iter := redisClient.Scan(ctx, 0, "cache:"+pattern, 0).Iterator()
	for iter.Next(ctx) {
		redisClient.Del(ctx, iter.Val())
	}
}
