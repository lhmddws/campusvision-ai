package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps go-redis for dormitory-service operations.
type Client struct {
	*goredis.Client
}

// NewClient creates a new Redis client from the config.
func NewClient(host string, port int, db int, password string) *Client {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 3,
	})
	return &Client{rdb}
}

// SetNX sets a key only if it does not already exist.
// Returns true if the key was set, false if it already exists.
// Used for deduplication with TTL.
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return c.Client.SetNX(ctx, key, value, ttl).Result()
}

// Get retrieves a string value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Client.Get(ctx, key).Result()
}

// Set sets a key with a TTL.
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.Client.Set(ctx, key, value, ttl).Err()
}

// Del deletes one or more keys.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.Client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire sets a TTL on an existing key.
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.Client.Expire(ctx, key, ttl).Err()
}

// Ping checks if the Redis server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}

// Close closes the Redis client.
func (c *Client) Close() error {
	return c.Client.Close()
}

// Default event dedup TTL (1 hour, matching Java's DEDUP_TTL_SECONDS=3600).
const DefaultDedupTTL = 3600 * time.Second

// CheckAndSetDedup checks if an event has already been processed (for deduplication).
// Key format: "dedup:{camera_id}:{frame_sequence}"
// Returns true if this is a new event (not yet processed).
func (c *Client) CheckAndSetDedup(ctx context.Context, cameraID string, frameSequence int) (bool, error) {
	key := fmt.Sprintf("dedup:%s:%d", cameraID, frameSequence)
	return c.SetNX(ctx, key, "1", DefaultDedupTTL)
}
