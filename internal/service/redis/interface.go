package redis

import (
	"context"
	"time"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type ServiceInterface interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetExpire(ctx context.Context, key string, ttl time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	CacheUserSession(ctx context.Context, sessionID string, userID int, ttl time.Duration) error
	GetUserSession(ctx context.Context, sessionID string) (int, error)
	DeleteUserSession(ctx context.Context, sessionID string) error

	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	CacheUserBehavior(ctx context.Context, userID int, data interface{}, ttl time.Duration) error
	GetUserBehavior(ctx context.Context, userID int, dest interface{}) error
	CacheMetrics(ctx context.Context, metricType string, data interface{}, ttl time.Duration) error
	GetMetrics(ctx context.Context, metricType string, dest interface{}) error

	AddToSet(ctx context.Context, key string, values ...interface{}) error
	GetSet(ctx context.Context, key string) ([]string, error)
	IsInSet(ctx context.Context, key string, value interface{}) (bool, error)

	AddToSortedSet(ctx context.Context, key string, score float64, member interface{}) error
	GetTopFromSortedSet(ctx context.Context, key string, count int64) ([]string, error)

	SetHash(ctx context.Context, key, field string, value interface{}) error
	GetHash(ctx context.Context, key, field string, dest interface{}) error
	GetAllHash(ctx context.Context, key string) (map[string]string, error)

	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error
	Health(ctx context.Context) error
	Close() error
}
