package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"time"
)

type Service struct {
	client *redis.Client
}

func NewRedisService(config RedisConfig) *Service {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Printf("✅ Connected to Redis at %s:%s", config.Host, config.Port)
	return &Service{client: client}
}

func (r *Service) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, jsonValue, ttl).Err()
}

func (r *Service) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get value: %w", err)
	}

	return json.Unmarshal([]byte(val), dest)
}

func (r *Service) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *Service) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

func (r *Service) SetExpire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *Service) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

func (r *Service) CacheUserSession(ctx context.Context, sessionID string, userID int, ttl time.Duration) error {
	sessionData := map[string]interface{}{
		"user_id":    userID,
		"created_at": time.Now().Unix(),
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return r.Set(ctx, key, sessionData, ttl)
}

func (r *Service) GetUserSession(ctx context.Context, sessionID string) (int, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	var sessionData map[string]interface{}
	err := r.Get(ctx, key, &sessionData)
	if err != nil {
		return 0, err
	}

	userID, ok := sessionData["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user_id in session")
	}

	return int(userID), nil
}

func (r *Service) DeleteUserSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.Delete(ctx, key)
}

func (r *Service) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	pipe := r.client.Pipeline()

	incr := pipe.Incr(ctx, key)

	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := incr.Val()
	return count <= int64(limit), nil
}

func (r *Service) CacheUserBehavior(ctx context.Context, userID int, data interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("user_behavior:%d", userID)
	return r.Set(ctx, key, data, ttl)
}

func (r *Service) GetUserBehavior(ctx context.Context, userID int, dest interface{}) error {
	key := fmt.Sprintf("user_behavior:%d", userID)
	return r.Get(ctx, key, dest)
}

func (r *Service) CacheMetrics(ctx context.Context, metricType string, data interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("metrics:%s", metricType)
	return r.Set(ctx, key, data, ttl)
}

func (r *Service) GetMetrics(ctx context.Context, metricType string, dest interface{}) error {
	key := fmt.Sprintf("metrics:%s", metricType)
	return r.Get(ctx, key, dest)
}

func (r *Service) AddToSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.SAdd(ctx, key, values...).Err()
}

func (r *Service) GetSet(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *Service) IsInSet(ctx context.Context, key string, value interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, value).Result()
}

func (r *Service) AddToSortedSet(ctx context.Context, key string, score float64, member interface{}) error {
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (r *Service) GetTopFromSortedSet(ctx context.Context, key string, count int64) ([]string, error) {
	return r.client.ZRevRange(ctx, key, 0, count-1).Result()
}

func (r *Service) SetHash(ctx context.Context, key, field string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.HSet(ctx, key, field, jsonValue).Err()
}

func (r *Service) GetHash(ctx context.Context, key, field string, dest interface{}) error {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("field not found: %s", field)
		}
		return fmt.Errorf("failed to get hash value: %w", err)
	}

	return json.Unmarshal([]byte(val), dest)
}

func (r *Service) GetAllHash(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

func (r *Service) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

func (r *Service) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *Service) Close() error {
	return r.client.Close()
}

func (r *Service) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
