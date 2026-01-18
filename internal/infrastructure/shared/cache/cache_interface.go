package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations.
// This provides a unified interface for different cache implementations.
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in the cache with optional TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Clear clears all values from the cache
	Clear(ctx context.Context) error

	// GetTTL returns the remaining TTL for a key
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// SetTTL sets the TTL for an existing key
	SetTTL(ctx context.Context, key string, ttl time.Duration) error

	// Keys returns all keys matching a pattern (if supported)
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Close closes the cache connection
	Close() error
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64
	Misses      int64
	KeysAdded   int64
	KeysEvicted int64
	KeysExpired int64
}

// Stats returns cache statistics
func (c Cache) Stats() CacheStats {
	// Default implementation - implementations should override
	return CacheStats{}
}
